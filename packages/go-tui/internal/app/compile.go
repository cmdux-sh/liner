package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	linerprogress "github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const (
	compilePaneIssues = iota
	compilePaneSources
)

func (m Model) startCompile() (Model, tea.Cmd) {
	events, done := m.runner.StartCompile(m.currentPath)
	m.compileEvents = events
	m.compileDone = done
	m.compileLines = []string{"Starting compile..."}
	m.compiling = true
	m.compileTotal = len(m.currentTape.Sources)
	m.compileDoneN = 0
	m.compileFailed = 0
	m.compileResult = nil
	m.compileWarningIndex = 0
	m.compilePane = compilePaneIssues
	m.compileSourceIndex = 0
	m.compileSourcesReviewed = false
	m.compileRepairAttempted = false
	m.compileRepairRetryCompileAfterRecovery = false
	m.compileRepairRebuildCorpusAfterRecovery = false
	m.compileErr = ""
	m.sourceRecoveryReview = false
	m.compileRows = initialCompileRows(m.currentTape.Sources)
	m.compileBar = newCompileProgress(compileProgressWidth(m.width))
	m.screen = screenCompile
	return m, tea.Batch(m.compileBar.SetPercent(0), waitCompileEvent(events, done))
}

func (m Model) startCompileReviewFromArtifacts() (Model, tea.Cmd) {
	health := projectCompileArtifactHealth(m.currentPath, m.currentTape.Sources, len(m.currentTape.Sources))
	m.screen = screenCompile
	m.compiling = false
	m.compileErr = ""
	m.compileWarningIndex = 0
	m.compilePane = compilePaneIssues
	m.compileSourceIndex = 0
	m.compileSourcesReviewed = false
	m.compileRepairAttempted = false
	m.compileRepairRetryCompileAfterRecovery = false
	m.compileRepairRebuildCorpusAfterRecovery = false
	m.sourceRecoveryReview = false
	m.compileRows = initialCompileRows(m.currentTape.Sources)
	for _, sourceIndex := range health.UnavailableIndexes {
		if sourceIndex < 0 || sourceIndex >= len(m.compileRows) {
			continue
		}
		m.compileRows[sourceIndex].Status = "failed"
		m.compileRows[sourceIndex].Detail = compileArtifactWarningForIndex(health.Warnings, sourceIndex, m.compileRows[sourceIndex].Source)
	}
	m.compileTotal = health.Summary.Total
	m.compileDoneN = health.Summary.Succeeded + health.Summary.Failed
	m.compileFailed = health.Summary.Failed
	m.compileResult = &core.CompileResultPayload{
		MixtapePath: projectAbsPath(m.currentPath, "MIXTAPE.md"),
		Warnings:    health.Warnings,
		Summary:     health.Summary,
	}
	m.compileLines = []string{
		fmt.Sprintf("Previous compile needs attention: %d/%d usable sources.", health.Summary.Succeeded, health.Summary.Total),
		"Review the source issues, then install JS rendering, replace sources, drop sources, or retry compile.",
	}
	m.compileBar = newCompileProgress(compileProgressWidth(m.width))
	return m, m.compileBar.SetPercent(1)
}

func (m *Model) applyCompileEvent(event core.CompileEvent) tea.Cmd {
	var cmd tea.Cmd
	switch event.Type {
	case "start":
		m.compileTotal = event.Total
		if event.Total == 0 {
			m.compileLines = append(m.compileLines, "No sources attached; packaging synthesis only.")
		}
		m.compileLines = append(m.compileLines, fmt.Sprintf("Fetching %d sources", event.Total))
		cmd = m.compileBar.SetPercent(m.compilePercent())
	case "source_start":
		index := m.nextCompileRow()
		label := compileSpecLabel(event.Spec)
		if index >= len(m.compileRows) {
			m.compileRows = append(m.compileRows, compileSourceRow{Status: "running", Type: compileSpecType(event.Spec), Source: label})
		} else {
			m.compileRows[index].Status = "running"
			if label != "" {
				m.compileRows[index].Source = label
			}
			if event.Spec != nil && event.Spec.Type != "" {
				m.compileRows[index].Type = event.Spec.Type
			}
		}
		m.compileLines = append(m.compileLines, "→ fetching "+fallbackText(label, "source"))
	case "source_done", "source_cached":
		title := ""
		if event.Title != nil {
			title = *event.Title
		}
		index := m.runningCompileRow()
		if index >= 0 {
			if event.Type == "source_cached" {
				m.compileRows[index].Status = "cached"
			} else {
				m.compileRows[index].Status = "done"
			}
			m.compileRows[index].Detail = fmt.Sprintf("%d chars", event.BodyChars)
			if title != "" {
				m.compileRows[index].Source = title
			}
		}
		m.compileDoneN++
		m.compileLines = append(m.compileLines, fmt.Sprintf("✓ %s (%d chars)", title, event.BodyChars))
		cmd = m.compileBar.SetPercent(m.compilePercent())
	case "source_failed":
		index := m.runningCompileRow()
		if index >= 0 {
			m.compileRows[index].Status = "failed"
			m.compileRows[index].Detail = event.Message
		}
		m.compileDoneN++
		m.compileFailed++
		m.compileLines = append(m.compileLines, "× "+event.Message)
		cmd = m.compileBar.SetPercent(m.compilePercent())
	case "finish":
		m.compileLines = append(m.compileLines, "Assembling MIXTAPE.md...")
	case "result":
		m.compileResult = event.Payload
		m.clampCompileWarningIndex()
		if event.Payload != nil {
			m.compileTotal = event.Payload.Summary.Total
			m.compileDoneN = event.Payload.Summary.Succeeded + event.Payload.Summary.Failed
			m.compileFailed = event.Payload.Summary.Failed
			m.compileLines = append(m.compileLines, fmt.Sprintf("Result: %d/%d usable sources", event.Payload.Summary.Succeeded, event.Payload.Summary.Total))
		} else {
			m.compileLines = append(m.compileLines, "Result received.")
		}
		cmd = m.compileBar.SetPercent(1)
	}
	if len(m.compileLines) > 200 {
		m.compileLines = m.compileLines[len(m.compileLines)-200:]
	}
	return cmd
}

func initialCompileRows(sources []tape.Source) []compileSourceRow {
	rows := make([]compileSourceRow, 0, len(sources))
	for _, src := range sources {
		rows = append(rows, compileSourceRow{
			Status: "queued",
			Type:   src.Type,
			Source: compileTapeSourceLabel(src),
		})
	}
	return rows
}

func (m Model) compilePercent() float64 {
	if m.compileResult != nil {
		return 1
	}
	if m.compileTotal <= 0 {
		if m.compiling {
			return 0.12
		}
		if len(m.compileLines) > 0 {
			return 1
		}
		return 0
	}
	return float64(min(m.compileDoneN, m.compileTotal)) / float64(m.compileTotal)
}

func (m Model) compileHasUsableResult() bool {
	return m.compileResult != nil && m.compileResult.Summary.Succeeded > 0
}

type compileArtifactHealth struct {
	HasMixtape         bool
	NeedsAttention     bool
	UnavailableIndexes []int
	Warnings           []core.CompileWarningPayload
	Summary            core.CompileSummary
}

type indexedCompileWarning struct {
	Index   int
	Warning core.CompileWarningPayload
}

type compileSourceListItem struct {
	Status     string
	Kind       string
	Type       string
	Source     string
	Detail     string
	OpenTarget string
}

func (m Model) actionableCompileWarnings() []indexedCompileWarning {
	if m.compileResult == nil {
		return nil
	}
	warnings := []indexedCompileWarning{}
	for index, warning := range m.compileResult.Warnings {
		if compileWarningBlocksProgress(warning) {
			warnings = append(warnings, indexedCompileWarning{Index: index, Warning: warning})
		}
	}
	return warnings
}

func (m Model) actionableCompileWarningCount() int {
	return len(m.actionableCompileWarnings())
}

