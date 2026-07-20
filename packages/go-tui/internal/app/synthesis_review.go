package app

import (
	"fmt"
	"math"
	"os"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

const (
	synthesisReviewStillCurrent = iota
	synthesisReviewPatch
)

type semanticReviewKind int

const (
	semanticReviewSynthesis semanticReviewKind = iota
	semanticReviewOperatingLayer
)

const (
	operatingLayerReviewLINER = iota
	operatingLayerReviewSkill
)

func newSynthesisReviewArea(width int) textarea.Model {
	area := newCreateArea(width)
	area.Placeholder = "Write the reviewed synthesis revision…"
	area.MaxHeight = 8
	area.Blur()
	return area
}

func newOperatingLayerReviewSkillArea(width int) textarea.Model {
	area := newCreateArea(width)
	area.Placeholder = "Write the reviewed Project Skill revision…"
	area.MaxHeight = 8
	area.Blur()
	return area
}

func newSynthesisReviewViewport(width int, height int) viewport.Model {
	return viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
}

func (m Model) startSynthesisReview() (Model, tea.Cmd) {
	if m.projectNextKind() != projectNextReviewSynthesis {
		m.err = "Liner Core does not currently require Review Synthesis."
		return m, nil
	}
	current, err := os.ReadFile(projectAbsPath(m.currentPath, "synthesis.md"))
	if err != nil {
		m.err = "Review Synthesis could not read the current synthesis: " + err.Error()
		return m, nil
	}
	m.screen = screenSynthesisReview
	m.synthesisReviewKind = semanticReviewSynthesis
	m.synthesisReviewChoice = synthesisReviewStillCurrent
	m.synthesisReviewEditing = false
	m.synthesisReviewPlan = nil
	m.synthesisReviewLoading = false
	m.synthesisReviewApplying = false
	m.synthesisReviewReconcile = false
	m.synthesisReviewCurrentText = string(current)
	m.operatingLayerReviewArtifact = operatingLayerReviewLINER
	m.operatingLayerReviewSkillPath = ""
	m.operatingLayerReviewSkillCurrentText = ""
	m.syncSemanticReviewCurrent(true)
	m.synthesisReviewArea.SetValue(string(current))
	m.synthesisReviewArea.Blur()
	m.note = "Review is write-free until you approve the exact Core Change Set."
	m.err = ""
	return m, nil
}

func (m Model) startPreparedSynthesisReview() (Model, tea.Cmd) {
	previousScreen := m.screen
	m, _ = m.startSynthesisReview()
	if m.screen != screenSynthesisReview || m.err != "" {
		return m, nil
	}
	// Planning is read-only, so keep the current surface stable until Core has
	// returned the complete approval record. This avoids flashing a half-ready
	// review screen between lifecycle steps.
	m.screen = previousScreen
	m.synthesisReviewLoading = true
	m.note = "Sources saved. Preparing the Synthesis approval…"
	return m, planSynthesisReview(m.runner, m.currentPath, map[string]any{
		"type":        "synthesis.review",
		"disposition": "still_current",
	})
}

func (m Model) startOperatingLayerReview() (Model, tea.Cmd) {
	if m.projectNextKind() != projectNextReviewOperatingLayer {
		m.err = "Liner Core does not currently require Review Operating Layer."
		return m, nil
	}
	current, err := os.ReadFile(projectAbsPath(m.currentPath, "LINER.md"))
	if err != nil {
		m.err = "Review Operating Layer could not read LINER.md: " + err.Error()
		return m, nil
	}
	skillPath := ""
	skillText := ""
	if snapshot := m.currentProjectSnapshot(); snapshot != nil && snapshot.Lifecycle.ProjectSkill.Path != nil {
		skillPath = strings.TrimSpace(*snapshot.Lifecycle.ProjectSkill.Path)
	}
	if skillPath != "" {
		skillContent, readErr := os.ReadFile(projectAbsPath(m.currentPath, skillPath))
		if readErr != nil {
			m.err = "Review Operating Layer could not read the declared Project Skill: " + readErr.Error()
			return m, nil
		}
		skillText = string(skillContent)
	}
	m.screen = screenSynthesisReview
	m.synthesisReviewKind = semanticReviewOperatingLayer
	m.synthesisReviewChoice = synthesisReviewStillCurrent
	m.operatingLayerReviewArtifact = operatingLayerReviewLINER
	m.synthesisReviewEditing = false
	m.synthesisReviewPlan = nil
	m.synthesisReviewLoading = false
	m.synthesisReviewApplying = false
	m.synthesisReviewReconcile = false
	m.synthesisReviewCurrentText = string(current)
	m.synthesisReviewArea.SetValue(string(current))
	m.synthesisReviewArea.Blur()
	m.operatingLayerReviewSkillPath = skillPath
	m.operatingLayerReviewSkillCurrentText = skillText
	m.operatingLayerReviewSkillArea.SetValue(skillText)
	m.operatingLayerReviewSkillArea.Blur()
	m.syncOperatingLayerReviewCurrent()
	m.note = "Review is write-free until you approve the exact Core Change Set."
	m.err = ""
	return m, nil
}

func (m *Model) syncOperatingLayerReviewCurrent() {
	m.syncSemanticReviewCurrent(true)
}

func (m Model) currentSemanticReviewText() string {
	if m.synthesisReviewKind == semanticReviewOperatingLayer && m.operatingLayerReviewArtifact == operatingLayerReviewSkill && m.operatingLayerReviewSkillPath != "" {
		return m.operatingLayerReviewSkillCurrentText
	}
	return m.synthesisReviewCurrentText
}

func (m *Model) syncSemanticReviewCurrent(reset bool) {
	position := m.synthesisReviewCurrent.ScrollPercent()
	width := max(20, m.synthesisReviewCurrent.Width())
	m.synthesisReviewCurrent.SetContent(wrapSynthesisReviewContent(m.currentSemanticReviewText(), width))
	if reset {
		m.synthesisReviewCurrent.GotoTop()
		return
	}
	maxOffset := max(0, m.synthesisReviewCurrent.TotalLineCount()-m.synthesisReviewCurrent.VisibleLineCount())
	m.synthesisReviewCurrent.SetYOffset(int(math.Round(position * float64(maxOffset))))
}

func (m Model) semanticReviewName() string {
	if m.synthesisReviewKind == semanticReviewOperatingLayer {
		return "Operating Layer"
	}
	return "Synthesis"
}

func (m Model) operatingLayerReviewCanSwitchArtifacts() bool {
	return m.synthesisReviewKind == semanticReviewOperatingLayer &&
		m.synthesisReviewChoice == synthesisReviewPatch &&
		m.operatingLayerReviewSkillPath != ""
}

func (m Model) activeSemanticReviewArtifactName() string {
	if m.synthesisReviewKind != semanticReviewOperatingLayer {
		return "synthesis"
	}
	if m.operatingLayerReviewArtifact == operatingLayerReviewSkill && m.operatingLayerReviewSkillPath != "" {
		return "Project Skill"
	}
	return "LINER.md"
}

func (m *Model) switchOperatingLayerReviewArtifact() bool {
	if !m.operatingLayerReviewCanSwitchArtifacts() {
		return false
	}
	m.activeSemanticReviewArea().Blur()
	if m.operatingLayerReviewArtifact == operatingLayerReviewLINER {
		m.operatingLayerReviewArtifact = operatingLayerReviewSkill
	} else {
		m.operatingLayerReviewArtifact = operatingLayerReviewLINER
	}
	m.syncOperatingLayerReviewCurrent()
	if m.synthesisReviewEditing {
		m.activeSemanticReviewArea().Focus()
		m.note = "Editing the proposed " + m.activeSemanticReviewArtifactName() + " revision. Tab switches artifacts; Ctrl+D finishes editing."
	} else {
		m.note = "Reviewing " + m.activeSemanticReviewArtifactName() + ". Tab switches Operating Layer artifacts."
	}
	return true
}

func (m Model) handleSynthesisReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if keyMsg.String() == "ctrl+c" {
		if m.synthesisReviewApplying || m.synthesisReviewReconcile {
			m.note = m.semanticReviewName() + " apply reconciliation cannot be interrupted. Wait for Core or replay the exact Change Set to recover its receipt."
			return m, nil
		}
		return m, tea.Quit
	}
	if m.synthesisReviewLoading {
		return m, nil
	}
	if m.synthesisReviewEditing {
		switch keyMsg.String() {
		case "ctrl+d", "esc":
			m.synthesisReviewEditing = false
			m.activeSemanticReviewArea().Blur()
			m.note = "Proposed revision remains local and unapplied. Press Enter for a Core preview."
			return m, nil
		case "tab", "shift+tab":
			m.switchOperatingLayerReviewArtifact()
			return m, nil
		}
		return m, nil
	}
	if m.synthesisReviewPlan != nil {
		switch keyMsg.String() {
		case "e":
			if m.synthesisReviewReconcile {
				m.note = "Receipt reconciliation is still required. Replay the exact Change Set before editing."
				return m, nil
			}
			m.synthesisReviewPlan = nil
			m.synthesisReviewChoice = synthesisReviewPatch
			m.beginSemanticReviewEditing(!m.semanticReviewHasLocalChanges())
			return m, nil
		case "esc":
			if m.synthesisReviewReconcile {
				m.note = "Receipt reconciliation is still required. Press Enter to replay this exact Change Set; Core will not duplicate committed work."
				return m, nil
			}
			m.synthesisReviewPlan = nil
			m.synthesisReviewReconcile = false
			m.note = "Discarded the unapplied Core Change Set. Canonical Project artifacts are unchanged."
			return m, nil
		case "enter":
			m.synthesisReviewLoading = true
			m.synthesisReviewApplying = true
			m.note = "Applying the exact reviewed " + m.semanticReviewName() + " disposition through Liner Core."
			return m, applySynthesisReview(m.runner, m.currentPath, *m.synthesisReviewPlan)
		case "up":
			m.synthesisReviewPlanView.ScrollUp(1)
		case "down":
			m.synthesisReviewPlanView.ScrollDown(1)
		case "pgup":
			m.synthesisReviewPlanView.PageUp()
		case "pgdown":
			m.synthesisReviewPlanView.PageDown()
		}
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.screen = screenProject
		m.synthesisReviewReconcile = false
		m.note = "Review " + m.semanticReviewName() + " closed without changing the Project."
	case "left":
		m.synthesisReviewChoice = synthesisReviewStillCurrent
	case "right":
		m.synthesisReviewChoice = synthesisReviewPatch
	case "tab", "shift+tab":
		m.switchOperatingLayerReviewArtifact()
	case "e":
		m.synthesisReviewChoice = synthesisReviewPatch
		m.beginSemanticReviewEditing(!m.semanticReviewHasLocalChanges())
	case "d":
		m.synthesisReviewChoice = synthesisReviewStillCurrent
		m.synthesisReviewArea.SetValue(m.synthesisReviewCurrentText)
		m.operatingLayerReviewSkillArea.SetValue(m.operatingLayerReviewSkillCurrentText)
		m.note = "Discarded the local revision. The canonical " + m.semanticReviewName() + " is unchanged."
	case "up":
		m.synthesisReviewCurrent.ScrollUp(1)
	case "down":
		m.synthesisReviewCurrent.ScrollDown(1)
	case "pgup":
		m.synthesisReviewCurrent.PageUp()
	case "pgdown":
		m.synthesisReviewCurrent.PageDown()
	case "enter":
		if m.synthesisReviewChoice == synthesisReviewPatch && !m.semanticReviewHasLocalChanges() {
			m.beginSemanticReviewEditing(true)
			return m, nil
		}
		operationType := "synthesis.review"
		if m.synthesisReviewKind == semanticReviewOperatingLayer {
			operationType = "operating_layer.review"
		}
		operation := map[string]any{"type": operationType, "disposition": "still_current"}
		if m.synthesisReviewChoice == synthesisReviewPatch {
			if m.synthesisReviewKind == semanticReviewOperatingLayer {
				linerContent := m.synthesisReviewArea.Value()
				skillContent := m.operatingLayerReviewSkillArea.Value()
				if strings.TrimSpace(linerContent) == "" {
					m.err = "A LINER.md revision cannot be empty. Edit the proposal or choose Still current."
					return m, nil
				}
				if linerContent != m.synthesisReviewCurrentText {
					operation["liner_content"] = linerContent
				}
				if m.operatingLayerReviewSkillPath != "" && skillContent != m.operatingLayerReviewSkillCurrentText {
					if strings.TrimSpace(skillContent) == "" {
						m.err = "A Project Skill revision cannot be empty. Edit the proposal or choose Still current."
						return m, nil
					}
					operation["skill_content"] = skillContent
				}
				if _, linerChanged := operation["liner_content"]; !linerChanged {
					if _, skillChanged := operation["skill_content"]; !skillChanged {
						m.err = "Approve revision requires a change to LINER.md or the Project Skill. Edit a proposal or choose Still current."
						return m, nil
					}
				}
			} else {
				content := m.synthesisReviewArea.Value()
				if strings.TrimSpace(content) == "" {
					m.err = "A synthesis revision cannot be empty. Edit the proposal or choose Still current."
					return m, nil
				}
				if content == m.synthesisReviewCurrentText {
					m.err = "No synthesis changes detected. Edit the proposal, or choose Approve unchanged."
					return m, nil
				}
				operation["content"] = content
			}
			operation["disposition"] = "patch"
		}
		m.synthesisReviewLoading = true
		m.synthesisReviewApplying = false
		m.synthesisReviewReconcile = false
		m.note = "Asking Liner Core for a write-free " + m.semanticReviewName() + " Change Set preview."
		return m, planSynthesisReview(m.runner, m.currentPath, operation)
	}
	return m, nil
}

