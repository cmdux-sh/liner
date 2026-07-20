package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	liplist "charm.land/lipgloss/v2/list"

	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const fullMethodologyAction = "Build the corpus before compiling."

func (m Model) viewReport() string {
	reportWidth := styles.ClampWidth(m.width - 8)
	report := renderReportBody(m.currentTape, m.sourceItems, reportWidth)
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Title.Render("Research Report"),
		"",
		lipgloss.NewStyle().Width(styles.ClampWidth(m.width-4)).PaddingLeft(2).Render(report),
	)
}

func reportSummary(current tape.Tape, items []source.StagedSource) string {
	return strings.Join(reportSummaryLines(current, items, false), "\n")
}

func renderReportBody(current tape.Tape, items []source.StagedSource, width int) string {
	return strings.Join(reportBodyLines(current, items, true, width), "\n")
}

func renderReportNextAction(current tape.Tape, items []source.StagedSource) string {
	return renderNextCue(reportNextAction(current, items))
}

func reportNextAction(current tape.Tape, items []source.StagedSource) string {
	return fullMethodologyAction
}

const projectCompleteNextAction = "Open LINER.md."

func renderNextCue(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return ""
	}
	return styles.NextCueTitle.Render("> Next:") + " " + styles.NextActionText.Render(action)
}

func reportSummaryLines(current tape.Tape, items []source.StagedSource, styled bool) []string {
	lines := reportBodyLines(current, items, styled, 0)
	lines = append(lines, "")
	lines = append(lines,
		sectionLabel("Next", styled),
		fullMethodologyAction,
	)
	return lines
}

func reportBodyLines(current tape.Tape, items []source.StagedSource, styled bool, width int) []string {
	sourceCount, active, hasSources := reportSourceState(current, items)
	jtbd := "Not set."
	if current.JTBD != nil && strings.TrimSpace(*current.JTBD) != "" {
		jtbd = strings.TrimSpace(*current.JTBD)
	}
	lines := []string{}
	if hasSources {
		lines = append(lines,
			sectionLabel("User-Provided Sources", styled),
			fmt.Sprintf("Added: %d", sourceCount),
			fmt.Sprintf("Active: %d", active),
			fmt.Sprintf("Inactive: %d", sourceCount-active),
			"",
		)
	}
	lines = append(lines,
		sectionLabel("Job to Be Done", styled),
	)
	lines = append(lines, reportParagraphLines(jtbd, styled, width)...)
	if len(current.JTBDClarifications) > 0 {
		lines = append(lines, "", sectionLabel("Clarify Job", styled))
		for _, item := range current.JTBDClarifications {
			answer := strings.TrimSpace(item.Answer)
			if answer == "" {
				continue
			}
			lines = append(lines, reportClarificationLines(item.Question, answer, styled, width)...)
		}
	}
	lines = append(lines,
		"",
		sectionLabel("What happened", styled),
	)
	happened := []string{
		"Setup context was loaded.",
		"The Job to Be Done was translated into a capability brief and research plan.",
	}
	if hasSources {
		happened = append(happened,
			"User-Provided Sources were classified and stored locally.",
			"User-Provided Sources will be used as inputs during candidate discovery and evaluation.",
			"The corpus build still needs to run before compile.",
		)
	} else {
		happened = append(happened,
			"The setup answers were used for this first pass.",
			"The corpus build still needs to run before compile.",
		)
	}
	lines = append(lines, reportListLines(happened, styled, width, true)...)
	return lines
}

func sectionLabel(label string, styled bool) string {
	if styled {
		return styles.ReportSection.Render(label)
	}
	return label
}

func reportParagraphLines(value string, styled bool, width int) []string {
	if !styled || width <= 0 {
		return []string{value}
	}
	lines := wrapWords(value, width)
	for i, line := range lines {
		lines[i] = styles.ReportBody.Render(line)
	}
	return lines
}

func reportClarificationLines(question string, answer string, styled bool, width int) []string {
	question = strings.TrimSpace(question)
	answer = strings.TrimSpace(answer)
	if question == "" || answer == "" {
		return nil
	}
	if !styled {
		return []string{"Question: " + question, "Answer: " + answer, ""}
	}
	bodyWidth := max(24, width-4)
	var lines []string
	for i, line := range wrapWords(question, bodyWidth) {
		prefix := styles.ReportListAccent.Render("Q ")
		if i > 0 {
			prefix = "  "
		}
		lines = append(lines, prefix+styles.ReportBody.Render(line))
	}
	for i, line := range wrapWords(answer, bodyWidth) {
		prefix := styles.ReportListAccent.Render("A ")
		if i > 0 {
			prefix = "  "
		}
		lines = append(lines, prefix+styles.ReportBody.Render(line))
	}
	lines = append(lines, "")
	return lines
}

func reportListLines(items []string, styled bool, width int, checked bool) []string {
	if len(items) == 0 {
		return nil
	}
	if !styled {
		lines := make([]string, 0, len(items))
		for _, item := range items {
			lines = append(lines, "- "+item)
		}
		return lines
	}
	itemWidth := width - 4
	if itemWidth <= 0 {
		itemWidth = 92
	}
	itemWidth = max(20, itemWidth)
	renderItems := make([]any, 0, len(items))
	for _, item := range items {
		renderItems = append(renderItems, strings.Join(wrapWords(item, itemWidth), "\n  "))
	}
	enumerator := liplist.Bullet
	enumeratorStyle := styles.ReportListAccent
	if checked {
		enumerator = func(_ liplist.Items, _ int) string { return "✓ " }
		enumeratorStyle = styles.ReportListMarker
	}
	rendered := liplist.New(renderItems...).
		Enumerator(enumerator).
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(styles.ReportBody).
		String()
	return strings.Split(rendered, "\n")
}

func reportSourceState(current tape.Tape, items []source.StagedSource) (int, int, bool) {
	if len(items) == 0 && len(current.Sources) > 0 {
		return len(current.Sources), len(current.Sources), true
	}
	active := 0
	for _, item := range items {
		if item.Active {
			active++
		}
	}
	return len(items), active, len(items) > 0
}
