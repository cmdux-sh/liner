package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"gopkg.in/yaml.v3"

	"github.com/cmdux/liner/packages/go-tui/internal/agent"
	linerprogress "github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

var methodologyPhaseOrder = []string{
	"framing",
	"candidates",
	"evaluation",
	"quality",
	"synthesis",
	"assembly",
}

var methodologyPhaseNames = map[string]string{
	"framing":     "Framing",
	"candidates":  "Candidate discovery",
	"evaluation":  "Evaluation",
	"quality":     "Quality checks",
	"synthesis":   "Synthesis",
	"assembly":    "Assembly",
	"improvement": "Improve Corpus",
}

var methodologyArtifacts = map[string]string{
	"framing":    "working/01-jtbd-and-knowledge-map.md",
	"candidates": "working/02-candidate-longlist.md",
	"evaluation": "working/03-evaluation.yaml",
	"quality":    "working/04-quality-checks.md",
	"synthesis":  "synthesis.md",
	"assembly":   "working/07-tape-draft.yaml",
}

func (m Model) startResearch() (Model, tea.Cmd) {
	m.stopMethodology("")
	m.screen = screenResearch
	m.researchDone = false
	m.methodologyFailed = false
	m.methodologyCancelled = false
	m.methodologyLastErr = ""
	m.methodologyFailureKind = ""
	m.methodologyPrimaryFailure = ""
	m.methodologyRecovery = ""
	m.methodologyDiagnostics = nil
	m.methodologyRawLog = nil
	m.methodologyLogPath = ""
	m.methodologyPhaseID = ""
	m.methodologyEventCount = 0
	m.methodologyLastEventFrame = m.fxFrame
	m.ensureBoardItems()
	m.note = ""

	startIndex, notes, err := m.nextMethodologyPhaseIndex()
	if err != nil {
		m.researchDone = true
		m.err = err.Error()
		m.researchLines = []string{"Corpus build could not continue.", err.Error()}
		m.syncMethodologyLog(true)
		return m, nil
	}
	m.researchStep = startIndex
	m.methodologyPhaseIndex = startIndex
	m.researchLines = []string{
		"Starting Corpus Creation...",
		fmt.Sprintf("Queued %d phases.", len(methodologyPhaseOrder)),
	}
	m.researchLines = append(m.researchLines, notes...)
	if startIndex > 0 && startIndex < len(methodologyPhaseOrder) {
		m.researchLines = append(m.researchLines, fmt.Sprintf("Continuing at %s.", methodologyPhaseLabel(methodologyPhaseOrder[startIndex])))
	}
	if startIndex >= len(methodologyPhaseOrder) {
		m.researchDone = true
		m.note = "Corpus build is already complete. Starting compile."
		m.syncMethodologyLog(true)
		return m.startCompile()
	}
	m.syncMethodologyLog(true)
	return m.startMethodologyPhase(startIndex, m.shouldResumeMethodologyPhase(startIndex))
}

func (m Model) startMethodologyPhase(index int, resume bool) (Model, tea.Cmd) {
	if index < 0 || index >= len(methodologyPhaseOrder) {
		m.researchDone = true
		return m, nil
	}
	if strings.TrimSpace(m.currentPath) == "" {
		m.researchDone = true
		m.err = "Cannot build the corpus without a project path."
		m.syncMethodologyLog(true)
		return m, nil
	}

	script := resolveHeadlessRunnerScript()
	if script == "" {
		m.researchDone = true
		m.err = "Could not find the Corpus Builder runner. Run npm --prefix packages/tui run build, or set LINER_HEADLESS_RUNNER."
		m.syncMethodologyLog(true)
		return m, nil
	}

	phaseID := methodologyPhaseOrder[index]
	corpusPath := projectCorpusPath(m.currentPath)
	ctx, cancel := context.WithCancel(context.Background())
	run, err := agent.Runner{ScriptPath: script}.Start(ctx, agent.RunArgs{
		Project: corpusPath,
		PhaseID: phaseID,
		Agent:   "auto",
		Resume:  resume,
	})
	if err != nil {
		cancel()
		m.researchDone = true
		m.err = err.Error()
		m.syncMethodologyLog(true)
		return m, nil
	}

	m.methodologyCancel = cancel
	m.methodologyEvents = run.Events
	m.methodologyDone = run.Done
	m.methodologyPhaseIndex = index
	m.methodologyPhaseID = phaseID
	m.methodologyEventCount = 0
	m.methodologyLastEventFrame = m.fxFrame
	m.methodologyFailed = false
	m.methodologyCancelled = false
	m.methodologyLastErr = ""
	m.methodologyFailureKind = ""
	m.methodologyPrimaryFailure = ""
	m.methodologyRecovery = ""
	m.methodologyDiagnostics = nil
	m.methodologyLogPath = ""
	m.researchDone = false
	m.methodologyRunID++
	runID := m.methodologyRunID
	verb := "Starting"
	if resume {
		verb = "Resuming"
	}
	m.researchLines = append(m.researchLines, fmt.Sprintf("%s %s.", verb, methodologyPhaseLabel(phaseID)))
	m.syncMethodologyLog(true)
	return m, waitMethodologyEvent(run.Events, run.Done, runID)
}

