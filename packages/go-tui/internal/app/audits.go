package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

type auditFile struct {
	Name    string
	RelPath string
	Path    string
	Type    string
	Updated string
}

type contradictionFinding struct {
	Severity string
	RelPath  string
	Line     int
	Evidence string
	Reason   string
}

type auditFindingPreview struct {
	Subject        string
	Status         string
	Evidence       string
	Recommendation string
}

const (
	contradictionCleanupStartMarker = "<!-- liner:contradiction-decisions:start -->"
	contradictionCleanupEndMarker   = "<!-- liner:contradiction-decisions:end -->"
)

func newAuditTable(width int, height int) table.Model {
	return newDataTable(auditColumns(width), []table.Row{}, width, height, true)
}

func auditColumns(width int) []table.Column {
	width = max(width, 64)
	nameWidth := max(18, width-42)
	return []table.Column{
		{Title: "Audit", Width: nameWidth},
		{Title: "Type", Width: 16},
		{Title: "Updated", Width: 12},
	}
}

func (m Model) startAudits() (Model, tea.Cmd) {
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if len(items) == 0 && !m.canRunAudits() {
		m.err = "No audits found in working/audits/."
		return m, nil
	}
	m.auditItems = items
	m.applyAuditItems(items)
	m.screen = screenAudits
	if len(items) == 0 {
		m.note = "No audit reports yet."
	} else {
		m.note = "Loaded " + intLabel(len(items), "audit report") + "."
	}
	return m, nil
}

func (m Model) canRunAudits() bool {
	project := strings.TrimSpace(m.currentPath)
	if project == "" {
		return false
	}
	for _, rel := range contradictionAuditBaseInputs() {
		if projectFileExists(project, rel) {
			return true
		}
	}
	if countTopLevelRegularFiles(filepath.Join(project, "skills"), ".md") > 0 {
		return true
	}
	return hasAcceptedTapeSources(project)
}

func (m *Model) applyAuditItems(items []auditFile) {
	m.auditItems = items
	m.auditTable.SetRows(auditRows(items))
}

func loadAuditFiles(project string) ([]auditFile, error) {
	dir := filepath.Join(project, "working", "audits")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []auditFile{}, nil
		}
		return nil, err
	}
	items := make([]auditFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		items = append(items, auditFile{
			Name:    name,
			RelPath: filepath.Join("working", "audits", entry.Name()),
			Path:    path,
			Type:    auditType(name, path),
			Updated: info.ModTime().Format("2006-01-02"),
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		if items[i].Updated == items[j].Updated {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return items[i].Updated > items[j].Updated
	})
	return items, nil
}

func auditType(name string, path string) string {
	value := strings.ToLower(name)
	data, err := os.ReadFile(path)
	if err == nil {
		value += "\n" + strings.ToLower(string(data))
	}
	switch {
	case strings.Contains(value, "agent cleanup packet") || strings.Contains(value, "agent-cleanup-packet"):
		return "cleanup packet"
	case containsAny(value, []string{
		"composition-nesting",
		"composition nesting",
		"composition-route-audit",
		"composition route audit",
		"composition-route-resolution",
		"composition route resolution",
		"composition-promotion-readiness",
		"composition promotion readiness",
		"composition-merge-draft",
		"composition merge draft",
		"composition-liner-blend",
		"composition liner blend",
		"composition `liner.md` blend",
		"composition-skill-conflicts",
		"composition skill conflicts",
		"composition skill conflict review",
		"composition-copy-packet",
		"composition copy packet",
		"composition-copy-apply",
		"composition copy apply",
		"composition-production-merge",
		"composition production merge",
		"composition-apply",
		"composition routing apply",
	}):
		return "composition"
	case strings.Contains(value, "contradiction"):
		return "contradiction"
	case strings.Contains(value, "skill") && (strings.Contains(value, "alignment") || strings.Contains(value, "corpus")):
		return "skill alignment"
	case strings.Contains(value, "source-note") || strings.Contains(value, "source note"):
		return "source notes"
	case strings.Contains(value, "liner.md generation") || strings.Contains(value, "liner-md-generation"):
		return "generation"
	default:
		return "audit"
	}
}

func auditRows(items []auditFile) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, table.Row{item.Name, item.Type, item.Updated})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"No audits found", "", ""})
	}
	return rows
}

func (m Model) selectedAudit() (auditFile, bool) {
	index := m.auditTable.Cursor()
	if index < 0 || index >= len(m.auditItems) {
		return auditFile{}, false
	}
	return m.auditItems[index], true
}

