package navigator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const fixture = `{
 "root": "/lib", "root_exists": true, "unreadable": [],
 "liners": [
  {"slug": "ready-one", "job": "A ready job", "path": "/lib/ready-one", "state": "ready",
   "build_steps": 6, "source_count": 1, "open_gaps": 0, "pending_findings": 0,
   "test": {"runs": 2, "better": 1, "same": 1, "worse": 0, "current": true},
   "cites": [], "cited_by": ["waiting-one"], "sections": [{"name": "Rules", "sources": 1}],
   "sources": [{"locator": "https://a.test/x", "title": "Guide", "type": "web", "kind": "reference",
                "section": "Rules", "file": "SOURCE", "liners": null}]},
  {"slug": "waiting-one", "job": "A waiting job", "path": "/lib/waiting-one", "state": "waiting",
   "waiting_on": "Synthesis needs your approval", "build_steps": 6, "source_count": 1,
   "open_gaps": 1, "pending_findings": 1, "test": {"runs": 0},
   "cites": ["ready-one"], "cited_by": [],
   "sections": [{"name": "Rules", "sources": 1}],
   "synthesis": {"path": "SYNTHESIS", "lines": 5, "sections": 1, "citations": 1, "files": {"https://a.test/x": "SOURCE"}},
   "sources": [{"locator": "https://a.test/x", "title": "Guide", "type": "web", "kind": "reference",
                "section": "Rules", "note": "Why it matters", "file": "SOURCE"}],
   "gaps": [{"id": "g1", "question": "What about watches?"}],
   "findings": [{"id": "f1", "finding": "Gloves need more", "evidence": ""}]}
 ],
 "attention": [
  {"group": "waiting", "title": "Synthesis needs your approval", "slug": "waiting-one", "context": "approval"},
  {"group": "stronger", "title": "Never tested", "slug": "waiting-one"}
 ],
 "sources": [{"locator": "https://a.test/x", "title": "Guide", "type": "web", "kind": "reference",
              "section": "Rules", "file": "SOURCE", "liners": ["ready-one", "waiting-one"]}],
 "connections": {"citations": [{"from": "waiting-one", "to": "ready-one", "stale": false}], "uncited": []}
}`