func (m *Model) applyMethodologyEvent(event agent.Event) {
	m.methodologyEventCount++
	m.methodologyLastEventFrame = m.fxFrame
	if len(event.Raw) > 0 {
		m.methodologyRawLog = append(m.methodologyRawLog, string(event.Raw))
	}
	switch event.Kind {
	case "runner_failure":
		if message := strings.TrimSpace(event.Message); message != "" {
			m.methodologyFailureKind = strings.TrimSpace(event.FailureKind)
			m.methodologyPrimaryFailure = message
			m.methodologyRecovery = strings.TrimSpace(event.Recovery)
		}
	case "runner_diagnostic":
		if message := strings.TrimSpace(event.Message); message != "" {
			m.methodologyDiagnostics = append(m.methodologyDiagnostics, message)
		}
	case "runner_cancelled":
		m.methodologyCancelled = true
		m.methodologyPrimaryFailure = strings.TrimSpace(event.Message)
		m.methodologyRecovery = strings.TrimSpace(event.Recovery)
	case "runner_error":
		if m.methodologyPrimaryFailure == "" {
			m.methodologyPrimaryFailure = strings.TrimSpace(event.Message)
			m.methodologyRecovery = "Retry this phase. If it fails again, inspect the full runner log."
		}
	case "runner_done":
		m.methodologyLogPath = strings.TrimSpace(event.LogPath)
	}
	if line := methodologyEventLine(event); line != "" {
		if event.Kind != "tool_done" || !m.replaceMethodologyToolStart(event, line) {
			m.appendMethodologyLine(line)
		}
	}
	if len(m.researchLines) > 200 {
		m.researchLines = m.researchLines[len(m.researchLines)-200:]
	}
	m.syncMethodologyLog(false)
}

func (m *Model) appendMethodologyLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if len(m.researchLines) == 0 {
		m.researchLines = append(m.researchLines, line)
		return
	}
	lastBase, lastCount := splitMethodologyRepeat(m.researchLines[len(m.researchLines)-1])
	if lastBase == line {
		m.researchLines[len(m.researchLines)-1] = formatMethodologyRepeat(line, lastCount+1)
		return
	}
	m.researchLines = append(m.researchLines, line)
}

func (m *Model) replaceMethodologyToolStart(event agent.Event, line string) bool {
	name := cleanToolName(event.Name)
	if name == "" || len(m.researchLines) == 0 {
		return false
	}
	startLine := "Tool started: " + name
	lastBase, _ := splitMethodologyRepeat(m.researchLines[len(m.researchLines)-1])
	if lastBase != startLine {
		return false
	}
	m.researchLines[len(m.researchLines)-1] = line
	if len(m.researchLines) < 2 {
		return true
	}
	previousBase, previousCount := splitMethodologyRepeat(m.researchLines[len(m.researchLines)-2])
	currentBase, currentCount := splitMethodologyRepeat(m.researchLines[len(m.researchLines)-1])
	if previousBase == currentBase {
		m.researchLines[len(m.researchLines)-2] = formatMethodologyRepeat(previousBase, previousCount+currentCount)
		m.researchLines = m.researchLines[:len(m.researchLines)-1]
	}
	return true
}

