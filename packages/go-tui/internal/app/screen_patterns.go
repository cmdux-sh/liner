package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

type labelValueRow struct {
	Label    string
	Value    string
	MaxLines int
}

type choiceOption struct {
	Label  string
	Detail string
}

func (m Model) renderLoadingTitle(title string, loading bool) string {
	return renderLoadingTitle(title, loading, m.loadingTitleSpinnerView())
}

func (m Model) loadingTitleSpinnerView() string {
	for _, spinnerView := range []string{
		m.researchSpin.View(),
		m.compileSpin.View(),
		m.clarifySpin.View(),
		m.operatingLayerSpin.View(),
	} {
		if cleaned := cleanSpinnerView(spinnerView); cleaned != "" {
			return cleaned
		}
	}
	return ""
}

func renderLoadingTitle(title string, loading bool, spinnerView string) string {
	rendered := styles.Title.Render(title)
	if !loading {
		return rendered
	}
	spinnerView = cleanSpinnerView(spinnerView)
	if spinnerView == "" {
		return rendered
	}
	return rendered + " " + spinnerView
}

func cleanSpinnerView(spinnerView string) string {
	spinnerView = strings.TrimSpace(spinnerView)
	if spinnerView == "" || spinnerView == "(error)" {
		return ""
	}
	return spinnerView
}

func renderChoiceSelector(options []choiceOption, cursor int) string {
	if len(options) == 0 {
		return ""
	}
	selected := selectedChoiceIndex(options, cursor)
	parts := make([]string, 0, len(options))
	for index, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = "Option"
		}
		if index == selected {
			parts = append(parts, styles.AccentText.Render(label))
			continue
		}
		parts = append(parts, styles.MutedText.Render(label))
	}
	return strings.Join(parts, styles.MutedText.Render("    "))
}

func renderChoiceDetail(width int, options []choiceOption, cursor int) string {
	if len(options) == 0 {
		return ""
	}
	detail := strings.TrimSpace(options[selectedChoiceIndex(options, cursor)].Detail)
	if detail == "" {
		return ""
	}
	return "\n" + styles.Subtitle.Width(width).Render(detail) + "\n"
}

func selectedChoiceIndex(options []choiceOption, cursor int) int {
	if len(options) == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= len(options) {
		return len(options) - 1
	}
	return cursor
}

