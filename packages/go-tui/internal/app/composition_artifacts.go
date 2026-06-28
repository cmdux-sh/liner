package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func writeCompositionNesting(project string, parent string, children []compositionChild) (compositionNestResult, error) {
	if len(children) == 0 {
		return compositionNestResult{}, fmt.Errorf("at least one child reference is required")
	}
	now := time.Now()
	workingDir := filepath.Join(project, "working", "composition")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		return compositionNestResult{}, err
	}
	result := compositionNestResult{
		LineageRel: "lineage.yaml",
		DraftRel:   compositionDraftRelPath,
		AuditRel:   filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-nesting.md"),
		Children:   children,
	}
	lineagePath := filepath.Join(project, result.LineageRel)
	if data, err := os.ReadFile(lineagePath); err == nil {
		result.PreviousCopyRel = filepath.Join("working", "composition", now.Format("2006-01-02-150405")+"-previous-lineage.yaml")
		if err := os.WriteFile(filepath.Join(project, result.PreviousCopyRel), data, 0o644); err != nil {
			return compositionNestResult{}, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return compositionNestResult{}, err
	}
	lineage := compositionLineage{
		Parent:       parent,
		Mode:         "nested",
		Updated:      now.Format(time.RFC3339),
		Children:     children,
		PreviousCopy: result.PreviousCopyRel,
		History: []compositionLineageEvent{{
			Date:  now.Format("2006-01-02 15:04:05"),
			Event: "Created nested child routing plan from Composition.",
			Audit: result.AuditRel,
			Draft: result.DraftRel,
		}},
	}
	lineageData, err := yaml.Marshal(lineage)
	if err != nil {
		return compositionNestResult{}, err
	}
	if err := os.WriteFile(lineagePath, lineageData, 0o644); err != nil {
		return compositionNestResult{}, err
	}
	if err := os.WriteFile(filepath.Join(project, result.DraftRel), []byte(renderCompositionLinerDraft(parent, children)), 0o644); err != nil {
		return compositionNestResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return compositionNestResult{}, err
	}
	if err := os.WriteFile(filepath.Join(project, result.AuditRel), []byte(renderCompositionNestingAudit(now, result)), 0o644); err != nil {
		return compositionNestResult{}, err
	}
	return result, nil
}

func renderCompositionLinerDraft(parent string, children []compositionChild) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Composition Routing Draft\n\n", parent)
	b.WriteString("Review this section before merging it into `LINER.md`. It keeps child mixtapes nested by reference instead of copying their sources into the parent.\n\n")
	b.WriteString("## Child Mixtape Routing\n\n")
	b.WriteString("| Child | Route | Status | Reference |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, child := range children {
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` |\n",
			escapeMarkdownTable(child.Name),
			escapeMarkdownTable(child.Route),
			escapeMarkdownTable(child.Status),
			child.Ref,
		)
	}
	b.WriteString("\n## Routing Rules\n\n")
	b.WriteString("- Load the parent `LINER.md` first to decide whether the request belongs to the parent or a child.\n")
	b.WriteString("- Route specialized requests to the narrowest child whose route matches the job.\n")
	b.WriteString("- Keep child scope intact; do not merge child source claims into the parent unless a later merge audit says the source sets are small and non-overlapping.\n")
	b.WriteString("- If two children disagree, name the conflict and create an audit before changing either operating layer.\n")
	return b.String()
}

func renderCompositionNestingAudit(now time.Time, result compositionNestResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Nesting Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Created a nested composition plan. Child mixtapes remain referenced by path; no child sources were copied into the parent.\n\n")
	b.WriteString("## Written Artifacts\n\n")
	fmt.Fprintf(&b, "- `%s`\n", result.LineageRel)
	fmt.Fprintf(&b, "- `%s`\n", result.DraftRel)
	if result.PreviousCopyRel != "" {
		fmt.Fprintf(&b, "- `%s` previous lineage backup\n", result.PreviousCopyRel)
	}
	b.WriteString("\n## Children\n\n")
	b.WriteString("| Child | Route | Status | Reference |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, child := range result.Children {
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` |\n",
			escapeMarkdownTable(child.Name),
			escapeMarkdownTable(child.Route),
			escapeMarkdownTable(child.Status),
			child.Ref,
		)
	}
	b.WriteString("\n## Follow-Up\n\n")
	b.WriteString("- Review `working/LINER-composition-draft.md` before editing `LINER.md`.\n")
	b.WriteString("- Run contradiction audits if child scopes overlap or statuses show `review`.\n")
	b.WriteString("- Choose merge only after confirming the child source sets are small and non-overlapping.\n")
	return b.String()
}

func writeCompositionMergeDraft(project string, parent string, input compositionRouteAuditInput) (string, error) {
	now := time.Now()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(project, compositionDraftRelPath), []byte(renderCompositionMergeDraft(parent, input)), 0o644); err != nil {
		return "", err
	}
	auditRel := filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-merge-draft.md")
	if err := os.WriteFile(filepath.Join(project, auditRel), []byte(renderCompositionMergeDraftAudit(now, input)), 0o644); err != nil {
		return "", err
	}
	return auditRel, nil
}