func (m Model) handleAuditsKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case "r":
		return m.runContradictionAudit()
	case "s":
		return m.runSkillCorpusAudit()
	case "n":
		return m.runSourceNoteAudit()
	case "c":
		return m.runSourceNoteCleanupDraft()
	case "f":
		return m.runContradictionCleanupDraft()
	case "g":
		return m.runSkillGroundingDraftFromAudit()
	case "p":
		return m.createAuditAgentCleanupPacket()
	case "enter":
		if item, ok := m.selectedAudit(); ok {
			return m.openPreview(item.RelPath)
		}
	case "o":
		if item, ok := m.selectedAudit(); ok {
			m.note = "Opened " + item.RelPath
			return m, openPath(item.Path)
		}
	}
	return m, nil
}

func (m Model) runSkillGroundingDraftFromAudit() (Model, tea.Cmd) {
	item, ok := m.selectedAudit()
	if !ok || item.Type != "skill alignment" {
		m.err = "Select a skill-corpus alignment audit before drafting skill repairs."
		return m, nil
	}
	skill, err := skillRepairCandidateFromAudit(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	return m.startSkillGroundingReviewForItem(skill)
}

func skillRepairCandidateFromAudit(project string, report auditFile) (skillFile, error) {
	skills, err := loadSkillFiles(project)
	if err != nil {
		return skillFile{}, err
	}
	if len(skills) == 0 {
		return skillFile{}, fmt.Errorf("No skills found in skills/.")
	}
	byRel := map[string]skillFile{}
	byBase := map[string]skillFile{}
	for _, skill := range skills {
		byRel[filepath.ToSlash(skill.RelPath)] = skill
		byBase[filepath.Base(skill.RelPath)] = skill
	}
	body, err := os.ReadFile(report.Path)
	if err != nil {
		return skillFile{}, fmt.Errorf("Could not read selected skill audit: %w", err)
	}
	for _, finding := range parseAuditFindingPreviews(string(body)) {
		if !auditFindingNeedsSkillRepair(finding.Status) {
			continue
		}
		subject := filepath.ToSlash(strings.TrimSpace(finding.Subject))
		if skill, ok := byRel[subject]; ok {
			return skill, nil
		}
		if skill, ok := byBase[filepath.Base(subject)]; ok {
			return skill, nil
		}
	}
	for _, skill := range skills {
		if skill.Status == "needs grounding" || skill.Status == "needs boundaries" {
			return skill, nil
		}
	}
	return skillFile{}, fmt.Errorf("No repairable skill findings found in the selected audit.")
}

func auditFindingNeedsSkillRepair(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return containsAny(status, []string{"needs grounding", "needs boundaries", "weak corpus signal"})
}

func (m Model) createAuditAgentCleanupPacket() (Model, tea.Cmd) {
	item, ok := m.selectedAudit()
	if !ok {
		m.err = "Select an audit report before creating an agent cleanup packet."
		return m, nil
	}
	if item.Type == "cleanup packet" {
		m.err = "Select the source audit report, not an existing cleanup packet."
		return m, nil
	}
	packet, err := writeAuditAgentCleanupPacket(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(packet)
	m.note = "Wrote " + packet.RelPath + "."
	return m.openPreview(packet.RelPath)
}

func (m Model) handleContradictionCleanupReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenAudits
		return m, nil
	case "o":
		rel, err := m.currentContradictionCleanupDraftRel()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.note = "Opened " + rel
		return m, openPath(filepath.Join(m.currentPath, rel))
	case "d":
		return m.discardContradictionCleanupDraft()
	case "enter":
		return m.acceptContradictionCleanupDraft()
	}
	return m, nil
}

func (m Model) runContradictionCleanupDraft() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot create a contradiction cleanup draft without a project path."
		return m, nil
	}
	item, err := writeContradictionCleanupDraft(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	return m.startContradictionCleanupReview(item, "Review before applying contradiction decisions to LINER.md.")
}

func (m Model) startContradictionCleanupReview(item auditFile, note string) (Model, tea.Cmd) {
	data, err := os.ReadFile(item.Path)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.setPreviewContent(item.RelPath, string(data))
	m.screen = screenContradictionCleanupReview
	m.note = note
	return m, nil
}

func (m Model) handleSourceNoteCleanupReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenAudits
		return m, nil
	case "o":
		rel, err := m.currentSourceNoteCleanupDraftRel()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.note = "Opened " + rel
		return m, openPath(filepath.Join(m.currentPath, rel))
	case "d":
		return m.discardSourceNoteCleanupDraft()
	case "enter":
		return m.acceptSourceNoteCleanupDraft()
	}
	return m, nil
}

func (m Model) currentContradictionCleanupDraftRel() (string, error) {
	rel := filepath.Clean(strings.TrimSpace(m.previewRel))
	if rel == "" || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.Contains(filepath.Base(rel), "contradiction-cleanup-draft") {
		return "", fmt.Errorf("No contradiction cleanup draft is selected.")
	}
	return rel, nil
}

func (m Model) acceptContradictionCleanupDraft() (Model, tea.Cmd) {
	rel, err := m.currentContradictionCleanupDraftRel()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	item, err := applyContradictionCleanup(m.currentPath, rel)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	m.note = "Applied contradiction decisions to LINER.md."
	return m.openPreview(item.RelPath)
}

