package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	linerprogress "github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const (
	methodologySeedFullLowTokens  = 7_000_000
	methodologySeedFullHighTokens = 9_000_000
	localEstimateMinSamples       = 3
	globalEstimateMaxEntries      = 600
)

// Seeded from the 2026-05-28 Opus methodology pass until local history can take over.
var methodologySeedPhaseTokens = map[string]int{
	"framing":    350_000,
	"candidates": 3_200_000,
	"evaluation": 2_200_000,
	"quality":    1_100_000,
	"synthesis":  500_000,
	"assembly":   550_000,
}

func (m Model) projectRunEstimateView(width int, pendingDraft bool, compiled bool, sourceEntryPrimary bool) string {
	if pendingDraft || compiled || sourceEntryPrimary || strings.TrimSpace(m.currentPath) == "" {
		return ""
	}
	phase, ok := nextEstimatedMethodologyPhase(m.currentPath)
	if !ok {
		return ""
	}
	samples := collectEstimateSamples(m.baseDir)
	phaseTokens, phaseBasis := estimatePhaseTokensFromSamples(samples, phase)
	if phaseTokens <= 0 {
		return ""
	}
	fullEstimate, fullBasis := estimateFullMethodologyTokensFromSamples(samples)
	scopeWidth := min(26, max(18, width/3))
	estimateWidth := min(18, max(16, width/4))
	basisWidth := max(18, width-scopeWidth-estimateWidth-6)
	estimateTable := newDataTable(
		[]table.Column{
			{Title: "Scope", Width: scopeWidth},
			{Title: "Estimate", Width: estimateWidth},
			{Title: "Basis", Width: basisWidth},
		},
		[]table.Row{
			{"next: " + methodologyPhaseLabel(phase), "~" + formatCompactTokenCount(phaseTokens) + " tokens", phaseBasis},
			{"full corpus build", fullEstimate, fullBasis},
		},
		width,
		3,
		false,
	)
	return estimateTable.View()
}

func nextEstimatedMethodologyPhase(project string) (string, bool) {
	if strings.TrimSpace(project) == "" {
		return "", false
	}
	current := linerprogress.Read(projectCorpusPath(project))
	if current.Step >= len(linerprogress.PhaseOrder) {
		return "", false
	}
	phase := linerprogress.PhaseOrder[current.Step]
	if index, ok := methodologyIndexForProgressPhase(phase); ok {
		if index >= len(methodologyPhaseOrder) {
			return "", false
		}
		return methodologyPhaseOrder[index], true
	}
	switch phase {
	case linerprogress.PhaseGate0:
		return "candidates", true
	case linerprogress.PhaseGate1:
		return "evaluation", true
	case linerprogress.PhaseGate2:
		return "synthesis", true
	default:
		return "", false
	}
}

type estimateSamples struct {
	local  map[string][]int
	global map[string][]int
}

type phaseTokenRecord struct {
	Phase      string
	Tokens     int
	Source     string
	ObservedAt time.Time
}

