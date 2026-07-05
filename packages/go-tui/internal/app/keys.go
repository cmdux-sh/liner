package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/source"
)

type keyBindings struct {
	OpenProject key.Binding
	NewProject  key.Binding
	Import      key.Binding
	Refresh     key.Binding
	NewLine     key.Binding
	Next        key.Binding
	AddSources  key.Binding
	Compile     key.Binding
	Preview     key.Binding
	Copy        key.Binding
	Share       key.Binding
	Liner       key.Binding
	Skills      key.Binding
	Audits      key.Binding
	Home        key.Binding
	Settings    key.Binding
	Back        key.Binding
	Quit        key.Binding
	QuitKey     key.Binding
	Help        key.Binding
	Review      key.Binding
	Save        key.Binding
	Ingest      key.Binding
	Finish      key.Binding
	Toggle      key.Binding
	Remove      key.Binding
	Start       key.Binding
	EditMore    key.Binding
	OpenFolder  key.Binding
	OpenReview  key.Binding
	Retry       key.Binding
	Scroll      key.Binding
}

var bindings = keyBindings{
	OpenProject: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	NewProject:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Import:      key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import")),
	Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	NewLine:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "new line")),
	Next:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "next step")),
	AddSources:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add sources")),
	Compile:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compile")),
	Preview:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "preview")),
	Copy:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
	Share:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "archive")),
	Liner:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "LINER.md")),
	Skills:      key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "skills")),
	Audits:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "audits")),
	Home:        key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "home")),
	Settings:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "settings")),
	Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:        key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	QuitKey:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more help")),
	Review:      key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "review import")),
	Save:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save sources")),
	Ingest:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "add source")),
	Finish:      key.NewBinding(key.WithKeys("f", "ctrl+d"), key.WithHelp("f", "finish")),
	Toggle:      key.NewBinding(key.WithKeys("space", " "), key.WithHelp("space", "toggle active")),
	Remove:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove")),
	Start:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save sources")),
	EditMore:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "edit more")),
	OpenFolder:  key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "open folder")),
	OpenReview:  key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open folder")),
	Retry:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
	Scroll:      key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "scroll")),
}

func settingsKeyLabel() string {
	return bindings.Settings.Help().Key
}

type screenHelp struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h screenHelp) ShortHelp() []key.Binding {
	return h.short
}

func (h screenHelp) FullHelp() [][]key.Binding {
	return h.full
}