func (m Model) finishMethodologyPhase(err error) (Model, tea.Cmd) {
	m.methodologyCancel = nil
	m.methodologyEvents = nil
	m.methodologyDone = nil
	if err != nil {
		m.researchDone = true
		m.methodologyLastErr = err.Error()
		m.methodologyRawLog = append(m.methodologyRawLog, "[runner process] "+err.Error())
		if m.methodologyCancelled {
			m.methodologyFailed = false
			m.err = ""
			if m.methodologyPrimaryFailure == "" {
				m.methodologyPrimaryFailure = "AI run cancelled."
			}
			if m.methodologyRecovery == "" {
				m.methodologyRecovery = "Retry this phase when ready, or return to the project."
			}
			m.note = "The project is unchanged. Retry this phase when ready."
			m.appendMethodologyLine("AI run cancelled. Project state was preserved.")
			m.syncMethodologyLog(true)
			return m, nil
		}
		m.methodologyFailed = true
		if m.methodologyPrimaryFailure == "" {
			m.methodologyPrimaryFailure = err.Error()
		}
		if m.methodologyRecovery == "" {
			m.methodologyRecovery = "Retry this phase. If it fails again, inspect the full runner log."
		}
		m.err = ""
		m.note = m.methodologyRecovery
		m.appendMethodologyLine("Corpus Builder paused on error. " + m.methodologyRecovery)
		m.syncMethodologyLog(true)
		return m, nil
	}

	phaseID := m.methodologyPhaseID
	m.methodologyFailed = false
	m.methodologyCancelled = false
	m.methodologyLastErr = ""
	if phaseID == "improvement" {
		m.researchDone = true
		m.researchLines = append(m.researchLines, "Completed Improve Corpus staging.")
		m.syncMethodologyLog(true)
		m.improvementLoading = true
		m.note = "Asking Liner Core to classify the staged Source delta."
		if m.improvementBaseline == nil {
			return m, func() tea.Msg {
				return improvementDeltaPlannedMsg{err: fmt.Errorf("Improve Corpus lost its fixed Core Snapshot; refresh and retry")}
			}
		}
		return m, planImprovementDeltaFromBaselineCommand(m.runner, m.currentPath, *m.improvementBaseline)
	}
	m.researchStep = max(m.researchStep, m.methodologyPhaseIndex+1)
	m.researchLines = append(m.researchLines, fmt.Sprintf("Completed %s.", methodologyPhaseLabel(phaseID)))
	m.recordMethodologyProgress(phaseID)
	m.syncMethodologyLog(true)
	if refreshed, readErr := tape.ReadProject(m.currentPath); readErr == nil {
		m.currentTape = refreshed
	}

	nextIndex := m.methodologyPhaseIndex + 1
	if nextIndex < len(methodologyPhaseOrder) {
		return m.startMethodologyPhase(nextIndex, false)
	}

	m.researchDone = true
	m.methodologyEvents = nil
	m.methodologyDone = nil
	return m.startPreparedAssemblyReview()
}

func (m Model) retryMethodologyPhase() (Model, tea.Cmd) {
	if !m.methodologyFailed && !m.methodologyCancelled {
		return m, nil
	}
	if m.methodologyPhaseID == "improvement" {
		if m.improvementBaseline == nil {
			m.err = "Improve Corpus lost its fixed Core Snapshot. Refresh Project Flow, then retry."
			return m, nil
		}
		if err := prepareImprovementWorkspace(m.currentPath, *m.improvementBaseline); err != nil {
			m.err = "Could not rebuild the isolated improvement workspace: " + err.Error()
			return m, nil
		}
		m.err = ""
		m.note = ""
		m.researchDone = false
		m.methodologyFailed = false
		m.methodologyCancelled = false
		m.methodologyFailureKind = ""
		m.methodologyPrimaryFailure = ""
		m.methodologyRecovery = ""
		m.methodologyDiagnostics = nil
		m.researchLines = append(m.researchLines, "Retrying Improve Corpus in a fresh isolated agent session.")
		m.syncMethodologyLog(true)
		return m.startImprovementAgent(false)
	}
	index := m.methodologyPhaseIndex
	if index < 0 || index >= len(methodologyPhaseOrder) {
		index = m.researchStep
	}
	if index < 0 || index >= len(methodologyPhaseOrder) {
		m.err = "No failed corpus phase is available to retry."
		return m, nil
	}
	m.err = ""
	m.note = ""
	m.researchDone = false
	m.methodologyFailed = false
	m.methodologyCancelled = false
	m.methodologyFailureKind = ""
	m.methodologyPrimaryFailure = ""
	m.methodologyRecovery = ""
	m.methodologyDiagnostics = nil
	m.methodologyPhaseID = methodologyPhaseOrder[index]
	m.researchLines = append(m.researchLines, fmt.Sprintf("Retrying %s from the saved agent session.", methodologyPhaseLabel(m.methodologyPhaseID)))
	m.syncMethodologyLog(true)
	return m.startMethodologyPhase(index, true)
}