func (m Model) recoveredCompileWarningCount() int {
	if m.compileResult == nil {
		return 0
	}
	count := 0
	for _, warning := range m.compileResult.Warnings {
		if compileWarningRecoveredWithJS(warning) {
			count++
		}
	}
	return count
}

func projectCompileArtifactsNeedAttention(project string, sources []tape.Source, sourceCount int) bool {
	return projectCompileArtifactHealth(project, sources, sourceCount).NeedsAttention
}

func projectCompileArtifactHealth(project string, sources []tape.Source, sourceCount int) compileArtifactHealth {
	hasMixtape := projectFileExists(project, "MIXTAPE.md")
	total := sourceCount
	if len(sources) > 0 {
		total = len(sources)
	}
	warnings := compileArtifactWarnings(project)
	unavailable := unavailableCompiledSourceIndexes(project)
	for _, index := range unavailable {
		if index >= total {
			total = index + 1
		}
		if !compileWarningsContainSourceIndex(warnings, index, sources) {
			warnings = append(warnings, core.CompileWarningPayload{
				URL:      compileArtifactSourceLabel(index, sources),
				Severity: "error",
				Message:  "Compiled source file is unavailable. See compilation notes in MIXTAPE.md.",
			})
		}
	}
	failed := len(unavailable)
	if total == 0 && len(warnings) > 0 {
		total = len(warnings)
	}
	succeeded := max(0, total-failed)
	needsAttention := failed > 0
	for _, warning := range warnings {
		if compileWarningBlocksProgress(warning) {
			needsAttention = true
			break
		}
	}
	return compileArtifactHealth{
		HasMixtape:         hasMixtape,
		NeedsAttention:     hasMixtape && needsAttention,
		UnavailableIndexes: unavailable,
		Warnings:           warnings,
		Summary: core.CompileSummary{
			Total:     total,
			Succeeded: succeeded,
			Failed:    failed,
		},
	}
}

func compileArtifactWarnings(project string) []core.CompileWarningPayload {
	body, err := os.ReadFile(projectAbsPath(project, "MIXTAPE.md"))
	if err != nil {
		return nil
	}
	text := string(body)
	_, notes, ok := strings.Cut(text, "## Compilation notes")
	if !ok {
		return nil
	}
	warnings := []core.CompileWarningPayload{}
	for _, line := range strings.Split(notes, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- **") {
			continue
		}
		rest := strings.TrimPrefix(line, "- **")
		url, message, ok := strings.Cut(rest, "**")
		if !ok {
			continue
		}
		message = strings.TrimSpace(strings.TrimPrefix(message, "—"))
		warnings = append(warnings, core.CompileWarningPayload{
			URL:      strings.TrimSpace(url),
			Message:  message,
			Severity: compileArtifactWarningSeverity(message),
		})
	}
	return warnings
}

func compileArtifactWarningSeverity(message string) string {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "failed") ||
		strings.Contains(lower, "error") ||
		strings.Contains(lower, "unavailable") ||
		strings.Contains(lower, "js_required") ||
		strings.Contains(lower, "http 4") ||
		strings.Contains(lower, "http 5") {
		return "error"
	}
	return "warning"
}

func unavailableCompiledSourceIndexes(project string) []int {
	entries, err := os.ReadDir(projectAbsPath(project, "sources"))
	if err != nil {
		return nil
	}
	indexes := []int{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		body, err := os.ReadFile(projectAbsPath(project, filepath.Join("sources", entry.Name())))
		if err != nil || !strings.Contains(string(body), "_Source unavailable") {
			continue
		}
		if index, ok := compiledSourceIndexFromFilename(entry.Name()); ok {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func compiledSourceIndexFromFilename(name string) (int, bool) {
	prefix, _, ok := strings.Cut(name, "-")
	if !ok {
		return -1, false
	}
	number, err := strconv.Atoi(prefix)
	if err != nil || number <= 0 {
		return -1, false
	}
	return number - 1, true
}

func compileWarningsContainSourceIndex(warnings []core.CompileWarningPayload, sourceIndex int, sources []tape.Source) bool {
	if sourceIndex < 0 || sourceIndex >= len(sources) {
		return false
	}
	sourceID := strings.TrimSpace(compileTapeSourceLabel(sources[sourceIndex]))
	if sourceID == "" {
		return false
	}
	for _, warning := range warnings {
		if strings.TrimSpace(warning.URL) == sourceID || strings.Contains(warning.Message, sourceID) {
			return true
		}
	}
	return false
}

func compileArtifactSourceLabel(sourceIndex int, sources []tape.Source) string {
	if sourceIndex >= 0 && sourceIndex < len(sources) {
		return compileTapeSourceLabel(sources[sourceIndex])
	}
	return fmt.Sprintf("source %02d", sourceIndex+1)
}

func compileArtifactWarningForIndex(warnings []core.CompileWarningPayload, sourceIndex int, fallback string) string {
	sourceID := strings.TrimSpace(fallback)
	for _, warning := range warnings {
		if sourceID != "" && (strings.TrimSpace(warning.URL) == sourceID || strings.Contains(warning.Message, sourceID)) {
			return warning.Message
		}
	}
	return "Compiled source unavailable. Review compile issues, then retry."
}

func (m *Model) clampCompileWarningIndex() {
	if m.compileResult == nil || len(m.compileResult.Warnings) == 0 {
		m.compileWarningIndex = 0
		return
	}
	actionable := m.actionableCompileWarnings()
	if len(actionable) == 0 {
		if m.compileWarningIndex < 0 {
			m.compileWarningIndex = 0
		}
		if m.compileWarningIndex >= len(m.compileResult.Warnings) {
			m.compileWarningIndex = len(m.compileResult.Warnings) - 1
		}
		return
	}
	for _, item := range actionable {
		if item.Index == m.compileWarningIndex {
			return
		}
	}
	m.compileWarningIndex = actionable[0].Index
}

func (m Model) selectedCompileWarning() (core.CompileWarningPayload, bool) {
	if m.compileResult == nil {
		return core.CompileWarningPayload{}, false
	}
	for _, item := range m.actionableCompileWarnings() {
		if item.Index == m.compileWarningIndex {
			return item.Warning, true
		}
	}
	if actionable := m.actionableCompileWarnings(); len(actionable) > 0 {
		return actionable[0].Warning, true
	}
	return core.CompileWarningPayload{}, false
}

func (m Model) moveCompileWarningSelection(delta int) Model {
	actionable := m.actionableCompileWarnings()
	if len(actionable) == 0 {
		return m
	}
	selected := 0
	for index, item := range actionable {
		if item.Index == m.compileWarningIndex {
			selected = index
			break
		}
	}
	selected = min(max(selected+delta, 0), len(actionable)-1)
	m.compileWarningIndex = actionable[selected].Index
	return m
}

func (m Model) toggleCompilePane() Model {
	if m.compilePane == compilePaneSources {
		m.compilePane = compilePaneIssues
		return m
	}
	m.compilePane = compilePaneSources
	m.compileSourcesReviewed = true
	m.compileSourceIndex = clampCompileSourceIndex(m.compileSourceIndex, len(m.compileSourceListItems()))
	return m
}

func (m Model) moveCompileSourceSelection(delta int) Model {
	items := m.compileSourceListItems()
	m.compileSourceIndex = clampCompileSourceIndex(m.compileSourceIndex+delta, len(items))
	return m
}

func (m Model) selectedCompileSource() (compileSourceListItem, bool) {
	items := m.compileSourceListItems()
	if len(items) == 0 {
		return compileSourceListItem{}, false
	}
	index := clampCompileSourceIndex(m.compileSourceIndex, len(items))
	return items[index], true
}

func (m Model) openSelectedCompileSource() (Model, tea.Cmd) {
	item, ok := m.selectedCompileSource()
	if !ok || strings.TrimSpace(item.OpenTarget) == "" {
		m.err = "No source is selected."
		return m, nil
	}
	target := strings.TrimSpace(item.OpenTarget)
	if parsed, err := url.Parse(target); err == nil && parsed.Scheme != "" {
		return m, openPath(target)
	}
	if !filepath.IsAbs(target) {
		target = projectAbsPath(m.currentPath, target)
	}
	return m, openPath(target)
}

func (m Model) openSelectedCompileWarningSource() (Model, tea.Cmd) {
	warning, ok := m.selectedCompileWarning()
	if !ok || strings.TrimSpace(warning.URL) == "" {
		m.err = "No source issue is selected."
		return m, nil
	}
	m.note = "Opening source issue."
	return m, openPath(strings.TrimSpace(warning.URL))
}

func (m Model) dropSelectedCompileWarningSource() (Model, tea.Cmd) {
	warning, ok := m.selectedCompileWarning()
	if !ok {
		m.err = "No source issue is selected."
		return m, nil
	}
	sourceID := strings.TrimSpace(warning.URL)
	if sourceID == "" {
		m.err = "The selected source issue does not identify a source."
		return m, nil
	}
	updated, removed := dropTapeSourceByID(m.currentTape, sourceID)
	if removed == 0 {
		m.err = "No matching source found in tape.yaml for " + sourceID + "."
		return m, nil
	}
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "No project path is available for updating tape.yaml."
		return m, nil
	}
	if err := tape.WriteProject(m.currentPath, updated); err != nil {
		m.err = "Could not update tape.yaml: " + err.Error()
		return m, nil
	}
	m.currentTape = updated
	m.compileRows = removeCompileRowsForSource(m.compileRows, sourceID)
	m.note = fmt.Sprintf("Dropped %d source(s) from tape.yaml. Retry compile to refresh the result.", removed)
	return m, nil
}

func dropTapeSourceByID(current tape.Tape, sourceID string) (tape.Tape, int) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return current, 0
	}
	next := current
	next.Sources = make([]tape.Source, 0, len(current.Sources))
	removed := 0
	for _, src := range current.Sources {
		if tapeSourceMatchesID(src, sourceID) {
			removed++
			continue
		}
		next.Sources = append(next.Sources, src)
	}
	return next, removed
}

