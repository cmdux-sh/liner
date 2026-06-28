package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const (
	skillDraftDir              = "working/skills"
	skillDisabledStartMarker   = "<!-- liner:skill-disabled:start -->"
	skillDisabledEndMarker     = "<!-- liner:skill-disabled:end -->"
	skillDisableDraftSuffix    = "-disable-draft.md"
	skillEnableDraftSuffix     = "-enable-draft.md"
	skillDeprecateDraftSuffix  = "-deprecation-draft.md"
	skillGroundingDraftSuffix  = "-grounding-draft.md"
	skillGenerationDraftSuffix = "-draft.md"
)

type skillFile struct {
	Name    string
	RelPath string
	Path    string
	Status  string
	Updated string
}

func newSkillTable(width int, height int) table.Model {
	return newDataTable(skillColumns(width), []table.Row{}, width, height, true)
}

func skillColumns(width int) []table.Column {
	width = max(width, 64)
	nameWidth := max(18, width-42)
	return []table.Column{
		{Title: "Skill", Width: nameWidth},
		{Title: "Status", Width: 16},
		{Title: "Updated", Width: 12},
	}
}

func (m Model) startSkills() (Model, tea.Cmd) {
	items, err := loadSkillFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if len(items) == 0 && !m.canManageSkills() {
		m.err = "Compile MIXTAPE.md, generate LINER.md, or accept sources before creating skills."
		return m, nil
	}
	m.skillItems = items
	m.applySkillItems(items)
	m.screen = screenSkills
	if len(items) == 0 {
		m.note = "No skills yet."
	} else {
		m.note = "Loaded " + intLabel(len(items), "skill") + "."
	}
	return m, nil
}

func (m Model) canManageSkills() bool {
	project := strings.TrimSpace(m.currentPath)
	if project == "" {
		return false
	}
	capabilities := m.projectCapabilities()
	return capabilities.Skills > 0 ||
		projectFileExists(project, "MIXTAPE.md") ||
		projectFileExists(project, "LINER.md") ||
		hasAcceptedTapeSources(project)
}

func (m *Model) applySkillItems(items []skillFile) {
	m.skillItems = items
	m.skillTable.SetRows(skillRows(items))
}

func loadSkillFiles(project string) ([]skillFile, error) {
	dir := filepath.Join(project, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []skillFile{}, nil
		}
		return nil, err
	}
	items := make([]skillFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		status := "needs grounding"
		if data, err := os.ReadFile(path); err == nil {
			status = skillStatusFromBody(string(data))
		}
		items = append(items, skillFile{
			Name:    strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			RelPath: filepath.Join("skills", entry.Name()),
			Path:    path,
			Status:  status,
			Updated: info.ModTime().Format("2006-01-02"),
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func skillStatusFromBody(body string) string {
	if skillBodyDisabled(body) {
		return "disabled"
	}
	lower := strings.ToLower(body)
	if !hasSkillGrounding(lower) {
		return "needs grounding"
	}
	if !hasSkillBoundary(lower) {
		return "needs boundaries"
	}
	return "grounded"
}

func skillBodyDisabled(body string) bool {
	return strings.Contains(body, skillDisabledStartMarker) && strings.Contains(body, skillDisabledEndMarker)
}

func skillRows(items []skillFile) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.Name, item.Status, item.Updated})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"No skills found", "", ""})
	}
	return rows
}

func (m Model) selectedSkill() (skillFile, bool) {
	index := m.skillTable.Cursor()
	if index < 0 || index >= len(m.skillItems) {
		return skillFile{}, false
	}
	return m.skillItems[index], true
}

func (m Model) handleSkillsKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case "n":
		return m.startStarterSkillReview()
	case "x":
		return m.startSkillStateReview()
	case "r":
		return m.createSkillReadinessReport()
	case "g":
		return m.startSkillGroundingReview()
	case "d":
		return m.startSkillDeprecationReview()
	case "enter":
		if item, ok := m.selectedSkill(); ok {
			return m.openPreview(item.RelPath)
		}
	case "o":
		if item, ok := m.selectedSkill(); ok {
			m.note = "Opened " + item.RelPath
			return m, openPath(item.Path)
		}
	}
	return m, nil
}

func (m Model) createSkillReadinessReport() (Model, tea.Cmd) {
	item, ok := m.selectedSkill()
	if !ok {
		m.err = "Select a skill before creating a readiness report."
		return m, nil
	}
	audit, err := writeSkillReadinessReport(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.note = "Wrote " + audit.RelPath + "."
	return m.openPreview(audit.RelPath)
}

func (m Model) handleSkillReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenSkills
		return m, nil
	case "o":
		rel, err := m.currentSkillDraftRel()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.note = "Opened " + rel
		return m, openPath(filepath.Join(m.currentPath, rel))
	case "d":
		return m.discardSkillDraft()
	case "enter":
		return m.acceptSkillDraft()
	}
	return m, nil
}

