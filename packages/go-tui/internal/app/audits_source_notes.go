package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type sourceNoteFinding struct {
	Source         string
	Status         string
	Evidence       string
	Recommendation string
}

type sourceNoteAppliedChange struct {
	Source string
	Status string
	Note   string
}

func hasAcceptedTapeSources(project string) bool {
	t, err := tape.ReadProject(project)
	return err == nil && len(t.Sources) > 0
}

func writeSourceNoteQualityAudit(project string) (auditFile, error) {
	t, err := tape.ReadProject(project)
	if err != nil {
		if os.IsNotExist(err) {
			return auditFile{}, fmt.Errorf("No saved sources found in tape.yaml. Save an assembly draft before running source-note quality.")
		}
		return auditFile{}, err
	}
	if len(t.Sources) == 0 {
		return auditFile{}, fmt.Errorf("No saved sources found in tape.yaml. Save an assembly draft before running source-note quality.")
	}

	findings := sourceNoteQualityFindings(t.Sources)
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-source-note-quality"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderSourceNoteQualityAudit(now, t.Sources, findings)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "source notes",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeSourceNoteCleanupDraft(project string) (auditFile, error) {
	t, err := tape.ReadProject(project)
	if err != nil {
		if os.IsNotExist(err) {
			return auditFile{}, fmt.Errorf("No saved sources found in tape.yaml. Save an assembly draft before creating a source-note cleanup draft.")
		}
		return auditFile{}, err
	}
	if len(t.Sources) == 0 {
		return auditFile{}, fmt.Errorf("No saved sources found in tape.yaml. Save an assembly draft before creating a source-note cleanup draft.")
	}

	findings := sourceNoteQualityFindings(t.Sources)
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-source-note-cleanup-draft"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderSourceNoteCleanupDraft(now, t.Sources, findings)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "source notes",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func applySourceNoteCleanup(runner core.Runner, project string, draftRel string) (auditFile, int, error) {
	return applySourceNoteCleanupWithAuditWriter(runner, project, draftRel, os.WriteFile)
}

func applySourceNoteCleanupWithAuditWriter(runner core.Runner, project string, draftRel string, writeAudit func(string, []byte, os.FileMode) error) (auditFile, int, error) {
	draftRel = filepath.Clean(strings.TrimSpace(draftRel))
	if draftRel == "" || draftRel == "." || filepath.IsAbs(draftRel) || draftRel == ".." || strings.HasPrefix(draftRel, ".."+string(filepath.Separator)) {
		return auditFile{}, 0, fmt.Errorf("No reviewed source-note cleanup draft is selected.")
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); err != nil {
		if os.IsNotExist(err) {
			return auditFile{}, 0, fmt.Errorf("Source-note cleanup draft no longer exists: %s", draftRel)
		}
		return auditFile{}, 0, err
	}
	t, err := tape.ReadProject(project)
	if err != nil {
		if os.IsNotExist(err) {
			return auditFile{}, 0, fmt.Errorf("No saved sources found in tape.yaml. Save an assembly draft before applying source-note cleanup.")
		}
		return auditFile{}, 0, err
	}
	if len(t.Sources) == 0 {
		return auditFile{}, 0, fmt.Errorf("No saved sources found in tape.yaml. Save an assembly draft before applying source-note cleanup.")
	}

	snapshot, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		return auditFile{}, 0, err
	}
	findings := sourceNoteQualityFindings(t.Sources)
	changes := make([]sourceNoteAppliedChange, 0, len(findings))
	receipts := []string{}
	for index, finding := range findings {
		if finding.Status == "strong" {
			continue
		}
		note := sourceNoteDraftNote(finding.Source, t.Sources[index], finding.Status)
		if optionalString(t.Sources[index].Note) == note {
			continue
		}
		sourceID, err := sourceNoteCleanupSourceID(snapshot, compositionMergeSourceLabel(t.Sources[index]), finding.Source, receipts)
		if err != nil {
			return auditFile{}, 0, err
		}
		plan, err := runner.PlanMaintenance(project, core.SourceOperation("source.update", sourceID, map[string]any{"note": note}))
		if err != nil {
			return auditFile{}, 0, sourceNotePartialError(err, receipts)
		}
		if plan.ApprovalRequired {
			return auditFile{}, 0, sourceNotePartialError(fmt.Errorf("Liner Core classified the Source update as approval-required. No implicit approval was granted; open Maintain project and review the exact Change Set for Source %s", sourceID), receipts)
		}
		receipt, err := runner.ApplyMaintenance(project, plan, false)
		if err != nil {
			return auditFile{}, 0, sourceNotePartialError(err, receipts)
		}
		receipts = append(receipts, receipt.ReceiptPath)
		changes = append(changes, sourceNoteAppliedChange{
			Source: finding.Source,
			Status: finding.Status,
			Note:   note,
		})
	}
	if len(changes) == 0 {
		return auditFile{}, 0, fmt.Errorf("No source-note cleanup changes are needed.")
	}

	now := time.Now()
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, 0, sourceNotePartialError(err, receipts)
	}
	name := now.Format("2006-01-02-150405") + "-source-note-cleanup-apply"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := writeAudit(path, []byte(renderSourceNoteCleanupApplyAudit(now, draftRel, receipts, changes)), 0o644); err != nil {
		return auditFile{}, 0, sourceNotePartialError(err, receipts)
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "source notes",
		Updated: now.Format("2006-01-02"),
	}, len(changes), nil
}

