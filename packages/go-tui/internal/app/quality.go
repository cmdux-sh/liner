package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

var qualityKindPattern = regexp.MustCompile(`(?i)(\d+)\s+(reference|principle|prescription|example)s?\b`)

type qualityKindBalance struct {
	Counts       map[string]int
	ZeroDefended bool
}

type qualityPerspective struct {
	Name     string
	Status   string
	Evidence string
}

var qualityKindRoles = []string{"reference", "principle", "prescription", "example"}

func readQualityKindBalance(project string) (qualityKindBalance, bool) {
	if strings.TrimSpace(project) == "" {
		return qualityKindBalance{}, false
	}
	data, err := os.ReadFile(filepath.Join(project, "working", "04-quality-checks.md"))
	if err != nil {
		return qualityKindBalance{}, false
	}
	return parseQualityKindBalance(string(data))
}

func parseQualityKindBalance(markdown string) (qualityKindBalance, bool) {
	body := qualityTest5Body(markdown)
	for _, rawLine := range strings.Split(body, "\n") {
		if !strings.Contains(strings.ToLower(rawLine), "distribution:") {
			continue
		}
		counts := make(map[string]int, len(qualityKindRoles))
		for _, match := range qualityKindPattern.FindAllStringSubmatch(rawLine, -1) {
			var count int
			if _, err := fmt.Sscanf(match[1], "%d", &count); err != nil {
				continue
			}
			counts[strings.ToLower(match[2])] = count
		}
		if len(counts) != len(qualityKindRoles) {
			return qualityKindBalance{}, false
		}
		return qualityKindBalance{
			Counts:       counts,
			ZeroDefended: qualityKindZeroDefended(body),
		}, true
	}
	return qualityKindBalance{}, false
}

func readQualityPerspectives(project string) ([]qualityPerspective, bool) {
	if strings.TrimSpace(project) == "" {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(project, "working", "04-quality-checks.md"))
	if err != nil {
		return nil, false
	}
	return parseQualityPerspectives(string(data))
}

func parseQualityPerspectives(markdown string) ([]qualityPerspective, bool) {
	body := qualityPerspectivesBody(markdown)
	if strings.TrimSpace(body) == "" {
		return nil, false
	}
	var perspectives []qualityPerspective
	for _, rawLine := range strings.Split(body, "\n") {
		if strings.HasPrefix(rawLine, "  ") || strings.HasPrefix(rawLine, "\t") {
			continue
		}
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		name, evidence := splitPerspectiveBullet(strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		if name == "" {
			continue
		}
		perspectives = append(perspectives, qualityPerspective{
			Name:     name,
			Status:   qualityPerspectiveStatus(evidence),
			Evidence: evidence,
		})
	}
	return perspectives, len(perspectives) > 0
}

func qualityTest5Body(markdown string) string {
	return qualityHeadingBody(markdown, "## test 5")
}

func qualityPerspectivesBody(markdown string) string {
	if body := qualityHeadingBody(markdown, "### perspectives audit"); body != "" {
		return body
	}
	return qualityHeadingBody(markdown, "## perspectives audit")
}

func qualityHeadingBody(markdown string, headingPrefix string) string {
	lines := strings.Split(markdown, "\n")
	start := -1
	for index, line := range lines {
		normalized := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(normalized, headingPrefix) {
			start = index
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), "## ") {
			end = index
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func splitPerspectiveBullet(value string) (string, string) {
	for _, separator := range []string{" — ", " - "} {
		if parts := strings.SplitN(value, separator, 2); len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(value), ""
}

func qualityPerspectiveStatus(evidence string) string {
	lower := strings.ToLower(evidence)
	switch {
	case strings.Contains(lower, "stance-represented"):
		return "represented"
	case strings.Contains(lower, "concerns-addressed") || strings.Contains(lower, "argued sufficient"):
		return "concerns covered"
	case strings.Contains(lower, "stance-absent") || strings.Contains(lower, "concerns-absent"):
		return styles.WarningText.Render("gap")
	default:
		return "noted"
	}
}

func qualityKindZeroDefended(body string) bool {
	lower := strings.ToLower(body)
	for _, marker := range []string{"accepted:", "defense", "defended", "backfill", "argued sufficient", "recommendation:"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func (b qualityKindBalance) count(role string) int {
	return b.Counts[role]
}

func (b qualityKindBalance) status(role string) string {
	if b.count(role) > 0 {
		return "covered"
	}
	if b.ZeroDefended {
		return "zero defended"
	}
	return styles.WarningText.Render("needs defense")
}

func projectQualityBalanceView(project string, width int) string {
	balance, ok := readQualityKindBalance(project)
	if !ok {
		return ""
	}
	rows := make([]table.Row, 0, len(qualityKindRoles))
	for _, role := range qualityKindRoles {
		rows = append(rows, table.Row{role, fmt.Sprintf("%d", balance.count(role)), balance.status(role)})
	}
	statusWidth := max(18, width-28)
	qualityTable := newDataTable(
		[]table.Column{
			{Title: "Role", Width: 14},
			{Title: "Count", Width: 6},
			{Title: "Status", Width: statusWidth},
		},
		rows,
		width,
		len(rows)+1,
		false,
	)
	return qualityTable.View()
}

func projectQualityPerspectivesView(project string, width int) string {
	perspectives, ok := readQualityPerspectives(project)
	if !ok {
		return ""
	}
	perspectiveWidth := min(30, max(20, width/3))
	statusWidth := min(22, max(16, width/4))
	evidenceWidth := max(20, width-perspectiveWidth-statusWidth-12)
	rows := make([]table.Row, 0, len(perspectives))
	for _, perspective := range perspectives {
		rows = append(rows, table.Row{
			truncateMiddle(perspective.Name, perspectiveWidth),
			truncateMiddle(perspective.Status, statusWidth),
			truncateMiddle(perspective.Evidence, evidenceWidth),
		})
	}
	perspectiveTable := newDataTable(
		[]table.Column{
			{Title: "Perspective", Width: perspectiveWidth},
			{Title: "Coverage", Width: statusWidth},
			{Title: "Evidence", Width: evidenceWidth},
		},
		rows,
		width,
		min(len(rows)+1, 5),
		false,
	)
	return perspectiveTable.View()
}