func writeCompositionCopyPacket(project string, item compositionFile) (compositionCopyPacketResult, error) {
	input, err := compositionPromotionInput(project, item)
	if err != nil {
		return compositionCopyPacketResult{}, err
	}
	childProject, referenceIssue := compositionChildProjectReference(project, item)
	candidates := compositionCopyCandidates(childProject, referenceIssue)
	now := time.Now()
	if err := os.MkdirAll(filepath.Join(project, "working", "composition"), 0o755); err != nil {
		return compositionCopyPacketResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return compositionCopyPacketResult{}, err
	}
	base := now.Format("2006-01-02-150405") + "-" + slug(input.Child)
	packetRel := filepath.Join("working", "composition", base+"-copy-packet.md")
	auditRel := filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-copy-packet.md")
	if err := os.WriteFile(filepath.Join(project, packetRel), []byte(renderCompositionCopyPacket(now, input, childProject, referenceIssue, candidates)), 0o644); err != nil {
		return compositionCopyPacketResult{}, err
	}
	if err := os.WriteFile(filepath.Join(project, auditRel), []byte(renderCompositionCopyPacketAudit(now, packetRel, input, childProject, referenceIssue, candidates)), 0o644); err != nil {
		return compositionCopyPacketResult{}, err
	}
	return compositionCopyPacketResult{
		PacketRel: packetRel,
		AuditRel:  auditRel,
		Input:     input,
	}, nil
}

func writeCompositionCopyApply(project string, item compositionFile) (compositionCopyApplyResult, error) {
	input, err := compositionPromotionInput(project, item)
	if err != nil {
		return compositionCopyApplyResult{}, err
	}
	childProject, referenceIssue := compositionChildProjectReference(project, item)
	if referenceIssue != "" {
		return compositionCopyApplyResult{}, fmt.Errorf("cannot apply composition copy: %s", referenceIssue)
	}
	now := time.Now()
	baseRel := filepath.Join("working", "composition", "copied", slug(input.Child), now.Format("2006-01-02-150405"))
	baseDir := filepath.Join(project, baseRel)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return compositionCopyApplyResult{}, err
	}
	result := compositionCopyApplyResult{
		SnapshotRel:  baseRel,
		AuditRel:     filepath.Join("working", "audits", now.Format("2006-01-02-150405")+"-composition-copy-apply.md"),
		ChildProject: childProject,
		Input:        input,
	}
	for _, rel := range []string{"MIXTAPE.md", "LINER.md", "tape.yaml"} {
		file, skipped, err := copyCompositionExactFile(childProject, baseDir, rel)
		if err != nil {
			return compositionCopyApplyResult{}, err
		}
		if skipped != "" {
			result.Skipped = append(result.Skipped, skipped)
			continue
		}
		result.Files = append(result.Files, file)
	}
	files, skipped, err := copyCompositionTopLevelFiles(childProject, baseDir, "skills", "skills", ".md")
	if err != nil {
		return compositionCopyApplyResult{}, err
	}
	result.Files = append(result.Files, files...)
	result.Skipped = append(result.Skipped, skipped...)
	files, skipped, err = copyCompositionTreeFiles(childProject, baseDir, "local-sources", "local-sources", "")
	if err != nil {
		return compositionCopyApplyResult{}, err
	}
	result.Files = append(result.Files, files...)
	result.Skipped = append(result.Skipped, skipped...)
	files, skipped, err = copyCompositionTopLevelFiles(childProject, baseDir, filepath.Join("working", "audits"), "audits", ".md")
	if err != nil {
		return compositionCopyApplyResult{}, err
	}
	result.Files = append(result.Files, files...)
	result.Skipped = append(result.Skipped, skipped...)
	if err := os.WriteFile(filepath.Join(baseDir, "README.md"), []byte(renderCompositionCopyApplyManifest(now, result)), 0o644); err != nil {
		return compositionCopyApplyResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		return compositionCopyApplyResult{}, err
	}
	if err := os.WriteFile(filepath.Join(project, result.AuditRel), []byte(renderCompositionCopyApplyAudit(now, result)), 0o644); err != nil {
		return compositionCopyApplyResult{}, err
	}
	return result, nil
}

func copyCompositionExactFile(childProject string, snapshotDir string, rel string) (compositionCopiedFile, string, error) {
	sourcePath := projectAbsPath(childProject, rel)
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return compositionCopiedFile{}, rel + " missing in child project.", nil
		}
		return compositionCopiedFile{}, "", err
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return compositionCopiedFile{}, rel + " is not a regular file.", nil
	}
	bytes, err := copyCompositionRegularFile(sourcePath, filepath.Join(snapshotDir, rel))
	if err != nil {
		return compositionCopiedFile{}, "", err
	}
	return compositionCopiedFile{Artifact: rel, DestRel: rel, Bytes: bytes}, "", nil
}

