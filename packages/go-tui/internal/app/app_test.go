package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/agent"
	"github.com/cmdux/liner/packages/go-tui/internal/core"
	linerprogress "github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func TestMain(m *testing.M) {
	// Keep Project estimate rendering from reading or writing a developer's
	// real ~/.liner/run-estimates.jsonl during ordinary app tests. Individual
	// global-history tests opt back in with their own temp file.
	_ = os.Setenv("LINER_ESTIMATE_HISTORY", "")
	os.Exit(m.Run())
}

func assertNoBoxCorners(t *testing.T, view string) {
	t.Helper()
	for _, border := range []string{"╭", "╮", "╰", "╯", "┌", "┐", "└", "┘"} {
		if strings.Contains(view, border) {
			t.Fatalf("view should not render outer boxes:\n%s", view)
		}
	}
}

func assertViewLinesFit(t *testing.T, view string, width int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("rendered line wider than terminal: got %d, want <= %d\n%s", got, width, line)
		}
	}
}

func stripANSICodesForTest(value string) string {
	var out strings.Builder
	skipping := false
	for i := 0; i < len(value); i++ {
		if value[i] == 0x1b {
			skipping = true
			continue
		}
		if skipping {
			if (value[i] >= 'A' && value[i] <= 'Z') || (value[i] >= 'a' && value[i] <= 'z') {
				skipping = false
			}
			continue
		}
		out.WriteByte(value[i])
	}
	return out.String()
}

func lineContaining(t *testing.T, view string, text string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, text) {
			return line
		}
	}
	t.Fatalf("view missing line containing %q:\n%s", text, view)
	return ""
}

func assertTitleLineHasLoader(t *testing.T, view string, title string) {
	t.Helper()
	plain := stripANSICodesForTest(view)
	line := lineContaining(t, plain, title)
	if strings.TrimSpace(line) == title {
		t.Fatalf("title %q should include the loading spinner:\n%s", title, plain)
	}
}

func TestCreateJobFieldRendersWithinTerminalWidth(t *testing.T) {
	width := 118
	area := newCreateArea(createAreaWidth(width))
	area.SetValue("When I am collecting a long messy job story with multiple sentences, I want the setup editor to wrap inline so the footer stays below the field list instead of sliding under the content.")

	m := Model{
		screen:     screenCreate,
		width:      width,
		createStep: 1,
		createArea: area,
		createDraft: createDraft{
			Title:      "Mobile Design Foundations",
			JTBD:       area.Value(),
			Curator:    "Arturo",
			AddSources: true,
		},
	}

	view := m.viewCreate()
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("rendered line is wider than terminal: got %d, want <= %d\n%s", got, width, line)
		}
	}
}

func TestCreateViewUsesInlineSetupWithoutBoxes(t *testing.T) {
	m := Model{
		screen:      screenCreate,
		width:       118,
		createStep:  0,
		createInput: textinput.New(),
		createArea:  newCreateArea(createAreaWidth(118)),
		createDraft: createDraft{
			AddSources: true,
		},
	}
	m.createInput.Prompt = ""
	m.createInput.Placeholder = "..."
	m.createInput.SetWidth(createInputWidth(118))
	m.createInput.Focus()

	view := m.viewCreate()
	for _, border := range []string{"╭", "╮", "╰", "╯", "┌", "┐", "└", "┘"} {
		if strings.Contains(view, border) {
			t.Fatalf("setup view should not render boxes:\n%s", view)
		}
	}
	if !strings.Contains(view, "Name this mixtape.") {
		t.Fatalf("setup view should show field-specific instruction:\n%s", view)
	}
	if strings.Contains(strings.ToLower(view), "folder") {
		t.Fatalf("setup view should not explain generated folder names:\n%s", view)
	}
	if !strings.Contains(view, "> Name") {
		t.Fatalf("active row should own the input cursor:\n%s", view)
	}
}

func TestCreateViewShowsCurrentStepSubtitle(t *testing.T) {
	cases := []struct {
		step     int
		expected string
	}{
		{step: 0, expected: "Step 1 of 4 - Name the mixtape."},
		{step: 1, expected: "Step 2 of 4 - Define the AI-agent goal."},
		{step: 2, expected: "Step 3 of 4 - Name the curator."},
		{step: 3, expected: "Step 4 of 4 - Choose source capture."},
	}
	for _, tc := range cases {
		input := textinput.New()
		input.Prompt = ""
		input.SetWidth(createInputWidth(118))
		area := newCreateArea(createAreaWidth(118))
		m := Model{
			screen:      screenCreate,
			width:       118,
			createInput: input,
			createArea:  area,
			createDraft: createDraft{
				Title:      "Mobile Design Foundations",
				JTBD:       "When I design apps, I want better decisions.",
				Curator:    "Arturo",
				AddSources: true,
			},
		}
		m.setCreateField(tc.step)

		view := m.viewCreate()
		if !strings.Contains(view, tc.expected) {
			t.Fatalf("step %d subtitle missing %q:\n%s", tc.step, tc.expected, view)
		}
		if tc.step > 0 && strings.Contains(view, "Step 1 of 4 - Set up the mixtape.") {
			t.Fatalf("step %d should not show the stale setup subtitle:\n%s", tc.step, view)
		}
	}
}

func TestCreateSourceCaptureSelectorUsesCheckboxesWithoutInlineShortcut(t *testing.T) {
	m := Model{
		screen:     screenCreate,
		width:      118,
		createStep: 3,
		createDraft: createDraft{
			Title:      "Mobile Design Foundations",
			JTBD:       "When I design apps, I want better decisions.",
			Curator:    "Arturo",
			AddSources: true,
		},
	}

	view := m.viewCreate()
	for _, expected := range []string{"☑ Yes", "☐ No"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("source capture selector missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "←/→ switch") {
		t.Fatalf("selector should leave shortcut help to the footer:\n%s", view)
	}

	m.createDraft.AddSources = false
	view = m.viewCreate()
	for _, expected := range []string{"☐ Yes", "☑ No"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("source capture selector missing %q after toggle:\n%s", expected, view)
		}
	}
}

func TestBannerRendersWithinTerminalWidth(t *testing.T) {
	m := Model{
		width:  80,
		screen: screenProject,
		currentTape: tape.Tape{
			Title: "Mobile Design Foundations",
			Sources: []tape.Source{
				{Type: "web", URL: "https://example.com"},
			},
		},
	}

	banner := m.viewBanner()
	if got, want := lipgloss.Width(banner), chromeWidth(m.width); got > want {
		t.Fatalf("banner wider than chrome: got %d, want <= %d\n%s", got, want, banner)
	}
	plain := stripANSICodesForTest(banner)
	if !strings.Contains(plain, "liner v1") {
		t.Fatalf("banner should show v1 product version:\n%s", banner)
	}
	if strings.Contains(plain, "liner v2") {
		t.Fatalf("banner should not show stale v2 product version:\n%s", banner)
	}
	if got := strings.Count(plain, "/"); got != 30 {
		t.Fatalf("banner should use the fixed three-repeat texture, got %d slashes:\n%s", got, banner)
	}
	if got := strings.Count(strings.TrimRight(plain, " "), "\n"); got != 0 {
		t.Fatalf("banner should stay on one line, got %d line breaks:\n%s", got, banner)
	}
	if !strings.Contains(plain, "project") || !strings.Contains(plain, "sources") {
		t.Fatalf("banner should show metadata after fixed texture:\n%s", banner)
	}
}

func TestSplashRendersWithinTerminalWidth(t *testing.T) {
	m := Model{width: 80}
	splash := m.viewSplash()
	assertNoBoxCorners(t, splash)
	for _, line := range strings.Split(splash, "\n") {
		if got, want := lipgloss.Width(line), chromeWidth(m.width); got > want {
			t.Fatalf("splash line wider than chrome: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestHomeImportKeyOpensImportScreen(t *testing.T) {
	m := Model{
		screen: screenHome,
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	if got.screen != screenImport {
		t.Fatalf("expected import screen, got %v", got.screen)
	}
}

func TestHomeShowsCommandList(t *testing.T) {
	m := Model{screen: screenHome, width: 100}
	m.commands = newCommandList(70, 18)
	m.commands.SetItems(m.commandItems())

	view := m.viewHome()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"New Liner Project", "Projects", "Import Project", "Settings"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("home command list missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Provider Preferences", "Set Up Liner", "Start", "Choose what you want to do", "Home", "Open and manage Liner projects", "System", "↑/k up", "j down"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("home should render commands directly, found %q:\n%s", unexpected, view)
		}
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "enter") || !hasHelp(m.helpForScreen().ShortHelp(), "/") {
		t.Fatalf("home help should expose command selection and filtering")
	}
}

func TestHomeEnterRunsSelectedCommand(t *testing.T) {
	m := Model{screen: screenHome}
	m.commands = newCommandList(70, 18)
	m.commands.SetItems(m.commandItems())
	m.commands.Select(1)

	got, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil || got.screen != screenProjects {
		t.Fatalf("enter should run the selected command, screen=%v cmd=%v", got.screen, cmd)
	}
}

func TestProjectEscReturnsToProjectsBrowser(t *testing.T) {
	m := Model{
		screen:      screenProject,
		baseDir:     t.TempDir(),
		currentPath: filepath.Join(t.TempDir(), "design-engineer"),
	}

	got, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if got.screen != screenProjects {
		t.Fatalf("esc should return from Project to Projects, got %v", got.screen)
	}
	if cmd == nil {
		t.Fatal("esc should reload projects when returning to the Projects browser")
	}
}

func TestProjectShortHelpKeepsBackBeforeSettings(t *testing.T) {
	short := Model{screen: screenProject}.helpForScreen().ShortHelp()
	backIndex := -1
	settingsIndex := -1
	for index, binding := range short {
		switch binding.Help().Key {
		case "esc":
			backIndex = index
		case "g":
			settingsIndex = index
		}
	}
	if backIndex == -1 {
		t.Fatalf("project short help should include esc back, got %#v", short)
	}
	if settingsIndex != -1 && backIndex > settingsIndex {
		t.Fatalf("project short help should show esc before settings so it remains visible, got %#v", short)
	}
}

func TestHomeSettingsKeyOpensSettings(t *testing.T) {
	m := Model{screen: screenHome}
	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))
	if got.screen != screenSettings {
		t.Fatalf("expected settings screen, got %v", got.screen)
	}
}

func TestProjectsBrowserUsesTwoColumnsWithSelectedDetails(t *testing.T) {
	job := "When I ship an app, I want App Store launch guidance."
	items := []projectItem{{
		project: core.ProjectSummary{
			Path:        "/tmp/liner/design-engineering",
			Title:       "Design Engineering",
			Description: "A launch guidance mixtape.",
			JTBD:        &job,
			SourceCount: 24,
		},
		capabilities: capabilitySummary{
			HasLiner: true,
			Skills:   4,
			Audits:   2,
			Evals:    1,
			Children: 3,
			Lineage:  true,
		},
	}}
	m := Model{
		screen:       screenProjects,
		width:        118,
		height:       30,
		projectItems: items,
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	view := m.viewProjects()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Projects", "Open and manage Liner projects", "Name:", "Design Engineering", "Status:", "Project Complete", "Description:", "A launch guidance mixtape.", "Job:", "When I ship an app", "Store launch guidance", "Folder:", "/tmp/liner/design-engineering"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("projects browser missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"enter open", "Settings", "esc back", "Filter", "All projects", "Field", "Value", "Project:", "Sources:", "Skills:", "Audits:", "Impact:", "Children:"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("projects browser should leave %q to the footer:\n%s", unexpected, view)
		}
	}
	for _, oldHeader := range []string{"Project       State", "Src ", "Projects                        \n", "────────────────"} {
		if strings.Contains(view, oldHeader) {
			t.Fatalf("projects browser should keep details out of the left table; found %q:\n%s", oldHeader, view)
		}
	}
}

func TestProjectsBrowserDerivesPlaceholderDescription(t *testing.T) {
	job := "When I finish developing my app and I am ready to submit to the App Store, I want to know what Apple requires so I can launch successfully."
	item := projectItem{
		project: core.ProjectSummary{
			Title:       "iOS appstore launch",
			Description: pendingProjectDescription,
			Path:        "/tmp/liner/ios-appstore-launch",
			JTBD:        &job,
			SourceCount: 28,
		},
	}
	m := Model{
		screen:       screenProjects,
		width:        118,
		height:       24,
		projectItems: []projectItem{item},
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	view := m.viewProjects()
	for _, expected := range []string{"Description:", "Guidance for iOS appstore launch."} {
		if !strings.Contains(view, expected) {
			t.Fatalf("projects browser missing derived description %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, pendingProjectDescription) {
		t.Fatalf("projects browser should not surface pending placeholder:\n%s", view)
	}
}

func TestProjectsBrowserEmptyStateShowsLibrary(t *testing.T) {
	library := "/tmp/liner/mixtapes"
	m := Model{
		screen:       screenProjects,
		width:        100,
		height:       24,
		baseDir:      library,
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	view := m.viewProjects()
	for _, expected := range []string{"No projects found", "Selected:", "No project selected.", "Library:", library} {
		if !strings.Contains(view, expected) {
			t.Fatalf("projects browser empty state missing %q:\n%s", expected, view)
		}
	}
	assertViewLinesFit(t, view, styles.ClampWidth(m.width-4))
}

func TestProjectsBrowserFitsNarrowWidth(t *testing.T) {
	width := 80
	job := "When I finish developing my app description and after page, I need launch guidance that stays inside the project detail pane."
	item := projectItem{
		project: core.ProjectSummary{
			Title:       "Design Engineering With A Long Name",
			Description: "A long selected project description that should stay inside the project detail pane.",
			Path:        "/tmp/liner/design-engineering-with-a-very-long-folder-name",
			JTBD:        &job,
			SourceCount: 12,
		},
		capabilities: capabilitySummary{HasLiner: true, Skills: 4, Audits: 2, Evals: 1},
	}
	m := Model{
		screen:       screenProjects,
		width:        width,
		height:       24,
		projectItems: []projectItem{item},
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	view := m.viewProjects()
	assertNoBoxCorners(t, view)
	assertViewLinesFit(t, view, styles.ClampWidth(width-4))
}

func TestProjectsBrowserFolderPathStaysInDetailPane(t *testing.T) {
	width := 80
	job := "Choose an App Store launch path."
	item := projectItem{
		project: core.ProjectSummary{
			Title:       "iOS appstore launch",
			Description: "Guidance for iOS appstore launch.",
			Path:        "/Users/arturo/.codex/worktrees/9326/liner/mixtapes/ios-appstore-launch",
			JTBD:        &job,
			SourceCount: 28,
		},
	}
	m := Model{
		screen:       screenProjects,
		width:        width,
		height:       24,
		projectItems: []projectItem{item},
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	view := m.viewProjects()
	assertViewLinesFit(t, view, styles.ClampWidth(width-4))
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "ios-appstore-launch") && !strings.Contains(line, "iOS appstore launch") {
			if prefix := strings.TrimSpace(line[:min(len(line), projectBrowserListWidth(width)+2)]); prefix != "" {
				t.Fatalf("folder continuation should stay out of left pane, prefix=%q:\n%s", prefix, view)
			}
		}
	}
}

func TestProjectsHelpShowsSettingsAndBack(t *testing.T) {
	m := Model{screen: screenProjects}
	help := m.helpForScreen().ShortHelp()
	for _, keyName := range []string{"g", "esc"} {
		if !hasHelp(help, keyName) {
			t.Fatalf("projects short help should include %q, got %#v", keyName, help)
		}
	}
	if hasHelp(help, ",") {
		t.Fatalf("projects short help should not advertise comma for settings: %#v", help)
	}
}

func TestProjectsBrowserFilterNarrowsRowsAndEscClears(t *testing.T) {
	items := []projectItem{
		{
			project:      core.ProjectSummary{Path: "/tmp/liner/design", Title: "Design Engineering", SourceCount: 24},
			capabilities: capabilitySummary{HasLiner: true, Skills: 4},
		},
		{
			project:      core.ProjectSummary{Path: "/tmp/liner/research", Title: "Research Ops", SourceCount: 8},
			capabilities: capabilitySummary{Audits: 1},
		},
	}
	m := Model{
		screen:       screenProjects,
		width:        100,
		height:       30,
		projectItems: items,
		projectShown: items,
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	var cmd tea.Cmd
	m, cmd = m.handleKey(tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}))
	if cmd != nil || !m.homeFiltering {
		t.Fatalf("expected / to enter filter mode, filtering=%v cmd=%v", m.homeFiltering, cmd)
	}
	m, cmd = m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if cmd != nil || m.homeFilter != "q" {
		t.Fatalf("filter mode should treat q as text, filter=%q cmd=%v", m.homeFilter, cmd)
	}
	m, _ = m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if m.homeFilter != "" || len(m.projectShown) != 2 {
		t.Fatalf("esc should clear active filter, filter=%q shown=%d", m.homeFilter, len(m.projectShown))
	}
	m, _ = m.handleKey(tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}))
	for _, r := range []rune("design") {
		m, _ = m.handleKey(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
	if len(m.projectShown) != 1 || m.projectShown[0].project.Title != "Design Engineering" {
		t.Fatalf("expected design filter to show one design project, got %#v", m.projectShown)
	}
	view := m.viewProjects()
	if !strings.Contains(view, "Design Engineering") || strings.Contains(view, "Research Ops") {
		t.Fatalf("filtered projects view should only show design project:\n%s", view)
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "typing") && !hasHelp(m.helpForScreen().ShortHelp(), "type") {
		t.Fatalf("active filter help should expose typing, got %#v", m.helpForScreen().ShortHelp())
	}
	m, _ = m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if m.homeFiltering {
		t.Fatal("enter should leave filter mode active query in place")
	}
	m, _ = m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if m.homeFilter != "" || len(m.projectShown) != 2 {
		t.Fatalf("esc should clear filter and restore rows, filter=%q shown=%d", m.homeFilter, len(m.projectShown))
	}
}

func TestProjectsEnterOpensSelectedProjectTableRow(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Launch"}); err != nil {
		t.Fatal(err)
	}
	items := []projectItem{{
		project: core.ProjectSummary{
			Path:        project,
			Title:       "Launch",
			SourceCount: 1,
		},
	}}
	m := Model{
		screen:       screenProjects,
		width:        100,
		height:       30,
		projectItems: items,
		projectTable: newProjectTable(34, 10),
	}
	m.applyHomeProjectFilter()

	_, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected enter to open the selected project")
	}
	msg, ok := cmd().(projectOpenedMsg)
	if !ok {
		t.Fatalf("expected projectOpenedMsg, got %#v", msg)
	}
	if msg.err != nil || msg.path != project || msg.tape.Title != "Launch" {
		t.Fatalf("unexpected opened project message: %#v", msg)
	}
}

func TestProjectListDescriptionShowsV1MilestoneSignal(t *testing.T) {
	item := projectItem{
		project: core.ProjectSummary{
			Path:        "/tmp/liner/design-engineering",
			Title:       "Design Engineering",
			SourceCount: 24,
		},
		capabilities: capabilitySummary{
			HasLiner: true,
			Skills:   4,
			Audits:   2,
			Evals:    1,
			Children: 3,
			Lineage:  true,
		},
	}

	description := item.Description()
	for _, expected := range []string{"24 sources", "Project Complete", "/tmp/liner/design-engineering"} {
		if !strings.Contains(description, expected) {
			t.Fatalf("project description missing %q: %s", expected, description)
		}
	}
	for _, unexpected := range []string{"4 skills", "2 audits", "1 impact test", "3 child mixtapes", "lineage"} {
		if strings.Contains(description, unexpected) {
			t.Fatalf("project description should hide v2 capability %q: %s", unexpected, description)
		}
	}
}

func TestImportViewShowsFilePicker(t *testing.T) {
	width := 80
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "product-design.mixtape"), []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	picker := newImportPicker(dir, 32)
	pickerMsg := picker.Init()()
	var cmd tea.Cmd
	picker, cmd = picker.Update(pickerMsg)
	if cmd != nil {
		t.Fatal("file picker read update should not need another command")
	}
	m := Model{
		screen:       screenImport,
		width:        width,
		height:       32,
		baseDir:      filepath.Join(t.TempDir(), "mixtapes", "nested", "destination"),
		importPicker: picker,
	}

	view := m.viewImport()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Import Project", "Choose a .mixtape", "Folder:", "Destination:", "Sources:", "Use archived source files", "product-design.mixtape"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("import view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Import Mixtape", "Path", "Actions", "typed path"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("import view should not render old path/action UI %q:\n%s", unexpected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("import line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestImportBusyViewUsesTitleLoader(t *testing.T) {
	width := 80
	dir := t.TempDir()
	picker := newImportPicker(dir, 32)
	m := Model{
		screen:       screenImport,
		width:        width,
		height:       32,
		baseDir:      filepath.Join(t.TempDir(), "mixtapes"),
		importPicker: picker,
		importBusy:   true,
		researchSpin: newLoadingSpinner(),
	}

	view := m.viewImport()
	assertTitleLineHasLoader(t, view, "Import Project")
	if !strings.Contains(view, "Importing...") {
		t.Fatalf("busy import view should show import state:\n%s", view)
	}
}

func TestImportRefreshKeyReloadsPicker(t *testing.T) {
	dir := t.TempDir()
	picker := newImportPicker(dir, 24)
	pickerMsg := picker.Init()()
	var cmd tea.Cmd
	picker, cmd = picker.Update(pickerMsg)
	if cmd != nil {
		t.Fatal("file picker read update should not need another command")
	}
	if err := os.WriteFile(filepath.Join(dir, "later.mixtape"), []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:       screenImport,
		importPicker: picker,
	}

	next, reload := m.handleImportKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))
	if reload == nil {
		t.Fatal("refresh should return a picker reload command")
	}
	reloadMsg := reload()
	next.importPicker, cmd = next.importPicker.Update(reloadMsg)
	if cmd != nil {
		t.Fatal("file picker refresh update should not need another command")
	}
	if view := next.viewImport(); !strings.Contains(view, "later.mixtape") {
		t.Fatalf("refresh did not reload picker entries:\n%s", view)
	}
}

func TestCreateJobEnterContinuesToCurator(t *testing.T) {
	area := newCreateArea(46)
	area.SetValue("When I have a clear job, I want Enter to continue, so I can keep the setup flow moving.")

	input := textinput.New()
	m := Model{
		screen:      screenCreate,
		createStep:  1,
		createArea:  area,
		createInput: input,
		createDraft: createDraft{
			Title:      "Mobile Design Foundations",
			Curator:    "Arturo",
			AddSources: true,
		},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.createStep != 2 {
		t.Fatalf("expected enter to continue to curator, got step %d", got.createStep)
	}
	if got.createDraft.JTBD != area.Value() {
		t.Fatalf("expected job to be committed, got %q", got.createDraft.JTBD)
	}
}

func TestCreateSourceCaptureEnterBlocksExistingProjectName(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "art-director", "mixtape"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "art-director", "mixtape", "tape.yaml"), []byte("title: Art Director\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:     screenCreate,
		baseDir:    dir,
		createStep: 3,
		createDraft: createDraft{
			Title:      "Art Director",
			Slug:       "art-director",
			JTBD:       "I want this Liner to help an AI agent translate visual references into interface decisions.",
			Curator:    "cmdux",
			AddSources: false,
		},
	}

	got, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatal("existing project name should not start project creation")
	}
	if got.screen != screenCreate || got.createStep != 3 {
		t.Fatalf("expected to stay on source capture, screen=%v step=%d", got.screen, got.createStep)
	}
	if !strings.Contains(got.err, "already exists") || !strings.Contains(got.err, "Projects") {
		t.Fatalf("expected visible duplicate-project error, got %q", got.err)
	}
}

func TestNonKeyUpdateDoesNotClearVisibleError(t *testing.T) {
	m := Model{
		screen:             screenCreate,
		err:                "A project named \"art-director\" already exists.",
		compileSpin:        newLoadingSpinner(),
		researchSpin:       newLoadingSpinner(),
		clarifySpin:        newLoadingSpinner(),
		operatingLayerSpin: newLoadingSpinner(),
	}

	next, _ := m.Update(spinner.TickMsg{})
	got := next.(Model)
	if got.err == "" {
		t.Fatal("non-key updates should not clear visible errors")
	}
}

func TestProjectCreatedWithCustomSourcesStartsSourceEntry(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	input := textinput.New()
	m := Model{
		createDraft: createDraft{AddSources: true},
		sourceInput: input,
		sourceTable: newSourceTable(80, 8),
	}

	next, _ := m.Update(projectCreatedMsg{
		path: "/tmp/liner-flow",
		tape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	})
	got := next.(Model)

	if got.screen != screenSources {
		t.Fatalf("expected custom-source setup to open sources first, got %v", got.screen)
	}
	if got.clarifyLoading {
		t.Fatal("clarification should not start before custom sources are handled")
	}
}

func TestProjectCreatedWithoutCustomSourcesStartsClarification(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	m := Model{
		createDraft: createDraft{AddSources: false},
		sourceInput: textinput.New(),
		sourceTable: newSourceTable(80, 8),
		clarifyArea: newClarifyArea(64),
	}

	next, _ := m.Update(projectCreatedMsg{
		path: "/tmp/liner-flow",
		tape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	})
	got := next.(Model)

	if got.screen != screenClarify {
		t.Fatalf("expected no-source setup to clarify immediately, got %v", got.screen)
	}
	if !got.clarifyLoading {
		t.Fatal("expected clarification questions to be loading")
	}
}

func TestCreateDraftDescriptionUsesJobOrTitle(t *testing.T) {
	short := createDraftDescription(createDraft{
		Title: "Launch",
		JTBD:  "Help me understand what to include in a launch article.",
	})
	if short != "Help me understand what to include in a launch article." {
		t.Fatalf("expected short job description, got %q", short)
	}

	long := createDraftDescription(createDraft{
		Title: "iOS appstore launch",
		JTBD:  "When I finish developing my app and I am ready to submit to the App Store, I want to know everything Apple requires so I can launch successfully without missing required assets.",
	})
	if long != "Guidance for iOS appstore launch." {
		t.Fatalf("expected title fallback for long job, got %q", long)
	}
	if long == pendingProjectDescription {
		t.Fatal("new projects should not use the pending description placeholder")
	}
}

func TestOpenProjectDoesNotCreateLocalSourcesDirectory(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{
		Title:       "Launch",
		Description: "Demo",
		Curator:     "Arturo",
		Sources:     []tape.Source{},
	}); err != nil {
		t.Fatal(err)
	}

	msg := openProject(project)()
	opened, ok := msg.(projectOpenedMsg)
	if !ok {
		t.Fatalf("expected projectOpenedMsg, got %T", msg)
	}
	if opened.err != nil {
		t.Fatal(opened.err)
	}
	if _, err := os.Stat(filepath.Join(project, "local-sources")); !os.IsNotExist(err) {
		t.Fatalf("opening a project should not create local-sources, stat err=%v", err)
	}
}

func TestOpenProjectReadsLegacyRootLayoutAfterLinerMetadata(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("artifact: liner\nversion: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "tape.yaml"), []byte("title: Legacy\nversion: 1\ncurator: Arturo\ndescription: Demo\nsources: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := openProject(project)()
	opened, ok := msg.(projectOpenedMsg)
	if !ok {
		t.Fatalf("expected projectOpenedMsg, got %T", msg)
	}
	if opened.err != nil {
		t.Fatal(opened.err)
	}
	if opened.tape.Title != "Legacy" {
		t.Fatalf("expected legacy tape to open, got %+v", opened.tape)
	}
}

func TestSavedSourcesStartClarificationBeforeResearch(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	m := Model{
		screen:      screenSourceReview,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		sourceInput: textinput.New(),
		sourceTable: newSourceTable(80, 8),
		clarifyArea: newClarifyArea(64),
		createDraft: createDraft{AddSources: true},
	}

	next, _ := m.Update(sourceSavedMsg{
		preview: source.Preview{
			Sources: []tape.Source{{Type: "web", URL: "https://example.com"}},
		},
	})
	got := next.(Model)

	if got.screen != screenClarify {
		t.Fatalf("expected saved sources to lead to clarification, got %v", got.screen)
	}
	if !got.clarifyLoading {
		t.Fatal("expected clarification questions to load after saving sources")
	}
	if len(got.researchLines) != 0 {
		t.Fatalf("research should not start before clarification, got lines: %#v", got.researchLines)
	}
}

func TestProjectPrimaryActionAnswersInterruptedClarificationBeforeCorpus(t *testing.T) {
	project := t.TempDir()
	jtbd := "When looking at moodboards, I want to transfer the visual direction into a UI."
	current := tape.Tape{Title: "Art Director", JTBD: &jtbd, Sources: []tape.Source{}}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	working := filepath.Join(project, "working")
	if err := os.MkdirAll(working, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(working, "01-jtbd-and-knowledge-map.md"), []byte("TODO — Phase 1 replaces this with 4–8 sections.\nThe example bullets below are placeholders."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		currentPath: project,
		currentTape: current,
		clarifyArea: newClarifyArea(64),
	}

	if got := m.projectPrimaryLabel(); got != "Answer clarification questions" {
		t.Fatalf("expected clarification primary action, got %q", got)
	}
	if got := m.projectMilestoneNextAction(); !strings.Contains(got, "clarification questions") {
		t.Fatalf("expected clarification next action, got %q", got)
	}
	done, total, next := m.projectProgressCounts()
	if done != 0 || total == 0 || next != "Clarification" {
		t.Fatalf("expected clarification to be the missing next step, got done=%d total=%d next=%q", done, total, next)
	}

	got, cmd := m.primaryProjectAction()
	if got.screen != screenClarify || !got.clarifyLoading {
		t.Fatalf("expected Project Enter to load clarification, got screen=%v loading=%v", got.screen, got.clarifyLoading)
	}
	if cmd == nil {
		t.Fatal("expected missing draft to generate clarification questions")
	}
}

func TestProjectPrimaryActionRestoresClarificationDraft(t *testing.T) {
	project := t.TempDir()
	jtbd := "When looking at moodboards, I want to transfer the visual direction into a UI."
	current := tape.Tape{Title: "Art Director", JTBD: &jtbd, Sources: []tape.Source{}}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	if err := writeClarificationDraft(project, clarificationDraft{
		JTBD:      jtbd,
		Questions: []string{"Which visual references should anchor the research?", "What UI artifact should the mixtape help create?"},
		Answers:   []string{"Mood boards and editorial posters", "A website interface"},
		Step:      1,
	}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		currentPath: project,
		currentTape: current,
		clarifyArea: newClarifyArea(64),
	}

	got, cmd := m.primaryProjectAction()
	if cmd != nil {
		t.Fatalf("expected saved draft to restore without generating new questions")
	}
	if got.screen != screenClarify || got.clarifyLoading {
		t.Fatalf("expected restored clarification screen, got screen=%v loading=%v", got.screen, got.clarifyLoading)
	}
	if got.clarifyStep != 1 {
		t.Fatalf("expected draft to restore question index 1, got %d", got.clarifyStep)
	}
	if got.clarifyArea.Value() != "A website interface" {
		t.Fatalf("expected current answer to restore, got %q", got.clarifyArea.Value())
	}
}

func TestSourceIngestedGitHubRowsStayVisible(t *testing.T) {
	m := Model{
		screen:      screenSources,
		width:       118,
		currentPath: t.TempDir(),
		sourceInput: textinput.New(),
		sourceTable: newSourceTable(100, 8),
	}
	items := source.Stage([]tape.Source{
		{Type: "web", URL: "https://github.com/dlvhdr/gh-dash"},
		{Type: "web", URL: "https://github.com/yorukot/superfile"},
	}, true)

	next, _ := m.Update(sourceIngestedMsg{items: items})
	got := next.(Model)
	rows := got.sourceTable.Rows()

	if gotCount := len(rows); gotCount != 2 {
		t.Fatalf("expected two visible source rows, got %d: %#v", gotCount, rows)
	}
	for _, url := range []string{"https://github.com/dlvhdr/gh-dash", "https://github.com/yorukot/superfile"} {
		found := false
		for _, row := range rows {
			if len(row) > 2 && row[2] == url {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected source table to include %q, rows: %#v", url, rows)
		}
	}
	if strings.Contains(got.sourceTable.View(), "No sources recognized yet") {
		t.Fatalf("source table fell back to empty state:\n%s", got.sourceTable.View())
	}
	if got.visibleSourceCount() != 2 {
		t.Fatalf("expected banner count to include staged sources, got %d", got.visibleSourceCount())
	}
}

func TestSourceReviewUsesLocalSourcesTitleAndStatusIcons(t *testing.T) {
	m := Model{
		screen:      screenSourceReview,
		width:       118,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(100, 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{
		{Type: "web", URL: "https://developer.apple.com/design/human-interface-guidelines/"},
	}, true))

	view := m.viewSourceReview()
	for _, expected := range []string{"Review Local Sources", "✓", "Actions", "Toggle selected source"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("review view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Space toggles") {
		t.Fatalf("review view should not explain shortcuts in prose:\n%s", view)
	}
	if strings.Contains(view, "Review Import") {
		t.Fatalf("review view should not use ambiguous import title:\n%s", view)
	}
}

func TestSourceReviewSpaceTogglesActiveStatusIcon(t *testing.T) {
	m := Model{
		screen:      screenSourceReview,
		width:       118,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(100, 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{
		{Type: "web", URL: "https://developer.apple.com/design/human-interface-guidelines/"},
	}, true))

	got, _ := m.handleSourceReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Text: " "}))
	rows := got.sourceTable.Rows()

	if len(rows) != 1 || len(rows[0]) == 0 || rows[0][0] != "○" {
		t.Fatalf("expected source to render inactive marker after space, rows: %#v", rows)
	}
	if got.sourceItems[0].Active {
		t.Fatal("expected source item to be inactive after space toggle")
	}
	if !strings.Contains(got.note, "deactivated") {
		t.Fatalf("expected toggle feedback note, got %q", got.note)
	}
}

func TestClarificationSavedStartsResearchEvenAfterCustomSources(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	m := Model{
		createDraft: createDraft{AddSources: true},
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	}

	next, _ := m.Update(clarificationSavedMsg{
		tape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	})
	got := next.(Model)

	if got.screen != screenResearch {
		t.Fatalf("expected clarification to start research, got %v", got.screen)
	}
	if len(got.researchLines) == 0 {
		t.Fatal("expected research to start after clarification is saved")
	}
}

func TestReportWithPersonalSourcesDoesNotRequestSourceBoard(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	items := source.Stage([]tape.Source{
		{Type: "web", URL: "https://github.com/dlvhdr/gh-dash"},
		{Type: "web", URL: "https://github.com/yorukot/superfile"},
	}, true)

	report := reportSummary(tape.Tape{Title: "Launch", JTBD: &jtbd}, items)

	for _, expected := range []string{"Personal sources", "Added: 2", "Active: 2", "Inactive: 0", fullMethodologyAction} {
		if !strings.Contains(report, expected) {
			t.Fatalf("report missing %q:\n%s", expected, report)
		}
	}
	for _, disallowed := range []string{"Reviewed:", "Approved:", "Discarded:", "source board"} {
		if strings.Contains(report, disallowed) {
			t.Fatalf("report should not include %q for personal sources:\n%s", disallowed, report)
		}
	}
	if got := reportNextAction(tape.Tape{Title: "Launch", JTBD: &jtbd}, items); got != fullMethodologyAction {
		t.Fatalf("expected report next action to run methodology, got %q", got)
	}
}

func TestReportClarificationsSplitQuestionAndAnswer(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	current := tape.Tape{
		Title: "Launch",
		JTBD:  &jtbd,
		JTBDClarifications: []tape.Clarification{
			{
				Question: "Which Charm tools should the mixtape cover?",
				Answer:   "Bubble Tea, Lip Gloss, Bubbles, Huh, and Glamour.",
			},
		},
	}

	report := reportSummary(current, nil)
	if !strings.Contains(report, "Question: Which Charm tools should the mixtape cover?") {
		t.Fatalf("report missing question line:\n%s", report)
	}
	if !strings.Contains(report, "Answer: Bubble Tea, Lip Gloss, Bubbles, Huh, and Glamour.") {
		t.Fatalf("report missing answer line:\n%s", report)
	}
	if strings.Contains(report, "Question: Which Charm tools should the mixtape cover? Answer:") {
		t.Fatalf("question and answer should not be collapsed onto one line:\n%s", report)
	}
}

func TestStyledReportClarificationAnswerUsesBodyColor(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	current := tape.Tape{
		Title: "Launch",
		JTBD:  &jtbd,
		JTBDClarifications: []tape.Clarification{
			{
				Question: "Which Charm tools should the mixtape cover?",
				Answer:   "Bubble Tea.",
			},
		},
	}

	body := renderReportBody(current, nil, 100)
	want := styles.ReportListAccent.Render("A ") + styles.ReportBody.Render("Bubble Tea.")
	if !strings.Contains(body, want) {
		t.Fatalf("styled report should use orange A marker and normal answer text:\n%s", body)
	}
	bad := styles.NextActionTitle.Render("A ") + styles.NextActionTitle.Render("Bubble Tea.")
	if strings.Contains(body, bad) {
		t.Fatalf("styled report should not render the answer text as orange action copy:\n%s", body)
	}
}

func TestReportSummaryWithoutSourcesSkipsSourceReview(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	report := reportSummary(tape.Tape{Title: "Launch", JTBD: &jtbd}, nil)

	for _, disallowed := range []string{"Reviewed:", "Approved:", "Discarded:", "source board"} {
		if strings.Contains(report, disallowed) {
			t.Fatalf("report should not include %q when no sources exist:\n%s", disallowed, report)
		}
	}
	for _, expected := range []string{"AI-agent goal", "What happened", "Next", fullMethodologyAction} {
		if !strings.Contains(report, expected) {
			t.Fatalf("report missing %q:\n%s", expected, report)
		}
	}
}

func TestReportBodyKeepsNextActionOutOfReviewBox(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	current := tape.Tape{Title: "Launch", JTBD: &jtbd}

	body := renderReportBody(current, nil, 100)
	if strings.Contains(body, "Next") || strings.Contains(body, fullMethodologyAction) {
		t.Fatalf("report body should not include next action:\n%s", body)
	}

	next := renderReportNextAction(current, nil)
	if !strings.Contains(next, "Next") || !strings.Contains(next, fullMethodologyAction) {
		t.Fatalf("next action missing expected copy:\n%s", next)
	}
	if strings.Contains(next, "\n") {
		t.Fatalf("next action should render as one inline cue:\n%s", next)
	}
}

func TestNextCueUsesSofterOrangeThanActionAccent(t *testing.T) {
	next := renderNextCue("Add sources.")
	if !strings.Contains(next, styles.NextCueTitle.Render("> Next:")) {
		t.Fatalf("Next should use the softer orange cue style:\n%s", next)
	}
	if strings.Contains(next, styles.NextActionTitle.Render("> Next:")) {
		t.Fatalf("Next should not use the main action accent:\n%s", next)
	}
	complete := renderNextCue(projectCompleteNextAction)
	if !strings.Contains(complete, styles.NextCueTitle.Render("> Project complete")) {
		t.Fatalf("Project complete should render as a completed-state cue:\n%s", complete)
	}
	if strings.Contains(complete, "Next:") {
		t.Fatalf("Project complete cue should not present a next step:\n%s", complete)
	}

	m := Model{researchDone: true, note: "Review the corpus artifacts on disk."}
	cue := m.methodologyCue()
	if !strings.Contains(cue, styles.NextCueTitle.Render("Next: ")) {
		t.Fatalf("methodology Next cue should use the softer orange style:\n%s", cue)
	}
	if strings.Contains(cue, styles.NextActionTitle.Render("Next: ")) {
		t.Fatalf("methodology Next cue should not use the main action accent:\n%s", cue)
	}
}

func TestProgressStatusBlockFitsWidth(t *testing.T) {
	width := 80
	view := renderProgressStatusBlock(
		width,
		newTaskProgressBar(taskProgressWidth(width)),
		0.5,
		"Working",
		"Generating LINER.md.",
		"1/3 steps",
	)
	plain := stripANSICodesForTest(view)
	for _, expected := range []string{"Working", "Generating LINER.md.", "1/3 steps"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("progress block missing %q:\n%s", expected, plain)
		}
	}
	assertViewLinesFit(t, view, width)
}