func tapeSourceMatchesID(src tape.Source, sourceID string) bool {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return false
	}
	values := []string{src.URL}
	if src.Path != nil {
		values = append(values, *src.Path)
	}
	if src.Citation != nil {
		values = append(values, *src.Citation)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == sourceID {
			return true
		}
	}
	return false
}

func removeCompileRowsForSource(rows []compileSourceRow, sourceID string) []compileSourceRow {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return rows
	}
	next := make([]compileSourceRow, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Source) == sourceID {
			continue
		}
		next = append(next, row)
	}
	return next
}

func (m *Model) recordCompileProgress() {
	if strings.TrimSpace(m.currentPath) == "" || !m.compileHasUsableResult() {
		return
	}
	if _, err := linerprogress.MarkPhaseComplete(projectCorpusPath(m.currentPath), linerprogress.PhaseCompile); err != nil {
		m.compileLines = append(m.compileLines, "Could not update progress: "+err.Error())
	}
}

func (m Model) compileAttentionItems() []string {
	var items []string
	if m.compileResult != nil && m.compileResult.Summary.Total == 0 {
		if m.currentTape.JTBD != nil && strings.TrimSpace(*m.currentTape.JTBD) != "" {
			items = append(items, "No saved sources are attached yet. Build Corpus and save the draft sources before relying on this mixtape.")
		} else {
			items = append(items, "No sources are attached. Add sources or define the job before relying on this mixtape.")
		}
	}
	if issue := synthesisAttention(m.currentPath); issue != "" {
		items = append(items, issue)
	}
	return items
}

func (m Model) compileSourceEvaluationSummary() (evaluationIssueSummary, bool) {
	summary, ok := readEvaluationIssueSummary(m.currentPath, m.currentTape)
	return summary, ok && summary.HasIssues()
}

func (m Model) compileHasRetryableSourceEvaluationIssues() bool {
	summary, ok := m.compileSourceEvaluationSummary()
	return ok && (summary.DroppedCustom > 0 || summary.MissingCustom > 0 || summary.AcceptedIssues > 0)
}

func (m Model) compileHasDroppedCustomSourceIssues() bool {
	return len(readDroppedCustomURLSources(m.currentPath, m.currentTape)) > 0
}

func (m Model) compileHasSourceReviewItems() bool {
	if m.actionableCompileWarningCount() > 0 {
		return true
	}
	if summary, ok := m.compileSourceEvaluationSummary(); ok && summary.HasIssues() {
		return true
	}
	return false
}

func (m Model) compileHasRepairableSources() bool {
	return m.compileNeedsJSSetup() || m.actionableCompileWarningCount() > 0 || m.compileHasDroppedCustomSourceIssues()
}

func (m Model) compileNextActionLabel() string {
	switch {
	case m.compiling:
		return "Wait for compile to finish."
	case m.sourceRecoveryRunning:
		return "Wait for unavailable source retry to finish."
	case m.sourceRecoveryReview:
		if m.compileRepairRebuildCorpusAfterRecovery {
			return "Rebuild corpus."
		}
		if m.compileRepairAttempted && m.sourceRecovery != nil && m.sourceRecovery.Succeeded == 0 {
			return "View sources."
		}
		return "Continue to Compile Console."
	case m.jsSetupRunning:
		return "Wait for JS rendering setup to finish."
	case m.compilePane == compilePaneIssues && m.compileHasSourceReviewItems():
		return "View sources."
	case m.compilePane == compilePaneSources && !m.compileRepairAttempted && m.compileHasRepairableSources():
		return "Repair sources."
	case !m.compileHasUsableResult():
		if action := m.compileAttentionNextAction(); action != "" {
			return action
		}
		return "Compile MIXTAPE.md before continuing."
	default:
		return "Create the Operating Layer."
	}
}

func (m Model) viewCompileSourcesNext() Model {
	m.compilePane = compilePaneSources
	m.compileSourcesReviewed = true
	m.compileSourceIndex = clampCompileSourceIndex(m.compileSourceIndex, len(m.compileSourceListItems()))
	m.note = "Review sources, then repair them."
	m.err = ""
	return m
}

func (m Model) repairCompileSources() (Model, tea.Cmd) {
	m.compileSourcesReviewed = true
	m.compileRepairAttempted = true
	m.err = ""
	if m.compileNeedsJSSetup() {
		next, cmd := m.startJSSetupForCompile()
		next.compileSourcesReviewed = true
		next.compileRepairAttempted = true
		return next, cmd
	}
	if m.compileHasDroppedCustomSourceIssues() {
		next, cmd := m.retryExcludedLocalSources()
		next.compileSourcesReviewed = true
		next.compileRepairAttempted = true
		next.compileRepairRetryCompileAfterRecovery = true
		return next, cmd
	}
	return m.retryCompileOrSourceEvaluation()
}

func (m Model) retryCompileOrSourceEvaluation() (Model, tea.Cmd) {
	next, cmd := m.startCompile()
	next.compileRepairAttempted = true
	return next, cmd
}

func (m Model) friendlyCompileError(err error) string {
	var exitErr core.CompileExitError
	if errors.As(err, &exitErr) {
		switch exitErr.Code {
		case 2:
			if m.compileResult != nil {
				return fmt.Sprintf("Partial compile: %d/%d sources were usable. Review source issues before relying on MIXTAPE.md.", m.compileResult.Summary.Succeeded, m.compileResult.Summary.Total)
			}
			return "Partial compile. Review source issues before relying on MIXTAPE.md."
		case 3:
			if m.compileResult != nil {
				return "No usable sources were compiled. Add sources, check source access, or run setup for pages that need JavaScript."
			}
			return "No usable sources were compiled. Add sources or check source access."
		default:
			if strings.TrimSpace(exitErr.Stderr) != "" {
				return strings.TrimSpace(exitErr.Stderr)
			}
			return fmt.Sprintf("Compile exited with code %d.", exitErr.Code)
		}
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "Compile failed."
	}
	return message
}

func (m Model) nextCompileRow() int {
	for i, row := range m.compileRows {
		if row.Status == "queued" {
			return i
		}
	}
	return len(m.compileRows)
}