func (m Model) discardContradictionCleanupDraft() (Model, tea.Cmd) {
	rel, err := m.currentContradictionCleanupDraftRel()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := os.Remove(filepath.Join(m.currentPath, rel)); err != nil && !os.IsNotExist(err) {
		m.err = "Could not discard contradiction cleanup draft: " + err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m.screen = screenAudits
	m.note = "Discarded " + rel + "."
	return m, nil
}

func (m Model) runSourceNoteCleanupDraft() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot create a cleanup draft without a project path."
		return m, nil
	}
	item, err := writeSourceNoteCleanupDraft(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	return m.startSourceNoteCleanupReview(item, "Review before applying cleanup to tape.yaml.")
}

func (m Model) startSourceNoteCleanupReview(item auditFile, note string) (Model, tea.Cmd) {
	data, err := os.ReadFile(item.Path)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.setPreviewContent(item.RelPath, string(data))
	m.screen = screenSourceNoteCleanupReview
	m.note = note
	return m, nil
}

func (m Model) currentSourceNoteCleanupDraftRel() (string, error) {
	rel := filepath.Clean(strings.TrimSpace(m.previewRel))
	if rel == "" || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.Contains(filepath.Base(rel), "source-note-cleanup-draft") {
		return "", fmt.Errorf("No source-note cleanup draft is selected.")
	}
	return rel, nil
}

func (m Model) acceptSourceNoteCleanupDraft() (Model, tea.Cmd) {
	rel, err := m.currentSourceNoteCleanupDraftRel()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	item, updated, err := applySourceNoteCleanup(m.currentPath, rel)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	m.note = fmt.Sprintf("Applied %d source-note cleanup(s).", updated)
	return m.openPreview(item.RelPath)
}

func (m Model) discardSourceNoteCleanupDraft() (Model, tea.Cmd) {
	rel, err := m.currentSourceNoteCleanupDraftRel()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := os.Remove(filepath.Join(m.currentPath, rel)); err != nil && !os.IsNotExist(err) {
		m.err = "Could not discard source-note cleanup draft: " + err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m.screen = screenAudits
	m.note = "Discarded " + rel + "."
	return m, nil
}

func (m Model) selectAuditReport(item auditFile) Model {
	for index, current := range m.auditItems {
		if current.RelPath == item.RelPath {
			m.auditTable.SetCursor(index)
			break
		}
	}
	return m
}

func (m Model) runContradictionAudit() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot run an audit without a project path."
		return m, nil
	}
	item, err := writeContradictionAudit(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	m.screen = screenAudits
	m.note = "Wrote " + item.RelPath + "."
	return m, nil
}

func (m Model) runSkillCorpusAudit() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot run an audit without a project path."
		return m, nil
	}
	item, err := writeSkillCorpusAudit(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	m.screen = screenAudits
	m.note = "Wrote " + item.RelPath + "."
	return m, nil
}

func (m Model) runSourceNoteAudit() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot run an audit without a project path."
		return m, nil
	}
	item, err := writeSourceNoteQualityAudit(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items, err := loadAuditFiles(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.applyAuditItems(items)
	m = m.selectAuditReport(item)
	m.screen = screenAudits
	m.note = "Wrote " + item.RelPath + "."
	return m, nil
}

func (m Model) viewAudits() string {
	width := styles.ClampWidth(m.width - 4)
	tableView := newVisibleDataTable(
		auditColumns(width),
		auditRows(m.auditItems),
		width,
		max(5, min(len(m.auditItems)+1, max(5, m.height-18))),
		true,
		m.auditTable.Cursor(),
	)
	count := styles.Section.Render("Audits") + " " + styles.ReportBody.Render(intLabel(len(m.auditItems), "report"))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Audits"),
		styles.Subtitle.Render("Existing reports for contradictions, alignment, and source-note quality."),
		"",
		count,
		tableView.View(),
		"",
		styles.ReportSection.Render("Selected"),
		m.auditSelectedDetail(width),
		"",
		styles.ReportSection.Render("Findings"),
		m.auditFindingsPreview(width),
		"",
		styles.ReportSection.Render("Actions"),
		m.auditActionsTable(width).View(),
	)
}

func (m Model) auditSelectedDetail(width int) string {
	item, ok := m.selectedAudit()
	if !ok {
		return styles.SoftText.Render("No audit report selected.")
	}
	rows := []metadataTableRow{
		{Field: "Report", Value: item.Name},
		{Field: "Type", Value: item.Type},
		{Field: "Updated", Value: item.Updated},
		{Field: "Path", Value: filepath.ToSlash(item.RelPath)},
	}
	if body, err := os.ReadFile(item.Path); err == nil {
		for _, field := range []string{"Source audit", "Audit type", "Packet", "Draft", "Child reference", "Child project"} {
			if value, ok := markdownMetadataValue(string(body), field); ok {
				rows = append(rows, metadataTableRow{Field: auditMetadataFieldLabel(field), Value: value})
			}
		}
	}
	return newMetadataTable(width, dedupeMetadataRows(rows)).View()
}