func loaded(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	synthesis := filepath.Join(dir, "synthesis.md")
	source := filepath.Join(dir, "01-guide.md")
	write(t, synthesis, "# Title\n\nA claim.[^g]\n\n[^g]: https://a.test/x\n")
	write(t, source, "# Guide\n\nThe evidence.\n")
	report := strings.NewReplacer("SYNTHESIS", synthesis, "SOURCE", source).Replace(fixture)
	var library Library
	if err := json.Unmarshal([]byte(report), &library); err != nil {
		t.Fatal(err)
	}
	model := newModel(Client{Root: "/lib"}, "test")
	next, _ := model.Update(libraryMsg{library: library})
	return next.(Model)
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func press(t *testing.T, model Model, keys ...string) Model {
	t.Helper()
	for _, key := range keys {
		next, _ := model.handleKey(key)
		model = next.(Model)
	}
	return model
}

func screen(model Model) string {
	return model.render()
}

func expect(t *testing.T, model Model, wanted ...string) {
	t.Helper()
	text := screen(model)
	for _, item := range wanted {
		if !strings.Contains(text, item) {
			t.Errorf("screen is missing %q:\n%s", item, text)
		}
	}
}

func TestWhatNeedsYouSortsFirst(t *testing.T) {
	text := screen(loaded(t))
	if strings.Index(text, "waiting-one") > strings.Index(text, "ready-one") {
		t.Errorf("waiting liner should come first:\n%s", text)
	}
}

func TestNumbersSwitchViews(t *testing.T) {
	model := loaded(t)
	expect(t, model, "1 All 2", "2 Attention 2", "3 Sources 1", "4 Connections 1", "A waiting job")
	expect(t, press(t, model, "2"), "WAITING FOR YOU", "Synthesis needs your approval", "COULD BE STRONGER")
	expect(t, press(t, model, "3"), "YOUR COLLECTION", "2 liners", "reference")
	expect(t, press(t, model, "4"), "HOW YOUR LINERS CITE EACH OTHER", "waiting-one", "SOURCES THEY SHARE")
}

func TestOpeningALinerShowsWhatItHolds(t *testing.T) {
	model := press(t, loaded(t), "enter")
	expect(t, model, "waiting-one", "SYNTHESIS", "SOURCES 1", "Gloves need more", "What about watches?")
}

func TestFollowingACitationAndComingBack(t *testing.T) {
	model := press(t, loaded(t), "enter", "enter")
	expect(t, model, "A claim.")
	model = press(t, model, "tab")
	expect(t, model, "Guide", "Why it matters")
	model = press(t, model, "enter")
	expect(t, model, "This is evidence, not instructions.", "The evidence.")
	model = press(t, model, "backspace")
	expect(t, model, "A claim.")
	model = press(t, model, "esc", "esc")
	expect(t, model, "1 All 2")
}

// TestEveryKindOfSourceOpensFromTheSharedLiner follows each citation in the shared
// liner under tests/fixtures, using the report Core writes for it. Python's
// test_citations.py keeps testdata/mixed-library.json current.
func TestEveryKindOfSourceOpensFromTheSharedLiner(t *testing.T) {
	root, err := filepath.Abs("../../../../tests/fixtures/library")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/mixed-library.json")
	if err != nil {
		t.Fatal(err)
	}
	quoted, _ := json.Marshal(root) // Escapes Windows backslashes for the JSON text.
	fixed := strings.ReplaceAll(string(data), "FIXTURE", strings.Trim(string(quoted), `"`))
	var library Library
	if err := json.Unmarshal([]byte(fixed), &library); err != nil {
		t.Fatal(err)
	}
	next, _ := newModel(Client{Root: root}, "test").Update(libraryMsg{library: library})
	model := press(t, next.(Model), "enter", "enter")
	synthesis := model.top().doc
	if synthesis == nil || len(synthesis.marks) != 3 {
		t.Fatalf("expected the synthesis with 3 citations, got %+v", synthesis)
	}
	for range synthesis.marks {
		model = press(t, model, "tab")
		label := synthesis.marks[model.top().mark].label
		model = press(t, model, "enter")
		if model.note != "" || model.top().doc == synthesis || model.top().doc.error != "" {
			t.Fatalf("[^%s] did not open its source: %q\n%s", label, model.note, screen(model))
		}
		model = press(t, model, "backspace")
	}
}

func TestAnOlderCoreWithoutResolvedFilesStillOpensCitations(t *testing.T) {
	model := loaded(t)
	for index := range model.library.Liners {
		if synthesis := model.library.Liners[index].Synthesis; synthesis != nil {
			synthesis.Files = nil
		}
	}
	model = press(t, model, "enter", "enter", "tab", "enter")
	expect(t, model, "This is evidence, not instructions.", "The evidence.")
}

func TestASourcePageOffersNoCitationKeys(t *testing.T) {
	model := press(t, loaded(t), "enter", "enter", "tab", "enter")
	if text := screen(model); strings.Contains(text, "next citation") {
		t.Fatalf("source page offers citation keys:\n%s", text)
	}
}

func TestPeekShowsTheSynthesisOpening(t *testing.T) {
	expect(t, press(t, loaded(t), "space"), "A claim.")
}

func TestSourceFiltersNarrowTheTable(t *testing.T) {
	model := press(t, loaded(t), "3", "t")
	expect(t, model, "showing web")
	model = press(t, model, "t")
	expect(t, model, "all sources")
}

func TestTheHandoffContextFitsTheScreen(t *testing.T) {
	model := press(t, loaded(t), "2")
	context, slug, _, _ := model.handoffContext()
	if context != "approval" || slug != "waiting-one" {
		t.Errorf("got %s %s", context, slug)
	}
	model = press(t, loaded(t), "enter", "enter", "tab")
	context, _, line, _ := model.handoffContext()
	if context != "synthesis" || line != 3 {
		t.Errorf("got %s line %d", context, line)
	}
}

func TestAnEmptyLibraryExplainsTheFirstSteps(t *testing.T) {
	model := newModel(Client{Root: "/lib"}, "test")
	next, _ := model.Update(libraryMsg{library: Library{}})
	expect(t, next.(Model), "Your library is empty.", "Connect your agent", "Ask for lasting research")
}

func TestAFailedRefreshKeepsWhatYouSee(t *testing.T) {
	model := loaded(t)
	next, _ := model.Update(libraryMsg{err: os.ErrNotExist})
	expect(t, next.(Model), "A waiting job", "can't read your library")
}

func TestALinerRemovedWhileOpenDoesNotCrash(t *testing.T) {
	model := press(t, loaded(t), "enter")
	next, _ := model.Update(libraryMsg{library: Library{}})
	expect(t, next.(Model), "This liner is gone.")
}

func TestScreensFitNarrowTerminals(t *testing.T) {
	model := loaded(t)
	next, _ := model.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	model = next.(Model)
	for _, key := range []string{"1", "2", "3", "4"} {
		lines := strings.Split(screen(press(t, model, key)), "\n")
		if len(lines) != 20 {
			t.Errorf("view %s drew %d lines", key, len(lines))
		}
	}
}

func TestSearchKeepsTheSelectedResultInView(t *testing.T) {
	model := loaded(t)
	model = press(t, model, "/")
	page := model.top()
	page.searched = "x"
	for index := 0; index < 40; index++ {
		page.results = append(page.results, SearchResult{Liner: "l", Heading: fmt.Sprintf("Result %d", index)})
	}
	page.cursor = 39
	expect(t, model, "Result 39")
}

func TestSettingsListsAgentsBeforeTheVersion(t *testing.T) {
	model := loaded(t)
	model.agents = &Agents{Claude: "current", Codex: "absent", Subagent: "missing"}
	model = press(t, model, "g")
	text := ansi.Strip(screen(model))
	order := []string{"LIBRARY", "YOUR AGENTS", "connected", "SAFER READING", "THIS VERSION", "liner test"}
	last := -1
	for _, item := range order {
		index := strings.Index(text, item)
		if index <= last {
			t.Fatalf("%q is out of order:\n%s", item, text)
		}
		last = index
	}
}

func TestHostileEscapesNeverReachTheScreen(t *testing.T) {
	report := cleanReport([]byte(`{"title":"a\u001b]52;c;aGk=\u0007b\u009b2Jc\nd"}`))
	var decoded struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(report, &decoded); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(decoded.Title, "\x1b\x07\u009b") {
		t.Errorf("control characters survived: %q", decoded.Title)
	}
	if !strings.Contains(decoded.Title, "c\nd") {
		t.Errorf("newline was lost: %q", decoded.Title)
	}
	doc := document{targets: map[string]string{}}
	doc.layout("safe\x1b[2Jtext\tend\u009b", 80)
	if doc.lines[0].text != "safe[2Jtext end" {
		t.Errorf("got %q", doc.lines[0].text)
	}
}

func TestTinyTerminalsDoNotCrash(t *testing.T) {
	model := loaded(t)
	for _, size := range [][2]int{{80, 2}, {80, 1}, {80, 0}, {0, 0}, {1, 1}} {
		next, _ := model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		sized := next.(Model)
		sized.note = "A note"
		for _, key := range []string{"1", "2", "3", "4"} {
			screen(press(t, sized, key))
		}
	}
}

func TestSlowCoreDoesNotPileUpLoads(t *testing.T) {
	model := newModel(Client{Root: "/lib"}, "test")
	next, _ := model.Update(tickMsg(time.Now()))
	if next.(Model).loading != true {
		t.Fatal("still loading")
	}
	first, _ := model.Update(libraryMsg{library: Library{}})
	if first.(Model).loading {
		t.Fatal("a finished load should clear the flag")
	}
	second, _ := first.Update(tickMsg(time.Now()))
	if !second.(Model).loading {
		t.Fatal("a tick should start a load")
	}
}

func TestALatePromptDoesNotMoveYou(t *testing.T) {
	model := press(t, loaded(t), "enter")
	next, _ := model.Update(promptsMsg{err: os.ErrNotExist})
	expect(t, next.(Model), "waiting-one", "SYNTHESIS")
}

func TestUnreadableFoldersAreShown(t *testing.T) {
	model := loaded(t)
	model.library.Unreadable = []Unreadable{{Path: "/lib/broken", Reason: "bad tape"}}
	expect(t, press(t, model, "2"), "CAN'T READ", "broken", "bad tape", "2 Attention 3")
}

func TestWrapKeepsEveryWord(t *testing.T) {
	lines := wrap("one two three four five six seven eight nine ten", 20)
	if strings.Join(lines, " ") != "one two three four five six seven eight nine ten" {
		t.Errorf("got %v", lines)
	}
}