func (m Model) retrySourceEvaluation() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "No project is open."
		return m, nil
	}
	m.stopMethodology("")
	corpusPath := projectCorpusPath(m.currentPath)
	index, ok := methodologyIndexForProgressPhase(linerprogress.PhaseEvaluation)
	if !ok {
		m.err = "Source evaluation phase is not available."
		return m, nil
	}
	if err := linerprogress.Write(corpusPath, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseEvaluation)}); err != nil {
		m.err = "Could not reset source evaluation progress: " + err.Error()
		return m, nil
	}
	m.err = ""
	m.note = "Refreshing source evaluation."
	m.screen = screenResearch
	m.researchDone = false
	m.methodologyFailed = false
	m.methodologyCancelled = false
	m.methodologyLastErr = ""
	m.methodologyFailureKind = ""
	m.methodologyPrimaryFailure = ""
	m.methodologyRecovery = ""
	m.methodologyDiagnostics = nil
	m.methodologyRawLog = nil
	m.methodologyLogPath = ""
	m.methodologyPhaseID = ""
	m.methodologyEventCount = 0
	m.methodologyLastEventFrame = m.fxFrame
	m.researchStep = index
	m.methodologyPhaseIndex = index
	m.researchLines = []string{
		"Refreshing source evaluation.",
		"Queued Evaluation through Assembly. Framing and Candidate discovery are not rerunning.",
	}
	m.ensureBoardItems()
	m.syncMethodologyLog(true)
	return m.startMethodologyPhase(index, false)
}

func (m Model) shouldResumeMethodologyPhase(index int) bool {
	if index < 0 || index >= len(methodologyPhaseOrder) || strings.TrimSpace(m.currentPath) == "" {
		return false
	}
	entries, err := os.ReadDir(projectAbsPath(m.currentPath, filepath.Join(".liner-runs", methodologyPhaseOrder[index])))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			return true
		}
	}
	return false
}

func (m *Model) recordMethodologyProgress(phaseID string) {
	phase := linerprogress.PhaseID(phaseID)
	if phase == linerprogress.PhaseAssembly {
		m.researchLines = append(m.researchLines, "Assembly draft written. Progress waits for draft review acceptance.")
		return
	}
	if _, err := linerprogress.MarkPhaseComplete(projectCorpusPath(m.currentPath), phase); err != nil {
		m.researchLines = append(m.researchLines, "Could not update progress: "+err.Error())
		return
	}
	if gate, ok := linerprogress.GateAfterPhase(phase); ok {
		if _, err := linerprogress.AcceptGate(projectCorpusPath(m.currentPath), gate); err != nil {
			m.researchLines = append(m.researchLines, "Could not accept corpus checkpoint: "+err.Error())
			return
		}
		m.researchLines = append(m.researchLines, fmt.Sprintf("Accepted %s for the continuous corpus build.", gate))
	}
}

func (m *Model) stopMethodology(reason string) {
	m.methodologyRunID++
	if m.methodologyCancel != nil {
		m.methodologyCancel()
		m.methodologyCancel = nil
	}
	m.methodologyEvents = nil
	m.methodologyDone = nil
	if strings.TrimSpace(reason) != "" {
		m.researchDone = true
		m.methodologyFailed = false
		m.methodologyCancelled = true
		m.methodologyPrimaryFailure = "AI run cancelled."
		m.methodologyRecovery = "Retry this phase when ready, or return to the project."
		m.err = ""
		m.note = "AI run cancelled. Project state was preserved."
		m.researchLines = append(m.researchLines, reason)
		m.syncMethodologyLog(true)
	}
}