func copyCompositionTopLevelFiles(childProject string, snapshotDir string, sourceDirRel string, destDirRel string, extension string) ([]compositionCopiedFile, []string, error) {
	sourceDir := filepath.Join(childProject, sourceDirRel)
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []string{sourceDirRel + "/ missing in child project."}, nil
		}
		return nil, nil, err
	}
	files := []compositionCopiedFile{}
	skipped := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if extension != "" && strings.ToLower(filepath.Ext(entry.Name())) != extension {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, nil, err
		}
		if !info.Mode().IsRegular() {
			skipped = append(skipped, filepath.Join(sourceDirRel, entry.Name())+" is not a regular file.")
			continue
		}
		sourceRel := filepath.Join(sourceDirRel, entry.Name())
		destRel := filepath.Join(destDirRel, entry.Name())
		bytes, err := copyCompositionRegularFile(filepath.Join(childProject, sourceRel), filepath.Join(snapshotDir, destRel))
		if err != nil {
			return nil, nil, err
		}
		files = append(files, compositionCopiedFile{Artifact: sourceRel, DestRel: destRel, Bytes: bytes})
	}
	if len(files) == 0 && len(skipped) == 0 {
		skipped = append(skipped, sourceDirRel+"/ had no copyable files.")
	}
	return files, skipped, nil
}

func copyCompositionTreeFiles(childProject string, snapshotDir string, sourceDirRel string, destDirRel string, extension string) ([]compositionCopiedFile, []string, error) {
	sourceDir := projectAbsPath(childProject, sourceDirRel)
	info, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []string{sourceDirRel + "/ missing in child project."}, nil
		}
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, []string{sourceDirRel + "/ is not a directory."}, nil
	}
	files := []compositionCopiedFile{}
	skipped := []string{}
	err = filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if extension != "" && strings.ToLower(filepath.Ext(entry.Name())) != extension {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relFromDir, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		sourceRel := filepath.Join(sourceDirRel, relFromDir)
		if !info.Mode().IsRegular() {
			skipped = append(skipped, sourceRel+" is not a regular file.")
			return nil
		}
		destRel := filepath.Join(destDirRel, relFromDir)
		bytes, err := copyCompositionRegularFile(path, filepath.Join(snapshotDir, destRel))
		if err != nil {
			return err
		}
		files = append(files, compositionCopiedFile{Artifact: sourceRel, DestRel: destRel, Bytes: bytes})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 && len(skipped) == 0 {
		skipped = append(skipped, sourceDirRel+"/ had no copyable files.")
	}
	return files, skipped, nil
}

