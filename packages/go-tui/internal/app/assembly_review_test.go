package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func writeAssemblyReviewFixture(t *testing.T, project string, draft string, evaluation string) {
	t.Helper()
	for rel, body := range map[string]string{
		assemblyDraftRelPath:         draft,
		"working/03-evaluation.yaml": evaluation,
	} {
		path := projectAbsPath(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAssemblyReviewShowsDraftNoteForAttachedSourceWithoutChangingIt(t *testing.T) {
	project := t.TempDir()
	attachedID := "src_attached"
	attached := tape.Source{ID: &attachedID, Type: "web", URL: "https://attached.example.test/app", Priority: "required"}
	draftNote := "Evaluation retrieved only metadata because the page requires JavaScript."
	writeAssemblyReviewFixture(t, project, `sources:
  - type: web
    url: https://attached.example.test/app
    priority: required
    section: guidance
    note: `+draftNote+`
  - type: web
    url: https://research.example.test/study
    priority: required
    note: Recovery exposed only partial article text.
`, `candidates:
  - url: https://attached.example.test/app
    decision: kept
    fetch_status: metadata_only
  - url: https://research.example.test/study
    decision: kept
    fetch_status: partial
`)
	m := Model{
		currentPath: project,
		currentTape: tape.Tape{Title: "Synthetic", Sources: []tape.Source{attached}},
		sourceTable: newSourceTable(100, 8),
	}

	got, _ := m.startAssemblyReview()
	if got.err != "" {
		t.Fatalf("assembly review failed: %s", got.err)
	}
	if len(got.sourceItems) != 2 {
		t.Fatalf("expected two draft sources, got %#v", got.sourceItems)
	}
	saved := got.sourceItems[0]
	if saved.Source.ID == nil || *saved.Source.ID != attachedID || saved.Source.Note != nil {
		t.Fatalf("the attached Source sent to Core must stay the saved copy, got %#v", saved.Source)
	}
	if saved.DraftNote != draftNote {
		t.Fatalf("attached Source should carry the draft's note for review, got %q", saved.DraftNote)
	}
	if saved.FetchStatus != "metadata_only" || got.sourceItems[1].FetchStatus != "partial" {
		t.Fatalf("review rows should carry the evaluation's fetch_status, got %q and %q", saved.FetchStatus, got.sourceItems[1].FetchStatus)
	}
	if payload := sourceMaintenancePayload(saved.Source); payload["note"] != nil {
		t.Fatalf("the draft note must not reach Core's payload for a saved Source, got %#v", payload)
	}

	rows := assemblyRows(got.sourceItems, 96, 0)
	if rows[0][4] != "metadata" || rows[1][4] != "partial" {
		t.Fatalf("table should mark how much the agents could read, got %q and %q", rows[0][4], rows[1][4])
	}
	if strings.Contains(rows[0][5], "no note") {
		t.Fatalf("attached Source should not show no note, got %q", rows[0][5])
	}
}

func TestAssemblyReviewShowsFullSelectedNoteAt100x30(t *testing.T) {
	items := make([]tape.Source, 0, 24)
	longNote := "Evaluation retrieved only metadata because the page requires JavaScript; the agents could not read the guidance text itself, so this Source supports the claim only through its title and the publisher's summary until a rendered copy is attached."
	for index := range 24 {
		note := fmt.Sprintf("Short note for source %02d.", index+1)
		if index == 0 {
			note = longNote
		}
		section := "guidance"
		items = append(items, tape.Source{Type: "web", URL: fmt.Sprintf("https://example.test/source-%02d", index+1), Note: &note, Section: &section, Priority: "required"})
	}
	m := Model{
		screen:      screenAssemblyReview,
		width:       100,
		height:      30,
		help:        help.New(),
		sourceTable: newSourceTable(100, 8),
		note:        "Loaded proposed sources for review.",
		currentTape: tape.Tape{Title: "Synthetic"},
	}
	m.help.SetWidth(96)
	m.sourceItems = source.Stage(items, true)
	m.sourceItems[0].FetchStatus = "metadata_only"
	m.applySourceItems(m.sourceItems)
	// In the real flow Core's plan is ready as soon as the review opens.
	m.sourceMaintenancePlan = &core.ProjectChangeSet{Operations: []map[string]any{{"type": "source.add"}}}
	m.note = "Review the checked Sources. Press Enter to accept this selection."

	rendered := m.View().Content
	if got := lipgloss.Height(rendered); got > 30 {
		t.Fatalf("review should fit 100 x 30, got %d lines", got)
	}
	plain := stripANSICodesForTest(rendered)
	if strings.Contains(plain, "more content hidden") {
		t.Fatalf("the selected note should not be cut off by the frame:\n%s", plain)
	}
	flattened := strings.Join(strings.Fields(plain), " ")
	for _, expected := range []string{"Selected", "Read: Metadata only", "Note: " + longNote, "Ready to accept Sources", "Press Enter to accept 24 checked Sources"} {
		if !strings.Contains(flattened, expected) {
			t.Fatalf("100 x 30 review missing %q:\n%s", expected, plain)
		}
	}
	for _, line := range strings.Split(plain, "\n") {
		if lipgloss.Width(line) > 100 {
			t.Fatalf("line wider than the terminal: %q", line)
		}
	}
}

func TestAssemblyReviewShowsDraftSectionAndKindForAttachedSourceWithoutChangingIt(t *testing.T) {
	project := t.TempDir()
	attachedID := "src_attached"
	savedSectionID := "src_saved_section"
	savedSection := "saved-section"
	attached := tape.Source{ID: &attachedID, Type: "web", URL: "https://attached.example.test/app", Priority: "required"}
	sectioned := tape.Source{ID: &savedSectionID, Type: "web", URL: "https://sectioned.example.test/page", Priority: "required", Section: &savedSection}
	writeAssemblyReviewFixture(t, project, `sources:
  - type: web
    url: https://attached.example.test/app
    priority: required
    kind: guide
    section: guidance
    note: Agents placed this Source under guidance.
  - type: web
    url: https://sectioned.example.test/page
    priority: required
    kind: study
    section: draft-section
`, "candidates: []\n")
	m := Model{
		screen:      screenAssemblyReview,
		width:       100,
		height:      30,
		help:        help.New(),
		currentPath: project,
		currentTape: tape.Tape{Title: "Synthetic", Sources: []tape.Source{attached, sectioned}},
		sourceTable: newSourceTable(100, 8),
	}
	m.help.SetWidth(96)

	got, _ := m.startAssemblyReview()
	if got.err != "" {
		t.Fatalf("assembly review failed: %s", got.err)
	}
	first := got.sourceItems[0]
	if first.Source.Section != nil || first.Source.Kind != nil {
		t.Fatalf("the attached Source sent to Core must stay the saved copy, got %#v", first.Source)
	}
	if first.DraftSection != "guidance" || first.DraftKind != "guide" {
		t.Fatalf("the attached Source should carry the draft's section and kind for review, got %q and %q", first.DraftSection, first.DraftKind)
	}
	if payload := sourceMaintenancePayload(first.Source); payload["section"] != nil || payload["kind"] != nil {
		t.Fatalf("the draft's section and kind must not reach Core's payload for a saved Source, got %#v", payload)
	}

	rows := assemblyRows(got.sourceItems, 96, 0)
	if rows[0][1] != "guide" || rows[0][2] != "guidance" {
		t.Fatalf("the attached Source should show the draft's kind and section, not %q and %q", rows[0][1], rows[0][2])
	}
	if rows[1][2] != savedSection || rows[1][1] != "study" {
		t.Fatalf("a saved section wins over the draft's; the draft fills only the missing kind, got %q and %q", rows[1][2], rows[1][1])
	}

	plain := stripANSICodesForTest(got.View().Content)
	flattened := strings.Join(strings.Fields(plain), " ")
	for _, want := range []string{
		"Use: checked · guide · guidance · required",
		"Saved: No kind or section; the kind and section shown come from the draft. Accepting keeps the saved Source as it is.",
	} {
		if !strings.Contains(flattened, want) {
			t.Fatalf("selected detail missing %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "unsectioned") {
		t.Fatalf("no Source in this draft should read as unsectioned:\n%s", plain)
	}
}

func TestAssemblyReviewScrollsAVeryLongSelectedNoteAt100x30(t *testing.T) {
	words := make([]string, 0, 320)
	for index := range 320 {
		words = append(words, fmt.Sprintf("w%03d", index))
	}
	longNote := strings.Join(words, " ") + " END-OF-NOTE."
	savedNote := "A different saved note that the review also shows."
	items := make([]tape.Source, 0, 6)
	for index := range 6 {
		note := fmt.Sprintf("Short note %d.", index)
		if index == 0 {
			note = savedNote
		}
		items = append(items, tape.Source{Type: "web", URL: fmt.Sprintf("https://example.test/s-%d", index), Note: &note, Priority: "required"})
	}
	m := Model{
		screen:      screenAssemblyReview,
		width:       100,
		height:      30,
		help:        help.New(),
		sourceTable: newSourceTable(100, 8),
		currentTape: tape.Tape{Title: "Synthetic"},
	}
	m.help.SetWidth(96)
	m.sourceItems = source.Stage(items, true)
	m.sourceItems[0].DraftNote = longNote
	m.applySourceItems(m.sourceItems)
	m.sourceMaintenancePlan = &core.ProjectChangeSet{Operations: []map[string]any{{"type": "source.add"}}}
	m.note = "Review the checked Sources. Press Enter to accept this selection."

	check := func(stage string, m Model, expected ...string) string {
		t.Helper()
		rendered := m.View().Content
		if got := lipgloss.Height(rendered); got > 30 {
			t.Fatalf("%s: review should fit 100 x 30, got %d lines", stage, got)
		}
		plain := stripANSICodesForTest(rendered)
		if strings.Contains(plain, "more content hidden") {
			t.Fatalf("%s: the frame should not cut the review off:\n%s", stage, plain)
		}
		for _, want := range append(expected, "Ready to accept Sources", "Press Enter to accept 6 checked Sources") {
			if !strings.Contains(plain, want) {
				t.Fatalf("%s: missing %q:\n%s", stage, want, plain)
			}
		}
		return plain
	}
	check("top", m, "w000", "more below · pgup/pgdn scroll")
	for range 40 {
		m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	}
	plain := check("end", m, "END-OF-NOTE.", "Saved note: "+savedNote, "end; more above")
	if strings.Contains(plain, "w100 ") {
		t.Fatalf("the note should have scrolled:\n%s", plain)
	}
	if m.sourceTable.Cursor() != 0 {
		t.Fatalf("pgdn should scroll the note, not move the table cursor, got %d", m.sourceTable.Cursor())
	}
	m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if m.assemblyDetailOffset != 0 {
		t.Fatalf("moving to another Source should start its detail at the top, got %d", m.assemblyDetailOffset)
	}
	check("next source", m, "Short note 1.")
}