func (m Model) handleKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if key.Matches(keyMsg, bindings.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}
	if m.screen == screenCompile && m.sourceRecoveryRunning {
		if key.Matches(keyMsg, bindings.Quit) {
			return m, tea.Quit
		}
		m.note = "Unavailable source retry is still running. Wait for the retry result."
		m.err = ""
		return m, nil
	}
	if m.screen == screenCompile && m.sourceRecoveryReview {
		if key.Matches(keyMsg, bindings.Quit) {
			return m, tea.Quit
		}
		if key.Matches(keyMsg, bindings.Next) || keyMsg.String() == "enter" {
			return m.continueFromCompile()
		}
		if m.compileRepairRebuildCorpusAfterRecovery {
			m.note = "Press enter to refresh source evaluation."
		} else if m.compileRepairAttempted && m.sourceRecovery != nil && m.sourceRecovery.Succeeded == 0 {
			m.note = "Press enter to return to Sources."
		} else {
			m.note = "Press enter to return to Compile Console."
		}
		m.err = ""
		return m, nil
	}
	if m.screen == screenLinerReview && m.operatingLayerRunning {
		return m.handleLinerReviewKey(keyMsg)
	}
	if key.Matches(keyMsg, bindings.Home) && m.screen == screenResearch {
		m.stopMethodology("Paused by user.")
		m.researchDone = true
		m = m.showCommandHome()
		return m, nil
	}
	if key.Matches(keyMsg, bindings.Home) && m.supportsHomeShortcut() {
		m = m.showCommandHome()
		return m, nil
	}
	if m.screen == screenSources {
		return m.handleSourceKey(keyMsg)
	}
	if m.screen == screenSourceReview {
		return m.handleSourceReviewKey(keyMsg)
	}
	if m.screen == screenAssemblyReview {
		return m.handleAssemblyReviewKey(keyMsg)
	}
	if m.screen == screenLinerReview {
		return m.handleLinerReviewKey(keyMsg)
	}
	if m.screen == screenImprovementReview {
		return m.handleImprovementReviewKey(keyMsg)
	}
	if m.screen == screenSkills {
		return m.handleSkillsKey(keyMsg)
	}
	if m.screen == screenSkillReview {
		return m.handleSkillReviewKey(keyMsg)
	}
	if m.screen == screenAudits {
		return m.handleAuditsKey(keyMsg)
	}
	if m.screen == screenContradictionCleanupReview {
		return m.handleContradictionCleanupReviewKey(keyMsg)
	}
	if m.screen == screenSourceNoteCleanupReview {
		return m.handleSourceNoteCleanupReviewKey(keyMsg)
	}
	if m.screen == screenEvals {
		return m.handleEvalsKey(keyMsg)
	}
	if m.screen == screenComposition {
		return m.handleCompositionKey(keyMsg)
	}
	if m.screen == screenCompositionReview {
		return m.handleCompositionReviewKey(keyMsg)
	}
	if m.screen == screenImport {
		return m.handleImportKey(keyMsg)
	}
	if m.screen == screenOnboarding {
		return m.handleOnboardingKey(keyMsg)
	}
	if m.screen == screenResearch && key.Matches(keyMsg, bindings.Retry) {
		return m.retryMethodologyPhase()
	}
	if m.screen == screenProjects && (m.homeFiltering || (keyMsg.String() == "esc" && m.homeFilter != "")) {
		return m.handleHomeFilterKey(keyMsg)
	}
	if m.screen == screenHome && m.commandListFiltering() && !key.Matches(keyMsg, bindings.Quit) {
		return m, nil
	}

	if key.Matches(keyMsg, bindings.Quit) || (m.screen == screenHome && key.Matches(keyMsg, bindings.QuitKey)) {
		return m, tea.Quit
	}
	if key.Matches(keyMsg, bindings.Back) {
		if m.screen == screenHome {
			return m, nil
		}
		if m.screen == screenProjects {
			m = m.showCommandHome()
			return m, nil
		}
		if m.screen == screenResearch {
			m.stopMethodology("Paused by user.")
			m.researchDone = true
			m.screen = screenProject
			return m, nil
		}
		if m.screen == screenBoard {
			m.screen = screenReport
			return m, nil
		}
		if m.screen == screenPreview {
			m = m.closePreview()
			return m, nil
		}
		if m.screen == screenSettings {
			m = m.closeSettings()
			return m, nil
		}
		if m.screen == screenProject {
			if m.currentPath == "" {
				m = m.showCommandHome()
				return m, nil
			}
			m.screen = screenProjects
			return m, loadProjects(m.runner, m.baseDir)
		}
		m.screen = screenProject
		if m.currentPath == "" {
			m = m.showCommandHome()
		}
		return m, nil
	}

	if m.screen == screenHome {
		if key.Matches(keyMsg, bindings.Settings) {
			return m.startSettings(), nil
		}
		switch keyMsg.String() {
		case "enter":
			return m.runSelectedCommand()
		case "o":
			m.screen = screenProjects
			return m, loadProjects(m.runner, m.baseDir)
		case "n":
			m.startCreate()
		case "i":
			return m, m.startImport()
		case "r":
			return m, loadProjects(m.runner, m.baseDir)
		}
		return m, nil
	}

	if m.screen == screenProjects {
		if key.Matches(keyMsg, bindings.Settings) {
			return m.startSettings(), nil
		}
		switch keyMsg.String() {
		case "enter":
			if item, ok := m.selectedProjectItem(); ok {
				return m, openProject(item.project.Path)
			}
		case "/":
			m.homeFiltering = true
		case "n":
			m.startCreate()
		case "i":
			return m, m.startImport()
		case "r":
			return m, loadProjects(m.runner, m.baseDir)
		}
		return m, nil
	}

	if m.screen == screenSettings {
		return m.handleSettingsKey(keyMsg)
	}

	if m.screen == screenCreate {
		switch keyMsg.String() {
		case "enter":
			if m.createStep < createFieldCount()-1 {
				m.commitCreateInput()
				if !m.currentCreateFieldValid() {
					return m, nil
				}
				m.setCreateField(m.createStep + 1)
				return m, nil
			}
			return m.submitCreate()
		case "tab", "down":
			m.commitCreateInput()
			m.setCreateField((m.createStep + 1) % createFieldCount())
		case "shift+tab", "up":
			m.commitCreateInput()
			next := m.createStep - 1
			if next < 0 {
				next = createFieldCount() - 1
			}
			m.setCreateField(next)
		}
		if m.createStep == 3 {
			switch strings.ToLower(keyMsg.String()) {
			case "left", "right", " ":
				m.createDraft.AddSources = !m.createDraft.AddSources
			case "y":
				m.createDraft.AddSources = true
			case "n":
				m.createDraft.AddSources = false
			}
		}
		return m, nil
	}

	if m.screen == screenClarify {
		if m.clarifyLoading {
			return m, nil
		}
		if len(m.clarifyQuestions) == 0 {
			if keyMsg.String() == "enter" {
				return m.startClarificationFlow()
			}
			return m, nil
		}
		switch keyMsg.String() {
		case "enter":
			m.commitClarifyInput()
			if m.clarifyStep < len(m.clarifyQuestions)-1 {
				m.setClarifyField(m.clarifyStep + 1)
				m.persistClarificationDraft()
				return m, nil
			}
			return m.submitClarification()
		case "tab", "down":
			m.commitClarifyInput()
			m.setClarifyField((m.clarifyStep + 1) % len(m.clarifyQuestions))
			m.persistClarificationDraft()
		case "shift+tab", "up":
			m.commitClarifyInput()
			next := m.clarifyStep - 1
			if next < 0 {
				next = len(m.clarifyQuestions) - 1
			}
			m.setClarifyField(next)
			m.persistClarificationDraft()
		}
		return m, nil
	}

	if m.screen == screenProject {
		if key.Matches(keyMsg, bindings.Settings) {
			return m.startSettings(), nil
		}
		switch keyMsg.String() {
		case "up", "shift+tab":
			m.moveProjectPane(-1)
		case "down", "tab":
			m.moveProjectPane(1)
		case "enter":
			return m.primaryProjectAction()
		case "o":
			return m, openPath(m.currentPath)
		case "a":
			m.startSourceEntry()
		case "c":
			return m.startCompile()
		case "l":
			if m.projectCapabilities().HasLiner {
				return m.openPreview("LINER.md")
			}
			if m.projectCompileNeedsAttention() {
				m.err = "Review compile issues and retry compile before creating the Operating Layer."
				return m, nil
			}
			if m.canCreateOperatingLayer() {
				return m.startLinerDraftReview()
			}
			m.err = "Reach Corpus Ready before creating the Operating Layer."
		case "r":
			if m.canRegenerateOperatingLayer() {
				return m.startLinerDraftReview()
			}
		case "h":
			m = m.showCommandHome()
		}
		return m, nil
	}

	if m.screen == screenCompile {
		switch keyMsg.String() {
		case "enter":
			return m.continueFromCompile()
		case "up", "k":
			if m.compilePane == compilePaneSources {
				return m.moveCompileSourceSelection(-1), nil
			}
			return m.moveCompileWarningSelection(-1), nil
		case "down", "j":
			if m.compilePane == compilePaneSources {
				return m.moveCompileSourceSelection(1), nil
			}
			return m.moveCompileWarningSelection(1), nil
		case "a":
			m.startSourceEntry()
			return m, nil
		case "o":
			if m.compilePane == compilePaneSources {
				return m.openSelectedCompileSource()
			}
			return m.openSelectedCompileWarningSource()
		case "d":
			return m.dropSelectedCompileWarningSource()
		case "i":
			return m.startJSSetupForCompile()
		case "e":
			return m.retryExcludedLocalSources()
		case "r":
			return m.repairCompileSources()
		case "p":
			return m.previewCompiledMixtape()
		case "v":
			return m.openCompileLogPreview()
		case "y":
			return m, copyMixtape(m.currentPath)
		}
	}

	if m.screen == screenPreview {
		switch keyMsg.String() {
		case "y":
			if m.previewRel == "" || m.previewRel == "MIXTAPE.md" {
				return m, copyMixtape(m.currentPath)
			}
			m.err = "Copy is available from the MIXTAPE.md preview."
		case "ctrl+o":
			return m, openPath(m.currentPath)
		}
		return m, nil
	}

	if m.screen == screenResearch {
		return m, nil
	}

	if m.screen == screenReport {
		switch keyMsg.String() {
		case "enter":
			return m.startResearch()
		}
		return m, nil
	}

	if m.screen == screenBoard {
		return m.handleBoardKey(keyMsg)
	}

	return m, nil
}