func TestLoadingTitlePatternAddsSpinnerOnlyWhileRunning(t *testing.T) {
	loading := renderLoadingTitle("Compile Console", true, "⣾ ")
	idle := renderLoadingTitle("Compile Console", false, "⣾ ")
	invalidSpinner := renderLoadingTitle("Compile Console", true, "(error)")

	if strings.TrimSpace(stripANSICodesForTest(loading)) == "Compile Console" {
		t.Fatalf("loading title should include spinner:\n%s", loading)
	}
	if strings.TrimSpace(stripANSICodesForTest(idle)) != "Compile Console" {
		t.Fatalf("idle title should omit spinner:\n%s", idle)
	}
	if strings.TrimSpace(stripANSICodesForTest(invalidSpinner)) != "Compile Console" {
		t.Fatalf("invalid spinner should be ignored:\n%s", invalidSpinner)
	}
}

func TestBoldStyleIsReservedForTitles(t *testing.T) {
	for _, styled := range []string{
		styles.Title.Render("Project"),
		styles.ReportSection.Render("Health"),
	} {
		if !hasBoldANSICode(styled) {
			t.Fatalf("title style should be bold: %q", styled)
		}
	}
	for label, styled := range map[string]string{
		"section":     styles.Section.Render("Quality"),
		"subtitle":    styles.Subtitle.Render("Explains the section."),
		"next-cue":    styles.NextCueTitle.Render("> Next:"),
		"next-action": styles.NextActionTitle.Render(">"),
		"activity":    styles.ActivityPrompt.Render(">"),
	} {
		if hasBoldANSICode(styled) {
			t.Fatalf("%s style should not be bold: %q", label, styled)
		}
	}
}

func hasBoldANSICode(value string) bool {
	return strings.Contains(value, "\x1b[1m") || strings.Contains(value, "\x1b[1;")
}

func TestReportEnterStartsMethodologySafely(t *testing.T) {
	m := Model{
		screen:      screenReport,
		width:       118,
		currentTape: tape.Tape{Title: "Launch"},
	}

	next, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if next.screen == screenCompile {
		t.Fatalf("report enter should not compile before methodology runs")
	}
	if next.screen != screenResearch {
		t.Fatalf("expected report enter to open methodology, got %v", next.screen)
	}
	if !strings.Contains(next.err, "project path") {
		t.Fatalf("expected safe project path guard, got %q", next.err)
	}
}

func TestProjectPrimaryActionAfterNoSourceResearchIsMethodology(t *testing.T) {
	m := Model{
		screen:        screenProject,
		width:         118,
		currentPath:   "/tmp/liner-demo",
		researchReady: true,
		currentTape: tape.Tape{
			Title:       "Launch",
			Description: "A generated description.",
			Sources:     nil,
		},
	}

	view := m.viewProject()
	if strings.Contains(view, "Next:") {
		t.Fatalf("project panel should leave next action to the global footer:\n%s", view)
	}
	if got := m.nextAction(); got != "Continue Corpus Creation." {
		t.Fatalf("expected milestone next action to continue corpus creation, got %q", got)
	}
}

func TestProjectPrimaryActionWithJTBDAndNoSourcesRunsMethodology(t *testing.T) {
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       118,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	}

	got, _ := m.primaryProjectAction()

	if got.screen != screenResearch {
		t.Fatalf("expected no-custom-source project to start methodology, got %v", got.screen)
	}
	if got.screen == screenSources {
		t.Fatalf("no-custom-source project should not force Add Sources")
	}
}

func TestProjectViewWithJTBDAndNoSourcesShowsClarificationAction(t *testing.T) {
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	}

	view := m.viewProject()
	for _, expected := range []string{"Started", "Primary action", "Answer clarification questions", "Missing next", "Clarification"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "required before methodology") {
		t.Fatalf("no-custom-source project should not describe sources as required:\n%s", view)
	}
	if got := m.nextAction(); !strings.Contains(got, "clarification questions") {
		t.Fatalf("expected clarification next action, got %q", got)
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "enter") {
		t.Fatalf("expected enter help to continue clarification, got %#v", help)
	}
	if !hasHelpDesc(help, "clarify goal") {
		t.Fatalf("expected enter help to clarify goal, got %#v", help)
	}
}

func TestProjectHealthShowsAIRunnerReadinessForMethodology(t *testing.T) {
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		settings: settingsInfo{
			ConfigPath: "/tmp/.liner/config.yaml",
			EnvAgent:   "codex",
			Installed:  []string{"codex"},
		},
	}

	view := m.viewProject()
	for _, expected := range []string{"AI runner", "Codex"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project health missing AI runner readiness %q:\n%s", expected, view)
		}
	}
}

func TestProjectHealthShowsStatusFailureDiagnostics(t *testing.T) {
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{
			Title: "Launch",
			Sources: []tape.Source{
				{Type: "web", URL: "https://example.com"},
			},
		},
		runner:    core.Runner{Command: "/tmp/liner-core"},
		statusErr: "exit status 2: no such option: --no-write",
	}

	view := m.viewProject()
	for _, expected := range []string{
		"local fallback; status failed",
		"Status note",
		"status failed; using local project files",
		"Status cause",
		"no such option",
		"Core binary",
		"/tmp/liner-core",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project health missing status diagnostic %q:\n%s", expected, view)
		}
	}
}

func TestProjectHealthDiagnosticsFitNarrowWidth(t *testing.T) {
	width := 80
	m := Model{
		screen:      screenProject,
		width:       width,
		currentPath: "/tmp/liner/project-with-a-long-folder-name",
		currentTape: tape.Tape{
			Title:       "Launch With A Long Title",
			Description: "A generated description for the project workspace that should wrap cleanly.",
			Sources: []tape.Source{
				{Type: "web", URL: "https://example.com/sources/with/a/long/path"},
			},
		},
		runner:    core.Runner{Command: "/tmp/liner-core-with-a-long-path/liner"},
		statusErr: "exit status 2: no such option: --no-write on an older bundled core",
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	assertViewLinesFit(t, view, styles.ClampWidth(width-4))
}

func TestProjectHealthExplainsLocalFileFallback(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "working", "01-jtbd-and-knowledge-map.md"), []byte("# Framing\n\nReal framing output."), 0o644); err != nil {
		t.Fatal(err)
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	}

	view := m.viewProject()
	for _, expected := range []string{
		"local project files",
		"Status note",
		"using local files until liner status returns evidence",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project health missing local-inference note %q:\n%s", expected, view)
		}
	}
}

func TestProjectViewShowsRunEstimateForMethodologyAction(t *testing.T) {
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		projectPane: 5,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Usage", "Estimates the next corpus build", "Scope", "Estimate", "Basis", "next: Framing", "~350k tokens", "full corpus build", "~7M-9M tokens", "seed baseline"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project run estimate missing %q:\n%s", expected, view)
		}
	}
}

func TestProjectRunEstimateSkipsPendingGateWithoutAccepting(t *testing.T) {
	project := t.TempDir()
	if _, err := linerprogress.MarkPhaseComplete(project, linerprogress.PhaseFraming); err != nil {
		t.Fatal(err)
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		projectPane: 5,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Usage", "Estimates the next corpus build", "next: Candidate discovery", "~3.2M tokens"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project run estimate missing %q:\n%s", expected, view)
		}
	}
	if linerprogress.ReadGateState(project).Gate0Accepted {
		t.Fatal("viewing the run estimate should not accept gate0")
	}
}

func TestProjectViewShowsCompactMethodologyProgress(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "working", "01-jtbd-and-knowledge-map.md"), []byte("# Framing\n\nReal framing output."), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := linerprogress.MarkPhaseComplete(project, linerprogress.PhaseFraming); err != nil {
		t.Fatal(err)
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		projectPane: 1,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Flow", "1 of 4 steps complete", "Project Shell", "done", "Corpus Ready", "current", "Build Corpus", "Create Operating Layer", "queued"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project methodology progress missing %q:\n%s", expected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(m.width-4); got > want {
			t.Fatalf("project progress line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
	if linerprogress.ReadGateState(project).Gate0Accepted {
		t.Fatal("viewing methodology progress should not accept gate0")
	}
}

func TestProjectMethodologyProgressHiddenBeforeSignal(t *testing.T) {
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	if strings.Contains(view, "Corpus build progress") {
		t.Fatalf("project progress should stay hidden before methodology has a signal:\n%s", view)
	}
}

func TestProjectMethodologyProgressHiddenForSourceOnlyProject(t *testing.T) {
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{
			Title: "Launch",
			Sources: []tape.Source{
				{Type: "web", URL: "https://example.com", Priority: "required"},
			},
		},
	}

	view := m.viewProject()
	if strings.Contains(view, "Corpus build progress") {
		t.Fatalf("source-only project should not imply methodology progress:\n%s", view)
	}
}

func TestProjectMethodologyProgressHiddenForCompiledOnlyProject(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Launch\n\nReady."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       160,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	if strings.Contains(view, "Corpus build progress") {
		t.Fatalf("compiled-only project should not imply methodology progress:\n%s", view)
	}
}

func TestProjectMethodologyProgressUsesLoadedStatus(t *testing.T) {
	project := t.TempDir()
	exitCode := 0
	logPath := ".liner-runs/framing/run.jsonl"
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		statusPath:  project,
		projectPane: 1,
		status: &core.ProjectStatus{
			Progress: core.StatusProgress{Step: 2, Total: 10, Source: "file"},
			Phases: []core.StatusPhase{
				{
					ID:     "framing",
					Label:  "Framing",
					Index:  0,
					Status: "complete",
					Artifact: &core.StatusArtifact{
						Path:           "working/01-jtbd-and-knowledge-map.md",
						Exists:         true,
						Bytes:          120,
						HasRealContent: true,
					},
					Runs: core.StatusRuns{Count: 1, LatestExitCode: &exitCode, LatestLogPath: &logPath},
				},
				{
					ID:     "gate0",
					Label:  "Confirm framing",
					Index:  1,
					Status: "complete",
					Gate:   &core.StatusGate{Key: "gate0Accepted", Accepted: true},
				},
				{
					ID:     "candidates",
					Label:  "Candidate discovery",
					Index:  2,
					Status: "in_progress",
					Artifact: &core.StatusArtifact{
						Path:           "working/02-candidate-longlist.md",
						Exists:         true,
						Bytes:          80,
						HasRealContent: false,
					},
				},
			},
		},
	}

	view := m.viewProject()
	for _, expected := range []string{"Flow", "1 of 4 steps complete", "Project Shell", "done", "Corpus Ready", "current", "add sources", "Create Operating Layer", "queued"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project status-backed progress missing %q:\n%s", expected, view)
		}
	}
}

func TestProjectStatusLoadedIgnoresStaleProject(t *testing.T) {
	m := Model{
		screen:      screenProject,
		currentPath: "/tmp/current",
	}

	next, _ := m.Update(projectStatusLoadedMsg{
		path:   "/tmp/other",
		status: core.ProjectStatus{Progress: core.StatusProgress{Step: 2}},
	})
	got := next.(Model)

	if got.status != nil || got.statusPath != "" {
		t.Fatalf("stale status should be ignored, got path %q status %#v", got.statusPath, got.status)
	}
}

func TestProjectStatusLoadedStoresCurrentProjectErrorAsFallbackOnly(t *testing.T) {
	m := Model{
		screen:      screenProject,
		currentPath: "/tmp/current",
	}

	next, _ := m.Update(projectStatusLoadedMsg{
		path: "/tmp/current",
		err:  fmt.Errorf("status unavailable"),
	})
	got := next.(Model)

	if got.status != nil {
		t.Fatalf("failed status load should not leave status payload: %#v", got.status)
	}
	if got.statusErr != "status unavailable" {
		t.Fatalf("expected stored status error for diagnostics, got %q", got.statusErr)
	}
	if got.err != "" {
		t.Fatalf("status fallback should not show a global error, got %q", got.err)
	}
}

func TestProjectRunEstimateUsesLocalPhaseMedian(t *testing.T) {
	library := t.TempDir()
	project := filepath.Join(library, "current")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := linerprogress.MarkPhaseComplete(project, linerprogress.PhaseFraming); err != nil {
		t.Fatal(err)
	}
	for index, tokens := range []int{9000, 1000, 5000} {
		sampleProject := filepath.Join(library, fmt.Sprintf("sample-%d", index))
		writePhaseTokenLog(t, sampleProject, "candidates", "run", tokens)
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		baseDir:     library,
		screen:      screenProject,
		width:       160,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		projectPane: 5,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Usage", "Estimates the next corpus build", "next: Candidate discovery", "~5k tokens", "local median (3)"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project local run estimate missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "~3.2M tokens") {
		t.Fatalf("local median should replace the candidate seed estimate:\n%s", view)
	}
}

func TestProjectRunEstimateUsesGlobalPhaseMedianWhenLocalSparse(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "run-estimates.jsonl")
	t.Setenv("LINER_ESTIMATE_HISTORY", historyPath)

	sourceLibrary := t.TempDir()
	sourceProject := filepath.Join(sourceLibrary, "current")
	if err := os.MkdirAll(sourceProject, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := linerprogress.MarkPhaseComplete(sourceProject, linerprogress.PhaseFraming); err != nil {
		t.Fatal(err)
	}
	for index, tokens := range []int{11000, 3000, 7000} {
		sampleProject := filepath.Join(sourceLibrary, fmt.Sprintf("source-sample-%d", index))
		writePhaseTokenLog(t, sampleProject, "candidates", "run", tokens)
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	source := Model{
		baseDir:     sourceLibrary,
		screen:      screenProject,
		width:       160,
		currentPath: sourceProject,
		currentTape: tape.Tape{Title: "Source", JTBD: &jtbd},
		projectPane: 5,
	}
	_ = source.viewProject()

	targetLibrary := t.TempDir()
	targetProject := filepath.Join(targetLibrary, "current")
	if err := os.MkdirAll(targetProject, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := linerprogress.MarkPhaseComplete(targetProject, linerprogress.PhaseFraming); err != nil {
		t.Fatal(err)
	}
	target := Model{
		baseDir:     targetLibrary,
		screen:      screenProject,
		width:       160,
		currentPath: targetProject,
		currentTape: tape.Tape{Title: "Target", JTBD: &jtbd},
		projectPane: 5,
	}

	view := target.viewProject()
	for _, expected := range []string{"Usage", "Estimates the next corpus build", "next: Candidate discovery", "~7k tokens", "global median (3)"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project global run estimate missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "~3.2M tokens") {
		t.Fatalf("global median should replace the candidate seed estimate:\n%s", view)
	}
}

func TestProjectRunEstimateUsesLocalFullMethodologyMedians(t *testing.T) {
	library := t.TempDir()
	project := filepath.Join(library, "current")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, phase := range methodologyPhaseOrder {
		for index, tokens := range []int{3000, 1000, 2000} {
			sampleProject := filepath.Join(library, fmt.Sprintf("%s-sample-%d", phase, index))
			writePhaseTokenLog(t, sampleProject, phase, "run", tokens)
		}
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		baseDir:     library,
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		projectPane: 5,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Usage", "Estimates the next corpus build", "next: Framing", "~2k tokens", "full corpus build", "~12k tokens", "local medians"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project full local run estimate missing %q:\n%s", expected, view)
		}
	}
}

func TestProjectRunEstimateUsesGlobalFullMethodologyMedians(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "run-estimates.jsonl")
	t.Setenv("LINER_ESTIMATE_HISTORY", historyPath)

	sourceLibrary := t.TempDir()
	sourceProject := filepath.Join(sourceLibrary, "current")
	if err := os.MkdirAll(sourceProject, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, phase := range methodologyPhaseOrder {
		for index, tokens := range []int{3000, 1000, 2000} {
			sampleProject := filepath.Join(sourceLibrary, fmt.Sprintf("%s-source-%d", phase, index))
			writePhaseTokenLog(t, sampleProject, phase, "run", tokens)
		}
	}
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	source := Model{
		baseDir:     sourceLibrary,
		screen:      screenProject,
		width:       100,
		currentPath: sourceProject,
		currentTape: tape.Tape{Title: "Source", JTBD: &jtbd},
		projectPane: 5,
	}
	_ = source.viewProject()

	targetLibrary := t.TempDir()
	targetProject := filepath.Join(targetLibrary, "current")
	if err := os.MkdirAll(targetProject, 0o755); err != nil {
		t.Fatal(err)
	}
	target := Model{
		baseDir:     targetLibrary,
		screen:      screenProject,
		width:       100,
		currentPath: targetProject,
		currentTape: tape.Tape{Title: "Target", JTBD: &jtbd},
		projectPane: 5,
	}

	view := target.viewProject()
	for _, expected := range []string{"Usage", "Estimates the next corpus build", "next: Framing", "~2k tokens", "full corpus build", "~12k tokens", "global medians"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project full global run estimate missing %q:\n%s", expected, view)
		}
	}
}

func TestClearEstimateHistoryRemovesGlobalEntriesAndBlocksOldLocalResync(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "run-estimates.jsonl")
	t.Setenv("LINER_ESTIMATE_HISTORY", historyPath)
	library := t.TempDir()
	project := filepath.Join(library, "sample")
	writePhaseTokenLog(t, project, "framing", "old-run", 1000)
	records := localPhaseTokenRecords(library, "")

	syncGlobalEstimateHistory(records)
	if got := len(readGlobalEstimateEntries()); got != 1 {
		t.Fatalf("expected one synced global estimate entry, got %d", got)
	}

	path, count, err := clearGlobalEstimateHistory()
	if err != nil {
		t.Fatal(err)
	}
	if path != historyPath || count != 1 {
		t.Fatalf("unexpected clear result: path=%q count=%d", path, count)
	}
	if _, err := os.Stat(historyPath); !os.IsNotExist(err) {
		t.Fatalf("expected history file removed, stat err=%v", err)
	}
	if _, err := os.Stat(globalEstimateResetPath()); err != nil {
		t.Fatalf("expected reset marker, stat err=%v", err)
	}

	syncGlobalEstimateHistory(records)
	if got := len(readGlobalEstimateEntries()); got != 0 {
		t.Fatalf("old local logs should not immediately re-seed cleared history, got %d entries", got)
	}
}

func writePhaseTokenLog(t *testing.T, project string, phase string, name string, inputTokens int) {
	t.Helper()
	dir := filepath.Join(project, ".liner-runs", phase)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"type":"turn.completed","usage":{"input_tokens":%d,"output_tokens":0}}`+"\n", inputTokens)
	if err := os.WriteFile(filepath.Join(dir, name+".jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeGlobalEstimateHistory(t *testing.T, path string, entries []globalEstimateEntry) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(data))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectLocalSourcesRowWaitsUntilFolderExists(t *testing.T) {
	project := t.TempDir()
	m := Model{
		screen:      screenProject,
		width:       118,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	for _, expected := range []string{"Launch", "Guidance for Launch.", "Health", "Field", "Value", "Status", "Sources", "Local files", "not created yet"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project workspace summary missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Current Liner project", "Mixtape"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("project workspace should not show old header/table copy %q:\n%s", unexpected, view)
		}
	}
	if strings.Contains(view, filepath.Join(project, "local-sources")) {
		t.Fatalf("project view should not show a missing local-sources path:\n%s", view)
	}

	if err := os.MkdirAll(filepath.Join(project, "local-sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	view = m.viewProject()
	if !strings.Contains(view, "Local files") || !strings.Contains(view, "local-sources") {
		t.Fatalf("project view should show existing local-sources path:\n%s", view)
	}
	if strings.Contains(view, "not created yet") {
		t.Fatalf("project view should stop showing the placeholder after local-sources exists:\n%s", view)
	}
}

func TestProjectViewSurfacesDroppedYouTubeEvaluationIssues(t *testing.T) {
	project := t.TempDir()
	working := filepath.Join(project, "working")
	if err := os.MkdirAll(working, 0o755); err != nil {
		t.Fatal(err)
	}
	evaluation := `candidates:
  - url: https://www.youtube.com/watch?v=one11111111
    title: YouTube source one
    decision: dropped
    section: Foundations
    rationale: YouTube returned no readable transcript body, and yt-dlp returned 429.
  - url: https://www.youtube.com/live/two22222222
    title: YouTube live source two
    decision: dropped
    section: Foundations
    rationale: The direct YouTube fetch produced no readable transcript and recovery failed.
  - url: https://example.com/kept
    decision: kept
    fetch_status: readable
    content_quality: high
    evidence:
      - The source includes a concrete example.
      - The source names a practical limitation.
`
	if err := os.WriteFile(filepath.Join(working, "03-evaluation.yaml"), []byte(evaluation), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       150,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Launch",
			Sources: []tape.Source{
				{Type: "web", URL: "https://example.com/kept", Priority: "required"},
			},
		},
	}

	view := m.viewProject()
	for _, expected := range []string{"Evaluation issues", "2 YouTube", "transcript/access", "working/03-evaluation.yaml"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project health should surface dropped YouTube issue %q:\n%s", expected, view)
		}
	}

	m.projectPane = 2
	view = m.viewProject()
	for _, expected := range []string{"Sources", "Dropped candidates", "2 YouTube", "transcript/access"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project sources should surface dropped YouTube issue %q:\n%s", expected, view)
		}
	}
}

func TestProjectViewWithoutJTBDStillRequestsSources(t *testing.T) {
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	for _, expected := range []string{"Started", "Primary action", "Add sources"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project view missing %q:\n%s", expected, view)
		}
	}
	if got := m.nextAction(); got != "Continue Corpus Creation: add sources." {
		t.Fatalf("expected add-source next action, got %q", got)
	}
}

func TestProjectSourcePrimaryDoesNotDuplicateAddSourcesAction(t *testing.T) {
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	if !strings.Contains(view, "Primary action") || !strings.Contains(view, "Add sources") {
		t.Fatalf("source-primary project should show Add sources as the primary action:\n%s", view)
	}
	help := m.helpForScreen()
	if hasHelp(help.ShortHelp(), "a") || hasHelp(help.FullHelp()[0], "a") {
		t.Fatalf("source-primary project help should not advertise duplicate a/add sources controls: %#v %#v", help.ShortHelp(), help.FullHelp()[0])
	}
}

func TestProjectAddMoreSourcesStaysAvailableWhenMethodologyIsPrimary(t *testing.T) {
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	}

	view := m.viewProject()
	if !strings.Contains(view, "Sources") {
		t.Fatalf("methodology-primary project should keep source entry available:\n%s", view)
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "a") {
		t.Fatal("methodology-primary project should keep a/add sources in help")
	}
}

func TestProjectPrimaryActionRoutesPendingAssemblyDraft(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, assemblyDraftRelPath), []byte(`sources:
  - type: web
    url: https://example.com
    priority: required
`), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{
			Title:   "Launch",
			Sources: nil,
		},
		sourceTable: newSourceTable(100, 8),
	}

	got, _ := m.primaryProjectAction()

	if got.screen != screenAssemblyReview {
		t.Fatalf("expected pending draft to open assembly review, got %v: %s", got.screen, got.err)
	}
	if len(got.sourceItems) != 1 {
		t.Fatalf("expected draft source to be staged, got %#v", got.sourceItems)
	}
}

func TestProjectViewShowsPendingAssemblyDraftAsPrimaryAction(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, assemblyDraftRelPath), []byte(`sources:
  - type: web
    url: https://example.com
`), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	for _, expected := range []string{"Started", "Review draft sources"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project view missing %q:\n%s", expected, view)
		}
	}
	if got := m.nextAction(); got != "Review the assembly draft sources." {
		t.Fatalf("expected draft next action, got %q", got)
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "enter") {
		t.Fatalf("expected enter help to review draft, got %#v", help)
	}
}

func TestProjectPrimaryActionCreatesOperatingLayerForCorpusReadyProject(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Launch\n\nReady."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, _ := m.primaryProjectAction()

	if got.screen != screenLinerReview {
		t.Fatalf("expected corpus-ready project to open operating-layer review, got %v: %s", got.screen, got.err)
	}
}

func TestProjectPrimaryActionCreatesOperatingLayerForCanonicalCorpusReadyProject(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\nmixtape: mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "mixtape"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "mixtape", "MIXTAPE.md"), []byte("# Launch\n\nReady."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, _ := m.primaryProjectAction()

	if got.screen != screenLinerReview {
		t.Fatalf("expected canonical corpus-ready project to open operating-layer review, got %v: %s", got.screen, got.err)
	}
}

func TestProjectViewShowsCompiledMixtapeAsReadyState(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Launch\n\nReady."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	view := m.viewProject()
	for _, expected := range []string{"Corpus Ready", "Create Operating Layer"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project view missing %q:\n%s", expected, view)
		}
	}
	if got := m.nextAction(); got != "Create Operating Layer." {
		t.Fatalf("expected corpus-ready next action, got %q", got)
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "enter") {
		t.Fatalf("expected enter help to preview, got %#v", help)
	}
}

func TestProjectCompleteDoesNotOpenLinerFromPrimaryAction(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Launch\n\nReady."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Launch Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		statusPath:  project,
		status: &core.ProjectStatus{
			Snapshot:     core.StatusSnapshot{Milestone: "project_complete"},
			ProjectSkill: core.ProjectSkillStatus{Status: "active"},
		},
	}

	if got := m.nextAction(); got != projectCompleteNextAction {
		t.Fatalf("expected project-complete next action, got %q", got)
	}
	if activity := m.viewActivity(); !strings.Contains(activity, "Project complete") || strings.Contains(activity, "Open LINER.md") || strings.Contains(activity, "Next:") {
		t.Fatalf("expected completed project activity without open-next copy:\n%s", activity)
	}
	got, cmd := m.primaryProjectAction()
	if cmd != nil {
		t.Fatalf("project complete primary action should not return a command")
	}
	if got.screen != screenProject {
		t.Fatalf("project complete primary action should stay on project, got %v", got.screen)
	}
	help := m.helpForScreen().ShortHelp()
	if hasHelp(help, "enter") {
		t.Fatalf("project complete help should not advertise enter: %#v", help)
	}
	for _, keyName := range []string{"l", "a", "o"} {
		if !hasHelp(help, keyName) {
			t.Fatalf("project complete help missing %s: %#v", keyName, help)
		}
	}
}

func TestProjectViewShowsCapabilities(t *testing.T) {
	project := t.TempDir()
	for _, dir := range []string{
		"skills",
		filepath.Join("working", "audits"),
		filepath.Join("working", "evals", "runs", "2026-06-14"),
		"children",
	} {
		if err := os.MkdirAll(filepath.Join(project, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for rel, body := range map[string]string{
		"LINER.md":                                                           "# Launch\n",
		filepath.Join("skills", "critique.md"):                               "# Skill\n",
		filepath.Join("skills", "polish.md"):                                 "# Skill\n",
		filepath.Join("working", "audits", "contradictions.md"):              "# Audit\n",
		filepath.Join("working", "evals", "runs", "2026-06-14", "result.md"): "# Eval\n",
		filepath.Join("children", "ux-specialist.yaml"):                      "path: ../ux\n",
		"lineage.yaml": "children: []\n",
	} {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		projectPane: 4,
	}

	summary := m.projectCapabilities()
	if !summary.HasLiner || summary.Skills != 2 || summary.Audits != 1 || summary.Evals != 1 || summary.Children != 1 || !summary.Lineage {
		t.Fatalf("unexpected capability summary: %#v", summary)
	}

	view := m.viewProject()
	for _, expected := range []string{
		"Operating Layer",
		"Area",
		"Status",
		"Next",
		"LINER.md",
		"ready",
		"Preview or regenerate",
		"Project Skill",
		"blocked",
		"Reach Corpus Ready first",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Audit Review", "Run inside Operating Layer", "Impact tests", "impact-test", "1 artifact ready", "Composition", "1 + lineage", "Review child"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("v1 capabilities view should hide %q:\n%s", unexpected, view)
		}
	}
	assertNoBoxCorners(t, view)
}

func TestProjectPaneListShowsOnlySectionNames(t *testing.T) {
	m := Model{
		screen: screenProject,
		width:  120,
		currentTape: tape.Tape{
			Title: "Launch",
		},
	}

	view := m.projectPaneList(30)
	for _, expected := range []string{"Sections", "Health", "Flow", "Sources", "Artifacts", "Operating Layer", "Usage"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project pane list missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Ready", "complete", "done", "runs", "basic", "0/10", "1/10"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("project pane list should not show section statuses %q:\n%s", unexpected, view)
		}
	}
}

func TestProjectCapabilitiesShowMissingReadinessNextSteps(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       120,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		projectPane: 4,
	}

	view := m.viewProject()
	for _, expected := range []string{
		"LINER.md",
		"missing",
		"Create Operating Layer",
		"Project Skill",
		"missing",
		"Create Operating Layer",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("capabilities readiness view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Audit Review", "not run", "Run inside Operating Layer", "Impact tests", "not prepared", "Create taskset", "Composition", "Nest only when scope splits"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("v1 capabilities readiness view should hide %q:\n%s", unexpected, view)
		}
	}
	assertNoBoxCorners(t, view)
}

func TestProjectCapabilitiesFitNarrowWidth(t *testing.T) {
	width := 80
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       width,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch With A Long Title"},
		projectPane: 4,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	assertViewLinesFit(t, view, styles.ClampWidth(width-4))
}

func TestParseQualityKindBalance(t *testing.T) {
	markdown := `# Quality checks

## Test 5 — Source-kind balance

Distribution: 4 reference / 6 principle / 2 prescription / 0 example

Accepted: this mixtape is intentionally pattern-heavy, and synthesis will name the missing example boundary.
`
	balance, ok := parseQualityKindBalance(markdown)
	if !ok {
		t.Fatal("expected quality kind balance to parse")
	}
	if balance.count("reference") != 4 || balance.count("principle") != 6 || balance.count("prescription") != 2 || balance.count("example") != 0 {
		t.Fatalf("unexpected parsed quality counts: %#v", balance.Counts)
	}
	if !balance.ZeroDefended {
		t.Fatal("expected accepted zero-kind rationale to count as defended")
	}
}

func TestParseQualityPerspectives(t *testing.T) {
	markdown := `# Quality checks

## Test 4 — Framing-gap

### Perspectives audit
- Designer voice — stance-represented (Source 3, verified).
- Accessibility-first viewpoint — stance-absent, concerns-addressed by Sources 2 and 9. Argued sufficient: the corpus covers implementation duties.
  - Backfill attempt: searched accessibility CLI design.
- Unix-purist stance — stance-absent, concerns-absent after one search pass. Recommendation: name the tradeoff in synthesis.

## Test 5 — Source-kind balance

Distribution: 1 reference / 1 principle / 1 prescription / 1 example
`
	perspectives, ok := parseQualityPerspectives(markdown)
	if !ok {
		t.Fatal("expected perspectives audit to parse")
	}
	if len(perspectives) != 3 {
		t.Fatalf("expected three top-level perspectives, got %#v", perspectives)
	}
	if perspectives[0].Name != "Designer voice" || perspectives[0].Status != "represented" {
		t.Fatalf("unexpected represented perspective: %#v", perspectives[0])
	}
	if perspectives[1].Name != "Accessibility-first viewpoint" || perspectives[1].Status != "concerns covered" {
		t.Fatalf("unexpected concerns-covered perspective: %#v", perspectives[1])
	}
	if perspectives[2].Name != "Unix-purist stance" || !strings.Contains(perspectives[2].Status, "gap") {
		t.Fatalf("unexpected gap perspective: %#v", perspectives[2])
	}
}

func TestProjectQualityHelpersRenderKindBalanceAndPerspectives(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	quality := `# Quality checks

## Test 1 — Redundancy
No duplicate sources.

## Test 4 — Framing-gap

### Perspectives audit
- Designer voice — stance-represented (Source 3, verified).
- Unix-purist stance — stance-absent, concerns-absent after one search pass. Recommendation: name the tradeoff in synthesis.

## Test 5 — Source-kind balance

Distribution: 4 reference / 6 principle / 2 prescription / 0 example
`
	if err := os.WriteFile(filepath.Join(project, "working", "04-quality-checks.md"), []byte(quality), 0o644); err != nil {
		t.Fatal(err)
	}
	view := projectQualityBalanceView(project, 100)
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Role", "Count", "Status", "reference", "4", "principle", "6", "prescription", "2", "example", "0", "needs defense"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project quality balance missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Perspectives audit") || strings.Contains(view, "Designer voice") {
		t.Fatalf("quality section should not include perspectives audit:\n%s", view)
	}

	view = projectQualityPerspectivesView(project, 100)
	for _, expected := range []string{"Perspective", "Coverage", "Evidence", "Designer voice", "represented", "Unix-purist stance", "gap"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project perspectives missing %q:\n%s", expected, view)
		}
	}
}

func TestReadProjectRunUsageAggregatesRunnerLogs(t *testing.T) {
	project := t.TempDir()
	for _, dir := range []string{
		filepath.Join(".liner-runs", "framing"),
		filepath.Join(".liner-runs", "quality"),
		filepath.Join(".liner-runs", "synthesis"),
	} {
		if err := os.MkdirAll(filepath.Join(project, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	logs := map[string]string{
		filepath.Join(".liner-runs", "framing", "codex.jsonl"): strings.Join([]string{
			`{"type":"_liner_meta","agent":"codex","taskLabel":"framing"}`,
			`{"type":"turn.completed","usage":{"input_tokens":1000,"cached_input_tokens":500,"output_tokens":200}}`,
			`{"type":"_liner_close","exitCode":0}`,
		}, "\n") + "\n",
		filepath.Join(".liner-runs", "quality", "result.jsonl"): strings.Join([]string{
			`{"type":"_liner_meta","agent":"claude","taskLabel":"quality"}`,
			`{"type":"result","total_cost_usd":0.25,"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}`,
			`{"type":"_liner_close","exitCode":0}`,
		}, "\n") + "\n",
		filepath.Join(".liner-runs", "synthesis", "normalized.jsonl"): strings.Join([]string{
			`{"kind":"runner_start","phaseId":"synthesis"}`,
			`{"kind":"tokens","inputTokens":20,"outputTokens":7,"cacheReadTokens":3,"cacheCreationTokens":2}`,
			`{"kind":"runner_done","code":0}`,
		}, "\n") + "\n",
	}
	for rel, body := range logs {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	usage, ok := readProjectRunUsage(project)
	if !ok {
		t.Fatal("expected run usage to parse")
	}
	if usage.Runs != 3 {
		t.Fatalf("expected three runs, got %#v", usage)
	}
	if usage.Input != 1120 || usage.Output != 257 || usage.CacheRead != 513 || usage.CacheCreate != 7 {
		t.Fatalf("unexpected token totals: %#v", usage)
	}
	if !usage.CostKnown || usage.CostUSD != 0.25 {
		t.Fatalf("expected known cost, got %#v", usage)
	}
}

func TestProjectViewShowsRunUsage(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".liner-runs", "framing"), 0o755); err != nil {
		t.Fatal(err)
	}
	log := strings.Join([]string{
		`{"type":"_liner_meta","agent":"codex","taskLabel":"framing"}`,
		`{"type":"turn.completed","usage":{"input_tokens":1200,"cached_input_tokens":300,"output_tokens":45}}`,
		`{"type":"_liner_close","exitCode":0}`,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(project, ".liner-runs", "framing", "run.jsonl"), []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Token Surface"},
		projectPane: 5,
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Usage", "Shows token usage", "Metric", "Value", "runs", "1", "total tokens", "1,545", "input", "1,200", "output", "45", "cache read", "300", "cost", "not reported"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project run usage missing %q:\n%s", expected, view)
		}
	}
}

