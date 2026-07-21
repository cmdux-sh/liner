package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"gopkg.in/yaml.v3"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type capabilitySummary struct {
	HasLiner bool
	Skills   int
	Audits   int
	Evals    int
	Children int
	Lineage  bool
}

type projectPane struct {
	Title string
}

func newProjectTable(width int, height int) table.Model {
	return newDataTable(projectBrowserColumns(width), []table.Row{{"No projects found"}}, width, height, true)
}

func projectBrowserColumns(width int) []table.Column {
	return []table.Column{
		{Title: "Projects", Width: max(18, width-2)},
	}
}

func projectRows(items []projectItem, width int) []table.Row {
	nameWidth := projectBrowserColumns(width)[0].Width
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{
			truncateMiddle(fallbackText(item.project.Title, item.project.Name), nameWidth),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"No projects found"})
	}
	return rows
}

func filterProjectItems(items []projectItem, query string) []projectItem {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]projectItem(nil), items...)
	}
	filtered := make([]projectItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.FilterValue()), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (m *Model) applyHomeProjectFilter() {
	m.projectShown = filterProjectItems(m.projectItems, m.homeFilter)
	width := projectBrowserListWidth(m.width)
	m.projectTable.SetColumns(projectBrowserColumns(width))
	m.projectTable.SetWidth(width)
	m.projectTable.SetRows(projectRows(m.projectShown, width))
	if len(m.projectShown) == 0 {
		m.projectTable.SetCursor(0)
		return
	}
	if m.projectTable.Cursor() >= len(m.projectShown) {
		m.projectTable.SetCursor(len(m.projectShown) - 1)
	}
}

func homeProjectStatus(project core.ProjectSummary, summary capabilitySummary) string {
	switch {
	case projectCompileArtifactsNeedAttention(project.Path, nil, project.SourceCount):
		return "Compile Needs Attention"
	case summary.HasLiner:
		if state := savedProjectLifecycleState(project.Path); state.reviewRequired {
			return "Project Complete · Review Required"
		} else if state.stale {
			return "Project Complete · Stale"
		}
		return "Project Complete"
	case projectFileExists(project.Path, "MIXTAPE.md"):
		return "Corpus Ready"
	default:
		return "Started"
	}
}

type savedLifecycleState struct {
	stale          bool
	reviewRequired bool
}

func savedProjectLifecycleState(projectPath string) savedLifecycleState {
	raw, err := os.ReadFile(filepath.Join(projectPath, "liner.yaml"))
	if err != nil {
		return savedLifecycleState{}
	}
	var metadata struct {
		Status struct {
			Stale   bool `yaml:"stale"`
			Refresh struct {
				Synthesis struct {
					State string `yaml:"state"`
				} `yaml:"synthesis"`
				OperatingLayer struct {
					State string `yaml:"state"`
				} `yaml:"operating_layer"`
			} `yaml:"refresh"`
		} `yaml:"status"`
	}
	if yaml.Unmarshal(raw, &metadata) != nil {
		return savedLifecycleState{}
	}
	return savedLifecycleState{
		stale: metadata.Status.Stale,
		reviewRequired: metadata.Status.Refresh.Synthesis.State == "review_required" ||
			metadata.Status.Refresh.OperatingLayer.State == "review_required",
	}
}

func (m Model) primaryProjectAction() (Model, tea.Cmd) {
	if m.projectSnapshotDegraded() {
		m.err = "Project actions are read-only until Liner Core returns a trustworthy Project Snapshot. Press r to retry."
		return m, nil
	}
	if m.hasPendingAssemblyDraft() {
		if !m.projectMutationsAvailable() {
			m.err = "Liner Core reports this Project as read-only. Project writes are unavailable."
			return m, nil
		}
		return m.startPreparedAssemblyReview()
	}
	nextKind := m.projectNextKind()
	if nextKind != projectNextOpenLiner && !m.projectMutationsAvailable() {
		m.err = "Liner Core reports this Project as read-only. Project writes are unavailable."
		return m, nil
	}
	switch nextKind {
	case projectNextOpenLiner:
		return m.openPreview("LINER.md")
	case projectNextCreateOperatingLayer:
		return m.startLinerDraftReview()
	case projectNextReviewOperatingLayer:
		return m.startOperatingLayerReview()
	case projectNextRefreshStatus:
		m.projectSnapshotRefreshing = true
		m.note = "Refreshing the Core-owned Project status."
		m.err = ""
		return m, refreshProjectStatus(m.runner, m.currentPath)
	case projectNextReviewSynthesis:
		return m.startPreparedSynthesisReview()
	case projectNextCompileRefresh:
		return m.startCompile()
	}
	if m.currentProjectSnapshot() == nil && m.projectCompileNeedsAttention() {
		return m.startCompileReviewFromArtifacts()
	}
	if m.currentProjectSnapshot() == nil && m.isProjectComplete() {
		return m.openPreview("LINER.md")
	}
	if m.currentProjectSnapshot() == nil && m.hasCorpusReady() {
		return m.startLinerDraftReview()
	}
	if m.needsClarificationBeforeMethodology() {
		return m.startClarificationFlow()
	}
	if m.needsSourcesBeforeMethodology() {
		m.startSourceEntry()
		return m, nil
	}
	return m.startResearch()
}

