package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type compositionSkillConflictRow struct {
	ChildSkill  string
	ParentSkill string
	Status      string
	Reason      string
}

func (m Model) runCompositionSkillConflictReview() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before reviewing parent skill conflicts."
		return m, nil
	}
	report, err := writeCompositionSkillConflictReview(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.note = "Wrote " + report.RelPath + "."
	return m.openPreview(report.RelPath)
}

func writeCompositionSkillConflictReview(project string, item compositionFile) (auditFile, error) {
	input, err := compositionPromotionInput(project, item)
	if err != nil {
		return auditFile{}, err
	}
	childProject, referenceIssue := compositionChildProjectReference(project, item)
	if referenceIssue != "" {
		return auditFile{}, fmt.Errorf("cannot review child skill conflicts: %s", referenceIssue)
	}
	rows, err := compositionSkillConflictRows(project, childProject, slug(input.Child))
	if err != nil {
		return auditFile{}, err
	}
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-composition-skill-conflicts"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderCompositionSkillConflictReview(now, input, childProject, rows)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "composition",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func compositionSkillConflictRows(parentProject string, childProject string, childSlug string) ([]compositionSkillConflictRow, error) {
	childSkillsDir := filepath.Join(childProject, "skills")
	entries, err := os.ReadDir(childSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []compositionSkillConflictRow{{
				Status: "skipped",
				Reason: "child skills/ folder missing",
			}}, nil
		}
		return nil, err
	}
	rows := []compositionSkillConflictRow{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		childRel := filepath.Join("skills", entry.Name())
		childPath := filepath.Join(childProject, childRel)
		childData, err := os.ReadFile(childPath)
		if err != nil {
			return nil, err
		}
		targetRel := filepath.Join("skills", childSlug+"-"+entry.Name())
		targetPath := filepath.Join(parentProject, targetRel)
		status := "clear"
		reason := "no parent skill exists at the production merge target"
		if parentData, err := os.ReadFile(targetPath); err == nil {
			status = "duplicate"
			reason = "parent already has an identical namespaced skill"
			if !bytes.Equal(parentData, childData) {
				status = "conflict"
				reason = "parent already has a different skill at the production merge target"
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		} else if _, err := os.Stat(filepath.Join(parentProject, childRel)); err == nil {
			status = "overlap"
			reason = "parent has an un-namespaced skill with the same filename; review ownership before production merge"
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		rows = append(rows, compositionSkillConflictRow{
			ChildSkill:  childRel,
			ParentSkill: targetRel,
			Status:      status,
			Reason:      reason,
		})
	}
	if len(rows) == 0 {
		rows = append(rows, compositionSkillConflictRow{
			Status: "skipped",
			Reason: "child skills/ folder had no top-level markdown skills",
		})
	}
	sort.SliceStable(rows, func(i int, j int) bool {
		return strings.ToLower(rows[i].ChildSkill) < strings.ToLower(rows[j].ChildSkill)
	})
	return rows, nil
}

func renderCompositionSkillConflictReview(now time.Time, input compositionRouteAuditInput, childProject string, rows []compositionSkillConflictRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Skill Conflict Review\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local review checks selected child skills against the parent skill namespace before production merge. It does not copy, overwrite, disable, or rewrite any skills.\n\n")
	b.WriteString("## Selected Child\n\n")
	b.WriteString("| Child | Route | Status | Reference | Child Project |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| %s | %s | %s | `%s` | `%s` |\n\n",
		escapeMarkdownTable(input.Child),
		escapeMarkdownTable(input.Route),
		escapeMarkdownTable(input.Status),
		input.RelPath,
		escapeMarkdownTable(childProject),
	)
	b.WriteString("## Skill Findings\n\n")
	b.WriteString("| Child skill | Parent target | Status | Reason |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n",
			escapeMarkdownTable(row.ChildSkill),
			escapeMarkdownTable(row.ParentSkill),
			escapeMarkdownTable(row.Status),
			escapeMarkdownTable(row.Reason),
		)
	}
	b.WriteString("\n## Recommendations\n\n")
	b.WriteString("- Resolve `conflict` rows before pressing `x` for production merge; the merge will skip those target files instead of overwriting parent skills.\n")
	b.WriteString("- Treat `overlap` rows as ownership questions: keep parent-wide skills generic and child-specific skills namespaced.\n")
	b.WriteString("- Run skill-corpus alignment after meaningful child skills are merged into the parent.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No parent or child skill files were changed by this review.\n")
	b.WriteString("- No child sources, tape sources, or operating-layer files were copied.\n")
	return b.String()
}
