package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func sourceApplyNeedsFreshPlan(err error) bool {
	var maintenanceErr *core.MaintenanceError
	return errors.As(err, &maintenanceErr) && maintenanceErr.Report != nil && maintenanceErr.Report.Code == "stale_project"
}

const (
	sourceBatchPhasePlanning   = "planning"
	sourceBatchPhaseValidation = "validation"
	sourceBatchPhaseApply      = "apply"
	sourceBatchPhaseCancelled  = "cancelled"
	sourceBatchPhaseFailed     = "failed"
	sourceBatchPhaseComplete   = "complete"
)

func newSourceTable(width int, height int) table.Model {
	t := table.New(
		table.WithColumns(sourceColumns(width)),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithWidth(width),
		table.WithHeight(height),
	)
	t.SetStyles(dataTableStyles(true))
	return t
}

func sourceColumns(width int) []table.Column {
	width = max(width, 64)
	sourceWidth := max(18, width-59)
	return []table.Column{
		{Title: "Use", Width: 6},
		{Title: "Type", Width: 10},
		{Title: "Source", Width: sourceWidth},
		{Title: "Saved as", Width: 28},
	}
}

func (m *Model) applySourcePreview(preview source.Preview) {
	m.sourcePlan = preview
	m.sourceTable.SetRows(sourceRows(preview))
}

func (m *Model) applySourceItems(items []source.StagedSource) {
	m.sourceItems = items
	m.sourceTable.SetRows(stagedRows(items))
	m.sourceTable.SetCursor(clampSourceCursor(m.sourceTable.Cursor(), len(items)))
}

func sourceRows(preview source.Preview) []table.Row {
	rows := make([]table.Row, 0, len(preview.Sources))
	for _, src := range preview.Sources {
		kind, label, savedAs := source.Describe(src)
		rows = append(rows, table.Row{sourceUseMark(true), visibleSourceType(kind), label, savedAs})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "none", "No sources recognized yet", ""})
	}
	return rows
}

func stagedRows(items []source.StagedSource) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{sourceUseMark(item.Active), visibleSourceType(item.Type), item.Label, item.Destination})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "", "No sources added yet", ""})
	}
	return rows
}

func stagedRowsForColumns(items []source.StagedSource, columns []table.Column, selected int) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for index, item := range items {
		rows = append(rows, table.Row{
			sourceUseMarkForRow(item.Active, index == selected),
			truncateMiddle(visibleSourceType(item.Type), columns[1].Width),
			truncateMiddle(item.Label, columns[2].Width),
			truncateMiddle(item.Destination, columns[3].Width),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "", "No sources added yet", ""})
	}
	return rows
}

func sourceListRows(items []source.StagedSource, sourceWidth int, selected int) []table.Row {
	out := make([]table.Row, 0, len(items))
	for index, item := range items {
		out = append(out, table.Row{
			sourceUseMarkForRow(item.Active, index == selected),
			truncateMiddle(visibleSourceType(item.Type), 10),
			truncateMiddle(item.Label, sourceWidth),
		})
	}
	if len(out) == 0 {
		out = append(out, table.Row{"", "", "No sources added yet"})
	}
	return out
}

func sourceUseMark(active bool) string {
	if active {
		return "✓"
	}
	return "○"
}

func sourceUseMarkForRow(active bool, selected bool) string {
	marker := " "
	if selected {
		marker = ">"
	}
	return marker + " " + sourceUseMark(active)
}

func stagedCounts(items []source.StagedSource) map[string]int {
	counts := map[string]int{
		"web":        0,
		"youtube":    0,
		"local":      0,
		"article":    0,
		"source_doc": 0,
	}
	for _, item := range items {
		counts[visibleSourceType(item.Type)]++
	}
	return counts
}

func (m *Model) startSourceEntry() {
	returnScreen := screenProject
	if m.screen == screenCompile {
		returnScreen = screenCompile
	}
	m.sourceEntryReturnScreen = returnScreen
	m.sourceEntryReturnSet = true
	m.screen = screenSources
	m.sourceInput.Focus()
	if strings.TrimSpace(m.sourceInput.Value()) == "" {
		m.applySourcePreview(source.Preview{})
	}
}

