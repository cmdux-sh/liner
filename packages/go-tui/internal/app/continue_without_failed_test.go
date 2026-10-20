package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func partialCompileProject(t *testing.T) (string, tape.Tape) {
	t.Helper()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\nmixtape: mixtape\nstatus:\n  milestone: corpus_ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	usable := "https://example.test/usable"
	blocked := "https://example.test/blocked"
	current := tape.Tape{
		Title:   "Synthetic",
		Sources: []tape.Source{{Type: "web", URL: usable}, {Type: "web", URL: blocked}},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectAbsPath(project, "sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"synthesis.md":                 "# Synthesis\n\nReady.\n",
		"MIXTAPE.md":                   "# Synthetic\n\n## Compilation notes\n\n- **" + blocked + "** — Failed to fetch " + blocked + " — status: HTTP 403; the source blocked access.\n",
		"sources/01-usable.md":         "# Usable\n\nReadable text from the usable Source.\n",
		"sources/02-blocked-source.md": "# Missing\n\n_Source unavailable. See compilation notes in MIXTAPE.md._\n",
	}
	for rel, body := range files {
		if err := os.WriteFile(projectAbsPath(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return project, current
}

func TestCreateLayerFromProjectContinuesWithoutReviewedFailedSources(t *testing.T) {
	project, current := partialCompileProject(t)
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      30,
		currentPath: project,
		currentTape: current,
		help:        help.New(),
		compileBar:  newCompileProgress(48),
		statusPath:  project,
		status:      &core.ProjectStatus{Snapshot: core.StatusSnapshot{Milestone: "corpus_ready"}},
	}
	m.help.SetWidth(96)
	if !m.projectCompileNeedsAttention() {
		t.Fatal("fixture should have a failed Source")
	}

	review, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if review.err != "" || review.screen != screenCompile || review.compilePane != compilePaneSources {
		t.Fatalf("l should show the failed Sources rather than refuse: screen=%v pane=%d err=%q", review.screen, review.compilePane, review.err)
	}
	if !strings.Contains(review.note, "1 of 2 Sources are usable") || !strings.Contains(review.note, "press l to create the Operating Layer without them") {
		t.Fatalf("the review should say how to continue, got %q", review.note)
	}
	for _, row := range review.compileRows {
		if row.Status == "queued" {
			t.Fatalf("Sources compiled earlier should not read as queued: %#v", review.compileRows)
		}
	}
	if !hasHelpDesc(review.helpForScreen().ShortHelp(), "layer without failed") {
		t.Fatalf("every key the path needs should be listed: %#v", review.helpForScreen().ShortHelp())
	}
	if !hasHelpDesc(review.helpForScreen().ShortHelp(), "retry compile") {
		t.Fatalf("retry should stay listed beside continue: %#v", review.helpForScreen().ShortHelp())
	}

	layer, _ := review.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if layer.err != "" || layer.screen != screenLinerReview {
		t.Fatalf("l on the reviewed failures should continue to the Operating Layer: screen=%v err=%q", layer.screen, layer.err)
	}

	// Back on the Project screen in the same session, the review counts.
	layer.screen = screenProject
	again, _ := layer.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if again.err != "" || again.screen != screenLinerReview {
		t.Fatalf("after reviewing, l on the Project screen should continue directly: screen=%v err=%q", again.screen, again.err)
	}
}

func TestCompileContinueWithoutFailedWaitsForReviewAndUsableResult(t *testing.T) {
	m := Model{
		screen:      screenCompile,
		width:       100,
		height:      30,
		compilePane: compilePaneIssues,
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 2, Succeeded: 1, Failed: 1},
			Warnings:    []core.CompileWarningPayload{{URL: "https://example.test/blocked", Message: "The source blocked access.", Severity: "error"}},
		},
	}
	if hasHelpDesc(m.helpForScreen().ShortHelp(), "layer without failed") {
		t.Fatal("continue without failed Sources should appear only after the person has seen them")
	}
	shown, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if shown.compilePane != compilePaneSources || shown.screen != screenCompile {
		t.Fatalf("l before review should show the failed Sources first, got pane %d screen %v", shown.compilePane, shown.screen)
	}

	none := m
	none.compileResult = &core.CompileResultPayload{Summary: core.CompileSummary{Total: 1, Failed: 1}, Warnings: m.compileResult.Warnings}
	none.compileSourcesReviewed = true
	blocked, _ := none.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if !strings.Contains(blocked.err, "No usable MIXTAPE.md yet") {
		t.Fatalf("without usable Sources, l should explain, got %q", blocked.err)
	}
}

func TestCreateLayerShortcutAfterReviewStillNeedsUsableSources(t *testing.T) {
	project, current := partialCompileProject(t)
	// Every Source failed: the usable one is a stub too.
	if err := os.WriteFile(projectAbsPath(project, "sources/01-usable.md"), []byte("# Missing\n\n_Source unavailable. See compilation notes in MIXTAPE.md._\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      30,
		currentPath: project,
		currentTape: current,
		help:        help.New(),
		compileBar:  newCompileProgress(48),
		statusPath:  project,
		status:      &core.ProjectStatus{Snapshot: core.StatusSnapshot{Milestone: "corpus_ready"}},
	}
	review, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if review.compileHasUsableResult() {
		t.Fatal("fixture should have no usable Source")
	}
	review.screen = screenProject // Escape back to the Project screen.
	again, _ := review.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if again.screen == screenLinerReview || !strings.Contains(again.err, "No usable MIXTAPE.md yet") {
		t.Fatalf("a reviewed but unusable result must not create the Operating Layer: screen=%v err=%q", again.screen, again.err)
	}
}

func TestCreateLayerShortcutForgetsReviewWhenTheCompileChanges(t *testing.T) {
	project, current := partialCompileProject(t)
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      30,
		currentPath: project,
		currentTape: current,
		help:        help.New(),
		compileBar:  newCompileProgress(48),
		statusPath:  project,
		status:      &core.ProjectStatus{Snapshot: core.StatusSnapshot{Milestone: "corpus_ready"}},
	}
	review, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	review.screen = screenProject

	// A later compile (outside this screen) rewrites MIXTAPE.md.
	mixtape := projectAbsPath(project, "MIXTAPE.md")
	body, err := os.ReadFile(mixtape)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mixtape, append(body, []byte("\n- **https://example.test/new** — Failed to fetch.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	again, _ := review.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if again.screen != screenCompile || again.compilePane != compilePaneSources {
		t.Fatalf("after the compile changes, l should show the failed Sources again: screen=%v pane=%d", again.screen, again.compilePane)
	}
}
