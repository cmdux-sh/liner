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

	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type compositionProductionMergeResult struct {
	AuditRel     string
	TapeBackup   string
	ChildProject string
	Input        compositionRouteAuditInput
	Sources      []compositionMergedSource
	Skills       []compositionMergedSkill
	Skipped      []string
}

type compositionMergedSource struct {
	Label       string
	Status      string
	Destination string
	Reason      string
}

type compositionMergedSkill struct {
	SourceRel string
	DestRel   string
	Status    string
	Reason    string
}

func (m Model) mergeCompositionChild() (Model, tea.Cmd) {
	item, ok := m.selectedComposition()
	if !ok || item.Kind == "lineage" {
		m.err = "Select a child reference before merging child production artifacts."
		return m, nil
	}
	result, updated, err := writeCompositionProductionMerge(m.currentPath, item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.currentTape = updated
	m.note = "Merged child production artifacts; audit written to " + result.AuditRel + "."
	return m.openPreview(result.AuditRel)
}

func writeCompositionProductionMerge(project string, item compositionFile) (compositionProductionMergeResult, tape.Tape, error) {
	parent, err := tape.ReadProject(project)
	if err != nil {
		return compositionProductionMergeResult{}, tape.Tape{}, fmt.Errorf("parent tape.yaml is required before merging child production artifacts: %w", err)
	}
	input, err := compositionPromotionInput(project, item)
	if err != nil {
		return compositionProductionMergeResult{}, tape.Tape{}, err
	}
	childProject, referenceIssue := compositionChildProjectReference(project, item)
	if referenceIssue != "" {
		return compositionProductionMergeResult{}, tape.Tape{}, fmt.Errorf("cannot merge child production artifacts: %s", referenceIssue)
	}
	now := time.Now()
	result := compositionProductionMergeResult{
		AuditRel:     filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-production-merge.md"),
		ChildProject: childProject,
		Input:        input,
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return compositionProductionMergeResult{}, tape.Tape{}, err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "composition"), 0o755); err != nil {
		return compositionProductionMergeResult{}, tape.Tape{}, err
	}
	childSlug := slug(input.Child)
	childTape, err := tape.ReadProject(childProject)
	if err != nil {
		if os.IsNotExist(err) {
			result.Skipped = append(result.Skipped, "child tape.yaml missing; no child tape sources were merged.")
		} else {
			return compositionProductionMergeResult{}, tape.Tape{}, fmt.Errorf("could not read child tape.yaml: %w", err)
		}
	} else {
		sourceRows, updatedParent, err := compositionMergeChildSources(project, childProject, childSlug, input, parent, childTape, now)
		if err != nil {
			return compositionProductionMergeResult{}, tape.Tape{}, err
		}
		result.Sources = sourceRows
		if len(updatedParent.Sources) != len(parent.Sources) {
			backupRel, err := backupCompositionFile(project, "tape.yaml", now, "previous-tape.yaml")
			if err != nil {
				return compositionProductionMergeResult{}, tape.Tape{}, err
			}
			result.TapeBackup = backupRel
			if err := tape.WriteProject(project, updatedParent); err != nil {
				return compositionProductionMergeResult{}, tape.Tape{}, err
			}
			parent = updatedParent
		}
	}
	skillRows, err := compositionMergeChildSkills(project, childProject, childSlug)
	if err != nil {
		return compositionProductionMergeResult{}, tape.Tape{}, err
	}
	result.Skills = skillRows
	if len(result.Sources) == 0 && len(result.Skills) == 0 && len(result.Skipped) == 0 {
		result.Skipped = append(result.Skipped, "no child sources or skills were found to merge.")
	}
	if err := os.WriteFile(filepath.Join(project, result.AuditRel), []byte(renderCompositionProductionMergeAudit(now, result)), 0o644); err != nil {
		return compositionProductionMergeResult{}, tape.Tape{}, err
	}
	return result, parent, nil
}

func compositionMergeChildSources(parentProject string, childProject string, childSlug string, input compositionRouteAuditInput, parent tape.Tape, child tape.Tape, now time.Time) ([]compositionMergedSource, tape.Tape, error) {
	rows := []compositionMergedSource{}
	seen := map[string]bool{}
	for _, src := range parent.Sources {
		if key := compositionMergeSourceKey(src); key != "" {
			seen[key] = true
		}
	}
	for _, childSource := range child.Sources {
		source := childSource
		label := compositionMergeSourceLabel(childSource)
		destination := ""
		if childSource.Path != nil && strings.TrimSpace(*childSource.Path) != "" {
			rebased, destRel, skipped, err := compositionRebaseChildSourcePath(parentProject, childProject, childSlug, childSource)
			if err != nil {
				return rows, parent, err
			}
			if skipped != "" {
				rows = append(rows, compositionMergedSource{Label: label, Status: "skipped", Reason: skipped})
				continue
			}
			source = rebased
			destination = destRel
		}
		source.Note = compositionMergedSourceNote(source.Note, input, now)
		key := compositionMergeSourceKey(source)
		if key != "" && seen[key] {
			rows = append(rows, compositionMergedSource{Label: label, Status: "skipped", Destination: destination, Reason: "duplicate source already exists in parent tape.yaml"})
			continue
		}
		parent.Sources = append(parent.Sources, source)
		if key != "" {
			seen[key] = true
		}
		rows = append(rows, compositionMergedSource{Label: label, Status: "added", Destination: destination, Reason: "merged from child tape.yaml"})
	}
	if len(rows) == 0 {
		rows = append(rows, compositionMergedSource{Status: "skipped", Reason: "child tape.yaml had no saved sources"})
	}
	return rows, parent, nil
}