type globalEstimateEntry struct {
	Version    int    `json:"version"`
	Phase      string `json:"phase"`
	Tokens     int    `json:"tokens"`
	Source     string `json:"source"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

func collectEstimateSamples(baseDir string) estimateSamples {
	localRecords := localPhaseTokenRecords(baseDir, "")
	syncGlobalEstimateHistory(localRecords)
	samples := estimateSamples{
		local:  map[string][]int{},
		global: map[string][]int{},
	}
	for _, record := range localRecords {
		samples.local[record.Phase] = append(samples.local[record.Phase], record.Tokens)
	}
	for _, entry := range readGlobalEstimateEntries() {
		if !knownMethodologyPhase(entry.Phase) || entry.Tokens <= 0 {
			continue
		}
		samples.global[entry.Phase] = append(samples.global[entry.Phase], entry.Tokens)
	}
	return samples
}

func estimatePhaseTokensFromSamples(samples estimateSamples, phase string) (int, string) {
	local := samples.local[phase]
	if len(local) >= localEstimateMinSamples {
		return medianTokenCount(local), fmt.Sprintf("local median (%d)", len(local))
	}
	global := samples.global[phase]
	if len(global) >= localEstimateMinSamples {
		return medianTokenCount(global), fmt.Sprintf("global median (%d)", len(global))
	}
	return methodologySeedPhaseTokens[phase], "seed baseline"
}

func estimateFullMethodologyTokensFromSamples(samples estimateSamples) (string, string) {
	total := 0
	usedLocal := false
	usedGlobal := false
	for _, phase := range methodologyPhaseOrder {
		local := samples.local[phase]
		if len(local) >= localEstimateMinSamples {
			total += medianTokenCount(local)
			usedLocal = true
			continue
		}
		global := samples.global[phase]
		if len(global) >= localEstimateMinSamples {
			total += medianTokenCount(global)
			usedGlobal = true
			continue
		}
		return fmt.Sprintf("~%s-%s tokens", formatCompactTokenCount(methodologySeedFullLowTokens), formatCompactTokenCount(methodologySeedFullHighTokens)), "Opus seed run"
	}
	basis := "local medians"
	if usedGlobal && usedLocal {
		basis = "mixed medians"
	} else if usedGlobal {
		basis = "global medians"
	}
	return "~" + formatCompactTokenCount(total) + " tokens", basis
}

func localPhaseTokenRecords(baseDir string, phase string) []phaseTokenRecord {
	baseDir = strings.TrimSpace(baseDir)
	phase = strings.TrimSpace(phase)
	if baseDir == "" {
		return nil
	}
	projects, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}
	var records []phaseTokenRecord
	for _, project := range projects {
		if !project.IsDir() {
			continue
		}
		corpusPath := tape.CorpusPath(filepath.Join(baseDir, project.Name()))
		if phase != "" {
			records = append(records, localPhaseTokenRecordsInDir(filepath.Join(corpusPath, ".liner-runs", phase), phase)...)
			continue
		}
		runDir := filepath.Join(corpusPath, ".liner-runs")
		phaseDirs, err := os.ReadDir(runDir)
		if err != nil {
			continue
		}
		for _, phaseDir := range phaseDirs {
			if !phaseDir.IsDir() || !knownMethodologyPhase(phaseDir.Name()) {
				continue
			}
			records = append(records, localPhaseTokenRecordsInDir(filepath.Join(runDir, phaseDir.Name()), phaseDir.Name())...)
		}
	}
	return records
}

func localPhaseTokenRecordsInDir(phaseDir string, phase string) []phaseTokenRecord {
	if !knownMethodologyPhase(phase) {
		return nil
	}
	logs, err := os.ReadDir(phaseDir)
	if err != nil {
		return nil
	}
	var records []phaseTokenRecord
	for _, log := range logs {
		if log.IsDir() || !strings.HasSuffix(log.Name(), ".jsonl") {
			continue
		}
		logPath := filepath.Join(phaseDir, log.Name())
		usage, ok := parseRunUsageLog(logPath)
		if !ok || usage.totalTokens() <= 0 {
			continue
		}
		info, err := os.Stat(logPath)
		if err != nil {
			continue
		}
		source := estimateSourceKey(logPath)
		if source == "" {
			continue
		}
		records = append(records, phaseTokenRecord{
			Phase:      phase,
			Tokens:     usage.totalTokens(),
			Source:     source,
			ObservedAt: info.ModTime(),
		})
	}
	return records
}

func knownMethodologyPhase(phase string) bool {
	_, ok := methodologySeedPhaseTokens[phase]
	return ok
}

func syncGlobalEstimateHistory(records []phaseTokenRecord) {
	if len(records) == 0 {
		return
	}
	path := globalEstimateHistoryPath()
	if path == "" {
		return
	}
	resetAt := readGlobalEstimateResetAt()
	existing := readGlobalEstimateEntriesFromPath(path)
	seen := map[string]bool{}
	for _, entry := range existing {
		if entry.Source != "" {
			seen[entry.Source] = true
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	added := false
	for _, record := range records {
		if !resetAt.IsZero() && !record.ObservedAt.After(resetAt) {
			continue
		}
		if record.Source == "" || seen[record.Source] || record.Tokens <= 0 || !knownMethodologyPhase(record.Phase) {
			continue
		}
		existing = append(existing, globalEstimateEntry{
			Version:    1,
			Phase:      record.Phase,
			Tokens:     record.Tokens,
			Source:     record.Source,
			RecordedAt: now,
		})
		seen[record.Source] = true
		added = true
	}
	if !added {
		return
	}
	if len(existing) > globalEstimateMaxEntries {
		existing = existing[len(existing)-globalEstimateMaxEntries:]
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	var b strings.Builder
	for _, entry := range existing {
		if !knownMethodologyPhase(entry.Phase) || entry.Tokens <= 0 {
			continue
		}
		line, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	_ = os.WriteFile(path, []byte(b.String()), 0o644)
}

func readGlobalEstimateEntries() []globalEstimateEntry {
	return readGlobalEstimateEntriesFromPath(globalEstimateHistoryPath())
}

func readGlobalEstimateEntriesFromPath(path string) []globalEstimateEntry {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	entries := make([]globalEstimateEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry globalEstimateEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Version != 1 || entry.Tokens <= 0 || !knownMethodologyPhase(entry.Phase) {
			continue
		}
		entries = append(entries, entry)
	}
	if len(entries) > globalEstimateMaxEntries {
		entries = entries[len(entries)-globalEstimateMaxEntries:]
	}
	return entries
}

func globalEstimateHistoryPath() string {
	if value, ok := os.LookupEnv("LINER_ESTIMATE_HISTORY"); ok {
		return strings.TrimSpace(value)
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".liner", "run-estimates.jsonl")
}

func globalEstimateResetPath() string {
	path := globalEstimateHistoryPath()
	if path == "" {
		return ""
	}
	return path + ".reset"
}

func globalEstimateHistoryHasEntries() bool {
	return len(readGlobalEstimateEntries()) > 0
}

func clearGlobalEstimateHistory() (string, int, error) {
	path := globalEstimateHistoryPath()
	if path == "" {
		return "", 0, fmt.Errorf("run estimate history is disabled")
	}
	count := len(readGlobalEstimateEntriesFromPath(path))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return path, count, err
	}
	resetPath := globalEstimateResetPath()
	if resetPath != "" {
		if err := os.MkdirAll(filepath.Dir(resetPath), 0o755); err != nil {
			return path, count, err
		}
		resetAt := time.Now().UTC().Format(time.RFC3339Nano)
		if err := os.WriteFile(resetPath, []byte(resetAt+"\n"), 0o644); err != nil {
			return path, count, err
		}
	}
	return path, count, nil
}

func readGlobalEstimateResetAt() time.Time {
	path := globalEstimateResetPath()
	if path == "" {
		return time.Time{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (m Model) clearEstimateHistory() (Model, tea.Cmd) {
	_, count, err := clearGlobalEstimateHistory()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if count == 0 {
		m.note = "Run estimate history was already empty."
	} else {
		m.note = fmt.Sprintf("Cleared %d global run estimate sample(s). Local run logs stay on disk.", count)
	}
	return m, nil
}

func estimateSourceKey(path string) string {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		absolute,
		strconv.FormatInt(info.Size(), 10),
		strconv.FormatInt(info.ModTime().UnixNano(), 10),
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func medianTokenCount(samples []int) int {
	if len(samples) == 0 {
		return 0
	}
	values := append([]int(nil), samples...)
	sort.Ints(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func formatCompactTokenCount(value int) string {
	switch {
	case value >= 1_000_000:
		if value%1_000_000 == 0 {
			return fmt.Sprintf("%dM", value/1_000_000)
		}
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case value >= 1_000:
		if value%1_000 == 0 {
			return fmt.Sprintf("%dk", value/1_000)
		}
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	default:
		return formatCount(value)
	}
}