func (m Model) startSkillGroundingReview() (Model, tea.Cmd) {
	item, ok := m.selectedSkill()
	if !ok {
		m.err = "Select a skill before drafting grounding repairs."
		return m, nil
	}
	return m.startSkillGroundingReviewForItem(item)
}

func (m Model) startSkillGroundingReviewForItem(item skillFile) (Model, tea.Cmd) {
	body, err := os.ReadFile(item.Path)
	if err != nil {
		m.err = "Could not read skill: " + err.Error()
		return m, nil
	}
	rel := filepath.Join(skillDraftDir, item.Name+skillGroundingDraftSuffix)
	path := filepath.Join(m.currentPath, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	draft := renderSkillGroundingDraft(item, string(body), m.currentTape)
	if err := os.WriteFile(path, []byte(draft), 0o644); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.setPreviewContent(rel, draft)
	m.screen = screenSkillReview
	m.note = "Review grounding and boundary repairs before updating " + item.RelPath + "."
	return m, nil
}

func (m Model) startStarterSkillReview() (Model, tea.Cmd) {
	if !m.canManageSkills() {
		m.err = "Compile MIXTAPE.md, generate LINER.md, or accept sources before creating skills."
		return m, nil
	}
	base := uniqueSkillBase(m.currentPath, starterSkillBase(m.currentTape.Title, m.currentPath))
	rel := filepath.Join(skillDraftDir, base+"-draft.md")
	path := filepath.Join(m.currentPath, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	body := renderStarterSkillDraft(m.currentTape, base)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.setPreviewContent(rel, body)
	m.screen = screenSkillReview
	m.note = "Review the starter skill before writing it to skills/."
	return m, nil
}

func (m Model) startSkillDeprecationReview() (Model, tea.Cmd) {
	item, ok := m.selectedSkill()
	if !ok {
		m.err = "Select a skill before deprecating it."
		return m, nil
	}
	rel := filepath.Join(skillDraftDir, item.Name+"-deprecation-draft.md")
	path := filepath.Join(m.currentPath, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	body := renderSkillDeprecationDraft(item)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.setPreviewContent(rel, body)
	m.screen = screenSkillReview
	m.note = "Review before moving " + item.RelPath + " out of active skills."
	return m, nil
}

func (m Model) startSkillStateReview() (Model, tea.Cmd) {
	item, ok := m.selectedSkill()
	if !ok {
		m.err = "Select a skill before changing its active state."
		return m, nil
	}
	suffix := skillDisableDraftSuffix
	body := renderSkillDisableDraft(item)
	note := "Review before temporarily disabling " + item.RelPath + "."
	if item.Status == "disabled" {
		suffix = skillEnableDraftSuffix
		body = renderSkillEnableDraft(item)
		note = "Review before re-enabling " + item.RelPath + "."
	}
	rel := filepath.Join(skillDraftDir, item.Name+suffix)
	path := filepath.Join(m.currentPath, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.setPreviewContent(rel, body)
	m.screen = screenSkillReview
	m.note = note
	return m, nil
}

func starterSkillBase(title string, project string) string {
	base := slug(fallbackText(title, filepath.Base(project)))
	if base == "" {
		base = "starter"
	}
	if !strings.HasSuffix(base, "-skill") {
		base += "-skill"
	}
	return base
}

func uniqueSkillBase(project string, base string) string {
	base = fallbackText(slug(base), "starter-skill")
	for index := 1; ; index++ {
		candidate := base
		if index > 1 {
			candidate = fmt.Sprintf("%s-%d", base, index)
		}
		finalPath := filepath.Join(project, "skills", candidate+".md")
		draftPath := filepath.Join(project, skillDraftDir, candidate+"-draft.md")
		if !fileExists(finalPath) && !fileExists(draftPath) {
			return candidate
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (m Model) currentSkillDraftRel() (string, error) {
	rel := filepath.Clean(strings.TrimSpace(m.previewRel))
	name := filepath.Base(rel)
	if rel == "" || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		filepath.Dir(rel) != skillDraftDir || !strings.HasSuffix(name, skillGenerationDraftSuffix) {
		return "", fmt.Errorf("No skill draft is selected.")
	}
	return rel, nil
}

func skillRelFromDraftRel(draftRel string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(draftRel), skillGenerationDraftSuffix)
	if base == "" {
		return "", fmt.Errorf("No skill draft is selected.")
	}
	return filepath.Join("skills", base+".md"), nil
}

func isSkillDeprecationDraftRel(rel string) bool {
	return strings.HasSuffix(filepath.Base(rel), skillDeprecateDraftSuffix)
}

func skillRelFromDeprecationDraftRel(draftRel string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(draftRel), skillDeprecateDraftSuffix)
	if base == "" {
		return "", fmt.Errorf("No skill deprecation draft is selected.")
	}
	return filepath.Join("skills", base+".md"), nil
}

func isSkillStateDraftRel(rel string) bool {
	name := filepath.Base(rel)
	return strings.HasSuffix(name, skillDisableDraftSuffix) || strings.HasSuffix(name, skillEnableDraftSuffix)
}

func isSkillGroundingDraftRel(rel string) bool {
	return strings.HasSuffix(filepath.Base(rel), skillGroundingDraftSuffix)
}

func skillRelFromGroundingDraftRel(draftRel string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(draftRel), skillGroundingDraftSuffix)
	if base == "" {
		return "", fmt.Errorf("No skill grounding draft is selected.")
	}
	return filepath.Join("skills", base+".md"), nil
}

func skillStateDraftAction(draftRel string) (string, string, error) {
	name := filepath.Base(draftRel)
	switch {
	case strings.HasSuffix(name, skillDisableDraftSuffix):
		base := strings.TrimSuffix(name, skillDisableDraftSuffix)
		if base == "" {
			return "", "", fmt.Errorf("No skill state draft is selected.")
		}
		return filepath.Join("skills", base+".md"), "disable", nil
	case strings.HasSuffix(name, skillEnableDraftSuffix):
		base := strings.TrimSuffix(name, skillEnableDraftSuffix)
		if base == "" {
			return "", "", fmt.Errorf("No skill state draft is selected.")
		}
		return filepath.Join("skills", base+".md"), "enable", nil
	default:
		return "", "", fmt.Errorf("No skill state draft is selected.")
	}
}

func uniqueDeprecatedSkillRel(project string, skillRel string) string {
	base := strings.TrimSuffix(filepath.Base(skillRel), filepath.Ext(skillRel))
	ext := filepath.Ext(skillRel)
	for index := 1; ; index++ {
		name := base + ext
		if index > 1 {
			name = fmt.Sprintf("%s-%d%s", base, index, ext)
		}
		rel := filepath.Join("skills", "deprecated", name)
		if !fileExists(filepath.Join(project, rel)) {
			return rel
		}
	}
}

func (m Model) acceptSkillDraft() (Model, tea.Cmd) {
	draftRel, err := m.currentSkillDraftRel()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if isSkillDeprecationDraftRel(draftRel) {
		return m.acceptSkillDeprecationDraft(draftRel)
	}
	if isSkillStateDraftRel(draftRel) {
		return m.acceptSkillStateDraft(draftRel)
	}
	if isSkillGroundingDraftRel(draftRel) {
		return m.acceptSkillGroundingDraft(draftRel)
	}
	skillRel, err := skillRelFromDraftRel(draftRel)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	draftPath := filepath.Join(m.currentPath, draftRel)
	skillPath := filepath.Join(m.currentPath, skillRel)
	draft, err := os.ReadFile(draftPath)
	if err != nil {
		m.err = "Could not read skill draft: " + err.Error()
		return m, nil
	}
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if _, err := os.Stat(skillPath); err == nil {
		m.err = "Skill already exists: " + skillRel
		return m, nil
	} else if err != nil && !os.IsNotExist(err) {
		m.err = err.Error()
		return m, nil
	}
	if err := os.WriteFile(skillPath, draft, 0o644); err != nil {
		m.err = "Could not write skill: " + err.Error()
		return m, nil
	}
	audit, err := writeSkillGenerationAudit(m.currentPath, draftRel, skillRel, len(draft))
	if err != nil {
		m.err = "Skill was written, but the audit could not be saved: " + err.Error()
		return m, nil
	}
	_ = os.Remove(draftPath)
	items, err := loadSkillFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applySkillItems(items)
	m = m.selectSkillRel(skillRel)
	m.note = "Accepted " + skillRel + ". Audit: " + audit.RelPath
	return m.openPreview(skillRel)
}

func (m Model) acceptSkillGroundingDraft(draftRel string) (Model, tea.Cmd) {
	skillRel, err := skillRelFromGroundingDraftRel(draftRel)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	draftPath := filepath.Join(m.currentPath, draftRel)
	skillPath := filepath.Join(m.currentPath, skillRel)
	original, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.err = "Skill no longer exists: " + skillRel
			return m, nil
		}
		m.err = err.Error()
		return m, nil
	}
	draft, err := os.ReadFile(draftPath)
	if err != nil {
		m.err = "Could not read skill grounding draft: " + err.Error()
		return m, nil
	}
	backupRel := filepath.Join(skillDraftDir, time.Now().Format("2006-01-02-150405")+"-"+strings.TrimSuffix(filepath.Base(skillRel), filepath.Ext(skillRel))+"-backup.md")
	if err := os.MkdirAll(filepath.Join(m.currentPath, filepath.Dir(backupRel)), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := os.WriteFile(filepath.Join(m.currentPath, backupRel), original, 0o644); err != nil {
		m.err = "Could not back up skill: " + err.Error()
		return m, nil
	}
	if err := os.WriteFile(skillPath, draft, 0o644); err != nil {
		m.err = "Could not update skill: " + err.Error()
		return m, nil
	}
	audit, err := writeSkillGroundingAudit(m.currentPath, draftRel, skillRel, backupRel, len(original), len(draft))
	if err != nil {
		m.err = "Skill was updated, but the audit could not be saved: " + err.Error()
		return m, nil
	}
	_ = os.Remove(draftPath)
	items, err := loadSkillFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applySkillItems(items)
	m = m.selectSkillRel(skillRel)
	m.note = "Updated " + skillRel + " grounding. Audit: " + audit.RelPath
	return m.openPreview(skillRel)
}

func (m Model) acceptSkillDeprecationDraft(draftRel string) (Model, tea.Cmd) {
	skillRel, err := skillRelFromDeprecationDraftRel(draftRel)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	draftPath := filepath.Join(m.currentPath, draftRel)
	skillPath := filepath.Join(m.currentPath, skillRel)
	deprecatedRel := uniqueDeprecatedSkillRel(m.currentPath, skillRel)
	deprecatedPath := filepath.Join(m.currentPath, deprecatedRel)
	body, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.err = "Skill no longer exists: " + skillRel
			return m, nil
		}
		m.err = err.Error()
		return m, nil
	}
	if err := os.MkdirAll(filepath.Dir(deprecatedPath), 0o755); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := os.Rename(skillPath, deprecatedPath); err != nil {
		m.err = "Could not deprecate skill: " + err.Error()
		return m, nil
	}
	audit, err := writeSkillDeprecationAudit(m.currentPath, draftRel, skillRel, deprecatedRel, len(body))
	if err != nil {
		m.err = "Skill was deprecated, but the audit could not be saved: " + err.Error()
		return m, nil
	}
	_ = os.Remove(draftPath)
	items, err := loadSkillFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applySkillItems(items)
	m.note = "Deprecated " + skillRel + ". Audit: " + audit.RelPath
	return m.openPreview(audit.RelPath)
}

func (m Model) acceptSkillStateDraft(draftRel string) (Model, tea.Cmd) {
	skillRel, action, err := skillStateDraftAction(draftRel)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	draftPath := filepath.Join(m.currentPath, draftRel)
	skillPath := filepath.Join(m.currentPath, skillRel)
	body, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.err = "Skill no longer exists: " + skillRel
			return m, nil
		}
		m.err = err.Error()
		return m, nil
	}
	updated := string(body)
	if action == "disable" {
		updated = applySkillDisabledBlock(updated, draftRel)
	} else {
		var removed bool
		updated, removed = removeSkillDisabledBlock(updated)
		if !removed {
			m.err = "Skill is not disabled: " + skillRel
			return m, nil
		}
	}
	if err := os.WriteFile(skillPath, []byte(updated), 0o644); err != nil {
		m.err = "Could not update skill: " + err.Error()
		return m, nil
	}
	audit, err := writeSkillStateAudit(m.currentPath, draftRel, skillRel, action, len(body), len(updated))
	if err != nil {
		m.err = "Skill state changed, but the audit could not be saved: " + err.Error()
		return m, nil
	}
	_ = os.Remove(draftPath)
	items, err := loadSkillFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applySkillItems(items)
	m = m.selectSkillRel(skillRel)
	m.note = "Updated " + skillRel + " state. Audit: " + audit.RelPath
	return m.openPreview(skillRel)
}

