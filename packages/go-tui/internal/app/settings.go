package app

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/airunner"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"gopkg.in/yaml.v3"
)

type settingsInfo struct {
	ConfigPath          string
	ConfigAgent         string
	ConfigExecutable    string
	ConfigHome          string
	EnvAgent            string
	ProjectsDir         string
	OnboardingCompleted bool
	JSSetupPrompted     bool
	JSSetupCompleted    bool
	Installed           []string
	ProviderModels      map[string]string
	ProviderModelModes  map[string]string
	ProviderEfforts     map[string]string
}

var settingsProviders = []string{"codex", "claude"}

type settingsPane int

const (
	settingsPaneMenu settingsPane = iota
	settingsPaneProjectsFolder
	settingsPaneAIRunner
)

const (
	settingsMenuProjectsFolder = iota
	settingsMenuAIRunner
	settingsMenuCount
)

func newSettingsModelInput() textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "provider model ID"
	input.SetWidth(64)
	input.Blur()
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

func (m Model) startSettings() Model {
	if m.screen != screenSettings {
		m.previous = m.screen
	}
	m.settings = readSettingsInfo()
	m.settingsPane = settingsPaneMenu
	m.settingsMenuCursor = settingsMenuProjectsFolder
	m.prepareSettingsAIRunner()
	m.settingsCustomEditing = false
	m.settingsInput.Blur()
	m.screen = screenSettings
	m.err = ""
	return m
}

func (m *Model) prepareSettingsAIRunner() {
	m.settingsCursor = settingsProviderIndex(m.settings.preferredAgent())
	m.settingsRow = 0
	provider := settingsProviderAt(m.settingsCursor)
	m.settingsModelCursor = settingsModelIndex(provider, m.settings.providerModel(provider), m.settings.providerModelMode(provider))
	m.settingsEffortCursor = settingsEffortIndex(m.settings.providerEffort("codex"))
}

func (m Model) openSettingsAIRunner() Model {
	m.settings = readSettingsInfo()
	m.prepareSettingsAIRunner()
	m.settingsPane = settingsPaneAIRunner
	m.settingsCustomEditing = false
	m.settingsInput.Blur()
	m.err = ""
	return m
}

func (m Model) openSettingsProjectsFolder() Model {
	m.settings = readSettingsInfo()
	m.settingsPane = settingsPaneProjectsFolder
	m.settingsCustomEditing = false
	projectsDir := m.settings.ProjectsDir
	if projectsDir == "" {
		projectsDir = m.baseDir
	}
	m.settingsInput.SetValue(projectsDir)
	m.settingsInput.Placeholder = "absolute projects folder path"
	m.settingsInput.Focus()
	m.err = ""
	return m
}

func (m Model) handleSettingsKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.settingsPane == settingsPaneMenu {
		return m.handleSettingsMenuKey(keyMsg)
	}
	if m.settingsPane == settingsPaneProjectsFolder {
		return m.handleSettingsProjectsFolderKey(keyMsg)
	}
	if m.settingsCustomEditing {
		switch keyMsg.Key().Code {
		case tea.KeyEsc:
			m.settingsCustomEditing = false
			m.settingsInput.Blur()
			m.err = ""
			return m, nil
		case tea.KeyEnter:
			return m.saveSettingsCustomModel(), nil
		}
		return m, nil
	}
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		return m.showCommandHome(), nil
	case "s":
		return m.startOnboarding(), nil
	case "shift+tab":
		m.settingsRow = max(0, m.settingsRow-1)
		return m, nil
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		return m.returnToSettingsMenu(), nil
	case tea.KeyTab:
		m.settingsRow = min(settingsRowCount(settingsProviderAt(m.settingsCursor))-1, m.settingsRow+1)
		m.err = ""
		return m, nil
	case tea.KeyUp:
		return m.moveSettingsValue(-1), nil
	case tea.KeyDown:
		return m.moveSettingsValue(1), nil
	case tea.KeyLeft:
		m.settingsRow = max(0, m.settingsRow-1)
		m.err = ""
		return m, nil
	case tea.KeyRight:
		m.settingsRow = min(settingsRowCount(settingsProviderAt(m.settingsCursor))-1, m.settingsRow+1)
		m.err = ""
		return m, nil
	case tea.KeyEnter:
		return m.saveSettingsSelection(), nil
	}
	switch keyMsg.String() {
	case "up", "↑":
		return m.moveSettingsValue(-1), nil
	case "down", "↓":
		return m.moveSettingsValue(1), nil
	case "left", "←":
		m.settingsRow = max(0, m.settingsRow-1)
		return m, nil
	case "right", "→":
		m.settingsRow = min(settingsRowCount(settingsProviderAt(m.settingsCursor))-1, m.settingsRow+1)
		return m, nil
	}
	return m, nil
}