func (m Model) runningCompileRow() int {
	for i, row := range m.compileRows {
		if row.Status == "running" {
			return i
		}
	}
	for i := len(m.compileRows) - 1; i >= 0; i-- {
		if m.compileRows[i].Status == "queued" {
			return i
		}
	}
	return -1
}

func compileRowView(index int, row compileSourceRow, width int) string {
	marker := styles.Subtitle.Render("○")
	switch row.Status {
	case "running":
		marker = styles.NextActionTitle.Render(">")
	case "done", "cached":
		marker = styles.SuccessText.Render("✓")
	case "failed":
		marker = styles.ErrorText.Render("×")
	}
	labelWidth := max(18, width-28)
	label := truncateMiddle(row.Source, labelWidth)
	detail := row.Detail
	if detail == "" {
		detail = row.Status
	}
	return fmt.Sprintf("%s %02d  %s  %s", marker, index, styles.Subtitle.Render(row.Type), styles.ReportBody.Render(label+"  "+styles.Subtitle.Render(detail)))
}

func compileSpecLabel(spec *core.CompileSourceSpec) string {
	if spec == nil {
		return ""
	}
	if spec.URL != "" {
		return spec.URL
	}
	if spec.Note != nil && strings.TrimSpace(*spec.Note) != "" {
		return strings.TrimSpace(*spec.Note)
	}
	return spec.Type
}

func compileSpecType(spec *core.CompileSourceSpec) string {
	if spec == nil || spec.Type == "" {
		return "source"
	}
	return spec.Type
}

func compileTapeSourceLabel(src tape.Source) string {
	switch {
	case src.Citation != nil && strings.TrimSpace(*src.Citation) != "":
		return strings.TrimSpace(*src.Citation)
	case src.Path != nil && strings.TrimSpace(*src.Path) != "":
		return strings.TrimSpace(*src.Path)
	case strings.TrimSpace(src.URL) != "":
		return strings.TrimSpace(src.URL)
	default:
		return src.Type
	}
}

func compileLoaderMessage(frame int) string {
	return "Fetching sources, assembling MIXTAPE.md, and checking the result."
}

func synthesisAttention(project string) string {
	data, err := os.ReadFile(projectAbsPath(project, "synthesis.md"))
	if err != nil {
		return "synthesis.md is missing. Run the synthesis phase before trusting the compile."
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "synthesis.md is empty. Add the synthesis before trusting the compile."
	}
	if strings.Contains(text, "Replace this placeholder") || strings.Contains(text, "TODO —") {
		return "synthesis.md is still the starter placeholder. Run synthesis or replace it before relying on MIXTAPE.md."
	}
	return ""
}