func (m Model) discardSkillDraft() (Model, tea.Cmd) {
	draftRel, err := m.currentSkillDraftRel()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := os.Remove(filepath.Join(m.currentPath, draftRel)); err != nil && !os.IsNotExist(err) {
		m.err = "Could not discard skill draft: " + err.Error()
		return m, nil
	}
	items, err := loadSkillFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applySkillItems(items)
	m.screen = screenSkills
	m.note = "Discarded " + draftRel + "."
	return m, nil
}

func (m Model) selectSkillRel(rel string) Model {
	for index, current := range m.skillItems {
		if current.RelPath == rel {
			m.skillTable.SetCursor(index)
			break
		}
	}
	return m
}

func (m Model) viewSkills() string {
	width := styles.ClampWidth(m.width - 4)
	tableView := newVisibleDataTable(
		skillColumns(width),
		skillRows(m.skillItems),
		width,
		max(5, min(len(m.skillItems)+1, max(5, m.height-18))),
		true,
		m.skillTable.Cursor(),
	)
	count := styles.Section.Render("Skills") + " " + styles.ReportBody.Render(intLabel(len(m.skillItems), "file"))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Skills"),
		styles.Subtitle.Render("Existing reusable methods grounded in this mixtape."),
		"",
		count,
		tableView.View(),
		"",
		styles.ReportSection.Render("Selected"),
		m.skillSelectedDetail(width),
		"",
		styles.ReportSection.Render("Actions"),
		m.skillActionsTable(width).View(),
	)
}

