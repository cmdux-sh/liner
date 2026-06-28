package app

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

const (
	operatingFitAuditRelPath = "working/05-operating-fit-audit.md"
	qualityChecksRelPath     = "working/04-quality-checks.md"
)

const (
	improvementOptionRun int = iota
	improvementOptionSkip
)

var improvementOptions = []choiceOption{
	{
		Label:  "Improve now",
		Detail: "Run a focused improvement pass. Liner will use the quality notes to search for missing source roles, evaluate the new sources, and refresh the corpus before the Operating Layer.",
	},
	{
		Label:  "Skip",
		Detail: "Skip for now. Liner keeps the improvement notes, continues to the Operating Layer, and will offer this pass again if you run Compile for this project.",
	},
}

type operatingFitAudit struct {
	RelPath string
	Body    string
	Summary string
}

func improvementOptionAt(cursor int) int {
	if cursor == improvementOptionSkip {
		return improvementOptionSkip
	}
	return improvementOptionRun
}

func (m Model) startImprovementReview() Model {
	m.screen = screenImprovementReview
	m.improvementCursor = improvementOptionRun
	m.note = ""
	m.err = ""
	return m
}

func (m Model) handleImprovementReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "esc":
		m.screen = screenCompile
		m.err = ""
		return m, nil
	case "enter":
		return m.applyImprovementOption()
	case "shift+tab", "up", "left":
		return m.moveImprovementOption(-1), nil
	case "tab", "down", "right":
		return m.moveImprovementOption(1), nil
	case "p":
		if audit, ok := operatingFitImprovementAudit(m.currentPath); ok {
			return m.openPreview(audit.RelPath)
		}
		m.err = "No improvement notes are available to preview."
		return m, nil
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		m.screen = screenCompile
		m.err = ""
		return m, nil
	case tea.KeyLeft, tea.KeyUp:
		return m.moveImprovementOption(-1), nil
	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		return m.moveImprovementOption(1), nil
	case tea.KeyEnter:
		return m.applyImprovementOption()
	}
	return m, nil
}

func (m Model) moveImprovementOption(delta int) Model {
	if len(improvementOptions) == 0 {
		return m
	}
	m.improvementCursor = (selectedChoiceIndex(improvementOptions, m.improvementCursor) + delta + len(improvementOptions)) % len(improvementOptions)
	m.err = ""
	return m
}

func (m Model) applyImprovementOption() (Model, tea.Cmd) {
	switch improvementOptionAt(m.improvementCursor) {
	case improvementOptionSkip:
		next, cmd := m.startLinerDraftReview()
		next.note = "Skipped improvement pass for now. Notes remain in " + operatingFitAuditRelPath + "."
		return next, cmd
	default:
		return m.startImprovementPass()
	}
}

func (m Model) startImprovementPass() (Model, tea.Cmd) {
	index, ok := methodologyIndexForProgressPhase(progress.PhaseCandidates)
	if !ok {
		m.err = "Candidate discovery is not available for improvement."
		return m, nil
	}
	m.stopMethodology("")
	m.screen = screenResearch
	m.researchDone = false
	m.methodologyFailed = false
	m.methodologyLastErr = ""
	m.methodologyPhaseID = ""
	m.methodologyEventCount = 0
	m.methodologyLastEventFrame = m.fxFrame
	m.ensureBoardItems()
	m.note = ""
	m.err = ""
	m.researchStep = index
	m.methodologyPhaseIndex = index
	m.researchLines = []string{
		"Starting Improvement Pass...",
		"Using " + operatingFitAuditRelPath + " to target missing source roles.",
		"Queued Candidate discovery through Assembly.",
	}
	m.syncMethodologyLog(true)
	return m.startMethodologyPhase(index, false)
}

func (m Model) viewImprovementReview() string {
	width := styles.ClampWidth(m.width - 4)
	body := "Quality checks found a source-role gap before this corpus becomes operating guidance. You can let Liner run a focused second pass now, or skip for now and continue to the Operating Layer."
	parts := []string{
		styles.Title.Render("Improve Corpus"),
		styles.Subtitle.Render(m.currentTape.Title),
		"",
		styles.PrimaryText.Render(strings.Join(wrapWords(body, width), "\n")),
	}
	if audit, ok := operatingFitImprovementAudit(m.currentPath); ok {
		parts = append(parts, "", renderImprovementAuditSummary(width, m.currentPath, audit))
	}
	parts = append(parts,
		"",
		renderChoiceSelector(improvementOptions, m.improvementCursor),
		renderChoiceDetail(width, improvementOptions, m.improvementCursor),
	)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func renderImprovementAuditSummary(width int, project string, audit operatingFitAudit) string {
	rows := []labelValueRow{
		{Label: "Notes", Value: displayProjectPath(project, audit.RelPath)},
	}
	if audit.Summary != "" {
		rows = append(rows, labelValueRow{Label: "Gap", Value: audit.Summary})
	}
	return renderLabelValueBlock(width, rows, 0, 0)
}

func (m Model) improvementNextAction() string {
	if improvementOptionAt(m.improvementCursor) == improvementOptionSkip {
		return "Skip for now and continue to the Operating Layer."
	}
	return "Run the improvement pass before creating the Operating Layer."
}

func operatingFitImprovementRecommended(project string) bool {
	_, ok := operatingFitImprovementAudit(project)
	return ok
}

func operatingFitImprovementAudit(project string) (operatingFitAudit, bool) {
	for _, rel := range []string{operatingFitAuditRelPath, qualityChecksRelPath} {
		body, err := os.ReadFile(projectAbsPath(project, rel))
		if err != nil {
			continue
		}
		text := string(body)
		if !improvementRecommendationMarker(text) {
			continue
		}
		return operatingFitAudit{
			RelPath: filepath.ToSlash(rel),
			Body:    text,
			Summary: improvementAuditSummary(text),
		}, true
	}
	return operatingFitAudit{}, false
}

func improvementRecommendationMarker(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "status: improvement_recommended") ||
		strings.Contains(lower, "status: improvement recommended") ||
		strings.Contains(lower, "improvement recommended") ||
		strings.Contains(lower, "recommended improvement pass")
}

func improvementAuditSummary(body string) string {
	for _, prefix := range []string{"gap:", "why_it_matters:", "why it matters:", "recommendation:", "recommended pass:"} {
		if value := firstMarkdownValue(body, prefix); value != "" {
			return value
		}
	}
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(strings.TrimLeft(line, "-* "))
		if strings.Contains(strings.ToLower(clean), "source-role") || strings.Contains(strings.ToLower(clean), "source role") {
			return clean
		}
	}
	return ""
}

func firstMarkdownValue(body string, prefix string) string {
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(strings.TrimLeft(line, "-* "))
		if !strings.HasPrefix(strings.ToLower(clean), prefix) {
			continue
		}
		value := strings.TrimSpace(clean[len(prefix):])
		value = strings.Trim(value, "`")
		if value != "" {
			return value
		}
	}
	return ""
}