func TestSkillsScreenDiscoversAndPreviewsSkillFiles(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte("# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n\n## Boundaries\n\nDo not use outside the corpus.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "guided.md"), []byte("# Guided\n\n## Source Grounding\n\nMIXTAPE.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "loose.md"), []byte("# Loose\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "ignore.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		skillTable:  newSkillTable(100, 8),
	}

	got, _ := m.startSkills()

	if got.screen != screenSkills {
		t.Fatalf("expected skills screen, got %v: %s", got.screen, got.err)
	}
	if len(got.skillItems) != 3 {
		t.Fatalf("expected three markdown skills, got %#v", got.skillItems)
	}
	if got.skillItems[0].Name != "critique" || got.skillItems[0].Status != "grounded" {
		t.Fatalf("expected grounded critique skill first, got %#v", got.skillItems[0])
	}
	if got.skillItems[1].Name != "guided" || got.skillItems[1].Status != "needs boundaries" {
		t.Fatalf("expected guided skill to need boundaries, got %#v", got.skillItems[1])
	}
	if got.skillItems[2].Name != "loose" || got.skillItems[2].Status != "needs grounding" {
		t.Fatalf("expected loose skill to need grounding, got %#v", got.skillItems[2])
	}
	view := got.viewSkills()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Skills", "critique", "grounded", "guided", "needs boundaries", "loose", "needs grounding", "Selected", "Field", "Value", "Path", "skills/critique.md", "Actions", "enter / o", "Write readiness report", "Draft starter skill", "Temporarily disable selected skill", "Deprecate selected skill"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("skills view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Repair grounding or boundaries") {
		t.Fatalf("grounded selected skill should not foreground repair actions:\n%s", view)
	}
	if strings.Contains(view, "n drafts. g repairs grounding") {
		t.Fatalf("skills view should not render the old long command sentence:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "n") {
		t.Fatal("skills help should expose the new skill key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "r") {
		t.Fatal("skills help should expose the readiness report key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "x") {
		t.Fatal("skills help should expose the disable/enable key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "d") {
		t.Fatal("skills help should expose the deprecate key")
	}
	got.skillTable.SetCursor(1)
	needsView := got.viewSkills()
	if !strings.Contains(needsView, "Repair grounding or boundaries") {
		t.Fatalf("needs-boundaries selected skill should show repair action:\n%s", needsView)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "g") {
		t.Fatal("skills needs-boundaries help should expose the grounding repair key")
	}
	got.skillTable.SetCursor(0)

	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen != screenPreview || got.previewRel != filepath.Join("skills", "critique.md") {
		t.Fatalf("expected enter to preview selected skill, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if action := got.nextAction(); !strings.Contains(action, "skills/critique.md") {
		t.Fatalf("expected skill preview next action, got %q", action)
	}
}

func TestCreateSkillReadinessReportFromSelectedSkill(t *testing.T) {
	project := t.TempDir()
	skillsDir := filepath.Join(project, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillsDir, "loose.md")
	original := "# Loose\n\n## Method\n\n- Give advice.\n"
	if err := os.WriteFile(skillPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		skillTable:  newSkillTable(100, 8),
	}

	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))

	if got.screen != screenPreview {
		t.Fatalf("expected readiness report preview, got %v: %s", got.screen, got.err)
	}
	if !strings.Contains(filepath.ToSlash(got.previewRel), "skill-readiness") {
		t.Fatalf("expected skill readiness preview path, got %q", got.previewRel)
	}
	matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-loose-skill-readiness.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one skill readiness report, got %v", matches)
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{
		"# Skill Readiness Alignment Report",
		"Skill: `skills/loose.md`",
		"needs grounding",
		"needs boundaries",
		"maintenance missing",
		"No skill files, source files, `MIXTAPE.md`, `LINER.md`, or `tape.yaml` were changed",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("readiness report missing %q:\n%s", expected, report)
		}
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "skill alignment" {
		t.Fatalf("expected skill alignment audit, got %#v", items)
	}
	after, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Fatalf("readiness report should not mutate skill file:\n%s", string(after))
	}
}

func TestProjectHidesSkillsFromV1DefaultSurfaces(t *testing.T) {
	project := t.TempDir()
	missing := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}
	if hasHelp(missing.helpForScreen().ShortHelp(), "k") {
		t.Fatal("project without skills should not show k in short help")
	}
	if hasCommandTitle(missing.commandItems(), "Skills") {
		t.Fatal("project without skills should not show Skills command")
	}

	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	creatable := missing
	if hasHelp(creatable.helpForScreen().ShortHelp(), "k") {
		t.Fatal("compiled project without skills should not show k in v1 short help")
	}
	if hasCommandTitle(creatable.commandItems(), "Skills") {
		t.Fatal("compiled project without skills should not show Skills command in v1")
	}
	got, _ := creatable.handleKey(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	if got.screen != screenProject {
		t.Fatalf("expected k to stay on Project in v1, got %v: %s", got.screen, got.err)
	}

	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte("# Critique\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	available := missing
	if hasHelp(available.helpForScreen().ShortHelp(), "k") {
		t.Fatal("project with parked skills should not show k in v1 short help")
	}
	if hasCommandTitle(available.commandItems(), "Skills") {
		t.Fatal("project with parked skills should not show Skills command in v1")
	}

	got, _ = available.handleKey(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	if got.screen != screenProject {
		t.Fatalf("expected k to keep parked skills out of v1 Project routes, got %v: %s", got.screen, got.err)
	}
}

func TestCreateStarterSkillDraftOpensReviewWithoutWritingSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	jtbd := "When reviewing product UI, I want a grounded critique method."
	kind := "principle"
	section := "interaction"
	m := Model{
		screen:      screenSkills,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Product Design",
			JTBD:  &jtbd,
			Sources: []tape.Source{{
				Type:    "web",
				URL:     "https://example.com/hierarchy",
				Kind:    &kind,
				Section: &section,
			}},
		},
		skillTable: newSkillTable(100, 8),
		preview:    viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}

	got, _ := m.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))

	if got.screen != screenSkillReview {
		t.Fatalf("expected skill review, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != filepath.Join(skillDraftDir, "product-design-skill-draft.md") {
		t.Fatalf("expected product design skill draft, got %q", got.previewRel)
	}
	view := got.viewSkillReview()
	for _, expected := range []string{"Review Skill Draft", "Product Design", "Purpose", "Actions", "Write skill", "Open draft", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("skill review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter writes the skill.") {
		t.Fatalf("skill review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
	draft, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Product Design Starter Skill", "Source Grounding", "https://example.com/hierarchy", "## Method", "## Boundaries"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("skill draft missing %q:\n%s", expected, draft)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "product-design-skill.md")); !os.IsNotExist(err) {
		t.Fatalf("skill should not be written before acceptance, stat err=%v", err)
	}
}

func TestAcceptStarterSkillDraftWritesSkillAndAudit(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenSkills,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
	}
	got, _ := m.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || got.previewRel != filepath.Join("skills", "product-design-skill.md") {
		t.Fatalf("expected skill preview after accept, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected draft removal after accept, stat err=%v", err)
	}
	skill, err := os.ReadFile(filepath.Join(project, "skills", "product-design-skill.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Product Design Starter Skill", "## Source Grounding", "## Method", "## Boundaries"} {
		if !strings.Contains(string(skill), expected) {
			t.Fatalf("skill missing %q:\n%s", expected, skill)
		}
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-generation.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill generation audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Skill Generation Audit", draftRel, "skills/product-design-skill.md", "only after user acceptance"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("skill audit missing %q:\n%s", expected, audit)
		}
	}
	items, err := loadSkillFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "grounded" {
		t.Fatalf("expected accepted starter skill to be grounded, got %#v", items)
	}
}

func TestDiscardStarterSkillDraftDoesNotWriteSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenSkills,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
	}
	got, _ := m.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenSkills {
		t.Fatalf("expected discard to return to skills, got %v: %s", got.screen, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected draft removal after discard, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "product-design-skill.md")); !os.IsNotExist(err) {
		t.Fatalf("discard should not write skill, stat err=%v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-generation.md")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write skill audit, got %#v err=%v", matches, err)
	}
}

func TestSkillGroundingDraftOpensReviewWithoutMutatingSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Loose\n\n## Method\n\n- Give practical advice.\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "loose.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	kind := "book"
	sourcePath := "books/design.md"
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Product Design",
			Sources: []tape.Source{{
				Type: "local_file",
				Path: &sourcePath,
				Kind: &kind,
			}},
		},
		skillTable: newSkillTable(100, 8),
		preview:    viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()

	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))

	if got.screen != screenSkillReview {
		t.Fatalf("expected skill grounding review, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != filepath.Join(skillDraftDir, "loose-grounding-draft.md") {
		t.Fatalf("expected loose grounding draft, got %q", got.previewRel)
	}
	view := got.viewSkillReview()
	for _, expected := range []string{"Review Skill Grounding", "Source Grounding", "Actions", "Update skill", "Open draft", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("skill grounding review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter updates the skill.") {
		t.Fatalf("skill grounding review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
	draft, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Loose", "## Method", "## Source Grounding", "books/design.md", "## Boundaries", "## Maintenance"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("grounding draft missing %q:\n%s", expected, draft)
		}
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "loose.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != body {
		t.Fatalf("grounding draft should not mutate skill:\n%s", active)
	}
}

func TestRenderSkillGroundingDraftDoesNotDuplicateExistingSections(t *testing.T) {
	body := "# Grounded\n\n## Source Grounding\n\n- Use `MIXTAPE.md`.\n\n## Boundaries\n\n- Do not invent support.\n\n## Maintenance\n\n- Re-run audits.\n"
	draft := renderSkillGroundingDraft(skillFile{Name: "grounded"}, body, tape.Tape{})

	for _, heading := range []string{"## Source Grounding", "## Boundaries", "## Maintenance"} {
		if count := strings.Count(draft, heading); count != 1 {
			t.Fatalf("expected one %q heading, got %d:\n%s", heading, count, draft)
		}
	}
}

func TestAcceptSkillGroundingDraftUpdatesSkillWithBackupAndAudit(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Loose\n\n## Method\n\n- Give practical advice.\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "loose.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || got.previewRel != filepath.Join("skills", "loose.md") {
		t.Fatalf("expected skill preview after grounding accept, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected grounding draft removal after accept, stat err=%v", err)
	}
	updated, err := os.ReadFile(filepath.Join(project, "skills", "loose.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Loose", "## Method", "## Source Grounding", "## Boundaries", "## Maintenance"} {
		if !strings.Contains(string(updated), expected) {
			t.Fatalf("grounded skill missing %q:\n%s", expected, updated)
		}
	}
	backups, err := filepath.Glob(filepath.Join(project, "working", "skills", "*-loose-backup.md"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one loose skill backup, got %#v err=%v", backups, err)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != body {
		t.Fatalf("backup should preserve original skill:\n%s", backup)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-grounding.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill grounding audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Skill Grounding Audit", draftRel, "skills/loose.md", "Previous skill backup", "only after user acceptance"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("skill grounding audit missing %q:\n%s", expected, audit)
		}
	}
	items, err := loadSkillFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "grounded" {
		t.Fatalf("expected grounded skill status, got %#v", items)
	}
}

func TestDiscardSkillGroundingDraftDoesNotWriteSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Loose\n\n## Method\n\n- Give practical advice.\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "loose.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenSkills {
		t.Fatalf("expected discard to return to skills, got %v: %s", got.screen, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected grounding draft removal after discard, stat err=%v", err)
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "loose.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != body {
		t.Fatalf("discard should not write skill:\n%s", active)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-grounding.md")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write skill grounding audit, got %#v err=%v", matches, err)
	}
}

func TestDeprecateSkillDraftOpensReviewWithoutMovingSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()

	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenSkillReview {
		t.Fatalf("expected skill deprecation review, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != filepath.Join(skillDraftDir, "critique-deprecation-draft.md") {
		t.Fatalf("expected critique deprecation draft, got %q", got.previewRel)
	}
	view := got.viewSkillReview()
	for _, expected := range []string{"Review Skill Deprecation", "Move this skill", "Actions", "Archive skill", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("skill deprecation review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter archives the skill.") {
		t.Fatalf("skill deprecation review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
	draft, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Deprecate critique", "Move `skills/critique.md`", "skills/deprecated/critique.md", "does not move or edit"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("deprecation draft missing %q:\n%s", expected, draft)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "critique.md")); err != nil {
		t.Fatalf("active skill should remain before acceptance, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "deprecated", "critique.md")); !os.IsNotExist(err) {
		t.Fatalf("deprecated skill should not exist before acceptance, stat err=%v", err)
	}
}

func TestDisableSkillDraftOpensReviewWithoutMutatingSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()

	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))

	if got.screen != screenSkillReview {
		t.Fatalf("expected skill state review, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != filepath.Join(skillDraftDir, "critique-disable-draft.md") {
		t.Fatalf("expected critique disable draft, got %q", got.previewRel)
	}
	view := got.viewSkillReview()
	for _, expected := range []string{"Review Skill State", "Actions", "Update skill state", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("skill state review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter updates the skill state.") {
		t.Fatalf("skill state review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
	draft, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Disable critique", "managed disabled marker", "does not edit the skill until accepted"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("disable draft missing %q:\n%s", expected, draft)
		}
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "critique.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != body {
		t.Fatalf("disable draft should not mutate skill:\n%s", active)
	}
}

func TestAcceptSkillDisableDraftMarksSkillDisabledAndWritesAudit(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || got.previewRel != filepath.Join("skills", "critique.md") {
		t.Fatalf("expected disabled skill preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected disable draft removal after accept, stat err=%v", err)
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "critique.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{skillDisabledStartMarker, "Status: disabled", draftRel, "# Critique"} {
		if !strings.Contains(string(active), expected) {
			t.Fatalf("disabled skill missing %q:\n%s", expected, active)
		}
	}
	items, err := loadSkillFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "disabled" {
		t.Fatalf("expected disabled table status, got %#v", items)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-state.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill state audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Skill State Audit", draftRel, "Action: `disable`", "managed disabled marker was inserted"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("skill state audit missing %q:\n%s", expected, audit)
		}
	}
}

func TestAcceptSkillEnableDraftRemovesDisabledMarkerAndWritesAudit(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := applySkillDisabledBlock("# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n\n## Boundaries\n\nDo not use outside the corpus.\n", filepath.Join(skillDraftDir, "critique-disable-draft.md"))
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	if got.skillItems[0].Status != "disabled" {
		t.Fatalf("expected disabled skill before enable, got %#v", got.skillItems[0])
	}
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	draftRel := got.previewRel
	if draftRel != filepath.Join(skillDraftDir, "critique-enable-draft.md") {
		t.Fatalf("expected enable draft, got %q", draftRel)
	}

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || got.previewRel != filepath.Join("skills", "critique.md") {
		t.Fatalf("expected enabled skill preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "critique.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(active), skillDisabledStartMarker) || !strings.Contains(string(active), "# Critique") {
		t.Fatalf("enable should remove only disabled marker:\n%s", active)
	}
	items, err := loadSkillFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "grounded" {
		t.Fatalf("expected re-enabled grounded status, got %#v", items)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-state.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill state audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Skill State Audit", draftRel, "Action: `enable`", "managed disabled marker was removed"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("skill enable audit missing %q:\n%s", expected, audit)
		}
	}
}

func TestDiscardSkillDisableDraftDoesNotMutateSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenSkills {
		t.Fatalf("expected discard to return to skills, got %v: %s", got.screen, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected state draft removal after discard, stat err=%v", err)
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "critique.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != body {
		t.Fatalf("discard should not mutate skill:\n%s", active)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-state.md")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write skill state audit, got %#v err=%v", matches, err)
	}
}

func TestAcceptSkillDeprecationDraftMovesSkillAndWritesAudit(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "skill-deprecation") {
		t.Fatalf("expected skill deprecation audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "critique.md")); !os.IsNotExist(err) {
		t.Fatalf("active skill should be moved after acceptance, stat err=%v", err)
	}
	archived, err := os.ReadFile(filepath.Join(project, "skills", "deprecated", "critique.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(archived) != body {
		t.Fatalf("deprecated skill body changed:\n%s", archived)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected deprecation draft removal after accept, stat err=%v", err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-deprecation.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill deprecation audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Skill Deprecation Audit", draftRel, "skills/critique.md", "skills/deprecated/critique.md", "only after user acceptance"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("skill deprecation audit missing %q:\n%s", expected, audit)
		}
	}
	items, err := loadSkillFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("deprecated skill should leave active skill table, got %#v", items)
	}
}

func TestDiscardSkillDeprecationDraftDoesNotMoveSkill(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n"
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		skillTable:  newSkillTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startSkills()
	got, _ = got.handleSkillsKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	draftRel := got.previewRel

	got, _ = got.handleSkillReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenSkills {
		t.Fatalf("expected discard to return to skills, got %v: %s", got.screen, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("expected deprecation draft removal after discard, stat err=%v", err)
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "critique.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != body {
		t.Fatalf("discard should not rewrite active skill:\n%s", active)
	}
	if _, err := os.Stat(filepath.Join(project, "skills", "deprecated", "critique.md")); !os.IsNotExist(err) {
		t.Fatalf("discard should not archive skill, stat err=%v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-deprecation.md")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write deprecation audit, got %#v err=%v", matches, err)
	}
}

func TestAuditsScreenDiscoversAndPreviewsAuditReports(t *testing.T) {
	project := t.TempDir()
	auditDir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(auditDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contradictions := "# Contradictions\n\n## Findings\n\n| Severity | Location | Evidence | Reason |\n| --- | --- | --- | --- |\n| high | `MIXTAPE.md:3` | Rule conflicts with skill \\| exception | Names a conflict to resolve. |\n"
	if err := os.WriteFile(filepath.Join(auditDir, "2026-06-14-contradictions.md"), []byte(contradictions), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "2026-06-14-skill-corpus-alignment.md"), []byte("# Skill Corpus Alignment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "2026-06-14-source-note-quality.md"), []byte("# Source-Note Quality Audit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "ignore.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.startAudits()

	if got.screen != screenAudits {
		t.Fatalf("expected audits screen, got %v: %s", got.screen, got.err)
	}
	if len(got.auditItems) != 3 {
		t.Fatalf("expected three markdown audits, got %#v", got.auditItems)
	}
	types := []string{got.auditItems[0].Type, got.auditItems[1].Type, got.auditItems[2].Type}
	if !containsString(types, "contradiction") || !containsString(types, "skill alignment") || !containsString(types, "source notes") {
		t.Fatalf("expected classified audit types, got %#v", got.auditItems)
	}
	view := got.viewAudits()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Audits", "contradiction", "skill alignment", "source notes", "2026-06-14", "Selected", "Field", "Value", "Path", "Findings", "MIXTAPE.md:3", "Names a conflict", "Actions", "enter / o", "Draft contradiction cleanup", "Re-run contradiction audit", "reviewed apply"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("audits view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"r / s", "f / c", "Check source-note quality"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("contradiction-selected audit view should be contextual and not show %q:\n%s", unexpected, view)
		}
	}
	if strings.Contains(view, "r contradictions. f fix draft") {
		t.Fatalf("audits view should not render the old long command sentence:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "r") {
		t.Fatal("audits help should expose the contradiction audit key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "f") {
		t.Fatal("audits help should expose the contradiction cleanup key")
	}
	indexByType := map[string]int{}
	for index, item := range got.auditItems {
		indexByType[item.Type] = index
	}
	got.auditTable.SetCursor(indexByType["source notes"])
	sourceNotesView := got.viewAudits()
	for _, expected := range []string{"Draft source-note cleanup", "Re-check source-note quality"} {
		if !strings.Contains(sourceNotesView, expected) {
			t.Fatalf("source-notes audit view missing %q:\n%s", expected, sourceNotesView)
		}
	}
	if strings.Contains(sourceNotesView, "Draft contradiction cleanup") {
		t.Fatalf("source-notes audit view should not show contradiction cleanup:\n%s", sourceNotesView)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "c") {
		t.Fatal("source-notes audit help should expose the source-note cleanup key")
	}
	got.auditTable.SetCursor(indexByType["skill alignment"])
	skillAuditView := got.viewAudits()
	if !strings.Contains(skillAuditView, "Re-run skill-corpus audit") {
		t.Fatalf("skill-alignment audit view should show skill rerun:\n%s", skillAuditView)
	}
	if strings.Contains(skillAuditView, "Draft source-note cleanup") || strings.Contains(skillAuditView, "Draft contradiction cleanup") {
		t.Fatalf("skill-alignment audit view should not show cleanup actions:\n%s", skillAuditView)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "s") {
		t.Fatal("skill-alignment audit help should expose the skill audit key")
	}
	got.auditTable.SetCursor(indexByType["contradiction"])

	got, _ = got.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen != screenPreview || !strings.HasPrefix(got.previewRel, filepath.Join("working", "audits")) {
		t.Fatalf("expected enter to preview selected audit, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if action := got.nextAction(); !strings.Contains(action, "working/audits") {
		t.Fatalf("expected audit preview next action, got %q", action)
	}
}

func TestParseAuditFindingPreviewsReadsMarkdownTables(t *testing.T) {
	markdown := `# Source-Note Quality Audit

## Findings

| Source | Status | Evidence | Recommendation |
| --- | --- | --- | --- |
| https://example.com/a | missing note | note chars: 0 | Add a note with use \| boundary. |
| https://example.com/b | strong | note chars: 120 | Keep it. |
`

	findings := parseAuditFindingPreviews(markdown)

	if len(findings) != 2 {
		t.Fatalf("expected two findings, got %#v", findings)
	}
	if findings[0].Subject != "https://example.com/a" {
		t.Fatalf("unexpected subject: %#v", findings[0])
	}
	if findings[0].Status != "missing note" {
		t.Fatalf("unexpected status: %#v", findings[0])
	}
	if findings[0].Recommendation != "Add a note with use | boundary." {
		t.Fatalf("expected escaped pipe to be preserved, got %#v", findings[0])
	}
	if findings[1].Evidence != "note chars: 120" {
		t.Fatalf("unexpected second evidence: %#v", findings[1])
	}
}

func TestProjectHidesAuditsFromV1DefaultSurfaces(t *testing.T) {
	project := t.TempDir()
	missing := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}
	if hasHelp(missing.helpForScreen().ShortHelp(), "u") {
		t.Fatal("project without audits should not show u in short help")
	}
	if hasCommandTitle(missing.commandItems(), "Audits") {
		t.Fatal("project without audits should not show Audits command")
	}

	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	auditable := missing
	if hasHelp(auditable.helpForScreen().ShortHelp(), "u") {
		t.Fatal("compiled project should not show audits shortcut before reports exist")
	}
	if hasCommandTitle(auditable.commandItems(), "Audits") {
		t.Fatal("compiled project should not show Audits command before reports exist")
	}
	got, _ := auditable.handleKey(tea.KeyPressMsg(tea.Key{Code: 'u', Text: "u"}))
	if got.screen != screenProject {
		t.Fatalf("expected u to stay on Project in v1, got %v: %s", got.screen, got.err)
	}

	if err := os.MkdirAll(filepath.Join(project, "working", "audits"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "working", "audits", "contradictions.md"), []byte("# Contradictions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	available := missing
	if hasHelp(available.helpForScreen().ShortHelp(), "u") {
		t.Fatal("project with parked audits should not show u in v1 short help")
	}
	if hasCommandTitle(available.commandItems(), "Audits") {
		t.Fatal("project with parked audits should not show Audits command in v1")
	}

	got, _ = available.handleKey(tea.KeyPressMsg(tea.Key{Code: 'u', Text: "u"}))
	if got.screen != screenProject {
		t.Fatalf("expected u to keep parked audits out of v1 Project routes, got %v: %s", got.screen, got.err)
	}
}

func TestRunContradictionAuditWritesReport(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"MIXTAPE.md":          "# Mixtape\n\nAlways cite the corpus before giving a recommendation.\n",
		"LINER.md":            "# Operating Layer\n\nDo not answer outside the source boundary unless the user adds evidence.\n",
		"synthesis.md":        "# Synthesis\n\nHowever, lightweight examples can outweigh formal rules for early sketches.\n",
		"skills/critique.md":  "# Critique\n\nThis skill has a tension with the operating layer when examples are stale.\n",
		"skills/ignore.txt":   "never scanned",
		"working/ignore.md":   "never scanned",
		"local-sources/a.md":  "never scanned",
		"sources/source-1.md": "never scanned",
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))

	if got.screen != screenAudits {
		t.Fatalf("expected to stay in audits after running report, got %v: %s", got.screen, got.err)
	}
	if len(got.auditItems) != 1 || got.auditItems[0].Type != "contradiction" {
		t.Fatalf("expected one contradiction audit, got %#v", got.auditItems)
	}
	if got.auditTable.Cursor() != 0 {
		t.Fatalf("expected new audit selected, got cursor %d", got.auditTable.Cursor())
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-contradiction-audit.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one contradiction audit file, got %#v err=%v", audits, err)
	}
	body, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{"# Contradiction Audit", "MIXTAPE.md", "LINER.md", "skills/critique.md", "Always cite", "Do not answer", "No files were changed"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("audit report missing %q:\n%s", expected, report)
		}
	}
	if strings.Contains(report, "ignore.txt") || strings.Contains(report, "local-sources") {
		t.Fatalf("audit should only scan operating/corpus-facing inputs:\n%s", report)
	}
}

func TestCreateContradictionCleanupDraftOpensReviewWithoutWritingLiner(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"MIXTAPE.md":         "# Mixtape\n\nAlways cite the corpus before giving a recommendation.\n",
		"LINER.md":           "# Operating Layer\n\nExisting operating rules.\n",
		"skills/critique.md": "# Critique\n\nThis skill has a conflict with stale examples.\n",
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		auditTable:  newAuditTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'f', Text: "f"}))

	if got.screen != screenContradictionCleanupReview {
		t.Fatalf("expected contradiction cleanup review, got %v: %s", got.screen, got.err)
	}
	if !strings.Contains(got.previewRel, "contradiction-cleanup-draft") {
		t.Fatalf("expected contradiction cleanup draft preview, got rel=%q", got.previewRel)
	}
	view := got.viewContradictionCleanupReview()
	for _, expected := range []string{"Review Contradiction Cleanup", "Actions", "Apply to LINER.md", "Open draft", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("contradiction cleanup review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter applies to LINER.md.") {
		t.Fatalf("contradiction cleanup review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
	drafts, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-contradiction-cleanup-draft.md"))
	if err != nil || len(drafts) != 1 {
		t.Fatalf("expected one contradiction cleanup draft, got %#v err=%v", drafts, err)
	}
	draft, err := os.ReadFile(drafts[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Contradiction Cleanup Draft", "Proposed LINER.md Decisions", "MIXTAPE.md:3", "skills/critique.md:3", "Accepting applies this draft"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("contradiction cleanup draft missing %q:\n%s", expected, draft)
		}
	}
	liner, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(liner), contradictionCleanupStartMarker) {
		t.Fatalf("draft should not write LINER.md:\n%s", liner)
	}
}

func TestAcceptContradictionCleanupDraftAppliesManagedLinerSection(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"MIXTAPE.md":         "# Mixtape\n\nAlways cite the corpus before giving a recommendation.\n",
		"LINER.md":           "# Operating Layer\n\nExisting operating rules.\n",
		"skills/critique.md": "# Critique\n\nThis skill has a conflict with stale examples.\n",
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		auditTable:  newAuditTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'f', Text: "f"}))
	draftRel := got.previewRel

	got, _ = got.handleContradictionCleanupReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "contradiction-cleanup-apply") {
		t.Fatalf("expected apply audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	liner, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Existing operating rules", contradictionCleanupStartMarker, "# Contradiction Cleanup Draft", "Treat this hard rule as the default", contradictionCleanupEndMarker} {
		if !strings.Contains(string(liner), expected) {
			t.Fatalf("LINER.md missing %q after apply:\n%s", expected, liner)
		}
	}
	backups, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-contradiction-cleanup-LINER-backup.md"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one LINER backup, got %#v err=%v", backups, err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-contradiction-cleanup-apply.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one contradiction cleanup apply audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Contradiction Cleanup Apply Audit", draftRel, "LINER.md` was updated only after", "No source files, skill files"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("contradiction cleanup apply audit missing %q:\n%s", expected, audit)
		}
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); err != nil {
		t.Fatalf("reviewed contradiction cleanup draft should remain for provenance, stat err=%v", err)
	}
}

func TestDiscardContradictionCleanupDraftDoesNotWriteLiner(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"MIXTAPE.md": "# Mixtape\n\nAlways cite the corpus before giving a recommendation.\n",
		"LINER.md":   "# Operating Layer\n\nExisting operating rules.\n",
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		auditTable:  newAuditTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'f', Text: "f"}))
	draftRel := got.previewRel

	got, _ = got.handleContradictionCleanupReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenAudits {
		t.Fatalf("expected discard to return to audits, got %v: %s", got.screen, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("discard should remove contradiction cleanup draft, stat err=%v", err)
	}
	liner, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(liner), contradictionCleanupStartMarker) {
		t.Fatalf("discard should not write LINER.md:\n%s", liner)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-contradiction-cleanup-apply.md")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write apply audit, got %#v err=%v", matches, err)
	}
}

func TestRunSkillCorpusAuditWritesReport(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"MIXTAPE.md":                           "# Product Design\n\nUse hierarchy, interaction, and critique principles from the corpus.\n",
		"LINER.md":                             "# Operating Layer\n\nUse skills only inside their stated boundaries.\n",
		filepath.Join("skills", "critique.md"): "# Critique\n\n## Source Grounding\n\nMIXTAPE.md hierarchy and interaction sources.\n\n## Boundaries\n\nDo not critique outside the corpus.\n",
		filepath.Join("skills", "loose.md"):    "# Loose\n\nUse taste for anything.\n",
		filepath.Join("skills", "paused.md"):   applySkillDisabledBlock("# Paused\n\n## Source Grounding\n\nMIXTAPE.md.\n\n## Boundaries\n\nDo not use while paused.\n", filepath.Join(skillDraftDir, "paused-disable-draft.md")),
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))

	if got.screen != screenAudits {
		t.Fatalf("expected to stay in audits after running skill audit, got %v: %s", got.screen, got.err)
	}
	if len(got.auditItems) != 1 || got.auditItems[0].Type != "skill alignment" {
		t.Fatalf("expected one skill alignment audit, got %#v", got.auditItems)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-skill-corpus-alignment.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one skill alignment audit file, got %#v err=%v", audits, err)
	}
	body, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{"# Skill-Corpus Alignment Audit", "skills/critique.md", "aligned", "skills/loose.md", "needs grounding", "skills/paused.md", "disabled", "Managed disabled marker found", "No files were changed"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("skill alignment report missing %q:\n%s", expected, report)
		}
	}
}

func TestAuditSkillAlignmentCanDraftFirstSkillRepair(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"MIXTAPE.md":                           "# Product Design\n\nUse hierarchy and critique sources.\n",
		filepath.Join("skills", "grounded.md"): "# Grounded\n\n## Source Grounding\n\nMIXTAPE.md hierarchy and critique sources.\n\n## Boundaries\n\nDo not invent support outside product design.\n",
		filepath.Join("skills", "loose.md"):    "# Loose\n\nUse taste for anything.\n",
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		auditTable:  newAuditTable(100, 8),
	}
	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))
	view := got.viewAudits()
	for _, expected := range []string{"Draft first skill repair", "Re-run skill-corpus audit"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("skill alignment actions missing %q:\n%s", expected, view)
		}
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "g") {
		t.Fatal("skill alignment audit help should expose repair key")
	}

	got, _ = got.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))

	if got.screen != screenSkillReview {
		t.Fatalf("expected skill review screen, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != filepath.Join(skillDraftDir, "loose-grounding-draft.md") {
		t.Fatalf("expected loose grounding draft, got %q", got.previewRel)
	}
	draft, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Loose", "## Source Grounding", "## Boundaries", "## Maintenance"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("skill repair draft missing %q:\n%s", expected, draft)
		}
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "loose.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(active), "## Source Grounding") {
		t.Fatalf("audit repair draft should not mutate active skill before review:\n%s", active)
	}
}

func TestAuditAgentCleanupPacketFromSelectedReport(t *testing.T) {
	project := t.TempDir()
	files := map[string]string{
		"MIXTAPE.md":                        "# Product Design\n\nUse hierarchy and critique sources.\n",
		filepath.Join("skills", "loose.md"): "# Loose\n\nUse taste for anything.\n",
	}
	for rel, body := range files {
		path := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		auditTable:  newAuditTable(100, 8),
	}
	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))
	view := got.viewAudits()
	if !strings.Contains(view, "Create agent cleanup packet") {
		t.Fatalf("selected audit should show cleanup packet action:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "p") {
		t.Fatal("selected audit help should expose cleanup packet key")
	}

	got, _ = got.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "agent-cleanup-packet") {
		t.Fatalf("expected cleanup packet preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	got.screen = screenAudits
	packetView := got.viewAudits()
	for _, expected := range []string{"Source audit", "Source type", "skill alignment"} {
		if !strings.Contains(packetView, expected) {
			t.Fatalf("cleanup packet selected detail missing %q:\n%s", expected, packetView)
		}
	}
	got.screen = screenPreview
	packet, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Audit Agent Cleanup Packet",
		"Source audit:",
		"Audit type: skill alignment",
		"working/skills/<skill>-grounding-draft.md",
		"skills/loose.md",
		"Do not update production files directly",
		"No agent was run by the Go TUI",
		"No cleanup changes were applied by this packet",
	} {
		if !strings.Contains(string(packet), expected) {
			t.Fatalf("cleanup packet missing %q:\n%s", expected, packet)
		}
	}
	active, err := os.ReadFile(filepath.Join(project, "skills", "loose.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(active), "## Source Grounding") {
		t.Fatalf("cleanup packet should not mutate active skill:\n%s", active)
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	foundPacket := false
	for _, item := range items {
		if item.Type == "cleanup packet" {
			foundPacket = true
			break
		}
	}
	if !foundPacket {
		t.Fatalf("expected cleanup packet to be listed as cleanup packet, got %#v", items)
	}
}

func TestRunSkillCorpusAuditRequiresSkills(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		currentPath: project,
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))

	if got.screen != screenAudits {
		t.Fatalf("expected to remain on audits, got %v", got.screen)
	}
	if !strings.Contains(got.err, "No skills found") {
		t.Fatalf("expected missing-skill error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "audits")); !os.IsNotExist(err) {
		t.Fatalf("skill audit should not create audit dir on missing skills, stat err=%v", err)
	}
}

func TestRunSourceNoteAuditWritesReport(t *testing.T) {
	project := t.TempDir()
	strongNote := "Read for concrete interaction hierarchy guidance; scope to product UI decisions, not brand strategy."
	strongKind := "principle"
	strongSection := "interaction"
	thinNote := "Good article."
	if err := tape.WriteProject(project, tape.Tape{
		Title: "Product Design",
		Sources: []tape.Source{
			{
				Type:    "web",
				URL:     "https://example.com/strong",
				Note:    &strongNote,
				Kind:    &strongKind,
				Section: &strongSection,
			},
			{
				Type: "web",
				URL:  "https://example.com/missing-note",
			},
			{
				Type: "web",
				URL:  "https://example.com/thin",
				Note: &thinNote,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))

	if got.screen != screenAudits {
		t.Fatalf("expected to stay in audits after running source-note audit, got %v: %s", got.screen, got.err)
	}
	if len(got.auditItems) != 1 || got.auditItems[0].Type != "source notes" {
		t.Fatalf("expected one source-note audit, got %#v", got.auditItems)
	}
	if got.auditTable.Cursor() != 0 {
		t.Fatalf("expected new audit selected, got cursor %d", got.auditTable.Cursor())
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-source-note-quality.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one source-note audit file, got %#v err=%v", audits, err)
	}
	body, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{
		"# Source-Note Quality Audit",
		"https://example.com/strong",
		"strong",
		"https://example.com/missing-note",
		"missing note",
		"https://example.com/thin",
		"thin note",
		"No files were changed",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("source-note report missing %q:\n%s", expected, report)
		}
	}
}

func TestCreateSourceNoteCleanupDraftWritesReviewArtifact(t *testing.T) {
	project := t.TempDir()
	strongNote := "Read for concrete interaction hierarchy guidance; scope to product UI decisions, not brand strategy."
	strongKind := "principle"
	strongSection := "interaction"
	thinNote := "Good article."
	if err := tape.WriteProject(project, tape.Tape{
		Title: "Product Design",
		Sources: []tape.Source{
			{
				Type:    "web",
				URL:     "https://example.com/strong",
				Note:    &strongNote,
				Kind:    &strongKind,
				Section: &strongSection,
			},
			{
				Type: "web",
				URL:  "https://example.com/missing-note",
			},
			{
				Type: "web",
				URL:  "https://example.com/thin",
				Note: &thinNote,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.screen != screenSourceNoteCleanupReview {
		t.Fatalf("expected cleanup draft to open review, got %v: %s", got.screen, got.err)
	}
	if !strings.Contains(got.previewRel, "source-note-cleanup-draft") {
		t.Fatalf("expected source-note cleanup preview, got rel=%q", got.previewRel)
	}
	view := got.viewSourceNoteCleanupReview()
	for _, expected := range []string{"Review Source-Note Cleanup", "Actions", "Apply to tape.yaml", "Open draft", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("cleanup review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter applies to tape.yaml.") {
		t.Fatalf("source-note cleanup review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-source-note-cleanup-draft.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one source-note cleanup draft file, got %#v err=%v", audits, err)
	}
	body, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{
		"# Source-Note Cleanup Draft",
		"https://example.com/missing-note",
		"Add a curator note",
		"https://example.com/thin",
		"Expand the existing note",
		"Use `https://example.com/thin` as a web source",
		"No files were changed except this cleanup draft",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("source-note cleanup draft missing %q:\n%s", expected, report)
		}
	}
	if strings.Contains(report, "https://example.com/strong") {
		t.Fatalf("cleanup draft should skip strong source notes:\n%s", report)
	}
	accepted, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Sources[1].Note != nil {
		t.Fatalf("cleanup draft should not mutate missing note, got %#v", *accepted.Sources[1].Note)
	}
	if accepted.Sources[2].Note == nil || *accepted.Sources[2].Note != thinNote {
		t.Fatalf("cleanup draft should not mutate thin note, got %#v", accepted.Sources[2].Note)
	}
}

