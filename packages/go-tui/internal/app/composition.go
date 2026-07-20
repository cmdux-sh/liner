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
)

type compositionFile struct {
	Name    string
	RelPath string
	Path    string
	Kind    string
	Status  string
	Updated string
}

type compositionChild struct {
	Name   string `yaml:"name"`
	Ref    string `yaml:"ref"`
	Mode   string `yaml:"mode"`
	Route  string `yaml:"route"`
	Status string `yaml:"status"`
	Kind   string `yaml:"kind"`
}

type compositionLineage struct {
	Parent       string                    `yaml:"parent"`
	Mode         string                    `yaml:"mode"`
	Updated      string                    `yaml:"updated"`
	Children     []compositionChild        `yaml:"children"`
	History      []compositionLineageEvent `yaml:"history"`
	PreviousCopy string                    `yaml:"previous_copy,omitempty"`
}

type compositionLineageEvent struct {
	Date  string `yaml:"date"`
	Event string `yaml:"event"`
	Audit string `yaml:"audit"`
	Draft string `yaml:"draft"`
}

type compositionNestResult struct {
	LineageRel      string
	DraftRel        string
	AuditRel        string
	PreviousCopyRel string
	Children        []compositionChild
}

type compositionRouteAuditInput struct {
	Child       string
	RelPath     string
	Route       string
	Status      string
	Explicit    bool
	ReviewLines []string
}

type compositionRouteFinding struct {
	Severity       string
	Children       string
	Evidence       string
	Reason         string
	Recommendation string
}

type compositionCopyCandidate struct {
	Artifact       string
	Status         string
	Evidence       string
	Recommendation string
}

type compositionCopyPacketResult struct {
	PacketRel string
	AuditRel  string
	Input     compositionRouteAuditInput
}

type compositionCopyApplyResult struct {
	SnapshotRel  string
	AuditRel     string
	ChildProject string
	Input        compositionRouteAuditInput
	Files        []compositionCopiedFile
	Skipped      []string
}

type compositionCopiedFile struct {
	Artifact string
	DestRel  string
	Bytes    int64
}

const (
	compositionDraftRelPath = "working/LINER-composition-draft.md"
	compositionStartMarker  = "<!-- liner:composition-routing:start -->"
	compositionEndMarker    = "<!-- liner:composition-routing:end -->"
)

func newCompositionTable(width int, height int) table.Model {
	return newDataTable(compositionColumns(width), []table.Row{}, width, height, true)
}

func compositionColumns(width int) []table.Column {
	width = max(width, 72)
	nameWidth := max(22, width-50)
	return []table.Column{
		{Title: "Artifact", Width: nameWidth},
		{Title: "Kind", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "Updated", Width: 12},
	}
}

func (m Model) startComposition() (Model, tea.Cmd) {
	items, err := loadCompositionFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if len(items) == 0 {
		m.err = "No child references or lineage.yaml found."
		return m, nil
	}
	m.compositionItems = items
	m.applyCompositionItems(items)
	m.screen = screenComposition
	m.note = "Loaded " + intLabel(len(items), "composition artifact") + "."
	return m, nil
}

func (m *Model) applyCompositionItems(items []compositionFile) {
	m.compositionItems = items
	m.compositionTable.SetRows(compositionRows(items))
}