func (m Model) viewCompile() string {
	width := styles.ClampWidth(m.width - 4)
	sections := []string{
		m.renderLoadingTitle("Compile Console", m.compiling || m.jsSetupRunning || m.sourceRecoveryRunning),
		styles.Subtitle.Render("Fetch sources, assemble MIXTAPE.md, and report anything that needs attention."),
		"",
		m.viewCompileStatus(width),
	}
	if m.sourceRecoveryReview {
		sections = append(sections, "", m.viewSourceRecoveryReview(width))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}
	if m.sourceRecoveryRunning {
		sections = append(sections, "", m.viewSourceRecoveryWorking(width))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}
	if m.compilePane == compilePaneSources {
		if rows := m.viewCompileAllSources(width); rows != "" {
			sections = append(sections, "", rows)
		}
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}
	if result := m.viewCompileResult(width); result != "" {
		sections = append(sections, "", result)
	}
	if m.compileResult == nil {
		if logs := m.viewCompileLog(width); logs != "" {
			sections = append(sections, "", logs)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) compileSourceRowLimit() int {
	if len(m.compileRows) == 0 {
		return 0
	}
	reserved := 18
	limitCap := len(m.compileRows)
	if m.compileResult != nil || m.compileErr != "" || len(m.compileAttentionItems()) > 0 {
		reserved = 28
		limitCap = min(limitCap, 10)
	}
	if m.compileResult != nil && m.actionableCompileWarningCount() > 0 {
		reserved = 34
		limitCap = min(limitCap, 6)
	}
	limit := max(3, m.height-reserved)
	return min(len(m.compileRows), min(limitCap, limit))
}

func (m Model) previewCompiledMixtape() (Model, tea.Cmd) {
	if !m.compileHasUsableResult() {
		m.err = "No usable MIXTAPE.md result is ready to preview. Fix the compile notes, then retry."
		return m, nil
	}
	return m.openPreview("MIXTAPE.md")
}

func (m Model) continueFromCompile() (Model, tea.Cmd) {
	switch {
	case m.compiling:
		m.err = "Wait for compile to finish before creating the Operating Layer."
		return m, nil
	case m.jsSetupRunning:
		m.err = "Wait for JS rendering setup to finish before creating the Operating Layer."
		return m, nil
	case m.sourceRecoveryRunning:
		m.note = "Unavailable source retry is still running. Wait for the retry result."
		m.err = ""
		return m, nil
	case m.sourceRecoveryReview:
		if m.compileRepairRebuildCorpusAfterRecovery {
			recovery := m.sourceRecovery
			m.sourceRecoveryReview = false
			m.compileRepairRebuildCorpusAfterRecovery = false
			next, cmd := m.retrySourceEvaluation()
			next.sourceRecovery = recovery
			next.compileRepairAttempted = true
			next.compileSourcesReviewed = true
			next.note = "Recovered custom source content. Rebuilding the corpus so Liner can add it back."
			return next, cmd
		}
		m.sourceRecoveryReview = false
		m.err = ""
		if m.compileRepairAttempted && m.sourceRecovery != nil && m.sourceRecovery.Succeeded == 0 {
			m.compilePane = compilePaneSources
			m.note = "No custom sources recovered. Review sources, repair again, or add replacements."
			return m, nil
		}
		m.note = "Returned to Compile Console."
		return m, nil
	case m.compilePane == compilePaneIssues && m.compileHasSourceReviewItems():
		return m.viewCompileSourcesNext(), nil
	case m.compilePane == compilePaneSources && !m.compileRepairAttempted && m.compileHasRepairableSources():
		return m.repairCompileSources()
	case m.compileNeedsJSSetup():
		m.err = "Repair sources before creating the Operating Layer."
		return m, nil
	case m.compileAttentionNextAction() != "":
		m.err = m.compileAttentionNextAction()
		return m, nil
	case !m.compileHasUsableResult():
		m.err = "Compile MIXTAPE.md before creating the Operating Layer."
		return m, nil
	}
	if operatingFitImprovementRecommended(m.currentPath) {
		return m.startImprovementReview(), nil
	}
	return m.startLinerDraftReview()
}

func (m Model) viewCompileStatus(width int) string {
	status := "Ready"
	detail := "Compile has not started."
	switch {
	case m.compiling:
		status = "Working"
		detail = compileLoaderMessage(m.fxFrame)
	case m.sourceRecoveryRunning:
		status = "Working"
		detail = "Retrying unavailable sources without rebuilding the corpus."
	case m.sourceRecoveryReview:
		status = "Needs review"
		if m.compileRepairRebuildCorpusAfterRecovery {
			detail = "Unavailable source retry recovered content. Rebuild the corpus when ready."
		} else if m.compileRepairAttempted && m.sourceRecovery != nil && m.sourceRecovery.Succeeded == 0 {
			detail = "Unavailable source retry finished, but no custom sources recovered."
		} else {
			detail = "Unavailable source retry finished. Continue to Compile Console when ready."
		}
	case m.compileErr != "":
		status = "Needs attention"
		detail = m.compileErr
	case m.compileHasSourceReviewItems() && m.compileHasUsableResult():
		status = "Compiled with warnings"
		detail = ""
	case len(m.compileAttentionItems()) > 0:
		status = "Needs attention"
		detail = "Review the compile notes before relying on MIXTAPE.md."
	case m.compileResult != nil:
		status = "Compiled"
		detail = "MIXTAPE.md is ready to preview."
	}
	percent := m.compilePercent()
	counts := fmt.Sprintf("%d/%d sources", min(m.compileDoneN, max(m.compileTotal, m.compileDoneN)), m.compileTotal)
	if m.sourceRecoveryRunning {
		recoveryCount := len(readDroppedCustomURLSources(m.currentPath, m.currentTape))
		if recoveryCount == 0 {
			recoveryCount = 1
		}
		percent = 0.35
		counts = intLabel(recoveryCount, "retryable source")
	} else if m.sourceRecoveryReview && m.sourceRecovery != nil {
		percent = 1
		counts = fmt.Sprintf("%d retryable checked", m.sourceRecovery.Attempted)
	} else if m.compileResult != nil || (!m.compiling && m.compileTotal == 0 && len(m.compileLines) > 0) {
		percent = 1
	}
	if !m.sourceRecoveryRunning && !m.sourceRecoveryReview && m.compileTotal == 0 {
		counts = "0 sources"
	}
	return renderProgressStatusBlock(width, m.compileBar, percent, status, detail, counts)
}

func (m Model) viewCompileSources(width int) string {
	if len(m.compileRows) == 0 {
		return ""
	}
	limit := m.compileSourceRowLimit()
	start := max(0, len(m.compileRows)-limit)
	lines := []string{styles.ReportSection.Render("Sources")}
	if start > 0 {
		lines = append(lines, styles.Subtitle.Render(fmt.Sprintf("↑ %d earlier source(s)", start)))
	}
	tableRows := make([]table.Row, 0, len(m.compileRows[start:]))
	sourceWidth := max(20, width-52)
	detailWidth := 18
	for index, row := range m.compileRows[start:] {
		detail := row.Detail
		if detail == "" {
			detail = row.Status
		}
		detail = compileSourceDetailSummary(detail)
		tableRows = append(tableRows, table.Row{
			fmt.Sprintf("%02d", start+index+1),
			compileStatusLabel(row.Status),
			truncateMiddle(row.Type, 10),
			truncateMiddle(row.Source, sourceWidth),
			truncateMiddle(detail, detailWidth),
		})
	}
	sourceTable := newDataTable(
		[]table.Column{
			{Title: "#", Width: 4},
			{Title: "Status", Width: 9},
			{Title: "Type", Width: 10},
			{Title: "Source", Width: sourceWidth},
			{Title: "Detail", Width: detailWidth},
		},
		tableRows,
		width,
		min(limit+1, max(4, m.compileSourceRowLimit()+1)),
		false,
	)
	lines = append(lines, sourceTable.View())
	return strings.Join(lines, "\n")
}

func (m Model) viewCompileAllSources(width int) string {
	items := m.compileSourceListItems()
	if len(items) == 0 {
		return ""
	}
	index := clampCompileSourceIndex(m.compileSourceIndex, len(items))
	limit := min(12, max(6, m.height-24))
	start, end := visibleWindow(index, len(items), limit)
	sourceWidth := max(24, width/3)
	detailWidth := max(20, width-sourceWidth-50)
	tableRows := make([]table.Row, 0, end-start)
	for rowIndex, item := range items[start:end] {
		absoluteIndex := start + rowIndex
		selected := " "
		if absoluteIndex == index {
			selected = ">"
		}
		tableRows = append(tableRows, table.Row{
			selected,
			fmt.Sprintf("%02d", absoluteIndex+1),
			truncateMiddle(item.Status, 12),
			truncateMiddle(item.Kind, 15),
			truncateMiddle(item.Source, sourceWidth),
			truncateMiddle(item.Detail, detailWidth),
		})
	}
	lines := []string{styles.ReportSection.Render("Sources")}
	lines = append(lines, styles.Subtitle.Render(compileSourceListSummary(items)))
	if m.compilePane == compilePaneSources && !m.compileRepairAttempted && m.compileHasRepairableSources() {
		lines = append(lines, styles.Subtitle.Render("Repair sources first with r. This retries unavailable custom sources, then rebuilds the corpus if any recover."))
	} else if m.compilePane == compilePaneSources && m.compileRepairAttempted {
		lines = append(lines, styles.Subtitle.Render("You can continue with the current MIXTAPE.md, repair again with r, or add replacements with a."))
	}
	if start > 0 {
		lines = append(lines, styles.Subtitle.Render(fmt.Sprintf("↑ %d earlier source(s)", start)))
	}
	sourceTable := newDataTable(
		[]table.Column{
			{Title: "", Width: 2},
			{Title: "#", Width: 4},
			{Title: "Status", Width: 12},
			{Title: "Kind", Width: 15},
			{Title: "Source", Width: sourceWidth},
			{Title: "Detail", Width: detailWidth},
		},
		tableRows,
		width,
		min(len(tableRows)+1, limit+1),
		false,
	)
	lines = append(lines, sourceTable.View())
	if end < len(items) {
		lines = append(lines, styles.Subtitle.Render(fmt.Sprintf("↓ %d later source(s)", len(items)-end)))
	}
	if detail := m.compileSourceDetail(width); detail != "" {
		lines = append(lines, "", detail)
	}
	return strings.Join(lines, "\n")
}

func (m Model) compileSourceListItems() []compileSourceListItem {
	items := make([]compileSourceListItem, 0, len(m.currentTape.Sources)+len(readExcludedLocalSourceIssues(m.currentPath, m.currentTape)))
	warningKeys := map[string]bool{}
	if m.compileResult != nil {
		for _, item := range m.actionableCompileWarnings() {
			warning := item.Warning
			source := strings.TrimSpace(warning.URL)
			if source == "" {
				source = compileWarningSummary(warning)
			}
			for _, key := range issueKeysForURL(source) {
				warningKeys[key] = true
			}
			items = append(items, compileSourceListItem{
				Status:     fallbackText(warning.Severity, "error"),
				Kind:       "research source",
				Type:       "web",
				Source:     source,
				Detail:     compileWarningSummary(warning),
				OpenTarget: source,
			})
		}
	}
	for _, issue := range readExcludedLocalSourceIssues(m.currentPath, m.currentTape) {
		items = append(items, compileSourceListItem{
			Status:     issue.Status,
			Kind:       "custom source",
			Type:       issue.Type,
			Source:     issue.Source,
			Detail:     issue.Reason,
			OpenTarget: issue.OpenTarget,
		})
	}
	for index, src := range m.currentTape.Sources {
		if keySetContainsAny(warningKeys, issueKeysForSource(src)) {
			continue
		}
		row := compileSourceRow{Status: "queued", Type: src.Type, Source: compileTapeSourceLabel(src)}
		if index < len(m.compileRows) {
			row = m.compileRows[index]
		}
		detail := row.Detail
		if detail == "" {
			detail = row.Status
		}
		items = append(items, compileSourceListItem{
			Status:     compileStatusLabel(row.Status),
			Kind:       "research source",
			Type:       fallbackText(row.Type, src.Type),
			Source:     fallbackText(row.Source, compileTapeSourceLabel(src)),
			Detail:     compileSourceDetailSummary(detail),
			OpenTarget: compileSourceOpenTarget(src),
		})
	}
	return items
}

func (m Model) compileSourceDetail(width int) string {
	item, ok := m.selectedCompileSource()
	if !ok {
		return ""
	}
	rows := []labelValueRow{
		{Label: "Selected", Value: item.Status},
		{Label: "Kind", Value: item.Kind},
		{Label: "Source", Value: truncateMiddle(item.Source, max(24, width-18))},
		{Label: "Detail", Value: item.Detail},
	}
	return styles.ReportSection.Render("Source detail") + "\n" + renderLabelValueBlock(width, rows, 0, 0)
}

func compileSourceListSummary(items []compileSourceListItem) string {
	usable := 0
	retryable := 0
	needsCorpus := 0
	customNotUsed := 0
	attention := 0
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "retryable", "needs retry":
			retryable++
		case "needs corpus":
			needsCorpus++
		case "dropped", "not in tape", "excluded", "not selected":
			customNotUsed++
		case "failed", "error", "warning":
			attention++
		default:
			usable++
		}
	}
	parts := []string{}
	if attention > 0 {
		verb := "need"
		if attention == 1 {
			verb = "needs"
		}
		parts = append(parts, intLabel(attention, "source")+" "+verb+" attention")
	}
	if retryable > 0 {
		verb := "need"
		if retryable == 1 {
			verb = "needs"
		}
		parts = append(parts, intLabel(retryable, "custom source")+" "+verb+" retry")
	}
	if needsCorpus > 0 {
		verb := "need"
		if needsCorpus == 1 {
			verb = "needs"
		}
		parts = append(parts, intLabel(needsCorpus, "recovered custom source")+" "+verb+" Build Corpus")
	}
	if customNotUsed > 0 {
		parts = append(parts, intLabel(customNotUsed, "custom source")+" not used")
	}
	if usable > 0 {
		parts = append(parts, intLabel(usable, "usable source"))
	}
	return strings.Join(parts, " · ")
}