func (m Model) skillSelectedDetail(width int) string {
	item, ok := m.selectedSkill()
	if !ok {
		return styles.SoftText.Render("No skill selected.")
	}
	return newMetadataTable(width, []metadataTableRow{
		{Field: "Skill", Value: item.Name},
		{Field: "Status", Value: item.Status},
		{Field: "Updated", Value: item.Updated},
		{Field: "Path", Value: filepath.ToSlash(item.RelPath)},
	}).View()
}

func (m Model) skillActionsTable(width int) table.Model {
	item, ok := m.selectedSkill()
	if !ok {
		return newActionTable(width, []actionTableRow{
			{Key: "n", Action: "Draft starter skill", Writes: "review draft"},
		})
	}
	rows := []actionTableRow{
		{Key: "enter / o", Action: "Preview or open selected skill", Writes: "read-only"},
		{Key: "r", Action: "Write readiness report", Writes: "audit report"},
		{Key: "n", Action: "Draft starter skill", Writes: "review draft"},
	}
	if item.Status == "needs grounding" || item.Status == "needs boundaries" {
		rows = append(rows, actionTableRow{Key: "g", Action: "Repair grounding or boundaries", Writes: "review draft"})
	}
	if item.Status == "disabled" {
		rows = append(rows, actionTableRow{Key: "x", Action: "Re-enable selected skill", Writes: "review draft"})
		return newActionTable(width, rows)
	}
	rows = append(rows,
		actionTableRow{Key: "x", Action: "Temporarily disable selected skill", Writes: "review draft"},
		actionTableRow{Key: "d", Action: "Deprecate selected skill", Writes: "review draft"},
	)
	return newActionTable(width, rows)
}

