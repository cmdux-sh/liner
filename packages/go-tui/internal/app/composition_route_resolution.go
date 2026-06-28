package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type compositionRouteResolutionConflict struct {
	Token    string
	Inputs   []compositionRouteAuditInput
	Children string
	Routes   string
}

func (m Model) createCompositionRouteResolutionDraft() (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot draft route resolution without a project path."
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
	auditRel, err := writeCompositionRouteResolutionDraft(m.currentPath, fallbackText(m.currentTape.Title, filepath.Base(m.currentPath)), items)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	return m.startCompositionDraftReview("Wrote " + compositionDraftRelPath + " and " + auditRel + ". Review before applying.")
}

func writeCompositionRouteResolutionDraft(project string, parent string, items []compositionFile) (string, error) {
	inputs, err := compositionRouteAuditInputs(project, items)
	if err != nil {
		return "", err
	}
	if len(inputs) == 0 {
		return "", fmt.Errorf("Add at least one child reference before drafting route resolution.")
	}
	conflicts := compositionRouteResolutionConflicts(inputs)
	now := time.Now()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(project, compositionDraftRelPath), []byte(renderCompositionRouteResolutionDraft(parent, inputs, conflicts)), 0o644); err != nil {
		return "", err
	}
	auditRel := filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-route-resolution.md")
	if err := os.WriteFile(filepath.Join(project, auditRel), []byte(renderCompositionRouteResolutionAudit(now, inputs, conflicts)), 0o644); err != nil {
		return "", err
	}
	return auditRel, nil
}

func compositionRouteResolutionConflicts(inputs []compositionRouteAuditInput) []compositionRouteResolutionConflict {
	owners := map[string][]compositionRouteAuditInput{}
	for _, input := range inputs {
		for _, token := range compositionRouteTokens(input.Route) {
			owners[token] = append(owners[token], input)
		}
	}
	tokens := make([]string, 0, len(owners))
	for token, inputs := range owners {
		if len(inputs) > 1 {
			tokens = append(tokens, token)
		}
	}
	sort.Strings(tokens)
	conflicts := []compositionRouteResolutionConflict{}
	for _, token := range tokens {
		conflictInputs := owners[token]
		sort.SliceStable(conflictInputs, func(i int, j int) bool {
			return strings.ToLower(conflictInputs[i].Child) < strings.ToLower(conflictInputs[j].Child)
		})
		children := make([]string, 0, len(conflictInputs))
		routes := make([]string, 0, len(conflictInputs))
		for _, input := range conflictInputs {
			children = append(children, input.Child)
			routes = append(routes, input.Child+": "+input.Route)
		}
		conflicts = append(conflicts, compositionRouteResolutionConflict{
			Token:    token,
			Inputs:   conflictInputs,
			Children: strings.Join(children, ", "),
			Routes:   strings.Join(routes, "; "),
		})
	}
	return conflicts
}

func renderCompositionRouteResolutionDraft(parent string, inputs []compositionRouteAuditInput, conflicts []compositionRouteResolutionConflict) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Route Conflict Resolution Draft\n\n", parent)
	b.WriteString("Review this section before merging it into `LINER.md`. It turns route overlaps into explicit parent routing rules while keeping child mixtapes nested by reference.\n\n")
	b.WriteString("## Child Route Map\n\n")
	b.WriteString("| Child | Route | Status | Reference | Route source |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, input := range inputs {
		route := input.Route
		routeSource := "explicit"
		if !input.Explicit {
			route += " (inferred)"
			routeSource = "inferred"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n",
			escapeMarkdownTable(input.Child),
			escapeMarkdownTable(route),
			escapeMarkdownTable(input.Status),
			input.RelPath,
			routeSource,
		)
	}
	b.WriteString("\n## Shared Route Conflicts\n\n")
	if len(conflicts) == 0 {
		b.WriteString("No shared route tokens were found. Keep this draft only if you want to refresh the parent route table.\n\n")
	} else {
		b.WriteString("| Shared token | Children | Resolution rule |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, conflict := range conflicts {
			fmt.Fprintf(&b, "| %s | %s | %s |\n",
				escapeMarkdownTable(conflict.Token),
				escapeMarkdownTable(conflict.Children),
				escapeMarkdownTable("Ask one clarifying question when the request only names `"+conflict.Token+"`; otherwise route to the child whose full route has the strongest exact match."),
			)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Parent Routing Rules\n\n")
	b.WriteString("- Load the parent `LINER.md` first and use this route map before loading any child operating layer.\n")
	b.WriteString("- Route to the narrowest child when the request includes child-specific route words beyond a shared token.\n")
	b.WriteString("- Ask one clarifying question when a request only names a shared token and multiple children remain plausible.\n")
	b.WriteString("- If ambiguity remains after clarification, keep the child nested and record a route audit instead of silently choosing.\n")
	b.WriteString("- Do not copy child sources, skills, or `LINER.md` content through this route-resolution draft.\n")
	return b.String()
}

func renderCompositionRouteResolutionAudit(now time.Time, inputs []compositionRouteAuditInput, conflicts []compositionRouteResolutionConflict) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Route Resolution Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Created a reviewed parent routing draft from current child route references. Parent `LINER.md` is unchanged until Review Composition Draft is accepted.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Child references reviewed: %d\n", len(inputs))
	fmt.Fprintf(&b, "- Shared route conflicts found: %d\n", len(conflicts))
	b.WriteString("\n## Conflicts\n\n")
	if len(conflicts) == 0 {
		b.WriteString("- No shared route tokens were found.\n")
	} else {
		for _, conflict := range conflicts {
			fmt.Fprintf(&b, "- `%s`: %s\n", conflict.Token, conflict.Children)
		}
	}
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- No child files, sources, skills, or operating-layer files were changed.\n")
	b.WriteString("- This draft resolves parent routing behavior only after explicit review/apply.\n")
	return b.String()
}