func loadCompositionFiles(project string) ([]compositionFile, error) {
	items := []compositionFile{}
	childrenRoot := filepath.Join(project, "children")
	if _, err := os.Stat(childrenRoot); err == nil {
		err := filepath.WalkDir(childrenRoot, func(path string, entry os.DirEntry, err error) error {
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
			relFromRoot, err := filepath.Rel(childrenRoot, path)
			if err != nil {
				return err
			}
			relPath := filepath.Join("children", relFromRoot)
			items = append(items, compositionFile{
				Name:    relFromRoot,
				RelPath: relPath,
				Path:    path,
				Kind:    compositionKind(relFromRoot),
				Status:  compositionStatus(path),
				Updated: info.ModTime().Format("2006-01-02"),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	lineagePath := filepath.Join(project, "lineage.yaml")
	if info, err := os.Stat(lineagePath); err == nil && !info.IsDir() {
		items = append(items, compositionFile{
			Name:    "lineage.yaml",
			RelPath: "lineage.yaml",
			Path:    lineagePath,
			Kind:    "lineage",
			Status:  "history",
			Updated: info.ModTime().Format("2006-01-02"),
		})
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Slice(items, func(i int, j int) bool {
		if items[i].Kind == "lineage" && items[j].Kind != "lineage" {
			return false
		}
		if items[i].Kind != "lineage" && items[j].Kind == "lineage" {
			return true
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func compositionKind(rel string) string {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".md":
		return "child notes"
	case ".yaml", ".yml", ".json":
		return "child ref"
	default:
		return "child"
	}
}

func compositionStatus(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unreadable"
	}
	body := strings.ToLower(strings.TrimSpace(string(data)))
	switch {
	case body == "":
		return "empty"
	case strings.Contains(body, "disabled") || strings.Contains(body, "deprecated"):
		return "disabled"
	case strings.Contains(body, "warning") || strings.Contains(body, "conflict") || strings.Contains(body, "contradiction"):
		return "review"
	case strings.Contains(body, "path:") || strings.Contains(body, "mixtape") || strings.Contains(body, "liner"):
		return "ready"
	default:
		return "reference"
	}
}

func compositionRows(items []compositionFile) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.Name, item.Kind, item.Status, item.Updated})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"No composition artifacts found", "", "", ""})
	}
	return rows
}

func (m Model) selectedComposition() (compositionFile, bool) {
	index := m.compositionTable.Cursor()
	if index < 0 || index >= len(m.compositionItems) {
		return compositionFile{}, false
	}
	return m.compositionItems[index], true
}

func (m Model) handleCompositionKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case "n":
		return m.createCompositionNestingDraft()
	case "m":
		return m.createCompositionMergeDraft()
	case "b":
		return m.createCompositionLinerBlendDraft()
	case "s":
		return m.runCompositionSkillConflictReview()
	case "c":
		return m.createCompositionCopyPacket()
	case "a":
		return m.applyCompositionCopy()
	case "x":
		return m.mergeCompositionChild()
	case "r":
		return m.runCompositionRouteAudit()
	case "d":
		return m.createCompositionRouteResolutionDraft()
	case "p":
		return m.runCompositionPromotionAudit()
	case "enter":
		if item, ok := m.selectedComposition(); ok {
			return m.openPreview(item.RelPath)
		}
	case "o":
		if item, ok := m.selectedComposition(); ok {
			m.note = "Opened " + item.RelPath
			return m, openPath(item.Path)
		}
	}
	return m, nil
}

func (m Model) applyCompositionCopy() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before applying a copy snapshot."
		return m, nil
	}
	result, err := writeCompositionCopyApply(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.note = "Copied child snapshot to " + result.SnapshotRel + " and wrote " + result.AuditRel + "."
	return m.openPreview(result.AuditRel)
}

func (m Model) createCompositionCopyPacket() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before creating a copy packet."
		return m, nil
	}
	result, err := writeCompositionCopyPacket(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.note = "Wrote " + result.PacketRel + " and " + result.AuditRel + "."
	return m.openPreview(result.PacketRel)
}

func (m Model) runCompositionPromotionAudit() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before checking promotion readiness."
		return m, nil
	}
	report, err := writeCompositionPromotionAudit(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.note = "Wrote " + report.RelPath + "."
	return m.openPreview(report.RelPath)
}

func (m Model) runCompositionRouteAudit() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot run a composition audit without a project path."
		return m, nil
	}
	items := m.compositionItems
	if len(items) == 0 {
		var err error
		items, err = loadCompositionFiles(m.currentPath)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
	}
	report, err := writeCompositionRouteAudit(m.currentPath, items)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.note = "Wrote " + report.RelPath + "."
	return m.openPreview(report.RelPath)
}

func (m Model) createCompositionNestingDraft() (Model, tea.Cmd) {
	children := compositionChildren(m.compositionItems)
	if len(children) == 0 {
		m.err = "Add at least one child reference before creating a nesting plan."
		return m, nil
	}
	result, err := writeCompositionNesting(m.currentPath, fallbackText(m.currentTape.Title, filepath.Base(m.currentPath)), children)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadCompositionFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyCompositionItems(items)
	return m.startCompositionDraftReview("Wrote " + result.LineageRel + ", " + result.DraftRel + ", and " + result.AuditRel + ". Review before applying.")
}