func auditMetadataFieldLabel(field string) string {
	switch field {
	case "Audit type":
		return "Source type"
	default:
		return field
	}
}

func (m Model) auditFindingsPreview(width int) string {
	item, ok := m.selectedAudit()
	if !ok {
		return styles.SoftText.Render("Select an audit report to inspect its structured findings.")
	}
	data, err := os.ReadFile(item.Path)
	if err != nil {
		return styles.ErrorText.Render("Could not read selected audit: " + err.Error())
	}
	findings := parseAuditFindingPreviews(string(data))
	if len(findings) == 0 {
		return styles.SoftText.Render("No structured findings table found. Preview the full report for details.")
	}
	tableView := auditFindingPreviewTable(width, findings)
	top := findings[0]
	detail := "Top finding: " + top.Evidence
	if top.Recommendation != "" && top.Recommendation != top.Evidence {
		detail += " Recommendation: " + top.Recommendation
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		tableView.View(),
		styles.MutedText.Width(width).Render(strings.Join(wrapWords(detail, width), "\n")),
	)
}

func auditFindingPreviewTable(width int, findings []auditFindingPreview) table.Model {
	width = max(width, 64)
	statusWidth := 16
	subjectWidth := max(20, width/3)
	actionWidth := max(24, width-statusWidth-subjectWidth-8)
	rows := make([]table.Row, 0, min(len(findings), 5))
	for _, finding := range findings {
		if len(rows) >= 5 {
			break
		}
		rows = append(rows, table.Row{
			truncateForTable(finding.Status, statusWidth),
			truncateForTable(finding.Subject, subjectWidth),
			truncateForTable(finding.Recommendation, actionWidth),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"none", "", ""})
	}
	return newDataTable(
		[]table.Column{
			{Title: "Status", Width: statusWidth},
			{Title: "Subject", Width: subjectWidth},
			{Title: "Action", Width: actionWidth},
		},
		rows,
		width,
		len(rows)+1,
		false,
	)
}

func parseAuditFindingPreviews(markdown string) []auditFindingPreview {
	lines := strings.Split(markdown, "\n")
	for index := 0; index < len(lines); index++ {
		if !strings.HasPrefix(strings.TrimSpace(lines[index]), "|") {
			continue
		}
		headers := splitMarkdownTableRow(lines[index])
		if len(headers) < 2 || index+1 >= len(lines) || !isMarkdownSeparatorRow(lines[index+1]) {
			continue
		}
		headerIndex := map[string]int{}
		for headerIndexValue, header := range headers {
			headerIndex[normalizeAuditTableHeader(header)] = headerIndexValue
		}
		subjectIndex := firstAuditColumn(headerIndex, []string{"location", "source", "skill", "children", "child", "audit", "route"}, 0)
		statusIndex := firstAuditColumn(headerIndex, []string{"status", "severity", "previous status", "current status"}, 1)
		evidenceIndex := firstAuditColumn(headerIndex, []string{"evidence", "decision", "applied note", "note brief"}, -1)
		recommendationIndex := firstAuditColumn(headerIndex, []string{"recommendation", "reason", "reasoning", "draft action", "decision", "applied note"}, -1)

		findings := []auditFindingPreview{}
		for rowIndex := index + 2; rowIndex < len(lines); rowIndex++ {
			line := strings.TrimSpace(lines[rowIndex])
			if !strings.HasPrefix(line, "|") {
				break
			}
			cells := splitMarkdownTableRow(line)
			if len(cells) < len(headers) {
				continue
			}
			finding := auditFindingPreview{
				Subject:        cellAt(cells, subjectIndex),
				Status:         cellAt(cells, statusIndex),
				Evidence:       cellAt(cells, evidenceIndex),
				Recommendation: cellAt(cells, recommendationIndex),
			}
			if finding.Recommendation == "" {
				finding.Recommendation = finding.Evidence
			}
			if finding.Evidence == "" {
				finding.Evidence = finding.Recommendation
			}
			if finding.Subject == "" && len(cells) > 0 {
				finding.Subject = cells[0]
			}
			if finding.Status == "" && len(cells) > 1 {
				finding.Status = cells[1]
			}
			if finding.Subject == "" && finding.Status == "" && finding.Recommendation == "" {
				continue
			}
			findings = append(findings, finding)
		}
		if len(findings) > 0 {
			return findings
		}
	}
	return nil
}

