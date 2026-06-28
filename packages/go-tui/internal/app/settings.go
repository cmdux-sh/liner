package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"gopkg.in/yaml.v3"
)

type settingsInfo struct {
	ConfigPath          string
	ConfigAgent         string
	EnvAgent            string
	ProjectsDir         string
	OnboardingCompleted bool
	JSSetupPrompted     bool
	JSSetupCompleted    bool
	Installed           []string
}

var settingsProviders = []string{"codex", "claude"}

func (m Model) startSettings() Model {
	if m.screen != screenSettings {
		m.previous = m.screen
	}
	m.settings = readSettingsInfo()
	m.settingsCursor = settingsProviderIndex(m.settings.preferredAgent())
	m.screen = screenSettings
	m.err = ""
	return m
}

func (m Model) handleSettingsKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h":
		return m.showCommandHome(), nil
	case "s":
		return m.startOnboarding(), nil
	case "shift+tab":
		return m.moveSettingsProvider(-1), nil
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		return m.closeSettings(), nil
	case tea.KeyLeft, tea.KeyUp:
		return m.moveSettingsProvider(-1), nil
	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		return m.moveSettingsProvider(1), nil
	case tea.KeyEnter:
		return m.saveSettingsAgentPreference(), nil
	}
	switch keyMsg.String() {
	case "left", "up", "←", "↑":
		m.moveSettingsProvider(-1)
	case "right", "down", "tab", "→", "↓":
		m.moveSettingsProvider(1)
	}
	return m, nil
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
	m.err = ""
	return m
}

func (m Model) saveSettingsAgentPreference() Model {
	info := readSettingsInfo()
	selected := settingsProviderAt(m.settingsCursor)
	if !containsAgentID(info.Installed, selected) {
		m.err = settingsProviderInstallMessage(selected)
		return m
	}
	if err := writeSettingsAgentPreference(selected); err != nil {
		m.err = "Could not save AI runner: " + err.Error()
		return m
	}
	m.settings = readSettingsInfo()
	m.settingsCursor = settingsProviderIndex(selected)
	m.note = "Saved " + settingsProviderName(selected) + " as the AI runner."
	return m
}

func (m Model) viewSettings() string {
	width := styles.ClampWidth(m.width - 4)
	info := m.cachedSettingsInfo()
	description := strings.Join(wrapWords("Choose the AI runner Liner uses to research sources and create project files.", width), "\n")
	return joinNonEmpty("\n",
		styles.Title.Render("Settings"),
		styles.PrimaryText.Render(description),
		"",
		settingsProviderSelectorView(m.settingsCursor),
		settingsProviderDetailsView(width, m.settingsCursor, info),
	)
}

func (m Model) cachedSettingsInfo() settingsInfo {
	if strings.TrimSpace(m.settings.ConfigPath) != "" {
		return m.settings
	}
	return readSettingsInfo()
}

func readSettingsInfo() settingsInfo {
	info := settingsInfo{
		ConfigPath: settingsConfigPath(),
		EnvAgent:   strings.ToLower(strings.TrimSpace(os.Getenv("LINER_AGENT"))),
		Installed:  installedSettingsAgents(),
	}
	raw := readSettingsConfig()
	if agent, ok := raw["agent"].(string); ok && (agent == "claude" || agent == "codex") {
		info.ConfigAgent = agent
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

func settingsConfigPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".liner", "config.yaml")
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
	switch id {
	case "claude":
		if strings.TrimSpace(os.Getenv("LINER_CLAUDE_BIN")) != "" {
			return true
		}
		_, err := exec.LookPath("claude")
		return err == nil
	case "codex":
		if strings.TrimSpace(os.Getenv("LINER_CODEX_BIN")) != "" {
			return true
		}
		_, err := exec.LookPath("codex")
		return err == nil
	default:
		return false
	}
}

func (info settingsInfo) preferredAgent() string {
	if isSettingsProvider(info.ConfigAgent) {
		return info.ConfigAgent
	}
	if active := info.activeAgent(); isSettingsProvider(active) {
		return active
	}
	return "codex"
}

func (info settingsInfo) activeAgent() string {
	if info.EnvAgent != "" && containsAgentID(info.Installed, info.EnvAgent) {
		return info.EnvAgent
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

func writeSettingsAgentPreference(agent string) error {
	return writeSettingsConfig(func(raw map[string]any) {
		raw["agent"] = agent
		if _, ok := raw["jsSetupPrompted"]; !ok {
			raw["jsSetupPrompted"] = false
		}
	})
}

func writeOnboardingConfig(projectsDir string, agent string, completed bool) error {
	projectsDir = normalizeProjectDir(projectsDir)
	return writeSettingsConfig(func(raw map[string]any) {
		raw["projects_dir"] = projectsDir
		raw["agent"] = agent
		raw["onboarding_completed"] = completed
		if _, ok := raw["jsSetupPrompted"]; !ok {
			raw["jsSetupPrompted"] = false
		}
	})
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
		return "Codex CLI."
	default:
		return settingsProviderName(provider) + " CLI."
	}
}

func settingsProviderChoiceDetail(provider string, info settingsInfo) string {
	if !containsAgentID(info.Installed, provider) {
		return settingsProviderInstallMessage(provider)
	}
	description := settingsProviderDescription(provider)
	if info.activeAgent() == provider {
		return description + " Active runner."
	}
	return description + " Press Enter to switch runners."
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
		return "Codex"
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