func waitMethodologyEvent(events <-chan agent.Event, done <-chan error, runID uint64) tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-events:
			if ok {
				return methodologyEventMsg{event: event, runID: runID}
			}
			if done == nil {
				return methodologyDoneMsg{runID: runID}
			}
			err, _ := <-done
			return methodologyDoneMsg{err: err, runID: runID}
		case err, ok := <-done:
			if !ok {
				return methodologyDoneMsg{runID: runID}
			}
			return methodologyDoneMsg{err: err, runID: runID}
		}
	}
}

func methodologyEventLine(event agent.Event) string {
	switch event.Kind {
	case "runner_start":
		verb := "Running"
		if event.Resume {
			verb = "Resuming"
		}
		return fmt.Sprintf("%s %s with %s.", verb, methodologyPhaseLabel(event.PhaseID), event.Agent)
	case "runner_error":
		return "Runner error: " + firstNonEmptyLine(event.Message)
	case "runner_failure":
		return "Runner failed: " + firstNonEmptyLine(event.Message)
	case "runner_diagnostic":
		return "Diagnostic: " + firstNonEmptyLine(event.Message)
	case "runner_cancelled":
		return "AI run cancelled."
	case "runner_done":
		if event.Code != nil && *event.Code != 0 {
			return fmt.Sprintf("Runner exited with code %d.", *event.Code)
		}
		return "Runner finished."
	case "text":
		return firstNonEmptyLine(event.Text)
	case "tool_start":
		return "Tool started: " + cleanToolName(event.Name)
	case "tool_done":
		status := "finished"
		if event.OK != nil && !*event.OK {
			status = "failed"
		}
		if event.Preview != "" {
			return fmt.Sprintf("Tool %s: %s.", status, firstNonEmptyLine(event.Preview))
		}
		if name := cleanToolName(event.Name); name != "" {
			return fmt.Sprintf("Tool %s: %s.", status, name)
		}
		return "Tool " + status + "."
	case "summary":
		return firstNonEmptyLine(event.FinalText)
	case "rate_limit":
		return "Rate limit: " + event.Status
	case "raw":
		if isNoisyAgentStderr(event.Text) {
			return ""
		}
		return firstNonEmptyLine(event.Text)
	default:
		return ""
	}
}

func (m Model) openMethodologyFullLog() (Model, tea.Cmd) {
	lines := m.methodologyRawLog
	if path := m.safeMethodologyLogPath(); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			lines = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		}
	}
	if len(lines) == 0 {
		lines = m.researchLines
	}
	m.previewBack = m.screen
	m.hasPreviewBack = true
	m.previewRel = "Corpus Builder full log"
	m.preview.SetContent(strings.Join(lines, "\n"))
	m.preview.GotoTop()
	m.screen = screenPreview
	return m, nil
}

func (m Model) safeMethodologyLogPath() string {
	path := filepath.Clean(strings.TrimSpace(m.methodologyLogPath))
	if path == "." || path == "" || strings.TrimSpace(m.currentPath) == "" {
		return ""
	}
	root := filepath.Clean(projectCorpusPath(m.currentPath))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return path
}

func isNoisyAgentStderr(line string) bool {
	value := strings.ToLower(strings.TrimSpace(line))
	if value == "" {
		return true
	}
	if isCosmeticCodexStartupWarning(line) {
		return true
	}
	if strings.Contains(value, "failed to load skill") {
		return true
	}
	if strings.Contains(value, "rmcp::transport::auth") {
		return true
	}
	if strings.Contains(value, "rmcp::transport::worker") && strings.Contains(value, "transport channel closed") {
		return true
	}
	if strings.Contains(value, "oauth") && (strings.Contains(value, "mcp") || strings.Contains(value, "connector")) {
		return true
	}
	if strings.Contains(value, "write_stdin failed: stdin is closed") {
		return true
	}
	return false
}