func sourceNoteCleanupSourceID(snapshot core.MaintenanceProjectSnapshot, locator string, sourceLabel string, receiptPaths []string) (string, error) {
	sourceIDs := maintenanceSourceIDsForLocator(snapshot, locator)
	if len(sourceIDs) == 0 {
		return "", sourceNotePartialError(fmt.Errorf("Core did not return an immutable Source ID for %s", sourceLabel), receiptPaths)
	}
	if len(sourceIDs) > 1 {
		return "", sourceNotePartialError(fmt.Errorf("Core returned ambiguous Source IDs for %s: %s", sourceLabel, strings.Join(sourceIDs, ", ")), receiptPaths)
	}
	return sourceIDs[0], nil
}

func sourceNotePartialError(err error, receiptPaths []string) error {
	if len(receiptPaths) == 0 {
		return err
	}
	return fmt.Errorf("Liner Core applied %d source-note update(s) before a later update failed. Durable receipts: %s. Refresh the Project before retrying: %w", len(receiptPaths), strings.Join(receiptPaths, ", "), err)
}

func sourceNoteQualityFindings(sources []tape.Source) []sourceNoteFinding {
	findings := make([]sourceNoteFinding, 0, len(sources))
	for index, source := range sources {
		note := strings.TrimSpace(optionalString(source.Note))
		issues := sourceNoteIssues(note)
		status := "strong"
		if len(issues) > 0 {
			status = strings.Join(issues, ", ")
		}
		findings = append(findings, sourceNoteFinding{
			Source:         sourceNoteLabel(source, index),
			Status:         status,
			Evidence:       sourceNoteEvidence(source, note),
			Recommendation: sourceNoteRecommendation(issues),
		})
	}
	return findings
}

func sourceNoteIssues(note string) []string {
	if note == "" {
		return []string{"missing note"}
	}

	var issues []string
	if len(note) < 40 || len(strings.Fields(note)) < 8 {
		issues = append(issues, "thin note")
	}
	if !sourceNoteHasUseCue(note) {
		issues = append(issues, "unclear use")
	}
	if !sourceNoteHasBoundaryCue(note) {
		issues = append(issues, "missing boundary")
	}
	return issues
}

func sourceNoteHasUseCue(note string) bool {
	value := " " + strings.ToLower(note) + " "
	return containsAny(value, []string{
		" read", " use", " look for", " skim", " compare", " anchor", " example",
		" extract", " apply", " evaluate", " watch", " listen", " cite", " ground",
	})
}