func (m Model) supportsHomeShortcut() bool {
	switch m.screen {
	case screenHome, screenSources, screenCreate, screenClarify:
		return false
	case screenOnboarding:
		return false
	case screenProjects:
		return !m.homeFiltering
	default:
		return true
	}
}

func (m Model) withNavigationHelp(help screenHelp) screenHelp {
	if m.screen != screenHome && m.screen != screenOnboarding {
		help.short = insertHelpBinding(help.short, bindings.Back, bindings.Help)
		if len(help.full) == 0 {
			help.full = [][]key.Binding{{bindings.Back, bindings.Help}}
		} else {
			last := len(help.full) - 1
			help.full[last] = insertHelpBinding(help.full[last], bindings.Back, bindings.Help)
		}
	}
	if m.supportsHomeShortcut() {
		help.short = insertHelpBinding(help.short, bindings.Home, bindings.Back)
		if len(help.full) == 0 {
			help.full = [][]key.Binding{{bindings.Home}}
		} else {
			last := len(help.full) - 1
			help.full[last] = insertHelpBinding(help.full[last], bindings.Home, bindings.Back)
		}
	}
	return help
}

func insertHelpBinding(items []key.Binding, binding key.Binding, before key.Binding) []key.Binding {
	if hasHelpBinding(items, binding) {
		return items
	}
	beforeKey := before.Help().Key
	out := make([]key.Binding, 0, len(items)+1)
	inserted := false
	for _, item := range items {
		if !inserted && item.Help().Key == beforeKey {
			out = append(out, binding)
			inserted = true
		}
		out = append(out, item)
	}
	if !inserted {
		out = append(out, binding)
	}
	return out
}

func hasHelpBinding(items []key.Binding, binding key.Binding) bool {
	target := binding.Help().Key
	for _, item := range items {
		if item.Help().Key == target {
			return true
		}
	}
	return false
}

func (m Model) handleSourceKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	value := strings.TrimSpace(m.sourceInput.Value())
	switch {
	case key.Matches(keyMsg, bindings.Quit):
		return m, tea.Quit
	case key.Matches(keyMsg, bindings.Back):
		return m.returnFromSourceEntry(), nil
	case sourceEntryHasListFocus(m) && key.Matches(keyMsg, bindings.Toggle):
		index := clampSourceCursor(m.sourceTable.Cursor(), len(m.sourceItems))
		m.sourceItems[index].Active = !m.sourceItems[index].Active
		if m.sourceItems[index].Active {
			m.note = "Source activated. It will be included in review."
		} else {
			m.note = "Source deactivated. It stays staged but will not be used."
		}
		m.applySourceItems(m.sourceItems)
		return m, writeSourceManifest(m.currentPath, m.sourceItems)
	case sourceEntryHasListFocus(m) && key.Matches(keyMsg, bindings.Remove):
		index := clampSourceCursor(m.sourceTable.Cursor(), len(m.sourceItems))
		removed := m.sourceItems[index].Label
		m.sourceItems = append(m.sourceItems[:index], m.sourceItems[index+1:]...)
		m.applySourceItems(m.sourceItems)
		if len(m.sourceItems) == 0 {
			m.note = "Removed " + removed + ". No sources staged."
		} else {
			m.note = "Removed " + removed + "."
		}
		return m, writeSourceManifest(m.currentPath, m.sourceItems)
	case key.Matches(keyMsg, bindings.Ingest):
		if value == "" {
			if len(m.sourceItems) > 0 {
				m.applySourceItems(m.sourceItems)
				m.screen = screenSourceReview
				m.note = "Review active sources before clarifying the job."
				return m, nil
			}
			m.err = "Paste one source first, or finish without custom sources."
			return m, nil
		}
		m.note = "Adding source..."
		return m, ingestSource(m.currentPath, value)
	case key.Matches(keyMsg, bindings.Finish) && (value == "" || keyMsg.String() == "ctrl+d"):
		if value != "" {
			m.err = "Add the pasted source first, or clear it before finishing."
			return m, nil
		}
		if len(m.sourceItems) == 0 {
			if m.sourceEntryReturnsToCompile() {
				m = m.returnFromSourceEntry()
				m.note = "No sources added. Returned to Compile Console."
				m.err = ""
				return m, nil
			}
			m.note = ""
			return m.startClarificationFlow()
		}
		m.applySourceItems(m.sourceItems)
		m.screen = screenSourceReview
		m.note = "Review active sources before clarifying the job."
		return m, nil
	case key.Matches(keyMsg, bindings.OpenFolder):
		if !m.canOpenLocalSources() {
			m.err = "No local-sources folder yet. Add a source first."
			return m, nil
		}
		return m, openPath(projectAbsPath(m.currentPath, "local-sources"))
	}
	return m, nil
}

func (m Model) handleHomeFilterKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.homeFilter = ""
		m.homeFiltering = false
	case "enter":
		m.homeFiltering = false
	case "backspace":
		m.homeFilter = dropLastRune(m.homeFilter)
	default:
		if keyMsg.Text != "" {
			m.homeFilter += keyMsg.Text
		}
	}
	m.applyHomeProjectFilter()
	return m, nil
}

func dropLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func (m Model) handleSourceReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(keyMsg, bindings.Quit):
		return m, tea.Quit
	case key.Matches(keyMsg, bindings.Back):
		m.screen = screenSources
		return m, nil
	case key.Matches(keyMsg, bindings.EditMore):
		m.screen = screenSources
		return m, nil
	case key.Matches(keyMsg, bindings.Toggle):
		index := m.sourceTable.Cursor()
		if index >= 0 && index < len(m.sourceItems) {
			m.sourceItems[index].Active = !m.sourceItems[index].Active
			if m.sourceItems[index].Active {
				m.note = "Source activated. It will be written to tape.yaml."
			} else {
				m.note = "Source deactivated. It stays in local-sources but will not be written to tape.yaml."
			}
			m.applySourceItems(m.sourceItems)
			return m, writeSourceManifest(m.currentPath, m.sourceItems)
		}
	case key.Matches(keyMsg, bindings.Remove):
		index := m.sourceTable.Cursor()
		if index >= 0 && index < len(m.sourceItems) {
			m.sourceItems = append(m.sourceItems[:index], m.sourceItems[index+1:]...)
			m.applySourceItems(m.sourceItems)
			return m, writeSourceManifest(m.currentPath, m.sourceItems)
		}
	case key.Matches(keyMsg, bindings.Start):
		if len(source.ActiveSources(m.sourceItems)) == 0 {
			m.err = "No active sources selected. Reactivate one or add more."
			return m, nil
		}
		m.note = "Saving active sources."
		if !m.sourceEntryReturnsToCompile() {
			m.note = "Saving active sources and preparing clarification."
		}
		return m, saveActiveSources(m.currentPath, m.sourceItems)
	case key.Matches(keyMsg, bindings.OpenReview):
		if !m.canOpenLocalSources() {
			m.err = "No local-sources folder yet. Add a source first."
			return m, nil
		}
		return m, openPath(projectAbsPath(m.currentPath, "local-sources"))
	}
	return m, nil
}