func TestAcceptSourceNoteCleanupDraftUpdatesTapeWithBackupAndAudit(t *testing.T) {
	project := t.TempDir()
	strongNote := "Read for concrete interaction hierarchy guidance; scope to product UI decisions, not brand strategy."
	thinNote := "Good article."
	if err := tape.WriteProject(project, tape.Tape{
		Title: "Product Design",
		Sources: []tape.Source{
			{
				Type: "web",
				URL:  "https://example.com/strong",
				Note: &strongNote,
			},
			{
				Type: "web",
				URL:  "https://example.com/missing-note",
			},
			{
				Type: "web",
				URL:  "https://example.com/thin",
				Note: &thinNote,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		auditTable:  newAuditTable(100, 8),
	}
	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))
	draftRel := got.previewRel

	got, _ = got.handleSourceNoteCleanupReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "source-note-cleanup-apply") {
		t.Fatalf("expected apply audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	accepted, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Sources[0].Note == nil || *accepted.Sources[0].Note != strongNote {
		t.Fatalf("strong note should remain unchanged, got %#v", accepted.Sources[0].Note)
	}
	for index, url := range []string{"https://example.com/missing-note", "https://example.com/thin"} {
		sourceIndex := index + 1
		if accepted.Sources[sourceIndex].Note == nil {
			t.Fatalf("expected source %s to receive a note", url)
		}
		note := *accepted.Sources[sourceIndex].Note
		if !strings.Contains(note, "Use "+url+" as a web source") || len(sourceNoteIssues(note)) != 0 {
			t.Fatalf("applied note for %s is not strong enough:\n%s\nissues=%#v", url, note, sourceNoteIssues(note))
		}
	}
	backups, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-source-note-cleanup-tape-backup.yaml"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one tape backup, got %#v err=%v", backups, err)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(backup), "Good article.") {
		t.Fatalf("backup should preserve previous tape notes:\n%s", backup)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-source-note-cleanup-apply.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one source-note apply audit, got %#v err=%v", audits, err)
	}
	body, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	report := string(body)
	for _, expected := range []string{
		"# Source-Note Cleanup Apply Audit",
		draftRel,
		"source-note-cleanup-tape-backup.yaml",
		"https://example.com/missing-note",
		"https://example.com/thin",
		"`tape.yaml` was updated only after the cleanup draft review step",
		"No source files were changed",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("source-note apply audit missing %q:\n%s", expected, report)
		}
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); err != nil {
		t.Fatalf("reviewed cleanup draft should remain for provenance, stat err=%v", err)
	}
}

func TestDiscardSourceNoteCleanupDraftRemovesDraftWithoutWritingTape(t *testing.T) {
	project := t.TempDir()
	thinNote := "Good article."
	if err := tape.WriteProject(project, tape.Tape{
		Title: "Product Design",
		Sources: []tape.Source{
			{
				Type: "web",
				URL:  "https://example.com/missing-note",
			},
			{
				Type: "web",
				URL:  "https://example.com/thin",
				Note: &thinNote,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
		auditTable:  newAuditTable(100, 8),
	}
	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))
	draftRel := got.previewRel

	got, _ = got.handleSourceNoteCleanupReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))

	if got.screen != screenAudits {
		t.Fatalf("expected discard to return to audits, got %v: %s", got.screen, got.err)
	}
	if _, err := os.Stat(filepath.Join(project, draftRel)); !os.IsNotExist(err) {
		t.Fatalf("discard should remove cleanup draft, stat err=%v", err)
	}
	accepted, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Sources[0].Note != nil {
		t.Fatalf("discard should not write missing note, got %#v", *accepted.Sources[0].Note)
	}
	if accepted.Sources[1].Note == nil || *accepted.Sources[1].Note != thinNote {
		t.Fatalf("discard should not rewrite thin note, got %#v", accepted.Sources[1].Note)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-source-note-cleanup-apply.md")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write apply audit, got %#v err=%v", matches, err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-source-note-cleanup-tape-backup.yaml")); err != nil || len(matches) != 0 {
		t.Fatalf("discard should not write tape backup, got %#v err=%v", matches, err)
	}
}

func TestRunSourceNoteAuditRequiresAcceptedSources(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Empty"}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		currentPath: project,
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))

	if got.screen != screenAudits {
		t.Fatalf("expected to remain on audits, got %v", got.screen)
	}
	if !strings.Contains(got.err, "No saved sources") {
		t.Fatalf("expected missing-source error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "audits")); !os.IsNotExist(err) {
		t.Fatalf("source-note audit should not create audit dir on missing sources, stat err=%v", err)
	}
}

func TestSourceNoteCleanupDraftRequiresAcceptedSources(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Empty"}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenAudits,
		width:       100,
		currentPath: project,
		auditTable:  newAuditTable(100, 8),
	}

	got, _ := m.handleAuditsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.screen != screenAudits {
		t.Fatalf("expected to remain on audits, got %v", got.screen)
	}
	if !strings.Contains(got.err, "No saved sources") {
		t.Fatalf("expected missing-source error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "audits")); !os.IsNotExist(err) {
		t.Fatalf("cleanup draft should not create audit dir on missing sources, stat err=%v", err)
	}
}