func copyCompositionRegularFile(sourcePath string, destPath string) (int64, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func compositionChildProjectReference(project string, item compositionFile) (string, string) {
	path := compositionFileDiskPath(project, item)
	raw, ok := explicitCompositionField(path, []string{"path", "project", "ref"})
	if !ok {
		return "", "No `path`, `project`, or `ref` field was found in the child reference."
	}
	childProject := strings.TrimSpace(raw)
	if childProject == "" {
		return "", "The child reference path is empty."
	}
	if !filepath.IsAbs(childProject) {
		childProject = filepath.Join(filepath.Dir(path), childProject)
	}
	childProject = filepath.Clean(childProject)
	info, err := os.Stat(childProject)
	if err != nil {
		if os.IsNotExist(err) {
			return childProject, "Referenced child project was not found."
		}
		return childProject, "Referenced child project could not be inspected: " + err.Error()
	}
	if !info.IsDir() {
		return childProject, "Referenced child path is not a project directory."
	}
	return childProject, ""
}

func compositionCopyCandidates(childProject string, referenceIssue string) []compositionCopyCandidate {
	if referenceIssue != "" {
		return []compositionCopyCandidate{{
			Artifact:       "child project",
			Status:         "blocked",
			Evidence:       referenceIssue,
			Recommendation: "Add a reviewed child project path before copying content into the parent.",
		}}
	}
	candidates := []compositionCopyCandidate{
		compositionCopyFileCandidate(childProject, "MIXTAPE.md", "corpus", "Copy as a parent source packet only after checking route ownership and source overlap."),
		compositionCopyFileCandidate(childProject, "LINER.md", "operating layer", "Keep as child operating guidance unless a promotion audit says the parent should own these rules."),
		compositionCopyTapeCandidate(childProject),
	}
	skillCount := countTopLevelRegularFiles(filepath.Join(childProject, "skills"), ".md")
	candidates = append(candidates, compositionCopyCountCandidate("skills/*.md", skillCount, "skill", "Merge skills only after grounding and boundary review."))
	localSourceCount := countRegularFiles(projectAbsPath(childProject, "local-sources"), "")
	candidates = append(candidates, compositionCopyCountCandidate("local-sources/", localSourceCount, "local source", "Copy local files only with provenance and source-note cleanup."))
	auditCount := countRegularFiles(filepath.Join(childProject, "working", "audits"), ".md")
	candidates = append(candidates, compositionCopyCountCandidate("working/audits/*.md", auditCount, "audit", "Keep relevant audits beside copied artifacts for lineage."))
	return candidates
}

func compositionCopyFileCandidate(childProject string, rel string, label string, recommendation string) compositionCopyCandidate {
	path := projectAbsPath(childProject, rel)
	info, err := os.Stat(path)
	if err != nil {
		status := "missing"
		evidence := rel + " was not found in the referenced child project."
		if !os.IsNotExist(err) {
			status = "unreadable"
			evidence = err.Error()
		}
		return compositionCopyCandidate{
			Artifact:       rel,
			Status:         status,
			Evidence:       evidence,
			Recommendation: "Resolve before treating the child " + label + " as copyable.",
		}
	}
	if info.IsDir() {
		return compositionCopyCandidate{
			Artifact:       rel,
			Status:         "blocked",
			Evidence:       rel + " is a directory, not a file.",
			Recommendation: "Review the child project layout before copying.",
		}
	}
	return compositionCopyCandidate{
		Artifact:       rel,
		Status:         "candidate",
		Evidence:       fmt.Sprintf("%s exists, %d bytes.", rel, info.Size()),
		Recommendation: recommendation,
	}
}

func compositionCopyTapeCandidate(childProject string) compositionCopyCandidate {
	t, err := tape.ReadProject(childProject)
	if err != nil {
		status := "missing"
		evidence := "`tape.yaml` was not found in the referenced child project."
		if !os.IsNotExist(err) {
			status = "unreadable"
			evidence = err.Error()
		}
		return compositionCopyCandidate{
			Artifact:       "tape.yaml",
			Status:         status,
			Evidence:       evidence,
			Recommendation: "Accept or repair the child tape before source copy review.",
		}
	}
	status := "candidate"
	if len(t.Sources) == 0 {
		status = "empty"
	}
	return compositionCopyCandidate{
		Artifact:       "tape.yaml",
		Status:         status,
		Evidence:       intLabel(len(t.Sources), "saved source") + " in child tape.",
		Recommendation: "Copy saved sources only through a reviewed merge that preserves notes, kind, priority, and provenance.",
	}
}

func compositionCopyCountCandidate(artifact string, count int, label string, recommendation string) compositionCopyCandidate {
	status := "candidate"
	evidence := intLabel(count, label) + " found."
	if count == 0 {
		status = "absent"
		evidence = "No " + label + " files found."
	}
	return compositionCopyCandidate{
		Artifact:       artifact,
		Status:         status,
		Evidence:       evidence,
		Recommendation: recommendation,
	}
}

func renderCompositionMergeDraft(parent string, input compositionRouteAuditInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Composition Merge Draft\n\n", parent)
	b.WriteString("Review this section before merging it into `LINER.md`. It promotes one child route into the parent operating layer by reference; no child sources are copied.\n\n")
	b.WriteString("## Promoted Child Route\n\n")
	b.WriteString("| Child | Route | Status | Reference | Explicit Route |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n",
		escapeMarkdownTable(input.Child),
		escapeMarkdownTable(input.Route),
		escapeMarkdownTable(input.Status),
		input.RelPath,
		yesNo(input.Explicit),
	)
	b.WriteString("\n## Parent Routing Update\n\n")
	fmt.Fprintf(&b, "- Treat `%s` as a promoted child route when a request matches `%s`.\n", input.Child, input.Route)
	b.WriteString("- Use the parent `LINER.md` to triage the request, then load the child reference for specialized execution.\n")
	b.WriteString("- Keep the child corpus and skills scoped to the child until a later copy/merge audit approves moving content into the parent.\n")
	b.WriteString("- If this route overlaps another child, run the route audit before applying additional merge drafts.\n\n")
	b.WriteString("## Review Notes\n\n")
	for _, line := range input.ReviewLines {
		fmt.Fprintf(&b, "- %s\n", escapeMarkdownTable(line))
	}
	return b.String()
}

func renderCompositionCopyPacket(now time.Time, input compositionRouteAuditInput, childProject string, referenceIssue string, candidates []compositionCopyCandidate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Composition Copy Packet\n\n", input.Child)
	fmt.Fprintf(&b, "Date: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This packet inventories the selected child mixtape before any content is copied into the parent. It is review-only: no child files, source files, skills, `LINER.md`, `MIXTAPE.md`, or `tape.yaml` were changed.\n\n")
	b.WriteString("## Selected Child\n\n")
	b.WriteString("| Child | Route | Status | Reference | Explicit Route | Child Project |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	projectValue := childProject
	if projectValue == "" {
		projectValue = "unresolved"
	}
	fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s | `%s` |\n",
		escapeMarkdownTable(input.Child),
		escapeMarkdownTable(input.Route),
		escapeMarkdownTable(input.Status),
		input.RelPath,
		yesNo(input.Explicit),
		escapeMarkdownTable(projectValue),
	)
	if referenceIssue != "" {
		fmt.Fprintf(&b, "\nReference issue: %s\n", referenceIssue)
	}
	b.WriteString("\n## Copy Candidates\n\n")
	b.WriteString("| Artifact | Status | Evidence | Review Recommendation |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, candidate := range candidates {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			escapeMarkdownTable(candidate.Artifact),
			escapeMarkdownTable(candidate.Status),
			escapeMarkdownTable(candidate.Evidence),
			escapeMarkdownTable(candidate.Recommendation),
		)
	}
	b.WriteString("\n## Review Rules\n\n")
	b.WriteString("- Run a route audit before copying if this child overlaps another route.\n")
	b.WriteString("- Copy child source claims only when source notes and skill boundaries survive in the parent.\n")
	b.WriteString("- Keep the child nested when its operating layer is more specific than the parent request.\n")
	b.WriteString("- Record a separate apply audit if a later flow actually copies or merges files.\n")
	return b.String()
}