func (m *Model) activeSemanticReviewArea() *textarea.Model {
	if m.synthesisReviewKind == semanticReviewOperatingLayer && m.operatingLayerReviewArtifact == operatingLayerReviewSkill && m.operatingLayerReviewSkillPath != "" {
		return &m.operatingLayerReviewSkillArea
	}
	return &m.synthesisReviewArea
}

func (m Model) semanticReviewHasLocalChanges() bool {
	if m.synthesisReviewKind == semanticReviewOperatingLayer {
		if m.synthesisReviewArea.Value() != m.synthesisReviewCurrentText {
			return true
		}
		return m.operatingLayerReviewSkillPath != "" && m.operatingLayerReviewSkillArea.Value() != m.operatingLayerReviewSkillCurrentText
	}
	return m.synthesisReviewArea.Value() != m.synthesisReviewCurrentText
}

func (m *Model) beginSemanticReviewEditing(noChangesYet bool) {
	m.synthesisReviewEditing = true
	m.activeSemanticReviewArea().Focus()
	m.err = ""
	prefix := ""
	if noChangesYet {
		prefix = "No changes yet. "
	}
	m.note = prefix + "Edit the proposed " + m.activeSemanticReviewArtifactName() + " revision. Ctrl+D finishes editing without writing canonical artifacts."
}