func sourceNoteHasBoundaryCue(note string) bool {
	value := " " + strings.ToLower(note) + " "
	return containsAny(value, []string{
		" limit", " scope", " boundary", " boundaries", " avoid", " only ", " except",
		" caveat", " risk", " unless", " outside", " not ", " cannot", " should not",
		" do not",
	})
}

func sourceNoteLabel(source tape.Source, index int) string {
	for _, value := range []string{
		optionalString(source.Citation),
		source.URL,
		optionalString(source.Path),
		source.Type,
	} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return fmt.Sprintf("source %d", index+1)
}

func sourceNoteEvidence(source tape.Source, note string) string {
	parts := []string{
		fmt.Sprintf("note chars: %d", len(note)),
		fmt.Sprintf("note words: %d", len(strings.Fields(note))),
		fmt.Sprintf("use cue: %s", yesNo(sourceNoteHasUseCue(note))),
		fmt.Sprintf("boundary cue: %s", yesNo(sourceNoteHasBoundaryCue(note))),
	}
	if source.Priority != "" {
		parts = append(parts, "priority: "+source.Priority)
	}
	if value := optionalString(source.Kind); value != "" {
		parts = append(parts, "kind: "+value)
	}
	if value := optionalString(source.Section); value != "" {
		parts = append(parts, "section: "+value)
	}
	return strings.Join(parts, "; ")
}

func sourceNoteRecommendation(issues []string) string {
	if len(issues) == 0 {
		return "Keep this source note; re-audit after changing the source or its role."
	}
	if len(issues) > 1 {
		return "Address each note issue before relying on this source in skills or synthesis."
	}
	switch issues[0] {
	case "missing note":
		return "Add a curator note that names how to use the source and where its boundaries are."
	case "thin note":
		return "Expand the note with concrete read/use guidance."
	case "unclear use":
		return "Name what an agent should read, compare, or extract from this source."
	case "missing boundary":
		return "Add scope, caveat, or limitation language so the source cannot overreach."
	default:
		return "Review this source note before treating it as authoritative."
	}
}

func renderSourceNoteQualityAudit(now time.Time, sources []tape.Source, findings []sourceNoteFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Source-Note Quality Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local audit checks accepted `tape.yaml` sources for curator notes that tell an external agent how to use the source and where not to over-apply it. It does not rewrite any project files.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- `tape.yaml`: %d saved source(s)\n", len(sources))
	b.WriteString("\n## Findings\n\n")
	b.WriteString("| Source | Status | Evidence | Recommendation |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, finding := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			escapeMarkdownTable(finding.Source),
			escapeMarkdownTable(finding.Status),
			escapeMarkdownTable(finding.Evidence),
			escapeMarkdownTable(finding.Recommendation),
		)
	}
	b.WriteString("\n## Recommended Review\n\n")
	b.WriteString("- Add curator notes to any saved source marked `missing note` before compiling it into operating guidance.\n")
	b.WriteString("- Tighten thin or unclear notes so each source has an obvious role in the mixtape.\n")
	b.WriteString("- Add boundary language for sources that should only apply to specific situations, domains, or decisions.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No files were changed by this audit.\n")
	b.WriteString("- Apply decisions only after reviewing the evidence above.\n")
	return b.String()
}