func (m Model) sourceEntryReturnsToCompile() bool {
	return m.sourceEntryReturnSet && m.sourceEntryReturnScreen == screenCompile
}

func (m Model) returnFromSourceEntry() Model {
	target := screenProject
	if m.sourceEntryReturnSet {
		target = m.sourceEntryReturnScreen
	}
	m.screen = target
	m.sourceEntryReturnSet = false
	return m
}

func (m Model) canOpenLocalSources() bool {
	return strings.TrimSpace(m.currentPath) != "" && projectDirExists(m.currentPath, "local-sources")
}

func (m Model) viewSources() string {
	width := styles.ClampWidth(m.width - 4)
	tableWidth := styles.ClampWidth(m.width - 8)
	localFolder := styles.Subtitle.Render("Local folder will be created after the first saved source.")
	if m.canOpenLocalSources() {
		localLabel := "Local folder: "
		localPath := truncateMiddle(projectAbsPath(m.currentPath, "local-sources"), max(12, width-lipgloss.Width(localLabel)))
		localFolder = styles.Subtitle.Render(localLabel + localPath)
	}
	pending := strings.TrimSpace(m.sourceInput.Value())
	pendingPreview := sourcePendingPreview(m.sourcePlan, pending)
	warnings := ""
	if len(m.sourceWarnings) > 0 {
		warnings = "\n" + styles.ErrorText.Render(strings.Join(m.sourceWarnings[:min(4, len(m.sourceWarnings))], "\n"))
	}
	actionRows := m.sourceEntryActionRows()
	sourceListWidth := max(24, tableWidth-24)
	sourceTable := newVisibleDataTable(
		[]table.Column{{Title: "Use", Width: 6}, {Title: "Type", Width: 10}, {Title: "Source", Width: sourceListWidth}},
		sourceListRows(m.sourceItems, sourceListWidth, m.sourceTable.Cursor()),
		tableWidth,
		sourceEntryTableHeight(m.height, len(m.sourceItems), len(actionRows), len(m.sourceWarnings)),
		true,
		m.sourceTable.Cursor(),
	)
	addedTitle := styles.Section.Render("Added sources")
	if summary := m.sourceEntryListSummary(); summary != "" {
		addedTitle += " " + styles.Subtitle.Render(summary)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Title.Render("Add Sources"),
		localFolder,
		"",
		lipgloss.NewStyle().Width(width).Render(styles.ReportSection.Render("Source")+" "+m.sourceInput.View()),
		pendingPreview,
		"",
		addedTitle,
		sourceTable.View(),
		"",
		styles.ReportSection.Render("Actions"),
		newActionTable(width, actionRows).View(),
		warnings,
	)
}

func (m Model) sourceEntryActionRows() []actionTableRow {
	addAction := "Add pending source"
	addWrites := "staged source"
	if strings.TrimSpace(m.sourceInput.Value()) == "" && len(m.sourceItems) > 0 {
		addAction = "Review sources"
		addWrites = "review screen"
	}
	rows := []actionTableRow{
		{Key: "enter", Action: addAction, Writes: addWrites},
	}
	if sourceEntryHasListFocus(m) {
		rows = append(rows,
			actionTableRow{Key: "space", Action: "Toggle selected source", Writes: "active flag"},
			actionTableRow{Key: "d", Action: "Remove selected source", Writes: "staged list"},
		)
	}
	rows = append(rows, actionTableRow{Key: "f", Action: "Finish source entry", Writes: "review or clarify"})
	if m.canOpenLocalSources() {
		rows = append(rows, actionTableRow{Key: "ctrl+o", Action: "Open local-sources", Writes: "read-only"})
	}
	return rows
}

func (m Model) sourceEntryActionsTable(width int) table.Model {
	return newActionTable(width, m.sourceEntryActionRows())
}