func (m Model) handleSettingsMenuKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		return m.showCommandHome(), nil
	case "up", "↑":
		return m.moveSettingsMenuCursor(-1), nil
	case "down", "↓":
		return m.moveSettingsMenuCursor(1), nil
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		return m.closeSettings(), nil
	case tea.KeyUp:
		return m.moveSettingsMenuCursor(-1), nil
	case tea.KeyDown:
		return m.moveSettingsMenuCursor(1), nil
	case tea.KeyEnter:
		if m.settingsMenuCursor == settingsMenuAIRunner {
			return m.openSettingsAIRunner(), nil
		}
		return m.openSettingsProjectsFolder(), nil
	}
	return m, nil
}

func (m Model) moveSettingsMenuCursor(delta int) Model {
	m.settingsMenuCursor = (m.settingsMenuCursor + delta + settingsMenuCount) % settingsMenuCount
	m.err = ""
	return m
}

func (m Model) handleSettingsProjectsFolderKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		return m.returnToSettingsMenu(), nil
	case tea.KeyEnter:
		return m.saveSettingsProjectsFolder()
	}
	return m, nil
}

func (m Model) returnToSettingsMenu() Model {
	m.settings = readSettingsInfo()
	m.settingsPane = settingsPaneMenu
	m.settingsCustomEditing = false
	m.settingsInput.Blur()
	m.err = ""
	return m
}

func (m Model) moveSettingsValue(delta int) Model {
	if m.settingsRow == 0 {
		return m.moveSettingsProvider(delta)
	}
	if m.settingsRow == 2 {
		options := airunner.ReasoningEffortOptions("codex")
		m.settingsEffortCursor = (m.settingsEffortCursor + delta + len(options)) % len(options)
		m.err = ""
		return m
	}
	options := airunner.ModelOptions(settingsProviderAt(m.settingsCursor))
	if len(options) > 0 {
		m.settingsModelCursor = (m.settingsModelCursor + delta + len(options)) % len(options)
		if settingsProviderAt(m.settingsCursor) == "codex" && (options[m.settingsModelCursor].Auto || options[m.settingsModelCursor].Custom) {
			m.settingsEffortCursor = 0
		}
	}
	m.err = ""
	return m
}

func (m Model) closeSettings() Model {
	if m.previous != screenSettings {
		m.screen = m.previous
	} else if m.currentPath != "" {
		m.screen = screenProject
	} else {
		m = m.showCommandHome()
	}
	return m
}

func (m Model) moveSettingsProvider(delta int) Model {
	count := len(settingsProviders)
	if count == 0 {
		return m
	}
	m.settingsCursor = (m.settingsCursor + delta + count) % count
	provider := settingsProviderAt(m.settingsCursor)
	m.settingsModelCursor = settingsModelIndex(provider, m.settings.providerModel(provider), m.settings.providerModelMode(provider))
	if m.settingsRow >= settingsRowCount(provider) {
		m.settingsRow = settingsRowCount(provider) - 1
	}
	m.err = ""
	return m
}

func (m Model) saveSettingsSelection() Model {
	provider := settingsProviderAt(m.settingsCursor)
	if !containsAgentID(readSettingsInfo().Installed, provider) {
		m.err = settingsProviderInstallMessage(provider)
		return m
	}
	options := airunner.ModelOptions(provider)
	if m.settingsModelCursor < 0 || m.settingsModelCursor >= len(options) {
		m.settingsModelCursor = 0
	}
	option := options[m.settingsModelCursor]
	model := option.Value
	modelMode := ""
	if provider == "codex" {
		switch {
		case option.Auto:
			modelMode = "auto"
		case option.Value == "" && !option.Custom:
			modelMode = "default"
		}
	}
	if option.Custom {
		model = m.settings.providerModel(provider)
		if m.settingsRow == 1 || model == "" || airunner.IsCuratedModel(provider, model) {
			return m.beginSettingsCustomModel()
		}
	}
	effort := ""
	if provider == "codex" {
		effortOptions := airunner.ReasoningEffortOptions("codex")
		if m.settingsEffortCursor < 0 || m.settingsEffortCursor >= len(effortOptions) {
			m.settingsEffortCursor = 0
		}
		effort = effortOptions[m.settingsEffortCursor].Value
	}
	return m.commitSettingsSelection(provider, model, modelMode, effort)
}