func shouldEditSourceText(m Model, keyMsg tea.KeyPressMsg) bool {
	switch {
	case sourceEntryHasListFocus(m) && key.Matches(keyMsg, bindings.Scroll, bindings.Toggle, bindings.Remove):
		return false
	case key.Matches(keyMsg, bindings.Quit, bindings.Back, bindings.Ingest, bindings.OpenFolder, bindings.Help):
		return false
	case key.Matches(keyMsg, bindings.Finish):
		return strings.TrimSpace(m.sourceInput.Value()) != "" && keyMsg.String() != "ctrl+d"
	default:
		return true
	}
}

func sourceEntryShouldUpdateTable(m Model, keyMsg tea.KeyPressMsg) bool {
	return sourceEntryHasListFocus(m) && key.Matches(keyMsg, bindings.Scroll)
}

func shouldEditCreateText(m Model, keyMsg tea.KeyPressMsg) bool {
	if m.createStep == 3 {
		return false
	}
	switch keyMsg.String() {
	case "enter", "tab", "shift+tab", "up", "down", "esc":
		return false
	default:
		return !key.Matches(keyMsg, bindings.Help, bindings.Quit)
	}
}

func shouldEditClarifyText(keyMsg tea.KeyPressMsg) bool {
	switch keyMsg.String() {
	case "enter", "tab", "shift+tab", "up", "down", "esc":
		return false
	default:
		return !key.Matches(keyMsg, bindings.Help, bindings.Quit)
	}
}

func (m Model) helpForScreen() screenHelp {
	return m.withNavigationHelp(m.baseHelpForScreen())
}