func (m Model) sourceEntryListSummary() string {
	total := len(m.sourceItems)
	if total == 0 {
		return ""
	}
	cursor := clampSourceCursor(m.sourceTable.Cursor(), total)
	return fmt.Sprintf("%d total · %d active · selected %d of %d", total, len(source.ActiveSources(m.sourceItems)), cursor+1, total)
}

func sourceEntryHasListFocus(m Model) bool {
	return strings.TrimSpace(m.sourceInput.Value()) == "" && len(m.sourceItems) > 0
}

func clampSourceCursor(cursor int, total int) int {
	if total <= 0 || cursor < 0 {
		return 0
	}
	if cursor >= total {
		return total - 1
	}
	return cursor
}

func sourceEntryTableHeight(terminalHeight int, sourceCount int, actionRows int, warningCount int) int {
	rows := max(1, sourceCount) + 1
	if terminalHeight <= 0 {
		return min(8, max(3, rows))
	}
	warnings := 0
	if warningCount > 0 {
		warnings = min(4, warningCount) + 1
	}
	reserved := 14 + actionRows + warnings
	available := max(3, terminalHeight-reserved)
	return min(rows, available)
}

func sourcePendingPreview(preview source.Preview, pending string) string {
	if pending == "" {
		return styles.Subtitle.Render("No pending source. Paste a URL, article, file path, repo, or local document.")
	}
	counts := []string{
		fmt.Sprintf("web %d", preview.WebURLs),
		fmt.Sprintf("youtube %d", preview.YouTubeURLs),
		fmt.Sprintf("local %d", preview.LocalFiles),
		fmt.Sprintf("articles %d", preview.CapturedArticles),
		fmt.Sprintf("source docs %d", preview.Skills),
	}
	return styles.Section.Render("Pending: ") + strings.Join(counts, "  ")
}