func splitMarkdownTableRow(row string) []string {
	row = strings.TrimSpace(row)
	if strings.HasPrefix(row, "|") {
		row = row[1:]
	}
	if strings.HasSuffix(row, "|") {
		row = row[:len(row)-1]
	}
	var cells []string
	var current strings.Builder
	escaped := false
	for _, r := range row {
		if r == '|' && !escaped {
			cells = append(cells, cleanMarkdownTableCell(current.String()))
			current.Reset()
			continue
		}
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		current.WriteRune(r)
	}
	if escaped {
		current.WriteRune('\\')
	}
	cells = append(cells, cleanMarkdownTableCell(current.String()))
	return cells
}

func cleanMarkdownTableCell(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") && len(value) >= 2 {
		value = strings.TrimPrefix(strings.TrimSuffix(value, "`"), "`")
	}
	value = strings.ReplaceAll(value, "\\|", "|")
	return strings.Join(strings.Fields(value), " ")
}

func isMarkdownSeparatorRow(row string) bool {
	cells := splitMarkdownTableRow(row)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		value := strings.Trim(cell, " :-")
		if value != "" {
			return false
		}
	}
	return true
}

func normalizeAuditTableHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "`", "")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func firstAuditColumn(indexes map[string]int, names []string, fallback int) int {
	for _, name := range names {
		if index, ok := indexes[name]; ok {
			return index
		}
	}
	return fallback
}

func cellAt(cells []string, index int) string {
	if index < 0 || index >= len(cells) {
		return ""
	}
	return cells[index]
}

func truncateForTable(value string, width int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if width <= 0 || len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func (m Model) auditActionsTable(width int) table.Model {
	item, ok := m.selectedAudit()
	if !ok {
		return newActionTable(width, []actionTableRow{
			{Key: "r", Action: "Run contradiction audit", Writes: "audit report"},
			{Key: "s", Action: "Run skill-corpus audit", Writes: "audit report"},
			{Key: "n", Action: "Check source-note quality", Writes: "audit report"},
		})
	}
	rows := []actionTableRow{
		{Key: "enter / o", Action: "Preview or open selected report", Writes: "read-only"},
	}
	if item.Type != "cleanup packet" {
		rows = append(rows, actionTableRow{Key: "p", Action: "Create agent cleanup packet", Writes: "external plan"})
	}
	switch item.Type {
	case "contradiction":
		rows = append(rows,
			actionTableRow{Key: "f", Action: "Draft contradiction cleanup", Writes: "reviewed apply"},
			actionTableRow{Key: "r", Action: "Re-run contradiction audit", Writes: "audit report"},
		)
	case "source notes":
		rows = append(rows,
			actionTableRow{Key: "c", Action: "Draft source-note cleanup", Writes: "reviewed apply"},
			actionTableRow{Key: "n", Action: "Re-check source-note quality", Writes: "audit report"},
		)
	case "skill alignment":
		rows = append(rows,
			actionTableRow{Key: "g", Action: "Draft first skill repair", Writes: "review draft"},
			actionTableRow{Key: "s", Action: "Re-run skill-corpus audit", Writes: "audit report"},
		)
	default:
		rows = append(rows,
			actionTableRow{Key: "r", Action: "Run contradiction audit", Writes: "audit report"},
			actionTableRow{Key: "s", Action: "Run skill-corpus audit", Writes: "audit report"},
			actionTableRow{Key: "n", Action: "Check source-note quality", Writes: "audit report"},
		)
	}
	return newActionTable(width, rows)
}

func (m Model) viewContradictionCleanupReview() string {
	width := styles.ClampWidth(m.width - 4)
	preview := m.preview
	preview.SetWidth(max(1, width))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Review Contradiction Cleanup"),
		styles.Subtitle.Render("Proposed LINER.md contradiction decisions. Review before applying."),
		"",
		preview.View(),
		styles.ReportSection.Render("Actions"),
		newReviewActionTable(width, "Apply to LINER.md", "LINER.md + audit", "Audits").View(),
	)
}

func (m Model) viewSourceNoteCleanupReview() string {
	width := styles.ClampWidth(m.width - 4)
	preview := m.preview
	preview.SetWidth(max(1, width))
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Title.Render("Review Source-Note Cleanup"),
		styles.Subtitle.Render("Proposed tape.yaml source-note edits. Review before applying."),
		"",
		preview.View(),
		styles.ReportSection.Render("Actions"),
		newReviewActionTable(width, "Apply to tape.yaml", "tape.yaml + audit", "Audits").View(),
	)
}

