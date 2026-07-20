package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/agent"
	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

func TestRunnerPreferencesCleanRoomSmoke(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("LINER_RUNNER_PREFERENCES_SMOKE_ROOT"))
	if root == "" {
		t.Skip("set LINER_RUNNER_PREFERENCES_SMOKE_ROOT through npm run smoke:runner-preferences")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	projects := filepath.Join(root, "projects")
	codexHome := filepath.Join(root, "provider-homes", "codex")
	claudeHome := filepath.Join(root, "provider-homes", "claude")
	captures := filepath.Join(root, "captures")
	binDir := filepath.Join(root, "fake-provider-bin")
	for _, path := range []string{home, projects, codexHome, claudeHome, captures} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	executableSuffix := ""
	if runtime.GOOS == "windows" {
		executableSuffix = ".exe"
	}
	codexBin := filepath.Join(binDir, "codex"+executableSuffix)
	claudeBin := filepath.Join(binDir, "claude"+executableSuffix)
	for _, path := range []string{codexBin, claudeBin} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("native fake provider was not prepared: %s: %v", path, err)
		}
	}

	t.Setenv("HOME", home)
	t.Setenv("LINER_DIR", projects)
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CODEX_BIN", codexBin)
	t.Setenv("LINER_CODEX_HOME", codexHome)
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("LINER_CLAUDE_BIN", claudeBin)
	t.Setenv("LINER_CLAUDE_HOME", claudeHome)
	t.Setenv("CLAUDE_CONFIG_DIR", claudeHome)
	t.Setenv("LINER_SMOKE_ROOT", root)

	if got := core.DefaultBaseDir(); filepath.Clean(got) != filepath.Clean(projects) {
		t.Fatalf("isolated Project library = %q, want %q", got, projects)
	}
	if err := writeOnboardingConfig(projects, "codex", true); err != nil {
		t.Fatal(err)
	}

	m := Model{screen: screenProject, width: 118, settingsInput: newSettingsModelInput()}.startSettings()
	landing := m.viewSettings()
	for _, expected := range []string{"Projects folder", "AI runner"} {
		if !strings.Contains(landing, expected) {
			t.Fatalf("Settings landing missing %q", expected)
		}
	}
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	// OpenAI model: GPT-5.6 Sol; Thinking effort: Maximum.
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight})) // Model
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))  // OpenAI default
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))  // GPT-5.6 Sol
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight})) // Thinking effort
	for range 6 {
		m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	// Claude model: Opus, selected through the same Settings interaction.
	m = Model{screen: screenProject, width: 118, settingsInput: newSettingsModelInput()}.startSettings().openSettingsAIRunner()
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))  // Claude
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight})) // Model
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))  // Sonnet
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))  // Opus
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	// Return the active provider to OpenAI and enable Auto for fresh AI tasks.
	m = Model{screen: screenProject, width: 118, settingsInput: newSettingsModelInput()}.startSettings().openSettingsAIRunner()
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})) // OpenAI default
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})) // Auto
	m, _ = m.handleSettingsKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if m.err != "" {
		t.Fatalf("Settings smoke failed: %s", m.err)
	}

	restarted := Model{screen: screenProject, width: 118, settingsInput: newSettingsModelInput()}.startSettings().openSettingsAIRunner()
	if restarted.settings.preferredAgent() != "codex" || restarted.settings.providerModel("codex") != "" || restarted.settings.providerModelMode("codex") != "auto" || restarted.settings.providerEffort("codex") != "" {
		t.Fatalf("restarted OpenAI preferences = agent:%q model:%q mode:%q effort:%q", restarted.settings.preferredAgent(), restarted.settings.providerModel("codex"), restarted.settings.providerModelMode("codex"), restarted.settings.providerEffort("codex"))
	}
	openAIView := restarted.viewSettings()
	for _, expected := range []string{"OpenAI", "Auto", "Thinking effort", "Auto by task"} {
		if !strings.Contains(openAIView, expected) {
			t.Fatalf("restarted OpenAI Settings missing %q", expected)
		}
	}
	restarted = restarted.moveSettingsProvider(1)
	if restarted.settings.providerModel("claude") != "opus" {
		t.Fatalf("restarted Claude model = %q, want opus", restarted.settings.providerModel("claude"))
	}
	claudeView := restarted.viewSettings()
	if !strings.Contains(claudeView, "Opus") || strings.Contains(claudeView, "Thinking effort") {
		t.Fatalf("restarted Claude Settings did not preserve its independent model or hid OpenAI-only effort")
	}

	clarifyProject := createSmokeProject(t, projects, "openai-clarify")
	t.Setenv("LINER_AGENT", "codex")
	t.Setenv("LINER_SMOKE_CAPTURE", "codex-clarify")
	questions, err := core.GenerateClarifyingQuestions(filepath.Join(clarifyProject, "mixtape"), "Help an agent test isolated runner preferences.", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 {
		t.Fatalf("Clarify Job questions = %#v", questions)
	}
	assertSmokeCapture(t, filepath.Join(captures, "codex-clarify.txt"), []string{
		"CODEX_HOME=" + codexHome,
		"--model gpt-5.6-luna",
		`model_reasoning_effort="high"`,
	})

	runnerScript := strings.TrimSpace(os.Getenv("LINER_HEADLESS_RUNNER"))
	skillPath := strings.TrimSpace(os.Getenv("LINER_SKILL_PATH"))
	if runnerScript == "" || skillPath == "" {
		t.Fatal("smoke requires the built headless runner and tracked skill path")
	}

	openAIProject := createSmokeProject(t, projects, "openai-methodology")
	openAICorpus := filepath.Join(openAIProject, "mixtape")
	if err := os.MkdirAll(filepath.Join(openAICorpus, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(openAICorpus, "working", "01-jtbd-and-knowledge-map.md"), []byte("# Framing\n\nIsolated smoke framing.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_SMOKE_CAPTURE", "codex-methodology")
	runSmokeMethodologyPhase(t, runnerScript, skillPath, openAIProject, "candidates", "codex")
	assertSmokeCapture(t, filepath.Join(captures, "codex-methodology.txt"), []string{
		"CODEX_HOME=" + codexHome,
		"--model gpt-5.6-luna",
		`model_reasoning_effort="high"`,
	})

	openAIJudgmentProject := createSmokeProject(t, projects, "openai-judgment")
	t.Setenv("LINER_SMOKE_CAPTURE", "codex-judgment")
	runSmokeMethodologyPhase(t, runnerScript, skillPath, openAIJudgmentProject, "framing", "codex")
	assertSmokeCapture(t, filepath.Join(captures, "codex-judgment.txt"), []string{
		"CODEX_HOME=" + codexHome,
		"--model gpt-5.6-sol",
		`model_reasoning_effort="medium"`,
	})

	claudeProject := createSmokeProject(t, projects, "claude-methodology")
	t.Setenv("LINER_SMOKE_CAPTURE", "claude-methodology")
	runSmokeMethodologyPhase(t, runnerScript, skillPath, claudeProject, "framing", "claude")
	assertSmokeCapture(t, filepath.Join(captures, "claude-methodology.txt"), []string{
		"CLAUDE_CONFIG_DIR=" + claudeHome,
		"MAX_THINKING_TOKENS=0",
		"--model opus",
	})

	for _, path := range []string{
		settingsConfigPath(),
		filepath.Join(openAICorpus, "working", "02-candidate-longlist.md"),
		filepath.Join(openAICorpus, ".liner-runs", "candidates"),
		filepath.Join(openAIJudgmentProject, "mixtape", "working", "01-jtbd-and-knowledge-map.md"),
		filepath.Join(openAIJudgmentProject, "mixtape", ".liner-runs", "framing"),
		filepath.Join(claudeProject, "mixtape", "working", "01-jtbd-and-knowledge-map.md"),
		filepath.Join(claudeProject, "mixtape", ".liner-runs", "framing"),
		filepath.Join(clarifyProject, "mixtape", ".liner-runs", "jtbd-clarify"),
	} {
		absolute, absErr := filepath.Abs(path)
		if absErr != nil || !realPathWithinRoot(root, absolute) {
			t.Fatalf("writable smoke path escaped isolated root: %q", path)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("expected isolated smoke artifact %q: %v", path, statErr)
		}
	}
}

func createSmokeProject(t *testing.T, projects string, name string) string {
	t.Helper()
	root := filepath.Join(projects, name)
	corpus := filepath.Join(root, "mixtape")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "liner.yaml"), []byte(fmt.Sprintf("version: 1\nid: smoke-%s\n", name)), 0o644); err != nil {
		t.Fatal(err)
	}
	tapeBody := fmt.Sprintf("title: %s\ndescription: Offline isolated runner smoke.\nversion: 1\ncurator: Liner test\nmode: methodology\njtbd: Help an agent prove isolated runner preferences.\ntags: [smoke]\nsources: []\n", name)
	if err := os.WriteFile(filepath.Join(corpus, "tape.yaml"), []byte(tapeBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runSmokeMethodologyPhase(t *testing.T, runnerScript string, skillPath string, project string, phase string, provider string) {
	t.Helper()
	run, err := (agent.Runner{ScriptPath: runnerScript}).Start(context.Background(), agent.RunArgs{
		Project: project, PhaseID: phase, Agent: provider, SkillPath: skillPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	for range run.Events {
	}
	if err := <-run.Done; err != nil {
		t.Fatalf("%s methodology smoke: %v", provider, err)
	}
}

func assertSmokeCapture(t *testing.T, path string, expected []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, value := range expected {
		if !strings.Contains(text, value) {
			t.Fatalf("capture %s missing %q:\n%s", path, value, text)
		}
	}
}

func realPathWithinRoot(root string, path string) bool {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalPath)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func runSmokeProviderHelper() int {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
	if name != "codex" && name != "claude" {
		fmt.Fprintln(os.Stderr, "unknown smoke provider helper:", name)
		return 2
	}
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "--version" || args[0] == "--help") {
		if name == "codex" {
			fmt.Println("codex-cli 99.0 Usage: codex exec")
		} else {
			fmt.Println("claude-code 99.0 Usage: claude -p --print")
		}
		return 0
	}
	if len(args) >= 2 && ((name == "codex" && args[0] == "login" && args[1] == "status") || (name == "claude" && args[0] == "auth" && args[1] == "status")) {
		fmt.Println("Authenticated")
		return 0
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	captureName := os.Getenv("LINER_SMOKE_CAPTURE")
	capture := filepath.Join(os.Getenv("LINER_SMOKE_ROOT"), "captures", captureName+".txt")
	cwd, _ := os.Getwd()
	lines := []string{"PWD=" + cwd, "HOME=" + os.Getenv("HOME")}
	if name == "codex" {
		lines = append(lines, "CODEX_HOME="+os.Getenv("CODEX_HOME"))
	} else {
		lines = append(lines, "CLAUDE_CONFIG_DIR="+os.Getenv("CLAUDE_CONFIG_DIR"), "MAX_THINKING_TOKENS="+os.Getenv("MAX_THINKING_TOKENS"))
	}
	lines = append(lines, "LINER_DIR="+os.Getenv("LINER_DIR"), "ARGS="+strings.Join(args, " "))
	if err := os.WriteFile(capture, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if name == "codex" && captureName == "codex-methodology" {
		if err := os.MkdirAll(filepath.Join(cwd, "working"), 0o755); err != nil {
			return 1
		}
		if err := os.WriteFile(filepath.Join(cwd, "working", "02-candidate-longlist.md"), []byte("# Candidates\n\n- https://example.com/isolated-openai\n"), 0o644); err != nil {
			return 1
		}
		fmt.Println(`{"type":"thread.started","thread_id":"smoke-methodology"}`)
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"Candidate artifact written."}}`)
		return 0
	}
	if name == "codex" && captureName == "codex-judgment" {
		if err := os.MkdirAll(filepath.Join(cwd, "working"), 0o755); err != nil {
			return 1
		}
		if err := os.WriteFile(filepath.Join(cwd, "working", "01-jtbd-and-knowledge-map.md"), []byte("# Framing\n\nOffline OpenAI framing artifact.\n"), 0o644); err != nil {
			return 1
		}
		fmt.Println(`{"type":"thread.started","thread_id":"smoke-judgment"}`)
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"Framing artifact written."}}`)
		return 0
	}
	if name == "codex" {
		fmt.Println(`{"type":"thread.started","thread_id":"smoke-clarify"}`)
		fmt.Println(`{"type":"item.completed","item":{"type":"agent_message","text":"[\"Which isolated output matters?\",\"What should stay outside production?\"]"}}`)
		return 0
	}
	if err := os.MkdirAll(filepath.Join(cwd, "working"), 0o755); err != nil {
		return 1
	}
	if err := os.WriteFile(filepath.Join(cwd, "working", "01-jtbd-and-knowledge-map.md"), []byte("# Framing\n\nOffline Claude framing artifact.\n"), 0o644); err != nil {
		return 1
	}
	fmt.Println(`{"type":"assistant","message":{"content":[{"type":"text","text":"Framing artifact written."}]}}`)
	return 0
}

func isSmokeProviderHelper() bool {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
	return name == "codex" || name == "claude"
}
