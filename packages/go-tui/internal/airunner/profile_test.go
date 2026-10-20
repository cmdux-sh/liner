package airunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSavedProfileWinsPATHAndExplicitOverridesWinSavedProfile(t *testing.T) {
	pathDir := t.TempDir()
	pathExecutable := filepath.Join(pathDir, "codex")
	if err := os.WriteFile(pathExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	saved := &SavedProfile{Agent: "codex", Executable: "/saved/bin/codex", ConfigHome: "/saved/codex-home"}
	t.Setenv("PATH", pathDir)
	t.Setenv("LINER_CODEX_BIN", "")
	t.Setenv("LINER_CODEX_HOME", "")
	t.Setenv("CODEX_HOME", "")

	descriptor, ok := ResolveProvider("codex", saved)
	if !ok {
		t.Fatal("expected saved Codex profile to resolve")
	}
	if descriptor.Bin != saved.Executable || descriptor.ConfigHome != saved.ConfigHome {
		t.Fatalf("saved descriptor = %#v, want executable %q and home %q", descriptor, saved.Executable, saved.ConfigHome)
	}

	t.Setenv("LINER_CODEX_BIN", "/override/bin/codex")
	t.Setenv("CODEX_HOME", "/native/codex-home")
	descriptor, ok = ResolveProvider("codex", saved)
	if !ok {
		t.Fatal("expected overridden Codex profile to resolve")
	}
	if descriptor.Bin != "/override/bin/codex" || descriptor.ConfigHome != "/native/codex-home" {
		t.Fatalf("overridden descriptor = %#v", descriptor)
	}

	t.Setenv("LINER_CODEX_HOME", "/liner/codex-home")
	descriptor, _ = ResolveProvider("codex", saved)
	if descriptor.ConfigHome != "/liner/codex-home" {
		t.Fatalf("LINER_CODEX_HOME did not win: %#v", descriptor)
	}
}

func TestEnvironmentReplacesNativeConfigHomeWithoutCredentials(t *testing.T) {
	descriptor := Descriptor{ID: "claude", ConfigHome: "/saved/claude-home"}
	environment := Environment([]string{"PATH=/bin", "CLAUDE_CONFIG_DIR=/wrong", "SECRET=value"}, descriptor)
	want := map[string]bool{
		"PATH=/bin":                            true,
		"CLAUDE_CONFIG_DIR=/saved/claude-home": true,
		"SECRET=value":                         true,
	}
	if len(environment) != len(want) {
		t.Fatalf("environment = %#v", environment)
	}
	for _, entry := range environment {
		if !want[entry] {
			t.Fatalf("unexpected environment entry %q in %#v", entry, environment)
		}
	}
}

func TestResolveRejectsUnknownExplicitProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "mystery")

	_, err := Resolve()
	if err == nil || !strings.Contains(err.Error(), "LINER_AGENT=mystery") {
		t.Fatalf("Resolve error = %v, want explicit invalid override", err)
	}
}

func TestReadConfigKeepsRunnerWhenOneProviderModelIsMalformed(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent: codex\nrunner:\n  agent: codex\n  executable: /saved/codex\n  config_home: /saved/home\nprovider_preferences:\n  codex: malformed\n  claude:\n    model: ' opus '\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	config := ReadConfig()
	if config.Runner == nil || config.Runner.Executable != "/saved/codex" {
		t.Fatalf("runner invalidated by provider preference: %#v", config)
	}
	if got := config.ProviderPreferences["claude"].Model; got != "claude-opus-5-5" {
		t.Fatalf("a saved Opus alias should move to Opus 5.5, got %q", got)
	}
	if got := config.ProviderPreferences["codex"].Model; got != "" {
		t.Fatalf("malformed Codex model = %q, want default", got)
	}
}

func TestReadConfigDropsOnlyMalformedReasoningEffort(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent: codex\nrunner:\n  agent: codex\n  executable: /saved/codex\n  config_home: /saved/home\nprovider_preferences:\n  codex:\n    model: gpt-5.6-sol\n    reasoning_effort: turbo\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	config := ReadConfig()
	preference := config.ProviderPreferences["codex"]
	if preference.Model != "gpt-5.6-sol" || preference.ReasoningEffort != "" {
		t.Fatalf("preference = %#v, want model with default effort", preference)
	}
}

func TestResolveAppliesActiveProviderModel(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent: codex\nrunner:\n  agent: codex\n  executable: /saved/codex\n  config_home: /saved/home\nprovider_preferences:\n  codex:\n    model: gpt-5.6-terra\n    reasoning_effort: max\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CODEX_BIN", "")

	descriptor, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Model != "gpt-5.6-terra" {
		t.Fatalf("resolved model = %q", descriptor.Model)
	}
	if descriptor.ReasoningEffort != "max" {
		t.Fatalf("resolved effort = %q", descriptor.ReasoningEffort)
	}
}

