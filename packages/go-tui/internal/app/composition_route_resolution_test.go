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

func TestCreateCompositionRouteResolutionDraftFromOverlaps(t *testing.T) {
	project := t.TempDir()
	paths := map[string]string{
		filepath.Join("children", "ux-specialist.yaml"):       "route: research, flows, IA\nstatus: ready\n",
		filepath.Join("children", "research-specialist.yaml"): "route: research, interviews\nstatus: ready\n",
	}
	for rel, body := range paths {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Parent\n\nExisting operating rules.\n"), 0o644); err != nil {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenCompositionReview {
		t.Fatalf("expected composition review after route resolution draft, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != compositionDraftRelPath {
		t.Fatalf("expected route resolution draft preview, got %q", got.previewRel)
	}
	draft, err := os.ReadFile(filepath.Join(project, compositionDraftRelPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Product Design Route Conflict Resolution Draft",
		"Child Route Map",
		"Shared Route Conflicts",
		"research",
		"ux-specialist",
		"research-specialist",
		"Ask one clarifying question",
		"Route to the narrowest child",
		"Do not copy child sources",
	} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("route resolution draft missing %q:\n%s", expected, string(draft))
		}
	}
	parent, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(parent) != "# Parent\n\nExisting operating rules.\n" {
		t.Fatalf("route resolution draft should not mutate parent LINER.md before review:\n%s", parent)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-route-resolution.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one route resolution audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Composition Route Resolution Audit", "Shared route conflicts found: 1", "`research`: research-specialist, ux-specialist", "Parent `LINER.md` is unchanged"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("route resolution audit missing %q:\n%s", expected, string(audit))
		}
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing for route resolution, got %#v", items)
	}
}

func TestCreateCompositionRouteResolutionRequiresChildReferences(t *testing.T) {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if !strings.Contains(got.err, "Add at least one child reference") {
		t.Fatalf("expected missing child route-resolution guard, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, compositionDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("route resolution draft should not be created without children, stat err=%v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-route-resolution.md")); err != nil || len(matches) != 0 {
		t.Fatalf("route resolution audit should not be created without children, got %#v err=%v", matches, err)
	}
}