func isCosmeticCodexStartupWarning(line string) bool {
	return strings.Contains(line, "codex_core_plugins::manifest: ignoring interface.defaultPrompt") ||
		strings.Contains(line, "codex_core_skills::loader: ignoring interface.icon_small") ||
		strings.Contains(line, "codex_core_skills::loader: ignoring interface.icon_large")
}

func splitMethodologyRepeat(line string) (string, int) {
	line = strings.TrimSpace(line)
	index := strings.LastIndex(line, " (x")
	if index < 0 || !strings.HasSuffix(line, ")") {
		return line, 1
	}
	countText := strings.TrimSuffix(line[index+3:], ")")
	count, err := strconv.Atoi(countText)
	if err != nil || count < 2 {
		return line, 1
	}
	return line[:index], count
}

func formatMethodologyRepeat(line string, count int) string {
	if count <= 1 {
		return line
	}
	return fmt.Sprintf("%s (x%d)", line, count)
}

func (m Model) nextMethodologyPhaseIndex() (int, []string, error) {
	var notes []string
	if strings.TrimSpace(m.currentPath) == "" {
		return 0, notes, nil
	}
	for {
		corpusPath := projectCorpusPath(m.currentPath)
		current := linerprogress.Read(corpusPath)
		if current.Step >= len(linerprogress.PhaseOrder) {
			return len(methodologyPhaseOrder), notes, nil
		}
		phase := linerprogress.PhaseOrder[current.Step]
		if note, recovered := recoverCompletedMethodologyPhase(corpusPath, phase); recovered {
			notes = append(notes, note)
			continue
		}
		if index, ok := methodologyIndexForProgressPhase(phase); ok {
			return index, notes, nil
		}
		if !isProgressGate(phase) {
			return 0, notes, fmt.Errorf("unknown corpus build step %s", phase)
		}
		if _, err := linerprogress.AcceptGate(corpusPath, phase); err != nil {
			return 0, notes, err
		}
		notes = append(notes, fmt.Sprintf("Accepted pending %s for the continuous corpus build.", phase))
	}
}

func recoverCompletedMethodologyPhase(corpusPath string, phase linerprogress.PhaseID) (string, bool) {
	if phase != linerprogress.PhaseEvaluation || !validCompletedEvaluationArtifact(corpusPath) {
		return "", false
	}
	before := linerprogress.Read(corpusPath)
	after, err := linerprogress.MarkPhaseComplete(corpusPath, phase)
	if err != nil || after.Step <= before.Step {
		return "", false
	}
	return "Recovered completed Evaluation from working/03-evaluation.yaml.", true
}

type methodologyEvaluationArtifact struct {
	Candidates []methodologyEvaluationCandidate `yaml:"candidates"`
}

type methodologyEvaluationCandidate struct {
	URL            string   `yaml:"url"`
	Title          string   `yaml:"title"`
	Decision       string   `yaml:"decision"`
	Section        string   `yaml:"section"`
	Rationale      string   `yaml:"rationale"`
	FetchStatus    string   `yaml:"fetch_status"`
	ContentQuality string   `yaml:"content_quality"`
	Evidence       []string `yaml:"evidence"`
}

func validCompletedEvaluationArtifact(corpusPath string) bool {
	data, err := os.ReadFile(filepath.Join(corpusPath, "working", "03-evaluation.yaml"))
	if err != nil {
		return false
	}
	var artifact methodologyEvaluationArtifact
	if err := yaml.Unmarshal(data, &artifact); err != nil {
		return false
	}
	if len(artifact.Candidates) == 0 {
		return false
	}
	expected := countUniqueURLs(filepath.Join(corpusPath, "working", "02-candidate-longlist.md"))
	seen := map[string]bool{}
	for _, candidate := range artifact.Candidates {
		url := normalizeMethodologyURL(candidate.URL)
		if url == "" || seen[url] || !validEvaluationDecision(candidate) {
			return false
		}
		seen[url] = true
	}
	return expected == 0 || len(seen) >= expected
}

func validEvaluationDecision(candidate methodologyEvaluationCandidate) bool {
	decision := strings.ToLower(strings.TrimSpace(candidate.Decision))
	switch decision {
	case "dropped":
		return true
	case "kept", "trim", "trimmed":
		fetchStatus := strings.ToLower(strings.TrimSpace(candidate.FetchStatus))
		contentQuality := strings.ToLower(strings.TrimSpace(candidate.ContentQuality))
		return (fetchStatus == "readable" || fetchStatus == "partial") &&
			(contentQuality == "high" || contentQuality == "medium") &&
			len(candidate.Evidence) >= 2
	default:
		return false
	}
}