func TestDefaultsRunSol61AndSonnet55AtMediumAndHonorManualPreferences(t *testing.T) {
	auto := ApplyDefaults(Descriptor{ID: "codex", ModelMode: "auto"})
	if auto.Model != "gpt-6.1-sol" || auto.ReasoningEffort != "medium" || !auto.ModelIsAuto || !auto.EffortIsAuto {
		t.Fatalf("OpenAI Auto = %#v, want Sol 6.1 at Medium marked Auto", auto)
	}
	unset := ApplyDefaults(Descriptor{ID: "codex"})
	if unset.Model != "gpt-6.1-sol" || unset.ReasoningEffort != "medium" {
		t.Fatalf("OpenAI with no preference = %#v, want Sol 6.1 at Medium", unset)
	}
	autoHigh := ApplyDefaults(Descriptor{ID: "codex", ModelMode: "auto", ReasoningEffort: "high"})
	if autoHigh.Model != "gpt-6.1-sol" || autoHigh.ReasoningEffort != "high" || autoHigh.EffortIsAuto {
		t.Fatalf("an effort chosen in Settings should survive Auto: %#v", autoHigh)
	}
	manual := ApplyDefaults(Descriptor{ID: "codex", Model: "gpt-5.6-terra", ReasoningEffort: "xhigh"})
	if manual.Model != "gpt-5.6-terra" || manual.ReasoningEffort != "xhigh" || manual.ModelIsAuto {
		t.Fatalf("manual preference changed by Auto: %#v", manual)
	}
	providerDefault := ApplyDefaults(Descriptor{ID: "codex", ModelMode: "default"})
	if providerDefault.Model != "" || providerDefault.ReasoningEffort != "" {
		t.Fatalf("provider default changed by Auto: %#v", providerDefault)
	}

	claude := ApplyDefaults(Descriptor{ID: "claude"})
	if claude.Model != "claude-sonnet-5-5" || claude.ReasoningEffort != "medium" || !claude.ModelIsAuto || !claude.EffortIsAuto {
		t.Fatalf("Claude with no preference = %#v, want Sonnet 5.5 at Medium marked as defaults", claude)
	}
	opus := ApplyDefaults(Descriptor{ID: "claude", Model: "claude-opus-5-5", ReasoningEffort: "high"})
	if opus.Model != "claude-opus-5-5" || opus.ReasoningEffort != "high" || opus.ModelIsAuto || opus.EffortIsAuto {
		t.Fatalf("Claude choices from Settings should stay as chosen: %#v", opus)
	}
}

func TestCatalogOffersOnlyTheNewModels(t *testing.T) {
	labels := func(provider string) []string {
		out := []string{}
		for _, option := range ModelOptions(provider) {
			out = append(out, option.Label)
		}
		return out
	}
	if got := strings.Join(labels("codex"), ", "); got != "Auto, OpenAI default, Sol 6.1, Custom model ID" {
		t.Fatalf("OpenAI models = %s", got)
	}
	if got := strings.Join(labels("claude"), ", "); got != "Sonnet 5.5, Opus 5.5, Custom model ID" {
		t.Fatalf("Claude models = %s", got)
	}
	if got := ModelLabel("codex", "gpt-5.6-sol"); got != "Custom: gpt-5.6-sol" {
		t.Fatalf("a saved GPT-5.6 model should keep working as a custom ID, got %q", got)
	}
	if got := ModelLabel("claude", ""); got != "Sonnet 5.5" {
		t.Fatalf("an unset Claude model should read as Sonnet 5.5, got %q", got)
	}
	if got := ReasoningEffortLabel("claude", ""); got != "Medium" {
		t.Fatalf("an unset Claude effort should read as Medium, got %q", got)
	}
	if ValidReasoningEffort("claude", "none") || !ValidReasoningEffort("claude", "max") {
		t.Fatal("Claude efforts should match Claude Code's --effort levels")
	}
}

func TestReadConfigKeepsClaudeEffortAndMigratesTheSonnetAlias(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent: claude\nprovider_preferences:\n  claude:\n    model: sonnet\n    reasoning_effort: high\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	preference := ReadConfig().ProviderPreferences["claude"]
	if preference.Model != "claude-sonnet-5-5" || preference.ReasoningEffort != "high" {
		t.Fatalf("Claude preference = %#v, want Sonnet 5.5 at High", preference)
	}
}

func TestReadConfigPreservesOpenAIModelModeWithoutAConcreteModel(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent: codex\nprovider_preferences:\n  codex:\n    model_mode: auto\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	preference := ReadConfig().ProviderPreferences["codex"]
	if preference.Model != "" || preference.ModelMode != "auto" {
		t.Fatalf("Auto preference = %#v", preference)
	}
}