func (m Model) viewSourceReview() string {
	width := styles.ClampWidth(m.width - 4)
	tableWidth := styles.ClampWidth(m.width - 8)
	counts := stagedCounts(m.sourceItems)
	columns := sourceColumns(tableWidth)
	reviewTable := newVisibleDataTable(
		columns,
		stagedRowsForColumns(m.sourceItems, columns, m.sourceTable.Cursor()),
		tableWidth,
		sourceReviewTableHeight(len(m.sourceItems)),
		true,
		m.sourceTable.Cursor(),
	)
	summary := []string{
		fmt.Sprintf("total %d", len(m.sourceItems)),
		fmt.Sprintf("active %d", len(source.ActiveSources(m.sourceItems))),
		fmt.Sprintf("web %d", counts["web"]),
		fmt.Sprintf("youtube %d", counts["youtube"]),
		fmt.Sprintf("local %d", counts["local"]),
		fmt.Sprintf("articles %d", counts["article"]),
		fmt.Sprintf("source docs %d", counts["source_doc"]),
		fmt.Sprintf("warnings %d", len(m.sourceWarnings)),
	}
	warnings := ""
	if len(m.sourceWarnings) > 0 {
		warnings = styles.ErrorText.Render(strings.Join(m.sourceWarnings[:min(5, len(m.sourceWarnings))], "\n"))
	}
	summaryView := styles.Section.Render(strings.Join(wrapWords(strings.Join(summary, "  "), width), "\n"))
	sections := []string{
		styles.Title.Render("Review User-Provided Sources"),
		styles.Subtitle.Render("Choose which Sources from the Source Inbox Liner should use."),
		summaryView,
		"",
	}
	if progress := m.sourceBatchProgressView(); progress != "" {
		sections = append(sections, progress, "")
	}
	sections = append(sections,
		reviewTable.View(),
		"",
		styles.ReportSection.Render("Selected"),
		m.sourceReviewSelectedDetail(width),
		"",
		styles.ReportSection.Render("Actions"),
		m.sourceReviewActionsTable(width).View(),
		warnings,
	)
	if m.sourceMaintenancePlan != nil {
		next := "Continue to Clarify Job"
		if m.sourceEntryReturnsToCompile() {
			next = "Return to Compile and retry with the saved Sources"
		}
		sections = append(sections, "", sourceApprovalView(width, *m.sourceMaintenancePlan, len(source.ActiveSources(m.sourceItems)), next))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func sourceApprovalView(width int, plan core.ProjectChangeSet, active int, next string) string {
	adds := 0
	unchanged := 0
	for _, operation := range plan.Operations {
		switch operation["type"] {
		case "source.add":
			adds++
		case "source.noop":
			unchanged++
		}
	}
	result := fmt.Sprintf("%d new %s", adds, pluralize(adds, "Source", "Sources"))
	if unchanged > 0 {
		result += fmt.Sprintf(" · %d already present", unchanged)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Ready to save Sources"),
		styles.AccentText.Render(fmt.Sprintf("Press Enter to save %d active %s for this Project.", active, pluralize(active, "Source", "Sources"))),
		renderLabelValueBlock(width, []labelValueRow{
			{Label: "Result", Value: result},
			{Label: "Change", Value: "Additive only; existing Sources stay unchanged"},
			{Label: "Next", Value: next},
		}, 0, 0),
	)
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (m Model) sourceBatchProgressView() string {
	if !m.sourceBatchRunning && m.sourceBatchPhase != sourceBatchPhaseCancelled && m.sourceBatchPhase != sourceBatchPhaseFailed {
		return ""
	}
	phase := map[string]string{
		sourceBatchPhasePlanning:   "Planning Core Change Set",
		sourceBatchPhaseValidation: "Validating atomic batch",
		sourceBatchPhaseApply:      "Atomic apply",
		sourceBatchPhaseCancelled:  "Cancelled before atomic apply",
		sourceBatchPhaseFailed:     "Paused on failure",
	}[m.sourceBatchPhase]
	lines := []string{
		styles.ReportSection.Render("Source batch"),
		styles.Subtitle.Render(fmt.Sprintf("%d/%d Sources prepared", m.sourceBatchPrepared, m.sourceBatchTotal)),
		styles.NextActionText.Render("Phase: " + phase),
	}
	if m.sourceBatchRunning {
		cue := "Press esc to cancel at the next safe boundary."
		if m.sourceBatchPhase == sourceBatchPhaseApply {
			cue = "Atomic apply cannot be interrupted. Press esc to stop after Core finishes or rolls back."
		}
		lines = append(lines, styles.MutedText.Render(cue))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func visibleSourceType(sourceType string) string {
	if sourceType == "skill" {
		return "source_doc"
	}
	return sourceType
}

func (m Model) sourceReviewActionsTable(width int) table.Model {
	return newActionTable(width, []actionTableRow{
		{Key: "enter", Action: "Save active sources", Writes: "Liner Core Change Set"},
		{Key: "space", Action: "Toggle selected source", Writes: "active flag"},
		{Key: "d", Action: "Remove selected source", Writes: "review list"},
		{Key: "a", Action: "Add more sources", Writes: "source inbox"},
	})
}

func maintenancePlanView(width int, plan core.ProjectChangeSet, approvalKey string) string {
	lines := core.MaintenancePreviewLines(plan)
	for index, line := range lines {
		lines[index] = strings.Join(wrapSynthesisReviewLine(line, width), "\n")
	}
	reviewRequirement := "Review this change, then press " + approvalKey + " to apply it."
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Core Change Set preview"),
		styles.AccentText.Render(reviewRequirement),
		styles.Subtitle.Render(strings.Join(lines, "\n")),
	)
}

func (m Model) sourceReviewSelectedDetail(width int) string {
	index := m.sourceTable.Cursor()
	if index < 0 || index >= len(m.sourceItems) {
		return styles.SoftText.Render("No source selected.")
	}
	item := m.sourceItems[index]
	state := "active"
	if !item.Active {
		state = "inactive"
	}
	return sourceMetadataTable(width, item, state, "")
}

func sourceMetadataTable(width int, item source.StagedSource, state string, reviewNote string) string {
	rows := []metadataTableRow{
		{Field: "Source", Value: item.Label},
		{Field: "Type", Value: visibleSourceType(fallbackText(item.Type, item.Source.Type))},
		{Field: "Status", Value: state},
		{Field: "Saved as", Value: fallbackText(item.Destination, "not saved yet")},
		{Field: "Priority", Value: fallbackText(item.Source.Priority, "required")},
		{Field: "Kind", Value: firstSourceText(item.Source.Kind, "unspecified")},
		{Field: "Section", Value: firstSourceText(item.Source.Section, "unsectioned")},
		{Field: "Note", Value: firstSourceText(item.Source.Note, "no note")},
	}
	if strings.TrimSpace(reviewNote) != "" {
		rows = append(rows, metadataTableRow{Field: "Review note", Value: reviewNote})
	}
	return newMetadataTable(width, rows).View()
}

func sourceReviewTableHeight(count int) int {
	return min(8, max(3, count+2))
}

func previewSources(project string, input string) tea.Cmd {
	return func() tea.Msg {
		preview, err := source.Import(input, project, false)
		return sourcePreviewMsg{preview: preview, err: err}
	}
}

func ingestSource(project string, input string) tea.Cmd {
	return func() tea.Msg {
		items, warnings, err := source.Ingest(input, project)
		return sourceIngestedMsg{items: items, warnings: warnings, err: err}
	}
}

func writeSourceManifest(project string, items []source.StagedSource) tea.Cmd {
	return func() tea.Msg {
		return sourceManifestSavedMsg{err: source.WriteManifests(project, items)}
	}
}

func saveActiveSources(runner core.Runner, project string, items []source.StagedSource, pending *core.ProjectChangeSet, approved bool) tea.Cmd {
	return func() tea.Msg {
		active := source.ActiveSources(items)
		plan, receipt, err := applySourceAdds(runner, project, active, pending, approved)
		if err == nil && plan == nil {
			err = source.WriteManifests(project, items)
		}
		return sourceSavedMsg{
			preview: source.Preview{Sources: active},
			plan:    plan,
			receipt: receipt,
			err:     err,
		}
	}
}

func (m Model) startInitialSourceBatch() (Model, tea.Cmd) {
	active := source.ActiveSources(m.sourceItems)
	if len(active) == 0 {
		m.err = "No active sources selected. Reactivate one or add more."
		return m, nil
	}
	m.sourceBatchRunID++
	runID := m.sourceBatchRunID
	m.sourceBatchRunning = true
	m.sourceBatchCancelRequested = false
	m.sourceBatchTotal = len(active)
	m.sourceBatchPrepared = len(active)
	m.err = ""
	if m.sourceMaintenancePlan != nil {
		if !m.sourceBatchPlanValidated {
			m.sourceBatchPhase = sourceBatchPhaseValidation
			m.note = "Validating the retained Core Change Set before atomic apply."
			return m, validateInitialSourceBatch(
				m.sourceItems,
				m.currentTape.Sources,
				m.currentProjectID(),
				*m.sourceMaintenancePlan,
				runID,
			)
		}
		m.sourceBatchPhase = sourceBatchPhaseApply
		m.note = "Applying the reviewed atomic Source batch. Core commit cannot be interrupted."
		return m, applyInitialSourceBatch(
			m.runner,
			m.currentPath,
			m.sourceItems,
			*m.sourceMaintenancePlan,
			m.sourceMaintenancePlan.ApprovalRequired,
			runID,
		)
	}
	m.sourceBatchPhase = sourceBatchPhasePlanning
	m.sourceBatchPlanValidated = false
	m.note = fmt.Sprintf("Preparing %d Sources and planning one atomic Core Change Set.", len(active))
	return m, planInitialSourceBatch(m.runner, m.currentPath, m.sourceItems, runID)
}

func planInitialSourceBatch(runner core.Runner, project string, items []source.StagedSource, runID uint64) tea.Cmd {
	return func() tea.Msg {
		active := source.ActiveSources(items)
		payloads := make([]map[string]any, 0, len(active))
		for _, item := range active {
			payloads = append(payloads, sourceMaintenancePayload(item))
		}
		plan, err := runner.PlanMaintenance(project, core.SourceBatchOperation(payloads))
		return sourceBatchPlannedMsg{
			preview: source.Preview{Sources: active},
			plan:    plan,
			err:     err,
			runID:   runID,
		}
	}
}

func validateInitialSourceBatch(
	items []source.StagedSource,
	existing []tape.Source,
	projectID *string,
	plan core.ProjectChangeSet,
	runID uint64,
) tea.Cmd {
	return func() tea.Msg {
		active := source.ActiveSources(items)
		err := validateInitialSourceBatchPlan(plan, active, existing, projectID)
		return sourceBatchValidatedMsg{
			preview: source.Preview{Sources: active},
			plan:    plan,
			err:     err,
			runID:   runID,
		}
	}
}

func validateInitialSourceBatchPlan(
	plan core.ProjectChangeSet,
	sources []tape.Source,
	existing []tape.Source,
	projectID *string,
) error {
	operationIndex := 0
	if operationIndex < len(plan.Operations) && plan.Operations[operationIndex]["type"] == "identity.assign_project" {
		if projectID != nil && strings.TrimSpace(*projectID) != "" {
			return fmt.Errorf("Core batch plan unexpectedly reassigns existing Project identity")
		}
		if assigned, _ := plan.Operations[operationIndex]["project_id"].(string); strings.TrimSpace(assigned) == "" || assigned != plan.ProjectID {
			return fmt.Errorf("Core batch plan assigns an unexpected Project identity")
		}
		operationIndex++
	}

	knownSourceIDs := make(map[string]string, len(existing)+len(sources))
	for index, item := range existing {
		assignedID := ""
		if item.ID != nil {
			assignedID = strings.TrimSpace(*item.ID)
		}
		if assignedID == "" {
			if operationIndex >= len(plan.Operations) {
				return fmt.Errorf("Core batch plan omits required identity for existing Source %d", index+1)
			}
			operation := plan.Operations[operationIndex]
			assignedIndex, ok := operationInteger(operation["index"])
			assignedID, _ = operation["source_id"].(string)
			if operation["type"] != "identity.assign_source" || !ok || assignedIndex != index || strings.TrimSpace(assignedID) == "" {
				return fmt.Errorf("Core batch plan has an unexpected identity operation for existing Source %d", index+1)
			}
			operationIndex++
		}
		knownSourceIDs[sourceIdentityKey(item)] = assignedID
	}

	if remaining := len(plan.Operations) - operationIndex; remaining != len(sources) {
		return fmt.Errorf("Core batch plan has %d Source outcomes for %d active Sources", remaining, len(sources))
	}
	for index, item := range sources {
		operation := plan.Operations[operationIndex+index]
		if operation["type"] != "source.add" && operation["type"] != "source.noop" {
			return fmt.Errorf("Core batch plan contains unrelated operation %q", operation["type"])
		}
		sourceID, _ := operation["source_id"].(string)
		if strings.TrimSpace(sourceID) == "" {
			return fmt.Errorf("Core batch plan Source outcome %d has no immutable Source ID", index+1)
		}
		switch operation["type"] {
		case "source.add":
			payload, ok := operation["source"].(map[string]any)
			if !ok || !reflect.DeepEqual(payload, sourceMaintenancePayload(item)) {
				return fmt.Errorf("Core batch plan operation %d does not match the reviewed Source", index+1)
			}
			identity := sourceIdentityKey(item)
			if knownID, exists := knownSourceIDs[identity]; exists && knownID != sourceID {
				return fmt.Errorf("Core batch plan adds duplicate reviewed Source %d instead of returning a no-op", index+1)
			}
			knownSourceIDs[identity] = sourceID
		case "source.noop":
			if operation["duplicate_classification"] != "exact_duplicate" {
				return fmt.Errorf("Core batch plan Source no-op %d is not an exact duplicate", index+1)
			}
			if knownSourceIDs[sourceIdentityKey(item)] != sourceID {
				return fmt.Errorf("Core batch plan Source no-op %d does not match the reviewed Source", index+1)
			}
		}
	}
	return nil
}

func (m Model) currentProjectID() *string {
	snapshot := m.currentProjectSnapshot()
	if snapshot == nil {
		return nil
	}
	return snapshot.ProjectID
}

func operationInteger(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	case string:
		integer, err := strconv.Atoi(typed)
		return integer, err == nil
	case json.Number:
		integer, err := strconv.Atoi(string(typed))
		return integer, err == nil
	}
	return 0, false
}

func sourceIdentityKey(item tape.Source) string {
	payload := sourceMaintenancePayload(item)
	if _, ok := payload["priority"]; !ok {
		payload["priority"] = "required"
	}
	if payload["type"] == "web" {
		if _, ok := payload["render"]; !ok {
			payload["render"] = "server"
		}
	}
	sourceType, _ := payload["type"].(string)
	if sourceType == "local_file" || sourceType == "skill" {
		if value, ok := payload["path"].(string); ok {
			payload["path"] = strings.TrimSpace(value)
		}
	}
	if sourceType == "local_file" {
		if value, ok := payload["citation"].(string); ok {
			payload["citation"] = strings.TrimSpace(value)
		}
	}
	if sourceType == "skill" {
		if value, ok := payload["url"].(string); ok {
			payload["url"] = strings.TrimSpace(value)
		}
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func applyInitialSourceBatch(runner core.Runner, project string, items []source.StagedSource, plan core.ProjectChangeSet, approved bool, runID uint64) tea.Cmd {
	return func() tea.Msg {
		active := source.ActiveSources(items)
		receipt, err := runner.ApplyMaintenance(project, plan, approved)
		var saved *core.ChangeReceipt
		if err == nil {
			saved = &receipt
			err = source.WriteManifests(project, items)
		}
		return sourceSavedMsg{
			preview: source.Preview{Sources: active},
			receipt: saved,
			err:     err,
			batch:   true,
			runID:   runID,
		}
	}
}

func applySourceAdds(runner core.Runner, project string, sources []tape.Source, pending *core.ProjectChangeSet, approved bool) (*core.ProjectChangeSet, *core.ChangeReceipt, error) {
	var lastReceipt *core.ChangeReceipt
	if pending != nil {
		receipt, err := runner.ApplyMaintenance(project, *pending, approved)
		if err != nil {
			return nil, nil, err
		}
		lastReceipt = &receipt
	}
	skippedAppliedSource := false
	for _, item := range sources {
		payload := sourceMaintenancePayload(item)
		if pending != nil && !skippedAppliedSource && changeSetAddsSource(*pending, payload) {
			skippedAppliedSource = true
			continue
		}
		plan, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", payload))
		if err != nil {
			return nil, lastReceipt, err
		}
		if plan.ApprovalRequired {
			return &plan, lastReceipt, nil
		}
		receipt, err := runner.ApplyMaintenance(project, plan, false)
		if err != nil {
			return nil, lastReceipt, err
		}
		lastReceipt = &receipt
	}
	return nil, lastReceipt, nil
}

func changeSetAddsSource(plan core.ProjectChangeSet, source map[string]any) bool {
	for _, operation := range plan.Operations {
		if operation["type"] != "source.add" {
			continue
		}
		candidate, ok := operation["source"].(map[string]any)
		if ok && reflect.DeepEqual(candidate, source) {
			return true
		}
	}
	return false
}

func sourceMaintenancePayload(item tape.Source) map[string]any {
	payload := map[string]any{"type": item.Type}
	if strings.TrimSpace(item.URL) != "" {
		payload["url"] = item.URL
	}
	for key, value := range map[string]*string{
		"path": item.Path, "citation": item.Citation, "note": item.Note,
		"section": item.Section, "render": item.Render, "kind": item.Kind,
		"content_hash": item.ContentHash,
	} {
		if value != nil {
			payload[key] = *value
		}
	}
	if strings.TrimSpace(item.Priority) != "" {
		payload["priority"] = item.Priority
	}
	return payload
}

func maintenanceReceiptNote(receipt *core.ChangeReceipt, fallback string) string {
	return fallback
}