func (m Model) baseHelpForScreen() screenHelp {
	helpKey := bindings.Help
	if m.help.ShowAll {
		helpKey.SetHelp("?", "less help")
	}
	switch m.screen {
	case screenSources:
		add := bindings.Ingest
		if strings.TrimSpace(m.sourceInput.Value()) == "" && len(m.sourceItems) > 0 {
			add.SetHelp("enter", "review")
		}
		short := []key.Binding{add, bindings.Finish, bindings.Back, helpKey}
		firstRow := []key.Binding{add, bindings.Finish}
		if sourceEntryHasListFocus(m) {
			short = []key.Binding{bindings.Scroll, add, bindings.Toggle, bindings.Remove, bindings.Finish, bindings.Back, helpKey}
			firstRow = []key.Binding{bindings.Scroll, add, bindings.Toggle, bindings.Remove, bindings.Finish}
		}
		if m.canOpenLocalSources() {
			short = []key.Binding{add, bindings.Finish, bindings.OpenFolder, bindings.Back, helpKey}
			firstRow = []key.Binding{add, bindings.Finish, bindings.OpenFolder}
			if sourceEntryHasListFocus(m) {
				short = []key.Binding{bindings.Scroll, add, bindings.Toggle, bindings.Remove, bindings.Finish, bindings.OpenFolder, bindings.Back, helpKey}
				firstRow = []key.Binding{bindings.Scroll, add, bindings.Toggle, bindings.Remove, bindings.Finish, bindings.OpenFolder}
			}
		}
		return screenHelp{
			short: short,
			full: [][]key.Binding{
				firstRow,
				{bindings.Back, bindings.Quit, helpKey},
			},
		}
	case screenSourceReview:
		firstRow := []key.Binding{bindings.Start, bindings.Toggle, bindings.Remove, bindings.EditMore}
		if m.canOpenLocalSources() {
			openFolder := bindings.OpenReview
			openFolder.SetHelp("o", "open folder")
			firstRow = append(firstRow, openFolder)
		}
		return screenHelp{
			short: []key.Binding{bindings.Start, bindings.Toggle, bindings.Remove, bindings.EditMore, bindings.Home, bindings.Back, helpKey},
			full: [][]key.Binding{
				firstRow,
				{bindings.Home, bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenAssemblyReview:
		accept := bindings.Start
		accept.SetHelp("enter", "accept")
		openDraft := bindings.OpenReview
		openDraft.SetHelp("o", "open draft")
		discard := bindings.Remove
		discard.SetHelp("d", "discard")
		return screenHelp{
			short: []key.Binding{accept, bindings.Toggle, openDraft, discard, bindings.Home, bindings.Back, helpKey},
			full: [][]key.Binding{
				{accept, bindings.Toggle, openDraft, discard},
				{bindings.Home, bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenLinerReview:
		next := bindings.Next
		next.SetHelp("enter", "create")
		if m.operatingLayerRunning {
			return screenHelp{
				short: []key.Binding{bindings.Quit, helpKey},
				full: [][]key.Binding{
					{bindings.Quit, bindings.QuitKey, helpKey},
				},
			}
		}
		if m.operatingLayerComplete {
			next.SetHelp("enter", "project")
			return screenHelp{
				short: []key.Binding{next, bindings.Home, bindings.Back, helpKey},
				full: [][]key.Binding{
					{next},
					{bindings.Home, bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
				},
			}
		}
		return screenHelp{
			short: []key.Binding{next, bindings.Home, bindings.Back, helpKey},
			full: [][]key.Binding{
				{next},
				{bindings.Home, bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenImprovementReview:
		option := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "option"))
		selectKey := bindings.Next
		selectKey.SetHelp("enter", "select")
		notes := bindings.Preview
		notes.SetHelp("p", "notes")
		return screenHelp{
			short: []key.Binding{option, selectKey, notes, bindings.Back, helpKey},
			full: [][]key.Binding{
				{option, selectKey, notes},
				{bindings.Home, bindings.Back, bindings.Quit, helpKey},
			},
		}
	case screenSkills:
		preview := bindings.OpenProject
		preview.SetHelp("enter", "preview")
		openFile := bindings.OpenReview
		openFile.SetHelp("o", "open file")
		create := bindings.NewProject
		create.SetHelp("n", "new skill")
		readiness := key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "readiness"))
		toggle := key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "disable/enable"))
		ground := bindings.Retry
		ground.SetHelp("g", "ground")
		deprecate := bindings.Remove
		deprecate.SetHelp("d", "deprecate")
		short := []key.Binding{bindings.Scroll, create, bindings.Back, helpKey}
		if item, ok := m.selectedSkill(); ok {
			short = []key.Binding{bindings.Scroll, preview, readiness, create, bindings.Back, helpKey}
			if item.Status == "needs grounding" || item.Status == "needs boundaries" {
				short = []key.Binding{bindings.Scroll, preview, readiness, ground, toggle, deprecate, bindings.Back, helpKey}
			} else if item.Status == "disabled" {
				short = []key.Binding{bindings.Scroll, preview, readiness, toggle, create, bindings.Back, helpKey}
			} else {
				short = []key.Binding{bindings.Scroll, preview, readiness, create, toggle, deprecate, bindings.Back, helpKey}
			}
		}
		return screenHelp{
			short: short,
			full:  [][]key.Binding{{bindings.Scroll, create, readiness, toggle, ground, deprecate, preview, openFile}, {bindings.Back, bindings.Quit, helpKey}},
		}
	case screenSkillReview:
		accept := bindings.Start
		accept.SetHelp("enter", "accept")
		openDraft := bindings.OpenReview
		openDraft.SetHelp("o", "open draft")
		discard := bindings.Remove
		discard.SetHelp("d", "discard")
		return screenHelp{
			short: []key.Binding{accept, openDraft, discard, bindings.Back, helpKey},
			full: [][]key.Binding{
				{accept, openDraft, discard},
				{bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenAudits:
		preview := bindings.OpenProject
		preview.SetHelp("enter", "preview")
		openFile := bindings.OpenReview
		openFile.SetHelp("o", "open file")
		runContradictions := bindings.Retry
		runContradictions.SetHelp("r", "contradictions")
		runSkills := bindings.Share
		runSkills.SetHelp("s", "skill audit")
		runSourceNotes := bindings.NewProject
		runSourceNotes.SetHelp("n", "source notes")
		cleanup := bindings.Compile
		cleanup.SetHelp("c", "note cleanup")
		contradictionCleanup := key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fix draft"))
		skillRepair := key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "repair skill"))
		cleanupPacket := key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "cleanup packet"))
		short := []key.Binding{bindings.Scroll, runContradictions, runSkills, runSourceNotes, bindings.Back, helpKey}
		if item, ok := m.selectedAudit(); ok {
			switch item.Type {
			case "contradiction":
				short = []key.Binding{bindings.Scroll, preview, cleanupPacket, contradictionCleanup, runContradictions, bindings.Back, helpKey}
			case "source notes":
				short = []key.Binding{bindings.Scroll, preview, cleanupPacket, cleanup, runSourceNotes, bindings.Back, helpKey}
			case "skill alignment":
				short = []key.Binding{bindings.Scroll, preview, cleanupPacket, skillRepair, runSkills, bindings.Back, helpKey}
			case "cleanup packet":
				short = []key.Binding{bindings.Scroll, preview, bindings.Back, helpKey}
			default:
				short = []key.Binding{bindings.Scroll, preview, cleanupPacket, runContradictions, runSkills, runSourceNotes, bindings.Back, helpKey}
			}
		}
		return screenHelp{
			short: short,
			full:  [][]key.Binding{{bindings.Scroll, runContradictions, runSkills, runSourceNotes, cleanup, contradictionCleanup, skillRepair, cleanupPacket, preview, openFile}, {bindings.Back, bindings.Quit, helpKey}},
		}
	case screenContradictionCleanupReview:
		apply := bindings.Start
		apply.SetHelp("enter", "apply")
		openDraft := bindings.OpenReview
		openDraft.SetHelp("o", "open draft")
		discard := bindings.Remove
		discard.SetHelp("d", "discard")
		return screenHelp{
			short: []key.Binding{apply, openDraft, discard, bindings.Back, helpKey},
			full: [][]key.Binding{
				{apply, openDraft, discard},
				{bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenSourceNoteCleanupReview:
		apply := bindings.Start
		apply.SetHelp("enter", "apply")
		openDraft := bindings.OpenReview
		openDraft.SetHelp("o", "open draft")
		discard := bindings.Remove
		discard.SetHelp("d", "discard")
		return screenHelp{
			short: []key.Binding{apply, openDraft, discard, bindings.Back, helpKey},
			full: [][]key.Binding{
				{apply, openDraft, discard},
				{bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenEvals:
		preview := bindings.OpenProject
		preview.SetHelp("enter", "preview")
		openFile := bindings.OpenReview
		openFile.SetHelp("o", "open file")
		taskset := bindings.NewProject
		taskset.SetHelp("t", "taskset")
		runPacket := bindings.Retry
		runPacket.SetHelp("r", "run packet")
		automation := bindings.AddSources
		automation.SetHelp("a", "runner packet")
		readiness := key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "readiness"))
		compare := bindings.Compile
		compare.SetHelp("c", "compare")
		judge := bindings.Share
		judge.SetHelp("j", "judge packet")
		short := []key.Binding{bindings.Scroll, preview, openFile, bindings.Back, helpKey}
		if item, ok := m.selectedEval(); ok {
			switch item.Area {
			case "taskset":
				short = []key.Binding{bindings.Scroll, preview, runPacket, taskset, bindings.Back, helpKey}
			case "run", "summary":
				short = []key.Binding{bindings.Scroll, preview, readiness, automation, compare, judge, bindings.Back, helpKey}
			case "comparison":
				short = []key.Binding{bindings.Scroll, preview, readiness, automation, judge, bindings.Back, helpKey}
			case "automation", "readiness":
				short = []key.Binding{bindings.Scroll, preview, readiness, compare, judge, bindings.Back, helpKey}
			case "judge":
				short = []key.Binding{bindings.Scroll, preview, readiness, compare, bindings.Back, helpKey}
			default:
				short = []key.Binding{bindings.Scroll, preview, taskset, bindings.Back, helpKey}
			}
		} else {
			short = []key.Binding{taskset, bindings.Back, helpKey}
		}
		return screenHelp{
			short: short,
			full:  [][]key.Binding{{bindings.Scroll, taskset, runPacket, readiness, automation, compare, judge, preview, openFile}, {bindings.Back, bindings.Quit, helpKey}},
		}
	case screenComposition:
		preview := bindings.OpenProject
		preview.SetHelp("enter", "preview")
		openFile := bindings.OpenReview
		openFile.SetHelp("o", "open file")
		nest := bindings.NewProject
		nest.SetHelp("n", "nest")
		resolveRoutes := key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "route draft"))
		merge := bindings.Compile
		merge.SetHelp("m", "merge draft")
		blend := bindings.AddSources
		blend.SetHelp("b", "blend LINER")
		skillConflicts := key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "skill conflicts"))
		copyPacket := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy packet"))
		copyApply := key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "copy snapshot"))
		productionMerge := key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "merge child"))
		routeAudit := bindings.Retry
		routeAudit.SetHelp("r", "route audit")
		promote := bindings.Share
		promote.SetHelp("p", "promote check")
		short := []key.Binding{bindings.Scroll, preview, openFile, bindings.Back, helpKey}
		if item, ok := m.selectedComposition(); ok && item.Kind != "lineage" {
			short = []key.Binding{bindings.Scroll, preview, promote, merge, blend, bindings.Back, helpKey}
		} else {
			short = []key.Binding{bindings.Scroll, nest, routeAudit, resolveRoutes, preview, bindings.Back, helpKey}
		}
		return screenHelp{
			short: short,
			full:  [][]key.Binding{{bindings.Scroll, nest, resolveRoutes, merge, blend, skillConflicts, copyPacket, copyApply, productionMerge, routeAudit, promote, preview, openFile}, {bindings.Back, bindings.Quit, helpKey}},
		}
	case screenCompositionReview:
		accept := bindings.Start
		accept.SetHelp("enter", "apply")
		openDraft := bindings.OpenReview
		openDraft.SetHelp("o", "open draft")
		discard := bindings.Remove
		discard.SetHelp("d", "discard")
		return screenHelp{
			short: []key.Binding{bindings.Scroll, accept, openDraft, discard, bindings.Back, helpKey},
			full: [][]key.Binding{
				{bindings.Scroll, accept, openDraft, discard},
				{bindings.Back, bindings.Quit, bindings.QuitKey, helpKey},
			},
		}
	case screenResearch:
		pause := bindings.Back
		pause.SetHelp("esc", "pause")
		if m.methodologyFailed {
			if m.methodologyLogScrollable() {
				return screenHelp{
					short: []key.Binding{bindings.Scroll, bindings.Retry, pause, helpKey},
					full:  [][]key.Binding{{bindings.Scroll, bindings.Retry, pause, bindings.Quit, helpKey}},
				}
			}
			return screenHelp{
				short: []key.Binding{bindings.Retry, pause, helpKey},
				full:  [][]key.Binding{{bindings.Retry, pause, bindings.Quit, helpKey}},
			}
		}
		if m.methodologyLogScrollable() {
			return screenHelp{
				short: []key.Binding{bindings.Scroll, pause, helpKey},
				full:  [][]key.Binding{{bindings.Scroll, pause, bindings.Quit, helpKey}},
			}
		}
		return screenHelp{
			short: []key.Binding{pause, helpKey},
			full:  [][]key.Binding{{pause, bindings.Quit, helpKey}},
		}
	case screenReport:
		next := bindings.OpenProject
		next.SetHelp("enter", "build corpus")
		return screenHelp{
			short: []key.Binding{next, bindings.Back, helpKey},
			full:  [][]key.Binding{{next, bindings.Back, bindings.Quit, helpKey}},
		}
	case screenBoard:
		compile := bindings.OpenProject
		compile.SetHelp("enter", "compile")
		return screenHelp{
			short: []key.Binding{bindings.Scroll, bindings.Toggle, compile, bindings.Back, helpKey},
			full:  [][]key.Binding{{bindings.Scroll, bindings.Toggle, compile}, {bindings.Back, bindings.Quit, helpKey}},
		}
	case screenProject:
		next := bindings.Next
		showNext := true
		if m.hasPendingAssemblyDraft() {
			next.SetHelp("enter", "review draft")
		} else if m.isProjectComplete() {
			next.SetHelp("enter", "LINER.md")
		} else if m.projectCompileNeedsAttention() {
			next.SetHelp("enter", "review compile")
		} else if m.hasCorpusReady() {
			next.SetHelp("enter", "operating layer")
		} else if m.needsClarificationBeforeMethodology() {
			next.SetHelp("enter", "clarify goal")
		} else if m.primaryProjectActionIsSourceEntry() {
			next.SetHelp("enter", "add sources")
		} else {
			next.SetHelp("enter", "corpus creation")
		}
		capabilities := m.projectCapabilities()
		sections := bindings.Scroll
		sections.SetHelp("↑/↓", "sections")
		short := []key.Binding{sections}
		firstRow := []key.Binding{sections}
		if showNext {
			short = append(short, next)
			firstRow = append(firstRow, next)
		}
		if capabilities.HasLiner {
			liner := bindings.Liner
			liner.SetHelp("l", "LINER.md")
			short = append(short, liner)
			firstRow = append(firstRow, liner)
			if m.canRegenerateOperatingLayer() {
				regenerate := bindings.Refresh
				regenerate.SetHelp("r", "regen layer")
				short = append(short, regenerate)
				firstRow = append(firstRow, regenerate)
			}
		} else if m.canCreateOperatingLayer() {
			liner := bindings.Liner
			liner.SetHelp("l", "create layer")
			short = append(short, liner)
			firstRow = append(firstRow, liner)
		}
		if !m.primaryProjectActionIsSourceEntry() {
			short = append(short, bindings.AddSources)
			firstRow = append(firstRow, bindings.AddSources)
		}
		openFolder := bindings.OpenReview
		openFolder.SetHelp("o", "open folder")
		short = append(short, openFolder)
		firstRow = append(firstRow, openFolder)
		if m.canShowManualCompileAction() {
			firstRow = append(firstRow, bindings.Compile)
		}
		short = append(short, bindings.Home, bindings.Back, bindings.Settings, bindings.QuitKey, helpKey)
		return screenHelp{
			short: short,
			full: [][]key.Binding{
				firstRow,
				{bindings.Home, bindings.Settings, bindings.Back, bindings.QuitKey, bindings.Quit, helpKey},
			},
		}
	case screenHome:
		run := bindings.OpenProject
		run.SetHelp("enter", "run")
		move := bindings.Scroll
		move.SetHelp("↑/↓", "choose")
		filter := key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
		return screenHelp{
			short: []key.Binding{run, move, filter, bindings.QuitKey, helpKey},
			full: [][]key.Binding{
				{run, move, filter},
				{bindings.NewProject, bindings.Import, bindings.Settings},
				{bindings.QuitKey, bindings.Quit, helpKey},
			},
		}
	case screenProjects:
		filter := key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
		if m.homeFiltering {
			typing := key.NewBinding(key.WithKeys("typing"), key.WithHelp("type", "filter"))
			done := bindings.OpenProject
			done.SetHelp("enter", "done")
			clear := bindings.Back
			clear.SetHelp("esc", "clear")
			return screenHelp{
				short: []key.Binding{typing, done, clear, bindings.QuitKey, helpKey},
				full:  [][]key.Binding{{typing, done, clear}, {bindings.QuitKey, bindings.Quit, helpKey}},
			}
		}
		return screenHelp{
			short: []key.Binding{bindings.OpenProject, filter, bindings.NewProject, bindings.Import, bindings.Settings, bindings.Back, bindings.Refresh, bindings.QuitKey, helpKey},
			full: [][]key.Binding{
				{bindings.OpenProject, filter, bindings.NewProject, bindings.Import, bindings.Settings, bindings.Refresh},
				{bindings.Home, bindings.Back, bindings.QuitKey, bindings.Quit, helpKey},
			},
		}
	case screenSettings:
		switchAgent := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "runner"))
		selectAgent := bindings.OpenProject
		selectAgent.SetHelp("enter", "select")
		setup := key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "setup"))
		return screenHelp{
			short: []key.Binding{switchAgent, selectAgent, setup, helpKey},
			full:  [][]key.Binding{{switchAgent, selectAgent, setup, bindings.Quit, helpKey}},
		}
	case screenOnboarding:
		if m.onboardingEditingDir {
			typing := key.NewBinding(key.WithKeys("typing"), key.WithHelp("type", "folder"))
			save := bindings.OpenProject
			save.SetHelp("enter", "save")
			cancel := bindings.Back
			cancel.SetHelp("esc", "cancel")
			return screenHelp{
				short: []key.Binding{typing, save, cancel, helpKey},
				full:  [][]key.Binding{{typing, save, cancel, bindings.Quit, helpKey}},
			}
		}
		next := bindings.OpenProject
		next.SetHelp("enter", "continue")
		if m.onboardingStep <= onboardingStepLibrary {
			editFolder := key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "change folder"))
			if m.settings.OnboardingCompleted {
				back := bindings.Back
				back.SetHelp("esc", "back")
				return screenHelp{
					short: []key.Binding{next, editFolder, back, bindings.QuitKey, helpKey},
					full:  [][]key.Binding{{next, editFolder, back, bindings.QuitKey, bindings.Quit, helpKey}},
				}
			}
			return screenHelp{
				short: []key.Binding{next, editFolder, bindings.QuitKey, helpKey},
				full:  [][]key.Binding{{next, editFolder, bindings.QuitKey, bindings.Quit, helpKey}},
			}
		}
		refresh := bindings.Refresh
		refresh.SetHelp("r", "refresh")
		back := bindings.Back
		back.SetHelp("esc", "library")
		switchAgent := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "runner"))
		if m.onboardingStep == onboardingStepProvider {
			next.SetHelp("enter", "continue")
			return screenHelp{
				short: []key.Binding{switchAgent, next, refresh, back, bindings.QuitKey, helpKey},
				full:  [][]key.Binding{{switchAgent, next, refresh}, {back, bindings.QuitKey, bindings.Quit, helpKey}},
			}
		}
		switchJS := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "option"))
		selectJS := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
		back.SetHelp("esc", "runner")
		if m.jsSetupRunning {
			return screenHelp{
				short: []key.Binding{helpKey},
				full:  [][]key.Binding{{helpKey}},
			}
		}
		return screenHelp{
			short: []key.Binding{switchJS, selectJS, back, bindings.QuitKey, helpKey},
			full:  [][]key.Binding{{switchJS, selectJS, back, bindings.QuitKey, bindings.Quit, helpKey}},
		}
	case screenImport:
		run := bindings.OpenProject
		run.SetHelp("enter", "import")
		move := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "choose"))
		folder := key.NewBinding(key.WithKeys("right", "left"), key.WithHelp("←/→", "folder"))
		return screenHelp{
			short: []key.Binding{move, folder, run, bindings.Refresh, bindings.Back, helpKey},
			full:  [][]key.Binding{{move, folder, run, bindings.Refresh}, {bindings.Home, bindings.Back, bindings.Quit, helpKey}},
		}
	case screenCreate:
		next := bindings.Next
		next.SetHelp("enter", "continue")
		field := key.NewBinding(key.WithKeys("tab", "up", "down"), key.WithHelp("tab/↑/↓", "field"))
		short := []key.Binding{field, next, bindings.Back, helpKey}
		full := [][]key.Binding{{field, next, bindings.Back, bindings.Quit, helpKey}}
		if m.createStep == 3 {
			choice := key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "switch"))
			short = []key.Binding{field, choice, next, bindings.Back, helpKey}
			full = [][]key.Binding{{field, choice, next, bindings.Back, bindings.Quit, helpKey}}
		}
		return screenHelp{
			short: short,
			full:  full,
		}
	case screenClarify:
		if m.clarifyLoading {
			return screenHelp{
				short: []key.Binding{bindings.Back, helpKey},
				full:  [][]key.Binding{{bindings.Back, bindings.Quit, helpKey}},
			}
		}
		next := bindings.Next
		if len(m.clarifyQuestions) == 0 {
			next.SetHelp("enter", "retry")
			return screenHelp{
				short: []key.Binding{next, bindings.Back, helpKey},
				full:  [][]key.Binding{{next, bindings.Back, bindings.Quit, helpKey}},
			}
		}
		next.SetHelp("enter", "continue")
		field := key.NewBinding(key.WithKeys("tab", "up", "down"), key.WithHelp("tab/↑/↓", "field"))
		return screenHelp{
			short: []key.Binding{field, next, bindings.Back, helpKey},
			full:  [][]key.Binding{{field, next, bindings.Back, bindings.Quit, helpKey}},
		}
	case screenCompile:
		if m.sourceRecoveryRunning {
			wait := key.NewBinding(key.WithKeys(""), key.WithHelp("wait", "source retry"))
			return screenHelp{
				short: []key.Binding{wait, helpKey},
				full:  [][]key.Binding{{wait, bindings.Quit, helpKey}},
			}
		}
		if m.sourceRecoveryReview {
			next := bindings.Next
			if m.compileRepairRebuildCorpusAfterRecovery {
				next.SetHelp("enter", "refresh eval")
			} else if m.compileRepairAttempted && m.sourceRecovery != nil && m.sourceRecovery.Succeeded == 0 {
				next.SetHelp("enter", "view sources")
			} else {
				next.SetHelp("enter", "continue")
			}
			return screenHelp{
				short: []key.Binding{next, helpKey},
				full:  [][]key.Binding{{next, bindings.Quit, helpKey}},
			}
		}
		next := bindings.Next
		switch {
		case m.compilePane == compilePaneIssues && m.compileHasUsableResult() && m.compileHasSourceReviewItems():
			next.SetHelp("enter", "view sources")
		case m.compilePane == compilePaneSources && !m.compileRepairAttempted && m.compileHasRepairableSources():
			next.SetHelp("enter", "repair")
		default:
			next.SetHelp("enter", "create layer")
		}
		addSources := bindings.AddSources
		addSources.SetHelp("a", "add sources")
		logs := key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "logs"))
		openSource := key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open source"))
		dropSource := key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "drop source"))
		installJS := key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install JS"))
		warningScroll := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "issue"))
		if m.compilePane == compilePaneSources {
			warningScroll.SetHelp("↑/↓", "source")
		}
		retry := bindings.Retry
		if m.compileHasRepairableSources() {
			retry.SetHelp("r", "repair sources")
		} else {
			retry.SetHelp("r", "retry compile")
		}
		previewKey := bindings.Preview
		short := []key.Binding{retry, bindings.Preview, bindings.Copy, addSources, bindings.Back, helpKey}
		full := [][]key.Binding{{retry, bindings.Preview, bindings.Copy}, {addSources, bindings.Back, bindings.Quit, helpKey}}
		if m.compileHasUsableResult() {
			short = []key.Binding{next, previewKey, retry, bindings.Copy, addSources, bindings.Back, helpKey}
			full = [][]key.Binding{{next, previewKey, retry, bindings.Copy}, {addSources, bindings.Back, bindings.Quit, helpKey}}
		}
		if len(m.compileLines) > 0 {
			short = []key.Binding{retry, bindings.Preview, logs, bindings.Copy, bindings.Back, helpKey}
			full = [][]key.Binding{{retry, bindings.Preview, logs, bindings.Copy}, {addSources, bindings.Back, bindings.Quit, helpKey}}
			if m.compileHasUsableResult() {
				short = []key.Binding{next, previewKey, retry, logs, bindings.Copy, bindings.Back, helpKey}
				full = [][]key.Binding{{next, previewKey, retry, logs, bindings.Copy}, {addSources, bindings.Back, bindings.Quit, helpKey}}
			}
		}
		if m.compileResult != nil && m.actionableCompileWarningCount() > 0 {
			warningsBlockNext := !m.compileHasUsableResult()
			warningFirstRow := []key.Binding{next, warningScroll, openSource, previewKey, retry}
			short = []key.Binding{next, warningScroll, openSource, previewKey, retry, bindings.Back, helpKey}
			if warningsBlockNext {
				short = []key.Binding{warningScroll, openSource, dropSource, addSources, previewKey, retry, bindings.Back, helpKey}
				warningFirstRow = []key.Binding{warningScroll, openSource, dropSource, addSources, previewKey, retry}
				if m.compileNeedsJSSetup() && !m.jsSetupRunning {
					short = []key.Binding{warningScroll, installJS, openSource, dropSource, addSources, previewKey, retry, bindings.Back, helpKey}
					warningFirstRow = []key.Binding{warningScroll, installJS, openSource, dropSource, addSources, previewKey, retry}
				}
			}
			if len(m.compileLines) > 0 {
				short = []key.Binding{next, warningScroll, openSource, previewKey, retry, logs, bindings.Back, helpKey}
				warningFirstRow = append(warningFirstRow, logs)
				if warningsBlockNext {
					short = []key.Binding{warningScroll, openSource, dropSource, addSources, previewKey, retry, logs, bindings.Back, helpKey}
					warningFirstRow = []key.Binding{warningScroll, openSource, dropSource, addSources, previewKey, retry, logs}
					if m.compileNeedsJSSetup() && !m.jsSetupRunning {
						short = []key.Binding{warningScroll, installJS, openSource, dropSource, addSources, previewKey, retry, logs, bindings.Back, helpKey}
						warningFirstRow = []key.Binding{warningScroll, installJS, openSource, dropSource, addSources, previewKey, retry, logs}
					}
				}
			}
			full = [][]key.Binding{warningFirstRow, {dropSource, addSources, bindings.Copy, bindings.Back, bindings.Quit, helpKey}}
		}
		if m.compileNeedsJSSetup() && !m.jsSetupRunning {
			short = insertHelpBinding(short, installJS, openSource)
			if len(full) > 0 {
				full[0] = insertHelpBinding(full[0], installJS, openSource)
			}
		}
		short = insertHelpBinding(short, addSources, bindings.Back)
		if len(full) > 0 {
			last := len(full) - 1
			full[last] = insertHelpBinding(full[last], addSources, bindings.Back)
		}
		return screenHelp{
			short: short,
			full:  full,
		}
	case screenPreview:
		if m.previewRel != "" && m.previewRel != "MIXTAPE.md" {
			return screenHelp{
				short: []key.Binding{bindings.Scroll, bindings.OpenFolder, bindings.Back, helpKey},
				full:  [][]key.Binding{{bindings.Scroll, bindings.OpenFolder}, {bindings.Back, bindings.Quit, helpKey}},
			}
		}
		return screenHelp{
			short: []key.Binding{bindings.Scroll, bindings.Copy, bindings.OpenFolder, bindings.Back, helpKey},
			full:  [][]key.Binding{{bindings.Scroll, bindings.Copy, bindings.OpenFolder}, {bindings.Back, bindings.Quit, helpKey}},
		}
	default:
		return screenHelp{
			short: []key.Binding{bindings.Back, bindings.QuitKey, helpKey},
			full:  [][]key.Binding{{bindings.Back, bindings.QuitKey, bindings.Quit, helpKey}},
		}
	}
}