func compositionRebaseChildSourcePath(parentProject string, childProject string, childSlug string, src tape.Source) (tape.Source, string, string, error) {
	if src.Path == nil || strings.TrimSpace(*src.Path) == "" {
		return src, "", "", nil
	}
	childSourcePath, ok := compositionResolveChildPath(childProject, *src.Path)
	if !ok {
		return src, "", "source path leaves the child project and was not copied: " + *src.Path, nil
	}
	info, err := os.Stat(childSourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return src, "", "source path missing in child project: " + *src.Path, nil
		}
		return src, "", "", err
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return src, "", "source path is not a regular file: " + *src.Path, nil
	}
	destRel := compositionMergedSourceDestRel(childProject, childSourcePath, childSlug)
	finalRel, _, err := copyCompositionMergeFile(childSourcePath, parentProject, destRel)
	if err != nil {
		return src, "", "", err
	}
	out := src
	out.Path = stringPointer(finalRel)
	return out, finalRel, "", nil
}

func compositionResolveChildPath(childProject string, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	var path string
	if filepath.IsAbs(raw) {
		path = filepath.Clean(raw)
	} else {
		path = filepath.Clean(projectAbsPath(childProject, raw))
	}
	if !compositionPathInside(childProject, path) {
		return "", false
	}
	return path, true
}

func compositionPathInside(root string, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func compositionMergedSourceDestRel(childProject string, sourcePath string, childSlug string) string {
	sourceRoot := childProject
	if compositionPathInside(projectCorpusPath(childProject), sourcePath) {
		sourceRoot = projectCorpusPath(childProject)
	}
	rel, err := filepath.Rel(sourceRoot, sourcePath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(sourcePath)
	}
	rel = filepath.Clean(rel)
	localPrefix := "local-sources" + string(os.PathSeparator)
	if strings.HasPrefix(rel, localPrefix) {
		rel = strings.TrimPrefix(rel, localPrefix)
	}
	if rel == "." || rel == "" {
		rel = filepath.Base(sourcePath)
	}
	return filepath.Join("local-sources", "composition", childSlug, rel)
}

func copyCompositionMergeFile(sourcePath string, project string, destRel string) (string, int64, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", 0, err
	}
	destRel = filepath.Clean(destRel)
	ext := filepath.Ext(destRel)
	stem := strings.TrimSuffix(destRel, ext)
	for index := 0; index < 100; index++ {
		candidateRel := destRel
		if index > 0 {
			candidateRel = fmt.Sprintf("%s-%d%s", stem, index+1, ext)
		}
		candidatePath := projectAbsPath(project, candidateRel)
		existing, err := os.ReadFile(candidatePath)
		if err == nil {
			if bytes.Equal(existing, data) {
				return candidateRel, int64(len(data)), nil
			}
			continue
		}
		if !os.IsNotExist(err) {
			return "", 0, err
		}
		if err := os.MkdirAll(filepath.Dir(candidatePath), 0o755); err != nil {
			return "", 0, err
		}
		if err := os.WriteFile(candidatePath, data, 0o644); err != nil {
			return "", 0, err
		}
		return candidateRel, int64(len(data)), nil
	}
	return "", 0, fmt.Errorf("could not find an available destination for %s", destRel)
}

func compositionMergeChildSkills(parentProject string, childProject string, childSlug string) ([]compositionMergedSkill, error) {
	childSkillsDir := filepath.Join(childProject, "skills")
	entries, err := os.ReadDir(childSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []compositionMergedSkill{{Status: "skipped", Reason: "child skills/ folder missing"}}, nil
		}
		return nil, err
	}
	rows := []compositionMergedSkill{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		sourceRel := filepath.Join("skills", entry.Name())
		sourcePath := filepath.Join(childProject, sourceRel)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}
		destRel := filepath.Join("skills", childSlug+"-"+entry.Name())
		destPath := filepath.Join(parentProject, destRel)
		if existing, err := os.ReadFile(destPath); err == nil {
			status := "skipped"
			reason := "parent already has identical namespaced skill"
			if !bytes.Equal(existing, data) {
				reason = "parent has a different skill at " + destRel + "; resolve manually before overwriting"
			}
			rows = append(rows, compositionMergedSkill{SourceRel: sourceRel, DestRel: destRel, Status: status, Reason: reason})
			continue
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return nil, err
		}
		rows = append(rows, compositionMergedSkill{SourceRel: sourceRel, DestRel: destRel, Status: "added", Reason: "copied from child skills/"})
	}
	if len(rows) == 0 {
		rows = append(rows, compositionMergedSkill{Status: "skipped", Reason: "child skills/ folder had no top-level markdown skills"})
	}
	sort.SliceStable(rows, func(i int, j int) bool {
		return strings.ToLower(rows[i].SourceRel) < strings.ToLower(rows[j].SourceRel)
	})
	return rows, nil
}