func (m Model) createCompositionMergeDraft() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before creating a merge draft."
		return m, nil
	}
	input, err := compositionPromotionInput(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	auditRel, err := writeCompositionMergeDraft(m.currentPath, fallbackText(m.currentTape.Title, filepath.Base(m.currentPath)), input)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	return m.startCompositionDraftReview("Wrote " + compositionDraftRelPath + " and " + auditRel + ". Review before applying.")
}

func (m Model) createCompositionLinerBlendDraft() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before blending child LINER.md."
		return m, nil
	}
	auditRel, err := writeCompositionLinerBlendDraft(m.currentPath, fallbackText(m.currentTape.Title, filepath.Base(m.currentPath)), item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	return m.startCompositionDraftReview("Wrote " + compositionDraftRelPath + " and " + auditRel + ". Review before applying.")
}

func (m Model) startCompositionDraftReview(note string) (Model, tea.Cmd) {
	draft, err := os.ReadFile(filepath.Join(m.currentPath, compositionDraftRelPath))
	if err != nil {
		m.err = "Could not read composition draft: " + err.Error()
		return m, nil
	}
	m.setPreviewContent(compositionDraftRelPath, string(draft))
	m.screen = screenCompositionReview
	m.note = note
	return m, nil
}

func (m Model) handleCompositionReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenComposition
		return m, nil
	case "o":
		m.note = "Opened composition draft."
		return m, openPath(filepath.Join(m.currentPath, compositionDraftRelPath))
	case "d":
		return m.discardCompositionDraft()
	case "enter":
		return m.acceptCompositionDraft()
	}
	return m, nil
}

func (m Model) acceptCompositionDraft() (Model, tea.Cmd) {
	m.err = legacyCoreWriterError("apply a composition draft").Error()
	return m, nil
}

func (m Model) discardCompositionDraft() (Model, tea.Cmd) {
	if err := os.Remove(filepath.Join(m.currentPath, compositionDraftRelPath)); err != nil && !os.IsNotExist(err) {
		m.err = "Could not discard composition draft: " + err.Error()
		return m, nil
	}
	m.screen = screenComposition
	m.note = "Discarded composition draft."
	return m, nil
}

func (m Model) viewCompositionReview() string {
	width := styles.ClampWidth(m.width - 4)
	preview := m.preview
	preview.SetWidth(max(1, width))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Review Composition Draft"),
		styles.Subtitle.Render("Composition routing section for LINER.md. Review before applying."),
		"",
		preview.View(),
		styles.ReportSection.Render("Actions"),
		newReviewActionTable(width, "Apply to LINER.md", "LINER.md + audit", "Composition").View(),
	)
}

func applyCompositionSection(existing string, draft string) string {
	section := compositionStartMarker + "\n" + strings.TrimSpace(draft) + "\n" + compositionEndMarker
	existing = strings.TrimSpace(existing)
	start := strings.Index(existing, compositionStartMarker)
	end := strings.Index(existing, compositionEndMarker)
	if start >= 0 && end >= start {
		end += len(compositionEndMarker)
		return strings.TrimSpace(existing[:start]) + "\n\n" + section + "\n\n" + strings.TrimSpace(existing[end:]) + "\n"
	}
	if existing == "" {
		return section + "\n"
	}
	return existing + "\n\n" + section + "\n"
}

func (m Model) viewComposition() string {
	width := styles.ClampWidth(m.width - 4)
	tableView := newVisibleDataTable(
		compositionColumns(width),
		compositionRows(m.compositionItems),
		width,
		max(5, min(len(m.compositionItems)+1, max(5, m.height-18))),
		true,
		m.compositionTable.Cursor(),
	)
	count := styles.Section.Render("Composition") + " " + styles.ReportBody.Render(intLabel(len(m.compositionItems), "artifact"))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Composition"),
		styles.Subtitle.Render("Child mixtape references and lineage for nested operating layers."),
		"",
		count,
		tableView.View(),
		"",
		styles.ReportSection.Render("Selected"),
		m.compositionSelectedDetail(width),
		"",
		styles.ReportSection.Render("Actions"),
		m.compositionActionsTable(width).View(),
	)
}