func (m Model) hasPendingAssemblyDraft() bool {
	if strings.TrimSpace(m.currentPath) == "" {
		return false
	}
	info, err := os.Stat(projectAbsPath(m.currentPath, assemblyDraftRelPath))
	return err == nil && !info.IsDir()
}

func (m Model) hasCompiledMixtape() bool {
	if strings.TrimSpace(m.currentPath) == "" {
		return false
	}
	return projectFileExists(m.currentPath, "MIXTAPE.md")
}

func (m Model) projectCompileNeedsAttention() bool {
	if strings.TrimSpace(m.currentPath) == "" || !m.hasCompiledMixtape() {
		return false
	}
	return projectCompileArtifactsNeedAttention(m.currentPath, m.currentTape.Sources, len(m.currentTape.Sources))
}

func (m Model) hasCorpusReady() bool {
	milestone := m.projectMilestone()
	return milestone == "corpus_ready" || milestone == "project_complete"
}

func (m Model) canShowManualCompileAction() bool {
	if m.hasCompiledMixtape() || len(m.currentTape.Sources) > 0 {
		return true
	}
	return projectFileExists(m.currentPath, "synthesis.md")
}

func (m Model) needsSourcesBeforeMethodology() bool {
	if len(m.currentTape.Sources) > 0 || m.researchReady {
		return false
	}
	if m.currentTape.JTBD != nil && strings.TrimSpace(*m.currentTape.JTBD) != "" {
		return false
	}
	return true
}

func (m Model) primaryProjectActionIsSourceEntry() bool {
	if m.hasPendingAssemblyDraft() || m.hasCompiledMixtape() {
		return false
	}
	return m.needsSourcesBeforeMethodology()
}

func (m Model) projectCapabilities() capabilitySummary {
	project := strings.TrimSpace(m.currentPath)
	summary := capabilitiesForProject(project)
	if snapshot := m.currentProjectSnapshot(); snapshot != nil {
		summary.HasLiner = snapshot.Lifecycle.Milestone == "project_complete" || snapshot.Lifecycle.OperatingLayer.State == "ready" || snapshot.Lifecycle.OperatingLayer.State == "stale"
	}
	return summary
}

func capabilitiesForProject(project string) capabilitySummary {
	if project == "" {
		return capabilitySummary{}
	}
	return capabilitySummary{
		HasLiner: projectFileExists(project, "LINER.md"),
		Skills:   countTopLevelRegularFiles(filepath.Join(project, "skills"), ".md"),
		Audits:   countRegularFiles(filepath.Join(project, "working", "audits"), ".md"),
		Evals:    countRegularFiles(filepath.Join(project, "working", "evals"), ""),
		Children: countRegularFiles(filepath.Join(project, "children"), ""),
		Lineage:  projectFileExists(project, "lineage.yaml"),
	}
}

func projectFileExists(project string, rel string) bool {
	info, err := os.Stat(projectAbsPath(project, rel))
	return err == nil && !info.IsDir()
}

func projectDirExists(project string, rel string) bool {
	info, err := os.Stat(projectAbsPath(project, rel))
	return err == nil && info.IsDir()
}

func projectAbsPath(project string, rel string) string {
	if projectCorpusRelative(rel) {
		return filepath.Join(tape.ProjectAt(project).Path, rel)
	}
	return filepath.Join(project, rel)
}

func displayProjectPath(project string, rel string) string {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if strings.TrimSpace(project) == "" {
		return rel
	}
	if projectCorpusRelative(rel) {
		abs := projectAbsPath(project, rel)
		if relative, err := filepath.Rel(project, abs); err == nil && relative != "." && !strings.HasPrefix(relative, "..") {
			return filepath.ToSlash(relative)
		}
	}
	return rel
}

func projectCorpusPath(project string) string {
	return tape.ProjectAt(project).Path
}

func projectCorpusRelative(rel string) bool {
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == "." || strings.HasPrefix(clean, "../") {
		return false
	}
	if clean == "MIXTAPE.md" || clean == "tape.yaml" || clean == "synthesis.md" ||
		clean == "sources" || strings.HasPrefix(clean, "sources/") ||
		clean == "local-sources" || strings.HasPrefix(clean, "local-sources/") ||
		clean == ".liner-progress.json" || clean == ".liner-gates.json" ||
		clean == ".liner-runs" || strings.HasPrefix(clean, ".liner-runs/") {
		return true
	}
	return strings.HasPrefix(clean, "working/0") || strings.HasPrefix(clean, "working/evaluation-decisions/")
}

func countRegularFiles(dir string, extension string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count += countRegularFiles(filepath.Join(dir, entry.Name()), extension)
			continue
		}
		if extension != "" && !strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			continue
		}
		count++
	}
	return count
}

func countTopLevelRegularFiles(dir string, extension string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if extension != "" && !strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			continue
		}
		count++
	}
	return count
}