func backupCompositionFile(project string, rel string, now time.Time, suffix string) (string, error) {
	data, err := os.ReadFile(projectAbsPath(project, rel))
	if err != nil {
		return "", err
	}
	backupRel := filepath.Join("working", "composition", now.Format("2006-01-02-150405")+"-"+suffix)
	if err := os.MkdirAll(filepath.Join(project, filepath.Dir(backupRel)), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(project, backupRel), data, 0o644); err != nil {
		return "", err
	}
	return backupRel, nil
}

func compositionMergeSourceKey(src tape.Source) string {
	if strings.TrimSpace(src.URL) != "" {
		return strings.ToLower(strings.TrimSpace(src.Type) + "|url|" + strings.TrimSpace(src.URL))
	}
	if src.Path != nil && strings.TrimSpace(*src.Path) != "" {
		return strings.ToLower(strings.TrimSpace(src.Type) + "|path|" + filepath.Clean(strings.TrimSpace(*src.Path)))
	}
	if src.Citation != nil && strings.TrimSpace(*src.Citation) != "" {
		return strings.ToLower(strings.TrimSpace(src.Type) + "|citation|" + strings.TrimSpace(*src.Citation))
	}
	return ""
}

func compositionMergeSourceLabel(src tape.Source) string {
	switch {
	case strings.TrimSpace(src.URL) != "":
		return src.URL
	case src.Path != nil && strings.TrimSpace(*src.Path) != "":
		return *src.Path
	case src.Citation != nil && strings.TrimSpace(*src.Citation) != "":
		return *src.Citation
	default:
		return fallbackText(src.Type, "source")
	}
}

func compositionMergedSourceNote(note *string, input compositionRouteAuditInput, now time.Time) *string {
	provenance := fmt.Sprintf("Merged from child `%s` via `%s` on %s. Keep child route `%s` in mind when reusing this source in the parent.", input.Child, input.RelPath, now.Format("2006-01-02"), input.Route)
	if note == nil || strings.TrimSpace(*note) == "" {
		return stringPointer(provenance)
	}
	merged := strings.TrimSpace(*note) + "\n\n" + provenance
	return stringPointer(merged)
}

func renderCompositionProductionMergeAudit(now time.Time, result compositionProductionMergeResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Production Merge Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Merged selected child production artifacts into the parent mixtape with namespacing, duplicate checks, and explicit provenance. This changed parent production files only through the reviewed Composition merge action.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Child reference: `%s`\n", result.Input.RelPath)
	fmt.Fprintf(&b, "- Child project: `%s`\n", result.ChildProject)
	fmt.Fprintf(&b, "- Route: %s\n", result.Input.Route)
	fmt.Fprintf(&b, "- Explicit route: %s\n", yesNo(result.Input.Explicit))
	if result.TapeBackup != "" {
		fmt.Fprintf(&b, "- Previous parent `tape.yaml` backup: `%s`\n", result.TapeBackup)
	}
	b.WriteString("\n## Source Merge\n\n")
	writeCompositionMergedSourcesTable(&b, result.Sources)
	b.WriteString("## Skill Merge\n\n")
	writeCompositionMergedSkillsTable(&b, result.Skills)
	writeCompositionSkippedList(&b, result.Skipped)
	b.WriteString("## Boundaries\n\n")
	b.WriteString("- Child `LINER.md` was not merged into parent `LINER.md`; routing remains managed by the composition section.\n")
	b.WriteString("- Child local files were copied under `local-sources/composition/<child>/` before parent tape paths were added.\n")
	b.WriteString("- Existing parent skills were not overwritten; conflicts are listed for manual review.\n")
	b.WriteString("- Run source-note, skill-corpus, and route audits after production merges that add meaningful new material.\n")
	return b.String()
}

func writeCompositionMergedSourcesTable(b *strings.Builder, rows []compositionMergedSource) {
	if len(rows) == 0 {
		b.WriteString("No child sources were found.\n\n")
		return
	}
	b.WriteString("| Source | Status | Parent destination | Reason |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n",
			escapeMarkdownTable(row.Label),
			escapeMarkdownTable(row.Status),
			escapeMarkdownTable(row.Destination),
			escapeMarkdownTable(row.Reason),
		)
	}
	b.WriteString("\n")
}

func writeCompositionMergedSkillsTable(b *strings.Builder, rows []compositionMergedSkill) {
	if len(rows) == 0 {
		b.WriteString("No child skills were found.\n\n")
		return
	}
	b.WriteString("| Child skill | Parent skill | Status | Reason |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(b, "| %s | `%s` | %s | %s |\n",
			escapeMarkdownTable(row.SourceRel),
			escapeMarkdownTable(row.DestRel),
			escapeMarkdownTable(row.Status),
			escapeMarkdownTable(row.Reason),
		)
	}
	b.WriteString("\n")
}

func stringPointer(value string) *string {
	return &value
}
