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
	if got := config.ProviderPreferences["claude"].Model; got != "opus" {
		t.Fatalf("Claude model = %q, want opus", got)
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

func TestAutoModelPolicyUsesApprovedOpenAITiersAndHonorsManualPreferences(t *testing.T) {
	auto := Descriptor{ID: "codex", ModelMode: "auto"}
	for _, test := range []struct {
		task   string
		model  string
		effort string
	}{
		{task: "jtbd-clarify", model: "gpt-5.6-luna", effort: "high"},
		{task: "candidates", model: "gpt-5.6-luna", effort: "high"},
		{task: "evaluation", model: "gpt-5.6-luna", effort: "high"},
		{task: "framing", model: "gpt-5.6-sol", effort: "medium"},
		{task: "quality", model: "gpt-5.6-sol", effort: "medium"},
		{task: "synthesis", model: "gpt-5.6-sol", effort: "medium"},
		{task: "improvement", model: "gpt-5.6-sol", effort: "medium"},
		{task: "assembly", model: "gpt-5.6-sol", effort: "medium"},
	} {
		got := ApplyAutoModelPolicy(auto, test.task)
		if got.Model != test.model || got.ReasoningEffort != test.effort {
			t.Fatalf("Auto %s = model:%q effort:%q, want model:%q effort:%q", test.task, got.Model, got.ReasoningEffort, test.model, test.effort)
		}
	}

	manual := ApplyAutoModelPolicy(Descriptor{ID: "codex", Model: "gpt-5.6-terra", ReasoningEffort: "xhigh"}, "candidates")
	if manual.Model != "gpt-5.6-terra" || manual.ReasoningEffort != "xhigh" {
		t.Fatalf("manual preference changed by Auto: %#v", manual)
	}
	providerDefault := ApplyAutoModelPolicy(Descriptor{ID: "codex", ModelMode: "default"}, "candidates")
	if providerDefault.Model != "" || providerDefault.ReasoningEffort != "" {
		t.Fatalf("provider default changed by Auto: %#v", providerDefault)
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