func renderCompositionCopyPacketAudit(now time.Time, packetRel string, input compositionRouteAuditInput, childProject string, referenceIssue string, candidates []compositionCopyCandidate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Copy Packet Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Created a review packet for one selected child before any content copy or merge.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Packet: `%s`\n", packetRel)
	fmt.Fprintf(&b, "- Child reference: `%s`\n", input.RelPath)
	if childProject != "" {
		fmt.Fprintf(&b, "- Child project: `%s`\n", childProject)
	}
	if referenceIssue != "" {
		fmt.Fprintf(&b, "- Reference issue: %s\n", referenceIssue)
	}
	fmt.Fprintf(&b, "- Candidate artifact rows: %d\n\n", len(candidates))
	b.WriteString("## Boundaries\n\n")
	b.WriteString("- No child files, source files, skills, `LINER.md`, `MIXTAPE.md`, or `tape.yaml` were copied or changed.\n")
	b.WriteString("- The packet is an inventory for later reviewed copy/merge work, not an apply action.\n")
	b.WriteString("- Keep nesting as the default until route ownership and source overlap are audited.\n")
	return b.String()
}

func renderCompositionCopyApplyManifest(now time.Time, result compositionCopyApplyResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Composition Copy Snapshot\n\n", result.Input.Child)
	fmt.Fprintf(&b, "Date: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Child project: `%s`\n\n", result.ChildProject)
	b.WriteString("This snapshot copies selected child mixtape artifacts into the parent `working/composition/copied/` area for review. It does not promote child sources, skills, or operating rules into parent production files.\n\n")
	b.WriteString("## Copied Files\n\n")
	writeCompositionCopiedFilesTable(&b, result.Files)
	writeCompositionSkippedList(&b, result.Skipped)
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- Parent `LINER.md`, `tape.yaml`, `skills/`, and `local-sources/` were not changed.\n")
	b.WriteString("- Use this snapshot as review evidence before a later source or skill merge.\n")
	b.WriteString("- Keep the child nested if its route remains more specific than the parent.\n")
	return b.String()
}

func renderCompositionCopyApplyAudit(now time.Time, result compositionCopyApplyResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Copy Apply Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Copied the selected child mixtape into a namespaced parent working snapshot for review. This is an apply action for the copy packet, not a production merge into the parent operating layer.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Child reference: `%s`\n", result.Input.RelPath)
	fmt.Fprintf(&b, "- Child project: `%s`\n", result.ChildProject)
	fmt.Fprintf(&b, "- Snapshot: `%s`\n", result.SnapshotRel)
	fmt.Fprintf(&b, "- Route: %s\n", result.Input.Route)
	fmt.Fprintf(&b, "- Explicit route: %s\n\n", yesNo(result.Input.Explicit))
	b.WriteString("## Copied Files\n\n")
	writeCompositionCopiedFilesTable(&b, result.Files)
	writeCompositionSkippedList(&b, result.Skipped)
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- Parent `LINER.md`, `tape.yaml`, `skills/`, and `local-sources/` were not changed.\n")
	b.WriteString("- Copied child source files remain review evidence until a separate merge updates parent production files.\n")
	b.WriteString("- Run source-note, skill-corpus, and route audits before promoting copied artifacts.\n")
	return b.String()
}

func writeCompositionCopiedFilesTable(b *strings.Builder, files []compositionCopiedFile) {
	if len(files) == 0 {
		b.WriteString("No files were copied.\n\n")
		return
	}
	b.WriteString("| Child artifact | Snapshot path | Bytes |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, file := range files {
		fmt.Fprintf(b, "| %s | `%s` | %d |\n",
			escapeMarkdownTable(file.Artifact),
			escapeMarkdownTable(file.DestRel),
			file.Bytes,
		)
	}
	b.WriteString("\n")
}

