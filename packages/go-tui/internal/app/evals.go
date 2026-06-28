package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type evalFile struct {
	Name    string
	RelPath string
	Path    string
	Area    string
	Updated string
}

func newEvalTable(width int, height int) table.Model {
	return newDataTable(evalColumns(width), []table.Row{}, width, height, true)
}

func evalColumns(width int) []table.Column {
	width = max(width, 64)
	nameWidth := max(18, width-42)
	return []table.Column{
		{Title: "Artifact", Width: nameWidth},
		{Title: "Area", Width: 16},
		{Title: "Updated", Width: 12},
	}
}

func (m Model) startEvals() (Model, tea.Cmd) {
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if len(items) == 0 && !m.canCreateEvalTaskset() {
		m.err = "No impact-test artifacts found in working/evals/."
		return m, nil
	}
	m.evalItems = items
	m.applyEvalItems(items)
	m.screen = screenEvals
	if len(items) == 0 {
		m.note = "No impact-test artifacts yet."
	} else {
		m.note = "Loaded " + intLabel(len(items), "impact-test artifact") + "."
	}
	return m, nil
}

func (m Model) canCreateEvalTaskset() bool {
	project := strings.TrimSpace(m.currentPath)
	if project == "" {
		return false
	}
	return m.hasCompiledMixtape() ||
		projectFileExists(project, "LINER.md") ||
		countTopLevelRegularFiles(filepath.Join(project, "skills"), ".md") > 0
}

func (m *Model) applyEvalItems(items []evalFile) {
	m.evalItems = items
	m.evalTable.SetRows(evalRows(items))
}

func loadEvalFiles(project string) ([]evalFile, error) {
	root := filepath.Join(project, "working", "evals")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return []evalFile{}, nil
		}
		return nil, err
	}
	items := []evalFile{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relFromRoot, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relPath := filepath.Join("working", "evals", relFromRoot)
		items = append(items, evalFile{
			Name:    relFromRoot,
			RelPath: relPath,
			Path:    path,
			Area:    evalArea(relFromRoot),
			Updated: info.ModTime().Format("2006-01-02"),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i int, j int) bool {
		if items[i].Updated == items[j].Updated {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return items[i].Updated > items[j].Updated
	})
	return items, nil
}

func evalArea(rel string) string {
	first := strings.Split(filepath.ToSlash(rel), "/")[0]
	switch first {
	case "tasksets":
		return "taskset"
	case "runs":
		return "run"
	case "summaries":
		return "summary"
	case "automation":
		return "automation"
	case "comparisons":
		return "comparison"
	case "readiness":
		return "readiness"
	case "judges":
		return "judge"
	default:
		return "eval"
	}
}

func evalRows(items []evalFile) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.Name, item.Area, item.Updated})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"No impact-test artifacts found", "", ""})
	}
	return rows
}

func (m Model) selectedEval() (evalFile, bool) {
	index := m.evalTable.Cursor()
	if index < 0 || index >= len(m.evalItems) {
		return evalFile{}, false
	}
	return m.evalItems[index], true
}

func (m Model) handleEvalsKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case "t":
		return m.createEvalTaskset()
	case "r":
		return m.createEvalRunPacket()
	case "a":
		return m.createEvalAutomationPacket()
	case "v":
		return m.createEvalReadinessReport()
	case "c":
		return m.createEvalComparisonReport()
	case "j":
		return m.createEvalJudgePacket()
	case "enter":
		if item, ok := m.selectedEval(); ok {
			return m.openPreview(item.RelPath)
		}
	case "o":
		if item, ok := m.selectedEval(); ok {
			m.note = "Opened " + item.RelPath
			return m, openPath(item.Path)
		}
	}
	return m, nil
}

