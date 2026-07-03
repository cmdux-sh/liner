package app

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

const (
	onboardingStepLibrary = iota
	onboardingStepProvider
	onboardingStepJS
)

const (
	onboardingJSOptionInstall = iota
	onboardingJSOptionSkip
)

var onboardingJSOptions = []choiceOption{
	{Label: "Install", Detail: "Download Playwright Chromium (about 150 MB on first run)."},
	{Label: "Skip", Detail: "Skip for now. Liner will offer this again if a source needs browser rendering."},
}

const (
	codexCLIDocsURL      = "https://developers.openai.com/codex/cli"
	claudeCodeDocsURL    = "https://docs.anthropic.com/en/docs/claude-code/quickstart"
	onboardingConfigDone = true
)

func newOnboardingDirInput(value string, width int) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "~/liner/projects"
	input.SetValue(normalizeProjectDir(value))
	input.SetWidth(onboardingDirInputWidth(width))
	input.Focus()
	inputStyles := textinput.DefaultDarkStyles()
	inputStyles.Focused.Text = styles.InputFocused
	inputStyles.Focused.Placeholder = styles.InputPlaceholder
	inputStyles.Focused.Prompt = styles.InputFocused
	inputStyles.Blurred.Text = styles.InputBlurred
	inputStyles.Blurred.Placeholder = styles.InputPlaceholder
	inputStyles.Cursor.Color = styles.InputCursor
	inputStyles.Cursor.Shape = tea.CursorBar
	input.SetStyles(inputStyles)
	return input
}

func (m Model) viewOnboarding() string {
	if m.onboardingStep <= onboardingStepLibrary {
		return m.viewOnboardingLibrary()
	}
	if m.onboardingStep == onboardingStepProvider {
		return m.viewOnboardingProvider()
	}
	return m.viewOnboardingJS()
}

