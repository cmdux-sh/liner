package airunner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type SavedProfile struct {
	Agent      string `yaml:"agent"`
	Executable string `yaml:"executable"`
	ConfigHome string `yaml:"config_home"`
}

type Config struct {
	Agent               string
	Runner              *SavedProfile
	ProviderPreferences map[string]ProviderPreference
}

type ProviderPreference struct {
	Model           string
	ModelMode       string
	ReasoningEffort string
}

type Descriptor struct {
	ID              string
	Name            string
	Bin             string
	ConfigHome      string
	Model           string
	ModelMode       string
	ReasoningEffort string
	ModelIsAuto     bool
	EffortIsAuto    bool
}

func ConfigPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".liner", "config.yaml")
}

func ReadConfig() Config {
	var raw struct {
		Agent               string         `yaml:"agent"`
		Runner              *SavedProfile  `yaml:"runner"`
		ProviderPreferences map[string]any `yaml:"provider_preferences"`
	}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return Config{}
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}
	}
	config := Config{Agent: normalizeAgent(raw.Agent), Runner: raw.Runner}
	if config.Runner == nil || normalizeAgent(config.Runner.Agent) == "" ||
		strings.TrimSpace(config.Runner.Executable) == "" || strings.TrimSpace(config.Runner.ConfigHome) == "" {
		config.Runner = nil
	} else {
		config.Runner.Agent = normalizeAgent(config.Runner.Agent)
		config.Runner.Executable = strings.TrimSpace(config.Runner.Executable)
		config.Runner.ConfigHome = strings.TrimSpace(config.Runner.ConfigHome)
	}
	for _, id := range []string{"claude", "codex"} {
		entry, ok := raw.ProviderPreferences[id].(map[string]any)
		if !ok {
			continue
		}
		model, ok := entry["model"].(string)
		preference := ProviderPreference{}
		if ok && strings.TrimSpace(model) != "" {
			preference.Model = strings.TrimSpace(model)
		}
		if id == "codex" {
			if mode, ok := entry["model_mode"].(string); ok && validModelMode(mode) {
				preference.ModelMode = strings.TrimSpace(mode)
			}
			if effort, ok := entry["reasoning_effort"].(string); ok && validReasoningEffort(effort) {
				preference.ReasoningEffort = strings.TrimSpace(effort)
			}
		}
		if preference.Model == "" && preference.ModelMode == "" && preference.ReasoningEffort == "" {
			continue
		}
		if config.ProviderPreferences == nil {
			config.ProviderPreferences = map[string]ProviderPreference{}
		}
		config.ProviderPreferences[id] = preference
	}
	return config
}

func Resolve() (Descriptor, error) {
	config := ReadConfig()
	if rawPreferred := strings.TrimSpace(os.Getenv("LINER_AGENT")); rawPreferred != "" {
		preferred := normalizeAgent(rawPreferred)
		if preferred == "" {
			return Descriptor{}, errors.New("LINER_AGENT=" + rawPreferred + " is not a supported provider")
		}
		if descriptor, ok := ResolveProvider(preferred, config.Runner); ok {
			return config.applyPreferences(descriptor), nil
		}
		return Descriptor{}, errors.New(unavailableMessage(preferred))
	}
	if config.Runner != nil {
		if descriptor, ok := ResolveProvider(config.Runner.Agent, config.Runner); ok {
			return config.applyPreferences(descriptor), nil
		}
	}
	if config.Agent != "" {
		if descriptor, ok := ResolveProvider(config.Agent, nil); ok {
			return config.applyPreferences(descriptor), nil
		}
	}
	for _, id := range []string{"claude", "codex"} {
		if descriptor, ok := ResolveProvider(id, nil); ok {
			return config.applyPreferences(descriptor), nil
		}
	}
	return Descriptor{}, errors.New("no Claude Code or OpenAI runner found on PATH or in Settings")
}

func (config Config) applyPreferences(descriptor Descriptor) Descriptor {
	preference := config.ProviderPreferences[descriptor.ID]
	descriptor.Model = preference.Model
	descriptor.ModelMode = preference.ModelMode
	if descriptor.ID == "codex" {
		descriptor.ReasoningEffort = preference.ReasoningEffort
	}
	return descriptor
}