func (m Model) createEvalReadinessReport() (Model, tea.Cmd) {
	item, ok := m.selectedEval()
	if !ok {
		m.err = "Select a run packet, summary, comparison, runner packet, or judge packet before creating a readiness report."
		return m, nil
	}
	report, err := writeEvalReadinessReport(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyEvalItems(items)
	m = m.selectEvalArtifact(report)
	m.note = "Wrote " + report.RelPath + "."
	return m.openPreview(report.RelPath)
}

func (m Model) createEvalAutomationPacket() (Model, tea.Cmd) {
	item, ok := m.selectedEval()
	if !ok {
		m.err = "Select a run packet, summary, or comparison before creating a runner packet."
		return m, nil
	}
	packet, err := writeEvalAutomationPacket(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyEvalItems(items)
	m = m.selectEvalArtifact(packet)
	m.note = "Wrote " + packet.RelPath + "."
	return m.openPreview(packet.RelPath)
}

func (m Model) createEvalJudgePacket() (Model, tea.Cmd) {
	item, ok := m.selectedEval()
	if !ok {
		m.err = "Select a run packet, summary, or comparison before creating a judge packet."
		return m, nil
	}
	packet, err := writeEvalJudgePacket(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyEvalItems(items)
	m = m.selectEvalArtifact(packet)
	m.note = "Wrote " + packet.RelPath + "."
	return m.openPreview(packet.RelPath)
}

func (m Model) createEvalComparisonReport() (Model, tea.Cmd) {
	item, ok := m.selectedEval()
	if !ok {
		m.err = "Select a run packet or summary before creating a comparison."
		return m, nil
	}
	report, err := writeEvalComparisonReport(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyEvalItems(items)
	m = m.selectEvalArtifact(report)
	m.note = "Wrote " + report.RelPath + "."
	return m.openPreview(report.RelPath)
}

func (m Model) createEvalRunPacket() (Model, tea.Cmd) {
	item, ok := m.selectedEval()
	if !ok {
		m.err = "Select a taskset before creating a run packet."
		return m, nil
	}
	if item.Area != "taskset" {
		m.err = "Select a taskset before creating a run packet."
		return m, nil
	}
	runItem, err := writeEvalRunPacket(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyEvalItems(items)
	m = m.selectEvalArtifact(runItem)
	m.screen = screenEvals
	m.note = "Wrote " + filepath.Dir(runItem.RelPath) + " and a summary template."
	return m, nil
}

func (m Model) createEvalTaskset() (Model, tea.Cmd) {
	if !m.canCreateEvalTaskset() {
		m.err = "Compile MIXTAPE.md, generate LINER.md, or add skills before creating an impact test."
		return m, nil
	}
	t, err := tape.ReadProject(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	item, err := writeEvalTaskset(m.currentPath, t)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadEvalFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyEvalItems(items)
	m = m.selectEvalArtifact(item)
	m.screen = screenEvals
	m.note = "Wrote " + item.RelPath + "."
	return m, nil
}

func (m Model) selectEvalArtifact(item evalFile) Model {
	for index, current := range m.evalItems {
		if current.RelPath == item.RelPath {
			m.evalTable.SetCursor(index)
			break
		}
	}
	return m
}

func (m Model) viewEvals() string {
	width := styles.ClampWidth(m.width - 4)
	tableView := newVisibleDataTable(
		evalColumns(width),
		evalRows(m.evalItems),
		width,
		max(5, min(len(m.evalItems)+1, max(5, m.height-18))),
		true,
		m.evalTable.Cursor(),
	)
	count := styles.Section.Render("Impact Tests") + " " + styles.ReportBody.Render(intLabel(len(m.evalItems), "artifact"))
	parts := []string{
		styles.Title.Render("Impact Tests"),
		styles.Subtitle.Render("Compare baseline, corpus, operating layer, and skills runs without in-TUI model execution."),
		"",
		count,
		tableView.View(),
		"",
		styles.ReportSection.Render("Selected"),
		m.evalSelectedDetail(width),
		"",
	}
	if coverage, ok := m.evalCoverageTable(width); ok {
		parts = append(parts,
			styles.ReportSection.Render("Run coverage"),
			coverage.View(),
			"",
		)
	}
	parts = append(parts,
		styles.ReportSection.Render("Actions"),
		m.evalActionsTable(width).View(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) evalSelectedDetail(width int) string {
	item, ok := m.selectedEval()
	if !ok {
		return styles.SoftText.Render("No artifact selected.")
	}
	rows := []metadataTableRow{
		{Field: "Artifact", Value: item.Name},
		{Field: "Area", Value: item.Area},
		{Field: "Updated", Value: item.Updated},
		{Field: "Path", Value: filepath.ToSlash(item.RelPath)},
	}
	if body, err := os.ReadFile(item.Path); err == nil {
		for _, field := range []string{"Taskset", "Run packet", "Summary scores", "Source artifact"} {
			if value, ok := markdownMetadataValue(string(body), field); ok {
				rows = append(rows, metadataTableRow{Field: field, Value: value})
			}
		}
	}
	if runRel, ok := evalRunRelForItem(item); ok {
		rows = appendEvalRunMetadataRows(rows, m.currentPath, runRel)
	} else if runRel, ok, err := evalJudgeRunRelForItem(m.currentPath, item); err == nil && ok {
		rows = appendEvalRunMetadataRows(rows, m.currentPath, runRel)
	}
	return newMetadataTable(width, dedupeMetadataRows(rows)).View()
}

func appendEvalRunMetadataRows(rows []metadataTableRow, project string, runRel string) []metadataTableRow {
	rows = append(rows,
		metadataTableRow{Field: "Run packet", Value: filepath.ToSlash(runRel)},
		metadataTableRow{Field: "Summary scores", Value: filepath.ToSlash(evalSummaryRelForRun(runRel))},
	)
	if tasksetRel := evalTasksetRelForRun(project, runRel); tasksetRel != "" {
		rows = append(rows, metadataTableRow{Field: "Taskset", Value: filepath.ToSlash(tasksetRel)})
	}
	return rows
}

func (m Model) evalCoverageTable(width int) (table.Model, bool) {
	item, ok := m.selectedEval()
	if !ok {
		return table.Model{}, false
	}
	runRel, ok := evalRunRelForItem(item)
	if !ok {
		var err error
		runRel, ok, err = evalJudgeRunRelForItem(m.currentPath, item)
		if err != nil {
			return table.Model{}, false
		}
	}
	if !ok {
		return table.Model{}, false
	}
	comparisons := evalVariantComparisons(
		m.currentPath,
		runRel,
		evalSummaryScores(filepath.Join(m.currentPath, evalSummaryRelForRun(runRel))),
	)
	rows := make([]table.Row, 0, len(comparisons))
	for _, comparison := range comparisons {
		rows = append(rows, table.Row{
			comparison.Variant.Label,
			evalVariantCoverageStatus(comparison),
			evalScoreCoverageStatus(comparison),
			evalVariantContextStatus(m.currentPath, comparison.Variant),
		})
	}
	return newDataTable(evalCoverageColumns(width), rows, width, len(rows)+1, false), true
}

func evalCoverageColumns(width int) []table.Column {
	width = max(width, 72)
	return []table.Column{
		{Title: "Variant", Width: 18},
		{Title: "Outputs", Width: 12},
		{Title: "Scores", Width: 12},
		{Title: "Context", Width: max(24, width-48)},
	}
}

func (m Model) evalActionsTable(width int) table.Model {
	item, ok := m.selectedEval()
	if !ok {
		return newActionTable(width, []actionTableRow{
			{Key: "t", Action: "Create impact taskset", Writes: "taskset"},
		})
	}
	rows := []actionTableRow{
		{Key: "enter / o", Action: "Preview or open selected artifact", Writes: "read-only"},
	}
	switch item.Area {
	case "taskset":
		rows = append(rows,
			actionTableRow{Key: "r", Action: "Create variant run packet", Writes: "run packet + summary"},
			actionTableRow{Key: "t", Action: "Create another impact taskset", Writes: "taskset"},
		)
	case "run", "summary":
		rows = append(rows,
			actionTableRow{Key: "v", Action: "Create readiness report", Writes: "readiness report"},
			actionTableRow{Key: "a", Action: "Create runner packet", Writes: "external-run plan"},
			actionTableRow{Key: "c", Action: "Compare variant outputs", Writes: "comparison"},
			actionTableRow{Key: "j", Action: "Create judge packet", Writes: "judge packet"},
		)
	case "comparison":
		rows = append(rows,
			actionTableRow{Key: "v", Action: "Create readiness report", Writes: "readiness report"},
			actionTableRow{Key: "a", Action: "Create runner packet", Writes: "external-run plan"},
			actionTableRow{Key: "j", Action: "Create judge packet", Writes: "judge packet"},
		)
	case "automation", "readiness":
		rows = append(rows,
			actionTableRow{Key: "v", Action: "Create readiness report", Writes: "readiness report"},
			actionTableRow{Key: "c", Action: "Compare variant outputs", Writes: "comparison"},
			actionTableRow{Key: "j", Action: "Create judge packet", Writes: "judge packet"},
		)
	case "judge":
		rows = append(rows,
			actionTableRow{Key: "v", Action: "Create readiness report", Writes: "readiness report"},
			actionTableRow{Key: "c", Action: "Compare variant outputs", Writes: "comparison"},
		)
	default:
		rows = append(rows, actionTableRow{Key: "t", Action: "Create impact taskset", Writes: "taskset"})
	}
	return newActionTable(width, rows)
}