func (m Model) beginSettingsCustomModel() Model {
	provider := settingsProviderAt(m.settingsCursor)
	m.settingsCustomEditing = true
	m.settingsInput.SetValue("")
	if current := m.settings.providerModel(provider); current != "" && !airunner.IsCuratedModel(provider, current) {
		m.settingsInput.SetValue(current)
	}
	m.settingsInput.Placeholder = "provider model ID"
	m.settingsInput.Focus()
	m.err = ""
	return m
}

func (m Model) commitSettingsSelection(provider string, model string, modelMode string, effort string) Model {
	if err := writeSettingsSelection(provider, model, modelMode, effort); err != nil {
		m.err = "Could not save Settings: " + err.Error()
		return m
	}
	m.settings = readSettingsInfo()
	m.settingsCursor = settingsProviderIndex(provider)
	m.settingsModelCursor = settingsModelIndex(provider, model, m.settings.providerModelMode(provider))
	m.settingsEffortCursor = settingsEffortIndex(m.settings.providerEffort("codex"))
	m.settingsCustomEditing = false
	m.settingsInput.Blur()
	modelLabel := airunner.ModelLabel(provider, model)
	if modelMode == "auto" {
		modelLabel = "Auto"
	}
	m.note = "Saved " + settingsProviderName(provider) + " with " + modelLabel
	if provider == "codex" {
		effortLabel := airunner.ReasoningEffortLabel(effort)
		if modelMode == "auto" && effort == "" {
			effortLabel = "Auto by task"
		}
		m.note += " and " + effortLabel + " Thinking effort"
	}
	m.note += ". New runs use these preferences."
	m.err = ""
	return m.returnToSettingsMenu()
}

func (m Model) saveSettingsProjectsFolder() (Model, tea.Cmd) {
	dir := normalizeProjectDir(m.settingsInput.Value())
	if dir == "" {
		m.err = "Projects folder is required."
		return m, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.err = "Could not create projects folder: " + err.Error()
		return m, nil
	}
	if err := writeSettingsProjectsFolder(dir); err != nil {
		m.err = "Could not save projects folder: " + err.Error()
		return m, nil
	}
	m.baseDir = dir
	m.settings = readSettingsInfo()
	m.note = "Projects folder updated."
	m.err = ""
	m = m.returnToSettingsMenu()
	return m, loadProjects(m.runner, m.baseDir)
}

func (m Model) saveSettingsCustomModel() Model {
	provider := settingsProviderAt(m.settingsCursor)
	model := strings.TrimSpace(m.settingsInput.Value())
	if model == "" {
		m.err = "Custom model ID cannot be blank."
		return m
	}
	effort := ""
	if provider == "codex" {
		effortOptions := airunner.ReasoningEffortOptions("codex")
		if m.settingsEffortCursor >= 0 && m.settingsEffortCursor < len(effortOptions) {
			effort = effortOptions[m.settingsEffortCursor].Value
		}
	}
	return m.commitSettingsSelection(provider, model, "", effort)
}

func (m Model) viewSettings() string {
	switch m.settingsPane {
	case settingsPaneProjectsFolder:
		return m.viewSettingsProjectsFolder()
	case settingsPaneAIRunner:
		return m.viewSettingsAIRunner()
	default:
		return m.viewSettingsMenu()
	}
}