func (m Model) viewSkillReview() string {
	width := styles.ClampWidth(m.width - 4)
	preview := m.preview
	preview.SetWidth(max(1, width))
	title := "Review Skill Draft"
	subtitle := "Starter skill for this mixtape. Review before writing to skills/."
	acceptAction := "Write skill"
	acceptWrites := "skills/ + audit"
	if isSkillDeprecationDraftRel(m.previewRel) {
		title = "Review Skill Deprecation"
		subtitle = "Move this skill out of active use. Review before changing skills/."
		acceptAction = "Archive skill"
		acceptWrites = "deprecated skill + audit"
	} else if isSkillStateDraftRel(m.previewRel) {
		title = "Review Skill State"
		subtitle = "Temporarily disable or re-enable this skill. Review before editing skills/."
		acceptAction = "Update skill state"
		acceptWrites = "skill marker + audit"
	} else if isSkillGroundingDraftRel(m.previewRel) {
		title = "Review Skill Grounding"
		subtitle = "Proposed Source Grounding and Boundaries repairs. Review before replacing the skill."
		acceptAction = "Update skill"
		acceptWrites = "skill + audit"
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render(title),
		styles.Subtitle.Render(subtitle),
		"",
		preview.View(),
		styles.ReportSection.Render("Actions"),
		newReviewActionTable(width, acceptAction, acceptWrites, "Skills").View(),
	)
}

