package app

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func (m Model) viewCommands() string {
	return m.commands.View()
}

func newCommandList(width int, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(styles.Accent).
		BorderForeground(styles.Accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(styles.Muted).
		BorderForeground(styles.Muted)
	delegate.Styles.FilterMatch = styles.AccentText.Underline(true)

	commands := list.New([]list.Item{}, delegate, width, height)
	commands.Title = ""
	commands.SetShowStatusBar(false)
	commands.SetShowTitle(false)
	commands.SetShowHelp(false)
	commands.Styles.DefaultFilterCharacterMatch = styles.AccentText.Underline(true)
	commands.Styles.Filter.Cursor.Color = styles.Accent
	commands.Styles.Filter.Focused.Prompt = styles.AccentText
	commands.Styles.Filter.Blurred.Prompt = styles.AccentText
	return commands
}

func (m Model) viewHome() string {
	return m.viewCommands()
}

func (m Model) showCommandHome() Model {
	m.screen = screenHome
	m.commands.SetItems(m.commandItems())
	m.commands.SetFilterText("")
	m.commands.SetFilterState(list.Unfiltered)
	return m
}

func (m Model) commandListFiltering() bool {
	return m.commands.FilterState() == list.Filtering
}

func (m Model) runSelectedCommand() (Model, tea.Cmd) {
	item, ok := m.commands.SelectedItem().(commandItem)
	if !ok || item.run == nil {
		return m, nil
	}
	return item.run(m)
}

func (m Model) commandItems() []list.Item {
	hasProject := strings.TrimSpace(m.currentPath) != ""
	items := []list.Item{
		commandItem{"New Liner Project", "Create a project and add sources", func(m Model) (Model, tea.Cmd) { m.startCreate(); return m, nil }},
	}
	if hasProject {
		items = append(items, commandItem{"Current project", "Return to the open Liner project", func(m Model) (Model, tea.Cmd) {
			m.screen = screenProject
			return m, nil
		}})
	}
	items = append(items,
		commandItem{"Projects", "Browse Liner projects", func(m Model) (Model, tea.Cmd) { m.screen = screenProjects; return m, loadProjects(m.runner, m.baseDir) }},
		commandItem{"Import Project", "Choose a .mixtape project file", func(m Model) (Model, tea.Cmd) {
			return m, m.startImport()
		}},
		commandItem{"Settings", "Change projects folder and AI runner", func(m Model) (Model, tea.Cmd) { return m.startSettings(), nil }},
	)
	if hasProject {
		items = append(items, commandItem{"Add sources", "Paste URLs, files, articles, and local documents", func(m Model) (Model, tea.Cmd) {
			m.startSourceEntry()
			return m, nil
		}})
	}
	if m.canOpenSourceBoard() {
		items = append(items, commandItem{"Review Sources", "Review and toggle saved sources", func(m Model) (Model, tea.Cmd) {
			return m.startSourceBoard()
		}})
	}
	if hasProject {
		items = append(items, commandItem{"Build Corpus", "Run or resume corpus creation", func(m Model) (Model, tea.Cmd) {
			return m.startResearch()
		}})
	}
	if hasProject {
		items = append(items, commandItem{"Compile MIXTAPE.md", "Build MIXTAPE.md from saved sources", func(m Model) (Model, tea.Cmd) { return m.startCompile() }})
	}
	if hasProject && m.hasDroppedCustomSources() {
		items = append(items, commandItem{"Retry dropped sources", "Fetch dropped custom sources without rebuilding the corpus", func(m Model) (Model, tea.Cmd) {
			return m.retryDroppedCustomSources()
		}})
	}
	if m.hasCompiledMixtape() {
		items = append(items, commandItem{"Preview MIXTAPE.md", "Render the compiled mixtape", func(m Model) (Model, tea.Cmd) { return m.openPreview("MIXTAPE.md") }})
	}
	if m.projectCapabilities().HasLiner {
		items = append(items, commandItem{"Preview LINER.md", "Render project instructions", func(m Model) (Model, tea.Cmd) {
			return m.openPreview("LINER.md")
		}})
		if m.canRegenerateOperatingLayer() {
			items = append(items, commandItem{"Regenerate Operating Layer", "Rewrite LINER.md and SKILL.md from the current corpus", func(m Model) (Model, tea.Cmd) {
				return m.startLinerDraftReview()
			}})
		}
	} else if m.canCreateOperatingLayer() {
		items = append(items, commandItem{"Create Operating Layer", "Write LINER.md, SKILL.md, and local status", func(m Model) (Model, tea.Cmd) {
			return m.startLinerDraftReview()
		}})
	}
	if hasProject && projectDirExists(m.currentPath, "local-sources") {
		items = append(items, commandItem{"Open local-sources", "Open the local drop folder", func(m Model) (Model, tea.Cmd) {
			return m, openPath(projectAbsPath(m.currentPath, "local-sources"))
		}})
	}
	if hasProject && globalEstimateHistoryHasEntries() {
		items = append(items, commandItem{"Reset cost estimates", "Forget saved corpus-build token samples", func(m Model) (Model, tea.Cmd) {
			return m.clearEstimateHistory()
		}})
	}
	return items
}

func (m Model) footerHelp() string {
	helpMap := m.helpForScreen()
	if m.screen != screenHome {
		helpMap = withoutQuitKeyBinding(helpMap)
	}
	if m.help.ShowAll {
		helpKey := bindings.Help
		helpKey.SetHelp("?", "less help")
		helpMap = withHelpBinding(helpMap, helpKey)
		return m.help.View(helpMap)
	}
	width := m.help.Width()
	if width <= 0 {
		width = styles.ClampWidth(m.width - 4)
	}
	return wrappedShortHelp(m.help, helpMap.ShortHelp(), width)
}

func wrappedShortHelp(helpModel help.Model, bindings []key.Binding, width int) string {
	if len(bindings) == 0 {
		return ""
	}
	if width <= 0 {
		width = 60
	}
	separator := helpModel.Styles.ShortSeparator.Inline(true).Render(helpModel.ShortSeparator)
	var lines []string
	current := ""
	for _, binding := range bindings {
		if !binding.Enabled() {
			continue
		}
		helpText := binding.Help()
		if strings.TrimSpace(helpText.Key) == "" || strings.TrimSpace(helpText.Desc) == "" {
			continue
		}
		item := helpModel.Styles.ShortKey.Inline(true).Render(helpText.Key) + " " +
			helpModel.Styles.ShortDesc.Inline(true).Render(helpText.Desc)
		if current == "" {
			current = wrapHelpItem(item, helpText.Key+" "+helpText.Desc, width)
			continue
		}
		next := current + separator + item
		if lipgloss.Width(next) <= width {
			current = next
			continue
		}
		lines = append(lines, current)
		current = wrapHelpItem(item, helpText.Key+" "+helpText.Desc, width)
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}

func wrapHelpItem(styled string, plain string, width int) string {
	if lipgloss.Width(styled) <= width {
		return styled
	}
	return truncateMiddle(plain, width)
}

func withoutQuitKeyBinding(helpMap screenHelp) screenHelp {
	filter := func(items []key.Binding) []key.Binding {
		out := make([]key.Binding, 0, len(items))
		for _, item := range items {
			if item.Help().Key == bindings.QuitKey.Help().Key {
				continue
			}
			out = append(out, item)
		}
		return out
	}
	full := make([][]key.Binding, len(helpMap.full))
	for i, group := range helpMap.full {
		full[i] = filter(group)
	}
	return screenHelp{
		short: filter(helpMap.short),
		full:  full,
	}
}

func withHelpBinding(helpMap screenHelp, replacement key.Binding) screenHelp {
	replace := func(bindings []key.Binding) []key.Binding {
		out := make([]key.Binding, len(bindings))
		copy(out, bindings)
		for i, binding := range out {
			if binding.Help().Key == replacement.Help().Key {
				out[i] = replacement
			}
		}
		return out
	}
	full := make([][]key.Binding, len(helpMap.full))
	for i, group := range helpMap.full {
		full[i] = replace(group)
	}
	return screenHelp{
		short: replace(helpMap.short),
		full:  full,
	}
}