func (m Model) viewOnboardingLibrary() string {
	width := styles.ClampWidth(m.width - 4)
	copy := strings.Join(wrapWords("Liner keeps projects in a visible local folder so they're easy to inspect, move, back up, and use outside the current terminal folder.", width), "\n")
	rows := []labelValueRow{
		{Label: "Projects folder", Value: m.baseDir},
		{Label: "Config", Value: settingsConfigPath()},
	}
	parts := []string{
		styles.Title.Render("Setup"),
		styles.Subtitle.Render("Set up Liner's local projects folder."),
		"",
		styles.Subtitle.Render(copy),
		renderLabelValueBlock(width, rows, 2, 1),
	}
	if m.onboardingEditingDir {
		parts = append(parts,
			styles.Section.Render("Projects folder"),
			m.onboardingDirInput.View(),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) viewOnboardingProvider() string {
	width := styles.ClampWidth(m.width - 4)
	info := m.cachedSettingsInfo()
	parts := []string{
		styles.Title.Render("Setup"),
		styles.Subtitle.Render("Choose the AI runner Liner uses to research sources and create project files."),
		"",
		styles.PrimaryText.Render(onboardingProviderSummary(info)),
		"",
		settingsProviderSelectorView(m.onboardingProviderCursor),
		settingsProviderDetailsView(width, m.onboardingProviderCursor, info),
	}
	if len(info.Installed) == 0 {
		parts = append(parts,
			styles.Section.Render("Install one AI runner, then refresh."),
			styles.Subtitle.Render("Codex CLI: "+codexCLIDocsURL),
			styles.Subtitle.Render("Claude Code quickstart: "+claudeCodeDocsURL),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) viewOnboardingJS() string {
	width := styles.ClampWidth(m.width - 4)
	copy := strings.Join(wrapWords("Some web sources only reveal useful text after JavaScript runs. Liner can install Playwright's headless Chromium so compile can recover those pages instead of keeping tiny stubs.", width), "\n")
	parts := []string{
		m.renderLoadingTitle("Setup", m.jsSetupRunning),
		styles.Subtitle.Render("Set up browser-backed source extraction."),
		"",
		styles.PrimaryText.Render(copy),
	}
	if m.jsSetupRunning {
		parts = append(parts, "", renderWaitStatusBlock(width, "Installing", "Downloading Playwright Chromium. First run can take a few minutes.", "browser setup in progress"))
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}
	if m.settings.JSSetupCompleted {
		parts = append(parts, "", styles.ReportSection.Render("Ready")+"  "+styles.NextActionText.Render("JS rendering support is already marked ready."))
	} else {
		parts = append(parts,
			"",
			renderChoiceSelector(onboardingJSOptions, m.onboardingJSCursor),
			renderChoiceDetail(width, onboardingJSOptions, m.onboardingJSCursor),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) startOnboarding() Model {
	if m.screen != screenOnboarding {
		m.previous = m.screen
	}
	settings := readSettingsInfo()
	if settings.ProjectsDir != "" {
		m.baseDir = settings.ProjectsDir
	}
	m.settings = settings
	m.screen = screenOnboarding
	m.onboardingStep = onboardingStepLibrary
	m.onboardingEditingDir = false
	m.onboardingDirInput = newOnboardingDirInput(m.baseDir, m.width)
	m.onboardingProviderCursor = settingsProviderIndex(onboardingDefaultProvider(settings))
	m.onboardingJSCursor = onboardingJSOptionInstall
	m.err = ""
	m.note = ""
	return m
}

func onboardingProviderSummary(info settingsInfo) string {
	switch len(info.Installed) {
	case 0:
		return "No Codex or Claude Code runner found."
	case 1:
		return settingsProviderName(info.Installed[0]) + " installed; it will be active."
	default:
		return "Claude Code and Codex are installed; choose one."
	}
}

func (m Model) selectedOnboardingAgent() string {
	return m.onboardingAgentForSave(m.settings)
}

func (m Model) onboardingAgentForSave(info settingsInfo) string {
	if len(info.Installed) == 0 {
		return ""
	}
	return settingsProviderAt(m.onboardingProviderCursor)
}

func onboardingDefaultProvider(info settingsInfo) string {
	if active := info.activeAgent(); active != "" {
		return active
	}
	return info.preferredAgent()
}

func onboardingJSOptionAt(cursor int) int {
	if cursor == onboardingJSOptionSkip {
		return onboardingJSOptionSkip
	}
	return onboardingJSOptionInstall
}

func (m Model) handleOnboardingKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.onboardingEditingDir {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.onboardingEditingDir = false
			m.onboardingDirInput.SetValue(m.baseDir)
			m.err = ""
			return m, nil
		case "enter":
			return m.saveOnboardingProjectLibrary()
		}
		return m, nil
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	if m.onboardingStep <= onboardingStepLibrary {
		switch keyMsg.String() {
		case "esc":
			if m.settings.OnboardingCompleted {
				if m.previous != screenOnboarding {
					m.screen = m.previous
				} else {
					m.screen = screenHome
				}
			}
			return m, nil
		case "enter":
			return m.continueOnboardingFromLibrary()
		case "e":
			m.onboardingEditingDir = true
			m.onboardingDirInput.SetValue(m.baseDir)
			m.onboardingDirInput.Focus()
			return m, nil
		}
		return m, nil
	}

	if m.onboardingStep == onboardingStepProvider {
		switch keyMsg.String() {
		case "esc":
			m.onboardingStep = onboardingStepLibrary
			return m, nil
		case "r":
			return m.refreshOnboardingProviders(), nil
		case "enter":
			return m.continueOnboardingFromProvider()
		case "shift+tab", "up", "left":
			return m.moveOnboardingProvider(-1), nil
		case "tab", "down", "right":
			return m.moveOnboardingProvider(1), nil
		}
		switch keyMsg.Key().Code {
		case tea.KeyEsc:
			m.onboardingStep = onboardingStepLibrary
			return m, nil
		case tea.KeyLeft, tea.KeyUp:
			return m.moveOnboardingProvider(-1), nil
		case tea.KeyRight, tea.KeyDown, tea.KeyTab:
			return m.moveOnboardingProvider(1), nil
		case tea.KeyEnter:
			return m.continueOnboardingFromProvider()
		}
		return m, nil
	}
	if m.jsSetupRunning {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.onboardingStep = onboardingStepProvider
		return m, nil
	case "enter":
		return m.applyOnboardingJSOption()
	case "shift+tab", "up", "left":
		return m.moveOnboardingJSOption(-1), nil
	case "tab", "down", "right":
		return m.moveOnboardingJSOption(1), nil
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		m.onboardingStep = onboardingStepProvider
		return m, nil
	case tea.KeyLeft, tea.KeyUp:
		return m.moveOnboardingJSOption(-1), nil
	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		return m.moveOnboardingJSOption(1), nil
	case tea.KeyEnter:
		return m.applyOnboardingJSOption()
	}
	return m, nil
}

func (m Model) saveOnboardingProjectLibrary() (Model, tea.Cmd) {
	dir := normalizeProjectDir(m.onboardingDirInput.Value())
	if dir == "" {
		m.err = "Projects folder is required."
		return m, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.err = "Could not create projects folder: " + err.Error()
		return m, nil
	}
	m.baseDir = dir
	m.onboardingDirInput.SetValue(dir)
	m.onboardingEditingDir = false
	m.note = "Projects folder set to " + dir + "."
	return m, loadProjects(m.runner, m.baseDir)
}

func (m Model) continueOnboardingFromLibrary() (Model, tea.Cmd) {
	dir := normalizeProjectDir(m.baseDir)
	if dir == "" {
		m.err = "Projects folder is required."
		return m, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.err = "Could not create projects folder: " + err.Error()
		return m, nil
	}
	m.baseDir = dir
	m.onboardingDirInput.SetValue(dir)
	m.onboardingStep = onboardingStepProvider
	m.settings = readSettingsInfo()
	m.onboardingProviderCursor = settingsProviderIndex(onboardingDefaultProvider(m.settings))
	m.note = ""
	return m, loadProjects(m.runner, m.baseDir)
}

func (m Model) continueOnboardingFromProvider() (Model, tea.Cmd) {
	info := readSettingsInfo()
	m.settings = info
	agent := m.onboardingAgentForSave(info)
	if agent != "" && !containsAgentID(info.Installed, agent) {
		m.err = settingsProviderInstallMessage(agent)
		return m, nil
	}
	if info.OnboardingCompleted && info.JSSetupPrompted {
		return m.finishOnboarding()
	}
	m.onboardingStep = onboardingStepJS
	m.onboardingJSCursor = onboardingJSOptionInstall
	m.note = ""
	m.err = ""
	return m, nil
}

func (m Model) finishOnboarding() (Model, tea.Cmd) {
	info := readSettingsInfo()
	m.settings = info
	agent := m.onboardingAgentForSave(info)
	if agent != "" && !containsAgentID(info.Installed, agent) {
		m.err = settingsProviderInstallMessage(agent)
		return m, nil
	}
	if err := writeOnboardingConfig(m.baseDir, agent, onboardingConfigDone); err != nil {
		m.err = "Could not save onboarding config: " + err.Error()
		return m, nil
	}
	m.settings = readSettingsInfo()
	m.settingsCursor = settingsProviderIndex(m.settings.preferredAgent())
	m.commands.SetItems(m.commandItems())
	m.screen = screenHome
	if agent == "" {
		m.note = "Onboarding saved. Install Claude Code or Codex CLI, then refresh Settings."
	} else {
		m.note = "Onboarding saved. " + settingsProviderName(agent) + " is active."
	}
	return m, loadProjects(m.runner, m.baseDir)
}

func (m Model) startJSSetupForOnboarding() (Model, tea.Cmd) {
	m.jsSetupRunning = true
	m.jsSetupFromOnboarding = true
	m.jsSetupRetryCompile = false
	m.note = "Installing JS rendering support."
	m.err = ""
	return m, setupJS(m.runner)
}

func (m Model) applyOnboardingJSOption() (Model, tea.Cmd) {
	if onboardingJSOptionAt(m.onboardingJSCursor) == onboardingJSOptionSkip {
		return m.skipOnboardingJSSetup()
	}
	return m.startJSSetupForOnboarding()
}

func (m Model) skipOnboardingJSSetup() (Model, tea.Cmd) {
	if err := writeJSSetupConfig(true, false); err != nil {
		m.err = "Could not save JS setup preference: " + err.Error()
		return m, nil
	}
	next, cmd := m.finishOnboarding()
	next.note = "Onboarding saved. JS rendering skipped; compile can offer it later if needed."
	return next, cmd
}

func (m Model) refreshOnboardingProviders() Model {
	m.settings = readSettingsInfo()
	m.onboardingProviderCursor = settingsProviderIndex(onboardingDefaultProvider(m.settings))
	m.note = "Runner check refreshed."
	m.err = ""
	return m
}

func (m Model) moveOnboardingProvider(delta int) Model {
	count := len(settingsProviders)
	if count == 0 {
		return m
	}
	m.onboardingProviderCursor = (m.onboardingProviderCursor + delta + count) % count
	m.err = ""
	return m
}

func (m Model) moveOnboardingJSOption(delta int) Model {
	count := len(onboardingJSOptions)
	if count == 0 {
		return m
	}
	m.onboardingJSCursor = (m.onboardingJSCursor + delta + count) % count
	m.err = ""
	return m
}

func shouldEditOnboardingDir(keyMsg tea.KeyPressMsg) bool {
	switch keyMsg.String() {
	case "enter", "esc", "ctrl+c":
		return false
	default:
		return true
	}
}

func onboardingDirInputWidth(width int) int {
	if width <= 0 {
		return 80
	}
	return max(18, styles.ClampWidth(width-4))
}