func renderStarterSkillDraft(current tape.Tape, base string) string {
	title := fallbackText(current.Title, "Mixtape")
	jtbd := "No JTBD recorded. Keep the skill narrow and source-bound."
	if current.JTBD != nil && strings.TrimSpace(*current.JTBD) != "" {
		jtbd = strings.TrimSpace(*current.JTBD)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Starter Skill\n\n", title)
	b.WriteString("## Purpose\n\n")
	fmt.Fprintf(&b, "Use this skill when work on `%s` needs a repeatable method instead of a one-off answer.\n\n", title)
	b.WriteString("## When To Use\n\n")
	fmt.Fprintf(&b, "- The user asks for a concrete pass related to: %s\n", jtbd)
	b.WriteString("- The answer needs the mixtape's sources, boundaries, and operating layer.\n")
	b.WriteString("- The work should produce a reviewable artifact or decision, not just commentary.\n\n")
	b.WriteString("## Source Grounding\n\n")
	b.WriteString("- Start from `MIXTAPE.md` and `LINER.md` when available.\n")
	if len(current.Sources) == 0 {
		b.WriteString("- No saved sources are recorded in `tape.yaml` yet; treat this skill as a placeholder until sources are saved.\n")
	} else {
		for index, source := range current.Sources {
			if index >= 5 {
				fmt.Fprintf(&b, "- Plus %d more saved source(s) in `tape.yaml`.\n", len(current.Sources)-index)
				break
			}
			label := sourceNoteLabel(source, index)
			kind := optionalString(source.Kind)
			if kind == "" {
				kind = fallbackText(source.Type, "source")
			}
			section := optionalString(source.Section)
			if section == "" {
				section = "general corpus"
			}
			fmt.Fprintf(&b, "- `%s` — %s for %s.\n", label, kind, section)
		}
	}
	b.WriteString("\n## Method\n\n")
	b.WriteString("1. Restate the user's job in one sentence.\n")
	b.WriteString("2. Name the source or section that should carry the most weight.\n")
	b.WriteString("3. Apply the method in a short sequence of decisions or recommendations.\n")
	b.WriteString("4. Call out uncertainty, missing evidence, or where the source boundary stops.\n")
	b.WriteString("5. Leave the user with the next concrete artifact or action.\n\n")
	b.WriteString("## Boundaries\n\n")
	b.WriteString("- Do not invent coverage outside the accepted corpus.\n")
	b.WriteString("- Do not treat examples as universal rules unless `LINER.md` says to.\n")
	b.WriteString("- Ask for another source when the request depends on a domain the mixtape does not cover.\n\n")
	b.WriteString("## Maintenance\n\n")
	fmt.Fprintf(&b, "- Rename `%s` once the method has a more specific role.\n", base)
	b.WriteString("- Re-run skill-corpus alignment after editing this file.\n")
	b.WriteString("- Update Source Grounding when new saved sources materially change the method.\n")
	return b.String()
}

func renderSkillGroundingDraft(item skillFile, body string, current tape.Tape) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		trimmed = "# " + item.Name + "\n\n"
	}
	var b strings.Builder
	b.WriteString(trimmed)
	if !strings.HasSuffix(trimmed, "\n") {
		b.WriteString("\n")
	}
	lower := strings.ToLower(trimmed)
	if !hasSkillGrounding(lower) {
		b.WriteString("\n## Source Grounding\n\n")
		b.WriteString("- Start from `MIXTAPE.md` and `LINER.md` before applying this skill.\n")
		if len(current.Sources) == 0 {
			b.WriteString("- No saved sources are recorded in `tape.yaml`; keep this skill provisional until the corpus is saved.\n")
		} else {
			for index, source := range current.Sources {
				if index >= 5 {
					fmt.Fprintf(&b, "- Plus %d more saved source(s) in `tape.yaml`.\n", len(current.Sources)-index)
					break
				}
				label := sourceNoteLabel(source, index)
				kind := optionalString(source.Kind)
				if kind == "" {
					kind = fallbackText(source.Type, "source")
				}
				fmt.Fprintf(&b, "- `%s` — %s.\n", label, kind)
			}
		}
	}
	if !hasSkillBoundary(lower) {
		b.WriteString("\n## Boundaries\n\n")
		b.WriteString("- Do not apply this skill outside the accepted corpus without asking for another source.\n")
		b.WriteString("- Treat examples as examples, not universal rules, unless `LINER.md` says otherwise.\n")
		b.WriteString("- Name missing evidence before giving confident recommendations.\n")
	}
	if !hasSkillMaintenance(lower) {
		b.WriteString("\n## Maintenance\n\n")
		b.WriteString("- Re-run skill-corpus alignment after accepting this grounding repair.\n")
		b.WriteString("- Edit this draft before applying if a different source, boundary, or operating rule should carry more weight.\n")
	}
	return b.String()
}

func hasSkillMaintenance(body string) bool {
	return strings.Contains(body, "maintenance")
}

func renderSkillDeprecationDraft(item skillFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Deprecate %s\n\n", item.Name)
	b.WriteString("## Proposed Action\n\n")
	fmt.Fprintf(&b, "Move `%s` to `skills/deprecated/%s.md` so it no longer appears as an active skill.\n\n", item.RelPath, item.Name)
	b.WriteString("## Review Checklist\n\n")
	b.WriteString("- Confirm this skill is stale, superseded, or outside the current corpus boundary.\n")
	b.WriteString("- Confirm `LINER.md` or another active skill does not still route work to it.\n")
	b.WriteString("- Run skill-corpus alignment after deprecating if this skill affected active behavior.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- This draft does not move or edit the skill until accepted.\n")
	b.WriteString("- Accepting archives the skill file and writes a deprecation audit.\n")
	return b.String()
}

func renderSkillDisableDraft(item skillFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Disable %s\n\n", item.Name)
	b.WriteString("## Proposed Action\n\n")
	fmt.Fprintf(&b, "Insert a managed disabled marker into `%s` so it remains on disk but no longer appears as an active method.\n\n", item.RelPath)
	b.WriteString("## Review Checklist\n\n")
	b.WriteString("- Confirm this is temporary; use deprecate when the skill should leave active skills permanently.\n")
	b.WriteString("- Confirm `LINER.md` or another active skill can route around this method while disabled.\n")
	b.WriteString("- Re-enable only after the source grounding, scope, or method issue is resolved.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- This draft does not edit the skill until accepted.\n")
	b.WriteString("- Accepting inserts only the managed disabled marker and writes a state audit.\n")
	return b.String()
}