func (m Model) compositionSelectedDetail(width int) string {
	item, ok := m.selectedComposition()
	if !ok {
		return styles.SoftText.Render("No composition artifact selected.")
	}
	rows := []metadataTableRow{
		{Field: "Artifact", Value: item.Name},
		{Field: "Kind", Value: item.Kind},
		{Field: "Status", Value: item.Status},
		{Field: "Updated", Value: item.Updated},
		{Field: "Path", Value: filepath.ToSlash(item.RelPath)},
	}
	if item.Kind == "lineage" {
		for _, field := range []string{"parent", "mode", "updated"} {
			if value, ok := explicitCompositionField(item.Path, []string{field}); ok {
				rows = append(rows, metadataTableRow{Field: titleASCII(field), Value: value})
			}
		}
	} else {
		if route := compositionRoute(item); route != "" {
			rows = append(rows, metadataTableRow{Field: "Route", Value: route})
		}
		if reference, ok := explicitCompositionField(item.Path, []string{"path", "project", "ref"}); ok {
			rows = append(rows, metadataTableRow{Field: "Reference", Value: reference})
		}
	}
	return newMetadataTable(width, dedupeMetadataRows(rows)).View()
}

func (m Model) compositionActionsTable(width int) table.Model {
	item, ok := m.selectedComposition()
	if !ok {
		return newActionTable(width, []actionTableRow{
			{Key: "n", Action: "Refresh nested child routing", Writes: "lineage + draft"},
			{Key: "r", Action: "Audit child route overlap", Writes: "audit report"},
		})
	}
	if item.Kind == "lineage" {
		return newActionTable(width, []actionTableRow{
			{Key: "enter / o", Action: "Preview or open lineage", Writes: "read-only"},
			{Key: "n", Action: "Refresh nested child routing", Writes: "lineage + draft"},
			{Key: "r", Action: "Audit child route overlap", Writes: "audit report"},
			{Key: "d", Action: "Draft route conflict resolution", Writes: "review draft"},
		})
	}
	return newActionTable(width, []actionTableRow{
		{Key: "enter / o", Action: "Preview or open selected child", Writes: "read-only"},
		{Key: "p", Action: "Check promotion readiness", Writes: "audit report"},
		{Key: "m / b", Action: "Draft merge route or LINER blend", Writes: "review draft"},
		{Key: "s", Action: "Review parent skill conflicts", Writes: "audit report"},
		{Key: "c / a", Action: "Create copy packet or review snapshot", Writes: "packet / snapshot"},
		{Key: "x", Action: "Advanced: production-merge child", Writes: "tape + skills + audit"},
	})
}

func compositionChildren(items []compositionFile) []compositionChild {
	children := make([]compositionChild, 0, len(items))
	for _, item := range items {
		if item.Kind == "lineage" {
			continue
		}
		children = append(children, compositionChild{
			Name:   compositionChildName(item),
			Ref:    item.RelPath,
			Mode:   "nested",
			Route:  compositionRoute(item),
			Status: item.Status,
			Kind:   item.Kind,
		})
	}
	sort.Slice(children, func(i int, j int) bool {
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})
	return children
}

func compositionChildName(item compositionFile) string {
	name := strings.TrimSuffix(filepath.Base(item.Name), filepath.Ext(item.Name))
	name = strings.TrimSpace(strings.ReplaceAll(name, "_", "-"))
	if name == "" {
		return "child"
	}
	return name
}

func compositionRoute(item compositionFile) string {
	if route, ok := explicitCompositionRoute(item.Path); ok {
		return route
	}
	return strings.ReplaceAll(compositionChildName(item), "-", ", ")
}

func explicitCompositionRoute(path string) (string, bool) {
	return explicitCompositionField(path, []string{"route", "scope", "focus", "tags"})
}

func explicitCompositionField(path string, keys []string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		clean := strings.TrimSpace(line)
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "-"))
		lower := strings.ToLower(clean)
		for _, key := range keys {
			prefix := strings.ToLower(key) + ":"
			if strings.HasPrefix(lower, prefix) {
				value := strings.TrimSpace(clean[len(prefix):])
				value = strings.Trim(value, `"'`)
				if value != "" {
					return value, true
				}
			}
		}
	}
	return "", false
}