func renderSourceNoteCleanupDraft(now time.Time, sources []tape.Source, findings []sourceNoteFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Source-Note Cleanup Draft\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This draft turns the source-note quality audit into reviewable editing work. It does not rewrite `tape.yaml` or source files.\n\n")
	b.WriteString("## Proposed Edits\n\n")
	needsWork := 0
	for _, finding := range findings {
		if finding.Status != "strong" {
			needsWork++
		}
	}
	if needsWork == 0 {
		b.WriteString("No source-note cleanup actions are required by the local heuristic. Re-run after adding or changing sources.\n\n")
	} else {
		b.WriteString("| Source | Current Status | Evidence | Draft Action | Note Brief |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for index, finding := range findings {
			if finding.Status == "strong" {
				continue
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				escapeMarkdownTable(finding.Source),
				escapeMarkdownTable(finding.Status),
				escapeMarkdownTable(finding.Evidence),
				escapeMarkdownTable(sourceNoteCleanupAction(finding.Status)),
				escapeMarkdownTable(sourceNoteDraftBrief(finding.Source, sources[index], finding.Status)),
			)
		}
		b.WriteString("\n")
	}
	b.WriteString("## How To Apply\n\n")
	b.WriteString("- Review this draft in the TUI, then accept it to request Source metadata Change Sets from Liner Core.\n")
	b.WriteString("- Each revised note should say how to use the source and where its advice stops applying.\n")
	b.WriteString("- Never edit `tape.yaml` directly; keep the Core receipts and rerun the source-note audit after apply.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No files were changed except this cleanup draft.\n")
	b.WriteString("- Do not copy these briefs blindly; review them against the source before editing accepted notes.\n")
	return b.String()
}

func renderSourceNoteCleanupApplyAudit(now time.Time, draftRel string, receiptPaths []string, changes []sourceNoteAppliedChange) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Source-Note Cleanup Apply Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This audit records reviewed source-note cleanup changes applied to `tape.yaml`.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Reviewed draft: `%s`\n", draftRel)
	for _, receiptPath := range receiptPaths {
		fmt.Fprintf(&b, "- Core Change Receipt: `%s`\n", receiptPath)
	}
	fmt.Fprintf(&b, "- Updated source note(s): %d\n\n", len(changes))
	b.WriteString("## Applied Changes\n\n")
	b.WriteString("| Source | Previous Status | Applied Note |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, change := range changes {
		fmt.Fprintf(&b, "| %s | %s | %s |\n",
			escapeMarkdownTable(change.Source),
			escapeMarkdownTable(change.Status),
			escapeMarkdownTable(change.Note),
		)
	}
	b.WriteString("\n## Decision Log\n\n")
	b.WriteString("- Source notes were updated by atomic Liner Core Change Sets only after the cleanup draft review step.\n")
	b.WriteString("- No source files were changed by this apply action.\n")
	return b.String()
}

func sourceNoteCleanupAction(status string) string {
	switch {
	case strings.Contains(status, "missing note"):
		return "Add a curator note."
	case strings.Contains(status, "thin note"):
		return "Expand the existing note."
	case strings.Contains(status, "unclear use"):
		return "Name what to read, compare, or extract."
	case strings.Contains(status, "missing boundary"):
		return "Add scope and limitation language."
	default:
		return "Review the accepted note."
	}
}

func sourceNoteDraftBrief(label string, source tape.Source, status string) string {
	kind := optionalString(source.Kind)
	if kind == "" {
		kind = source.Type
	}
	section := optionalString(source.Section)
	if section == "" {
		section = "the mixtape's job"
	}
	brief := fmt.Sprintf("Use `%s` as a %s source for %s; name one concrete read/use cue and one boundary.", label, kind, section)
	if strings.Contains(status, "missing boundary") && !strings.Contains(status, "unclear use") {
		brief = fmt.Sprintf("Keep the current use cue for `%s`, then add where this source should not be over-applied.", label)
	}
	return brief
}

func sourceNoteDraftNote(label string, source tape.Source, status string) string {
	kind := optionalString(source.Kind)
	if kind == "" {
		kind = source.Type
	}
	section := optionalString(source.Section)
	if section == "" {
		section = "the mixtape's job"
	}
	note := fmt.Sprintf("Use %s as a %s source for %s. Read it for one concrete cue, and limit its guidance to that source's documented scope.", label, kind, section)
	if strings.Contains(status, "missing boundary") && !strings.Contains(status, "unclear use") {
		note = fmt.Sprintf("Keep the current use cue for %s, then limit this source to its documented scope so it is not over-applied.", label)
	}
	return note
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
