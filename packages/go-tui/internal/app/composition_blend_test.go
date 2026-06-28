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

func TestCreateCompositionLinerBlendDraftReadsChildLiner(t *testing.T) {
	project := t.TempDir()
	childProject := filepath.Join(project, "ux-specialist")
	childRel := filepath.Join("children", "ux-specialist.yaml")
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(childProject, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, childRel), []byte("path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	childLiner := `# UX Specialist Operating Layer

## Scope

- Must use interview evidence before proposing flows.
- Avoid visual polish decisions unless the UI specialist route is delegated.
- When research and IA conflict, prefer the latest validated user journey.
`
	if err := os.WriteFile(filepath.Join(childProject, "LINER.md"), []byte(childLiner), 0o644); err != nil {
		t.Fatal(err)
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'b', Text: "b"}))

	if got.screen != screenCompositionReview {
		t.Fatalf("expected composition review after child LINER blend, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != compositionDraftRelPath {
		t.Fatalf("expected blend draft preview, got %q", got.previewRel)
	}
	draft, err := os.ReadFile(filepath.Join(project, compositionDraftRelPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Product Design Child LINER Blend Draft",
		"Child Operating Signals",
		"UX Specialist Operating Layer",
		"Must use interview evidence",
		"Avoid visual polish decisions",
		"load the parent `LINER.md` first",
		"Do not copy child source claims",
		filepath.ToSlash(childRel),
	} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("child LINER blend draft missing %q:\n%s", expected, string(draft))
		}
	}
	parent, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(parent) != "# Parent\n\nExisting operating rules.\n" {
		t.Fatalf("blend draft should not mutate parent LINER.md before review:\n%s", parent)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-liner-blend.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one child LINER blend audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Composition LINER Blend Audit",
		"reviewed child `LINER.md` blend draft",
		"Parent `LINER.md` is unchanged",
		"Extracted operating signals",
	} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("child LINER blend audit missing %q:\n%s", expected, string(audit))
		}
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing for child LINER blend, got %#v", items)
	}
}

func TestCreateCompositionLinerBlendRequiresChildLiner(t *testing.T) {
	project := t.TempDir()
	childProject := filepath.Join(project, "ux-specialist")
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(childProject, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "children", "ux-specialist.yaml"), []byte("path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"), 0o644); err != nil {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'b', Text: "b"}))

	if !strings.Contains(got.err, "LINER.md is missing") {
		t.Fatalf("expected missing child LINER guard, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, compositionDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("blend draft should not be created without child LINER.md, stat err=%v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-liner-blend.md")); err != nil || len(matches) != 0 {
		t.Fatalf("blend audit should not be created without child LINER.md, got %#v err=%v", matches, err)
	}
}
