package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func fallbackText(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func titleASCII(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func markdownMetadataValue(body string, label string) (string, bool) {
	prefix := strings.ToLower(strings.TrimSpace(label)) + ":"
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(line)
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "-"))
		if !strings.HasPrefix(strings.ToLower(clean), prefix) {
			continue
		}
		value := strings.TrimSpace(clean[len(prefix):])
		if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") && strings.Count(value, "`") == 2 {
			value = strings.Trim(value, "`")
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func truncateMiddle(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	left := max(1, (width-1)/2)
	right := max(1, width-left-1)
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if left+right >= len(runes) {
		return value
	}
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

func wrapWords(value string, width int) []string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return []string{""}
	}
	if width <= 0 {
		return []string{value}
	}
	var lines []string
	for _, word := range strings.Fields(value) {
		wordParts := splitWordToWidth(word, width)
		for _, part := range wordParts {
			if len(lines) == 0 {
				lines = append(lines, part)
				continue
			}
			current := lines[len(lines)-1]
			if lipgloss.Width(current)+1+lipgloss.Width(part) <= width {
				lines[len(lines)-1] = current + " " + part
				continue
			}
			lines = append(lines, part)
		}
	}
	return lines
}

func splitWordToWidth(word string, width int) []string {
	if width <= 0 || lipgloss.Width(word) <= width {
		return []string{word}
	}
	parts := []string{}
	var current strings.Builder
	for _, r := range word {
		candidate := current.String() + string(r)
		if current.Len() > 0 && lipgloss.Width(candidate) > width {
			parts = append(parts, current.String())
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func padLines(value string, height int) string {
	current := lipgloss.Height(value)
	if current >= height {
		return value
	}
	return value + strings.Repeat("\n", height-current)
}

func sourceInputWidth(width int) int {
	if width <= 0 {
		return 76
	}
	return max(8, styles.ClampWidth(width-4)-lipgloss.Width("> ")-6)
}

func slug(value string) string {
	value = strings.ToLower(value)
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func tail[T any](values []T, n int) []T {
	if len(values) <= n {
		return values
	}
	return values[len(values)-n:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