func renderSkillEnableDraft(item skillFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Enable %s\n\n", item.Name)
	b.WriteString("## Proposed Action\n\n")
	fmt.Fprintf(&b, "Remove the managed disabled marker from `%s` so it returns to active use.\n\n", item.RelPath)
	b.WriteString("## Review Checklist\n\n")
	b.WriteString("- Confirm the reason for disabling this skill has been resolved or documented.\n")
	b.WriteString("- Confirm the skill still has source grounding and boundaries.\n")
	b.WriteString("- Run skill-corpus alignment after re-enabling if the corpus changed.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- This draft does not edit the skill until accepted.\n")
	b.WriteString("- Accepting removes only the managed disabled marker and writes a state audit.\n")
	return b.String()
}

func applySkillDisabledBlock(body string, draftRel string) string {
	if skillBodyDisabled(body) {
		return body
	}
	block := skillDisabledStartMarker + "\n" +
		"> Status: disabled\n" +
		"> Disabled by reviewed draft: `" + draftRel + "`\n" +
		"> Remove through the Skills screen enable flow.\n" +
		skillDisabledEndMarker
	body = strings.TrimLeft(body, "\n")
	if body == "" {
		return block + "\n"
	}
	return block + "\n\n" + body
}

func removeSkillDisabledBlock(body string) (string, bool) {
	start := strings.Index(body, skillDisabledStartMarker)
	end := strings.Index(body, skillDisabledEndMarker)
	if start < 0 || end < start {
		return body, false
	}
	end += len(skillDisabledEndMarker)
	prefix := strings.TrimRight(body[:start], "\n")
	suffix := strings.TrimLeft(body[end:], "\n")
	switch {
	case prefix == "":
		return suffix, true
	case suffix == "":
		return prefix + "\n", true
	default:
		return prefix + "\n\n" + suffix, true
	}
}