func compileSourceOpenTarget(src tape.Source) string {
	if strings.TrimSpace(src.URL) != "" {
		return strings.TrimSpace(src.URL)
	}
	if src.Path != nil && strings.TrimSpace(*src.Path) != "" {
		return strings.TrimSpace(*src.Path)
	}
	if src.Citation != nil && strings.TrimSpace(*src.Citation) != "" {
		return strings.TrimSpace(*src.Citation)
	}
	return ""
}

func visibleWindow(index int, count int, limit int) (int, int) {
	if count <= 0 {
		return 0, 0
	}
	limit = min(max(limit, 1), count)
	start := index - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > count {
		start = count - limit
	}
	return start, start + limit
}

func clampCompileSourceIndex(index int, count int) int {
	if count <= 0 {
		return 0
	}
	return min(max(index, 0), count-1)
}

func compileStatusLabel(status string) string {
	switch status {
	case "running":
		return "working"
	case "done", "cached", "failed", "queued":
		return status
	default:
		return fallbackText(status, "queued")
	}
}

func (m Model) viewCompileResult(width int) string {
	if m.compileResult == nil && m.compileErr == "" {
		if m.compiling || len(m.compileAttentionItems()) == 0 {
			return ""
		}
	}
	lines := []string{styles.ReportSection.Render("Result")}
	if m.compileResult != nil {
		summary := m.compileResult.Summary
		outcome := "compiled"
		if m.compileHasSourceReviewItems() {
			outcome = "compiled with warnings"
		}
		if m.compileRepairAttempted && m.compileHasSourceReviewItems() {
			outcome = "repair finished with warnings"
		} else if m.compileRepairAttempted {
			outcome = "repair finished"
		}
		lines = append(lines, styles.NextActionTitle.Render("● "+outcome)+"  "+styles.NextActionText.Render("MIXTAPE.md is ready with "+intLabel(summary.Succeeded, "usable source")+"."))
		if m.compileResult.MixtapePath != "" {
			lines = append(lines, styles.Subtitle.Render("wrote "+m.compileResult.MixtapePath))
		}
	}
	for _, item := range m.compileAttentionItems() {
		lines = append(lines, styles.NextActionTitle.Render("! ")+styles.NextActionText.Render(item))
	}
	summaryLines := m.compileResultSummaryLines()
	if len(summaryLines) > 0 {
		lines = append(lines, "", styles.ReportSection.Render("Summary"))
		for _, line := range summaryLines {
			lines = append(lines, styles.NextActionText.Render(line))
		}
	}
	if recovery := m.viewSourceRecoveryResult(width); recovery != "" {
		lines = append(lines, "", recovery)
	}
	if m.compileErr != "" && !m.compileHasUsableResult() {
		lines = append(lines, styles.ErrorText.Render(m.compileErr))
	}
	if m.compileNeedsJSSetup() || m.jsSetupRunning {
		lines = append(lines, "", m.viewCompileJSSetup(width))
	}
	if recovered := m.recoveredCompileWarningCount(); recovered > 0 {
		lines = append(lines, styles.SuccessText.Render(fmt.Sprintf("● recovered %d source(s) with browser rendering", recovered))+"  "+styles.NextActionText.Render("included in MIXTAPE.md"))
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m Model) compileResultSummaryLines() []string {
	lines := []string{}
	if count := m.actionableCompileWarningCount(); count > 0 {
		verb := "need"
		if count == 1 {
			verb = "needs"
		}
		lines = append(lines, intLabel(count, "research source")+" "+verb+" attention.")
	}
	if issues := readExcludedLocalSourceIssues(m.currentPath, m.currentTape); len(issues) > 0 {
		retryable := 0
		needsCorpus := 0
		for _, issue := range issues {
			switch strings.ToLower(strings.TrimSpace(issue.Status)) {
			case "retryable":
				retryable++
			case "needs corpus":
				needsCorpus++
			}
		}
		if needsCorpus > 0 {
			verb := "need"
			object := "them"
			if needsCorpus == 1 {
				verb = "needs"
				object = "it"
			}
			lines = append(lines, intLabel(needsCorpus, "recovered custom source")+" "+verb+" Build Corpus before Liner can use "+object+".")
		}
		if retryable > 0 {
			lines = append(lines, intLabel(retryable, "unavailable custom source")+" can be retried.")
		}
		if other := len(issues) - retryable - needsCorpus; other > 0 {
			verb := "were"
			if other == 1 {
				verb = "was"
			}
			lines = append(lines, intLabel(other, "saved custom source")+" "+verb+" not used.")
		}
	}
	if summary, ok := m.compileSourceEvaluationSummary(); ok {
		lines = append(lines, "Source notes: "+summary.Display(m.currentPath)+".")
	}
	if m.compileNeedsJSSetup() {
		lines = append(lines, "JS rendering is missing for at least one repairable source; repair will install it first.")
	}
	return lines
}

func (m Model) viewSourceRecoveryResult(width int) string {
	if m.sourceRecovery == nil {
		return ""
	}
	result := *m.sourceRecovery
	title := "Unavailable source retry"
	if m.compileRepairAttempted {
		title = "Repair result"
	}
	lines := []string{
		styles.ReportSection.Render(title),
		styles.NextActionText.Render(fmt.Sprintf("%d retryable checked, %d recovered, %d still unavailable", result.Attempted, result.Succeeded, result.Failed)),
	}
	if result.Succeeded > 0 {
		lines = append(lines, styles.SuccessText.Render("● saved recovered source content")+"  "+styles.NextActionText.Render("Available for the next Build Corpus run."))
	}
	for _, source := range result.Sources {
		if source.SavedTo == "" {
			continue
		}
		lines = append(lines, styles.Subtitle.Render("saved "+truncateMiddle(source.SavedTo, max(20, width-8))))
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewSourceRecoveryReview(width int) string {
	result := sourceRecoveryResult{}
	if m.sourceRecovery != nil {
		result = *m.sourceRecovery
	}
	lines := []string{
		styles.ReportSection.Render("Unavailable source retry"),
		styles.NextActionText.Render(fmt.Sprintf("%d retryable checked, %d recovered, %d still unavailable", result.Attempted, result.Succeeded, result.Failed)),
	}
	if strings.TrimSpace(m.sourceRecoveryError) != "" {
		lines = append(lines, styles.ErrorText.Render("Retry error: "+m.sourceRecoveryError))
	} else if result.Succeeded > 0 {
		message := "Recovered source content was saved under local-sources/recovered/. Continue to Compile Console, then run Build Corpus so the AI can reconsider it."
		if m.compileRepairRebuildCorpusAfterRecovery {
			message = "Recovered source content was saved under local-sources/recovered/. Press enter to rebuild the corpus from Candidate discovery so Liner can add it back."
		}
		for _, line := range wrapWords(message, width) {
			lines = append(lines, styles.SuccessText.Render("● ")+styles.NextActionText.Render(line))
		}
	} else {
		message := "No retryable unavailable sources were recovered. Try again after changing network/cookies, or add replacement source content manually."
		if m.compileRepairAttempted {
			message = "No custom sources were recovered. Return to sources to retry, open the URLs, or add replacement content manually."
		}
		for _, line := range wrapWords(message, width) {
			lines = append(lines, styles.NextActionTitle.Render("! ")+styles.NextActionText.Render(line))
		}
	}
	if len(result.Sources) > 0 {
		lines = append(lines, "", renderSourceRecoveryRows(width, result.Sources))
	}
	continueText := "Press enter to return to Compile Console."
	if m.compileRepairRebuildCorpusAfterRecovery {
		continueText = "Press enter to rebuild the corpus."
	} else if m.compileRepairAttempted && result.Succeeded == 0 {
		continueText = "Press enter to return to Sources."
	}
	lines = append(lines, "", styles.NextActionTitle.Render("> Continue: ")+styles.NextActionText.Render(continueText))
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m Model) viewSourceRecoveryWorking(width int) string {
	sources := readDroppedCustomURLSources(m.currentPath, m.currentTape)
	lines := []string{
		styles.ReportSection.Render("Retry unavailable sources"),
	}
	for _, line := range wrapWords("Liner is retrying only sources marked retryable. Build Corpus and compile are not running.", width) {
		lines = append(lines, styles.NextActionText.Render(line))
	}
	for _, line := range wrapWords("If a source is recovered, Liner saves a local copy under local-sources/recovered/ and asks you to run Build Corpus.", width) {
		lines = append(lines, styles.Subtitle.Render(line))
	}
	if len(sources) == 0 {
		lines = append(lines, "", styles.Subtitle.Render("Waiting for the recovery result."))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
	}
	sourceWidth := max(28, width-42)
	rows := make([]table.Row, 0, len(sources))
	for _, source := range sources {
		rows = append(rows, table.Row{
			"fetching",
			visibleSourceType(source.Item.Source.Type),
			truncateMiddle(localSourceIssueLabel(source.Item), sourceWidth),
			truncateMiddle(source.Reason, 18),
		})
	}
	sourceTable := newDataTable(
		[]table.Column{
			{Title: "Status", Width: 10},
			{Title: "Type", Width: 9},
			{Title: "Source", Width: sourceWidth},
			{Title: "Prior issue", Width: 18},
		},
		rows,
		width,
		min(len(rows)+1, 8),
		false,
	)
	lines = append(lines, "", sourceTable.View())
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func renderSourceRecoveryRows(width int, sources []sourceRecoverySource) string {
	sourceWidth := max(24, width-48)
	rows := make([]table.Row, 0, len(sources))
	for _, source := range sources {
		detail := source.Message
		if detail == "" {
			detail = source.SavedTo
		}
		if detail == "" {
			detail = source.Reason
		}
		rows = append(rows, table.Row{
			truncateMiddle(source.Status, 10),
			truncateMiddle(visibleSourceType(source.Type), 10),
			truncateMiddle(source.URL, sourceWidth),
			truncateMiddle(detail, 20),
		})
	}
	sourceTable := newDataTable(
		[]table.Column{
			{Title: "Status", Width: 10},
			{Title: "Type", Width: 10},
			{Title: "Retried source", Width: sourceWidth},
			{Title: "Result", Width: 20},
		},
		rows,
		width,
		min(len(rows)+1, 8),
		false,
	)
	return sourceTable.View()
}

func renderExcludedLocalSources(width int, issues []excludedLocalSourceIssue) string {
	if len(issues) == 0 {
		return ""
	}
	sourceWidth := max(22, width/4)
	actionWidth := max(18, min(28, width/4))
	reasonWidth := max(24, width-sourceWidth-actionWidth-36)
	rows := make([]table.Row, 0, len(issues))
	for _, issue := range issues {
		rows = append(rows, table.Row{
			truncateMiddle(issue.Status, 12),
			truncateMiddle(visibleSourceType(issue.Type), 10),
			truncateMiddle(issue.Source, sourceWidth),
			truncateMiddle(issue.Reason, reasonWidth),
			truncateMiddle(issue.NextAction, actionWidth),
		})
	}
	sourceTable := newDataTable(
		[]table.Column{
			{Title: "State", Width: 12},
			{Title: "Type", Width: 10},
			{Title: "Saved source", Width: sourceWidth},
			{Title: "What happened", Width: reasonWidth},
			{Title: "Next", Width: actionWidth},
		},
		rows,
		width,
		min(len(rows)+1, 10),
		false,
	)
	return styles.ReportSection.Render("Custom sources not used") + "\n" + renderExcludedLocalSourceHint(issues) + "\n" + sourceTable.View()
}

func renderExcludedLocalSourceHint(issues []excludedLocalSourceIssue) string {
	retryable := 0
	for _, issue := range issues {
		if strings.EqualFold(strings.TrimSpace(issue.Status), "retryable") {
			retryable++
		}
	}
	if retryable > 0 {
		verb := "are"
		if len(issues) == 1 {
			verb = "is"
		}
		return styles.Subtitle.Render(fmt.Sprintf("%s %s missing from the tape. Open Compile Console and press r to repair the %s; recovered sources are saved for the next Build Corpus run.", intLabel(len(issues), "saved custom source"), verb, intLabel(retryable, "custom source")))
	}
	return styles.Subtitle.Render("Saved custom sources are missing from the tape. Add replacement content with a, or run Build Corpus after fixing the source files.")
}

func (m Model) viewCompileJSSetup(width int) string {
	status := "JS rendering needed"
	detail := "Install Playwright Chromium, then retry this compile."
	percent := 0.35
	if m.jsSetupRunning {
		status = "Installing JS rendering"
		detail = "Downloading Playwright Chromium. First run can take a few minutes."
		percent = jsSetupProgressPercent(m.fxFrame)
	}
	action := "Press i to install JS rendering. If setup succeeds, Liner retries this compile automatically."
	if m.jsSetupRunning {
		action = "Wait for setup to finish. If it succeeds, Liner retries this compile automatically."
	}
	lines := []string{
		styles.ReportSection.Render("JS rendering"),
		renderProgressStatusBlock(width, m.compileBar, percent, status, detail, "browser setup"),
		styles.Subtitle.Render(action),
	}
	return strings.Join(lines, "\n")
}

func (m Model) compileWarningsTable(width int) table.Model {
	if m.compileResult == nil {
		return newDataTable(
			[]table.Column{{Title: "Warnings", Width: max(1, width-4)}},
			[]table.Row{},
			width,
			1,
			false,
		)
	}
	sourceWidth := max(18, width/3)
	messageWidth := max(20, width-sourceWidth-24)
	actionable := m.actionableCompileWarnings()
	rows := make([]table.Row, 0, len(actionable))
	for _, item := range actionable {
		selected := " "
		if item.Index == m.compileWarningIndex {
			selected = ">"
		}
		warning := item.Warning
		message := compileWarningSummary(warning)
		rows = append(rows, table.Row{
			selected,
			truncateMiddle(fallbackText(warning.Severity, "warning"), 8),
			truncateMiddle(warning.URL, sourceWidth),
			truncateMiddle(message, messageWidth),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "", "", ""})
	}
	return newDataTable(
		[]table.Column{
			{Title: "", Width: 2},
			{Title: "Level", Width: 8},
			{Title: "Source", Width: sourceWidth},
			{Title: "Message", Width: messageWidth},
		},
		rows,
		width,
		min(len(rows)+1, 6),
		false,
	)
}

func (m Model) compileWarningDetail(width int) string {
	warning, ok := m.selectedCompileWarning()
	if !ok {
		return ""
	}
	rows := []labelValueRow{
		{Label: "Selected", Value: fallbackText(warning.Severity, "warning")},
		{Label: "Source", Value: truncateMiddle(warning.URL, max(24, width-18))},
		{Label: "Message", Value: compileWarningSummary(warning)},
		{Label: "Recommendation", Value: compileWarningRecommendation(warning)},
	}
	return styles.ReportSection.Render("Issue detail") + "\n" + renderLabelValueBlock(width, rows, 0, 0)
}

func compileWarningRecommendation(warning core.CompileWarningPayload) string {
	message := strings.ToLower(warning.Message)
	switch {
	case compileWarningRecoveredWithJS(warning):
		return "Liner recovered this source with browser rendering and included the rendered text in MIXTAPE.md. You can continue; open it only if you want to inspect the captured evidence."
	case compileWarningNeedsJSSetup(warning):
		return "Install JS rendering from this screen, then retry compile. If the source still fails after browser setup, replace or drop it."
	case strings.Contains(message, "very short") || strings.Contains(message, "short"):
		return "Open the source if it is important. If the page has no readable body, drop it from tape.yaml and retry compile; otherwise keep it only as a weak pointer."
	case strings.Contains(message, "transcript"):
		return "Find a video transcript or a stronger written source, add that replacement, then retry compile."
	case strings.Contains(message, "failed") || strings.Contains(message, "could not") || strings.Contains(message, "unavailable"):
		return "Open the source to verify access. If it cannot be fetched, replace it with a readable source or drop it before retrying."
	default:
		return "Open the source to verify the evidence. Drop or replace it if it does not add usable support, then retry compile."
	}
}

func compileSourceDetailSummary(detail string) string {
	detail = strings.Join(strings.Fields(detail), " ")
	if detail == "" {
		return ""
	}
	if compileMessageNeedsJSSetup(detail) {
		return "needs browser"
	}
	return detail
}

func compileWarningSummary(warning core.CompileWarningPayload) string {
	message := strings.Join(strings.Fields(warning.Message), " ")
	if message == "" {
		return fallbackText(warning.Severity, "warning")
	}
	lower := strings.ToLower(message)
	switch {
	case compileWarningRecoveredWithJS(warning):
		return "Recovered with browser rendering and included in MIXTAPE.md."
	case compileWarningNeedsJSSetup(warning):
		return "This page needs browser rendering before Liner can extract readable text."
	case strings.Contains(lower, "http 404") || strings.Contains(lower, "not_found") || strings.Contains(lower, "not found"):
		return "The source was not found."
	case strings.Contains(lower, "status: http 403") || strings.Contains(lower, "status 403") || strings.Contains(lower, "forbidden"):
		return "The source blocked access."
	case strings.Contains(lower, "bot-detection") || strings.Contains(lower, "bot detection"):
		return "The source appears to block automated fetching."
	case strings.Contains(lower, "very short") || strings.Contains(lower, "too short"):
		return "The fetched text was too short to trust as evidence."
	case strings.Contains(lower, "transcript"):
		return "The video transcript was unavailable."
	default:
		return truncateMiddle(message, 140)
	}
}

func compileWarningRecoveredWithJS(warning core.CompileWarningPayload) bool {
	message := strings.ToLower(warning.Message)
	return strings.Contains(message, "auto-fell back to render: js") ||
		strings.Contains(message, "recovered this source with js rendering") ||
		(strings.Contains(message, "js rendering") && strings.Contains(message, "included in mixtape.md"))
}

func compileWarningBlocksProgress(warning core.CompileWarningPayload) bool {
	message := strings.ToLower(warning.Message)
	severity := strings.ToLower(strings.TrimSpace(warning.Severity))
	switch {
	case compileWarningRecoveredWithJS(warning):
		return false
	case compileWarningNeedsJSSetup(warning):
		return true
	case severity == "error":
		return true
	case strings.Contains(message, "404") ||
		strings.Contains(message, "not_found") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "failed") ||
		strings.Contains(message, "could not") ||
		strings.Contains(message, "unavailable") ||
		strings.Contains(message, "partial") ||
		strings.Contains(message, "very short") ||
		strings.Contains(message, "too short"):
		return true
	default:
		return false
	}
}

func (m Model) compileHasBlockingWarnings() bool {
	if m.compileResult == nil {
		return false
	}
	for _, warning := range m.compileResult.Warnings {
		if compileWarningBlocksProgress(warning) {
			return true
		}
	}
	return false
}

func (m Model) selectedBlockingCompileWarning() (core.CompileWarningPayload, bool) {
	if m.compileResult == nil {
		return core.CompileWarningPayload{}, false
	}
	if warning, ok := m.selectedCompileWarning(); ok && compileWarningBlocksProgress(warning) {
		return warning, true
	}
	for _, warning := range m.compileResult.Warnings {
		if compileWarningBlocksProgress(warning) {
			return warning, true
		}
	}
	return core.CompileWarningPayload{}, false
}

func (m Model) compileAttentionNextAction() string {
	if _, ok := m.selectedBlockingCompileWarning(); ok && !m.compileHasUsableResult() {
		return "Resolve source issues before creating the Operating Layer."
	}
	if len(m.compileAttentionItems()) > 0 {
		if m.compileResult != nil && m.compileResult.Summary.Total == 0 {
			if m.currentTape.JTBD != nil && strings.TrimSpace(*m.currentTape.JTBD) != "" {
				return "Build Corpus and save draft sources before compiling again."
			}
			return "Add sources or define the job before compiling again."
		}
		return "Resolve the compile notes, then retry compile."
	}
	if strings.TrimSpace(m.compileErr) != "" && !m.compileHasUsableResult() {
		return "Review the compile error, then retry compile."
	}
	return ""
}

func compileWarningNextAction(warning core.CompileWarningPayload) string {
	message := strings.ToLower(warning.Message)
	switch {
	case compileWarningRecoveredWithJS(warning):
		return "This source was recovered with JS rendering and included in MIXTAPE.md."
	case compileWarningNeedsJSSetup(warning):
		return "Press i to install JS rendering, then retry compile."
	case strings.Contains(message, "404") ||
		strings.Contains(message, "not_found") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "failed") ||
		strings.Contains(message, "could not") ||
		strings.Contains(message, "unavailable") ||
		strings.Contains(message, "partial"):
		return "Open the source with o, drop it with d, or add a replacement with a."
	case strings.Contains(message, "very short") || strings.Contains(message, "short"):
		return "Open the source with o. If it has no readable body, drop it with d or replace it with a."
	case strings.Contains(message, "transcript"):
		return "Add a transcript or stronger written source with a, then retry compile."
	default:
		return "Review the selected source issue, then open, drop, replace, or retry."
	}
}

func (m Model) startJSSetupForCompile() (Model, tea.Cmd) {
	if !m.compileNeedsJSSetup() {
		m.err = "The selected compile issues do not need JS rendering setup."
		return m, nil
	}
	if m.jsSetupRunning {
		return m, nil
	}
	m.jsSetupRunning = true
	m.jsSetupRetryCompile = true
	m.jsSetupFromOnboarding = false
	m.note = "Installing JS rendering support."
	m.err = ""
	return m, setupJS(m.runner)
}

func (m Model) viewCompileLog(width int) string {
	if len(m.compileLines) == 0 {
		return ""
	}
	lines := tail(m.compileLines, max(4, min(8, m.height-16)))
	return styles.ReportSection.Render("Log") + "\n" +
		styles.MutedText.Width(width).Render(strings.Join(lines, "\n")) + "\n" +
		styles.Subtitle.Render("v opens the full compile log")
}

func (m Model) openCompileLogPreview() (Model, tea.Cmd) {
	if len(m.compileLines) == 0 {
		m.err = "No compile log yet."
		return m, nil
	}
	body := "# Compile Log\n\n```text\n" + strings.Join(m.compileLines, "\n") + "\n```"
	return m.openPreviewContent("compile log", body)
}

func newCompileProgress(width int) progress.Model {
	return newTaskProgressBar(width)
}

func compileProgressWidth(width int) int {
	return taskProgressWidth(width)
}

func waitCompileEvent(events <-chan core.CompileEvent, done <-chan error) tea.Cmd {
	return func() tea.Msg {
		if events == nil {
			return compileDoneMsg{}
		}
		if event, ok := <-events; ok {
			return compileEventMsg{event: event}
		}
		if done == nil {
			return compileDoneMsg{}
		}
		return compileDoneMsg{err: <-done}
	}
}
