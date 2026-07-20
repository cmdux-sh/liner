package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func TestMergeCompositionChildPromotesSourcesAndSkillsWithAudit(t *testing.T) {
	project := t.TempDir()
	childProject := filepath.Join(project, "ux-specialist")
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{
		filepath.Join(childProject, "skills"),
		filepath.Join(childProject, "local-sources"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	duplicateURL := "https://example.com/shared"
	childRel := filepath.Join("children", "ux-specialist.yaml")
	if err := os.WriteFile(filepath.Join(project, childRel), []byte("path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tape.WriteProject(project, tape.Tape{
		Title: "Product Design",
		Sources: []tape.Source{{
			Type: "web",
			URL:  duplicateURL,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	childLocalRel := "local-sources/interview.md"
	childNote := "Use this interview for UX research patterns."
	if err := tape.WriteProject(childProject, tape.Tape{
		Title: "UX Specialist",
		Sources: []tape.Source{
			{Type: "web", URL: duplicateURL},
			{Type: "web", URL: "https://example.com/new-ux"},
			{Type: "local_file", Path: &childLocalRel, Note: &childNote},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, childLocalRel), []byte("# Interview\n\nResearch notes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "skills", "ux-research.md"), []byte("# UX Research\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "skills", "critique.md"), []byte("# UX Critique Child\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "ux-specialist-critique.md"), []byte("# Existing Parent Critique\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:           screenProject,
		width:            110,
		height:           34,
		currentPath:      project,
		currentTape:      tape.Tape{Title: "Product Design"},
		compositionTable: newCompositionTable(110, 8),
		preview:          viewport.New(viewport.WithWidth(90), viewport.WithHeight(12)),
	}
	got, _ := m.startComposition()

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	if !strings.Contains(got.err, coreWriterRemediation) {
		t.Fatalf("expected Core writer refusal, got %q", got.err)
	}
	unchanged, err := tape.ReadProject(project)
	if err != nil || len(unchanged.Sources) != 1 {
		t.Fatalf("legacy refusal must preserve parent tape, tape=%#v err=%v", unchanged, err)
	}
	if strings.Contains(got.err, coreWriterRemediation) {
		return
	}

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "composition-production-merge") {
		t.Fatalf("expected production merge audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	updated, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Sources) != 3 {
		t.Fatalf("expected duplicate skipped and two child sources added, got %#v", updated.Sources)
	}
	if len(got.currentTape.Sources) != len(updated.Sources) {
		t.Fatalf("model tape was not refreshed after merge: model=%d disk=%d", len(got.currentTape.Sources), len(updated.Sources))
	}
	if !tapeHasURL(updated, "https://example.com/new-ux") {
		t.Fatalf("expected new child web source in parent tape: %#v", updated.Sources)
	}
	local := sourceByType(updated, "local_file")
	if local.Path == nil || *local.Path != filepath.Join("local-sources", "composition", "ux-specialist", "interview.md") {
		t.Fatalf("expected rebased local source path, got %#v", local.Path)
	}
	if local.Note == nil || !strings.Contains(*local.Note, "Merged from child `ux-specialist`") {
		t.Fatalf("expected provenance note on rebased local source, got %#v", local.Note)
	}
	if _, err := os.Stat(filepath.Join(project, *local.Path)); err != nil {
		t.Fatalf("expected copied local source file, err=%v", err)
	}
	if body, err := os.ReadFile(filepath.Join(project, "skills", "ux-specialist-ux-research.md")); err != nil || string(body) != "# UX Research\n" {
		t.Fatalf("expected namespaced child skill copy, body=%q err=%v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(project, "skills", "ux-specialist-critique.md")); err != nil || string(body) != "# Existing Parent Critique\n" {
		t.Fatalf("expected conflicting parent skill to remain unchanged, body=%q err=%v", body, err)
	}
	backups, err := filepath.Glob(filepath.Join(project, "working", "composition", "*-previous-tape.yaml"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one parent tape backup, got %#v err=%v", backups, err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-production-merge.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one production merge audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Composition Production Merge Audit",
		"duplicate source already exists",
		filepath.Join("local-sources", "composition", "ux-specialist", "interview.md"),
		"skills/ux-specialist-ux-research.md",
		"parent has a different skill",
		"Previous parent `tape.yaml` backup",
	} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("production merge audit missing %q:\n%s", expected, string(audit))
		}
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing for production merge, got %#v", items)
	}
}

func TestMergeCompositionChildRequiresChildSelection(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "lineage.yaml"), []byte("parent: Product Design\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tape.WriteProject(project, tape.Tape{Title: "Product Design"}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:           screenProject,
		width:            100,
		height:           32,
		currentPath:      project,
		currentTape:      tape.Tape{Title: "Product Design"},
		compositionTable: newCompositionTable(100, 8),
	}
	got, _ := m.startComposition()

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))

	if got.err != "Select a child reference before merging child production artifacts." {
		t.Fatalf("expected selected-child production merge guard, got %q", got.err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-production-merge.md")); err != nil || len(matches) != 0 {
		t.Fatalf("production merge should not create audit for lineage, got %#v err=%v", matches, err)
	}
}

func tapeHasURL(tapeFile tape.Tape, url string) bool {
	for _, src := range tapeFile.Sources {
		if src.URL == url {
			return true
		}
	}
	return false
}

func sourceByType(tapeFile tape.Tape, sourceType string) tape.Source {
	for _, src := range tapeFile.Sources {
		if src.Type == sourceType {
			return src
		}
	}
	return tape.Source{}
}