func shouldEditSemanticReviewText(keyMsg tea.KeyPressMsg) bool {
	switch keyMsg.String() {
	case "ctrl+d", "esc", "tab", "shift+tab":
		return false
	default:
		return true
	}
}

func planSynthesisReview(runner core.Runner, project string, operation map[string]any) tea.Cmd {
	return func() tea.Msg {
		plan, err := runner.PlanMaintenance(project, operation)
		return synthesisReviewPlannedMsg{plan: plan, err: err}
	}
}

func applySynthesisReview(runner core.Runner, project string, plan core.ProjectChangeSet) tea.Cmd {
	return func() tea.Msg {
		receipt, err := runner.ApplyMaintenance(project, plan, true)
		return synthesisReviewAppliedMsg{receipt: receipt, err: err}
	}
}

func (m Model) viewSynthesisReview() string {
	width := styles.ClampWidth(m.width - 4)
	title := "Review " + m.semanticReviewName()
	if m.synthesisReviewPlan != nil {
		subtitle := "The accepted Sources are saved. Confirm the current synthesis before Liner compiles the Project."
		if len(m.synthesisReviewPlan.Operations) == 1 && m.synthesisReviewPlan.Operations[0]["disposition"] == "patch" {
			subtitle = "Review the replacement text below. Nothing changes until you press Enter."
		} else if m.synthesisReviewKind == semanticReviewOperatingLayer {
			subtitle = "Confirm the current operating-layer artifacts. Nothing is rewritten until you press Enter."
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			styles.Title.Render(title),
			styles.Subtitle.Render(strings.Join(wrapWords(subtitle, width), "\n")),
			"",
			m.synthesisReviewPlanView.View(),
			styles.SoftText.Render(strings.Join(wrapWords(synthesisReviewPlanPosition(m.synthesisReviewPlanView), width), "\n")),
		)
	}
	subtitle := "The synthesis is the source-grounded point of view Liner places at the top of MIXTAPE.md. Confirm that it represents the accepted Sources before Compile."
	currentLabel := "Synthesis awaiting approval"
	stillCurrentLabel := "Approve unchanged"
	patchLabel := "Edit before approval"
	stillCurrentDetail := "Choose this when the synthesis below accurately represents the accepted Sources. Liner records your approval without rewriting synthesis.md."
	patchDetail := "Choose this when the synthesis needs correction. Edit it, then review the exact Core Change Set before synthesis.md is replaced."
	if m.synthesisReviewKind == semanticReviewOperatingLayer {
		subtitle = "LINER.md and the Project Skill tell future AI sessions how to use the corpus. Confirm that both still match the refreshed corpus before the Project becomes current."
		currentLabel = "Current " + m.activeSemanticReviewArtifactName()
		stillCurrentDetail = "Choose this when the current LINER.md and Project Skill still match the refreshed corpus. Liner records approval without rewriting them."
		patchDetail = "Choose this when either artifact needs correction. Edit it, then review the exact Core Change Set before anything is replaced."
	}
	sections := []string{
		styles.Title.Render(title),
		styles.Subtitle.Render(strings.Join(wrapWords(subtitle, width), "\n")),
	}
	if m.synthesisReviewKind == semanticReviewOperatingLayer && m.operatingLayerReviewSkillPath != "" {
		sections = append(sections, "", styles.ReportSection.Render("Artifact"), m.operatingLayerReviewArtifactSelector(width))
	}
	sections = append(sections,
		"",
		styles.ReportSection.Render(currentLabel),
		m.synthesisReviewCurrent.View(),
		styles.SoftText.Render(strings.Join(wrapWords(synthesisReviewPosition(m.synthesisReviewCurrent), width), "\n")),
		"",
		styles.ReportSection.Render("Decision"),
		synthesisDispositionOption(width, stillCurrentLabel, stillCurrentDetail, m.synthesisReviewChoice == synthesisReviewStillCurrent),
		synthesisDispositionOption(width, patchLabel, patchDetail, m.synthesisReviewChoice == synthesisReviewPatch),
	)
	if m.synthesisReviewChoice == synthesisReviewPatch && (m.synthesisReviewEditing || m.semanticReviewHasLocalChanges()) {
		label := "Proposed " + m.activeSemanticReviewArtifactName() + " revision"
		if m.synthesisReviewKind == semanticReviewOperatingLayer && m.operatingLayerReviewArtifact == operatingLayerReviewSkill {
			label += " · " + m.operatingLayerReviewSkillPath
		}
		if m.synthesisReviewEditing {
			label += " · editing"
		}
		sections = append(sections, "", styles.ReportSection.Render(label), m.activeSemanticReviewArea().View())
	} else if m.synthesisReviewChoice == synthesisReviewPatch {
		instruction := "No local edit yet. Press Enter or e to edit the " + m.activeSemanticReviewArtifactName() + ". Ctrl+D finishes editing; Enter then previews the exact Core Change Set."
		sections = append(sections, "", styles.ReportSection.Render("Next step"), styles.SoftText.Render(strings.Join(wrapWords(instruction, width), "\n")))
	}
	if m.synthesisReviewLoading {
		sections = append(sections, "", styles.Section.Render("Liner Core request in progress…"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) operatingLayerReviewArtifactSelector(width int) string {
	linerMarker := "○ "
	linerStyle := styles.MutedText
	skillMarker := "○ "
	skillStyle := styles.MutedText
	if m.operatingLayerReviewArtifact == operatingLayerReviewSkill {
		skillMarker = "● "
		skillStyle = styles.AccentText
	} else {
		linerMarker = "● "
		linerStyle = styles.AccentText
	}
	linerOption := linerStyle.Render(linerMarker + "LINER.md")
	skillOption := skillStyle.Render(skillMarker + "Project Skill (" + m.operatingLayerReviewSkillPath + ")")
	selector := linerOption + "  " + skillOption
	if lipgloss.Width(selector) > width {
		selector = lipgloss.JoinVertical(lipgloss.Left, linerOption, skillOption)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		selector,
		styles.SoftText.Render("Tab switches the artifact being reviewed or edited."),
	)
}

func (m *Model) syncSynthesisReviewPlanView() {
	if m.synthesisReviewPlan == nil {
		m.synthesisReviewPlanView.SetContent("")
		return
	}
	width := max(20, styles.ClampWidth(m.width-8))
	plan := *m.synthesisReviewPlan
	disposition := "still_current"
	if len(plan.Operations) == 1 {
		disposition = fmt.Sprint(plan.Operations[0]["disposition"])
	}
	content := semanticReviewApprovalSummary(m.synthesisReviewKind, disposition, width)
	if len(plan.Operations) == 1 && plan.Operations[0]["disposition"] == "patch" {
		content = appendSemanticReviewProposal(content, plan.Operations[0], "content", "Replacement synthesis", width)
		content = appendSemanticReviewProposal(content, plan.Operations[0], "liner_content", "Replacement LINER.md", width)
		skillLabel := "Replacement Project Skill"
		if skillPath, ok := plan.Operations[0]["skill_path"].(string); ok && strings.TrimSpace(skillPath) != "" {
			skillLabel += " · " + skillPath
		}
		content = appendSemanticReviewProposal(content, plan.Operations[0], "skill_content", skillLabel, width)
	}
	m.synthesisReviewPlanView.SetContent(content)
	maxHeight := max(1, m.synthesisReviewPlanView.Height())
	m.synthesisReviewPlanView.SetHeight(min(maxHeight, max(1, m.synthesisReviewPlanView.TotalLineCount())))
	m.synthesisReviewPlanView.GotoTop()
}

func semanticReviewApprovalSummary(kind semanticReviewKind, disposition string, width int) string {
	artifact := "Synthesis"
	next := "Compile MIXTAPE.md"
	textEffect := "Unchanged"
	if disposition == "patch" {
		textEffect = "Replace with reviewed text below"
	}
	if kind == semanticReviewOperatingLayer {
		artifact = "Operating Layer"
		next = "Finish Project refresh"
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Ready to approve"),
		styles.AccentText.Render(strings.Join(wrapWords("Press Enter to approve the current "+artifact+" and continue.", width), "\n")),
		"",
		newMetadataTable(width, []metadataTableRow{
			{Field: "Text", Value: textEffect},
			{Field: "Records", Value: "Curator approval"},
			{Field: "Next", Value: next},
		}).View(),
	)
}

func semanticReviewChangeSetDetails(width int, plan core.ProjectChangeSet) string {
	lines := core.MaintenancePreviewLines(plan)
	for index, line := range lines {
		lines[index] = strings.Join(wrapSynthesisReviewLine(line, width), "\n")
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Core Change Set details"),
		styles.SoftText.Render("Approval checkpoint. Enter confirms exactly this Change Set."),
		styles.Subtitle.Render(strings.Join(lines, "\n")),
	)
}

func semanticReviewDetailsPlan(plan core.ProjectChangeSet) core.ProjectChangeSet {
	details := plan
	details.Operations = make([]map[string]any, len(plan.Operations))
	for index, operation := range plan.Operations {
		summary := make(map[string]any, len(operation))
		for key, value := range operation {
			switch key {
			case "content", "liner_content", "skill_content":
				continue
			default:
				summary[key] = value
			}
		}
		details.Operations[index] = summary
	}
	return details
}

func appendSemanticReviewProposal(content string, operation map[string]any, keyName string, label string, width int) string {
	proposed, ok := operation[keyName].(string)
	if !ok {
		return content
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		styles.ReportSection.Render(label),
		wrapSynthesisReviewContent(proposed, width),
	)
}

func wrapSynthesisReviewContent(content string, width int) string {
	lines := strings.Split(content, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapSynthesisReviewLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapSynthesisReviewLine(line string, width int) []string {
	if line == "" || width <= 0 {
		return []string{line}
	}
	characters := []rune(line)
	segments := make([]string, 0, 1)
	for start := 0; start < len(characters); {
		end := start
		lastBreak := -1
		cellWidth := 0
		for end < len(characters) {
			characterWidth := lipgloss.Width(string(characters[end]))
			if end > start && cellWidth+characterWidth > width {
				break
			}
			cellWidth += characterWidth
			end++
			if unicode.IsSpace(characters[end-1]) {
				lastBreak = end
			}
		}
		if end < len(characters) && lastBreak > start {
			end = lastBreak
		}
		segments = append(segments, string(characters[start:end]))
		start = end
	}
	return segments
}

func synthesisReviewPosition(viewport viewport.Model) string {
	return semanticReviewViewportPosition(viewport, "Lines", "synthesis")
}

func synthesisReviewPlanPosition(viewport viewport.Model) string {
	return semanticReviewViewportPosition(viewport, "Preview lines", "preview")
}

func semanticReviewViewportPosition(viewport viewport.Model, prefix string, documentName string) string {
	total := viewport.TotalLineCount()
	if total == 0 {
		return "No " + documentName + " content."
	}
	start := min(total, viewport.YOffset()+1)
	end := min(total, viewport.YOffset()+viewport.VisibleLineCount())
	position := "complete"
	switch {
	case !viewport.AtTop() && !viewport.AtBottom():
		position = "more above and below"
	case !viewport.AtTop():
		position = "end of " + documentName + "; more above"
	case !viewport.AtBottom():
		position = "more below"
	}
	return fmt.Sprintf("%s %d–%d of %d · %s · ↑/↓ one line · PgUp/PgDown one page", prefix, start, end, total, position)
}

func synthesisDispositionOption(width int, label string, detail string, selected bool) string {
	marker := "○ "
	labelStyle := styles.MutedText
	if selected {
		marker = "● "
		labelStyle = styles.AccentText
	}
	return lipgloss.JoinVertical(lipgloss.Left, labelStyle.Render(marker+label), styles.SoftText.Render(strings.Join(wrapWords(detail, width), "\n")))
}