func writeCompositionSkippedList(b *strings.Builder, skipped []string) {
	if len(skipped) == 0 {
		return
	}
	b.WriteString("## Skipped\n\n")
	for _, item := range skipped {
		fmt.Fprintf(b, "- %s\n", item)
	}
	b.WriteString("\n")
}

func renderCompositionMergeDraftAudit(now time.Time, input compositionRouteAuditInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Merge Draft Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Created a reviewed merge-routing draft for one selected child. This prepares a parent `LINER.md` update without copying child content or changing any operating-layer file yet.\n\n")
	b.WriteString("## Candidate\n\n")
	b.WriteString("| Child | Route | Status | Reference | Explicit Route |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n",
		escapeMarkdownTable(input.Child),
		escapeMarkdownTable(input.Route),
		escapeMarkdownTable(input.Status),
		input.RelPath,
		yesNo(input.Explicit),
	)
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- `LINER.md` is unchanged until the draft is accepted in Review Composition Draft.\n")
	b.WriteString("- No child sources, skills, or files were copied into the parent.\n")
	b.WriteString("- Run a route audit first if this child overlaps another child route.\n")
	return b.String()
}

func writeCompositionApplyAudit(project string, backupRel string, draftBytes int) error {
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	now := time.Now()
	path := filepath.Join(dir, now.Format("2006-01-02-150405")+"-composition-apply.md")
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Routing Apply Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Decision\n\n")
	b.WriteString("Applied the reviewed composition routing draft to `LINER.md` between managed composition markers.\n\n")
	b.WriteString("## Inputs\n\n")
	fmt.Fprintf(&b, "- Draft: `%s`\n", compositionDraftRelPath)
	fmt.Fprintf(&b, "- Draft bytes: %d\n", draftBytes)
	if backupRel != "" {
		fmt.Fprintf(&b, "- Previous `LINER.md` backup: `%s`\n", backupRel)
	} else {
		b.WriteString("- Previous `LINER.md`: not present; created a new operating layer containing the composition routing section.\n")
	}
	b.WriteString("\n## Boundaries\n\n")
	b.WriteString("- No child sources were copied into the parent.\n")
	b.WriteString("- Existing child references in `children/` remain the source of truth.\n")
	b.WriteString("- Future edits should update the composition draft, then apply again through review.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeCompositionRouteAudit(project string, items []compositionFile) (auditFile, error) {
	inputs, err := compositionRouteAuditInputs(project, items)
	if err != nil {
		return auditFile{}, err
	}
	if len(inputs) == 0 {
		return auditFile{}, fmt.Errorf("Add at least one child reference before running a composition route audit.")
	}
	findings := compositionRouteFindings(inputs)
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-composition-route-audit"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderCompositionRouteAudit(now, inputs, findings)), 0o644); err != nil {
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

func writeCompositionPromotionAudit(project string, item compositionFile) (auditFile, error) {
	input, err := compositionPromotionInput(project, item)
	if err != nil {
		return auditFile{}, err
	}
	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-composition-promotion-readiness"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderCompositionPromotionAudit(now, input)), 0o644); err != nil {
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

func compositionPromotionInput(project string, item compositionFile) (compositionRouteAuditInput, error) {
	if item.Kind == "lineage" {
		return compositionRouteAuditInput{}, fmt.Errorf("Select a child reference before checking promotion readiness.")
	}
	path := compositionFileDiskPath(project, item)
	if _, err := os.Stat(path); err != nil {
		return compositionRouteAuditInput{}, err
	}
	route, explicit := explicitCompositionRoute(path)
	if !explicit {
		route = strings.ReplaceAll(compositionChildName(item), "-", ", ")
	}
	status := item.Status
	if strings.TrimSpace(status) == "" {
		status = compositionStatus(path)
	}
	return compositionRouteAuditInput{
		Child:       compositionChildName(item),
		RelPath:     item.RelPath,
		Route:       route,
		Status:      status,
		Explicit:    explicit,
		ReviewLines: compositionReviewLines(path),
	}, nil
}

func compositionRouteAuditInputs(project string, items []compositionFile) ([]compositionRouteAuditInput, error) {
	inputs := []compositionRouteAuditInput{}
	for _, item := range items {
		if item.Kind == "lineage" {
			continue
		}
		path := compositionFileDiskPath(project, item)
		route, explicit := explicitCompositionRoute(path)
		if !explicit {
			route = strings.ReplaceAll(compositionChildName(item), "-", ", ")
		}
		inputs = append(inputs, compositionRouteAuditInput{
			Child:       compositionChildName(item),
			RelPath:     item.RelPath,
			Route:       route,
			Status:      item.Status,
			Explicit:    explicit,
			ReviewLines: compositionReviewLines(path),
		})
	}
	sort.Slice(inputs, func(i int, j int) bool {
		return strings.ToLower(inputs[i].Child) < strings.ToLower(inputs[j].Child)
	})
	return inputs, nil
}

func compositionFileDiskPath(project string, item compositionFile) string {
	if strings.TrimSpace(item.Path) != "" {
		return item.Path
	}
	return filepath.Join(project, item.RelPath)
}

func compositionReviewLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{"Could not read file: " + err.Error()}
	}
	lines := []string{}
	for index, line := range strings.Split(string(data), "\n") {
		clean := strings.TrimSpace(line)
		if clean == "" {
			continue
		}
		if containsAny(strings.ToLower(clean), []string{"warning", "conflict", "contradiction", "overlap", "deprecated", "disabled"}) {
			lines = append(lines, fmt.Sprintf("line %d: %s", index+1, clean))
		}
	}
	if len(lines) > 3 {
		return lines[:3]
	}
	return lines
}

func compositionRouteFindings(inputs []compositionRouteAuditInput) []compositionRouteFinding {
	findings := []compositionRouteFinding{}
	for _, input := range inputs {
		if !input.Explicit {
			findings = append(findings, compositionRouteFinding{
				Severity:       "low",
				Children:       input.Child,
				Evidence:       "`" + input.RelPath + "` has no explicit route, scope, focus, or tags field.",
				Reason:         "The parent will infer a route from the filename, which can hide intent as the tree grows.",
				Recommendation: "Add an explicit route before promoting or merging this child.",
			})
		}
		switch input.Status {
		case "review", "disabled", "empty", "unreadable":
			findings = append(findings, compositionRouteFinding{
				Severity:       compositionStatusSeverity(input.Status),
				Children:       input.Child,
				Evidence:       fmt.Sprintf("Status is `%s` for `%s`.", input.Status, input.RelPath),
				Reason:         "A child with this status should not silently become part of the parent operating route.",
				Recommendation: "Review the child reference before applying nested routing.",
			})
		}
		for _, line := range input.ReviewLines {
			reason := "The child file names a warning, overlap, or conflict that should become an explicit routing decision."
			recommendation := "Keep the child nested until the conflict is documented in an audit or resolved in the child."
			if strings.HasPrefix(line, "Could not read file:") {
				reason = "The child file could not be inspected, so the parent cannot safely route work to it."
				recommendation = "Fix the child reference or file permissions before applying nested routing."
			}
			findings = append(findings, compositionRouteFinding{
				Severity:       "high",
				Children:       input.Child,
				Evidence:       line,
				Reason:         reason,
				Recommendation: recommendation,
			})
		}
	}
	for _, finding := range compositionRouteOverlapFindings(inputs) {
		findings = append(findings, finding)
	}
	sort.SliceStable(findings, func(i int, j int) bool {
		return severityRank(findings[i].Severity) > severityRank(findings[j].Severity)
	})
	if len(findings) > 40 {
		return findings[:40]
	}
	return findings
}

func compositionRouteOverlapFindings(inputs []compositionRouteAuditInput) []compositionRouteFinding {
	owners := map[string]map[string]bool{}
	for _, input := range inputs {
		for _, token := range compositionRouteTokens(input.Route) {
			if owners[token] == nil {
				owners[token] = map[string]bool{}
			}
			owners[token][input.Child] = true
		}
	}
	tokens := make([]string, 0, len(owners))
	for token, children := range owners {
		if len(children) > 1 {
			tokens = append(tokens, token)
		}
	}
	sort.Strings(tokens)
	findings := []compositionRouteFinding{}
	for _, token := range tokens {
		children := mapKeys(owners[token])
		sort.Strings(children)
		findings = append(findings, compositionRouteFinding{
			Severity:       "medium",
			Children:       strings.Join(children, ", "),
			Evidence:       "Shared route token `" + token + "`.",
			Reason:         "Multiple children appear eligible for the same request language.",
			Recommendation: "Add a parent routing rule that chooses the narrowest child, or split the shared route into clearer scopes.",
		})
	}
	return findings
}

func compositionRouteTokens(route string) []string {
	replacer := strings.NewReplacer(",", " ", "/", " ", "&", " ", "+", " ", ";", " ", ":", " ", "(", " ", ")", " ")
	parts := strings.Fields(strings.ToLower(replacer.Replace(route)))
	tokens := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		token := strings.Trim(part, " ._-`'\"")
		if token == "" || compositionStopRouteToken(token) || seen[token] {
			continue
		}
		seen[token] = true
		tokens = append(tokens, token)
	}
	return tokens
}

