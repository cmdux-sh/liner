package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type compositionLinerSignal struct {
	Kind     string
	Evidence string
}

type compositionLinerBlendInput struct {
	RouteInput   compositionRouteAuditInput
	ChildProject string
	LinerRel     string
	LinerBytes   int64
	Signals      []compositionLinerSignal
}

func writeCompositionLinerBlendDraft(project string, parent string, item compositionFile) (string, error) {
	input, err := compositionPromotionInput(project, item)
	if err != nil {
		return "", err
	}
	childProject, referenceIssue := compositionChildProjectReference(project, item)
	if referenceIssue != "" {
		return "", fmt.Errorf("cannot blend child LINER.md: %s", referenceIssue)
	}
	blendInput, err := compositionLinerBlendInputForChild(input, childProject)
	if err != nil {
		return "", err
	}
	now := time.Now()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(project, compositionDraftRelPath), []byte(renderCompositionLinerBlendDraft(parent, blendInput)), 0o644); err != nil {
		return "", err
	}
	auditRel := filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-liner-blend.md")
	if err := os.WriteFile(filepath.Join(project, auditRel), []byte(renderCompositionLinerBlendAudit(now, blendInput)), 0o644); err != nil {
		return "", err
	}
	return auditRel, nil
}

func compositionLinerBlendInputForChild(input compositionRouteAuditInput, childProject string) (compositionLinerBlendInput, error) {
	linerRel := "LINER.md"
	linerPath := filepath.Join(childProject, linerRel)
	info, err := os.Stat(linerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return compositionLinerBlendInput{}, fmt.Errorf("cannot blend child LINER.md: %s is missing in the referenced child project", linerRel)
		}
		return compositionLinerBlendInput{}, fmt.Errorf("cannot inspect child LINER.md: %w", err)
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return compositionLinerBlendInput{}, fmt.Errorf("cannot blend child LINER.md: %s is not a regular file", linerRel)
	}
	body, err := os.ReadFile(linerPath)
	if err != nil {
		return compositionLinerBlendInput{}, fmt.Errorf("cannot read child LINER.md: %w", err)
	}
	signals := compositionLinerSignals(string(body))
	if len(signals) == 0 {
		signals = append(signals, compositionLinerSignal{
			Kind:     "review",
			Evidence: "Child LINER.md did not expose obvious headings or rule-like lines; review the file manually before applying.",
		})
	}
	return compositionLinerBlendInput{
		RouteInput:   input,
		ChildProject: childProject,
		LinerRel:     linerRel,
		LinerBytes:   info.Size(),
		Signals:      signals,
	}, nil
}

func compositionLinerSignals(body string) []compositionLinerSignal {
	signals := []compositionLinerSignal{}
	seen := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(line)
		if clean == "" {
			continue
		}
		kind := ""
		switch {
		case strings.HasPrefix(clean, "#"):
			kind = "section"
		case compositionLinerRuleLine(clean):
			kind = "rule"
		}
		if kind == "" {
			continue
		}
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "-"))
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "*"))
		clean = strings.Join(strings.Fields(clean), " ")
		if clean == "" || seen[strings.ToLower(clean)] {
			continue
		}
		seen[strings.ToLower(clean)] = true
		signals = append(signals, compositionLinerSignal{
			Kind:     kind,
			Evidence: truncateForTable(clean, 180),
		})
		if len(signals) >= 10 {
			break
		}
	}
	return signals
}

func compositionLinerRuleLine(line string) bool {
	lower := strings.ToLower(line)
	return containsAny(lower, []string{
		"always", "avoid", "boundary", "boundaries", "cannot", "delegate", "do not", "don't",
		"must", "never", "prefer", "route", "scope", "should", "unless", "when ",
	})
}

func renderCompositionLinerBlendDraft(parent string, input compositionLinerBlendInput) string {
	route := input.RouteInput.Route
	if !input.RouteInput.Explicit {
		route += " (inferred)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Child LINER Blend Draft\n\n", parent)
	b.WriteString("Review this section before merging it into `LINER.md`. It blends one child's operating layer into the parent routing rules by reference; child sources, skills, and files stay in the child project.\n\n")
	b.WriteString("## Selected Child\n\n")
	b.WriteString("| Child | Route | Status | Reference | Child Project | Child LINER.md |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| %s | %s | %s | `%s` | `%s` | `%s` (%d bytes) |\n",
		escapeMarkdownTable(input.RouteInput.Child),
		escapeMarkdownTable(route),
		escapeMarkdownTable(input.RouteInput.Status),
		input.RouteInput.RelPath,
		escapeMarkdownTable(input.ChildProject),
		input.LinerRel,
		input.LinerBytes,
	)
	b.WriteString("\n## Child Operating Signals\n\n")
	b.WriteString("| Kind | Evidence |\n")
	b.WriteString("| --- | --- |\n")
	for _, signal := range input.Signals {
		fmt.Fprintf(&b, "| %s | %s |\n",
			escapeMarkdownTable(signal.Kind),
			escapeMarkdownTable(signal.Evidence),
		)
	}
	b.WriteString("\n## Parent Routing Update\n\n")
	fmt.Fprintf(&b, "- When a request matches `%s`, load the parent `LINER.md` first, then load `%s` from `%s` before using child skills or corpus files.\n", input.RouteInput.Route, input.LinerRel, input.RouteInput.Child)
	b.WriteString("- Treat child operating rules as scoped to that child route; they do not override parent-wide safety, abstention, or source-use rules unless a reviewed audit says so.\n")
	b.WriteString("- If a child rule conflicts with a parent rule or another child, pause and record the conflict in an audit instead of silently choosing one.\n")
	b.WriteString("- Do not copy child source claims, skills, or tape sources through this blend draft; use the production merge action for those artifacts.\n\n")
	b.WriteString("## Review Notes\n\n")
	if len(input.RouteInput.ReviewLines) == 0 {
		b.WriteString("- No warning, overlap, conflict, disabled, or deprecated language was found in the child reference file.\n")
	} else {
		for _, line := range input.RouteInput.ReviewLines {
			fmt.Fprintf(&b, "- %s\n", escapeMarkdownTable(line))
		}
	}
	return b.String()
}

func renderCompositionLinerBlendAudit(now time.Time, input compositionLinerBlendInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition LINER Blend Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Created a reviewed child `LINER.md` blend draft for one selected child. This prepares a parent `LINER.md` routing update without changing parent or child operating-layer files yet.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Draft: `%s`\n", compositionDraftRelPath)
	fmt.Fprintf(&b, "- Child reference: `%s`\n", input.RouteInput.RelPath)
	fmt.Fprintf(&b, "- Child project: `%s`\n", input.ChildProject)
	fmt.Fprintf(&b, "- Child LINER.md: `%s` (%d bytes)\n", input.LinerRel, input.LinerBytes)
	fmt.Fprintf(&b, "- Extracted operating signals: %d\n", len(input.Signals))
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- Parent `LINER.md` is unchanged until the draft is accepted in Review Composition Draft.\n")
	b.WriteString("- Child `LINER.md`, child sources, child skills, and child tape files were not changed.\n")
	b.WriteString("- This action only prepares routing guidance; production source and skill movement stays behind the explicit merge action.\n")
	return b.String()
}