func (m Model) viewProjects() string {
	width := styles.ClampWidth(m.width - 4)
	listWidth := projectBrowserListWidth(m.width)
	gutterWidth := 2
	detailWidth := max(24, width-listWidth-gutterWidth)
	listHeight := projectBrowserHeight(m.height, len(m.projectShown))
	projectTable := m.projectTable
	projectTable.SetWidth(listWidth)
	projectTable.SetHeight(listHeight)
	projectTable.SetColumns(projectBrowserColumns(listWidth))
	projectTable.SetRows(projectRows(m.projectShown, listWidth))
	projectTable.SetCursor(m.projectTable.Cursor())
	leftLines := []string{
		styles.Title.Render("Projects"),
		styles.Subtitle.Render("Open and manage Liner projects"),
		"",
	}
	if filter := projectBrowserFilterLine(m.homeFilter, m.homeFiltering); filter != "" {
		leftLines = append(leftLines, filter)
	}
	leftLines = append(leftLines, projectBrowserListView(m.projectShown, projectTable.Cursor(), listWidth, listHeight))
	left := lipgloss.NewStyle().Width(listWidth).Render(lipgloss.JoinVertical(lipgloss.Left, leftLines...))
	right := lipgloss.NewStyle().Width(detailWidth).Render(m.projectBrowserSelectedDetail(detailWidth))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gutterWidth), right)
}

func projectBrowserListView(items []projectItem, cursor int, width int, height int) string {
	width = max(18, width)
	if len(items) == 0 {
		return styles.MutedText.Width(width).Render("No projects found")
	}
	height = max(1, height)
	cursor = max(0, min(cursor, len(items)-1))
	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	if start+height > len(items) {
		start = max(0, len(items)-height)
	}
	end := min(len(items), start+height)
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		title := truncateMiddle(fallbackText(items[index].project.Title, items[index].project.Name), max(1, width-1))
		line := " " + title
		if index == cursor {
			lines = append(lines, styles.TableSelectedFocused.Width(width).Render(line))
			continue
		}
		lines = append(lines, styles.PrimaryText.Width(width).Render(line))
	}
	return strings.Join(lines, "\n")
}

func projectBrowserListWidth(width int) int {
	return min(34, max(24, styles.ClampWidth(width-4)/3))
}

func projectBrowserHeight(height int, rowCount int) int {
	contentHeight := max(8, min(18, rowCount+2))
	if height <= 0 {
		return contentHeight
	}
	return min(contentHeight, max(8, height-12))
}

func (m Model) selectedProjectItem() (projectItem, bool) {
	index := m.projectTable.Cursor()
	if index < 0 || index >= len(m.projectShown) {
		return projectItem{}, false
	}
	return m.projectShown[index], true
}

func homeFilterLine(query string, active bool) string {
	label := "All projects"
	if strings.TrimSpace(query) != "" {
		label = query
	}
	if active {
		return fmt.Sprintf("%s  %s", styles.ReportSection.Render("Filter"), styles.AccentText.Render(label))
	}
	return fmt.Sprintf("%s  %s", styles.ReportSection.Render("Filter"), styles.Subtitle.Render(label))
}

func projectBrowserFilterLine(query string, active bool) string {
	if !active && strings.TrimSpace(query) == "" {
		return ""
	}
	return homeFilterLine(query, active)
}

func (m Model) projectBrowserSelectedDetail(width int) string {
	item, ok := m.selectedProjectItem()
	if !ok {
		return renderLabelValueBlock(width, []labelValueRow{
			{Label: "Selected", Value: "No project selected."},
			{Label: "Library", Value: m.baseDir},
		}, 0, 0)
	}
	title := fallbackText(item.project.Title, item.project.Name)
	jtbd := "not set"
	if item.project.JTBD != nil && strings.TrimSpace(*item.project.JTBD) != "" {
		jtbd = strings.TrimSpace(*item.project.JTBD)
	}
	return renderLabelValueBlock(width, []labelValueRow{
		{Label: "Name", Value: title},
		{Label: "Status", Value: homeProjectStatus(item.project, item.capabilities)},
		{Label: "Description", Value: projectSummaryDescription(item.project)},
		{Label: "Job", Value: jtbd, MaxLines: 3},
		{Label: "Folder", Value: item.project.Path},
	}, 0, 0)
}

