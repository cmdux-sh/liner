package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
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
	m.screen = screenSources
	m.sourceInput.Focus()
	if strings.TrimSpace(m.sourceInput.Value()) == "" {
		m.applySourcePreview(source.Preview{})
	}
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
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Title.Render("Review Local Sources"),
		styles.Subtitle.Render("Choose which pasted, local, and custom sources Liner should use."),
		summaryView,
		reviewTable.View(),
		"",
		styles.ReportSection.Render("Selected"),
		m.sourceReviewSelectedDetail(width),
		"",
		styles.ReportSection.Render("Actions"),
		m.sourceReviewActionsTable(width).View(),
		warnings,
	)
}

func visibleSourceType(sourceType string) string {
	if sourceType == "skill" {
		return "source_doc"
	}
	return sourceType
}

func (m Model) sourceReviewActionsTable(width int) table.Model {
	return newActionTable(width, []actionTableRow{
		{Key: "enter", Action: "Save active sources", Writes: "tape.yaml"},
		{Key: "space", Action: "Toggle selected source", Writes: "active flag"},
		{Key: "d", Action: "Remove selected source", Writes: "review list"},
		{Key: "a", Action: "Add more sources", Writes: "source inbox"},
	})
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

func saveSources(project string, input string) tea.Cmd {
	return func() tea.Msg {
		preview, err := source.Import(input, project, true)
		if err == nil {
			err = source.AppendToTape(project, preview.Sources)
		}
		return sourceSavedMsg{preview: preview, err: err}
	}
}

func saveActiveSources(project string, items []source.StagedSource) tea.Cmd {
	return func() tea.Msg {
		active := source.ActiveSources(items)
		err := source.WriteManifests(project, items)
		if err == nil {
			err = source.AppendToTape(project, active)
		}
		return sourceSavedMsg{
			preview: source.Preview{Sources: active},
			err:     err,
		}
	}
}
