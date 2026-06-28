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

func TestRunCompositionSkillConflictReviewWritesAudit(t *testing.T) {
	project := t.TempDir()
	childProject := filepath.Join(project, "ux-specialist")
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{
		filepath.Join(project, "skills"),
		filepath.Join(childProject, "skills"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, "children", "ux-specialist.yaml"), []byte("path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(childProject, "skills", "critique.md"):          "# Child Critique\n",
		filepath.Join(childProject, "skills", "flow.md"):              "# Child Flow\n",
		filepath.Join(childProject, "skills", "journey.md"):           "# Child Journey\n",
		filepath.Join(project, "skills", "ux-specialist-critique.md"): "# Parent Critique\n",
		filepath.Join(project, "skills", "flow.md"):                   "# Parent Flow\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "composition-skill-conflicts") {
		t.Fatalf("expected skill conflict audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-skill-conflicts.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill conflict audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(audit)
	for _, expected := range []string{
		"# Composition Skill Conflict Review",
		"skills/critique.md",
		"skills/ux-specialist-critique.md",
		"conflict",
		"different skill at the production merge target",
		"skills/flow.md",
		"overlap",
		"skills/journey.md",
		"clear",
		"No parent or child skill files were changed",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("skill conflict audit missing %q:\n%s", expected, body)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "ux-specialist-journey.md")); !os.IsNotExist(err) {
		t.Fatalf("skill conflict review should not copy clear child skills, stat err=%v", err)
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing for skill conflict review, got %#v", items)
	}
}

func TestRunCompositionSkillConflictReviewRequiresChildSelection(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "lineage.yaml"), []byte("parent: Product Design\n"), 0o644); err != nil {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))

	if got.err != "Select a child reference before reviewing parent skill conflicts." {
		t.Fatalf("expected selected-child skill conflict guard, got %q", got.err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-skill-conflicts.md")); err != nil || len(matches) != 0 {
		t.Fatalf("skill conflict review should not create audit for lineage, got %#v err=%v", matches, err)
	}
}