func TestEvalsScreenDiscoversAndPreviewsEvalArtifacts(t *testing.T) {
	project := t.TempDir()
	paths := map[string]string{
		filepath.Join("working", "evals", "tasksets", "terminal-flow.md"):          "# Taskset\n",
		filepath.Join("working", "evals", "runs", "2026-06-14", "with-liner.md"):   "# Run\n",
		filepath.Join("working", "evals", "summaries", "terminal-flow-summary.md"): "# Summary\n",
		filepath.Join("working", "evals", "comparisons", "terminal-flow.md"):       "# Comparison\n",
		filepath.Join("working", "evals", "judges", "terminal-flow.md"):            "# Judge\n",
		filepath.Join("working", "evals", "misc", "note.json"):                     `{"note":true}`,
	}
	for rel, body := range paths {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
		evalTable:   newEvalTable(100, 8),
	}

	got, _ := m.startEvals()

	if got.screen != screenEvals {
		t.Fatalf("expected evals screen, got %v: %s", got.screen, got.err)
	}
	if len(got.evalItems) != 6 {
		t.Fatalf("expected six eval artifacts, got %#v", got.evalItems)
	}
	areas := []string{got.evalItems[0].Area, got.evalItems[1].Area, got.evalItems[2].Area, got.evalItems[3].Area, got.evalItems[4].Area, got.evalItems[5].Area}
	for _, expected := range []string{"taskset", "run", "summary", "comparison", "judge", "eval"} {
		if !containsString(areas, expected) {
			t.Fatalf("expected area %q in %#v", expected, got.evalItems)
		}
	}
	view := got.viewEvals()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Impact Tests", "run", "summary", "comparison", "judge", "terminal-flow", "Selected", "Field", "Value", "Path", "Actions", "enter / o", "Create runner packet", "Create judge packet"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("evals view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Create variant run packet", "Compare variant outputs"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("comparison-selected eval view should not show %q:\n%s", unexpected, view)
		}
	}
	if strings.Contains(view, "t taskset. r run packet") {
		t.Fatalf("evals view should not render the old long command sentence:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "a") {
		t.Fatal("evals comparison help should expose the automation key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "j") {
		t.Fatal("evals comparison help should expose the judge-packet key")
	}
	tasksetIndex := -1
	runIndex := -1
	for index, item := range got.evalItems {
		switch item.Area {
		case "taskset":
			tasksetIndex = index
		case "run", "summary":
			if runIndex < 0 {
				runIndex = index
			}
		}
	}
	if tasksetIndex < 0 || runIndex < 0 {
		t.Fatalf("expected taskset and run/summary items, got %#v", got.evalItems)
	}
	got.evalTable.SetCursor(tasksetIndex)
	tasksetView := got.viewEvals()
	for _, expected := range []string{"Create variant run packet", "Create another impact taskset"} {
		if !strings.Contains(tasksetView, expected) {
			t.Fatalf("taskset-selected eval view missing %q:\n%s", expected, tasksetView)
		}
	}
	if strings.Contains(tasksetView, "Create judge packet") {
		t.Fatalf("taskset-selected eval view should not show judge actions:\n%s", tasksetView)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "t") {
		t.Fatal("impact tests help should expose the taskset key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "r") {
		t.Fatal("evals taskset help should expose the run packet key")
	}
	got.evalTable.SetCursor(runIndex)
	runView := got.viewEvals()
	for _, expected := range []string{"Create runner packet", "Compare variant outputs", "Create judge packet"} {
		if !strings.Contains(runView, expected) {
			t.Fatalf("run-selected eval view missing %q:\n%s", expected, runView)
		}
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "c") {
		t.Fatal("evals run help should expose the compare key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "j") {
		t.Fatal("evals run help should expose the judge-packet key")
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen != screenPreview || !strings.HasPrefix(got.previewRel, filepath.Join("working", "evals")) {
		t.Fatalf("expected enter to preview selected eval, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if action := got.nextAction(); !strings.Contains(action, "working/evals") {
		t.Fatalf("expected eval preview next action, got %q", action)
	}
}

func TestEvalsScreenShowsRunCoverageForSelectedRun(t *testing.T) {
	project := t.TempDir()
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	paths := map[string]string{
		"MIXTAPE.md":                         "# Mix\n",
		"LINER.md":                           "# Liner\n",
		filepath.Join("skills", "flow.md"):   "# Flow\n",
		filepath.Join(runRel, "README.md"):   "# Impact Test Run Packet\n",
		filepath.Join(runRel, "baseline.md"): "# Impact Variant: Baseline\n\n## Outputs\n\n### Task 1 Output\n\nUse a simple first step.\n\n### Task 2 Output\n\nRemove the helper text.\n\n### Task 3 Output\n\n_Paste output here._\n",
		filepath.Join(runRel, "corpus.md"):   "# Impact Variant: Corpus\n",
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): "# Impact Test Summary\n\n| Task | Variant | Score | Qualitative Notes |\n| --- | --- | --- | --- |\n| Task 1 | Baseline | 3 | clearer |\n| Task 2 | Baseline | 4 | specific |\n",
	}
	for rel, body := range paths {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       112,
		height:      34,
		currentPath: project,
		currentTape: tape.Tape{Title: "Terminal Flow"},
		evalTable:   newEvalTable(112, 8),
	}

	got, _ := m.startEvals()
	for index, item := range got.evalItems {
		if item.RelPath == filepath.Join(runRel, "README.md") {
			got.evalTable.SetCursor(index)
			break
		}
	}
	view := got.viewEvals()

	for _, expected := range []string{
		"Run coverage",
		"Baseline",
		"partial 2/3",
		"2/3 scored",
		"ready: load no project files",
		"Corpus",
		"empty",
		"unscored",
		"Skills",
		"1 skill file(s)",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("run coverage view missing %q:\n%s", expected, view)
		}
	}
	assertNoBoxCorners(t, view)
}

func TestEvalsScreenOpensEmptyForCompiledProject(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}

	got, _ := m.startEvals()

	if got.screen != screenEvals {
		t.Fatalf("expected evals screen, got %v: %s", got.screen, got.err)
	}
	if len(got.evalItems) != 0 {
		t.Fatalf("expected no eval artifacts yet, got %#v", got.evalItems)
	}
	if got.note != "No impact-test artifacts yet." {
		t.Fatalf("expected empty evals state note, got %q", got.note)
	}
	view := got.viewEvals()
	for _, expected := range []string{"No impact-test artifacts found", "Actions", "Create impact taskset"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("empty evals view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Create runner packet", "Create judge packet", "Compare variant outputs"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("empty evals view should not show %q:\n%s", unexpected, view)
		}
	}
}

func TestCreateEvalTasksetWritesStarter(t *testing.T) {
	project := t.TempDir()
	jtbd := "When reviewing terminal product flows, I want grounded critique so the next interaction is simpler."
	if err := tape.WriteProject(project, tape.Tape{
		Title: "Design Engineering",
		JTBD:  &jtbd,
		Sources: []tape.Source{{
			Type: "web",
			URL:  "https://example.com/source",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Liner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "skills", "critique.md"), []byte("# Critique\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenEvals,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}

	got, _ := m.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 't', Text: "t"}))

	if got.screen != screenEvals {
		t.Fatalf("expected to stay in evals after creating taskset, got %v: %s", got.screen, got.err)
	}
	if len(got.evalItems) != 1 || got.evalItems[0].Area != "taskset" {
		t.Fatalf("expected one taskset artifact, got %#v", got.evalItems)
	}
	if got.evalTable.Cursor() != 0 {
		t.Fatalf("expected new taskset selected, got cursor %d", got.evalTable.Cursor())
	}
	tasksets, err := filepath.Glob(filepath.Join(project, "working", "evals", "tasksets", "*-design-engineering-impact-taskset.md"))
	if err != nil || len(tasksets) != 1 {
		t.Fatalf("expected one impact taskset file, got %#v err=%v", tasksets, err)
	}
	body, err := os.ReadFile(tasksets[0])
	if err != nil {
		t.Fatal(err)
	}
	content := string(body)
	for _, expected := range []string{
		"# Impact Test Taskset: Design Engineering",
		"human review",
		"fresh agent/session",
		"baseline",
		"`MIXTAPE.md` + `LINER.md` + `skills/*.md`",
		"Task 1: Real User Request",
		"Task 2: Draft Critique",
		"Task 3: Boundary Check",
		"Human Rubric",
		"Impact delta",
		"Qualitative Notes",
		jtbd,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("impact taskset missing %q:\n%s", expected, content)
		}
	}
}

func TestCreateEvalRunPacketWritesVariantsAndSummary(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	tasksetRel := filepath.Join("working", "evals", "tasksets", "terminal-flow.md")
	tasksetBody := "# Impact Test Taskset: Terminal Flow\n\n## Tasks\n\n### Task 1: Real User Request\n"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(project, tasksetRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, tasksetRel), []byte(tasksetBody), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))

	if got.screen != screenEvals {
		t.Fatalf("expected to stay in evals after creating run packet, got %v: %s", got.screen, got.err)
	}
	if len(got.evalItems) != 7 {
		t.Fatalf("expected taskset, readme, four variants, and summary, got %#v", got.evalItems)
	}
	selected, ok := got.selectedEval()
	if !ok || selected.Area != "run" || filepath.Base(selected.RelPath) != "README.md" {
		t.Fatalf("expected run README to be selected, got %#v ok=%v", selected, ok)
	}
	runDirs, err := filepath.Glob(filepath.Join(project, "working", "evals", "runs", "*-terminal-flow"))
	if err != nil || len(runDirs) != 1 {
		t.Fatalf("expected one terminal-flow run dir, got %#v err=%v", runDirs, err)
	}
	for _, name := range []string{"README.md", "baseline.md", "corpus.md", "operating-layer.md", "skills.md"} {
		if _, err := os.Stat(filepath.Join(runDirs[0], name)); err != nil {
			t.Fatalf("expected run file %s, stat err=%v", name, err)
		}
	}
	summaries, err := filepath.Glob(filepath.Join(project, "working", "evals", "summaries", "*-terminal-flow-summary.md"))
	if err != nil || len(summaries) != 1 {
		t.Fatalf("expected one terminal-flow summary, got %#v err=%v", summaries, err)
	}
	readme, err := os.ReadFile(filepath.Join(runDirs[0], "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Impact Test Run Packet", tasksetRel, "baseline.md", "operating-layer.md", "summary"} {
		if !strings.Contains(string(readme), expected) {
			t.Fatalf("run README missing %q:\n%s", expected, string(readme))
		}
	}
	baseline, err := os.ReadFile(filepath.Join(runDirs[0], "baseline.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Impact Variant: Baseline", "Context to load: no project files", "load no project files", "Taskset Snapshot", "Terminal Flow"} {
		if !strings.Contains(string(baseline), expected) {
			t.Fatalf("baseline variant missing %q:\n%s", expected, string(baseline))
		}
	}
	summary, err := os.ReadFile(summaries[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Impact Test Summary", "Score Table", "Largest useful delta over baseline", "Did `LINER.md` change behavior?", "Source, note, skill, or boundary fixes"} {
		if !strings.Contains(string(summary), expected) {
			t.Fatalf("summary missing %q:\n%s", expected, string(summary))
		}
	}
}

func TestCreateEvalAutomationPacketFromRunPacket(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	for rel, body := range map[string]string{
		"MIXTAPE.md":                           "# Mixtape\n",
		"LINER.md":                             "# Liner\n",
		filepath.Join("skills", "critique.md"): "# Critique\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tasksetRel := filepath.Join("working", "evals", "tasksets", "terminal-flow.md")
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	files := map[string]string{
		tasksetRel:                         "# Eval Taskset: Terminal Flow\n",
		filepath.Join(runRel, "README.md"): "# Eval Run Packet\n\nTaskset: `" + filepath.ToSlash(tasksetRel) + "`\n",
		filepath.Join(runRel, "baseline.md"): `# Eval Variant: Baseline

### Task 1 Output

Baseline output.
`,
		filepath.Join(runRel, "corpus.md"):          "# Eval Variant: Corpus\n",
		filepath.Join(runRel, "operating-layer.md"): "# Eval Variant: Operating Layer\n",
		filepath.Join(runRel, "skills.md"):          "# Eval Variant: Skills\n",
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): `# Eval Run Summary

| Task | Variant | Score | Qualitative Notes |
| --- | --- | --- | --- |
| Task 1 | Baseline | 2 | generic |
`,
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startEvals()
	for index, item := range got.evalItems {
		if item.RelPath == filepath.Join(runRel, "README.md") {
			got.evalTable.SetCursor(index)
			break
		}
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "automation-packet") {
		t.Fatalf("expected automation packet preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	packets, err := filepath.Glob(filepath.Join(project, "working", "evals", "automation", "*-terminal-flow-automation-packet.md"))
	if err != nil || len(packets) != 1 {
		t.Fatalf("expected one automation packet, got %#v err=%v", packets, err)
	}
	packet, err := os.ReadFile(packets[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(packet)
	for _, expected := range []string{
		"# Impact Test Runner Packet",
		"external runner",
		"Impact Tests",
		filepath.ToSlash(runRel),
		filepath.ToSlash(tasksetRel),
		"baseline.md",
		"operating-layer.md",
		"ready",
		"1 skill file(s)",
		"No model runs were executed by the Go TUI",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("automation packet missing %q:\n%s", expected, body)
		}
	}
	items, err := loadEvalFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	hasAutomation := false
	for _, item := range items {
		if item.Area == "automation" {
			hasAutomation = true
		}
	}
	if !hasAutomation {
		t.Fatalf("automation packet should be listed as automation area, got %#v", items)
	}
}

func TestEvalsAutomationPacketShowsCoverageAndCanCompare(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	automationRel := filepath.Join("working", "evals", "automation", "terminal-flow-automation-packet.md")
	files := map[string]string{
		"MIXTAPE.md":                         "# Mixtape\n",
		filepath.Join(runRel, "README.md"):   "# Impact Test Run Packet\n",
		filepath.Join(runRel, "baseline.md"): "# Impact Variant: Baseline\n\n## Outputs\n\n### Task 1 Output\n\nBaseline output.\n\n### Task 2 Output\n\n_Paste output here._\n\n### Task 3 Output\n\n_Paste output here._\n",
		filepath.Join(runRel, "corpus.md"):   "# Impact Variant: Corpus\n",
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): "# Impact Test Summary\n\n| Task | Variant | Score | Qualitative Notes |\n| --- | --- | --- | --- |\n| Task 1 | Baseline | 2 | generic |\n",
		automationRel: "# Impact Test Runner Packet\n\nRun packet: `" + filepath.ToSlash(runRel) + "`\n",
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       112,
		height:      34,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(112, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startEvals()
	for index, item := range got.evalItems {
		if item.RelPath == automationRel {
			got.evalTable.SetCursor(index)
			break
		}
	}
	view := got.viewEvals()
	for _, expected := range []string{"Run packet", filepath.ToSlash(runRel), "Summary scores", "Run coverage", "Baseline", "partial 1/3", "1/3 scored", "Compare variant outputs", "Create judge packet"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("automation-selected view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Create impact taskset") {
		t.Fatalf("automation-selected view should not show generic taskset action:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "c") || !hasHelp(got.helpForScreen().ShortHelp(), "j") {
		t.Fatalf("automation-selected help should expose compare and judge actions")
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "comparison") {
		t.Fatalf("expected comparison preview from automation packet, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	comparisons, err := filepath.Glob(filepath.Join(project, "working", "evals", "comparisons", "*-terminal-flow-comparison.md"))
	if err != nil || len(comparisons) != 1 {
		t.Fatalf("expected one comparison report, got %#v err=%v", comparisons, err)
	}
}

func TestCreateEvalComparisonReportFromRunPacket(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	files := map[string]string{
		filepath.Join(runRel, "README.md"): "# Eval Run Packet\n",
		filepath.Join(runRel, "baseline.md"): `# Eval Variant: Baseline

## Outputs

### Task 1 Output

Baseline generic answer about interface polish and testing.

### Task 2 Output

Baseline critique misses source-backed rules.

### Task 3 Output

Baseline boundary answer overclaims outside the corpus.

## Human Notes
`,
		filepath.Join(runRel, "corpus.md"): `# Eval Variant: Corpus

## Outputs

### Task 1 Output

Corpus answer names a source-backed principle.

### Task 2 Output

Corpus critique catches one unsupported claim.

### Task 3 Output

_Paste output here._

## Human Notes
`,
		filepath.Join(runRel, "operating-layer.md"): `# Eval Variant: Operating Layer

## Outputs

### Task 1 Output

Operating layer answer follows the LINER routing rule and asks for missing context.

### Task 2 Output

Operating layer critique names the unsupported claim and proposes a concrete revision.

### Task 3 Output

Operating layer boundary answer narrows the request before giving advice.

## Human Notes
`,
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): `# Eval Run Summary

| Task | Variant | Score | Qualitative Notes |
| --- | --- | --- | --- |
| Task 1 | Baseline | 2 | generic |
| Task 1 | Operating Layer | 5 | follows LINER |
`,
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()
	for index, item := range got.evalItems {
		if item.RelPath == filepath.Join(runRel, "README.md") {
			got.evalTable.SetCursor(index)
			break
		}
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "comparison") {
		t.Fatalf("expected comparison preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	comparisons, err := filepath.Glob(filepath.Join(project, "working", "evals", "comparisons", "*-terminal-flow-comparison.md"))
	if err != nil || len(comparisons) != 1 {
		t.Fatalf("expected one comparison report, got %#v err=%v", comparisons, err)
	}
	report, err := os.ReadFile(comparisons[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(report)
	for _, expected := range []string{
		"# Impact Test Comparison Report",
		"Output Coverage",
		"What impact did each loaded layer add over baseline?",
		filepath.ToSlash(runRel),
		"Baseline generic answer",
		"2",
		"generic",
		"Operating layer answer follows the LINER routing rule",
		"5",
		"follows LINER",
		"missing file",
		"No model runs were executed",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("comparison report missing %q:\n%s", expected, body)
		}
	}
	items, err := loadEvalFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	hasComparison := false
	for _, item := range items {
		if item.Area == "comparison" {
			hasComparison = true
		}
	}
	if !hasComparison {
		t.Fatalf("comparison report should be listed as comparison area, got %#v", items)
	}
}

func TestCreateEvalJudgePacketFromComparison(t *testing.T) {
	project := t.TempDir()
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	comparisonRel := filepath.Join("working", "evals", "comparisons", "terminal-flow-comparison.md")
	files := map[string]string{
		filepath.Join(runRel, "README.md"): "# Eval Run Packet\n",
		filepath.Join(runRel, "baseline.md"): `# Eval Variant: Baseline

### Task 1 Output

Baseline generic answer about interface polish.
`,
		filepath.Join(runRel, "operating-layer.md"): `# Eval Variant: Operating Layer

### Task 1 Output

Operating layer answer follows LINER boundaries.
`,
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): `# Eval Run Summary

| Task | Variant | Score | Qualitative Notes |
| --- | --- | --- | --- |
| Task 1 | Baseline | 2 | generic |
| Task 1 | Operating Layer | 5 | follows LINER |
`,
		comparisonRel: "# Eval Comparison Report\n\nRun packet: `" + filepath.ToSlash(runRel) + "`\n",
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()
	for index, item := range got.evalItems {
		if item.RelPath == comparisonRel {
			got.evalTable.SetCursor(index)
			break
		}
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "judge-packet") {
		t.Fatalf("expected judge packet preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	judges, err := filepath.Glob(filepath.Join(project, "working", "evals", "judges", "*-terminal-flow-judge-packet.md"))
	if err != nil || len(judges) != 1 {
		t.Fatalf("expected one judge packet, got %#v err=%v", judges, err)
	}
	packet, err := os.ReadFile(judges[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(packet)
	for _, expected := range []string{
		"# Impact Test Judge Packet",
		"Judge Instructions",
		"Judge Score Table",
		"Baseline generic answer",
		"Operating layer answer follows LINER boundaries",
		"2",
		"5",
		"No judge was run by the TUI",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("judge packet missing %q:\n%s", expected, body)
		}
	}
	items, err := loadEvalFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	hasJudge := false
	for _, item := range items {
		if item.Area == "judge" {
			hasJudge = true
		}
	}
	if !hasJudge {
		t.Fatalf("judge packet should be listed as judge area, got %#v", items)
	}
}

func TestCreateEvalReadinessReportFromRunPacket(t *testing.T) {
	project := t.TempDir()
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	files := map[string]string{
		"MIXTAPE.md":                       "# Mixtape\n",
		filepath.Join(runRel, "README.md"): "# Impact Test Run Packet\n",
		filepath.Join(runRel, "baseline.md"): `# Impact Variant: Baseline

## Outputs

### Task 1 Output

Baseline output.

### Task 2 Output

_Paste output here._

### Task 3 Output

_Paste output here._
`,
		filepath.Join(runRel, "corpus.md"):          "# Impact Variant: Corpus\n",
		filepath.Join(runRel, "operating-layer.md"): "# Impact Variant: Operating Layer\n",
		filepath.Join(runRel, "skills.md"):          "# Impact Variant: Skills\n",
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): `# Impact Test Summary

| Task | Variant | Score | Qualitative Notes |
| --- | --- | --- | --- |
| Task 1 | Baseline | 2 | generic |
`,
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       112,
		height:      34,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(112, 8),
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startEvals()
	for index, item := range got.evalItems {
		if item.RelPath == filepath.Join(runRel, "README.md") {
			got.evalTable.SetCursor(index)
			break
		}
	}
	view := got.viewEvals()
	if !strings.Contains(view, "Create readiness report") {
		t.Fatalf("run-selected eval view should expose readiness report action:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "v") {
		t.Fatal("run-selected eval help should expose readiness key")
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'v', Text: "v"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "readiness") {
		t.Fatalf("expected readiness report preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	reports, err := filepath.Glob(filepath.Join(project, "working", "evals", "readiness", "*-terminal-flow-readiness.md"))
	if err != nil || len(reports) != 1 {
		t.Fatalf("expected one readiness report, got %#v err=%v", reports, err)
	}
	report, err := os.ReadFile(reports[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(report)
	for _, expected := range []string{
		"# Impact Test Readiness Report",
		filepath.ToSlash(runRel),
		"Readiness Matrix",
		"Baseline",
		"partial 1/3",
		"1/3 scored",
		"Run variant and paste missing task outputs.",
		"Operating Layer context: LINER.md missing",
		"Skills context: LINER.md missing; skills/*.md missing",
		"No model runs were executed by this readiness report",
		"No variant outputs, summary scores, source files, skills, or operating-layer files were changed",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("readiness report missing %q:\n%s", expected, body)
		}
	}
	items, err := loadEvalFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	hasReadiness := false
	for _, item := range items {
		if item.Area == "readiness" {
			hasReadiness = true
		}
	}
	if !hasReadiness {
		t.Fatalf("readiness report should be listed as readiness area, got %#v", items)
	}
}

func TestCreateEvalComparisonReportFromSummary(t *testing.T) {
	project := t.TempDir()
	runID := "2026-06-14-terminal-flow"
	runRel := filepath.Join("working", "evals", "runs", runID)
	files := map[string]string{
		filepath.Join(runRel, "README.md"): "# Eval Run Packet\n",
		filepath.Join(runRel, "baseline.md"): `# Eval Variant: Baseline

### Task 1 Output

Baseline output.
`,
		filepath.Join("working", "evals", "summaries", runID+"-summary.md"): "# Eval Run Summary\n",
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()
	summaryRel := filepath.Join("working", "evals", "summaries", runID+"-summary.md")
	for index, item := range got.evalItems {
		if item.RelPath == summaryRel {
			got.evalTable.SetCursor(index)
			break
		}
	}

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "comparison") {
		t.Fatalf("expected comparison preview from summary, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
}

func TestCreateEvalRunPacketRequiresTasksetSelection(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	summaryRel := filepath.Join("working", "evals", "summaries", "summary.md")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(project, summaryRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, summaryRel), []byte("# Summary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))

	if !strings.Contains(got.err, "Select a taskset") {
		t.Fatalf("expected taskset selection error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "evals", "runs")); !os.IsNotExist(err) {
		t.Fatalf("run packet should not be created without taskset selection, stat err=%v", err)
	}
}

func TestCreateEvalAutomationPacketRequiresRunSummaryOrComparison(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	tasksetRel := filepath.Join("working", "evals", "tasksets", "taskset.md")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(project, tasksetRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, tasksetRel), []byte("# Taskset\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))

	if !strings.Contains(got.err, "Select a run packet, summary, or comparison") {
		t.Fatalf("expected automation selection error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "evals", "automation")); !os.IsNotExist(err) {
		t.Fatalf("automation packet should not be created from taskset, stat err=%v", err)
	}
}

func TestCreateEvalJudgePacketRequiresRunSummaryOrComparison(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Design Engineering"}); err != nil {
		t.Fatal(err)
	}
	tasksetRel := filepath.Join("working", "evals", "tasksets", "taskset.md")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(project, tasksetRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, tasksetRel), []byte("# Taskset\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		height:      32,
		currentPath: project,
		currentTape: tape.Tape{Title: "Design Engineering"},
		evalTable:   newEvalTable(100, 8),
	}
	got, _ := m.startEvals()

	got, _ = got.handleEvalsKey(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))

	if !strings.Contains(got.err, "Select a run packet, summary, or comparison") {
		t.Fatalf("expected judge selection error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "evals", "judges")); !os.IsNotExist(err) {
		t.Fatalf("judge packet should not be created from taskset, stat err=%v", err)
	}
}

func TestProjectHidesImpactTestsFromV1Surfaces(t *testing.T) {
	project := t.TempDir()
	missing := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}
	if hasHelp(missing.helpForScreen().ShortHelp(), "e") {
		t.Fatal("project without evals should not show e in short help")
	}
	if hasCommandTitle(missing.commandItems(), "Impact Tests") {
		t.Fatal("project without evals should not show Impact Tests command")
	}

	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	starterReady := missing
	if hasHelp(starterReady.helpForScreen().ShortHelp(), "e") {
		t.Fatal("compiled project should not show e in short help before eval artifacts exist")
	}
	if hasCommandTitle(starterReady.commandItems(), "Impact Tests") {
		t.Fatal("compiled project should not show Impact Tests command before eval artifacts exist")
	}
	got, _ := starterReady.handleKey(tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	if got.screen != screenProject {
		t.Fatalf("expected e to stay on Project for compiled v1 project, got %v: %s", got.screen, got.err)
	}

	if err := os.MkdirAll(filepath.Join(project, "working", "evals", "runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "working", "evals", "runs", "result.md"), []byte("# Result\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	available := missing
	if hasHelp(available.helpForScreen().ShortHelp(), "e") {
		t.Fatal("project with parked evals should not show e in short help")
	}
	if hasCommandTitle(available.commandItems(), "Impact Tests") {
		t.Fatal("project with parked evals should not show Impact Tests command")
	}

	got, _ = available.handleKey(tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	if got.screen != screenProject {
		t.Fatalf("expected e to keep parked evals out of v1 Project routes, got %v: %s", got.screen, got.err)
	}
}

func TestCompositionScreenDiscoversAndPreviewsChildArtifacts(t *testing.T) {
	project := t.TempDir()
	paths := map[string]string{
		filepath.Join("children", "ux-specialist.yaml"): "path: ../ux-specialist\n",
		filepath.Join("children", "ui-specialist.md"):   "# UI Specialist\n\nwarning: visual rules overlap\n",
		"lineage.yaml": "parents:\n  - design-engineering\n",
	}
	for rel, body := range paths {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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

	if got.screen != screenComposition {
		t.Fatalf("expected composition screen, got %v: %s", got.screen, got.err)
	}
	if len(got.compositionItems) != 3 {
		t.Fatalf("expected two child artifacts plus lineage, got %#v", got.compositionItems)
	}
	kinds := []string{got.compositionItems[0].Kind, got.compositionItems[1].Kind, got.compositionItems[2].Kind}
	statuses := []string{got.compositionItems[0].Status, got.compositionItems[1].Status, got.compositionItems[2].Status}
	if !containsString(kinds, "child ref") || !containsString(kinds, "child notes") || !containsString(kinds, "lineage") {
		t.Fatalf("expected child and lineage kinds, got %#v", got.compositionItems)
	}
	if !containsString(statuses, "ready") || !containsString(statuses, "review") || !containsString(statuses, "history") {
		t.Fatalf("expected inferred composition statuses, got %#v", got.compositionItems)
	}
	view := got.viewComposition()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Composition", "ux-specialist", "ui-specialist", "lineage.yaml", "review", "Selected", "Field", "Value", "Route", "ui, specialist", "Path", "Actions", "enter / o", "Check promotion readiness", "m / b", "Draft merge route or LINER blend", "Review parent skill conflicts", "c / a", "Advanced: production-merge child"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("composition view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"n / r", "Nest children or audit routes"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("composition child view should be contextual and not show %q:\n%s", unexpected, view)
		}
	}
	if strings.Contains(view, "n nests. m drafts merge") {
		t.Fatalf("composition view should not render the old long command sentence:\n%s", view)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "m") {
		t.Fatal("composition child help should expose the merge-draft key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "b") {
		t.Fatal("composition child help should expose the child LINER blend key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "p") {
		t.Fatal("composition child help should expose the promote-check key")
	}
	lineageIndex := -1
	for index, item := range got.compositionItems {
		if item.Kind == "lineage" {
			lineageIndex = index
			break
		}
	}
	if lineageIndex < 0 {
		t.Fatal("expected lineage item")
	}
	got.compositionTable.SetCursor(lineageIndex)
	lineageView := got.viewComposition()
	for _, expected := range []string{"Preview or open lineage", "Refresh nested child routing", "Audit child route overlap", "Draft route conflict resolution"} {
		if !strings.Contains(lineageView, expected) {
			t.Fatalf("composition lineage view missing %q:\n%s", expected, lineageView)
		}
	}
	if strings.Contains(lineageView, "Advanced: production-merge child") {
		t.Fatalf("composition lineage view should not show child merge actions:\n%s", lineageView)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "n") {
		t.Fatal("composition lineage help should expose the nesting key")
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "d") {
		t.Fatal("composition lineage help should expose the route resolution draft key")
	}
	got.compositionTable.SetCursor(0)

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen != screenPreview || !strings.HasPrefix(got.previewRel, "children") {
		t.Fatalf("expected enter to preview selected composition artifact, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	if action := got.nextAction(); !strings.Contains(action, "children") {
		t.Fatalf("expected composition preview next action, got %q", action)
	}
}

func TestCreateCompositionNestingWritesLineageDraftAndAudit(t *testing.T) {
	project := t.TempDir()
	paths := map[string]string{
		filepath.Join("children", "ux-specialist.yaml"): "path: ../ux-specialist\nroute: research, flows, IA\n",
		filepath.Join("children", "ui-specialist.md"):   "# UI Specialist\n\nscope: hierarchy, layout, patterns\nwarning: visual rules overlap\n",
		"lineage.yaml": "parents:\n  - design-engineering\n",
	}
	for rel, body := range paths {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))

	if got.screen != screenCompositionReview {
		t.Fatalf("expected composition draft review after nesting, got %v: %s", got.screen, got.err)
	}
	if len(got.compositionItems) != 3 {
		t.Fatalf("expected children plus lineage after nesting, got %#v", got.compositionItems)
	}
	if got.previewRel != compositionDraftRelPath {
		t.Fatalf("expected composition draft preview, got %q", got.previewRel)
	}
	lineage, err := os.ReadFile(filepath.Join(project, "lineage.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	lineageBody := string(lineage)
	for _, expected := range []string{"parent: Product Design", "mode: nested", "ux-specialist", "children/ux-specialist.yaml", "research, flows, IA", "working/LINER-composition-draft.md", "previous_copy:"} {
		if !strings.Contains(lineageBody, expected) {
			t.Fatalf("lineage missing %q:\n%s", expected, lineageBody)
		}
	}
	backups, err := filepath.Glob(filepath.Join(project, "working", "composition", "*-previous-lineage.yaml"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one lineage backup, got %#v err=%v", backups, err)
	}
	draft, err := os.ReadFile(filepath.Join(project, "working", "LINER-composition-draft.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Product Design Composition Routing Draft", "Child Mixtape Routing", "research, flows, IA", "hierarchy, layout, patterns", "do not merge child source claims"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("composition draft missing %q:\n%s", expected, string(draft))
		}
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-nesting.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one composition audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Composition Nesting Audit", "Child mixtapes remain referenced by path", "no child sources were copied", "working/LINER-composition-draft.md"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("composition audit missing %q:\n%s", expected, string(audit))
		}
	}
}

func TestCreateCompositionMergeDraftOpensReviewWithoutCopyingChild(t *testing.T) {
	project := t.TempDir()
	childRel := filepath.Join("children", "ux-specialist.yaml")
	childBody := "path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, childRel), []byte(childBody), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:           screenProject,
		width:            100,
		height:           32,
		currentPath:      project,
		currentTape:      tape.Tape{Title: "Product Design"},
		compositionTable: newCompositionTable(100, 8),
		preview:          viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startComposition()

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'm', Text: "m"}))

	if got.screen != screenCompositionReview {
		t.Fatalf("expected composition review after merge draft, got %v: %s", got.screen, got.err)
	}
	if got.previewRel != compositionDraftRelPath {
		t.Fatalf("expected merge draft preview, got %q", got.previewRel)
	}
	draft, err := os.ReadFile(filepath.Join(project, compositionDraftRelPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Product Design Composition Merge Draft", "Promoted Child Route", "ux-specialist", "research, flows, IA", "no child sources are copied"} {
		if !strings.Contains(string(draft), expected) {
			t.Fatalf("composition merge draft missing %q:\n%s", expected, string(draft))
		}
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-merge-draft.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one composition merge draft audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Composition Merge Draft Audit", "without copying child content", "LINER.md` is unchanged", "No child sources, skills, or files were copied"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("composition merge audit missing %q:\n%s", expected, string(audit))
		}
	}
	active, err := os.ReadFile(filepath.Join(project, childRel))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != childBody {
		t.Fatalf("merge draft should not rewrite child reference:\n%s", active)
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); !os.IsNotExist(err) {
		t.Fatalf("merge draft should not write LINER.md before review, stat err=%v", err)
	}
}

func TestCreateCompositionCopyPacketInventoriesSelectedChild(t *testing.T) {
	project := t.TempDir()
	childProject := filepath.Join(project, "ux-specialist")
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(childProject, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(childProject, "local-sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(childProject, "working", "audits"), 0o755); err != nil {
		t.Fatal(err)
	}
	childRel := filepath.Join("children", "ux-specialist.yaml")
	childBody := "path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"
	if err := os.WriteFile(filepath.Join(project, childRel), []byte(childBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "MIXTAPE.md"), []byte("# UX Specialist\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "LINER.md"), []byte("# UX Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tape.WriteProject(childProject, tape.Tape{
		Title: "UX Specialist",
		Sources: []tape.Source{{
			Type: "web",
			URL:  "https://example.com/ux",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "skills", "ux-research.md"), []byte("# UX Research\n\n## Source Grounding\n\nUse MIXTAPE.md.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "local-sources", "interview.md"), []byte("# Interview\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "working", "audits", "route.md"), []byte("# Route Audit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:           screenProject,
		width:            100,
		height:           32,
		currentPath:      project,
		currentTape:      tape.Tape{Title: "Product Design"},
		compositionTable: newCompositionTable(100, 8),
		preview:          viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startComposition()

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "copy-packet") {
		t.Fatalf("expected copy packet preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	packet, err := os.ReadFile(filepath.Join(project, got.previewRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# ux-specialist Composition Copy Packet",
		"Copy Candidates",
		"MIXTAPE.md",
		"LINER.md",
		"tape.yaml",
		"1 saved source",
		"skills/*.md",
		"1 skill",
		"local-sources/",
		"working/audits/*.md",
		"no child files, source files, skills",
	} {
		if !strings.Contains(string(packet), expected) {
			t.Fatalf("composition copy packet missing %q:\n%s", expected, packet)
		}
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-copy-packet.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one composition copy packet audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Composition Copy Packet Audit", "before any content copy", "No child files, source files", got.previewRel} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("composition copy packet audit missing %q:\n%s", expected, audit)
		}
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing for copy packet, got %#v", items)
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); !os.IsNotExist(err) {
		t.Fatalf("copy packet should not write parent LINER.md, stat err=%v", err)
	}
}

func TestApplyCompositionCopySnapshotsSelectedChild(t *testing.T) {
	project := t.TempDir()
	childProject := filepath.Join(project, "ux-specialist")
	for _, dir := range []string{
		filepath.Join(project, "children"),
		filepath.Join(childProject, "skills"),
		filepath.Join(childProject, "local-sources", "research"),
		filepath.Join(childProject, "working", "audits"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	childRel := filepath.Join("children", "ux-specialist.yaml")
	childBody := "path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"
	if err := os.WriteFile(filepath.Join(project, childRel), []byte(childBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "MIXTAPE.md"), []byte("# UX Specialist\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "LINER.md"), []byte("# UX Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tape.WriteProject(childProject, tape.Tape{
		Title: "UX Specialist",
		Sources: []tape.Source{{
			Type: "web",
			URL:  "https://example.com/ux",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "skills", "ux-research.md"), []byte("# UX Research\n\n## Source Grounding\n\nUse MIXTAPE.md.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "local-sources", "interview.md"), []byte("# Interview\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "local-sources", "research", "notes.md"), []byte("# Research Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childProject, "working", "audits", "route.md"), []byte("# Route Audit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:           screenProject,
		width:            100,
		height:           32,
		currentPath:      project,
		currentTape:      tape.Tape{Title: "Product Design"},
		compositionTable: newCompositionTable(100, 8),
		preview:          viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	got, _ := m.startComposition()

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "composition-copy-apply") {
		t.Fatalf("expected copy apply audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	snapshots, err := filepath.Glob(filepath.Join(project, "working", "composition", "copied", "ux-specialist", "*"))
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("expected one copied child snapshot, got %#v err=%v", snapshots, err)
	}
	snapshot := snapshots[0]
	for _, rel := range []string{
		"MIXTAPE.md",
		"LINER.md",
		"tape.yaml",
		filepath.Join("skills", "ux-research.md"),
		filepath.Join("local-sources", "interview.md"),
		filepath.Join("local-sources", "research", "notes.md"),
		filepath.Join("audits", "route.md"),
		"README.md",
	} {
		if info, err := os.Stat(filepath.Join(snapshot, rel)); err != nil || info.IsDir() {
			t.Fatalf("expected copied snapshot file %s, stat=%v err=%v", rel, info, err)
		}
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-copy-apply.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one composition copy apply audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Composition Copy Apply Audit", "namespaced parent working snapshot", "MIXTAPE.md", "skills/ux-research.md", "local-sources/research/notes.md", "Parent `LINER.md`, `tape.yaml`, `skills/`, and `local-sources/` were not changed"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("copy apply audit missing %q:\n%s", expected, string(audit))
		}
	}
	readme, err := os.ReadFile(filepath.Join(snapshot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "Composition Copy Snapshot") || !strings.Contains(string(readme), "review evidence") {
		t.Fatalf("snapshot README missing review framing:\n%s", string(readme))
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing for copy apply, got %#v", items)
	}
	for _, rel := range []string{
		"LINER.md",
		"tape.yaml",
		filepath.Join("skills", "ux-research.md"),
		filepath.Join("local-sources", "interview.md"),
	} {
		if _, err := os.Stat(filepath.Join(project, rel)); !os.IsNotExist(err) {
			t.Fatalf("copy apply should not promote %s into parent production files, stat err=%v", rel, err)
		}
	}
}

func TestApplyCompositionCopyRequiresChildSelection(t *testing.T) {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))

	if got.err != "Select a child reference before applying a copy snapshot." {
		t.Fatalf("expected selected-child copy apply guard, got %q", got.err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "composition", "copied", "*")); err != nil || len(matches) != 0 {
		t.Fatalf("copy apply should not create a snapshot for lineage, got %#v err=%v", matches, err)
	}
}

func TestRunCompositionRouteAuditWritesOverlapReport(t *testing.T) {
	project := t.TempDir()
	paths := map[string]string{
		filepath.Join("children", "ux-specialist.yaml"):      "path: ../ux-specialist\nroute: research, flows, IA\n",
		filepath.Join("children", "research-systems.yaml"):   "path: ../research-systems\nroute: research, systems\n",
		filepath.Join("children", "ui-specialist.md"):        "# UI Specialist\n\nscope: hierarchy, layout\nwarning: overlaps UX critique\n",
		filepath.Join("children", "product-design-child.md"): "# Product Design Child\n\nUse as a parent-level bridge.\n",
		"lineage.yaml": "parents:\n  - design-engineering\n",
	}
	for rel, body := range paths {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(project, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "composition-route-audit") {
		t.Fatalf("expected route audit preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-route-audit.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one composition route audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(audit)
	for _, expected := range []string{
		"# Composition Route Audit",
		"Shared route token `research`",
		"research-systems, ux-specialist",
		"`children/product-design-child.md` has no explicit route",
		"line 4: warning: overlaps UX critique",
		"No child files, sources, or operating-layer files were changed",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("composition route audit missing %q:\n%s", expected, body)
		}
	}
	items, err := loadAuditFiles(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Type != "composition" {
		t.Fatalf("expected composition audit listing, got %#v", items)
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); !os.IsNotExist(err) {
		t.Fatalf("route audit should not write LINER.md, stat err=%v", err)
	}
}

func TestRunCompositionPromotionAuditWritesSelectedChildReport(t *testing.T) {
	project := t.TempDir()
	childRel := filepath.Join("children", "ux-specialist.yaml")
	childBody := "path: ../ux-specialist\nroute: research, flows, IA\nstatus: ready\n"
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, childRel), []byte(childBody), 0o644); err != nil {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))

	if got.screen != screenPreview || !strings.Contains(got.previewRel, "composition-promotion-readiness") {
		t.Fatalf("expected promotion readiness preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-promotion-readiness.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one promotion readiness audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(audit)
	for _, expected := range []string{
		"# Composition Promotion Readiness",
		"ux-specialist",
		"research, flows, IA",
		"candidate for promotion review",
		"No child content was copied into the parent",
		"separate apply audit",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("composition promotion audit missing %q:\n%s", expected, body)
		}
	}
	active, err := os.ReadFile(filepath.Join(project, childRel))
	if err != nil {
		t.Fatal(err)
	}
	if string(active) != childBody {
		t.Fatalf("promotion audit should not rewrite child reference:\n%s", active)
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); !os.IsNotExist(err) {
		t.Fatalf("promotion audit should not write LINER.md, stat err=%v", err)
	}
}

func TestRunCompositionPromotionAuditRequiresChildSelection(t *testing.T) {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))

	if got.err != "Select a child reference before checking promotion readiness." {
		t.Fatalf("expected selected-child guard, got %q", got.err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-promotion-readiness.md")); err != nil || len(matches) != 0 {
		t.Fatalf("promotion audit should not be created for lineage, got %#v err=%v", matches, err)
	}
}

func TestCreateCompositionMergeDraftRequiresChildSelection(t *testing.T) {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'm', Text: "m"}))

	if got.err != "Select a child reference before creating a merge draft." {
		t.Fatalf("expected selected-child merge guard, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, compositionDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("merge draft should not be created for lineage, stat err=%v", err)
	}
}

func TestCreateCompositionCopyPacketRequiresChildSelection(t *testing.T) {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if got.err != "Select a child reference before creating a copy packet." {
		t.Fatalf("expected selected-child copy guard, got %q", got.err)
	}
	if matches, err := filepath.Glob(filepath.Join(project, "working", "composition", "*-copy-packet.md")); err != nil || len(matches) != 0 {
		t.Fatalf("copy packet should not be created for lineage, got %#v err=%v", matches, err)
	}
}

func TestAcceptCompositionDraftAppliesRoutingToLiner(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Product Design\n\nExisting operating rules.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	draft := "# Product Design Composition Routing Draft\n\n## Child Mixtape Routing\n\n| Child | Route | Status | Reference |\n| --- | --- | --- | --- |\n| ux-specialist | research, flows, IA | ready | `children/ux-specialist.yaml` |\n"
	if err := os.WriteFile(filepath.Join(project, compositionDraftRelPath), []byte(draft), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompositionReview,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
	}

	got, _ := m.handleCompositionReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenPreview {
		t.Fatalf("expected apply to open LINER.md preview, got %v: %s", got.screen, got.err)
	}
	liner, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(liner)
	for _, expected := range []string{"Existing operating rules", compositionStartMarker, "Child Mixtape Routing", "ux-specialist", compositionEndMarker} {
		if !strings.Contains(body, expected) {
			t.Fatalf("LINER.md missing %q:\n%s", expected, body)
		}
	}
	if _, err := os.Stat(filepath.Join(project, compositionDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected composition draft removal after apply, stat err=%v", err)
	}
	backups, err := filepath.Glob(filepath.Join(project, "working", "composition", "*-previous-LINER.md"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one LINER backup, got %#v err=%v", backups, err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-apply.md"))
	if err != nil || len(audits) != 1 {
		t.Fatalf("expected one composition apply audit, got %#v err=%v", audits, err)
	}
	audit, err := os.ReadFile(audits[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# Composition Routing Apply Audit", "managed composition markers", "No child sources were copied"} {
		if !strings.Contains(string(audit), expected) {
			t.Fatalf("apply audit missing %q:\n%s", expected, string(audit))
		}
	}
}

func TestAcceptCompositionDraftReplacesManagedRoutingSection(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldSection := compositionStartMarker + "\nold route\n" + compositionEndMarker
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Product Design\n\n"+oldSection+"\n\nAfter section.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, compositionDraftRelPath), []byte("# New Routing\n\nnew route\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{screen: screenCompositionReview, width: 100, currentPath: project, currentTape: tape.Tape{Title: "Product Design"}}

	got, _ := m.acceptCompositionDraft()

	if got.screen != screenPreview {
		t.Fatalf("expected preview after apply, got %v: %s", got.screen, got.err)
	}
	liner, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(liner)
	if strings.Contains(body, "old route") || !strings.Contains(body, "new route") || !strings.Contains(body, "After section.") {
		t.Fatalf("managed section was not replaced correctly:\n%s", body)
	}
}

func TestCompositionReviewUsesPlainViewportWithoutOuterBox(t *testing.T) {
	m := Model{
		screen:  screenCompositionReview,
		width:   100,
		height:  32,
		preview: viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}
	m.setPreviewContent(compositionDraftRelPath, "# Composition Draft\n\nReview child routes before applying.")

	view := m.viewCompositionReview()

	for _, expected := range []string{"Review Composition Draft", "Review child routes", "Actions", "Apply to LINER.md", "Open draft", "Discard draft"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("composition review missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Enter applies to LINER.md.") {
		t.Fatalf("composition review should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	assertNoBoxCorners(t, view)
}

func TestDiscardCompositionDraftRemovesDraftWithoutWritingLiner(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, compositionDraftRelPath), []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompositionReview,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
	}

	got, _ := m.discardCompositionDraft()

	if got.screen != screenComposition {
		t.Fatalf("expected discard to return to composition, got %v", got.screen)
	}
	if _, err := os.Stat(filepath.Join(project, compositionDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected draft removal, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); !os.IsNotExist(err) {
		t.Fatalf("discard should not write LINER.md, stat err=%v", err)
	}
}

func TestCreateCompositionNestingRequiresChildReferences(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "lineage.yaml"), []byte("parents:\n  - design-engineering\n"), 0o644); err != nil {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))

	if !strings.Contains(got.err, "Add at least one child reference") {
		t.Fatalf("expected missing-child error, got %q", got.err)
	}
	if _, err := os.Stat(filepath.Join(project, "working", "LINER-composition-draft.md")); !os.IsNotExist(err) {
		t.Fatalf("nesting draft should not be created without children, stat err=%v", err)
	}
}

func TestRunCompositionRouteAuditRequiresChildReferences(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "lineage.yaml"), []byte("parents:\n  - design-engineering\n"), 0o644); err != nil {
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

	got, _ = got.handleCompositionKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))

	if !strings.Contains(got.err, "Add at least one child reference") {
		t.Fatalf("expected missing-child route audit error, got %q", got.err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-composition-route-audit.md"))
	if err != nil || len(audits) != 0 {
		t.Fatalf("route audit should not be created without children, got %#v err=%v", audits, err)
	}
}

func TestProjectHidesCompositionFromV1Surfaces(t *testing.T) {
	project := t.TempDir()
	missing := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Product Design"},
	}
	if hasHelp(missing.helpForScreen().ShortHelp(), "m") {
		t.Fatal("project without children or lineage should not show m in short help")
	}
	if hasCommandTitle(missing.commandItems(), "Composition") {
		t.Fatal("project without children or lineage should not show Composition command")
	}

	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "children", "ux-specialist.yaml"), []byte("path: ../ux\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	available := missing
	if hasHelp(available.helpForScreen().ShortHelp(), "m") {
		t.Fatal("project with children should not show m in v1 short help")
	}
	if hasCommandTitle(available.commandItems(), "Composition") {
		t.Fatal("project with children should not show Composition command in v1")
	}

	got, _ := available.handleKey(tea.KeyPressMsg(tea.Key{Code: 'm', Text: "m"}))
	if got.screen != screenProject {
		t.Fatalf("expected m to keep composition out of v1 Project routes, got %v: %s", got.screen, got.err)
	}
}

func TestProjectLKeyOpensLinerPreviewWhenAvailable(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))

	if got.screen != screenPreview {
		t.Fatalf("expected l to open LINER.md preview, got %v: %s", got.screen, got.err)
	}
}

func TestProjectLinerControlOnlyShowsWhenAvailable(t *testing.T) {
	project := t.TempDir()
	missing := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}
	if hasHelp(missing.helpForScreen().ShortHelp(), "l") {
		t.Fatal("missing LINER.md should not show l in short help")
	}
	if hasCommandTitle(missing.commandItems(), "Preview LINER.md") {
		t.Fatal("missing LINER.md should not show preview command")
	}
	if hasCommandTitle(missing.commandItems(), "Create Operating Layer") {
		t.Fatal("project without MIXTAPE.md should not show operating-layer command")
	}

	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compiled := missing
	if !hasHelp(compiled.helpForScreen().ShortHelp(), "l") {
		t.Fatal("compiled project without LINER.md should show l in short help")
	}
	if !hasCommandTitle(compiled.commandItems(), "Create Operating Layer") {
		t.Fatal("compiled project without LINER.md should show operating-layer command")
	}

	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	available := missing
	if !hasHelp(available.helpForScreen().ShortHelp(), "l") {
		t.Fatal("available LINER.md should show l in short help")
	}
	if !hasHelp(available.helpForScreen().ShortHelp(), "r") {
		t.Fatal("available LINER.md should show r regenerate in short help")
	}
	if !hasCommandTitle(available.commandItems(), "Preview LINER.md") {
		t.Fatal("available LINER.md should show preview command")
	}
	if !hasCommandTitle(available.commandItems(), "Regenerate Operating Layer") {
		t.Fatal("available LINER.md should show regenerate command")
	}
	if hasCommandTitle(available.commandItems(), "Create Operating Layer") {
		t.Fatal("available LINER.md should not show operating-layer command")
	}

	got, cmd := available.handleKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))
	if cmd != nil {
		t.Fatal("r should open the regeneration review before writing files")
	}
	if got.screen != screenLinerReview {
		t.Fatalf("r should open Operating Layer regeneration review, got %v: %s", got.screen, got.err)
	}
}

func TestProjectReopenRoutesPartialCompileBackToCompile(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\nmixtape: mixtape\nstatus:\n  milestone: corpus_ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceURL := "https://smarthistory.org/visual-analysis/?sidebar=the-basics-of-art-history"
	current := tape.Tape{
		Title: "Art Director",
		Sources: []tape.Source{{
			Type: "web",
			URL:  sourceURL,
		}},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectAbsPath(project, "sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	mixtape := "# Art Director\n\n## Compilation notes\n\n- **" + sourceURL + "** — Failed to fetch " + sourceURL + " — category: js_required; status: HTTP 403; body preview: <!DOCTYPE html>. Install JS-rendering support: liner setup-js\n"
	if err := os.WriteFile(projectAbsPath(project, "MIXTAPE.md"), []byte(mixtape), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectAbsPath(project, "sources/01-visual-analysis.md"), []byte("# Missing\n\n_Source unavailable. See compilation notes in MIXTAPE.md._\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       118,
		height:      34,
		currentPath: project,
		currentTape: current,
		help:        help.New(),
		compileBar:  newCompileProgress(48),
		statusPath:  project,
		status: &core.ProjectStatus{
			Snapshot: core.StatusSnapshot{Milestone: "corpus_ready"},
		},
	}
	m.help.SetWidth(114)

	if got := homeProjectStatus(core.ProjectSummary{Path: project, SourceCount: 1}, capabilitySummary{}); got != "Compile Needs Attention" {
		t.Fatalf("project browser should surface partial compile, got %q", got)
	}
	if got := m.projectMilestone(); got != "compile_attention" {
		t.Fatalf("project milestone should not trust stale corpus_ready status, got %q", got)
	}
	if got := m.projectPrimaryLabel(); got != "Review compile issues" {
		t.Fatalf("primary label should route to compile repair, got %q", got)
	}
	if got := m.projectMilestoneNextAction(); got != "Review compile issues and retry compile." {
		t.Fatalf("next action should route to compile repair, got %q", got)
	}
	if m.canCreateOperatingLayer() {
		t.Fatal("partial compile should block Operating Layer creation")
	}
	if hasCommandTitle(m.commandItems(), "Create Operating Layer") {
		t.Fatal("partial compile should hide Create Operating Layer command")
	}
	if !hasHelpDesc(m.helpForScreen().ShortHelp(), "review compile") {
		t.Fatalf("project help should route enter to compile repair: %#v", m.helpForScreen().ShortHelp())
	}
	if view := stripANSICodesForTest(m.viewProject()); !strings.Contains(view, "Compile Needs Attention") || !strings.Contains(view, "Review compile issues") {
		t.Fatalf("project view should explain compile repair state:\n%s", view)
	}
	rows := projectPipelineRows(project, current, m.currentProjectStatus())
	foundCompile := false
	for _, row := range rows {
		if row.Phase == "Compile" {
			foundCompile = true
			if row.State != "current" || !strings.Contains(row.Evidence, "needs attention") {
				t.Fatalf("compile row should be current and needs attention, got %#v", row)
			}
		}
	}
	if !foundCompile {
		t.Fatalf("expected compile row in pipeline: %#v", rows)
	}

	blocked, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if !strings.Contains(blocked.err, "Review compile issues") {
		t.Fatalf("l should explain why Operating Layer is blocked, got %q", blocked.err)
	}

	got, _ := m.primaryProjectAction()
	if got.screen != screenCompile {
		t.Fatalf("enter should reopen Compile screen, got %v", got.screen)
	}
	if got.compileResult == nil || len(got.compileResult.Warnings) != 1 {
		t.Fatalf("compile repair screen should reconstruct warning state: %#v", got.compileResult)
	}
	if !got.compileNeedsJSSetup() {
		t.Fatal("reconstructed compile issues should offer JS setup")
	}
	if action := got.nextAction(); !strings.Contains(action, "install JS rendering") {
		t.Fatalf("compile repair next action should offer JS setup, got %q", action)
	}
}

func TestProjectReopenRecoveredJSCompileDoesNotBlockOperatingLayer(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\nmixtape: mixtape\nstatus:\n  milestone: corpus_ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceURL := "https://smarthistory.org/visual-analysis/?sidebar=the-basics-of-art-history"
	current := tape.Tape{
		Title: "Art Director",
		Sources: []tape.Source{{
			Type: "web",
			URL:  sourceURL,
		}},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	mixtape := "# Art Director\n\n## Compilation notes\n\n- **" + sourceURL + "** — Recovered this source with JS rendering after the first fetch returned a JavaScript-only stub. The rendered content was included in MIXTAPE.md.\n"
	if err := os.WriteFile(projectAbsPath(project, "MIXTAPE.md"), []byte(mixtape), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       118,
		height:      34,
		currentPath: project,
		currentTape: current,
		help:        help.New(),
		compileBar:  newCompileProgress(48),
		statusPath:  project,
		status: &core.ProjectStatus{
			Snapshot: core.StatusSnapshot{Milestone: "corpus_ready"},
		},
	}
	m.help.SetWidth(114)

	if got := homeProjectStatus(core.ProjectSummary{Path: project, SourceCount: 1}, capabilitySummary{}); got != "Corpus Ready" {
		t.Fatalf("project browser should not surface recovered JS notes as compile attention, got %q", got)
	}
	if got := m.projectMilestone(); got != "corpus_ready" {
		t.Fatalf("recovered JS compile note should preserve corpus_ready status, got %q", got)
	}
	if !m.canCreateOperatingLayer() {
		t.Fatal("recovered JS compile note should not block Operating Layer creation")
	}
	if got := m.projectPrimaryLabel(); got != "Create Operating Layer" {
		t.Fatalf("primary label should continue to Operating Layer, got %q", got)
	}
	if hasCommandTitle(m.commandItems(), "Review compile issues") {
		t.Fatal("recovered JS compile note should not show compile repair command")
	}
}

func TestProjectLKeyMissingLinerShowsClearError(t *testing.T) {
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))

	if got.screen != screenProject {
		t.Fatalf("missing LINER.md should stay on project, got %v", got.screen)
	}
	if !strings.Contains(got.err, "Reach Corpus Ready before creating the Operating Layer") {
		t.Fatalf("expected clear missing LINER.md error, got %q", got.err)
	}
}

func TestProjectLKeyOpensOperatingLayerCreationPlan(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n\nUse the corpus."), 0o644); err != nil {
		t.Fatal(err)
	}
	jtbd := "When I need an operating layer, I want grounded rules."
	kind := "principle"
	section := "foundations"
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Launch",
			JTBD:  &jtbd,
			Sources: []tape.Source{{
				Type:    "web",
				URL:     "https://example.com",
				Kind:    &kind,
				Section: &section,
			}},
		},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))

	if got.screen != screenLinerReview {
		t.Fatalf("expected l to open Operating Layer creation plan, got %v: %s", got.screen, got.err)
	}
	view := stripANSICodesForTest(got.viewLinerReview())
	for _, expected := range []string{
		"Create Operating Layer",
		"Creates the operating instructions and root SKILL.md entrypoint for this corpus",
		"LINER.md",
		"Turns MIXTAPE.md into operating guidance",
		"when to abstain",
		"SKILL.md",
		"Adds the root skill entrypoint",
		"find this project by name",
		"load LINER.md and MIXTAPE.md in the right order",
		"stay inside the corpus",
		"Next: Create Operating Layer",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("Operating Layer plan missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Enter writes these local files now", "What this adds", "liner-launch", "liner.yaml", "instead of duplicating", "skills/liner-launch.md", "Project Status", "Creation Review", "Audit Review", "full Project Skill", "minimal launcher skill", "working/audits/"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("Operating Layer plan should not show %q:\n%s", unexpected, view)
		}
	}
	draft, err := buildLinerContent(project, m.currentTape)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Launch",
		"## How To Use This Project",
		"Do not present the project as a persona",
		"## Working Loop",
		"## Resource Map",
		"Project Skill: `liner-launch`",
		"## Source Use Rules",
		"principle 1",
		"foundations 1",
		"## Project Skill",
		"## Maintenance Rules",
	} {
		if !strings.Contains(draft, expected) {
			t.Fatalf("generated LINER.md missing %q:\n%s", expected, draft)
		}
	}
	for _, unexpected := range []string{"operating conscience", "## Impact Test Rules", "Available skill files", "## Creation Review", "LINER.md Generation Audit"} {
		if strings.Contains(draft, unexpected) {
			t.Fatalf("generated LINER.md should not contain %q:\n%s", unexpected, draft)
		}
	}
	if _, err := os.Stat(filepath.Join(project, legacyLinerDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("opening the Operating Layer screen should not write a draft, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); !os.IsNotExist(err) {
		t.Fatalf("LINER.md should not be written before creation, stat err=%v", err)
	}
}

func TestLinerContentKeepsProjectSkillSingular(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n\nUse the corpus."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	skills := map[string]string{
		filepath.Join("skills", "critique.md"): "# Critique\n\n## Source Grounding\n\nMIXTAPE.md\n\n## Boundaries\n\nUse only inside this corpus.\n",
		filepath.Join("skills", "loose.md"):    "# Loose\n\nUse taste for everything.\n",
	}
	for rel, body := range skills {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	content, err := buildLinerContent(project, tape.Tape{Title: "Design Ops"}, "active")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Project Skill: `liner-design-ops`",
		"Active Project Skill",
		"`SKILL.md`",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("project-skill LINER content missing %q:\n%s", expected, content)
		}
	}
	for _, unexpected := range []string{"Available skill files", "`skills/critique.md`", "`skills/loose.md`", "No separate skill files are generated", "skills/liner-design-ops.md"} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("project-skill LINER content should not expose plural skill inventory %q:\n%s", unexpected, content)
		}
	}
}

func TestOperatingLayerUsesSynthesisAndQualityContract(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n\nUse the corpus."), 0o644); err != nil {
		t.Fatal(err)
	}
	synthesis := `# Launch - Synthesis

This corpus treats launch pages as trust-building flow, not hero decoration. It turns research signals into interface priorities.

## Generative rules

- Lead with the trust signal before the decorative claim.
- Translate audience anxiety into visible proof, defaults, and recovery paths.
- Hand off decisions as tokens, layout primitives, and acceptance checks.
`
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte(synthesis), 0o644); err != nil {
		t.Fatal(err)
	}
	quality := `# Quality Checks

Core action: translate launch references into trust-building web interface rules.

## Test 8 — Capability-pattern fit

Pattern: reference-translation

Input/reference domains: service pages, product onboarding, and trust-supporting proof examples.

Target/caller handoff: tokens, hierarchy rules, proof modules, state behavior, and checks for implementation agents.

Finding: pass. The corpus supports a concrete operating layer rather than a generic summary.
`
	if err := os.WriteFile(filepath.Join(project, "working", "04-quality-checks.md"), []byte(quality), 0o644); err != nil {
		t.Fatal(err)
	}
	current := tape.Tape{
		Title:   "Launch",
		Version: 1,
		Sources: projectSkillRecommendationTestSources(),
	}

	content, err := buildLinerContent(project, current)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"## Corpus-Derived Operating Contract",
		"Core action: translate launch references into trust-building web interface rules.",
		"Operating thesis: This corpus treats launch pages as trust-building flow, not hero decoration.",
		"Capability pattern: reference-translation.",
		"Input domains: service pages, product onboarding, and trust-supporting proof examples.",
		"Caller handoff: tokens, hierarchy rules, proof modules, state behavior, and checks for implementation agents.",
		"## Required Method",
		"Lead with the trust signal before the decorative claim.",
		"Translate audience anxiety into visible proof, defaults, and recovery paths.",
		"## Quality Gate",
		"pass. The corpus supports a concrete operating layer rather than a generic summary.",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("generated LINER.md missing corpus contract %q:\n%s", expected, content)
		}
	}

	name, relPath, err := writeProjectSkillFile(project, current)
	if err != nil {
		t.Fatal(err)
	}
	if name != "liner-launch" || relPath != "SKILL.md" {
		t.Fatalf("unexpected skill identity: %s %s", name, relPath)
	}
	skill, err := os.ReadFile(filepath.Join(project, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"## Corpus Method",
		"Core action: translate launch references into trust-building web interface rules.",
		"Capability pattern: reference-translation.",
		"Caller handoff: tokens, hierarchy rules, proof modules, state behavior, and checks for implementation agents.",
		"Apply these corpus-derived rules:",
		"Lead with the trust signal before the decorative claim.",
	} {
		if !strings.Contains(string(skill), expected) {
			t.Fatalf("generated SKILL.md missing corpus method %q:\n%s", expected, skill)
		}
	}
}

func TestLinerContentReportsUnavailableCompiledSources(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n\nUse the corpus."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourceFiles := map[string]string{
		filepath.Join("sources", "01-usable.md"):      "# Usable\n\nSource body.",
		filepath.Join("sources", "02-unavailable.md"): "# Missing\n\n_Source unavailable. See compilation notes in MIXTAPE.md._",
	}
	for rel, body := range sourceFiles {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	current := tape.Tape{
		Title:   "Art Direction",
		Version: 1,
		Sources: []tape.Source{
			{Type: "web", URL: "https://example.com/usable"},
			{Type: "web", URL: "https://example.com/unavailable"},
		},
	}

	content, err := buildLinerContent(project, current)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Compiled availability: 1 usable compiled source file(s), 1 unavailable placeholder(s); do not cite unavailable sources as evidence."
	if !strings.Contains(content, expected) {
		t.Fatalf("LINER content missing availability warning %q:\n%s", expected, content)
	}
}

func TestLinerContentDoesNotExposeCompositionRouting(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Product Design\n\nUse the parent corpus."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "children"), 0o755); err != nil {
		t.Fatal(err)
	}
	children := map[string]string{
		filepath.Join("children", "ux-specialist.yaml"): "path: ../ux-specialist\nroute: research, flows, IA\n",
		filepath.Join("children", "ui-specialist.md"):   "# UI Specialist\n\nwarning: visual system rules overlap with parent.\n",
	}
	for rel, body := range children {
		if err := os.WriteFile(filepath.Join(project, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, "lineage.yaml"), []byte("parent: Product Design\nmode: nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	content, err := buildLinerContent(project, tape.Tape{Title: "Product Design"})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Project Skill: `liner-product-design`",
		"Active Project Skill",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("LINER content missing %q:\n%s", expected, content)
		}
	}
	for _, unexpected := range []string{"## Child Routing", "Composition route audit", "Composition merge", "ux-specialist", "`children/`"} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("LINER content should not expose composition routing %q:\n%s", unexpected, content)
		}
	}
}

func projectSkillRecommendationTestSources() []tape.Source {
	sections := []string{"foundations", "interaction", "implementation"}
	kinds := []string{"principle", "prescription", "principle", "prescription"}
	sources := make([]tape.Source, 0, 8)
	for i := 0; i < 8; i++ {
		section := sections[i%len(sections)]
		kind := kinds[i%len(kinds)]
		sourceType := "web"
		if i == 0 {
			sourceType = "skill"
		}
		sources = append(sources, tape.Source{
			Type:    sourceType,
			URL:     fmt.Sprintf("https://example.com/source-%d", i+1),
			Section: &section,
			Kind:    &kind,
		})
	}
	return sources
}

func TestLinerReviewShowsOperatingLayerOutputs(t *testing.T) {
	m := Model{
		screen: screenLinerReview,
		width:  100,
		height: 32,
		currentTape: tape.Tape{
			Sources: projectSkillRecommendationTestSources(),
		},
		preview: viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}

	view := m.viewLinerReview()
	plain := stripANSICodesForTest(view)

	for _, expected := range []string{
		"Create Operating Layer",
		"Creates the operating instructions and root SKILL.md entrypoint for this corpus",
		"LINER.md",
		"Turns MIXTAPE.md into operating guidance",
		"when to abstain",
		"SKILL.md",
		"Adds the root skill entrypoint",
		"find this project by name",
		"load LINER.md and MIXTAPE.md in the right order",
		"stay inside the corpus",
		"Next: Create Operating Layer",
	} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("LINER review missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Enter writes these local files now", "What this adds", "liner-project", "liner.yaml", "instead of duplicating", "Project Status", "Review me", "Creation Review", "Audit Review", "full Project Skill", "minimal launcher skill", "working/audits/", "Field", "Value", "Project Skill required", "Project Skill Decision", "This is the only required choice", "Decision required", "What Liner Will Do", "Writes", "Draft", "Purpose", "Open", "Press a to accept or x to decline", "Decline Project Skill", "Continue without Project Skill", "Actions", "[chosen]"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("LINER review should not show old decision UI %q:\n%s", unexpected, view)
		}
	}
	if got := m.nextAction(); got != "" {
		t.Fatalf("LINER review should not render a Next activity row, got %q", got)
	}
	assertNoBoxCorners(t, view)
}

func TestLinerReviewShowsMinimalProjectSkillForThinCorpus(t *testing.T) {
	reference := "reference"
	section := "reference"
	m := Model{
		screen: screenLinerReview,
		width:  100,
		currentTape: tape.Tape{Sources: []tape.Source{{
			Type:    "web",
			URL:     "https://example.com/reference",
			Kind:    &reference,
			Section: &section,
		}}},
		preview: viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}

	view := m.viewLinerReview()
	plain := stripANSICodesForTest(view)

	for _, expected := range []string{"root SKILL.md entrypoint", "SKILL.md", "Adds the root skill entrypoint", "Next: Create Operating Layer"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("LINER review missing minimal skill copy %q:\n%s", expected, plain)
		}
	}
	if strings.Contains(plain, "skill will stay short") || strings.Contains(plain, "Continue without Project Skill") || strings.Contains(plain, "doesn't recommend") || strings.Contains(plain, "minimal launcher skill") {
		t.Fatalf("Operating Layer screen should not frame Project Skill as a recommendation decision:\n%s", plain)
	}
}

func runOperatingLayerCreationForTest(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		next, nextCmd := m.Update(msg)
		model, ok := next.(Model)
		if !ok {
			t.Fatalf("expected Model update, got %T", next)
		}
		m = model
		cmd = nextCmd
	}
	return m
}

func TestLinerReviewEnterStartsOperatingLayerCreation(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, legacyLinerDraftRelPath), []byte("# Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:       screenLinerReview,
		width:        100,
		currentPath:  project,
		currentTape:  tape.Tape{Title: "Launch"},
		researchSpin: newLoadingSpinner(),
	}

	got, cmd := m.handleLinerReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if cmd == nil {
		t.Fatalf("expected Enter to start Operating Layer creation")
	}
	if !got.operatingLayerRunning || got.screen != screenLinerReview {
		t.Fatalf("expected running review screen, got running=%v screen=%v", got.operatingLayerRunning, got.screen)
	}
	rawView := got.viewLinerReview()
	assertTitleLineHasLoader(t, rawView, "Creating Operating Layer")
	view := stripANSICodesForTest(rawView)
	for _, expected := range []string{"Creating Operating Layer", "Working", "Generating LINER.md", "0/3 steps", "queued"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("running view should show %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Project complete") {
		t.Fatalf("running view should show artifact progress:\n%s", view)
	}
	got = runOperatingLayerCreationForTest(t, got, cmd)
	if got.screen != screenLinerReview || !got.operatingLayerComplete {
		t.Fatalf("expected completed Operating Layer screen, got screen=%v complete=%v err=%s", got.screen, got.operatingLayerComplete, got.err)
	}
	completeView := stripANSICodesForTest(got.viewLinerReview())
	for _, expected := range []string{"Project complete", "LINER.md and SKILL.md are ready", "3/3 steps", "Press Enter to go back to Project", "Next: Go back to Project"} {
		if !strings.Contains(completeView, expected) {
			t.Fatalf("completion view missing %q:\n%s", expected, completeView)
		}
	}
	if got.note != "" {
		t.Fatalf("completion screen should not rely on footer note, got %q", got.note)
	}
	if got.operatingLayerRunning || got.operatingLayerContent != "" {
		t.Fatalf("expected transient Operating Layer state to clear")
	}
	if got.projectSkillStatus() != "active" {
		t.Fatalf("expected completed status to keep active Project Skill, got %q", got.projectSkillStatus())
	}
	if got.projectMilestone() != "project_complete" {
		t.Fatalf("expected immediate project completion state, got %q", got.projectMilestone())
	}
	if _, err := os.Stat(filepath.Join(project, "LINER.md")); err != nil {
		t.Fatalf("expected Enter to write LINER.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, legacyLinerDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected draft removal, stat err=%v", err)
	}
	skill, err := os.ReadFile(filepath.Join(project, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected default Project Skill file to be created: %v", err)
	}
	for _, expected := range []string{"---", "name: liner-launch", "description:", "# liner-launch"} {
		if !strings.Contains(string(skill), expected) {
			t.Fatalf("SKILL.md missing %q:\n%s", expected, skill)
		}
	}
	metadata, err := os.ReadFile(filepath.Join(project, "liner.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"milestone: project_complete", "status: active", "name: liner-launch", "path: SKILL.md"} {
		if !strings.Contains(string(metadata), expected) {
			t.Fatalf("liner.yaml missing %q:\n%s", expected, metadata)
		}
	}
	got, cmd = got.handleLinerReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatalf("returning to Project should not need a command")
	}
	if got.screen != screenProject || got.operatingLayerComplete {
		t.Fatalf("expected Enter to return to Project, screen=%v complete=%v", got.screen, got.operatingLayerComplete)
	}
	if got.note != "Created Operating Layer." {
		t.Fatalf("expected project note after returning, got %q", got.note)
	}
}

func TestLinerReviewFitsNarrowWidth(t *testing.T) {
	width := 80
	m := Model{
		screen:  screenLinerReview,
		width:   width,
		height:  24,
		preview: viewport.New(viewport.WithWidth(60), viewport.WithHeight(8)),
	}
	view := m.viewLinerReview()
	assertNoBoxCorners(t, view)
	assertViewLinesFit(t, view, styles.ClampWidth(width-4))
}

func TestOperatingLayerCreationWritesLinerSkillAndMetadata(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacySkill := filepath.Join(project, "skills", "liner-launch.md")
	if err := os.MkdirAll(filepath.Dir(legacySkill), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacySkill, []byte("# Legacy skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenLinerReview,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", Sources: projectSkillRecommendationTestSources()},
	}

	got, cmd := m.handleLinerReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got = runOperatingLayerCreationForTest(t, got, cmd)

	if got.screen != screenLinerReview || !got.operatingLayerComplete {
		t.Fatalf("expected completed Operating Layer screen, got screen=%v complete=%v err=%s", got.screen, got.operatingLayerComplete, got.err)
	}
	if got.hasPreviewBack {
		t.Fatalf("expected create to clear preview back state")
	}
	if got.note != "" {
		t.Fatalf("completion screen should not rely on footer note, got %q", got.note)
	}
	if got.projectSkillStatus() != "active" {
		t.Fatalf("expected completed status to keep active Project Skill, got %q", got.projectSkillStatus())
	}
	liner, err := os.ReadFile(filepath.Join(project, "LINER.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(liner), "operating layer") {
		t.Fatalf("unexpected LINER.md:\n%s", liner)
	}
	if !strings.Contains(string(liner), "Project Skill: `liner-launch`") {
		t.Fatalf("LINER.md should reference default Project Skill:\n%s", liner)
	}
	skill, err := os.ReadFile(filepath.Join(project, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected Project Skill file: %v", err)
	}
	for _, expected := range []string{"---", "name: liner-launch", "description:", "## Source Grounding", "## Boundaries"} {
		if !strings.Contains(string(skill), expected) {
			t.Fatalf("SKILL.md missing %q:\n%s", expected, skill)
		}
	}
	if _, err := os.Stat(legacySkill); !os.IsNotExist(err) {
		t.Fatalf("expected legacy skills/liner-launch.md to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(project, legacyLinerDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected draft removal, stat err=%v", err)
	}
	audits, err := filepath.Glob(filepath.Join(project, "working", "audits", "*-operating-layer.md"))
	if err != nil || len(audits) != 0 {
		t.Fatalf("Operating Layer creation should not write an audit/review file, got %#v err=%v", audits, err)
	}
	metadata, err := os.ReadFile(filepath.Join(project, "liner.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"milestone: project_complete", "state: ready", "status: active", "name: liner-launch", "path: SKILL.md"} {
		if !strings.Contains(string(metadata), expected) {
			t.Fatalf("liner.yaml missing %q:\n%s", expected, metadata)
		}
	}
	if strings.Contains(string(metadata), "audit:") {
		t.Fatalf("liner.yaml should not record an audit for Operating Layer creation:\n%s", metadata)
	}
}

func TestLinerPreviewHidesMixtapeCopyShareControls(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))

	if !strings.Contains(got.viewPreview(), "Preview LINER.md") {
		t.Fatalf("expected LINER.md preview title:\n%s", got.viewPreview())
	}
	assertNoBoxCorners(t, got.viewPreview())
	if action := got.nextAction(); action != "Read LINER.md, or return to Project." {
		t.Fatalf("expected LINER.md next action, got %q", action)
	}
	help := got.helpForScreen().ShortHelp()
	for _, keyName := range []string{"y", "s"} {
		if hasHelp(help, keyName) {
			t.Fatalf("LINER.md preview should hide %s help", keyName)
		}
	}
	got, _ = got.handleKey(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	if !strings.Contains(got.err, "MIXTAPE.md preview") {
		t.Fatalf("expected MIXTAPE-only copy error, got %q", got.err)
	}
}

func TestMixtapePreviewKeepsCopyButHidesShareControls(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, _ := m.openPreview("MIXTAPE.md")

	if !strings.Contains(got.viewPreview(), "Preview MIXTAPE.md") {
		t.Fatalf("expected MIXTAPE.md preview title:\n%s", got.viewPreview())
	}
	assertNoBoxCorners(t, got.viewPreview())
	help := got.helpForScreen().ShortHelp()
	if !hasHelp(help, "y") {
		t.Fatalf("MIXTAPE.md preview should show copy help")
	}
	if hasHelp(help, "s") {
		t.Fatalf("MIXTAPE.md preview should hide share help: %#v", help)
	}
}

func TestCompileLogPreviewShowsFullLogAndReturnsToCompile(t *testing.T) {
	m := Model{
		screen:       screenCompile,
		width:        100,
		currentPath:  t.TempDir(),
		preview:      viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
		compileLines: []string{"fetching first source", "fetching second source", "Compile finished."},
	}

	view := m.viewCompileLog(styles.ClampWidth(m.width - 4))
	if !strings.Contains(view, "v opens the full compile log") {
		t.Fatalf("compile log tail should expose the full-log shortcut:\n%s", view)
	}
	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'v', Text: "v"}))
	if got.screen != screenPreview || got.previewRel != "compile log" {
		t.Fatalf("expected compile log preview, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
	for _, expected := range []string{"fetching first source", "fetching second source", "Compile finished."} {
		if !strings.Contains(got.viewPreview(), expected) {
			t.Fatalf("compile log preview missing %q:\n%s", expected, got.viewPreview())
		}
	}

	back, _ := got.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if back.screen != screenCompile {
		t.Fatalf("escape from compile log preview should return to compile, got %v", back.screen)
	}
}

func TestMixtapePreviewReturnsToOriginScreen(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompile,
		width:       100,
		currentPath: project,
		preview:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(10)),
	}

	got, _ := m.openPreview("MIXTAPE.md")
	if got.screen != screenPreview {
		t.Fatalf("expected preview screen, got %v", got.screen)
	}
	if next := got.nextAction(); !strings.Contains(next, "Compile Console") {
		t.Fatalf("preview next action should name return screen, got %q", next)
	}
	back, _ := got.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if back.screen != screenCompile {
		t.Fatalf("escape from preview should return to origin screen, got %v", back.screen)
	}
}

func hasHelp(bindings []key.Binding, keyName string) bool {
	for _, binding := range bindings {
		if binding.Help().Key == keyName {
			return true
		}
	}
	return false
}

func hasHelpDesc(bindings []key.Binding, desc string) bool {
	for _, binding := range bindings {
		if binding.Help().Desc == desc {
			return true
		}
	}
	return false
}

func hasCommandTitle(items []list.Item, title string) bool {
	for _, item := range items {
		if item.FilterValue() == title || item.(commandItem).Title() == title {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func helpModelWithAll() help.Model {
	model := help.New()
	model.ShowAll = true
	return model
}

func TestActionTableFitsNarrowWidth(t *testing.T) {
	width := 60
	view := newActionTable(width, []actionTableRow{
		{Key: "enter", Action: "Save checked sources", Writes: "source list + compile"},
		{Key: "space", Action: "Toggle selected source", Writes: "review state"},
	}).View()

	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("action table line wider than terminal: got %d, want <= %d\n%s", got, width, line)
		}
	}
}

func TestExpandedHelpStaysInFooter(t *testing.T) {
	width := 80
	m := Model{
		screen: screenProject,
		width:  width,
		help:   helpModelWithAll(),
		currentTape: tape.Tape{
			Title: "Launch",
		},
	}

	view := m.View().Content
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"enter", "add sources", "↑/↓", "sections", "?", "less help"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expanded help view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Keys", "Key     Action"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("expanded help should stay in the footer, found %q:\n%s", unexpected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("expanded help line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestProjectViewOmitsDuplicatedActionRow(t *testing.T) {
	width := 80
	m := Model{
		screen:      screenProject,
		width:       width,
		currentPath: "/tmp/liner-demo",
		currentTape: tape.Tape{
			Title:       "Launch",
			Description: "A generated description for the project workspace.",
			Sources: []tape.Source{
				{Type: "web", URL: "https://example.com"},
			},
		},
	}

	view := m.viewProject()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Launch", "A generated description for the project workspace.", "Sections", "Health", "Primary action", "Continue Corpus Creation"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("project view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Current Liner project", "Mixtape", "Actions", "enter  Continue Corpus Creation", "Commands"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("project view should leave action hints to the global footer, found %q:\n%s", unexpected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("project line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestProjectCompileActionStaysAdvancedUntilCompileReady(t *testing.T) {
	project := t.TempDir()
	early := Model{
		screen:      screenProject,
		width:       90,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Launch",
			JTBD:  stringPointer("Find the best research path."),
		},
	}
	view := early.viewProject()
	if strings.Contains(view, " Compile") {
		t.Fatalf("early project should not advertise manual compile inline:\n%s", view)
	}
	for _, binding := range early.helpForScreen().FullHelp()[0] {
		if hasHelp([]key.Binding{binding}, "c") {
			t.Fatalf("early project full help should not advertise manual compile: %#v", early.helpForScreen().FullHelp()[0])
		}
	}
	if !hasCommandTitle(early.commandItems(), "Compile MIXTAPE.md") {
		t.Fatal("Compile MIXTAPE.md should remain available from Home commands as an advanced action")
	}
	withSources := early
	withSources.currentTape.Sources = []tape.Source{{Type: "web", URL: "https://example.com/research"}}
	view = withSources.viewProject()
	if strings.Contains(view, " Compile") {
		t.Fatalf("project with saved sources should not duplicate manual compile in the body:\n%s", view)
	}
	if !hasHelp(withSources.helpForScreen().FullHelp()[0], "c") {
		t.Fatal("project with saved sources should expose manual compile in full help")
	}
}

func TestCommandListHidesUnavailableProjectActions(t *testing.T) {
	home := Model{screen: screenHome}
	for _, title := range []string{"Add sources", "Compile MIXTAPE.md", "Preview MIXTAPE.md", "Open local-sources"} {
		if hasCommandTitle(home.commandItems(), title) {
			t.Fatalf("home commands should not show project-only command %q", title)
		}
	}
	if !hasCommandTitle(home.commandItems(), "Settings") {
		t.Fatal("home commands should always show Settings")
	}
	for _, unexpected := range []string{"Provider Preferences", "Set Up Liner"} {
		if hasCommandTitle(home.commandItems(), unexpected) {
			t.Fatalf("home commands should not expose separate setup command %q", unexpected)
		}
	}

	project := t.TempDir()
	openProject := Model{
		screen:      screenProject,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Launch",
			JTBD:  stringPointer("Find the best research path."),
		},
	}
	for _, title := range []string{"Add sources", "Compile MIXTAPE.md"} {
		if !hasCommandTitle(openProject.commandItems(), title) {
			t.Fatalf("open project commands should show %q", title)
		}
	}
	if hasCommandTitle(openProject.commandItems(), "Open local-sources") {
		t.Fatal("project without local-sources directory should not show Open local-sources")
	}
	if hasCommandTitle(openProject.commandItems(), "Preview MIXTAPE.md") {
		t.Fatal("uncompiled project should not show Preview MIXTAPE.md")
	}

	if err := os.MkdirAll(filepath.Join(project, "local-sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !hasCommandTitle(openProject.commandItems(), "Open local-sources") {
		t.Fatal("project with local-sources directory should show Open local-sources")
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasCommandTitle(openProject.commandItems(), "Preview MIXTAPE.md") {
		t.Fatal("compiled project should show Preview MIXTAPE.md")
	}
}

func TestSettingsViewShowsProviderSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	configDir := filepath.Join(home, ".liner")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("agent: codex\nmodels:\n  codex:\n    candidates: gpt-5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{screen: screenSettings, width: 118}.startSettings()

	view := m.viewSettings()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{
		"Settings",
		"Choose the AI runner Liner uses to research sources and create project files.",
		"Codex",
		"Claude",
		styles.AccentText.Render("Codex"),
		styles.MutedText.Render("Claude"),
		"Codex CLI. Active runner.",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("settings view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Status:", "Runner:", "Provider Preferences", "Provider:", "Model:", "Commands", "create a new mixtape", "filter projects", "Recommendation", "Installed CLIs", "Model defaults", "Action", "LINER_AGENT", "Press a", "Field", "Value", "Saved", "Default", "Configuration", "Config file", "Env", "Overrides", "codex: candidates=gpt-5", "Codex uses"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("settings should not include command guidance %q:\n%s", unexpected, view)
		}
	}
}

func TestOnboardingLibraryViewIntroducesProjectsFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LINER_DIR", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "")
	baseDir := filepath.Join(home, "liner", "projects")
	m := Model{
		screen:               screenOnboarding,
		width:                118,
		baseDir:              baseDir,
		settings:             readSettingsInfo(),
		onboardingDirInput:   newOnboardingDirInput(baseDir, 118),
		onboardingStep:       onboardingStepLibrary,
		onboardingEditingDir: false,
	}

	view := m.viewOnboarding()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{
		"Setup",
		"Set up Liner's local projects folder.",
		"Liner keeps projects in a visible local folder",
		"Projects folder:",
		"liner/projects",
		".liner/config.yaml",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("onboarding library view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Liner V1", "Source:", "Saved projects_dir", "Default ~/liner/projects", "First Launch", "Project library", "local library", "Liner keeps Liner Projects", "advisor", "living layer", "Impact Tests", "Composition"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("onboarding should avoid parked or persona language %q:\n%s", unexpected, view)
		}
	}
	if got := m.nextAction(); got != "Choose an AI runner." {
		t.Fatalf("onboarding projects-folder next action = %q, want runner step", got)
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "e") || !hasHelpDesc(help, "change folder") {
		t.Fatalf("onboarding library help should explain folder editing: %#v", help)
	}
	banner := stripANSICodesForTest(m.viewBanner())
	if !strings.Contains(banner, "liner v1") || !strings.Contains(banner, "set up") {
		t.Fatalf("onboarding banner should show v1 setup context:\n%s", banner)
	}
	if strings.Contains(banner, "first launch") {
		t.Fatalf("onboarding banner should not say first launch:\n%s", banner)
	}
	assertViewLinesFit(t, view, styles.ClampWidth(m.width-4))
}

func TestOnboardingProviderViewShowsOfficialDocsWhenNoProviderInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "")
	baseDir := filepath.Join(home, "liner", "projects")
	settings := readSettingsInfo()
	m := Model{
		screen:                   screenOnboarding,
		width:                    118,
		baseDir:                  baseDir,
		settings:                 settings,
		onboardingStep:           onboardingStepProvider,
		onboardingProviderCursor: settingsProviderIndex(settings.preferredAgent()),
	}

	view := m.viewOnboarding()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{
		"Setup",
		"Choose the AI runner Liner uses to research sources and create project files.",
		"No Codex or Claude Code runner found.",
		"Codex CLI: https://developers.openai.com/codex/cli",
		"Claude Code quickstart: https://docs.anthropic.com/en/docs/claude-code/quickstart",
		styles.AccentText.Render("Codex"),
		styles.MutedText.Render("Claude"),
		"Codex CLI is not installed. Install Codex CLI to use it here.",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("onboarding provider view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Status:", "Runner:", "Liner V1", "First Launch", "Project library:", "Provider:", "Provider check:", "Config:", "Model:"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("onboarding provider view should avoid metadata row %q:\n%s", unexpected, view)
		}
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "r") {
		t.Fatalf("onboarding provider help should include refresh: %#v", m.helpForScreen().ShortHelp())
	}
	if got := m.nextAction(); got != "Choose JS rendering setup." {
		t.Fatalf("onboarding provider next action = %q, want JS setup step", got)
	}
}

func TestOnboardingProviderCursorShowsSelectedInstallState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	baseDir := filepath.Join(home, "liner", "projects")
	m := Model{
		screen:                   screenOnboarding,
		width:                    118,
		baseDir:                  baseDir,
		settings:                 readSettingsInfo(),
		onboardingStep:           onboardingStepProvider,
		onboardingProviderCursor: settingsProviderIndex("claude"),
	}

	view := m.viewOnboarding()
	for _, expected := range []string{
		"Codex installed; it will be active.",
		styles.MutedText.Render("Codex"),
		styles.AccentText.Render("Claude"),
		"Claude Code is not installed. Install the CLI version of Claude Code to use it here.",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("onboarding provider view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Status:", "Runner:", "Active runner."} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("selected uninstalled provider should not show %q:\n%s", unexpected, view)
		}
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "↑/↓") {
		t.Fatalf("onboarding provider help should advertise provider switching: %#v", m.helpForScreen().ShortHelp())
	}
}

func TestOnboardingProjectLibraryEditUpdatesBaseDir(t *testing.T) {
	home := t.TempDir()
	nextDir := filepath.Join(home, "custom", "projects")
	m := Model{
		screen:               screenOnboarding,
		width:                118,
		baseDir:              filepath.Join(home, "liner", "projects"),
		onboardingDirInput:   newOnboardingDirInput(nextDir, 118),
		onboardingEditingDir: true,
	}

	got, _ := m.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.err != "" {
		t.Fatalf("unexpected onboarding error: %s", got.err)
	}
	if got.baseDir != nextDir {
		t.Fatalf("expected edited project library, got %q want %q", got.baseDir, nextDir)
	}
	if _, err := os.Stat(nextDir); err != nil {
		t.Fatalf("expected project library to be created: %v", err)
	}
}

func TestOnboardingDoesNotSaveUninstalledSelectedProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	configPath := filepath.Join(home, ".liner", "config.yaml")
	baseDir := filepath.Join(home, "liner", "projects")
	m := Model{
		screen:                   screenOnboarding,
		width:                    118,
		baseDir:                  baseDir,
		settings:                 readSettingsInfo(),
		commands:                 newCommandList(70, 16),
		onboardingStep:           onboardingStepProvider,
		onboardingProviderCursor: settingsProviderIndex("claude"),
	}

	got, _ := m.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if !strings.Contains(got.err, "Claude Code is not installed") {
		t.Fatalf("expected missing Claude warning, got %q", got.err)
	}
	if data, err := os.ReadFile(configPath); err == nil && strings.Contains(string(data), "agent: claude") {
		t.Fatalf("should not save uninstalled Claude provider:\n%s", string(data))
	}
}

func TestOnboardingSavesSingleInstalledProviderAndPreservesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	configPath := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("models:\n  codex:\n    candidates: gpt-5\ncustom_field: keep-me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	baseDir := filepath.Join(home, "liner", "projects")
	m := Model{
		screen:                   screenOnboarding,
		width:                    118,
		baseDir:                  baseDir,
		settings:                 readSettingsInfo(),
		commands:                 newCommandList(70, 16),
		onboardingStep:           onboardingStepProvider,
		onboardingProviderCursor: settingsProviderIndex("codex"),
	}

	got, _ := m.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.err != "" {
		t.Fatalf("unexpected onboarding error: %s", got.err)
	}
	if got.screen != screenOnboarding || got.onboardingStep != onboardingStepJS {
		t.Fatalf("expected onboarding to continue to JS setup, got screen=%v step=%d", got.screen, got.onboardingStep)
	}
	got, _ = got.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	got, _ = got.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.err != "" {
		t.Fatalf("unexpected JS setup skip error: %s", got.err)
	}
	if got.screen != screenHome {
		t.Fatalf("expected onboarding to finish on Home, got %v", got.screen)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	for _, expected := range []string{
		"projects_dir: " + baseDir,
		"agent: codex",
		"onboarding_completed: true",
		"jsSetupPrompted: true",
		"jsSetupCompleted: false",
		"custom_field: keep-me",
		"candidates: gpt-5",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("onboarding config missing %q:\n%s", expected, config)
		}
	}
}

func TestOnboardingBothProvidersSavesSelectedProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "/tmp/fake-claude")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	baseDir := filepath.Join(home, "liner", "projects")
	m := Model{
		screen:                   screenOnboarding,
		width:                    118,
		baseDir:                  baseDir,
		settings:                 readSettingsInfo(),
		commands:                 newCommandList(70, 16),
		onboardingStep:           onboardingStepProvider,
		onboardingProviderCursor: settingsProviderIndex("codex"),
	}

	got, _ := m.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	got, _ = got.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.err != "" {
		t.Fatalf("unexpected onboarding error: %s", got.err)
	}
	if got.screen != screenOnboarding || got.onboardingStep != onboardingStepJS {
		t.Fatalf("expected provider selection to continue to JS setup, got screen=%v step=%d", got.screen, got.onboardingStep)
	}
	got, _ = got.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	got, _ = got.handleOnboardingKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	data, err := os.ReadFile(filepath.Join(home, ".liner", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "agent: claude") {
		t.Fatalf("expected selected provider to be saved:\n%s", string(data))
	}
}

func TestOnboardingJSSetupViewExplainsDownloadAndSkip(t *testing.T) {
	home := t.TempDir()
	m := Model{
		screen:         screenOnboarding,
		width:          118,
		baseDir:        filepath.Join(home, "liner", "projects"),
		settings:       settingsInfo{ConfigPath: filepath.Join(home, ".liner", "config.yaml")},
		onboardingStep: onboardingStepJS,
		compileBar:     newCompileProgress(48),
	}

	rawView := m.viewOnboarding()
	view := stripANSICodesForTest(rawView)
	for _, expected := range []string{
		"Set up browser-backed source extraction.",
		"Playwright's headless Chromium",
		"Install",
		"Skip",
		"Download Playwright Chromium (about 150 MB on first run)",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("JS setup onboarding view missing %q:\n%s", expected, view)
		}
	}
	for _, expected := range []string{styles.AccentText.Render("Install"), styles.MutedText.Render("Skip")} {
		if !strings.Contains(rawView, expected) {
			t.Fatalf("JS setup onboarding view missing styled option %q:\n%s", expected, rawView)
		}
	}
	for _, unexpected := range []string{
		"Optional",
		"Not installed by onboarding yet",
		"liner setup-js --yes",
		"Enter:",
		"S:",
		"Install:",
		"Skip:",
		"Skip for now. Liner will offer this again",
	} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("JS setup onboarding view should not show %q:\n%s", unexpected, view)
		}
	}
	m.onboardingJSCursor = onboardingJSOptionSkip
	rawSkipView := m.viewOnboarding()
	skipView := stripANSICodesForTest(rawSkipView)
	if !strings.Contains(skipView, "Skip for now. Liner will offer this again if a source needs browser rendering.") {
		t.Fatalf("JS setup skip selection should show skip detail:\n%s", skipView)
	}
	for _, expected := range []string{styles.AccentText.Render("Skip"), styles.MutedText.Render("Install")} {
		if !strings.Contains(rawSkipView, expected) {
			t.Fatalf("JS setup skip selection missing styled option %q:\n%s", expected, rawSkipView)
		}
	}
	for _, unexpected := range []string{"Download Playwright Chromium", "Install:", "Skip:"} {
		if strings.Contains(skipView, unexpected) {
			t.Fatalf("JS setup skip selection should not show %q:\n%s", unexpected, skipView)
		}
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "↑/↓") || !hasHelp(help, "enter") {
		t.Fatalf("JS setup onboarding help missing option/select: %#v", help)
	}
	for _, unexpected := range []string{"enter/y", "s"} {
		if hasHelp(help, unexpected) {
			t.Fatalf("JS setup onboarding help should not advertise %s: %#v", unexpected, help)
		}
	}
	if got := m.nextAction(); got != "Setup complete." {
		t.Fatalf("unexpected JS setup next action: %q", got)
	}
}

func TestOnboardingJSSetupRunningShowsMovingInstallState(t *testing.T) {
	m := Model{
		screen:         screenOnboarding,
		width:          118,
		baseDir:        "/tmp/liner/projects",
		onboardingStep: onboardingStepJS,
		compileBar:     newCompileProgress(48),
		jsSetupRunning: true,
		researchSpin:   newLoadingSpinner(),
	}

	rawView := m.viewOnboarding()
	assertTitleLineHasLoader(t, rawView, "Setup")
	view := stripANSICodesForTest(rawView)
	for _, expected := range []string{
		"Installing",
		"Downloading Playwright Chromium. First run can take a few minutes.",
		"browser setup",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("running JS setup view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{
		"Skip for now",
		"Download Playwright Chromium (about 150 MB on first run).",
	} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("running JS setup view should not show option detail %q:\n%s", unexpected, view)
		}
	}
	if strings.Contains(rawView, styles.AccentText.Render("Install")) ||
		strings.Contains(rawView, styles.MutedText.Render("Skip")) {
		t.Fatalf("running JS setup view should hide the option selector:\n%s", rawView)
	}

	m.fxFrame = 10
	nextView := m.viewOnboarding()
	if rawView == nextView {
		t.Fatalf("running JS setup view should animate across frames:\n%s", rawView)
	}
	help := m.helpForScreen().ShortHelp()
	if hasHelp(help, "↑/↓") || hasHelp(help, "enter") {
		t.Fatalf("running JS setup help should hide option/select controls: %#v", help)
	}
	if got := m.nextAction(); got != "Setup complete when JS rendering setup finishes." {
		t.Fatalf("unexpected running JS setup next action: %q", got)
	}
}

func TestJSSetupProgressPercentAnimatesWithinBounds(t *testing.T) {
	for _, frame := range []int{-5, 0, 1, 12, 35, 36, 72} {
		got := jsSetupProgressPercent(frame)
		if got < 0.20 || got > 0.75 {
			t.Fatalf("frame %d progress out of expected bounds: %.3f", frame, got)
		}
	}
	if jsSetupProgressPercent(0) == jsSetupProgressPercent(1) {
		t.Fatal("JS setup progress should move between adjacent frames")
	}
	if jsSetupProgressPercent(0) != jsSetupProgressPercent(36) {
		t.Fatal("JS setup progress should loop predictably")
	}
}

func TestCompileJSSetupWarningShowsInstallAction(t *testing.T) {
	m := Model{
		screen:     screenCompile,
		width:      118,
		height:     42,
		compileBar: newCompileProgress(48),
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 1, Succeeded: 0, Failed: 1},
			Warnings: []core.CompileWarningPayload{{
				URL:      "https://example.test/app",
				Message:  "render: js needs Playwright. Run: liner setup-js",
				Severity: "error",
			}},
		},
	}

	view := stripANSICodesForTest(m.viewCompile())
	for _, expected := range []string{"JS rendering needed", "Press i to install JS rendering", "Install JS rendering"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile view missing JS setup cue %q:\n%s", expected, view)
		}
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "i") {
		t.Fatalf("compile help should offer install JS action: %#v", m.helpForScreen().ShortHelp())
	}
	if got := compileWarningRecommendation(m.compileResult.Warnings[0]); !strings.Contains(got, "Install JS rendering") {
		t.Fatalf("expected JS setup recommendation, got %q", got)
	}
}

func TestCompileViewKeepsFooterVisibleWithTallWarnings(t *testing.T) {
	width := 118
	height := 34
	rows := make([]compileSourceRow, 35)
	sources := make([]tape.Source, 35)
	for i := range rows {
		url := fmt.Sprintf("https://smarthistory.org/source-%02d-with-a-long-readable-slug/", i+1)
		rows[i] = compileSourceRow{Status: "done", Type: "web", Source: url, Detail: "1200 chars"}
		sources[i] = tape.Source{Type: "web", URL: url}
	}
	rows[28].Status = "failed"
	rows[28].Detail = "Failed to fetch https://smarthistory.org/visual-analysis/?sidebar=the-basics-of-art-history — category: js_required; status: HTTP 403; retrying via render: js; body preview: <!DOCTYPE html><html><head><meta name=\"robots\" content=\"noindex,nofollow\"></head></html>. Run: liner setup-js"

	warnings := []core.CompileWarningPayload{}
	for i := 0; i < 5; i++ {
		warnings = append(warnings, core.CompileWarningPayload{
			URL:      fmt.Sprintf("https://smarthistory.org/source-%02d-with-a-long-readable-slug/", i+29),
			Severity: "error",
			Message:  "Failed to fetch https://smarthistory.org/visual-analysis/?sidebar=the-basics-of-art-history — category: js_required; status: HTTP 403; content-type: text/html; bot-detection interstitial; retrying via render: js; body preview: <!DOCTYPE html><html lang=\"en-US\"><head><title>Just a moment...</title><meta name=\"robots\" content=\"noindex,nofollow\"></head></html>. Run: liner setup-js",
		})
	}

	m := Model{
		screen:      screenCompile,
		width:       width,
		height:      height,
		help:        help.New(),
		compileBar:  newCompileProgress(48),
		currentTape: tape.Tape{Title: "Art Director", Sources: sources},
		compileRows: rows,
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 35, Succeeded: 29, Failed: 6},
			Warnings:    warnings,
		},
	}
	m.help.SetWidth(width - 4)

	rendered := m.View().Content
	view := stripANSICodesForTest(rendered)
	if got := lipgloss.Height(rendered); got > height {
		t.Fatalf("compile view should fit terminal height: got %d, want <= %d\n%s", got, height, view)
	}
	for _, expected := range []string{
		"Press i to install JS rendering",
		"i install JS",
		"↑/↓ issue",
		"This page needs browser",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"<!DOCTYPE", "liner setup-js"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("compile view should hide raw warning detail %q:\n%s", unexpected, view)
		}
	}
}

func TestCompileJSSetupRunningShowsWaitState(t *testing.T) {
	m := Model{
		screen:         screenCompile,
		width:          118,
		height:         42,
		compileBar:     newCompileProgress(48),
		jsSetupRunning: true,
		researchSpin:   newLoadingSpinner(),
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 1, Succeeded: 0, Failed: 1},
			Warnings: []core.CompileWarningPayload{{
				URL:      "https://example.test/app",
				Message:  "render: js needs Playwright. Run: liner setup-js",
				Severity: "error",
			}},
		},
	}

	rawView := m.viewCompile()
	assertTitleLineHasLoader(t, rawView, "Compile Console")
	view := stripANSICodesForTest(rawView)
	for _, expected := range []string{"Installing JS rendering", "Wait for setup to finish", "browser setup"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("running compile JS setup view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Press i to install JS rendering") {
		t.Fatalf("running compile JS setup view should not advertise install action:\n%s", view)
	}
	if hasHelp(m.helpForScreen().ShortHelp(), "i") {
		t.Fatalf("running compile JS setup help should not offer install JS: %#v", m.helpForScreen().ShortHelp())
	}
}

func TestCompileHTTP404WarningDoesNotSuggestJSSetup(t *testing.T) {
	warning := core.CompileWarningPayload{
		URL:      "https://danmall.com/posts/stealing-your-way-to-original-designs/",
		Message:  "Failed to fetch https://danmall.com/posts/stealing-your-way-to-original-designs/ — category: not_found; status: HTTP 404",
		Severity: "error",
	}
	m := Model{
		screen:     screenCompile,
		width:      118,
		height:     42,
		compileBar: newCompileProgress(48),
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 1, Succeeded: 0, Failed: 1},
			Warnings:    []core.CompileWarningPayload{warning},
		},
	}

	if m.compileNeedsJSSetup() {
		t.Fatal("HTTP 404 warning should not trigger JS setup")
	}
	if strings.Contains(stripANSICodesForTest(m.viewCompile()), "JS rendering needed") {
		t.Fatalf("404 compile view should not show JS setup cue:\n%s", m.viewCompile())
	}
	if hasHelp(m.helpForScreen().ShortHelp(), "i") {
		t.Fatalf("404 compile help should not offer JS install: %#v", m.helpForScreen().ShortHelp())
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "a") {
		t.Fatalf("404 compile help should offer add sources: %#v", m.helpForScreen().ShortHelp())
	}
	if got := compileWarningRecommendation(warning); strings.Contains(got, "Install JS rendering") {
		t.Fatalf("404 recommendation should not suggest JS setup: %q", got)
	}
	if got := m.nextAction(); got != "Open the source with o, drop it with d, or add a replacement with a." {
		t.Fatalf("404 next action should explain repair controls, got %q", got)
	}
	fullView := stripANSICodesForTest(m.View().Content)
	if got := strings.Count(fullView, "> Next:"); got != 1 {
		t.Fatalf("compile warning view should render exactly one next cue, got %d:\n%s", got, fullView)
	}
	for _, unexpected := range []string{
		"Fix the compile notes",
		"Preview MIXTAPE.md, or review the issue before trusting that source.",
	} {
		if strings.Contains(fullView, unexpected) {
			t.Fatalf("compile warning view should not show stale cue %q:\n%s", unexpected, fullView)
		}
	}
}

func TestNewStartsOnboardingWhenCompletionMarkerMissing(t *testing.T) {
	home := t.TempDir()
	baseDir := filepath.Join(home, "liner", "projects")
	fakeCore := filepath.Join(home, "liner-core")
	if err := os.WriteFile(fakeCore, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_BIN", fakeCore)
	t.Setenv("LINER_DIR", "")

	m, err := New(Options{BaseDir: baseDir})
	if err != nil {
		t.Fatal(err)
	}
	if m.screen != screenOnboarding {
		t.Fatalf("expected first launch onboarding, got %v", m.screen)
	}
	if m.baseDir != baseDir {
		t.Fatalf("expected supplied project library, got %q want %q", m.baseDir, baseDir)
	}
}

func TestNewSkipsOnboardingAfterCompletionMarker(t *testing.T) {
	home := t.TempDir()
	baseDir := filepath.Join(home, "liner", "projects")
	fakeCore := filepath.Join(home, "liner-core")
	if err := os.WriteFile(fakeCore, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("projects_dir: "+baseDir+"\nonboarding_completed: true\njsSetupPrompted: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_BIN", fakeCore)
	t.Setenv("LINER_DIR", "")

	m, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if m.screen != screenHome {
		t.Fatalf("expected completed onboarding to open Home, got %v", m.screen)
	}
	if m.baseDir != baseDir {
		t.Fatalf("expected configured project library, got %q want %q", m.baseDir, baseDir)
	}
}

func TestNewStartsJSSetupWhenPromptMissingAfterCompletion(t *testing.T) {
	home := t.TempDir()
	baseDir := filepath.Join(home, "liner", "projects")
	fakeCore := filepath.Join(home, "liner-core")
	if err := os.WriteFile(fakeCore, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("projects_dir: "+baseDir+"\nonboarding_completed: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_BIN", fakeCore)
	t.Setenv("LINER_DIR", "")

	m, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if m.screen != screenOnboarding || m.onboardingStep != onboardingStepJS {
		t.Fatalf("expected completed config without JS prompt to open JS setup, got screen=%v step=%d", m.screen, m.onboardingStep)
	}
}

func TestSettingsCanRestartOnboarding(t *testing.T) {
	home := t.TempDir()
	savedDir := filepath.Join(home, "liner", "projects")
	configPath := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("projects_dir: "+savedDir+"\nonboarding_completed: true\nagent: codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	m := Model{
		screen:   screenSettings,
		width:    118,
		baseDir:  filepath.Join(home, "old", "projects"),
		commands: newCommandList(70, 16),
	}
	m.commands.SetItems(m.commandItems())
	if !hasCommandTitle(m.commands.Items(), "Settings") {
		t.Fatalf("home commands should expose Settings")
	}

	got, _ := m.startSettings().handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: 's', Text: "s"}))
	if got.screen != screenOnboarding || got.onboardingStep != onboardingStepLibrary {
		t.Fatalf("expected setup to restart onboarding, got screen=%v step=%d", got.screen, got.onboardingStep)
	}
	if got.baseDir != savedDir {
		t.Fatalf("expected setup to use saved library, got %q want %q", got.baseDir, savedDir)
	}
	if !hasHelp(got.helpForScreen().ShortHelp(), "esc") {
		t.Fatalf("restarted onboarding should offer an esc back route: %#v", got.helpForScreen().ShortHelp())
	}
}

func TestSettingsViewFitsNarrowWidth(t *testing.T) {
	width := 80
	m := Model{
		screen: screenSettings,
		width:  width,
		settings: settingsInfo{
			ConfigPath: "/tmp/.liner/config.yaml",
			Installed:  []string{"claude", "codex"},
		},
	}

	view := m.viewSettings()
	assertNoBoxCorners(t, view)
	assertViewLinesFit(t, view, styles.ClampWidth(width-4))
}

func TestSettingsBannerIsAppLevel(t *testing.T) {
	m := Model{
		screen:      screenSettings,
		width:       118,
		currentTape: tape.Tape{Title: "iOS app store launch", Sources: []tape.Source{{Type: "web", URL: "https://example.com"}}},
	}

	banner := m.viewBanner()
	if !strings.Contains(banner, "settings") {
		t.Fatalf("settings banner should show app-level location:\n%s", banner)
	}
	for _, unexpected := range []string{"iOS app store launch", "sources"} {
		if strings.Contains(banner, unexpected) {
			t.Fatalf("settings banner should not show project metadata %q:\n%s", unexpected, banner)
		}
	}
}

func TestNonTextScreensAdvertiseHomeAndBack(t *testing.T) {
	for _, screen := range []screen{
		screenProjects,
		screenProject,
		screenSourceReview,
		screenAssemblyReview,
		screenLinerReview,
		screenSkills,
		screenSkillReview,
		screenAudits,
		screenContradictionCleanupReview,
		screenSourceNoteCleanupReview,
		screenEvals,
		screenComposition,
		screenCompositionReview,
		screenReport,
		screenBoard,
		screenCompile,
		screenPreview,
		screenImport,
		screenSettings,
	} {
		m := Model{screen: screen}
		help := m.helpForScreen().ShortHelp()
		if !hasHelp(help, "h") {
			t.Fatalf("screen %v short help should include h home, got %#v", screen, help)
		}
		if !hasHelp(help, "esc") {
			t.Fatalf("screen %v short help should include esc back, got %#v", screen, help)
		}
	}
}

func TestTextEntryScreensDoNotAdvertiseHomeShortcut(t *testing.T) {
	for _, screen := range []screen{screenSources, screenCreate, screenClarify} {
		m := Model{screen: screen}
		help := m.helpForScreen().ShortHelp()
		if hasHelp(help, "h") {
			t.Fatalf("text-entry screen %v should not advertise h home, got %#v", screen, help)
		}
	}
}

func TestSourceBoardEscReturnsToProject(t *testing.T) {
	m := Model{screen: screenBoard}
	got, _ := m.handleBoardKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if got.screen != screenProject {
		t.Fatalf("expected esc on Review Sources to return to Project, got %v", got.screen)
	}
}

func TestProviderPreferencesListSavesSelectedProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "/tmp/fake-claude")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	if err := os.MkdirAll(filepath.Join(home, ".liner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".liner", "config.yaml"), []byte("agent: claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{screen: screenProject, width: 100}.startSettings()

	got, _ := m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if view := got.viewSettings(); !strings.Contains(view, "Press Enter to switch runners") {
		t.Fatalf("inactive installed provider should invite switching:\n%s", view)
	}
	got, _ = got.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.err != "" {
		t.Fatalf("unexpected settings error: %s", got.err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".liner", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "agent: codex") {
		t.Fatalf("expected provider preference to save codex:\n%s", string(data))
	}
}

func TestProviderPreferencesDoesNotSaveUninstalledProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CODEX_BIN", "/tmp/fake-codex")
	configPath := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("agent: codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{screen: screenProject, width: 100}.startSettings()

	got, _ := m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if view := got.viewSettings(); !strings.Contains(view, "Claude Code is not installed. Install the CLI version of Claude Code to use it here.") {
		t.Fatalf("uninstalled provider should show install guidance:\n%s", view)
	}
	got, _ = got.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if !strings.Contains(got.err, "Claude Code is not installed") {
		t.Fatalf("expected uninstalled provider warning, got %q", got.err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "agent: claude") {
		t.Fatalf("should not save an uninstalled provider:\n%s", string(data))
	}
}

func TestSettingsViewHasNoNextCue(t *testing.T) {
	m := Model{
		screen: screenSettings,
		width:  118,
		settings: settingsInfo{
			ConfigPath: "/tmp/.liner/config.yaml",
			Installed:  []string{"codex"},
		},
	}

	view := m.View().Content
	if strings.Contains(view, "Next:") {
		t.Fatalf("settings should not render Next:\n%s", view)
	}
}

func TestLabelValueBlockPatternRendersWithoutTableHeader(t *testing.T) {
	view := renderLabelValueBlock(48, []labelValueRow{
		{Label: "Status", Value: "Active"},
		{Label: "Runner", Value: "Claude Code CLI."},
	}, 2, 1)

	for _, expected := range []string{"Status:", "Active", "Runner:", "Claude Code CLI"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("label/value block missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Field", "Value"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("label/value block should not render table header %q:\n%s", unexpected, view)
		}
	}
	if !strings.HasPrefix(view, "\n\n") || !strings.HasSuffix(view, "\n") {
		t.Fatalf("label/value block should preserve breathing room:\n%q", view)
	}
	assertViewLinesFit(t, view, 48)
}

func TestChoiceSelectorPatternRendersInactiveMutedAndDetailMuted(t *testing.T) {
	options := []choiceOption{
		{Label: "Install", Detail: "Download Playwright Chromium."},
		{Label: "Skip", Detail: "Skip for now."},
	}
	view := renderChoiceSelector(options, 0) + renderChoiceDetail(48, options, 0)

	for _, expected := range []string{
		styles.AccentText.Render("Install"),
		styles.MutedText.Render("Skip"),
		styles.Subtitle.Width(48).Render("Download Playwright Chromium."),
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("choice selector pattern missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Install:", "Skip:", "Field", "Value"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("choice selector pattern should not render %q:\n%s", unexpected, view)
		}
	}
	assertViewLinesFit(t, view, 48)
}

func TestDataTableHeadersAlignWithCells(t *testing.T) {
	view := newDataTable(
		[]table.Column{
			{Title: "Phase", Width: 18},
			{Title: "Status", Width: 10},
			{Title: "Evidence", Width: 24},
		},
		[]table.Row{{"Framing", "done", "working/01.md"}},
		80,
		2,
		false,
	).View()
	plain := stripANSICodesForTest(view)
	lines := strings.Split(plain, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header, separator, and row lines:\n%s", plain)
	}
	header := lines[0]
	row := lines[2]
	for _, pair := range []struct {
		header string
		value  string
	}{
		{header: "Phase", value: "Framing"},
		{header: "Status", value: "done"},
		{header: "Evidence", value: "working/01.md"},
	} {
		if got, want := strings.Index(row, pair.value), strings.Index(header, pair.header); got != want {
			t.Fatalf("%s header and %s value should align at the same column, got %d want %d:\n%s", pair.header, pair.value, got, want, plain)
		}
	}
}

func TestVisibleDataTableKeepsCursorInViewportAfterHeightClamp(t *testing.T) {
	rows := make([]table.Row, 0, 20)
	for index := 0; index < 20; index++ {
		rows = append(rows, table.Row{fmt.Sprintf("row-%02d", index)})
	}

	view := newVisibleDataTable(
		[]table.Column{{Title: "Name", Width: 12}},
		rows,
		24,
		5,
		true,
		12,
	).View()

	if !strings.Contains(stripANSICodesForTest(view), "row-12") {
		t.Fatalf("selected row should stay visible after table height is clamped:\n%s", view)
	}
}

func TestColonDoesNotNavigateToCommandSurface(t *testing.T) {
	m := Model{screen: screenProject, currentPath: t.TempDir()}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: ':', Text: ":"}))
	if got.screen != screenProject {
		t.Fatalf("colon should not navigate away from project, got %v", got.screen)
	}
}

func TestCommandPaletteCanClearRunEstimateHistory(t *testing.T) {
	historyPath := filepath.Join(t.TempDir(), "run-estimates.jsonl")
	t.Setenv("LINER_ESTIMATE_HISTORY", historyPath)
	writeGlobalEstimateHistory(t, historyPath, []globalEstimateEntry{
		{Version: 1, Phase: "framing", Tokens: 1000, Source: "seed", RecordedAt: "2026-06-14T00:00:00Z"},
	})
	project := t.TempDir()
	m := Model{
		screen:      screenProject,
		width:       100,
		currentPath: project,
		currentTape: tape.Tape{Title: "Launch", JTBD: stringPointer("Estimate a run.")},
	}

	if !hasCommandTitle(m.commandItems(), "Reset cost estimates") {
		t.Fatal("expected Home commands to expose estimate-history clearing when global history exists")
	}
	if strings.Contains(m.viewProject(), "Reset cost estimates") {
		t.Fatalf("estimate-history clearing should stay out of the Project body:\n%s", m.viewProject())
	}
	var clear commandItem
	for _, item := range m.commandItems() {
		candidate, ok := item.(commandItem)
		if ok && candidate.title == "Reset cost estimates" {
			clear = candidate
			break
		}
	}
	if clear.title == "" {
		t.Fatal("missing clear command item")
	}

	got, _ := clear.run(m)
	if got.err != "" {
		t.Fatalf("clear command should not fail, got %q", got.err)
	}
	if !strings.Contains(got.note, "Cleared 1 global run estimate sample") {
		t.Fatalf("expected clear note, got %q", got.note)
	}
	if _, err := os.Stat(historyPath); !os.IsNotExist(err) {
		t.Fatalf("expected history file removed, stat err=%v", err)
	}
	if hasCommandTitle(got.commandItems(), "Reset cost estimates") {
		t.Fatal("clear command should disappear after history is removed")
	}
}

func TestCommandListUsesLinearListWithoutOuterBox(t *testing.T) {
	width := 80
	commands := newCommandList(68, 14)
	commands.SetItems([]list.Item{
		commandItem{title: "Add sources", desc: "Paste URLs, files, articles, and local documents"},
		commandItem{title: "Preview MIXTAPE.md", desc: "Render the compiled mixtape"},
	})
	m := Model{
		screen:   screenHome,
		width:    width,
		commands: commands,
	}

	view := m.viewCommands()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Add sources", "Preview MIXTAPE.md"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("command list missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"System", "↑/k up", "j down", "/ filter"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("command list should leave navigation help to the footer, found %q:\n%s", unexpected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), min(chromeWidth(width), max(60, width-8)); got > want {
			t.Fatalf("command list line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestCommandListSelectionOnlyColorsTitle(t *testing.T) {
	commands := newCommandList(68, 14)
	commands.SetItems([]list.Item{
		commandItem{title: "Add sources", desc: "Paste URLs, files, articles, and local documents"},
		commandItem{title: "Preview MIXTAPE.md", desc: "Render the compiled mixtape"},
	})
	commands.Select(0)

	view := commands.View()
	titleLine := lineContaining(t, view, "Add sources")
	descLine := lineContaining(t, view, "Paste URLs, files, articles, and local documents")

	if !strings.Contains(titleLine, "38;2;255;90;31") {
		t.Fatalf("selected title should use orange accent:\n%s", titleLine)
	}
	if strings.Contains(descLine, "38;2;255;90;31") {
		t.Fatalf("selected description should stay grey, not orange:\n%s", descLine)
	}
	if !strings.Contains(descLine, "38;2;139;140;137") {
		t.Fatalf("selected description should use muted grey:\n%s", descLine)
	}
}

func TestHomeViewUsesOnlyFooterNavigation(t *testing.T) {
	width := 100
	m := Model{
		screen:   screenHome,
		width:    width,
		help:     help.New(),
		commands: newCommandList(70, 18),
	}
	m.commands.SetItems(m.commandItems())

	view := m.View().Content
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"liner", "home", "New Liner Project", "enter", "run", "/", "filter"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("home view missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"System", "↑/k up", "j down", "• / filter", "Keys"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("home view should keep navigation in the footer only, found %q:\n%s", unexpected, view)
		}
	}
}

func TestSourceBoardIsAdvancedCommandWhenSourcesExist(t *testing.T) {
	project := t.TempDir()
	missing := Model{
		screen:      screenProject,
		currentPath: project,
		currentTape: tape.Tape{Title: "Empty"},
	}
	if hasCommandTitle(missing.commandItems(), "Review Sources") {
		t.Fatal("Review Sources should stay hidden until there are sources to review")
	}
	available := Model{
		screen:      screenProject,
		currentPath: project,
		currentTape: tape.Tape{
			Title: "Launch",
			Sources: []tape.Source{{
				Type: "web",
				URL:  "https://example.com/research",
			}},
		},
	}
	if !hasCommandTitle(available.commandItems(), "Review Sources") {
		t.Fatal("Review Sources should be available as an advanced command when sources exist")
	}
	view := available.viewProject()
	if strings.Contains(view, "Review Sources") {
		t.Fatalf("Review Sources should not be in the Project body:\n%s", view)
	}
	var board commandItem
	for _, item := range available.commandItems() {
		if command, ok := item.(commandItem); ok && command.title == "Review Sources" {
			board = command
			break
		}
	}
	got, _ := board.run(available)
	if got.screen != screenBoard {
		t.Fatalf("expected Review Sources command to open board, got %v", got.screen)
	}
	if len(got.sourceItems) != 1 || got.sourceItems[0].Label == "" {
		t.Fatalf("expected Review Sources command to stage tape sources, got %#v", got.sourceItems)
	}
	if !strings.Contains(got.note, "advanced source table") {
		t.Fatalf("expected advanced source-table note, got %q", got.note)
	}
}

func TestAddSourcesUsesLinearInputWithoutBox(t *testing.T) {
	width := 80
	input := textinput.New()
	input.Placeholder = "Paste one URL, article, file path, repo, or local document..."
	input.SetWidth(sourceInputWidth(width))
	input.Focus()
	m := Model{
		screen:      screenSources,
		width:       width,
		currentPath: t.TempDir(),
		sourceInput: input,
	}

	view := m.viewSources()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Add Sources", "Source", "Added sources", "No pending source", "Actions", "Add pending source", "Finish source entry"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("add sources view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Press f when done") {
		t.Fatalf("add sources view should use an action table instead of the old shortcut sentence:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("add sources line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestAddSourcesLocalFolderControlsWaitUntilFolderExists(t *testing.T) {
	width := 140
	project := t.TempDir()
	input := textinput.New()
	input.SetWidth(sourceInputWidth(width))
	m := Model{
		screen:      screenSources,
		width:       width,
		currentPath: project,
		sourceInput: input,
	}

	view := m.viewSources()
	if !strings.Contains(view, "Local folder will be created after the first saved source.") {
		t.Fatalf("add sources view should explain that the folder is created later:\n%s", view)
	}
	if strings.Contains(view, "Local folder:") {
		t.Fatalf("add sources view should not show a missing folder path:\n%s", view)
	}
	if strings.Contains(view, "Open local-sources") {
		t.Fatalf("add sources view should not show open-folder action before the folder exists:\n%s", view)
	}
	if hasHelp(m.helpForScreen().ShortHelp(), "ctrl+o") {
		t.Fatal("add sources help should hide ctrl+o until local-sources exists")
	}

	got, cmd := m.handleSourceKey(tea.KeyPressMsg(tea.Key{Code: 'o', Mod: tea.ModCtrl}))
	if cmd != nil {
		t.Fatal("ctrl+o should not open a missing local-sources folder")
	}
	if got.err != "No local-sources folder yet. Add a source first." {
		t.Fatalf("expected missing-folder error, got %q", got.err)
	}

	if err := os.MkdirAll(filepath.Join(project, "local-sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	view = m.viewSources()
	if !strings.Contains(view, "Local folder:") || !strings.Contains(view, "local-sources") {
		t.Fatalf("add sources view should show the existing local folder:\n%s", view)
	}
	if !strings.Contains(view, "Open local-sources") {
		t.Fatalf("add sources view should show open-folder action once local-sources exists:\n%s", view)
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "ctrl+o") {
		t.Fatal("add sources help should show ctrl+o once local-sources exists")
	}
	got, cmd = m.handleSourceKey(tea.KeyPressMsg(tea.Key{Code: 'o', Mod: tea.ModCtrl}))
	if cmd == nil {
		t.Fatal("ctrl+o should open the existing local-sources folder")
	}
	if got.err != "" {
		t.Fatalf("expected no error when local-sources exists, got %q", got.err)
	}
}

func TestAddSourcesShowsScrollableStagedListActions(t *testing.T) {
	width := 118
	input := textinput.New()
	input.SetWidth(sourceInputWidth(width))
	input.Focus()
	m := Model{
		screen:      screenSources,
		width:       width,
		height:      34,
		currentPath: t.TempDir(),
		sourceInput: input,
		sourceTable: newSourceTable(styles.ClampWidth(width-8), 8),
	}
	sources := make([]tape.Source, 0, 8)
	for i := 1; i <= 8; i++ {
		sources = append(sources, tape.Source{Type: "web", URL: fmt.Sprintf("https://example.com/source-%02d", i)})
	}
	m.applySourceItems(source.Stage(sources, true))

	view := m.viewSources()
	for _, expected := range []string{"8 total", "8 active", "selected 1 of 8", "https://example.com/source-08", "Toggle selected source", "Remove selected source", "> ✓"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("add sources staged list missing %q:\n%s", expected, view)
		}
	}
	for _, keyName := range []string{"↑/↓", "space", "d"} {
		if !hasHelp(m.helpForScreen().ShortHelp(), keyName) {
			t.Fatalf("add sources help should include %q for staged list, got %#v", keyName, m.helpForScreen().ShortHelp())
		}
	}

	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	got := next.(Model)
	if got.sourceTable.Cursor() != 1 {
		t.Fatalf("down should select the second staged source, got cursor %d", got.sourceTable.Cursor())
	}
	got, _ = got.handleSourceKey(tea.KeyPressMsg(tea.Key{Code: ' ', Text: " "}))
	if got.sourceItems[1].Active {
		t.Fatalf("space should deactivate the selected staged source")
	}
	next, _ = got.Update(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	got = next.(Model)
	if len(got.sourceItems) != 7 {
		t.Fatalf("d should remove one staged source, got %#v", got.sourceItems)
	}
	if strings.Contains(got.sourceInput.Value(), "d") {
		t.Fatalf("remove key should not type into the empty paste input, got %q", got.sourceInput.Value())
	}
	for _, item := range got.sourceItems {
		if item.Label == "https://example.com/source-02" {
			t.Fatalf("selected source should have been removed: %#v", got.sourceItems)
		}
	}
}

func TestAddSourcesListActionsDoNotStealTypedInput(t *testing.T) {
	width := 100
	input := textinput.New()
	input.SetWidth(sourceInputWidth(width))
	input.SetValue("https://example.com/")
	input.Focus()
	m := Model{
		screen:      screenSources,
		width:       width,
		height:      30,
		currentPath: t.TempDir(),
		sourceInput: input,
		sourceTable: newSourceTable(styles.ClampWidth(width-8), 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{{Type: "web", URL: "https://example.com/source"}}, true))

	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	got := next.(Model)
	if len(got.sourceItems) != 1 {
		t.Fatalf("typing d with pending input should not remove staged sources: %#v", got.sourceItems)
	}
	if !strings.Contains(got.sourceInput.Value(), "d") {
		t.Fatalf("typing d with pending input should edit the paste bar, got %q", got.sourceInput.Value())
	}
	if hasHelp(got.helpForScreen().ShortHelp(), "d") {
		t.Fatalf("remove help should be hidden while pending input is present")
	}
}

func TestSourceReviewUsesPlainTableWithoutOuterBox(t *testing.T) {
	m := Model{
		screen:      screenSourceReview,
		width:       118,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(100, 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{
		{Type: "web", URL: "https://developer.apple.com/design/human-interface-guidelines/"},
	}, true))

	view := m.viewSourceReview()
	assertNoBoxCorners(t, view)
	if !strings.Contains(view, "Saved as") {
		t.Fatalf("review view should expose table headers:\n%s", view)
	}
	for _, expected := range []string{"Actions", "Save active sources", "Toggle selected source", "Remove selected source", "Add more sources"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("review view should show action table item %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Space toggles") {
		t.Fatalf("review view should not explain shortcuts in prose:\n%s", view)
	}
}

func TestSourceReviewShowsSelectedSourceDetail(t *testing.T) {
	width := 80
	m := Model{
		screen:      screenSourceReview,
		width:       width,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(styles.ClampWidth(width-8), 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{
		{Type: "web", URL: "https://example.com/research/with/a/very/long/path/that/needs/context"},
	}, true))

	view := m.viewSourceReview()
	for _, expected := range []string{"Selected", "Field", "Value", "Source", "Type", "web", "Status", "active", "Saved as", "fetch on compile", "Kind", "unspecified"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("source review selected detail missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Type: web  Status: active") {
		t.Fatalf("source review should use metadata rows instead of prose detail:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("source review line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestSourceReviewMarksAndTogglesSelectedRow(t *testing.T) {
	m := Model{
		screen:      screenSourceReview,
		width:       100,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(100, 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{
		{Type: "web", URL: "https://example.com/first"},
		{Type: "web", URL: "https://example.com/second"},
	}, true))
	m.sourceTable.SetCursor(1)

	view := m.viewSourceReview()
	if !strings.Contains(view, "> ✓") || !strings.Contains(view, "https://example.com/second") {
		t.Fatalf("source review should visibly mark the selected row:\n%s", view)
	}

	got, _ := m.handleSourceReviewKey(tea.KeyPressMsg(tea.Key{Code: ' ', Text: " "}))
	if got.sourceItems[0].Active != true || got.sourceItems[1].Active != false {
		t.Fatalf("space should toggle the selected row only: %#v", got.sourceItems)
	}
	if got.sourceTable.Cursor() != 1 {
		t.Fatalf("toggle should preserve table cursor, got %d", got.sourceTable.Cursor())
	}
}

func TestSourceReviewLocalFolderActionWaitsUntilFolderExists(t *testing.T) {
	project := t.TempDir()
	m := Model{
		screen:      screenSourceReview,
		width:       118,
		currentPath: project,
		sourceTable: newSourceTable(100, 8),
	}
	m.applySourceItems(source.Stage([]tape.Source{
		{Type: "web", URL: "https://example.com/research"},
	}, true))

	if hasHelp(m.helpForScreen().FullHelp()[0], "o") {
		t.Fatal("source review full help should hide open folder until local-sources exists")
	}
	got, cmd := m.handleSourceReviewKey(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	if cmd != nil {
		t.Fatal("o should not open a missing local-sources folder")
	}
	if got.err != "No local-sources folder yet. Add a source first." {
		t.Fatalf("expected missing-folder error, got %q", got.err)
	}

	if err := os.MkdirAll(filepath.Join(project, "local-sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !hasHelp(m.helpForScreen().FullHelp()[0], "o") {
		t.Fatal("source review full help should show open folder once local-sources exists")
	}
	got, cmd = m.handleSourceReviewKey(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	if cmd == nil {
		t.Fatal("o should open the existing local-sources folder")
	}
	if got.err != "" {
		t.Fatalf("expected no error when local-sources exists, got %q", got.err)
	}
}

func TestSourceBoardUsesTableWithoutPanels(t *testing.T) {
	m := Model{
		screen: screenBoard,
		width:  118,
		height: 32,
	}
	m.sourceItems = source.Stage([]tape.Source{
		{Type: "web", URL: "https://example.com/active"},
		{Type: "web", URL: "https://example.com/inactive"},
	}, true)
	m.sourceItems[1].Active = false
	m.sourceItems[1].Status = "needs review"

	view := m.viewBoard()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Personal Sources", "Use", "Status", "Source", "✓", "○", "active", "needs review", "Selected", "Field", "Value", "Saved as"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("source board missing %q:\n%s", expected, view)
		}
	}
}

func TestSourceBoardFitsNarrowWidth(t *testing.T) {
	width := 60
	m := Model{
		screen:     screenBoard,
		width:      width,
		height:     28,
		boardIndex: 0,
		sourceItems: []source.StagedSource{{
			ID:          "source-1",
			Type:        "local_file",
			Label:       "A very long captured research note with a descriptive title that used to stretch the source board",
			Destination: "local-sources/captured/research/a-very-long-captured-research-note-with-a-descriptive-title.md",
			Active:      true,
			Status:      "needs review",
		}},
	}

	view := m.viewBoard()
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Personal Sources", "Use", "Saved as", "Selected", "Field", "Value", "needs review"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("narrow source board missing %q:\n%s", expected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("source board line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestSourceBoardToggleUsesSelectedRow(t *testing.T) {
	m := Model{
		currentPath: t.TempDir(),
		boardIndex:  1,
	}
	m.sourceItems = source.Stage([]tape.Source{
		{Type: "web", URL: "https://example.com/first"},
		{Type: "web", URL: "https://example.com/second"},
	}, true)

	got, _ := m.handleBoardKey(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace}))
	if !got.sourceItems[0].Active {
		t.Fatal("first row should not be toggled")
	}
	if got.sourceItems[1].Active {
		t.Fatal("selected second row should be toggled inactive")
	}
}

func TestResearchViewUsesMethodologyTableWithoutPanels(t *testing.T) {
	m := Model{
		screen:        screenResearch,
		width:         100,
		height:        32,
		currentTape:   tape.Tape{Title: "Launch"},
		researchLines: []string{"Starting research...", "Loaded setup context."},
		researchSpin:  newLoadingSpinner(),
	}

	view := m.viewResearch()
	assertNoBoxCorners(t, view)
	assertTitleLineHasLoader(t, view, "Build Corpus")
	for _, expected := range []string{"Build Corpus", "Progress", "Phases", "Framing", "Candidate discovery", "Artifact", "working"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("methodology view missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Setup context") {
		t.Fatalf("methodology view should not show the old placeholder setup row:\n%s", view)
	}
}

func TestMethodologyLogUsesScrollableViewport(t *testing.T) {
	lines := make([]string, 0, 24)
	for i := 1; i <= 24; i++ {
		lines = append(lines, "Log line "+string(rune('A'+i-1)))
	}
	lines[23] = "Log line X " + strings.Repeat("very long validation detail ", 12)
	m := Model{
		screen:        screenResearch,
		width:         100,
		height:        32,
		currentTape:   tape.Tape{Title: "Launch"},
		researchLines: lines,
	}
	m.syncMethodologyLog(true)

	if got := methodologyLogHeight(m.height); got != 3 {
		t.Fatalf("methodology log should show exactly three visible rows, got %d", got)
	}
	logView := stripANSICodesForTest(m.methodologyLogViewport(40, 3).View())
	logLines := strings.Split(logView, "\n")
	if len(logLines) != 3 {
		t.Fatalf("methodology log viewport should render three physical rows, got %d:\n%s", len(logLines), logView)
	}
	for _, line := range logLines {
		if got := lipgloss.Width(line); got > 40 {
			t.Fatalf("methodology log row should be clipped to viewport width, got %d:\n%s", got, line)
		}
	}
	view := m.viewResearch()
	if !strings.Contains(view, "Log line X") {
		t.Fatalf("expected viewport to start at latest log lines:\n%s", view)
	}
	if strings.Contains(view, "Log line A") {
		t.Fatalf("expected viewport to hide earlier log lines until scrolled:\n%s", view)
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "↑/↓") {
		t.Fatal("long methodology logs should expose scroll help")
	}

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	got := updated.(Model)
	if got.researchLog.YOffset() >= m.researchLog.YOffset() {
		t.Fatalf("expected up key to scroll methodology log, before=%d after=%d", m.researchLog.YOffset(), got.researchLog.YOffset())
	}
}

func TestMethodologyLogDoesNotAutoFollowWhenUserScrolledUp(t *testing.T) {
	lines := make([]string, 0, 24)
	for i := 1; i <= 24; i++ {
		lines = append(lines, "Log line "+string(rune('A'+i-1)))
	}
	m := Model{
		screen:        screenResearch,
		width:         100,
		height:        32,
		currentTape:   tape.Tape{Title: "Launch"},
		researchLines: lines,
	}
	m.syncMethodologyLog(true)
	m.researchLog.ScrollUp(4)
	before := m.researchLog.YOffset()

	m.applyMethodologyEvent(agent.Event{Kind: "text", Text: "A newly streamed event"})

	if got := m.researchLog.YOffset(); got != before {
		t.Fatalf("new events should preserve manual scroll position, before=%d after=%d", before, got)
	}
	if !strings.Contains(m.researchLog.GetContent(), "A newly streamed event") {
		t.Fatalf("expected new event in viewport content:\n%s", m.researchLog.GetContent())
	}
}

func TestMethodologyQuietStatusAppearsAfterQuietRun(t *testing.T) {
	m := Model{
		screen:                    screenResearch,
		width:                     100,
		height:                    32,
		currentTape:               tape.Tape{Title: "Launch"},
		researchLines:             []string{"Running Evaluation with codex.", "Tool started: web_search"},
		methodologyPhaseID:        "evaluation",
		methodologyEvents:         make(chan agent.Event),
		methodologyDone:           make(chan error),
		methodologyLastEventFrame: 1,
		fxFrame:                   methodologyQuietFrameDelay + 2,
	}

	view := m.viewResearch()
	for _, expected := range []string{"Still running Evaluation", "tool calls stay quiet"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("quiet methodology view missing %q:\n%s", expected, view)
		}
	}
}

func TestMethodologyQuietStatusWaitsForDelay(t *testing.T) {
	m := Model{
		screen:                    screenResearch,
		width:                     100,
		height:                    32,
		currentTape:               tape.Tape{Title: "Launch"},
		researchLines:             []string{"Running Evaluation with codex.", "Tool started: web_search"},
		methodologyPhaseID:        "evaluation",
		methodologyEvents:         make(chan agent.Event),
		methodologyDone:           make(chan error),
		methodologyLastEventFrame: 1,
		fxFrame:                   methodologyQuietFrameDelay - 1,
	}

	if view := m.viewResearch(); strings.Contains(view, "Still running") {
		t.Fatalf("fresh methodology events should not show quiet status:\n%s", view)
	}
}

func TestMethodologyStartWithoutProjectPathDoesNotSpawnRunner(t *testing.T) {
	m := Model{
		currentTape: tape.Tape{Title: "Launch"},
	}

	got, cmd := m.startResearch()
	if cmd != nil {
		t.Fatal("methodology should not launch a runner without a project path")
	}
	if got.screen != screenResearch || !got.researchDone {
		t.Fatalf("expected methodology screen to stop safely, got screen=%v done=%v", got.screen, got.researchDone)
	}
	if !strings.Contains(got.err, "project path") {
		t.Fatalf("expected helpful project path error, got %q", got.err)
	}
}

func TestMethodologyResumeIndexSkipsPendingGate(t *testing.T) {
	project := t.TempDir()
	if _, err := linerprogress.MarkPhaseComplete(project, linerprogress.PhaseFraming); err != nil {
		t.Fatal(err)
	}
	mode := "methodology"
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch", Mode: &mode}}

	index, notes, err := m.nextMethodologyPhaseIndex()
	if err != nil {
		t.Fatal(err)
	}
	if index != 1 {
		t.Fatalf("expected methodology to continue at candidates, got index %d", index)
	}
	if !linerprogress.ReadGateState(project).Gate0Accepted {
		t.Fatal("expected pending quick-mode gate0 to be accepted")
	}
	if !strings.Contains(strings.Join(notes, "\n"), "gate0") {
		t.Fatalf("expected gate note, got %#v", notes)
	}
}

func TestMethodologyResumeRecoversCompletedEvaluationArtifact(t *testing.T) {
	project := t.TempDir()
	working := filepath.Join(project, "working")
	if err := os.MkdirAll(working, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(working, "02-candidate-longlist.md"), []byte("- https://example.com/one\n- https://example.com/two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	evaluation := `candidates:
  - url: https://example.com/one
    decision: kept
    fetch_status: readable
    content_quality: high
    evidence:
      - Useful evidence.
      - Second useful evidence.
  - url: https://example.com/two
    decision: dropped
`
	if err := os.WriteFile(filepath.Join(working, "03-evaluation.yaml"), []byte(evaluation), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseEvaluation)}); err != nil {
		t.Fatal(err)
	}
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch"}}

	index, notes, err := m.nextMethodologyPhaseIndex()
	if err != nil {
		t.Fatal(err)
	}
	if index != 3 {
		t.Fatalf("expected recovered evaluation to continue at quality, got index %d", index)
	}
	if got := linerprogress.Read(project).Step; got != linerprogress.PhaseIndex(linerprogress.PhaseQuality) {
		t.Fatalf("expected progress to advance to quality, got %d", got)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "Recovered completed Evaluation") {
		t.Fatalf("expected recovery note, got %#v", notes)
	}
}

func TestMethodologyResumeDoesNotRecoverPartialEvaluationArtifact(t *testing.T) {
	project := t.TempDir()
	working := filepath.Join(project, "working")
	if err := os.MkdirAll(working, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(working, "02-candidate-longlist.md"), []byte("- https://example.com/one\n- https://example.com/two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	evaluation := `candidates:
  - url: https://example.com/one
    decision: kept
    fetch_status: readable
    content_quality: high
    evidence:
      - Useful evidence.
`
	if err := os.WriteFile(filepath.Join(working, "03-evaluation.yaml"), []byte(evaluation), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseEvaluation)}); err != nil {
		t.Fatal(err)
	}
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch"}}

	index, notes, err := m.nextMethodologyPhaseIndex()
	if err != nil {
		t.Fatal(err)
	}
	if index != 2 {
		t.Fatalf("expected partial evaluation to stay on evaluation, got index %d", index)
	}
	if got := linerprogress.Read(project).Step; got != linerprogress.PhaseIndex(linerprogress.PhaseEvaluation) {
		t.Fatalf("partial evaluation should not advance progress, got %d", got)
	}
	if strings.Contains(strings.Join(notes, "\n"), "Recovered completed Evaluation") {
		t.Fatalf("partial evaluation should not add recovery note, got %#v", notes)
	}
}

func TestMethodologyStartResumesPhaseWithExistingRunLog(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".liner-runs", "candidates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".liner-runs", "candidates", "previous.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseCandidates)}); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "runner.cjs")
	if err := os.WriteFile(script, []byte(`const has = (flag) => process.argv.includes(flag);
const value = (flag) => process.argv[process.argv.indexOf(flag) + 1] || "";
process.stdout.write(JSON.stringify({
  kind: "runner_start",
  phaseId: value("--phase"),
  agent: "codex",
  resume: has("--resume")
}) + "\n");
`), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINER_HEADLESS_RUNNER", script)
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch"}}

	got, cmd := m.startResearch()
	if cmd == nil {
		t.Fatal("expected methodology runner command")
	}
	msg := cmd()
	eventMsg, ok := msg.(methodologyEventMsg)
	if !ok {
		t.Fatalf("expected methodology event, got %#v", msg)
	}
	if eventMsg.event.PhaseID != "candidates" || !eventMsg.event.Resume {
		t.Fatalf("expected candidates to resume, got %#v", eventMsg.event)
	}
	if !strings.Contains(strings.Join(got.researchLines, "\n"), "Resuming Candidate discovery") {
		t.Fatalf("expected resume copy in log, got %#v", got.researchLines)
	}
}

func TestMethodologyFailureViewShowsRetryCue(t *testing.T) {
	m := Model{
		screen:                screenResearch,
		width:                 100,
		height:                32,
		currentTape:           tape.Tape{Title: "Launch"},
		researchLines:         []string{"Starting Corpus Creation...", "Corpus Builder paused on error."},
		researchStep:          1,
		researchDone:          true,
		methodologyFailed:     true,
		methodologyPhaseIndex: 1,
		methodologyPhaseID:    "candidates",
	}

	view := m.viewResearch()
	for _, expected := range []string{"failed", "Retry this phase"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("methodology failure view missing %q:\n%s", expected, view)
		}
	}
}

func TestMethodologyLogCompactsRepeatedToolRuns(t *testing.T) {
	m := Model{}
	ok := true

	for i := 0; i < 3; i++ {
		m.applyMethodologyEvent(agent.Event{Kind: "tool_start", Name: "web_search"})
		m.applyMethodologyEvent(agent.Event{Kind: "tool_done", Name: "web_search", OK: &ok})
	}

	if len(m.researchLines) != 1 {
		t.Fatalf("expected one compacted log row, got %#v", m.researchLines)
	}
	if m.researchLines[0] != "Tool finished: web_search. (x3)" {
		t.Fatalf("unexpected compacted row: %#v", m.researchLines)
	}
	if m.methodologyEventCount != 6 {
		t.Fatalf("expected all events to still be counted, got %d", m.methodologyEventCount)
	}
}

func TestMethodologyRawLogHidesCosmeticCodexWarnings(t *testing.T) {
	for _, warning := range []string{
		"2026-06-14T08:14:22Z WARN codex_core_plugins::manifest: ignoring interface.defaultPrompt[0]: prompt must be at most 128 characters",
		"2026-06-14T08:14:22Z WARN codex_core_skills::loader: ignoring interface.icon_small: icon path with '..' must resolve under plugin assets/",
		"2026-06-14T08:14:22Z WARN codex_core_skills::loader: ignoring interface.icon_large: icon path with '..' must resolve under plugin assets/",
		"WARN rmcp::transport::auth: Token refresh not possible, re-authorization required.",
		"ERROR codex_core::session::session: failed to load skill /path/SKILL.md: invalid YAML",
		"ERROR rmcp::transport::worker: worker quit with fatal: Transport channel closed, when Client HTTP request (HttpRequest)",
		"ERROR rmcp::transport::worker: worker quit with fatal: Transport channel closed, when Auth (AuthorizationRequired)",
		"router: write_stdin failed: stdin is closed",
	} {
		if got := methodologyEventLine(agent.Event{Kind: "raw", Text: warning}); got != "" {
			t.Fatalf("expected cosmetic warning to be hidden, got %q", got)
		}
	}

	visible := methodologyEventLine(agent.Event{Kind: "raw", Text: "ERROR codex auth failed: run `codex login`"})
	if !strings.Contains(visible, "codex auth failed") {
		t.Fatalf("expected real Codex auth error to stay visible, got %q", visible)
	}
}

func TestMethodologyProgressAutoAcceptsQuickGate(t *testing.T) {
	project := t.TempDir()
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch"}}

	m.recordMethodologyProgress("framing")

	if got := linerprogress.Read(project).Step; got != 2 {
		t.Fatalf("expected quick-mode framing to advance through gate0, got step %d", got)
	}
	if !linerprogress.ReadGateState(project).Gate0Accepted {
		t.Fatal("expected gate0 to be auto-accepted in quick mode")
	}
}

func TestMethodologyProgressAcceptsGateInMethodologyMode(t *testing.T) {
	project := t.TempDir()
	mode := "methodology"
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch", Mode: &mode}}

	m.recordMethodologyProgress("framing")

	if got := linerprogress.Read(project).Step; got != 2 {
		t.Fatalf("expected methodology framing to advance through gate0, got step %d", got)
	}
	if !linerprogress.ReadGateState(project).Gate0Accepted {
		t.Fatal("expected gate0 to be accepted in the continuous Go run")
	}
	if !strings.Contains(strings.Join(m.researchLines, "\n"), "continuous corpus build") {
		t.Fatalf("expected continuous-run gate note, got %#v", m.researchLines)
	}
}

func TestMethodologyProgressWaitsForAssemblyDraftAcceptance(t *testing.T) {
	project := t.TempDir()
	m := Model{currentPath: project, currentTape: tape.Tape{Title: "Launch"}}

	m.recordMethodologyProgress("assembly")

	if got := linerprogress.Read(project).Step; got != 0 {
		t.Fatalf("assembly should not advance before draft acceptance, got step %d", got)
	}
	if !strings.Contains(strings.Join(m.researchLines, "\n"), "draft review acceptance") {
		t.Fatalf("expected assembly wait note, got %#v", m.researchLines)
	}
}

func TestAssemblyReviewMissingDraftStopsWithHelpfulError(t *testing.T) {
	project := t.TempDir()
	m := Model{currentPath: project}

	got, cmd := m.startAssemblyReview()
	if cmd != nil {
		t.Fatal("missing draft should not return a command")
	}
	if got.screen == screenAssemblyReview {
		t.Fatalf("missing draft should not enter review screen")
	}
	if !strings.Contains(got.err, assemblyDraftRelPath) {
		t.Fatalf("expected draft path in error, got %q", got.err)
	}
}

func TestAcceptAssemblyDraftWritesTapeAndProgress(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	current := tape.Tape{
		Title:       "Launch",
		Description: "Demo",
		Curator:     "Arturo",
		Sources: []tape.Source{
			{Type: "web", URL: "https://old.example.com", Priority: "required"},
		},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	draft := []byte(`sources:
  - type: web
    url: https://new.example.com
    priority: required
    section: foundations
`)
	if err := os.WriteFile(filepath.Join(project, assemblyDraftRelPath), draft, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseAssembly)}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		currentPath: project,
		currentTape: current,
		sourceTable: newSourceTable(100, 8),
		compileBar:  newCompileProgress(48),
	}

	got, _ := m.startAssemblyReview()
	if got.screen != screenAssemblyReview {
		t.Fatalf("expected assembly review screen, got %v: %s", got.screen, got.err)
	}
	got, _ = got.acceptAssemblyDraft()

	updated, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Sources) != 1 || updated.Sources[0].URL != "https://new.example.com" {
		t.Fatalf("draft sources were not written to tape: %#v", updated.Sources)
	}
	if got := linerprogress.Read(project).Step; got != linerprogress.PhaseIndex(linerprogress.PhaseCompile) {
		t.Fatalf("expected progress to advance to compile, got %d", got)
	}
	if _, err := os.Stat(filepath.Join(project, assemblyDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected draft to be removed, stat err=%v", err)
	}
}

func TestAssemblyReviewMarksAndTogglesSelectedDraftSource(t *testing.T) {
	reference := "reference"
	example := "example"
	m := Model{
		screen:      screenAssemblyReview,
		width:       100,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(100, 8),
	}
	m.sourceItems = source.Stage([]tape.Source{
		{Type: "web", URL: "https://example.com/first", Kind: &reference},
		{Type: "web", URL: "https://example.com/second", Kind: &example},
	}, true)
	m.applySourceItems(m.sourceItems)
	m.sourceTable.SetCursor(1)

	view := m.viewAssemblyReview()
	if !strings.Contains(view, "> ✓") || !strings.Contains(view, "https://example.com/second") {
		t.Fatalf("assembly review should visibly mark the selected row:\n%s", view)
	}

	got, _ := m.handleAssemblyReviewKey(tea.KeyPressMsg(tea.Key{Code: ' ', Text: " "}))
	if got.sourceItems[0].Active != true || got.sourceItems[1].Active != false {
		t.Fatalf("space should toggle the selected draft source only: %#v", got.sourceItems)
	}
	if got.sourceTable.Cursor() != 1 {
		t.Fatalf("toggle should preserve table cursor, got %d", got.sourceTable.Cursor())
	}
}

func TestAssemblyReviewKeepsSelectedDraftSourceVisibleWhenScrolled(t *testing.T) {
	m := Model{
		screen:      screenAssemblyReview,
		width:       118,
		height:      34,
		currentPath: t.TempDir(),
		sourceTable: newSourceTable(100, 8),
	}
	sources := make([]tape.Source, 0, 14)
	for index := 0; index < 14; index++ {
		sources = append(sources, tape.Source{
			Type: "web",
			URL:  fmt.Sprintf("https://example.com/source-%02d", index),
		})
	}
	m.sourceItems = source.Stage(sources, true)
	m.applySourceItems(m.sourceItems)
	m.sourceTable.SetCursor(10)

	view := m.viewAssemblyReview()
	selectedLine := lineContaining(t, view, "source-10")
	if !strings.Contains(selectedLine, "> ✓") {
		t.Fatalf("selected draft source should stay visible in the table, got line %q:\n%s", selectedLine, view)
	}
}

func TestAcceptAssemblyDraftWritesLocalFileWithoutEmptyURL(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	current := tape.Tape{
		Title:       "Local Smoke",
		Description: "Demo",
		Curator:     "Arturo",
		Sources:     []tape.Source{},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	draft := []byte(`sources:
  - type: local_file
    path: local-sources/note.md
    citation: Local acceptance fixture note
    priority: required
    section: local evidence
    kind: example
    note: Deterministic local-file source.
`)
	if err := os.WriteFile(filepath.Join(project, assemblyDraftRelPath), draft, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseAssembly)}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		currentPath: project,
		currentTape: current,
		sourceTable: newSourceTable(100, 8),
		compileBar:  newCompileProgress(48),
	}

	got, _ := m.startAssemblyReview()
	if got.screen != screenAssemblyReview {
		t.Fatalf("expected assembly review screen, got %v: %s", got.screen, got.err)
	}
	got, _ = got.acceptAssemblyDraft()

	updated, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Sources) != 1 {
		t.Fatalf("expected one local source, got %#v", updated.Sources)
	}
	if updated.Sources[0].Type != "local_file" || updated.Sources[0].Path == nil || *updated.Sources[0].Path != "local-sources/note.md" {
		t.Fatalf("local_file draft source was not preserved: %#v", updated.Sources[0])
	}
	raw, err := os.ReadFile(filepath.Join(project, "tape.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "url:") {
		t.Fatalf("accepted local_file source should not write an empty url field:\n%s", raw)
	}
}

func TestDiscardAssemblyDraftRemovesDraftWithoutChangingTape(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	current := tape.Tape{
		Title: "Launch",
		Sources: []tape.Source{
			{Type: "web", URL: "https://old.example.com", Priority: "required"},
		},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	draft := []byte(`sources:
  - type: web
    url: https://new.example.com
`)
	if err := os.WriteFile(filepath.Join(project, assemblyDraftRelPath), draft, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseAssembly)}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		currentPath:  project,
		currentTape:  current,
		sourceTable:  newSourceTable(100, 8),
		researchDone: true,
	}

	got, _ := m.startAssemblyReview()
	got, _ = got.discardAssemblyDraft()

	updated, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Sources) != 1 || updated.Sources[0].URL != "https://old.example.com" {
		t.Fatalf("discard changed tape sources: %#v", updated.Sources)
	}
	if got := linerprogress.Read(project).Step; got != linerprogress.PhaseIndex(linerprogress.PhaseAssembly) {
		t.Fatalf("discard should not advance progress, got step %d", got)
	}
	if got.screen != screenProject {
		t.Fatalf("discard should return to project, got %v", got.screen)
	}
	if _, err := os.Stat(filepath.Join(project, assemblyDraftRelPath)); !os.IsNotExist(err) {
		t.Fatalf("expected draft to be removed, stat err=%v", err)
	}
}

func TestAssemblyReviewKeepsActionsInFooterOnly(t *testing.T) {
	note := "Canonical source. Use first."
	section := "foundations"
	kind := "canonical"
	m := Model{
		screen:      screenAssemblyReview,
		width:       100,
		sourceTable: newSourceTable(100, 8),
	}
	m.sourceItems = source.Stage([]tape.Source{{
		Type:    "web",
		URL:     "https://example.com",
		Note:    &note,
		Section: &section,
		Kind:    &kind,
	}}, true)
	m.applySourceItems(m.sourceItems)

	view := m.viewAssemblyReview()
	for _, expected := range []string{"Review Draft Sources", "Kind", "Section", "Note", "canonical", "foundations", "Canonical source", "Selected", "Field", "Value", "Status", "checked"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("assembly review missing %q:\n%s", expected, view)
		}
	}
	for _, unexpected := range []string{"Review note", "Assembly proposed this source", "should not enter", "Actions", "Save checked sources", "Toggle selected source", "Open draft", "Discard draft", "source list + compile", "remove draft", "Space toggles. o opens"} {
		if strings.Contains(view, unexpected) {
			t.Fatalf("assembly review body should leave action copy to the footer, found %q:\n%s", unexpected, view)
		}
	}
	help := m.helpForScreen().ShortHelp()
	for _, keyName := range []string{"enter", "space", "o", "d"} {
		if !hasHelp(help, keyName) {
			t.Fatalf("assembly review footer should include %q, got %#v", keyName, help)
		}
	}
}

func TestOpenCommandForPlatforms(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "xdg-open" {
			return "/usr/bin/xdg-open", nil
		}
		return "", os.ErrNotExist
	}

	cmd, args, err := openCommandFor("darwin", "/tmp/project", lookPath)
	if err != nil || cmd != "open" || len(args) != 1 || args[0] != "/tmp/project" {
		t.Fatalf("unexpected darwin opener: cmd=%q args=%#v err=%v", cmd, args, err)
	}

	cmd, args, err = openCommandFor("linux", "/tmp/project", lookPath)
	if err != nil || cmd != "/usr/bin/xdg-open" || len(args) != 1 || args[0] != "/tmp/project" {
		t.Fatalf("unexpected linux opener: cmd=%q args=%#v err=%v", cmd, args, err)
	}

	cmd, args, err = openCommandFor("windows", `C:\tmp\project`, lookPath)
	if err != nil || cmd != "rundll32" || len(args) != 2 || args[0] != "url.dll,FileProtocolHandler" {
		t.Fatalf("unexpected windows opener: cmd=%q args=%#v err=%v", cmd, args, err)
	}
}

func TestOpenCommandForLinuxWithoutOpenerFailsClearly(t *testing.T) {
	_, _, err := openCommandFor("linux", "/tmp/project", func(string) (string, error) {
		return "", os.ErrNotExist
	})
	if err == nil || !strings.Contains(err.Error(), "xdg-open") {
		t.Fatalf("expected missing opener guidance, got %v", err)
	}
}

func TestClipboardCommandForPlatforms(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", os.ErrNotExist
	}

	cmd, args, err := clipboardCommandFor("darwin", lookPath)
	if err != nil || cmd != "pbcopy" || len(args) != 0 {
		t.Fatalf("unexpected darwin clipboard command: cmd=%q args=%#v err=%v", cmd, args, err)
	}

	cmd, args, err = clipboardCommandFor("linux", lookPath)
	if err != nil || cmd != "/usr/bin/wl-copy" || len(args) != 0 {
		t.Fatalf("unexpected linux clipboard command: cmd=%q args=%#v err=%v", cmd, args, err)
	}

	cmd, args, err = clipboardCommandFor("windows", lookPath)
	if err != nil || cmd != "clip" || len(args) != 0 {
		t.Fatalf("unexpected windows clipboard command: cmd=%q args=%#v err=%v", cmd, args, err)
	}
}

func TestClipboardCommandForLinuxWithoutHelperFailsClearly(t *testing.T) {
	_, _, err := clipboardCommandFor("linux", func(string) (string, error) {
		return "", os.ErrNotExist
	})
	if err == nil || !strings.Contains(err.Error(), "wl-copy") {
		t.Fatalf("expected missing clipboard guidance, got %v", err)
	}
}

func TestCopyMixtapeMissingFileReportsHelpfulError(t *testing.T) {
	msg, ok := copyMixtape(t.TempDir())().(mixtapeCopiedMsg)
	if !ok {
		t.Fatalf("expected mixtapeCopiedMsg")
	}
	if msg.err == nil || !strings.Contains(msg.err.Error(), "MIXTAPE.md") {
		t.Fatalf("expected MIXTAPE.md read error, got %v", msg.err)
	}
}

func TestCompileSourcesRenderAsTable(t *testing.T) {
	m := Model{
		width:  100,
		height: 32,
		compileRows: []compileSourceRow{
			{Status: "running", Type: "web", Source: "https://example.com/really/long/source/path", Detail: "fetching"},
			{Status: "queued", Type: "local", Source: "local-sources/book.md"},
		},
	}

	view := m.viewCompileSources(styles.ClampWidth(m.width - 4))
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Sources", "Status", "working", "local-sources/book.md"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile source table missing %q:\n%s", expected, view)
		}
	}
}

func TestCompileResultUsesPlainSectionWithoutBox(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("Curated synthesis ready."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		width:       100,
		currentPath: project,
		compileResult: &core.CompileResultPayload{
			MixtapePath: filepath.Join(project, "MIXTAPE.md"),
			Summary:     core.CompileSummary{Total: 2, Succeeded: 2, Failed: 0},
		},
	}

	view := m.viewCompileResult(styles.ClampWidth(m.width - 4))
	assertNoBoxCorners(t, view)
	for _, expected := range []string{"Result", "compiled", "2/2 usable sources", "MIXTAPE.md"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile result missing %q:\n%s", expected, view)
		}
	}
}

func TestCompileResultWarningsRespectNarrowWidth(t *testing.T) {
	width := 60
	m := Model{
		width: width,
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 2, Succeeded: 1, Failed: 1},
			Warnings: []core.CompileWarningPayload{{
				URL:     "https://example.com/a/very/long/source/path/that/used/to/consume/the/warning/row",
				Message: "The fetched body was partial after several attempts and should be reviewed before relying on the mixtape.",
			}},
		},
	}

	view := m.viewCompileResult(styles.ClampWidth(width - 4))
	assertNoBoxCorners(t, view)
	for _, expected := range []string{
		"1 source issue",
		"Issue detail",
		"Message",
		"Recommendation",
		"Drop",
		"retry compile",
		"https://e",
		"arning/row",
		"The fetch",
		"partial after several",
		"attempts and should",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile warning view missing %q:\n%s", expected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("compile warning line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestCompileWarningSelectionAndHelpUsePerSourceActions(t *testing.T) {
	m := Model{
		screen: screenCompile,
		width:  100,
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 3, Succeeded: 1, Failed: 2},
			Warnings: []core.CompileWarningPayload{
				{URL: "https://example.com/first", Message: "first failed", Severity: "warning"},
				{URL: "https://example.com/second", Message: "second failed", Severity: "error"},
			},
		},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))

	if got.compileWarningIndex != 1 {
		t.Fatalf("expected selected warning index 1, got %d", got.compileWarningIndex)
	}
	view := got.viewCompileResult(styles.ClampWidth(got.width - 4))
	for _, expected := range []string{"https://example.com/second", "second failed", "Issue detail", "Recommendation"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile warning selection view missing %q:\n%s", expected, view)
		}
	}
	for _, keyName := range []string{"p", "o", "d", "r"} {
		if !hasHelp(got.helpForScreen().ShortHelp(), keyName) {
			t.Fatalf("compile warning help should expose %s", keyName)
		}
	}
	if hasHelp(got.helpForScreen().ShortHelp(), "enter") {
		t.Fatalf("compile warning help should not make enter the primary action: %#v", got.helpForScreen().ShortHelp())
	}
}

func TestCompileEnterStartsOperatingLayerAfterUsableMixtape(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("# Synthesis\nUse the compiled evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompile,
		width:       100,
		currentPath: project,
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 1, Succeeded: 1},
		},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenLinerReview {
		t.Fatalf("enter should start Operating Layer review, screen=%v err=%s", got.screen, got.err)
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "enter") || !hasHelpDesc(help, "operating layer") {
		t.Fatalf("compile help should expose enter as Operating Layer action, got %#v", help)
	}
	if !hasHelp(help, "p") {
		t.Fatalf("compile help should keep p as preview action, got %#v", help)
	}
	if got := m.nextAction(); !strings.Contains(got, "Create the Operating Layer") || !strings.Contains(got, "p to preview") {
		t.Fatalf("compile next action should point to Operating Layer and optional preview, got %q", got)
	}
}

func TestCompileRecoveredJSFallbackWarningStillContinues(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("# Synthesis\nUse the compiled evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	warning := core.CompileWarningPayload{
		URL:      "https://medium.com/eightshapes-llc/space-in-design-systems-188bcbae0d62",
		Message:  "server-rendered fetch hit a JS stub; auto-fell back to render: js for https://medium.com/eightshapes-llc/space-in-design-systems-188bcbae0d62",
		Severity: "warning",
	}
	m := Model{
		screen:      screenCompile,
		width:       118,
		currentPath: project,
		compileResult: &core.CompileResultPayload{
			MixtapePath: filepath.Join(project, "MIXTAPE.md"),
			Summary:     core.CompileSummary{Total: 1, Succeeded: 1, Failed: 0},
			Warnings:    []core.CompileWarningPayload{warning},
		},
	}

	if compileWarningBlocksProgress(warning) {
		t.Fatal("recovered JS fallback warning should not block the next project phase")
	}
	if got := compileWarningRecommendation(warning); !strings.Contains(got, "recovered") || !strings.Contains(got, "included") {
		t.Fatalf("recovered warning recommendation should explain what happened, got %q", got)
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "enter") {
		t.Fatalf("recovered source help should expose continue, got %#v", help)
	}
	if hasHelp(help, "o") || hasHelpDesc(help, "warning") {
		t.Fatalf("recovered source help should not expose warning inspection actions, got %#v", help)
	}
	view := m.viewCompileResult(styles.ClampWidth(m.width - 4))
	if !strings.Contains(view, "recovered 1 source(s) with browser rendering") || !strings.Contains(view, "included in MIXTAPE.md") {
		t.Fatalf("recovered source should render as a success note, got:\n%s", view)
	}
	if strings.Contains(view, "source issue") || strings.Contains(view, "Issue detail") {
		t.Fatalf("recovered source should not render warning table/details, got:\n%s", view)
	}
	if got := m.nextAction(); !strings.Contains(got, "Corpus Ready. Create the Operating Layer") {
		t.Fatalf("recovered source next action should point to Operating Layer, got %q", got)
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenLinerReview {
		t.Fatalf("enter should continue to Operating Layer review, screen=%v err=%s", got.screen, got.err)
	}
}

func TestCompileEnterShowsImprovementReviewWhenAuditRecommendsPass(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("# Synthesis\nUse the compiled evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	audit := "status: improvement_recommended\ngap: Missing high-craft case studies for translating visual references into UI direction.\n"
	if err := os.WriteFile(filepath.Join(project, operatingFitAuditRelPath), []byte(audit), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompile,
		width:       118,
		currentPath: project,
		currentTape: tape.Tape{Title: "Art Director"},
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 1, Succeeded: 1},
		},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if got.screen != screenImprovementReview {
		t.Fatalf("enter should show improvement review, screen=%v err=%s", got.screen, got.err)
	}
	rawView := got.viewImprovementReview()
	view := stripANSICodesForTest(rawView)
	for _, expected := range []string{
		"Improve Corpus",
		"Quality checks found a source-role gap",
		"Notes:",
		"working/05-operating-fit-audit.md",
		"Missing high-craft case studies",
		"Improve now",
		"Skip",
		"Run a focused improvement pass",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("improvement view missing %q:\n%s", expected, view)
		}
	}
	for _, expected := range []string{styles.AccentText.Render("Improve now"), styles.MutedText.Render("Skip")} {
		if !strings.Contains(rawView, expected) {
			t.Fatalf("improvement view missing styled option %q:\n%s", expected, rawView)
		}
	}
	if got := got.nextAction(); got != "Run the improvement pass before creating the Operating Layer." {
		t.Fatalf("unexpected improvement next action: %q", got)
	}
	help := got.helpForScreen().ShortHelp()
	if !hasHelp(help, "↑/↓") || !hasHelpDesc(help, "option") || !hasHelpDesc(help, "select") {
		t.Fatalf("improvement help should expose option/select controls: %#v", help)
	}
}

func TestImprovementReviewSkipContinuesToOperatingLayer(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("# Synthesis\nUse the compiled evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, operatingFitAuditRelPath), []byte("status: improvement_recommended\ngap: Missing direct examples.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:            screenImprovementReview,
		width:             118,
		currentPath:       project,
		currentTape:       tape.Tape{Title: "Art Director"},
		improvementCursor: improvementOptionSkip,
	}

	rawView := m.viewImprovementReview()
	view := stripANSICodesForTest(rawView)
	if !strings.Contains(view, "Skip for now. Liner keeps the improvement notes") ||
		!strings.Contains(view, "will offer this pass again") ||
		!strings.Contains(view, "if you run Compile for this project") {
		t.Fatalf("skip selection should show skip detail:\n%s", view)
	}
	for _, expected := range []string{styles.AccentText.Render("Skip"), styles.MutedText.Render("Improve now")} {
		if !strings.Contains(rawView, expected) {
			t.Fatalf("skip selection missing styled option %q:\n%s", expected, rawView)
		}
	}
	if got := m.nextAction(); got != "Skip for now and continue to the Operating Layer." {
		t.Fatalf("unexpected skip next action: %q", got)
	}

	got, _ := m.handleImprovementReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen != screenLinerReview {
		t.Fatalf("skip should continue to Operating Layer review, screen=%v err=%s", got.screen, got.err)
	}
	if !strings.Contains(got.note, "Skipped improvement pass for now") {
		t.Fatalf("skip should leave confirmation note, got %q", got.note)
	}
}

func TestCompilePPreviewsUsableMixtape(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "MIXTAPE.md"), []byte("# Ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("# Synthesis\nUse the compiled evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompile,
		width:       100,
		currentPath: project,
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 1, Succeeded: 1},
		},
	}

	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))

	if got.screen != screenPreview || got.previewRel != "MIXTAPE.md" {
		t.Fatalf("p should preview MIXTAPE.md, screen=%v rel=%q err=%s", got.screen, got.previewRel, got.err)
	}
}

func TestDropSelectedCompileWarningSourceRemovesMatchingTapeSource(t *testing.T) {
	project := t.TempDir()
	current := tape.Tape{
		Title: "Compile Recourse",
		Sources: []tape.Source{
			{Type: "web", URL: "https://example.com/keep"},
			{Type: "web", URL: "https://example.com/drop"},
		},
	}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenCompile,
		width:       100,
		currentPath: project,
		currentTape: current,
		compileRows: initialCompileRows(current.Sources),
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 2, Succeeded: 1, Failed: 1},
			Warnings: []core.CompileWarningPayload{{
				URL:      "https://example.com/drop",
				Message:  "not found",
				Severity: "error",
			}},
		},
	}

	got, _ := m.dropSelectedCompileWarningSource()

	if got.err != "" {
		t.Fatalf("drop should not fail, got %q", got.err)
	}
	if !strings.Contains(got.note, "Dropped 1 source") {
		t.Fatalf("expected drop note, got %q", got.note)
	}
	updated, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Sources) != 1 || updated.Sources[0].URL != "https://example.com/keep" {
		t.Fatalf("unexpected sources after drop: %#v", updated.Sources)
	}
	if len(got.compileRows) != 1 || got.compileRows[0].Source != "https://example.com/keep" {
		t.Fatalf("expected compile rows to remove dropped source, got %#v", got.compileRows)
	}
}

func TestCompileProgressAdvancesAfterUsableResult(t *testing.T) {
	project := t.TempDir()
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseCompile)}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		currentPath: project,
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 2, Succeeded: 1, Failed: 1},
		},
	}

	m.recordCompileProgress()

	if got := linerprogress.Read(project).Step; got != len(linerprogress.PhaseOrder) {
		t.Fatalf("expected compile progress to complete, got step %d", got)
	}
}

func TestCompileProgressWaitsWithoutUsableResult(t *testing.T) {
	project := t.TempDir()
	if err := linerprogress.Write(project, linerprogress.Progress{Step: linerprogress.PhaseIndex(linerprogress.PhaseCompile)}); err != nil {
		t.Fatal(err)
	}
	m := Model{
		currentPath: project,
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 2, Succeeded: 0, Failed: 2},
		},
	}

	m.recordCompileProgress()

	if got := linerprogress.Read(project).Step; got != linerprogress.PhaseIndex(linerprogress.PhaseCompile) {
		t.Fatalf("expected failed compile to stay on compile, got step %d", got)
	}
}

func TestCompileProgressSkipsWithoutCurrentProject(t *testing.T) {
	m := Model{
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 1, Succeeded: 1},
		},
	}

	m.recordCompileProgress()

	if len(m.compileLines) > 0 {
		t.Fatalf("expected no progress warning without a project path, got %v", m.compileLines)
	}
}

func TestCompileViewCallsOutZeroSourcesAndPlaceholderSynthesis(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "synthesis.md"), []byte("Replace this placeholder with the curator's distilled understanding."), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:       screenCompile,
		width:        118,
		currentPath:  dir,
		compileBar:   newCompileProgress(compileProgressWidth(118)),
		compileLines: []string{"Starting compile...", "Result: 0/0 usable sources", "Compile finished."},
		compileResult: &core.CompileResultPayload{
			MixtapePath: filepath.Join(dir, "MIXTAPE.md"),
			Summary:     core.CompileSummary{Total: 0, Succeeded: 0, Failed: 0},
		},
	}

	view := m.viewCompile()
	for _, expected := range []string{"Needs attention", "No sources are attached", "synthesis.md is still the starter placeholder"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile view missing %q:\n%s", expected, view)
		}
	}
	if got := m.nextAction(); !strings.Contains(got, "Add sources or define the job") {
		t.Fatalf("expected attention next action, got %q", got)
	}
}

func TestCompileRunningViewUsesTitleLoader(t *testing.T) {
	m := Model{
		screen:       screenCompile,
		width:        118,
		height:       36,
		compiling:    true,
		compileTotal: 3,
		compileDoneN: 1,
		compileBar:   newCompileProgress(compileProgressWidth(118)),
		compileLines: []string{"Starting compile...", "Fetching 3 sources"},
		researchSpin: newLoadingSpinner(),
	}

	view := m.viewCompile()
	assertTitleLineHasLoader(t, view, "Compile Console")
	if !strings.Contains(view, "Working") {
		t.Fatalf("running compile view should show working status:\n%s", view)
	}
}

func TestCompileViewTellsNoSourceJTBDProjectToRunMethodology(t *testing.T) {
	jtbd := "When I need a terminal design corpus, I want Liner to research it for me."
	m := Model{
		screen:      screenCompile,
		width:       118,
		currentPath: t.TempDir(),
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
		compileBar:  newCompileProgress(compileProgressWidth(118)),
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 0, Succeeded: 0, Failed: 0},
		},
	}

	view := m.viewCompile()
	for _, expected := range []string{"No saved sources", "Build Corpus", "save the draft sources"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compile view missing %q:\n%s", expected, view)
		}
	}
}

func TestCompileExitErrorUsesHelpfulPartialMessage(t *testing.T) {
	m := Model{
		compileResult: &core.CompileResultPayload{
			Summary: core.CompileSummary{Total: 5, Succeeded: 3, Failed: 2},
		},
	}

	message := m.friendlyCompileError(core.CompileExitError{Code: 2})
	if !strings.Contains(message, "Partial compile: 3/5 sources were usable") {
		t.Fatalf("unexpected partial compile message: %q", message)
	}
	if strings.Contains(message, "exit status") {
		t.Fatalf("message should not expose raw exit status: %q", message)
	}
}

func TestClarificationAnswersAppendToTape(t *testing.T) {
	current := tape.Tape{Title: "Launch"}
	questions := []clarifyQuestion{
		{question: "Which platform should the research prioritize?"},
		{question: "What output should the mixtape support?"},
		{question: "Which references should anchor quality?"},
	}
	answers := []string{
		"Design engineers",
		"Pick better interaction patterns",
		"",
	}

	next := applyClarificationAnswers(current, questions, answers)
	if got, want := len(next.JTBDClarifications), 2; got != want {
		t.Fatalf("clarification count = %d, want %d", got, want)
	}
	if next.JTBDClarifications[0].Question != questions[0].question {
		t.Fatalf("unexpected first question: %#v", next.JTBDClarifications[0])
	}
	if next.JTBDClarifications[1].Answer != answers[1] {
		t.Fatalf("unexpected second answer: %#v", next.JTBDClarifications[1])
	}
}

func TestClarificationScreenUsesGeneratedQuestions(t *testing.T) {
	jtbd := "When I design iOS apps, I want award-level interaction guidance."
	m := Model{
		screen:      screenClarify,
		width:       100,
		clarifyArea: newClarifyArea(64),
		currentTape: tape.Tape{Title: "iOS", JTBD: &jtbd},
	}
	m.setClarifyQuestions([]string{"Which iOS surfaces should the research prioritize?"})

	view := m.viewClarify()
	if !strings.Contains(view, "Which iOS surfaces should the research prioritize?") {
		t.Fatalf("generated question missing:\n%s", view)
	}
	if strings.Contains(view, "Who is this mixtape for?") {
		t.Fatalf("clarify screen should not use hardcoded questions:\n%s", view)
	}
}

func TestClarificationFailureRetriesOnEnter(t *testing.T) {
	jtbd := "When I design web interfaces, I want art-direction guidance."
	m := Model{
		screen:       screenClarify,
		width:        100,
		currentPath:  t.TempDir(),
		currentTape:  tape.Tape{Title: "Art Director", JTBD: &jtbd},
		clarifyArea:  newClarifyArea(64),
		clarifyError: "OpenAI Codex could not generate clarification questions after retry",
	}

	view := m.viewClarify()
	if !strings.Contains(view, "Press Enter to retry question generation") {
		t.Fatalf("clarification failure should offer retry:\n%s", view)
	}
	if action := m.nextAction(); action != "Retry clarification question generation." {
		t.Fatalf("unexpected next action: %q", action)
	}

	next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := next.(Model)
	if !got.clarifyLoading || got.clarifyError != "" {
		t.Fatalf("enter should restart clarification generation, loading=%v err=%q", got.clarifyLoading, got.clarifyError)
	}
	if cmd == nil {
		t.Fatal("expected retry command")
	}
}

func TestClarificationTypingQDoesNotQuitAndAutosavesDraft(t *testing.T) {
	project := t.TempDir()
	jtbd := "When I design TUIs, I want a clean interaction model."
	current := tape.Tape{Title: "TUI", JTBD: &jtbd}
	if err := tape.WriteProject(project, current); err != nil {
		t.Fatal(err)
	}
	m := Model{
		screen:      screenClarify,
		width:       100,
		currentPath: project,
		currentTape: current,
		clarifyArea: newClarifyArea(64),
	}
	m.setClarifyQuestions([]string{"Which examples should guide this mixtape?"})

	next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	got := next.(Model)
	if got.screen != screenClarify {
		t.Fatalf("typing q should stay on clarification screen, got %v", got.screen)
	}
	if got.clarifyArea.Value() != "q" || got.clarifyAnswers[0] != "q" {
		t.Fatalf("typing q should update the answer, area=%q answers=%#v", got.clarifyArea.Value(), got.clarifyAnswers)
	}
	if cmd == nil {
		t.Fatal("expected textarea cursor command after typing")
	}
	data, err := os.ReadFile(projectAbsPath(project, clarificationDraftRelPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"q"`) {
		t.Fatalf("expected clarification draft to persist typed answer:\n%s", data)
	}
}

func TestClarificationAnswerWrapsWithinTerminalWidth(t *testing.T) {
	jtbd := "When I design TUIs, I want a clean interaction model."
	width := 88
	m := Model{
		screen:      screenClarify,
		width:       width,
		clarifyArea: newClarifyArea(clarifyAreaWidth(width)),
		currentTape: tape.Tape{Title: "TUI", JTBD: &jtbd},
	}
	m.setClarifyQuestions([]string{"Which examples should guide this mixtape?"})
	m.clarifyArea.SetValue("Superfile, Daytona, gh dash, Expo CLI, and Charm examples should guide the interaction patterns and visual rhythm.")

	view := m.viewClarify()
	for _, line := range strings.Split(view, "\n") {
		if got, want := lipgloss.Width(line), styles.ClampWidth(width-4); got > want {
			t.Fatalf("clarify line wider than terminal: got %d, want <= %d\n%s", got, want, line)
		}
	}
}

func TestClarificationLoadingUsesTapeLoader(t *testing.T) {
	jtbd := "When I design iOS apps, I want award-level interaction guidance."
	m := Model{
		screen:         screenClarify,
		width:          100,
		clarifyLoading: true,
		currentTape:    tape.Tape{Title: "iOS", JTBD: &jtbd},
		researchSpin:   newLoadingSpinner(),
	}

	view := m.viewClarify()
	assertTitleLineHasLoader(t, view, "Clarify Goal")
	for _, expected := range []string{"AI is working on it.", "Reading the AI-agent goal and preparing clarification questions."} {
		if !strings.Contains(view, expected) {
			t.Fatalf("loader missing %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "Wait") || strings.Contains(view, "wait") {
		t.Fatalf("loader should not use generic wait copy:\n%s", view)
	}
}

func TestClarificationLoaderMessageDoesNotRotate(t *testing.T) {
	first := clarifyLoaderMessage(0)
	later := clarifyLoaderMessage(999)

	if first == "" {
		t.Fatal("loader message should not be empty")
	}
	if first != later {
		t.Fatalf("loader message should stay stable, got %q then %q", first, later)
	}
}

func TestCompileLoaderMessageDoesNotRotate(t *testing.T) {
	first := compileLoaderMessage(0)
	later := compileLoaderMessage(999)

	if first == "" {
		t.Fatal("compile loader message should not be empty")
	}
	if first != later {
		t.Fatalf("compile loader message should stay stable, got %q then %q", first, later)
	}
	if !strings.Contains(first, "MIXTAPE.md") || strings.Contains(strings.ToLower(first), "wait") {
		t.Fatalf("compile loader should be concrete and not generic wait copy, got %q", first)
	}
}

func TestStyledReportUsesListMarkersAndDimSectionTitles(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	current := tape.Tape{Title: "Launch", JTBD: &jtbd}

	body := renderReportBody(current, nil, 92)
	if strings.Contains(body, "- Setup context") {
		t.Fatalf("styled report should not use raw dash bullets:\n%s", body)
	}
	if !strings.Contains(body, "✓") {
		t.Fatalf("styled report should use completed-state list markers:\n%s", body)
	}
	if !strings.Contains(body, styles.ReportSection.Render("AI-agent goal")) {
		t.Fatalf("styled report should use dim report section titles:\n%s", body)
	}
}

func TestReportViewIsUnboxed(t *testing.T) {
	jtbd := "Help me understand what to include in a launch article."
	m := Model{
		width:       118,
		currentTape: tape.Tape{Title: "Launch", JTBD: &jtbd},
	}

	view := m.viewReport()
	for _, border := range []string{"┌", "┐", "└", "┘"} {
		if strings.Contains(view, border) {
			t.Fatalf("report view should not render a content box:\n%s", view)
		}
	}
}