func renderLabelValueBlock(width int, rows []labelValueRow, topPadding int, bottomPadding int) string {
	labelWidth := labelValueMaxLabelWidth(rows)
	if labelWidth <= 0 {
		labelWidth = len("Status:")
	}
	lines := make([]string, 0, len(rows)+topPadding+bottomPadding)
	for range topPadding {
		lines = append(lines, "")
	}
	for _, row := range rows {
		lines = append(lines, renderLabelValueRow(width, labelWidth, row)...)
	}
	for range bottomPadding {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func renderProgressStatusBlock(width int, bar progress.Model, percent float64, status string, detail string, count string) string {
	if strings.TrimSpace(status) == "" {
		status = "Working"
	}
	if strings.TrimSpace(count) == "" {
		count = fmt.Sprintf("%d%%", int(percent*100))
	}
	statusLine := styles.ReportSection.Render(status)
	if strings.TrimSpace(detail) != "" {
		statusLine += "  " + styles.NextActionText.Render(detail)
	}
	return lipgloss.NewStyle().Width(width).Render(
		statusLine + "\n" +
			bar.ViewAs(percent) + "  " + styles.Subtitle.Render(count),
	)
}

func renderWaitStatusBlock(width int, status string, detail string, count string) string {
	return renderActiveWaitStatusBlock(width, status, detail, count, "")
}

func renderActiveWaitStatusBlock(width int, status string, detail string, count string, spinnerView string) string {
	if strings.TrimSpace(status) == "" {
		status = "Working"
	}
	statusLine := styles.ReportSection.Render(status)
	if strings.TrimSpace(detail) != "" {
		statusLine += "  " + styles.NextActionText.Render(detail)
	}
	lines := []string{statusLine}
	if strings.TrimSpace(count) != "" {
		countLine := styles.Subtitle.Render(count)
		if activity := cleanSpinnerView(spinnerView); activity != "" {
			countLine = activity + " " + countLine
		}
		lines = append(lines, countLine)
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func newTaskProgressBar(width int) progress.Model {
	return progress.New(
		progress.WithWidth(width),
		progress.WithoutPercentage(),
		progress.WithColors(styles.ProgressTrack, styles.ProgressFill),
	)
}

func taskProgressWidth(width int) int {
	if width <= 0 {
		return 48
	}
	return max(20, min(64, styles.ClampWidth(width-4)-18))
}

func labelValueMaxLabelWidth(rows []labelValueRow) int {
	width := 0
	for _, row := range rows {
		label := strings.TrimSpace(row.Label)
		if label == "" {
			continue
		}
		width = max(width, lipgloss.Width(label+":"))
	}
	return width
}

func renderLabelValueRow(width int, labelWidth int, row labelValueRow) []string {
	label := strings.TrimSpace(row.Label)
	if label == "" {
		label = "Item"
	}
	labelText := label + ":"
	if padding := labelWidth - lipgloss.Width(labelText); padding > 0 {
		labelText += strings.Repeat(" ", padding)
	}
	valueWidth := max(10, width-labelWidth-1)
	valueLines := wrapLabelValue(row.Value, valueWidth)
	valueLines = clampWrappedLines(valueLines, valueWidth, row.MaxLines)
	if len(valueLines) == 0 {
		valueLines = []string{""}
	}
	out := []string{
		styles.MutedText.Render(labelText) + " " + styles.PrimaryText.Render(valueLines[0]),
	}
	indent := strings.Repeat(" ", labelWidth+1)
	for _, line := range valueLines[1:] {
		out = append(out, indent+styles.PrimaryText.Render(line))
	}
	return out
}

func clampWrappedLines(lines []string, width int, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	clamped := append([]string(nil), lines[:maxLines]...)
	last := strings.TrimSpace(clamped[len(clamped)-1])
	if width <= 1 {
		clamped[len(clamped)-1] = "…"
		return clamped
	}
	if lipgloss.Width(last)+lipgloss.Width("…") <= width {
		clamped[len(clamped)-1] = last + "…"
		return clamped
	}
	prefix, _ := splitDisplayWidth(last, width-lipgloss.Width("…"))
	clamped[len(clamped)-1] = strings.TrimRight(prefix, " ") + "…"
	return clamped
}

func wrapLabelValue(value string, width int) []string {
	value = strings.Join(strings.Fields(fallbackText(value, "none")), " ")
	if value == "" {
		return []string{""}
	}
	if width <= 0 {
		return []string{value}
	}
	lines := []string{}
	for _, word := range strings.Fields(value) {
		for lipgloss.Width(word) > width {
			part, rest := splitDisplayWidth(word, width)
			if part == "" {
				break
			}
			if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) != "" {
				lines = append(lines, part)
			} else {
				lines[len(lines)-1] = part
			}
			word = rest
		}
		if word == "" {
			continue
		}
		if len(lines) == 0 {
			lines = append(lines, word)
			continue
		}
		current := lines[len(lines)-1]
		if lipgloss.Width(current)+1+lipgloss.Width(word) <= width {
			lines[len(lines)-1] = current + " " + word
			continue
		}
		lines = append(lines, word)
	}
	return lines
}

func splitDisplayWidth(value string, width int) (string, string) {
	if width <= 0 || value == "" {
		return "", value
	}
	var left strings.Builder
	used := 0
	boundaryIndex := -1
	boundaryWidth := 0
	for index, r := range value {
		next := lipgloss.Width(string(r))
		if used > 0 && used+next > width {
			if boundaryIndex > 0 && boundaryWidth >= min(8, max(1, width/2)) {
				return value[:boundaryIndex], value[boundaryIndex:]
			}
			return left.String(), value[index:]
		}
		left.WriteRune(r)
		used += next
		if r == '/' || r == '\\' {
			boundaryIndex = index + len(string(r))
			boundaryWidth = used
		}
	}
	return left.String(), ""
}