func (m Model) viewSettingsMenu() string {
	width := styles.ClampWidth(m.width - 4)
	info := m.cachedSettingsInfo()
	rows := []struct {
		label  string
		detail string
	}{
		{label: "Projects folder", detail: settingsProjectsFolderSummary(info)},
		{label: "AI runner", detail: settingsAIRunnerSummary(info)},
	}
	lines := []string{
		styles.Title.Render("Settings"),
		styles.PrimaryText.Render("Choose app-level preferences."),
		"",
	}
	for index, row := range rows {
		prefix := "  "
		labelStyle := styles.MutedText
		if index == m.settingsMenuCursor {
			prefix = "> "
			labelStyle = styles.AccentText
		}
		lines = append(lines, labelStyle.Render(prefix+row.label))
		lines = append(lines, styles.Subtitle.Render("  "+truncateMiddle(row.detail, max(20, width-2))))
		if index < len(rows)-1 {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewSettingsProjectsFolder() string {
	width := styles.ClampWidth(m.width - 4)
	description := "Choose where Liner finds and creates projects. Saving creates the folder if needed. Existing projects are not moved."
	return joinNonEmpty(
		"\n",
		styles.Title.Render("Projects folder"),
		styles.PrimaryText.Render(strings.Join(wrapWords(description, width), "\n")),
		"",
		m.settingsInput.View(),
	)
}

func (m Model) viewSettingsAIRunner() string {
	width := styles.ClampWidth(m.width - 4)
	info := m.cachedSettingsInfo()
	description := strings.Join(wrapWords("Choose the AI runner Liner uses to research sources and create project files.", width), "\n")
	provider := settingsProviderAt(m.settingsCursor)
	parts := []string{
		styles.Title.Render("AI runner"),
		styles.PrimaryText.Render(description),
		"",
		settingsColumnsView(width, provider, m.settingsRow, m.settingsCursor, m.settingsModelCursor, m.settingsEffortCursor, m.settings),
	}
	if m.settingsCustomEditing {
		parts = append(parts, "", styles.Section.Render("Custom model ID"), m.settingsInput.View())
	}
	parts = append(parts, "", settingsFocusedChoiceDetail(width, provider, m.settingsRow, m.settingsCursor, m.settingsModelCursor, m.settingsEffortCursor, info))
	parts = append(parts, styles.Subtitle.Render(strings.Join(wrapWords(settingsNewRunCopy(provider), width), "\n")))
	return joinNonEmpty("\n", parts...)
}

func settingsProjectsFolderSummary(info settingsInfo) string {
	if info.ProjectsDir == "" {
		return "Not set"
	}
	return info.ProjectsDir
}

func settingsAIRunnerSummary(info settingsInfo) string {
	provider := info.preferredAgent()
	if provider == "" {
		provider = info.activeAgent()
	}
	if provider == "" {
		return "Not set"
	}
	parts := []string{settingsProviderName(provider)}
	model := info.providerModel(provider)
	if provider == "codex" && model == "" && info.providerModelMode(provider) != "default" {
		parts = append(parts, "Auto", "Auto by task")
	} else {
		parts = append(parts, airunner.ModelLabel(provider, model))
		if provider == "codex" {
			parts = append(parts, airunner.ReasoningEffortLabel(info.providerEffort(provider)))
		}
	}
	return strings.Join(parts, " · ")
}

func settingsColumnsView(width int, provider string, focusedColumn int, providerCursor int, modelCursor int, effortCursor int, info settingsInfo) string {
	providerWidth, modelWidth, effortWidth := settingsColumnWidths(width, provider)
	columns := []string{
		settingsChoiceColumn("Provider", settingsProviderOptions(info), providerCursor, focusedColumn == 0, providerWidth),
		settingsChoiceColumn("Model", settingsModelOptions(provider, info), modelCursor, focusedColumn == 1, modelWidth),
	}
	if provider == "codex" {
		effortOptions := airunner.ReasoningEffortOptions("codex")
		choices := make([]choiceOption, 0, len(effortOptions))
		for index, option := range effortOptions {
			label := option.Label
			if index == 0 && settingsModelOptionAt(provider, modelCursor).Auto {
				label = "Auto by task"
			}
			choices = append(choices, choiceOption{Label: label})
		}
		columns = append(columns, settingsChoiceColumn("Thinking effort", choices, effortCursor, focusedColumn == 2, effortWidth))
	}
	separator := styles.MutedText.Render("  ")
	view := lipgloss.JoinHorizontal(lipgloss.Top, columns[0], separator, columns[1])
	if len(columns) == 3 {
		view = lipgloss.JoinHorizontal(lipgloss.Top, view, separator, columns[2])
	}
	return view
}

func settingsColumnWidths(width int, provider string) (int, int, int) {
	providerWidth := min(16, max(12, width/5))
	if provider != "codex" {
		return providerWidth, min(34, max(20, width-providerWidth-2)), 0
	}
	effortWidth := min(22, max(17, width/4))
	modelWidth := min(34, max(20, width-providerWidth-effortWidth-4))
	return providerWidth, modelWidth, effortWidth
}

func settingsChoiceColumn(label string, options []choiceOption, cursor int, focused bool, width int) string {
	prefix := "  "
	if focused {
		prefix = "> "
	}
	lines := []string{styles.Section.Render(truncateMiddle(prefix+label, width))}
	for index, option := range options {
		value := truncateMiddle(option.Label, width)
		if index == cursor {
			lines = append(lines, styles.AccentText.Render(value))
		} else {
			lines = append(lines, styles.MutedText.Render(value))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func settingsFocusedChoiceDetail(width int, provider string, focusedColumn int, providerCursor int, modelCursor int, effortCursor int, info settingsInfo) string {
	switch focusedColumn {
	case 0:
		return settingsProviderDetailsView(width, providerCursor, info)
	case 2:
		return strings.TrimSpace(settingsEffortDetail(width, modelCursor, effortCursor, info))
	default:
		return settingsModelDetail(width, provider, modelCursor, info)
	}
}

func (m Model) cachedSettingsInfo() settingsInfo {
	if strings.TrimSpace(m.settings.ConfigPath) != "" {
		return m.settings
	}
	return readSettingsInfo()
}

func readSettingsInfo() settingsInfo {
	info := settingsInfo{
		ConfigPath:         settingsConfigPath(),
		EnvAgent:           strings.ToLower(strings.TrimSpace(os.Getenv("LINER_AGENT"))),
		Installed:          installedSettingsAgents(),
		ProviderModels:     map[string]string{},
		ProviderModelModes: map[string]string{},
		ProviderEfforts:    map[string]string{},
	}
	raw := readSettingsConfig()
	runnerConfig := airunner.ReadConfig()
	info.ConfigAgent = runnerConfig.Agent
	if runnerConfig.Runner != nil {
		info.ConfigAgent = runnerConfig.Runner.Agent
		info.ConfigExecutable = runnerConfig.Runner.Executable
		info.ConfigHome = runnerConfig.Runner.ConfigHome
	}
	for provider, preference := range runnerConfig.ProviderPreferences {
		info.ProviderModels[provider] = preference.Model
		info.ProviderModelModes[provider] = preference.ModelMode
		info.ProviderEfforts[provider] = preference.ReasoningEffort
	}
	if info.ConfigExecutable != "" && !containsAgentID(info.Installed, info.ConfigAgent) {
		info.Installed = append(info.Installed, info.ConfigAgent)
	}
	if projectsDir, ok := raw["projects_dir"].(string); ok {
		info.ProjectsDir = normalizeProjectDir(projectsDir)
	}
	if completed, ok := raw["onboarding_completed"].(bool); ok {
		info.OnboardingCompleted = completed
	}
	if prompted, ok := raw["jsSetupPrompted"].(bool); ok {
		info.JSSetupPrompted = prompted
	}
	if completed, ok := raw["jsSetupCompleted"].(bool); ok {
		info.JSSetupCompleted = completed
	}
	return info
}

func (info settingsInfo) providerModel(provider string) string {
	return strings.TrimSpace(info.ProviderModels[provider])
}

func (info settingsInfo) providerModelMode(provider string) string {
	return strings.TrimSpace(info.ProviderModelModes[provider])
}

func (info settingsInfo) providerEffort(provider string) string {
	return strings.TrimSpace(info.ProviderEfforts[provider])
}

func settingsConfigPath() string {
	return airunner.ConfigPath()
}

func readSettingsConfig() map[string]any {
	raw := map[string]any{}
	if data, err := os.ReadFile(settingsConfigPath()); err == nil {
		_ = yaml.Unmarshal(data, &raw)
	}
	return raw
}

func installedSettingsAgents() []string {
	out := []string{}
	for _, id := range settingsProviders {
		if settingsAgentInstalled(id) {
			out = append(out, id)
		}
	}
	return out
}

func settingsAgentInstalled(id string) bool {
	return settingsAgentExecutable(id) != ""
}

func settingsAgentExecutable(id string) string {
	return airunner.DetectExecutable(id)
}

func settingsAgentConfigHome(id string) string {
	return airunner.ResolveConfigHome(id, nil)
}

func (info settingsInfo) preferredAgent() string {
	if isSettingsProvider(info.EnvAgent) {
		return info.EnvAgent
	}
	if active := info.activeAgent(); isSettingsProvider(active) {
		return active
	}
	if isSettingsProvider(info.ConfigAgent) {
		return info.ConfigAgent
	}
	return "codex"
}

func (info settingsInfo) activeAgent() string {
	if info.EnvAgent != "" {
		if containsAgentID(info.Installed, info.EnvAgent) {
			return info.EnvAgent
		}
		return ""
	}
	if info.ConfigAgent != "" && containsAgentID(info.Installed, info.ConfigAgent) {
		return info.ConfigAgent
	}
	if len(info.Installed) == 1 {
		return info.Installed[0]
	}
	return ""
}

func (info settingsInfo) activeProviderLabel() string {
	active := info.activeAgent()
	if active == "" {
		if isSettingsProvider(info.EnvAgent) {
			return settingsProviderName(info.EnvAgent) + " unavailable"
		}
		if len(info.Installed) > 1 {
			return "not selected"
		}
		return "none"
	}
	return settingsProviderName(active)
}

func containsAgentID(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeSettingsSelection(provider string, model string, modelMode string, effort string) error {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	modelMode = strings.TrimSpace(modelMode)
	effort = strings.TrimSpace(effort)
	executable, configHome := effectiveSettingsRunner(provider, readSettingsInfo())
	return writeSettingsConfig(func(raw map[string]any) {
		raw["agent"] = provider
		runner, ok := raw["runner"].(map[string]any)
		if !ok {
			runner = map[string]any{}
		}
		runner["agent"] = provider
		runner["executable"] = executable
		runner["config_home"] = configHome
		raw["runner"] = runner
		if _, ok := raw["jsSetupPrompted"]; !ok {
			raw["jsSetupPrompted"] = false
		}

		preferences, ok := raw["provider_preferences"].(map[string]any)
		if !ok {
			preferences = map[string]any{}
		}
		entry, ok := preferences[provider].(map[string]any)
		if !ok {
			entry = map[string]any{}
		}
		if model == "" {
			delete(entry, "model")
		} else {
			entry["model"] = model
		}
		if provider == "codex" {
			if modelMode == "auto" || modelMode == "default" {
				entry["model_mode"] = modelMode
			} else {
				delete(entry, "model_mode")
			}
			if effort == "" {
				delete(entry, "reasoning_effort")
			} else {
				entry["reasoning_effort"] = effort
			}
		}
		if len(entry) == 0 {
			delete(preferences, provider)
		} else {
			preferences[provider] = entry
		}
		if len(preferences) == 0 {
			delete(raw, "provider_preferences")
		} else {
			raw["provider_preferences"] = preferences
		}
	})
}

func writeSettingsProjectsFolder(projectsDir string) error {
	projectsDir = normalizeProjectDir(projectsDir)
	return writeSettingsConfig(func(raw map[string]any) {
		raw["projects_dir"] = projectsDir
	})
}

func writeOnboardingConfig(projectsDir string, agent string, completed bool) error {
	projectsDir = normalizeProjectDir(projectsDir)
	executable, configHome := effectiveSettingsRunner(agent, readSettingsInfo())
	return writeSettingsConfig(func(raw map[string]any) {
		raw["projects_dir"] = projectsDir
		raw["agent"] = agent
		raw["runner"] = map[string]any{
			"agent":       agent,
			"executable":  executable,
			"config_home": configHome,
		}
		raw["onboarding_completed"] = completed
		if _, ok := raw["jsSetupPrompted"]; !ok {
			raw["jsSetupPrompted"] = false
		}
	})
}

func effectiveSettingsRunner(agent string, info settingsInfo) (string, string) {
	var saved *airunner.SavedProfile
	if info.ConfigAgent == agent && info.ConfigExecutable != "" && info.ConfigHome != "" {
		saved = &airunner.SavedProfile{Agent: agent, Executable: info.ConfigExecutable, ConfigHome: info.ConfigHome}
	}
	descriptor, ok := airunner.ResolveProvider(agent, saved)
	if !ok {
		return "", airunner.ResolveConfigHome(agent, saved)
	}
	return descriptor.Bin, descriptor.ConfigHome
}

func writeJSSetupConfig(prompted bool, completed bool) error {
	return writeSettingsConfig(func(raw map[string]any) {
		raw["jsSetupPrompted"] = prompted
		raw["jsSetupCompleted"] = completed
	})
}

func writeSettingsConfig(update func(map[string]any)) error {
	path := settingsConfigPath()
	raw := readSettingsConfig()
	update(raw)
	data, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	header := "# Liner user config - generated by the TUI.\n\n"
	return os.WriteFile(path, []byte(header+string(data)), 0o644)
}

func normalizeProjectDir(path string) string {
	path = expandUserPath(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func expandUserPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home := os.Getenv("HOME")
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		if home == "" {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func settingsProviderSelectorView(cursor int) string {
	return renderChoiceSelector(settingsProviderOptions(settingsInfo{}), cursor)
}

func settingsModelIndex(provider string, model string, modelMode string) int {
	model = strings.TrimSpace(model)
	modelMode = strings.TrimSpace(modelMode)
	options := airunner.ModelOptions(provider)
	if provider == "codex" && model == "" {
		for index, option := range options {
			if modelMode == "default" && !option.Auto && !option.Custom && option.Value == "" {
				return index
			}
			if modelMode != "default" && option.Auto {
				return index
			}
		}
	}
	for index, option := range options {
		if !option.Auto && !option.Custom && option.Value == model {
			return index
		}
	}
	if model != "" {
		for index, option := range options {
			if option.Custom {
				return index
			}
		}
	}
	return 0
}

func settingsRowCount(provider string) int {
	if provider == "codex" {
		return 3
	}
	return 2
}

func settingsEffortIndex(effort string) int {
	for index, option := range airunner.ReasoningEffortOptions("codex") {
		if option.Value == strings.TrimSpace(effort) {
			return index
		}
	}
	return 0
}

func settingsEffortDetail(width int, modelCursor int, effortCursor int, info settingsInfo) string {
	modelOptions := airunner.ModelOptions("codex")
	effortOptions := airunner.ReasoningEffortOptions("codex")
	if effortCursor < 0 || effortCursor >= len(effortOptions) {
		effortCursor = 0
	}
	if modelCursor < 0 || modelCursor >= len(modelOptions) {
		modelCursor = settingsModelIndex("codex", info.providerModel("codex"), info.providerModelMode("codex"))
	}
	selectedModel := modelOptions[modelCursor]
	selectedEffort := effortOptions[effortCursor]
	if selectedModel.Auto && selectedEffort.Value == "" {
		detail := "Auto uses Luna + High for Clarify Job, candidate research, and evaluation; Sol + Medium for framing, quality, synthesis, improvement, and assembly."
		return "\n" + styles.Subtitle.Width(width).Render(detail) + "\n"
	}
	detail := "Let the selected OpenAI model use its default reasoning effort."
	if selectedEffort.Value != "" {
		detail = "Use " + selectedEffort.Label + " OpenAI reasoning effort for new runs."
	}
	if selectedModel.Custom || (info.providerModel("codex") != "" && !airunner.IsCuratedModel("codex", info.providerModel("codex"))) {
		detail += " Compatibility is unverified for a custom model; OpenAI remains authoritative."
	} else if selectedModel.Value != "" {
		detail += " This effort is supported by the curated GPT-5.6 model."
	}
	return "\n" + styles.Subtitle.Width(width).Render(detail) + "\n"
}

func settingsNewRunCopy(provider string) string {
	if provider == "codex" {
		return "Model and Thinking effort changes apply to new runs. Resumed sessions keep the settings they started with."
	}
	return "Model changes apply to new runs. Resumed sessions keep the model they started with."
}

func settingsModelOptions(provider string, info settingsInfo) []choiceOption {
	model := info.providerModel(provider)
	options := airunner.ModelOptions(provider)
	out := make([]choiceOption, 0, len(options))
	for _, option := range options {
		label := option.Label
		if option.Custom && model != "" && !airunner.IsCuratedModel(provider, model) {
			label = "Custom: " + model
		}
		detail := "Use " + label + " for new " + settingsProviderName(provider) + " runs."
		if option.Auto {
			detail = "Let Liner choose the OpenAI model and Thinking effort for each new task: Luna + High for research-heavy work, Sol + Medium for higher-judgment work."
		} else if option.Value == "" && !option.Custom {
			detail = "Let the provider CLI choose its default model for new runs."
		} else if option.Custom {
			if model != "" && !airunner.IsCuratedModel(provider, model) {
				detail = "Use " + label + " for new " + settingsProviderName(provider) + " runs. Compatibility is unverified for a custom model; " + settingsProviderName(provider) + " remains authoritative."
			} else {
				detail = "Enter any model ID supported by the " + settingsProviderDescription(provider)
			}
		}
		out = append(out, choiceOption{Label: label, Detail: detail})
	}
	return out
}

func settingsModelOptionAt(provider string, cursor int) airunner.ModelOption {
	options := airunner.ModelOptions(provider)
	if cursor < 0 || cursor >= len(options) {
		return options[0]
	}
	return options[cursor]
}

func settingsModelDetail(width int, provider string, cursor int, info settingsInfo) string {
	return renderChoiceDetail(width, settingsModelOptions(provider, info), cursor)
}

func settingsProviderDetailsView(width int, cursor int, info settingsInfo) string {
	return renderChoiceDetail(width, settingsProviderOptions(info), cursor)
}

func settingsProviderOptions(info settingsInfo) []choiceOption {
	options := make([]choiceOption, 0, len(settingsProviders))
	for _, provider := range settingsProviders {
		options = append(options, choiceOption{
			Label:  settingsProviderName(provider),
			Detail: settingsProviderChoiceDetail(provider, info),
		})
	}
	return options
}

func settingsProviderDescription(provider string) string {
	switch provider {
	case "claude":
		return "Claude Code CLI."
	case "codex":
		return "OpenAI, using the Codex CLI."
	default:
		return settingsProviderName(provider) + " CLI."
	}
}

func settingsProviderChoiceDetail(provider string, info settingsInfo) string {
	if !containsAgentID(info.Installed, provider) {
		if info.EnvAgent == provider {
			binEnv := "LINER_CODEX_BIN"
			if provider == "claude" {
				binEnv = "LINER_CLAUDE_BIN"
			}
			return settingsProviderName(provider) + " is selected by LINER_AGENT, but its executable is unavailable. Set " + binEnv + " or update LINER_AGENT."
		}
		return settingsProviderInstallMessage(provider)
	}
	description := settingsProviderDescription(provider)
	executable, configHome := effectiveSettingsRunner(provider, info)
	if info.activeAgent() == provider {
		description += " Active runner."
	} else {
		description += " Press Enter to switch runners."
	}
	return description +
		" Executable: " + executable + ". Config home: " + configHome + "." +
		" Resolution: " + settingsRunnerResolutionSource(provider, info) + "." +
		" Readiness: preflight required before methodology."
}

func settingsRunnerResolutionSource(provider string, info settingsInfo) string {
	if info.EnvAgent == provider {
		return "environment override"
	}
	if provider == "claude" {
		if strings.TrimSpace(os.Getenv("LINER_CLAUDE_BIN")) != "" ||
			strings.TrimSpace(os.Getenv("LINER_CLAUDE_HOME")) != "" ||
			strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")) != "" {
			return "environment override"
		}
	} else if provider == "codex" {
		if strings.TrimSpace(os.Getenv("LINER_CODEX_BIN")) != "" ||
			strings.TrimSpace(os.Getenv("LINER_CODEX_HOME")) != "" ||
			strings.TrimSpace(os.Getenv("CODEX_HOME")) != "" {
			return "environment override"
		}
	}
	if info.ConfigAgent == provider && info.ConfigExecutable != "" && info.ConfigHome != "" {
		return "saved profile"
	}
	return "automatic detection"
}

func settingsProviderInstallMessage(provider string) string {
	switch provider {
	case "claude":
		return "Claude Code is not installed. Install the CLI version of Claude Code to use it here."
	case "codex":
		return "Codex CLI is not installed. Install Codex CLI to use it here."
	default:
		return settingsProviderName(provider) + " is not installed."
	}
}

func settingsProviderName(provider string) string {
	switch provider {
	case "claude":
		return "Claude"
	case "codex":
		return "OpenAI"
	default:
		return provider
	}
}

func settingsProviderAt(index int) string {
	if len(settingsProviders) == 0 {
		return ""
	}
	if index < 0 || index >= len(settingsProviders) {
		return settingsProviders[0]
	}
	return settingsProviders[index]
}

func settingsProviderIndex(provider string) int {
	for index, id := range settingsProviders {
		if id == provider {
			return index
		}
	}
	return 0
}

func isSettingsProvider(provider string) bool {
	for _, id := range settingsProviders {
		if id == provider {
			return true
		}
	}
	return false
}

func joinNonEmpty(sep string, values ...string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return strings.Join(out, sep)
}