func compositionStopRouteToken(token string) bool {
	if len(token) < 3 && token != "ai" && token != "ia" && token != "ui" && token != "ux" {
		return true
	}
	switch token {
	case "and", "the", "for", "from", "into", "with", "without", "child", "children", "specialist", "mixtape", "mixtapes", "route", "routes":
		return true
	default:
		return false
	}
}

func compositionStatusSeverity(status string) string {
	switch status {
	case "review", "unreadable":
		return "high"
	case "disabled", "empty":
		return "medium"
	default:
		return "low"
	}
}

func severityRank(severity string) int {
	switch severity {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func renderCompositionRouteAudit(now time.Time, inputs []compositionRouteAuditInput, findings []compositionRouteFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Route Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local audit checks child mixtape routes for overlap, missing route declarations, and child files that already name warnings or conflicts. It does not rewrite any project files.\n\n")
	b.WriteString("## Children\n\n")
	b.WriteString("| Child | Route | Status | Reference |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, input := range inputs {
		route := input.Route
		if !input.Explicit {
			route += " (inferred)"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` |\n",
			escapeMarkdownTable(input.Child),
			escapeMarkdownTable(route),
			escapeMarkdownTable(input.Status),
			input.RelPath,
		)
	}
	b.WriteString("\n## Findings\n\n")
	if len(findings) == 0 {
		b.WriteString("No obvious route overlaps or child-status warnings were found by the local heuristic.\n\n")
	} else {
		b.WriteString("| Severity | Child or overlap | Evidence | Reason | Recommendation |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, finding := range findings {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				escapeMarkdownTable(finding.Severity),
				escapeMarkdownTable(finding.Children),
				escapeMarkdownTable(finding.Evidence),
				escapeMarkdownTable(finding.Reason),
				escapeMarkdownTable(finding.Recommendation),
			)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Recommended Review\n\n")
	b.WriteString("- Keep nesting as the default when child scopes overlap or require specialized judgment.\n")
	b.WriteString("- Promote or merge only after the audit explains why one child owns the shared route.\n")
	b.WriteString("- If a child file names a conflict, document the chosen rule before applying or regenerating parent routing.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No child files, sources, or operating-layer files were changed by this audit.\n")
	b.WriteString("- Apply decisions only after reviewing the findings above.\n")
	return b.String()
}

func renderCompositionPromotionAudit(now time.Time, input compositionRouteAuditInput) string {
	recommendations := compositionPromotionRecommendations(input)
	readiness := compositionPromotionReadiness(input)
	var b strings.Builder
	fmt.Fprintf(&b, "# Composition Promotion Readiness\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local audit checks whether one selected child mixtape is ready to be promoted or considered for merge. It does not rewrite `LINER.md`, `lineage.yaml`, child files, or source files.\n\n")
	b.WriteString("## Selected Child\n\n")
	b.WriteString("| Child | Route | Status | Reference | Route source |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	routeSource := "explicit"
	route := input.Route
	if !input.Explicit {
		routeSource = "inferred"
		route += " (inferred)"
	}
	fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n\n",
		escapeMarkdownTable(input.Child),
		escapeMarkdownTable(route),
		escapeMarkdownTable(input.Status),
		input.RelPath,
		routeSource,
	)
	b.WriteString("## Readiness\n\n")
	fmt.Fprintf(&b, "- %s\n\n", readiness)
	b.WriteString("## Review Signals\n\n")
	if len(input.ReviewLines) == 0 {
		b.WriteString("- No local warning, conflict, overlap, disabled, or deprecated language was found in the child reference.\n\n")
	} else {
		for _, line := range input.ReviewLines {
			fmt.Fprintf(&b, "- %s\n", line)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Recommendations\n\n")
	for _, recommendation := range recommendations {
		fmt.Fprintf(&b, "- %s\n", recommendation)
	}
	b.WriteString("\n## Decision Log\n\n")
	b.WriteString("- No child content was copied into the parent by this audit.\n")
	b.WriteString("- Keep nesting as the default unless the reviewed evidence says the parent should own this route directly.\n")
	b.WriteString("- If promotion is accepted later, update `LINER.md` through a reviewed composition draft and record a separate apply audit.\n")
	return b.String()
}

func compositionPromotionReadiness(input compositionRouteAuditInput) string {
	if len(input.ReviewLines) > 0 {
		return "keep nested until named warnings or conflicts are resolved"
	}
	if input.Status == "review" || input.Status == "disabled" || input.Status == "empty" || input.Status == "unreadable" {
		return "keep nested until child status is resolved"
	}
	if !input.Explicit {
		return "needs route review before promotion"
	}
	return "candidate for promotion review"
}

func compositionPromotionRecommendations(input compositionRouteAuditInput) []string {
	recommendations := []string{}
	if !input.Explicit {
		recommendations = append(recommendations, "Add an explicit route, scope, focus, or tags field before promotion.")
	}
	if input.Status != "ready" && input.Status != "reference" {
		recommendations = append(recommendations, "Resolve the child status before the parent routes work to it directly.")
	}
	if len(input.ReviewLines) > 0 {
		recommendations = append(recommendations, "Resolve or document the child warnings before promotion or merge.")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Promotion can be reviewed if the parent should answer this route directly instead of delegating to the child.")
	}
	recommendations = append(recommendations, "Do not merge child source claims into the parent until source sets are audited as small and non-overlapping.")
	return recommendations
}