func (m Model) viewSplash() string {
	width := chromeWidth(m.width)
	title := styles.BrandDot.Render("o") + " " + styles.Brand.Render("liner")
	lines := []string{
		title,
		"",
		styles.Title.Render("The right context"),
		styles.Subtitle.Render("for the right job."),
		"",
		styles.SoftText.Render("Build one focused mixtape from the sources and files that matter."),
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m Model) viewProject() string {
	width := styles.ClampWidth(m.width - 4)
	title := fallbackText(m.currentTape.Title, "Untitled Liner")
	description := projectTapeDescription(m.currentTape)
	parts := []string{
		styles.Title.Render(truncateMiddle(title, width)),
		lipgloss.NewStyle().Width(width).Render(styles.Subtitle.Render(strings.Join(wrapLabelValue(description, width), "\n"))),
		"",
	}
	if m.projectSnapshotDegraded() {
		status := "Core Project Snapshot unavailable"
		if m.projectSnapshotLoading {
			status = "Loading Core Project Snapshot"
		}
		parts = append(parts, projectSectionDetail(
			width,
			"Project status",
			"This Project is read-only until Liner Core returns trustworthy lifecycle state.",
			newMetadataTable(width, []metadataTableRow{
				{Field: "Status", Value: status},
				{Field: "Diagnostic", Value: m.projectSnapshotDiagnostic()},
				{Field: "Core binary", Value: m.projectCoreBinaryValue(width)},
				{Field: "Action", Value: "Retry"},
			}).View(),
		))
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}
	if width >= 92 {
		listWidth := min(30, max(22, width/4))
		detailWidth := max(40, width-listWidth-4)
		parts = append(parts, lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(listWidth).Render(m.projectPaneList(listWidth)),
			lipgloss.NewStyle().Width(detailWidth).PaddingLeft(2).Render(m.projectPaneDetail(detailWidth)),
		))
	} else {
		parts = append(parts, m.projectPaneList(width), "", m.projectPaneDetail(width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) projectPaneList(width int) string {
	panes := m.projectPanes()
	if len(panes) == 0 {
		return ""
	}
	index := clampProjectPaneIndex(m.projectPane, len(panes))
	lines := []string{
		styles.ReportSection.Render("Sections"),
	}
	for i, pane := range panes {
		marker := " "
		titleStyle := styles.PrimaryText
		if i == index {
			marker = ">"
			titleStyle = styles.AccentText
		}
		titleWidth := max(8, width-2)
		line := fmt.Sprintf("%s %s", marker, titleStyle.Render(truncateMiddle(pane.Title, titleWidth)))
		lines = append(lines, line)
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (m Model) projectPanes() []projectPane {
	return []projectPane{
		{Title: "Health"},
		{Title: "Flow"},
		{Title: "Sources"},
		{Title: "Artifacts"},
		{Title: "Operating Layer"},
		{Title: "Usage"},
	}
}

func (m Model) projectPaneDetail(width int) string {
	panes := m.projectPanes()
	index := clampProjectPaneIndex(m.projectPane, len(panes))
	switch index {
	case 1:
		return m.projectFlowDetail(width)
	case 2:
		return m.projectSourcesDetail(width)
	case 3:
		return m.projectArtifactsDetail(width)
	case 4:
		return m.projectCapabilitiesDetail(width)
	case 5:
		return m.projectUsageDetail(width)
	default:
		return m.projectHealthDetail(width)
	}
}

func clampProjectPaneIndex(index int, count int) int {
	if count <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
}

func (m Model) projectHealthDetail(width int) string {
	done, total, next := m.projectProgressCounts()
	if m.projectNextKind() == projectNextReviewOperatingLayer {
		next = "Review Operating Layer"
	} else if m.projectNextKind() == projectNextReviewSynthesis {
		next = "Review Synthesis"
	}
	progress := "not started"
	if total > 0 {
		progress = fmt.Sprintf("%d of %d steps complete", done, total)
	}
	if next == "" && total > 0 && done >= total {
		next = "No corpus build steps missing."
	}
	rows := []metadataTableRow{
		{Field: "Status", Value: m.projectStatusLabel()},
		{Field: "Primary action", Value: m.projectPrimaryLabel()},
		{Field: "Project flow", Value: progress},
		{Field: "Missing next", Value: next},
		{Field: "Status source", Value: m.projectStatusSourceLabel()},
	}
	if summary, ok := readEvaluationIssueSummary(m.currentPath, m.currentTape); ok {
		rows = append(rows, metadataTableRow{Field: "Evaluation issues", Value: summary.Display(m.currentPath)})
	}
	if m.shouldShowProviderReadiness() {
		rows = append(rows, metadataTableRow{Field: "AI runner", Value: m.projectProviderReadiness()})
	}
	if m.statusErr != "" {
		rows = append(rows,
			metadataTableRow{Field: "Status note", Value: "status failed; using local project files"},
			metadataTableRow{Field: "Status cause", Value: m.statusErr},
			metadataTableRow{Field: "Core binary", Value: m.projectCoreBinaryValue(width)},
		)
	} else if m.currentProjectStatus() == nil && projectPipelineHasSignal(m.currentPath) {
		rows = append(rows, metadataTableRow{Field: "Status note", Value: "using local files until liner status returns evidence"})
	}
	rows = append(rows,
		metadataTableRow{Field: "Sources", Value: fmt.Sprintf("%d", len(m.currentTape.Sources))},
		metadataTableRow{Field: "Local source files", Value: m.projectLocalSourcesValue()},
		metadataTableRow{Field: "Folder", Value: m.currentPath},
	)
	return projectSectionDetail(
		width,
		"Health",
		"Shows readiness, the current primary action, corpus evidence, and local project files.",
		newMetadataTable(width, rows).View(),
	)
}

func (m Model) projectFlowDetail(width int) string {
	rows := m.projectFlowRows()
	if len(rows) == 0 {
		return styles.SoftText.Render("No project flow found yet.")
	}
	phaseWidth := min(22, max(16, width/3))
	stateWidth := 10
	evidenceWidth := max(22, width-phaseWidth-stateWidth-8)
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, table.Row{
			truncateMiddle(row.Phase, phaseWidth),
			truncateMiddle(row.State, stateWidth),
			truncateMiddle(row.Evidence, evidenceWidth),
		})
	}
	done, total, next := projectProgressCountsFromRows(rows)
	summary := fmt.Sprintf("%d of %d steps complete", done, total)
	if next != "" {
		summary += "; next: " + next
	}
	return projectSectionDetail(
		width,
		"Flow",
		summary,
		newDataTable(
			[]table.Column{
				{Title: "Step", Width: phaseWidth},
				{Title: "Status", Width: stateWidth},
				{Title: "Evidence", Width: evidenceWidth},
			},
			tableRows,
			width,
			len(tableRows)+1,
			false,
		).View(),
	)
}

func (m Model) projectFlowRows() []projectPipelineRow {
	if snapshot := m.currentProjectSnapshot(); snapshot != nil {
		return projectFlowRowsFromSnapshot(*snapshot)
	}
	startedState := "done"
	startedEvidence := "project folder"
	corpusState := "queued"
	corpusEvidence := "Build Corpus"
	if m.projectCompileNeedsAttention() {
		corpusState = "current"
		corpusEvidence = "review compile issues"
	} else if m.hasCorpusReady() {
		corpusState = "done"
		corpusEvidence = displayProjectPath(m.currentPath, "MIXTAPE.md")
	} else if !m.isProjectComplete() {
		corpusState = "current"
		if m.needsClarificationBeforeMethodology() {
			corpusEvidence = "complete Clarify Job"
		} else if m.primaryProjectActionIsSourceEntry() {
			corpusEvidence = "add sources"
		}
	}
	operatingState := "queued"
	operatingEvidence := "LINER.md + Project Skill"
	if m.projectCapabilities().HasLiner && m.projectSkillReady() {
		operatingState = "done"
		operatingEvidence = "LINER.md + " + m.projectSkillDisplayPath()
	} else if m.hasCorpusReady() {
		operatingState = "current"
	}
	completeState := "queued"
	completeEvidence := "liner.yaml"
	if m.isProjectComplete() {
		completeState = "done"
	}
	return []projectPipelineRow{
		{Phase: "Project Shell", State: startedState, Evidence: startedEvidence},
		{Phase: "Corpus Ready", State: corpusState, Evidence: corpusEvidence, Current: corpusState == "current"},
		{Phase: "Create Operating Layer", State: operatingState, Evidence: operatingEvidence, Current: operatingState == "current"},
		{Phase: "Project Complete", State: completeState, Evidence: completeEvidence, Current: completeState == "current"},
	}
}

func projectFlowRowsFromSnapshot(snapshot core.MaintenanceProjectSnapshot) []projectPipelineRow {
	lifecycle := snapshot.Lifecycle
	corpusState := "current"
	corpusEvidence := "Continue Corpus Creation"
	operatingState := "queued"
	operatingEvidence := "LINER.md + Project Skill"
	completeState := "queued"
	if lifecycle.Milestone == "corpus_ready" || lifecycle.Milestone == "project_complete" {
		corpusState = "done"
		corpusEvidence = lifecycle.Corpus.Evidence
		operatingState = "current"
	}
	if lifecycle.Milestone == "project_complete" {
		operatingState = "done"
		operatingEvidence = lifecycle.OperatingLayer.Evidence
		completeState = "done"
	}
	if lifecycle.Stale {
		if lifecycle.Corpus.State == "stale" {
			corpusState = "stale"
		}
		if lifecycle.OperatingLayer.State == "stale" {
			operatingState = "stale"
		}
		if lifecycle.Refresh != nil {
			switch {
			case lifecycle.Refresh.Synthesis.State == "review_required":
				corpusEvidence = "Review Synthesis required"
			case lifecycle.Refresh.Corpus.State == "compile_required":
				corpusEvidence = "Compile refresh required"
			}
			if lifecycle.Refresh.OperatingLayer.State == "review_required" {
				operatingEvidence = "Review Operating Layer required"
			}
		}
	}
	return []projectPipelineRow{
		{Phase: "Project Shell", State: "done", Evidence: "Core Project Snapshot"},
		{Phase: "Corpus Ready", State: corpusState, Evidence: corpusEvidence, Current: corpusState == "current" || corpusState == "stale"},
		{Phase: "Create Operating Layer", State: operatingState, Evidence: operatingEvidence, Current: operatingState == "current" || operatingState == "stale"},
		{Phase: "Project Complete", State: completeState, Evidence: "liner.yaml"},
	}
}

func (m Model) projectSourcesDetail(width int) string {
	rows := []metadataTableRow{
		{Field: "Saved sources", Value: fmt.Sprintf("%d", len(m.currentTape.Sources))},
		{Field: "Local source files", Value: m.projectLocalSourcesValue()},
		{Field: "Synthesis", Value: optionalArtifactNote(m.currentPath, "synthesis.md")},
		{Field: "Quality checks", Value: optionalArtifactNote(m.currentPath, filepath.Join("working", "04-quality-checks.md"))},
	}
	if summary, ok := readEvaluationIssueSummary(m.currentPath, m.currentTape); ok {
		rows = append(rows, metadataTableRow{Field: "Dropped candidates", Value: summary.Display(m.currentPath)})
	}
	body := newMetadataTable(width, rows).View()
	if excluded := readExcludedLocalSourceIssues(m.currentPath, m.currentTape); len(excluded) > 0 {
		body += "\n\n" + renderExcludedLocalSources(width, excluded)
	}
	return projectSectionDetail(
		width,
		"Sources",
		"Shows the source set Liner will use when it builds and refreshes the Mixtape.",
		body,
	)
}

func (m Model) projectArtifactsDetail(width int) string {
	rows := m.projectArtifactRows()
	fileWidth := min(26, max(16, width/3))
	statusWidth := 10
	noteWidth := max(22, width-fileWidth-statusWidth-8)
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, table.Row{
			truncateMiddle(row.Field, fileWidth),
			truncateMiddle(row.Value, statusWidth),
			truncateMiddle(row.Detail, noteWidth),
		})
	}
	return projectSectionDetail(
		width,
		"Artifacts",
		"Shows the local files that make a Liner project usable after the corpus is ready.",
		newDataTable(
			[]table.Column{
				{Title: "File", Width: fileWidth},
				{Title: "Status", Width: statusWidth},
				{Title: "Notes", Width: noteWidth},
			},
			tableRows,
			width,
			len(tableRows)+1,
			false,
		).View(),
	)
}

type projectArtifactRow struct {
	Field  string
	Value  string
	Detail string
}

func (m Model) projectArtifactRows() []projectArtifactRow {
	skillPath := m.projectSkillDisplayPath()
	return []projectArtifactRow{
		{Field: displayProjectPath(m.currentPath, "MIXTAPE.md"), Value: readyMissing(m.hasCompiledMixtape()), Detail: "compiled context packet"},
		{Field: "LINER.md", Value: readyMissing(m.projectCapabilities().HasLiner), Detail: "instructions for AI sessions"},
		{Field: skillPath, Value: readyMissing(m.projectSkillReady()), Detail: "small local skill for loading this project"},
		{Field: "liner.yaml", Value: readyMissing(projectFileExists(m.currentPath, "liner.yaml") || m.isProjectComplete()), Detail: "local project status"},
	}
}

func readyMissing(ready bool) string {
	if ready {
		return "ready"
	}
	return "missing"
}

func (m Model) projectUsageDetail(width int) string {
	pendingDraft := m.hasPendingAssemblyDraft()
	compiled := m.hasCompiledMixtape()
	sourceEntryPrimary := m.primaryProjectActionIsSourceEntry()
	if usageView := projectRunUsageView(m.currentPath, width); usageView != "" {
		return projectSectionDetail(
			width,
			"Usage",
			"Shows token usage and cost evidence from local corpus builds.",
			usageView,
		)
	}
	if estimateView := m.projectRunEstimateView(width, pendingDraft, compiled, sourceEntryPrimary); estimateView != "" {
		return projectSectionDetail(
			width,
			"Usage",
			"Estimates the next corpus build from local or seeded token history.",
			estimateView,
		)
	}
	return projectSectionDetail(
		width,
		"Usage",
		"Shows run usage when available; otherwise estimates upcoming corpus-build cost.",
		styles.SoftText.Render("No run usage or estimate available yet."),
	)
}

func (m Model) projectCapabilitiesDetail(width int) string {
	return projectSectionDetail(
		width,
		"Operating Layer",
		"Tracks LINER.md and root SKILL.md for this local Liner project.",
		m.projectCapabilitiesSummaryTable(width),
	)
}

func projectSectionDetail(width int, title string, description string, content string) string {
	parts := []string{styles.ReportSection.Render(title)}
	if strings.TrimSpace(description) != "" {
		parts = append(parts, styles.Subtitle.Render(strings.Join(wrapLabelValue(description, width), "\n")))
	}
	if strings.TrimSpace(content) != "" {
		parts = append(parts, "", content)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) projectProgressCounts() (int, int, string) {
	if m.currentProjectSnapshot() != nil {
		return projectProgressCountsFromRows(m.projectFlowRows())
	}
	rows := projectPipelineRows(m.currentPath, m.currentTape, m.currentProjectStatus())
	done, total, next := projectProgressCountsFromRows(rows)
	if m.needsClarificationBeforeMethodology() {
		next = "Clarify Job"
	}
	return done, total, next
}

func projectProgressCountsFromRows(rows []projectPipelineRow) (int, int, string) {
	done := 0
	next := ""
	for _, row := range rows {
		if row.State == "done" {
			done++
			continue
		}
		if next == "" {
			next = row.Phase
		}
	}
	return done, len(rows), next
}

func (m Model) projectStatusLabel() string {
	label := projectMilestoneLabel(m.projectMilestone())
	if snapshot := m.currentProjectSnapshot(); snapshot != nil && snapshot.Lifecycle.Stale {
		label += " (stale)"
	} else if status := m.currentProjectStatus(); status != nil && status.Snapshot.Stale {
		label += " (stale)"
	}
	return label
}

func (m Model) projectPrimaryLabel() string {
	nextKind := m.projectNextKind()
	if nextKind == projectNextUnavailable {
		if m.currentProjectSnapshot() != nil {
			return "Read-only Core guidance"
		}
		return "Retry Project Snapshot"
	}
	if m.hasPendingAssemblyDraft() {
		return "Review draft sources"
	}
	switch nextKind {
	case projectNextOpenLiner:
		return "Open LINER.md"
	case projectNextCreateOperatingLayer:
		return "Create Operating Layer"
	case projectNextReviewOperatingLayer:
		return "Review Operating Layer"
	case projectNextRefreshStatus:
		return "Refresh Status"
	case projectNextReviewSynthesis:
		return "Review Synthesis"
	case projectNextCompileRefresh:
		return "Compile refreshed corpus"
	}
	switch {
	case m.projectCompileNeedsAttention():
		return "Review compile issues"
	case m.isProjectComplete():
		return "Open LINER.md"
	case m.hasCorpusReady():
		return "Create Operating Layer"
	case m.needsClarificationBeforeMethodology():
		return "Continue Clarify Job"
	case m.primaryProjectActionIsSourceEntry():
		return "Add sources"
	default:
		return "Continue Corpus Creation"
	}
}

func (m Model) projectMilestoneNextAction() string {
	nextKind := m.projectNextKind()
	if nextKind == projectNextUnavailable {
		return ""
	}
	if m.hasPendingAssemblyDraft() {
		return "Review the assembly draft sources."
	}
	switch nextKind {
	case projectNextOpenLiner:
		return projectCompleteNextAction
	case projectNextCreateOperatingLayer:
		return "Create Operating Layer."
	case projectNextReviewOperatingLayer:
		return "Review Operating Layer."
	case projectNextRefreshStatus:
		return "Refresh Status through Liner Core."
	case projectNextReviewSynthesis:
		return "Review Synthesis before Compile."
	case projectNextCompileRefresh:
		return "Compile the reviewed corpus refresh."
	}
	switch m.projectMilestone() {
	case "project_complete":
		return projectCompleteNextAction
	case "corpus_ready":
		return "Create Operating Layer."
	case "compile_attention":
		return "Review compile issues and retry compile."
	default:
		if m.needsClarificationBeforeMethodology() {
			return "Complete Clarify Job before building the corpus."
		}
		if m.needsSourcesBeforeMethodology() {
			return "Continue Corpus Creation: add sources."
		}
		return "Continue Corpus Creation."
	}
}

func (m Model) projectMilestone() string {
	if snapshot := m.currentProjectSnapshot(); snapshot != nil {
		return snapshot.Lifecycle.Milestone
	}
	if m.projectCompileNeedsAttention() {
		return "compile_attention"
	}
	if status := m.currentProjectStatus(); status != nil {
		switch status.Snapshot.Milestone {
		case "started", "corpus_ready", "project_complete":
			return status.Snapshot.Milestone
		}
	}
	if m.projectCapabilities().HasLiner && m.projectSkillReady() {
		return "project_complete"
	}
	if m.hasCompiledMixtape() {
		return "corpus_ready"
	}
	return "started"
}

func projectMilestoneLabel(milestone string) string {
	switch milestone {
	case "project_complete":
		return "Project Complete"
	case "corpus_ready":
		return "Corpus Ready"
	case "compile_attention":
		return "Compile Needs Attention"
	default:
		return "Started"
	}
}

func (m Model) isProjectComplete() bool {
	return m.projectMilestone() == "project_complete" && m.projectCapabilities().HasLiner
}

func (m Model) projectSkillReady() bool {
	return m.projectSkillStatus() == "active"
}

func (m Model) projectSkillStatus() string {
	if snapshot := m.currentProjectSnapshot(); snapshot != nil {
		return snapshot.Lifecycle.ProjectSkill.Status
	}
	if status := m.currentProjectStatus(); status != nil {
		if status.ProjectSkill.Status == "active" {
			return "active"
		}
	}
	return "missing"
}

func (m Model) projectStatusSourceLabel() string {
	if m.currentProjectSnapshot() != nil {
		return "Core Project Snapshot"
	}
	if status := m.currentProjectStatus(); status != nil {
		source := strings.TrimSpace(status.Progress.Source)
		if source == "" {
			source = "status"
		}
		return "liner status (" + source + ")"
	}
	if m.statusErr != "" {
		return "local fallback; status failed"
	}
	if projectPipelineHasSignal(m.currentPath) {
		return "local project files"
	}
	return "no corpus evidence yet"
}

func (m Model) shouldShowProviderReadiness() bool {
	if m.hasPendingAssemblyDraft() || m.hasCompiledMixtape() || m.primaryProjectActionIsSourceEntry() {
		return false
	}
	return true
}

func (m Model) projectProviderReadiness() string {
	info := m.settings
	if strings.TrimSpace(info.ConfigPath) == "" {
		return "Not checked; open Settings to choose one"
	}
	active := info.activeAgent()
	if active != "" {
		return info.activeProviderLabel()
	}
	if len(info.Installed) > 0 {
		return info.activeProviderLabel() + "; choose a runner in Settings"
	}
	return "None found; install the Claude Code or Codex CLI, then open Settings"
}

func (m Model) projectCoreBinaryValue(width int) string {
	command := strings.TrimSpace(m.runner.Command)
	if command == "" {
		return "not resolved"
	}
	if len(m.runner.Args) > 0 {
		command += " " + strings.Join(m.runner.Args, " ")
	}
	return truncateMiddle(command, max(16, width-lipgloss.Width("Core binary  ")))
}

func (m *Model) moveProjectPane(delta int) {
	count := len(m.projectPanes())
	if count == 0 {
		m.projectPane = 0
		return
	}
	m.projectPane = clampProjectPaneIndex(m.projectPane+delta, count)
}

func (m Model) projectLocalSourcesValue() string {
	value := "not created yet"
	if m.canOpenLocalSources() {
		value = projectAbsPath(m.currentPath, "local-sources")
	}
	return value
}

type capabilityReadinessRow struct {
	Area   string
	Status string
	Next   string
}

func (m Model) projectCapabilitiesSummaryTable(width int) string {
	rows := m.projectCapabilityReadinessRows()
	width = max(width, 60)
	usableWidth := max(40, width-8)
	areaWidth := min(18, max(12, usableWidth/4))
	statusWidth := min(24, max(14, usableWidth/3))
	nextWidth := max(20, usableWidth-areaWidth-statusWidth)
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, table.Row{
			truncateMiddle(row.Area, areaWidth),
			truncateMiddle(row.Status, statusWidth),
			truncateMiddle(row.Next, nextWidth),
		})
	}
	return newDataTable(
		[]table.Column{
			{Title: "Area", Width: areaWidth},
			{Title: "Status", Width: statusWidth},
			{Title: "Next", Width: nextWidth},
		},
		tableRows,
		width,
		len(tableRows)+1,
		false,
	).View()
}