func validModelMode(value string) bool {
	switch strings.TrimSpace(value) {
	case "auto", "default":
		return true
	default:
		return false
	}
}

func validReasoningEffort(value string) bool {
	switch strings.TrimSpace(value) {
	case "none", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func ResolveProvider(id string, saved *SavedProfile) (Descriptor, bool) {
	id = normalizeAgent(id)
	provider, ok := providerFor(id)
	if !ok {
		return Descriptor{}, false
	}
	executable := strings.TrimSpace(os.Getenv(provider.binEnv))
	if executable == "" && saved != nil && normalizeAgent(saved.Agent) == id {
		executable = strings.TrimSpace(saved.Executable)
	}
	if executable == "" {
		executable = DetectExecutable(id)
	}
	if executable == "" {
		return Descriptor{}, false
	}
	return Descriptor{ID: id, Name: providerName(id), Bin: executable, ConfigHome: ResolveConfigHome(id, saved)}, true
}

func DetectExecutable(id string) string {
	provider, ok := providerFor(id)
	if !ok {
		return ""
	}
	if configured := strings.TrimSpace(os.Getenv(provider.binEnv)); configured != "" {
		return configured
	}
	path, err := exec.LookPath(provider.command)
	if err != nil {
		return ""
	}
	if absolute, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absolute)
	}
	return path
}

func ResolveConfigHome(id string, saved *SavedProfile) string {
	provider, ok := providerFor(id)
	if !ok {
		return ""
	}
	if configured := strings.TrimSpace(os.Getenv(provider.homeEnv)); configured != "" {
		return cleanPath(configured)
	}
	if configured := strings.TrimSpace(os.Getenv(provider.nativeHomeEnv)); configured != "" {
		return cleanPath(configured)
	}
	if saved != nil && normalizeAgent(saved.Agent) == normalizeAgent(id) && strings.TrimSpace(saved.ConfigHome) != "" {
		return cleanPath(saved.ConfigHome)
	}
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, provider.defaultDir)
}

func Environment(base []string, descriptor Descriptor) []string {
	provider, ok := providerFor(descriptor.ID)
	if !ok || strings.TrimSpace(descriptor.ConfigHome) == "" {
		return append([]string(nil), base...)
	}
	return setEnvironment(base, provider.nativeHomeEnv, descriptor.ConfigHome)
}

type provider struct {
	command       string
	name          string
	binEnv        string
	homeEnv       string
	nativeHomeEnv string
	defaultDir    string
}

func providerFor(id string) (provider, bool) {
	switch normalizeAgent(id) {
	case "claude":
		return provider{command: "claude", name: "Claude", binEnv: "LINER_CLAUDE_BIN", homeEnv: "LINER_CLAUDE_HOME", nativeHomeEnv: "CLAUDE_CONFIG_DIR", defaultDir: ".claude"}, true
	case "codex":
		return provider{command: "codex", name: "OpenAI", binEnv: "LINER_CODEX_BIN", homeEnv: "LINER_CODEX_HOME", nativeHomeEnv: "CODEX_HOME", defaultDir: ".codex"}, true
	default:
		return provider{}, false
	}
}

func providerName(id string) string {
	provider, _ := providerFor(id)
	return provider.name
}

func normalizeAgent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "claude" || value == "codex" {
		return value
	}
	return ""
}

func unavailableMessage(id string) string {
	if id == "claude" {
		return "claude is selected by LINER_AGENT, but its executable is unavailable; set LINER_CLAUDE_BIN or update LINER_AGENT"
	}
	return "codex is selected by LINER_AGENT, but its executable is unavailable; set LINER_CODEX_BIN or update LINER_AGENT"
}

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home := os.Getenv("HOME")
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		if path == "~" {
			path = home
		} else if home != "" {
			path = filepath.Join(home, path[2:])
		}
	}
	if absolute, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(path)
}

func setEnvironment(base []string, key string, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(base)+1)
	for _, entry := range base {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return append(out, prefix+value)
}