var methodologyURLPattern = regexp.MustCompile(`https?://[^\s)\]>"']+`)

func countUniqueURLs(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	seen := map[string]bool{}
	for _, match := range methodologyURLPattern.FindAllString(string(data), -1) {
		if url := normalizeMethodologyURL(match); url != "" {
			seen[url] = true
		}
	}
	return len(seen)
}

func normalizeMethodologyURL(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimRight(value, ".,;")
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return ""
	}
	return value
}

func methodologyIndexForProgressPhase(phase linerprogress.PhaseID) (int, bool) {
	switch phase {
	case linerprogress.PhaseFraming:
		return 0, true
	case linerprogress.PhaseCandidates:
		return 1, true
	case linerprogress.PhaseEvaluation:
		return 2, true
	case linerprogress.PhaseQuality:
		return 3, true
	case linerprogress.PhaseSynthesis:
		return 4, true
	case linerprogress.PhaseAssembly:
		return 5, true
	case linerprogress.PhaseCompile:
		return len(methodologyPhaseOrder), true
	default:
		return 0, false
	}
}

func isProgressGate(phase linerprogress.PhaseID) bool {
	return phase == linerprogress.PhaseGate0 || phase == linerprogress.PhaseGate1 || phase == linerprogress.PhaseGate2
}

func (m Model) researchPercent() float64 {
	steps := len(methodologyPhaseOrder)
	if steps == 0 {
		return 0
	}
	if m.researchDone && m.researchStep >= steps {
		return 1
	}
	return float64(min(m.researchStep, steps)) / float64(steps)
}

func (m Model) researchPhaseLabels() []string {
	labels := make([]string, 0, len(methodologyPhaseOrder))
	for _, phase := range methodologyPhaseOrder {
		labels = append(labels, methodologyPhaseLabel(phase))
	}
	return labels
}

func (m Model) researchPhaseArtifacts() []string {
	artifacts := make([]string, 0, len(methodologyPhaseOrder))
	for _, phase := range methodologyPhaseOrder {
		artifacts = append(artifacts, methodologyArtifacts[phase])
	}
	return artifacts
}

func (m Model) hasReviewableSources() bool {
	return len(m.sourceItems) > 0 || len(m.currentTape.Sources) > 0
}

func (m *Model) ensureBoardItems() {
	if len(m.sourceItems) > 0 || len(m.currentTape.Sources) == 0 {
		return
	}
	m.sourceItems = source.Stage(m.currentTape.Sources, true)
	m.applySourceItems(m.sourceItems)
}

func methodologyPhaseLabel(phase string) string {
	if label := methodologyPhaseNames[phase]; label != "" {
		return label
	}
	return phase
}

func tapeMode(current tape.Tape) string {
	if current.Mode == nil || strings.TrimSpace(*current.Mode) == "" {
		return "quick"
	}
	return strings.TrimSpace(*current.Mode)
}

func resolveHeadlessRunnerScript() string {
	if env := os.Getenv("LINER_HEADLESS_RUNNER"); env != "" && pathExists(env) {
		return env
	}

	var roots []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		roots = append(roots, exeDir, filepath.Dir(exeDir))
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 12; i++ {
			roots = append(roots, dir)
			next := filepath.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
	}

	for _, root := range roots {
		for _, rel := range []string{
			filepath.Join("dist", "agents", "headless-runner.js"),
			filepath.Join("..", "dist", "agents", "headless-runner.js"),
			filepath.Join("packages", "tui", "dist", "agents", "headless-runner.js"),
		} {
			candidate := filepath.Clean(filepath.Join(root, rel))
			if pathExists(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func cleanToolName(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "mcp__") {
		parts := strings.Split(name, "__")
		return parts[len(parts)-1]
	}
	return name
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len([]rune(line)) > 160 {
				runes := []rune(line)
				return string(runes[:159]) + "..."
			}
			return line
		}
	}
	return ""
}