func writeContradictionAudit(project string) (auditFile, error) {
	inputs, err := contradictionAuditInputs(project)
	if err != nil {
		return auditFile{}, err
	}
	if len(inputs) == 0 {
		return auditFile{}, fmt.Errorf("No auditable files found. Compile MIXTAPE.md, generate LINER.md, or add skills first.")
	}
	findings, omitted, err := contradictionFindings(inputs)
	if err != nil {
		return auditFile{}, err
	}

	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-contradiction-audit"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderContradictionAudit(now, inputs, findings, omitted)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "contradiction",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeAuditAgentCleanupPacket(project string, item auditFile) (auditFile, error) {
	body, err := os.ReadFile(item.Path)
	if err != nil {
		return auditFile{}, fmt.Errorf("Could not read selected audit: %w", err)
	}
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	typeSlug := slug(item.Type)
	if typeSlug == "" {
		typeSlug = "audit"
	}
	name := now.Format("2006-01-02-150405") + "-" + typeSlug + "-agent-cleanup-packet"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	findings := parseAuditFindingPreviews(string(body))
	if err := os.WriteFile(path, []byte(renderAuditAgentCleanupPacket(now, item, findings)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "cleanup packet",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func renderAuditAgentCleanupPacket(now time.Time, item auditFile, findings []auditFindingPreview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Audit Agent Cleanup Packet\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Source audit: `%s`\n", item.RelPath)
	fmt.Fprintf(&b, "Audit type: %s\n\n", item.Type)
	b.WriteString("## Purpose\n\n")
	b.WriteString("Use this packet to ask an external agent to draft cleanup changes from a reviewed audit. The Go TUI prepares the scope and allowed outputs; it does not run an agent or apply edits automatically.\n\n")
	b.WriteString("## Allowed Outputs\n\n")
	for _, line := range auditAgentAllowedOutputs(item.Type) {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	b.WriteString("\n## Hard Boundaries\n\n")
	b.WriteString("- Do not update production files directly: `LINER.md`, `MIXTAPE.md`, `tape.yaml`, `skills/*.md`, source files, or child project files.\n")
	b.WriteString("- Write reviewable drafts only, then return to the TUI review/apply flow.\n")
	b.WriteString("- Preserve citations, source notes, skill boundaries, and audit reasoning when proposing changes.\n")
	b.WriteString("- If evidence is missing or contradictory, write a question or follow-up audit recommendation instead of guessing.\n\n")
	b.WriteString("## Findings To Address\n\n")
	if len(findings) == 0 {
		b.WriteString("No structured findings table was parsed from the selected audit. Read the source audit directly and draft only reviewable cleanup artifacts.\n\n")
	} else {
		b.WriteString("| Status | Subject | Evidence | Recommended action |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for index, finding := range findings {
			if index >= 12 {
				break
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				escapeMarkdownTable(finding.Status),
				escapeMarkdownTable(finding.Subject),
				escapeMarkdownTable(finding.Evidence),
				escapeMarkdownTable(finding.Recommendation),
			)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Completion Checklist\n\n")
	b.WriteString("- Draft files are written only under `working/`.\n")
	b.WriteString("- Every proposed change names the audit finding or source evidence that caused it.\n")
	b.WriteString("- No review/apply audit is written until a human accepts the draft in the TUI.\n")
	b.WriteString("- The final response names the draft paths and any blocked findings.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No agent was run by the Go TUI.\n")
	b.WriteString("- No cleanup changes were applied by this packet.\n")
	b.WriteString("- Treat this as an external cleanup plan until agent execution has explicit configuration and review gates.\n")
	return b.String()
}

func auditAgentAllowedOutputs(auditType string) []string {
	switch auditType {
	case "contradiction":
		return []string{
			"`working/audits/<date>-contradiction-cleanup-draft.md` with proposed operating-layer decisions.",
			"Optional follow-up contradiction audit notes under `working/audits/`.",
		}
	case "source notes":
		return []string{
			"`working/audits/<date>-source-note-cleanup-draft.md` with proposed `tape.yaml` note repairs.",
			"Optional list of sources that need new evidence before cleanup.",
		}
	case "skill alignment":
		return []string{
			"`working/skills/<skill>-grounding-draft.md` with Source Grounding and Boundaries repairs.",
			"Optional skill follow-up notes under `working/audits/` when the corpus is insufficient.",
		}
	case "composition":
		return []string{
			"`working/LINER-composition-draft.md` with scoped parent routing updates.",
			"Optional composition audit notes under `working/audits/`.",
		}
	default:
		return []string{
			"Reviewable draft artifacts under `working/` only.",
			"Follow-up audit notes under `working/audits/` when no safe draft is possible.",
		}
	}
}

func writeContradictionCleanupDraft(project string) (auditFile, error) {
	inputs, err := contradictionAuditInputs(project)
	if err != nil {
		return auditFile{}, err
	}
	if len(inputs) == 0 {
		return auditFile{}, fmt.Errorf("No auditable files found. Compile MIXTAPE.md, generate LINER.md, or add skills first.")
	}
	findings, omitted, err := contradictionFindings(inputs)
	if err != nil {
		return auditFile{}, err
	}
	if len(findings) == 0 {
		return auditFile{}, fmt.Errorf("No contradiction markers found to draft cleanup decisions.")
	}
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-contradiction-cleanup-draft"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderContradictionCleanupDraft(now, inputs, findings, omitted)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "contradiction",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func applyContradictionCleanup(project string, draftRel string) (auditFile, error) {
	draftPath := filepath.Join(project, draftRel)
	draft, err := os.ReadFile(draftPath)
	if err != nil {
		return auditFile{}, fmt.Errorf("Could not read contradiction cleanup draft: %w", err)
	}
	linerPath := filepath.Join(project, "LINER.md")
	existing, err := os.ReadFile(linerPath)
	if err != nil && !os.IsNotExist(err) {
		return auditFile{}, fmt.Errorf("Could not read LINER.md: %w", err)
	}
	backupRel := ""
	if err == nil {
		backupRel = filepath.Join("working", "audits", time.Now().Format("2006-01-02-150405")+"-contradiction-cleanup-LINER-backup.md")
		if err := os.MkdirAll(filepath.Join(project, filepath.Dir(backupRel)), 0o755); err != nil {
			return auditFile{}, err
		}
		if err := os.WriteFile(filepath.Join(project, backupRel), existing, 0o644); err != nil {
			return auditFile{}, fmt.Errorf("Could not back up LINER.md: %w", err)
		}
	}
	updated := applyContradictionCleanupSection(string(existing), string(draft))
	if err := os.WriteFile(linerPath, []byte(updated), 0o644); err != nil {
		return auditFile{}, fmt.Errorf("Could not write LINER.md: %w", err)
	}
	return writeContradictionCleanupApplyAudit(project, draftRel, backupRel, len(draft))
}

func applyContradictionCleanupSection(existing string, draft string) string {
	section := contradictionCleanupStartMarker + "\n" + strings.TrimSpace(draft) + "\n" + contradictionCleanupEndMarker
	existing = strings.TrimSpace(existing)
	start := strings.Index(existing, contradictionCleanupStartMarker)
	end := strings.Index(existing, contradictionCleanupEndMarker)
	if start >= 0 && end >= start {
		end += len(contradictionCleanupEndMarker)
		return strings.TrimSpace(existing[:start]) + "\n\n" + section + "\n\n" + strings.TrimSpace(existing[end:]) + "\n"
	}
	if existing == "" {
		return section + "\n"
	}
	return existing + "\n\n" + section + "\n"
}

func writeContradictionCleanupApplyAudit(project string, draftRel string, backupRel string, draftBytes int) (auditFile, error) {
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-contradiction-cleanup-apply"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	var b strings.Builder
	fmt.Fprintf(&b, "# Contradiction Cleanup Apply Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This audit records reviewed contradiction decisions applied to `LINER.md` from the Go Audit Center.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Reviewed draft: `%s`\n", draftRel)
	fmt.Fprintf(&b, "- Draft bytes: %d\n", draftBytes)
	if backupRel != "" {
		fmt.Fprintf(&b, "- Previous `LINER.md` backup: `%s`\n", backupRel)
	} else {
		b.WriteString("- Previous `LINER.md`: not present; created a new operating layer containing the contradiction decisions.\n")
	}
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- `LINER.md` was updated only after the contradiction cleanup draft review step.\n")
	b.WriteString("- No source files, skill files, `MIXTAPE.md`, or `tape.yaml` were changed.\n")
	b.WriteString("- Use a follow-up contradiction audit to confirm the decisions now explain the tension.\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "contradiction",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func contradictionAuditBaseInputs() []string {
	return []string{"LINER.md", "MIXTAPE.md", "synthesis.md", filepath.Join("working", "04-quality-checks.md")}
}

func contradictionAuditInputs(project string) ([]auditFile, error) {
	var inputs []auditFile
	for _, rel := range contradictionAuditBaseInputs() {
		path := projectAbsPath(project, rel)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		inputs = append(inputs, auditFile{Name: rel, RelPath: rel, Path: path, Updated: info.ModTime().Format("2006-01-02")})
	}
	skillsDir := filepath.Join(project, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(skillsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		rel := filepath.Join("skills", entry.Name())
		inputs = append(inputs, auditFile{Name: rel, RelPath: rel, Path: path, Updated: info.ModTime().Format("2006-01-02")})
	}
	sort.Slice(inputs, func(i int, j int) bool {
		return strings.ToLower(inputs[i].RelPath) < strings.ToLower(inputs[j].RelPath)
	})
	return inputs, nil
}

func contradictionFindings(inputs []auditFile) ([]contradictionFinding, int, error) {
	findings := []contradictionFinding{}
	omitted := 0
	for _, input := range inputs {
		data, err := os.ReadFile(input.Path)
		if err != nil {
			return nil, 0, err
		}
		for index, line := range strings.Split(string(data), "\n") {
			finding, ok := contradictionFindingForLine(input.RelPath, index+1, line)
			if !ok {
				continue
			}
			if len(findings) >= 40 {
				omitted++
				continue
			}
			findings = append(findings, finding)
		}
	}
	return findings, omitted, nil
}

func contradictionFindingForLine(rel string, lineNumber int, line string) (contradictionFinding, bool) {
	text := strings.TrimSpace(strings.TrimPrefix(line, "-"))
	text = strings.TrimSpace(strings.TrimPrefix(text, "*"))
	if text == "" {
		return contradictionFinding{}, false
	}
	lower := strings.ToLower(text)
	switch {
	case containsAny(lower, []string{"contradict", "conflict", "tension", "inconsistent"}):
		return contradictionFinding{Severity: "high", RelPath: rel, Line: lineNumber, Evidence: text, Reason: "Names a contradiction, conflict, or tension that should be resolved or documented."}, true
	case containsAny(lower, []string{"always", "never", "must", "do not", "don't", "cannot", "should not"}):
		return contradictionFinding{Severity: "medium", RelPath: rel, Line: lineNumber, Evidence: text, Reason: "Hard rule. Compare it against exceptions and neighboring skills before applying broadly."}, true
	case containsAny(lower, []string{"unless", "except", "however", "but ", "trade-off", "tradeoff", "outweigh"}):
		return contradictionFinding{Severity: "low", RelPath: rel, Line: lineNumber, Evidence: text, Reason: "Boundary or exception marker. Check whether the operating layer captures the same limitation."}, true
	default:
		return contradictionFinding{}, false
	}
}

func renderContradictionAudit(now time.Time, inputs []auditFile, findings []contradictionFinding, omitted int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Contradiction Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local audit scans operating-layer and corpus-facing files for hard rules, exception language, and explicit tension markers. It does not rewrite any project files.\n\n")
	b.WriteString("## Inputs\n\n")
	for _, input := range inputs {
		fmt.Fprintf(&b, "- `%s`\n", input.RelPath)
	}
	b.WriteString("\n## Findings\n\n")
	if len(findings) == 0 {
		b.WriteString("No obvious contradiction markers were found by the local heuristic.\n\n")
	} else {
		b.WriteString("| Severity | Location | Evidence | Reason |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, finding := range findings {
			fmt.Fprintf(&b, "| %s | `%s:%d` | %s | %s |\n",
				escapeMarkdownTable(finding.Severity),
				finding.RelPath,
				finding.Line,
				escapeMarkdownTable(finding.Evidence),
				escapeMarkdownTable(finding.Reason),
			)
		}
		if omitted > 0 {
			fmt.Fprintf(&b, "\n%d additional marker(s) were omitted to keep this first-pass report readable.\n\n", omitted)
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("## Recommended Review\n\n")
	b.WriteString("- Resolve true conflicts in the most specific artifact first: skill file, then `LINER.md`, then source notes.\n")
	b.WriteString("- Preserve useful tensions by naming the condition where each rule applies.\n")
	b.WriteString("- If a source is wrong or outdated, update the corpus before changing the operating layer.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No files were changed by this audit.\n")
	b.WriteString("- Apply decisions only after reviewing the evidence above.\n")
	return b.String()
}

func renderContradictionCleanupDraft(now time.Time, inputs []auditFile, findings []contradictionFinding, omitted int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Contradiction Cleanup Draft\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This draft proposes operating-layer decisions for contradiction markers found in corpus-facing files. It does not edit source files, skill files, `MIXTAPE.md`, or `tape.yaml` until reviewed and accepted into `LINER.md`.\n\n")
	b.WriteString("## Inputs\n\n")
	for _, input := range inputs {
		fmt.Fprintf(&b, "- `%s`\n", input.RelPath)
	}
	b.WriteString("\n## Proposed LINER.md Decisions\n\n")
	b.WriteString("| Severity | Location | Decision | Reasoning |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, finding := range findings {
		fmt.Fprintf(&b, "| %s | `%s:%d` | %s | %s |\n",
			escapeMarkdownTable(finding.Severity),
			finding.RelPath,
			finding.Line,
			escapeMarkdownTable(contradictionCleanupDecision(finding)),
			escapeMarkdownTable(contradictionCleanupReasoning(finding)),
		)
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "\n%d additional marker(s) were omitted to keep this draft reviewable.\n", omitted)
	}
	b.WriteString("\n## Review Checklist\n\n")
	b.WriteString("- Confirm each proposed decision preserves the most specific artifact's scope.\n")
	b.WriteString("- Edit this draft before applying if a source, skill, or operating rule should win for a different reason.\n")
	b.WriteString("- After applying, re-run contradiction audit to confirm the tension is documented.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No files were changed except this cleanup draft.\n")
	b.WriteString("- Accepting applies this draft to `LINER.md` between managed contradiction-decision markers.\n")
	return b.String()
}

func contradictionCleanupDecision(finding contradictionFinding) string {
	switch finding.Severity {
	case "high":
		return "Name the conflict and require the agent to choose the narrower source or skill before acting."
	case "medium":
		return "Treat this hard rule as the default only when no documented exception applies."
	default:
		return "Preserve this exception as a boundary condition in the operating layer."
	}
}

func contradictionCleanupReasoning(finding contradictionFinding) string {
	return finding.Reason + " Evidence: " + finding.Evidence
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func escapeMarkdownTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