func (m Model) projectCapabilityReadinessRows() []capabilityReadinessRow {
	summary := m.projectCapabilities()
	return []capabilityReadinessRow{
		m.linerReadinessRow(summary),
		m.projectSkillReadinessRow(),
	}
}

func (m Model) linerReadinessRow(summary capabilitySummary) capabilityReadinessRow {
	if summary.HasLiner {
		return capabilityReadinessRow{Area: "LINER.md", Status: "ready", Next: "Preview or regenerate"}
	}
	if m.projectCompileNeedsAttention() {
		return capabilityReadinessRow{Area: "LINER.md", Status: "blocked", Next: "Review compile issues"}
	}
	if m.hasCorpusReady() {
		return capabilityReadinessRow{Area: "LINER.md", Status: "missing", Next: "Create Operating Layer"}
	}
	return capabilityReadinessRow{Area: "LINER.md", Status: "blocked", Next: "Reach Corpus Ready first"}
}

func (m Model) projectSkillReadinessRow() capabilityReadinessRow {
	status := m.projectSkillStatus()
	if status == "active" {
		name := "active"
		if projectStatus := m.currentProjectStatus(); projectStatus != nil && projectStatus.ProjectSkill.Name != nil && strings.TrimSpace(*projectStatus.ProjectSkill.Name) != "" {
			name = *projectStatus.ProjectSkill.Name
		}
		return capabilityReadinessRow{Area: "Project Skill", Status: name, Next: "Represented in LINER.md"}
	}
	if m.projectCompileNeedsAttention() {
		return capabilityReadinessRow{Area: "Project Skill", Status: "blocked", Next: "Review compile issues"}
	}
	if m.hasCorpusReady() {
		return capabilityReadinessRow{Area: "Project Skill", Status: "missing", Next: "Create Operating Layer"}
	}
	return capabilityReadinessRow{Area: "Project Skill", Status: "blocked", Next: "Reach Corpus Ready first"}
}

func (m Model) projectSkillDisplayPath() string {
	if projectStatus := m.currentProjectStatus(); projectStatus != nil && projectStatus.ProjectSkill.Path != nil && strings.TrimSpace(*projectStatus.ProjectSkill.Path) != "" {
		return filepath.ToSlash(strings.TrimSpace(*projectStatus.ProjectSkill.Path))
	}
	_, skillPath := projectSkillProposal(m.currentTape)
	return filepath.ToSlash(skillPath)
}

func loadProjects(r core.Runner, baseDir string) tea.Cmd {
	return func() tea.Msg {
		projects, err := r.ListProjects(baseDir)
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func loadProjectStatus(r core.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		status, err := r.ProjectStatus(path)
		return projectStatusLoadedMsg{path: path, status: status, err: err}
	}
}

func openProject(path string) tea.Cmd {
	return func() tea.Msg {
		t, err := tape.ReadProject(path)
		return projectOpenedMsg{path: path, tape: t, err: err}
	}
}

func researchReportExists(project string) bool {
	info, err := os.Stat(filepath.Join(project, "research-report.md"))
	return err == nil && !info.IsDir()
}