func writeSkillGenerationAudit(project string, draftRel string, skillRel string, size int) (auditFile, error) {
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-skill-generation"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	var b strings.Builder
	fmt.Fprintf(&b, "# Skill Generation Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This audit records a reviewed starter skill created from the Go Skills manager.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Reviewed draft: `%s`\n", draftRel)
	fmt.Fprintf(&b, "- Written skill: `%s`\n", skillRel)
	fmt.Fprintf(&b, "- Draft bytes: %d\n\n", size)
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- The skill was written only after user acceptance from the review screen.\n")
	b.WriteString("- No source files, `MIXTAPE.md`, or `LINER.md` were changed by this action.\n")
	b.WriteString("- Run skill-corpus alignment after editing the generated starter skill.\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "skill alignment",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeSkillStateAudit(project string, draftRel string, skillRel string, action string, beforeSize int, afterSize int) (auditFile, error) {
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-skill-state"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	var b strings.Builder
	fmt.Fprintf(&b, "# Skill State Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This audit records a reviewed skill enable/disable change from the Go Skills manager.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Reviewed draft: `%s`\n", draftRel)
	fmt.Fprintf(&b, "- Skill: `%s`\n", skillRel)
	fmt.Fprintf(&b, "- Action: `%s`\n", action)
	fmt.Fprintf(&b, "- Bytes before: %d\n", beforeSize)
	fmt.Fprintf(&b, "- Bytes after: %d\n\n", afterSize)
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- The skill state changed only after user acceptance from the review screen.\n")
	if action == "disable" {
		b.WriteString("- The skill file remained in place; only a managed disabled marker was inserted.\n")
	} else {
		b.WriteString("- Only the managed disabled marker was removed from the skill file.\n")
	}
	b.WriteString("- No source files, `MIXTAPE.md`, `LINER.md`, or saved sources were changed by this action.\n")
	b.WriteString("- Re-run skill-corpus alignment after changing active skill state.\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "skill alignment",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeSkillGroundingAudit(project string, draftRel string, skillRel string, backupRel string, beforeSize int, afterSize int) (auditFile, error) {
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-skill-grounding"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	var b strings.Builder
	fmt.Fprintf(&b, "# Skill Grounding Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This audit records a reviewed Source Grounding and Boundaries repair from the Go Skills manager.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Reviewed draft: `%s`\n", draftRel)
	fmt.Fprintf(&b, "- Skill: `%s`\n", skillRel)
	fmt.Fprintf(&b, "- Previous skill backup: `%s`\n", backupRel)
	fmt.Fprintf(&b, "- Bytes before: %d\n", beforeSize)
	fmt.Fprintf(&b, "- Bytes after: %d\n\n", afterSize)
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- The skill was replaced only after user acceptance from the review screen.\n")
	b.WriteString("- The previous skill body was backed up before writing the reviewed draft.\n")
	b.WriteString("- No source files, `MIXTAPE.md`, `LINER.md`, or saved sources were changed by this action.\n")
	b.WriteString("- Re-run skill-corpus alignment after accepting grounding repairs.\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "skill alignment",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeSkillReadinessReport(project string, item skillFile) (auditFile, error) {
	body, err := os.ReadFile(item.Path)
	if err != nil {
		return auditFile{}, err
	}
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	base := fallbackText(slug(item.Name), "skill")
	name := now.Format("2006-01-02-150405") + "-" + base + "-skill-readiness"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderSkillReadinessReport(now, item, string(body))), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "skill alignment",
		Updated: now.Format("2006-01-02"),
	}, nil
}

type skillReadinessRow struct {
	Status         string
	Evidence       string
	Recommendation string
}

func renderSkillReadinessReport(now time.Time, item skillFile, body string) string {
	lower := strings.ToLower(body)
	rows := []skillReadinessRow{}
	if hasSkillGrounding(lower) {
		rows = append(rows, skillReadinessRow{
			Status:         "grounded",
			Evidence:       "Source Grounding section found.",
			Recommendation: "Keep source references current as the mixtape changes.",
		})
	} else {
		rows = append(rows, skillReadinessRow{
			Status:         "needs grounding",
			Evidence:       "Source Grounding section missing.",
			Recommendation: "Draft grounding repair from Skills or Audit Center before relying on the skill.",
		})
	}
	if hasSkillBoundary(lower) {
		rows = append(rows, skillReadinessRow{
			Status:         "bounded",
			Evidence:       "Boundaries section or limitation language found.",
			Recommendation: "Keep boundaries aligned with LINER.md operating rules.",
		})
	} else {
		rows = append(rows, skillReadinessRow{
			Status:         "needs boundaries",
			Evidence:       "Boundaries or limitation language missing.",
			Recommendation: "Add explicit boundaries before relying on the skill.",
		})
	}
	if hasSkillMaintenance(lower) {
		rows = append(rows, skillReadinessRow{
			Status:         "maintenance present",
			Evidence:       "Maintenance guidance found.",
			Recommendation: "Use this guidance after corpus or operating-rule changes.",
		})
	} else {
		rows = append(rows, skillReadinessRow{
			Status:         "maintenance missing",
			Evidence:       "Maintenance section missing.",
			Recommendation: "Add update and re-audit instructions.",
		})
	}
	if skillBodyDisabled(body) {
		rows = append(rows, skillReadinessRow{
			Status:         "disabled",
			Evidence:       "Managed disabled marker found.",
			Recommendation: "Keep disabled until a review confirms the skill should return to active use.",
		})
	} else {
		rows = append(rows, skillReadinessRow{
			Status:         "active",
			Evidence:       "No disabled marker found.",
			Recommendation: "Keep active if the method is still in scope.",
		})
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Skill Readiness Alignment Report\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local report checks one selected skill from the Go Skills manager. It does not edit skill files, sources, `MIXTAPE.md`, `LINER.md`, or `tape.yaml`.\n\n")
	b.WriteString("## Selected Skill\n\n")
	fmt.Fprintf(&b, "- Skill: `%s`\n", filepath.ToSlash(item.RelPath))
	fmt.Fprintf(&b, "- Status: `%s`\n", item.Status)
	fmt.Fprintf(&b, "- Bytes: %d\n\n", len(body))
	b.WriteString("## Checks\n\n")
	b.WriteString("| Skill | Status | Evidence | Recommendation |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			escapeMarkdownTable(filepath.ToSlash(item.RelPath)),
			escapeMarkdownTable(row.Status),
			escapeMarkdownTable(row.Evidence),
			escapeMarkdownTable(row.Recommendation),
		)
	}
	b.WriteString("\n## Decision Log\n\n")
	b.WriteString("- No skill files, source files, `MIXTAPE.md`, `LINER.md`, or `tape.yaml` were changed by this report.\n")
	b.WriteString("- Use reviewed repair, disable, enable, or deprecation actions from Skills after reading the findings.\n")
	b.WriteString("- Use Audit Center grounding repair for `needs grounding` or `needs boundaries` rows when the skill should remain active.\n")
	return b.String()
}

func writeSkillDeprecationAudit(project string, draftRel string, skillRel string, deprecatedRel string, size int) (auditFile, error) {
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-skill-deprecation"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	var b strings.Builder
	fmt.Fprintf(&b, "# Skill Deprecation Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This audit records a reviewed skill deprecation from the Go Skills manager.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Reviewed draft: `%s`\n", draftRel)
	fmt.Fprintf(&b, "- Active skill: `%s`\n", skillRel)
	fmt.Fprintf(&b, "- Archived skill: `%s`\n", deprecatedRel)
	fmt.Fprintf(&b, "- Skill bytes: %d\n\n", size)
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- The active skill was moved only after user acceptance from the review screen.\n")
	b.WriteString("- No source files, `MIXTAPE.md`, `LINER.md`, or saved sources were changed by this action.\n")
	b.WriteString("- Re-run skill-corpus alignment if this skill was referenced by active operating guidance.\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "skill alignment",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func intLabel(count int, singular string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + singular + "s"
}
