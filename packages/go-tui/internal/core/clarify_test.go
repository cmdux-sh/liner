package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseClarifyingQuestionsFromJSONArray(t *testing.T) {
	body := `["Which platform matters most?", "What should the mixtape help produce?"]`
	got := ParseClarifyingQuestions(body)
	want := []string{"Which platform matters most?", "What should the mixtape help produce?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("questions = %#v, want %#v", got, want)
	}
}

func TestParseClarifyingQuestionsFromFencedObjects(t *testing.T) {
	body := "```json\n[{\"question\":\"Which audience should the research assume?\"}]\n```"
	got := ParseClarifyingQuestions(body)
	want := []string{"Which audience should the research assume?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("questions = %#v, want %#v", got, want)
	}
}

func TestParseClarifyingQuestionsRejectsProse(t *testing.T) {
	if got := ParseClarifyingQuestions("Here are some questions."); len(got) != 0 {
		t.Fatalf("questions = %#v, want none", got)
	}
}

func TestBuildClarifyPromptUsesCapabilityGoal(t *testing.T) {
	prompt := buildClarifyPrompt("Help my AI agent translate moodboards into UI direction.")
	for _, expected := range []string{
		"capability goal",
		"future AI agent",
		"source categories",
		"Do not ask the user",
		"runtime inputs and output contract",
		"generic checklist",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(prompt, "just-typed JTBD") {
		t.Fatalf("prompt should not use old JTBD-first language:\n%s", prompt)
	}
}

func TestGenerateClarifyingQuestionsRetriesAgentFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "codex")
	counter := filepath.Join(dir, "attempts")
	body := `#!/bin/sh
count=0
if [ -f "$LINER_RETRY_COUNT" ]; then
  count=$(cat "$LINER_RETRY_COUNT")
fi
count=$((count + 1))
printf "%s" "$count" > "$LINER_RETRY_COUNT"
if [ "$count" -eq 1 ]; then
  echo "temporary clarify failure" >&2
  exit 42
fi
printf '{"type":"item.completed","item":{"type":"agent_message","text":"[\"What should the future agent produce?\"]"}}\n'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINER_AGENT", "codex")
	t.Setenv("LINER_CODEX_BIN", script)
	t.Setenv("LINER_RETRY_COUNT", counter)
	t.Setenv("HOME", dir)

	got, err := GenerateClarifyingQuestions(dir, "Help my AI agent read references.", 5_000_000_000)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"What should the future agent produce?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("questions = %#v, want %#v", got, want)
	}
	attempts, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(attempts)) != "2" {
		t.Fatalf("expected one retry, counter = %q", attempts)
	}
}

func TestGenerateClarifyingQuestionsUsesSavedCodexProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	configHome := filepath.Join(home, "saved-codex-home")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(home, "saved-codex")
	body := `#!/bin/sh
if [ "$CODEX_HOME" != "$EXPECTED_CONFIG_HOME" ]; then
  echo "wrong CODEX_HOME: $CODEX_HOME" >&2
  exit 41
fi
if [ "$PWD" != "$EXPECTED_CWD_REAL" ]; then
  echo "wrong cwd: $PWD" >&2
  exit 42
fi
if [ "$1" != "exec" ] || [ "$2" != "--cd" ] || [ "$3" != "$EXPECTED_CWD" ]; then
  echo "wrong args: $*" >&2
  exit 43
fi
case " $* " in *" --json "*) ;; *) echo "wrong args: $*" >&2; exit 43 ;; esac
case " $* " in *" --model gpt-5.6-sol "*) ;; *) echo "missing model args: $*" >&2; exit 43 ;; esac
case " $* " in *' -c model_reasoning_effort="max" '*) ;; *) echo "missing effort args: $*" >&2; exit 43 ;; esac
input=$(/bin/cat)
case "$input" in *"$EXPECTED_INPUT"*) ;; *) echo "wrong input: $input" >&2; exit 44 ;; esac
printf '{"type":"item.completed","item":{"type":"agent_message","text":"[\"Which output should OpenAI produce?\"]"}}\n'
`
	writeExecutable(t, script, body)
	writeRunnerConfig(t, home, "codex", script, configHome, "gpt-5.6-sol", "max")
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CODEX_BIN", "")
	t.Setenv("LINER_CODEX_HOME", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("EXPECTED_CONFIG_HOME", configHome)
	t.Setenv("EXPECTED_CWD", project)
	t.Setenv("EXPECTED_CWD_REAL", evaluatedPath(t, project))
	t.Setenv("EXPECTED_INPUT", "Help my AI agent read references.")

	got, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err != nil {
		t.Fatalf("%v\n%s", err, clarifyErrorLog(err))
	}
	want := []string{"Which output should OpenAI produce?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("questions = %#v, want %#v", got, want)
	}
}

func TestGenerateClarifyingQuestionsUsesSavedClaudeProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	configHome := filepath.Join(home, "saved-claude-home")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(home, "saved-claude")
	body := `#!/bin/sh
if [ "$CLAUDE_CONFIG_DIR" != "$EXPECTED_CONFIG_HOME" ]; then
  echo "wrong CLAUDE_CONFIG_DIR: $CLAUDE_CONFIG_DIR" >&2
  exit 41
fi
if [ "$PWD" != "$EXPECTED_CWD_REAL" ]; then
  echo "wrong cwd: $PWD" >&2
  exit 42
fi
if [ "$*" != "-p --output-format stream-json --verbose --model opus" ]; then
  echo "wrong args: $*" >&2
  exit 43
fi
input=$(/bin/cat)
case "$input" in *"$EXPECTED_INPUT"*) ;; *) echo "wrong input: $input" >&2; exit 44 ;; esac
printf '{"type":"result","result":"[\"Which output should Claude produce?\"]"}\n'
`
	writeExecutable(t, script, body)
	writeRunnerConfig(t, home, "claude", script, configHome, "opus")
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_CLAUDE_BIN", "")
	t.Setenv("LINER_CLAUDE_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("EXPECTED_CONFIG_HOME", configHome)
	t.Setenv("EXPECTED_CWD", project)
	t.Setenv("EXPECTED_CWD_REAL", evaluatedPath(t, project))
	t.Setenv("EXPECTED_INPUT", "Help my AI agent read references.")

	got, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err != nil {
		t.Fatalf("%v\n%s", err, clarifyErrorLog(err))
	}
	want := []string{"Which output should Claude produce?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("questions = %#v, want %#v", got, want)
	}
}

func TestGenerateClarifyingQuestionsStopsOnExplicitModelRejection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	script := filepath.Join(home, "codex")
	counter := filepath.Join(home, "attempts")
	body := `#!/bin/sh
count=0
if [ -f "$LINER_RETRY_COUNT" ]; then count=$(cat "$LINER_RETRY_COUNT"); fi
printf "%s" "$((count + 1))" > "$LINER_RETRY_COUNT"
echo 'The model gpt-5.6-sol does not exist or you do not have access to model' >&2
exit 1
`
	writeExecutable(t, script, body)
	writeRunnerConfig(t, home, "codex", script, filepath.Join(home, "codex-home"), "gpt-5.6-sol")
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_RETRY_COUNT", counter)

	_, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err == nil {
		t.Fatal("expected model rejection")
	}
	message := err.Error()
	for _, expected := range []string{"rejected configured model", "gpt-5.6-sol", "Settings", "did not substitute"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error missing %q: %s", expected, message)
		}
	}
	attempts, readErr := os.ReadFile(counter)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.TrimSpace(string(attempts)) != "1" {
		t.Fatalf("explicit model rejection should not retry, attempts = %q", attempts)
	}
}

func TestGenerateClarifyingQuestionsStopsOnExplicitEffortRejection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	script := filepath.Join(home, "codex")
	counter := filepath.Join(home, "attempts")
	body := `#!/bin/sh
count=0
if [ -f "$LINER_RETRY_COUNT" ]; then count=$(cat "$LINER_RETRY_COUNT"); fi
printf "%s" "$((count + 1))" > "$LINER_RETRY_COUNT"
echo 'invalid model_reasoning_effort max for this model' >&2
exit 1
`
	writeExecutable(t, script, body)
	writeRunnerConfig(t, home, "codex", script, filepath.Join(home, "codex-home"), "gpt-5.6-sol", "max")
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_RETRY_COUNT", counter)

	_, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err == nil {
		t.Fatal("expected effort rejection")
	}
	message := err.Error()
	for _, expected := range []string{"rejected configured Thinking effort", "max", "Settings", "did not substitute"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error missing %q: %s", expected, message)
		}
	}
	attempts, readErr := os.ReadFile(counter)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.TrimSpace(string(attempts)) != "1" {
		t.Fatalf("explicit effort rejection should not retry, attempts = %q", attempts)
	}
}

func TestGenerateClarifyingQuestionsFallsBackFromAutoModelPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	script := filepath.Join(home, "codex")
	counter := filepath.Join(home, "attempts")
	body := `#!/bin/sh
count=0
if [ -f "$LINER_RETRY_COUNT" ]; then count=$(/bin/cat "$LINER_RETRY_COUNT"); fi
printf "%s" "$((count + 1))" > "$LINER_RETRY_COUNT"
case "$*" in
  *model_reasoning_effort*) echo 'invalid model_reasoning_effort high for this model' >&2; exit 1 ;;
esac
printf '{"type":"thread.started","thread_id":"auto-fallback"}\n'
printf '{"type":"item.completed","item":{"type":"agent_message","text":"[\"Which fallback output matters?\"]"}}\n'
`
	writeExecutable(t, script, body)
	writeRunnerConfig(t, home, "codex", script, filepath.Join(home, "codex-home"))
	configPath := filepath.Join(home, ".liner", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configData = append(configData, []byte("provider_preferences:\n  codex:\n    model_mode: auto\n")...)
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("LINER_AGENT", "")
	t.Setenv("LINER_RETRY_COUNT", counter)

	got, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err != nil {
		t.Fatalf("%v\n%s", err, clarifyErrorLog(err))
	}
	want := []string{"Which fallback output matters?"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("questions = %#v, want %#v", got, want)
	}
	attempts, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(attempts)) != "2" {
		t.Fatalf("Auto model policy rejection should retry once, attempts = %q", attempts)
	}
}

func TestGenerateClarifyingQuestionsKeepsRawFailureInLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	script := filepath.Join(home, "codex")
	body := `#!/bin/sh
echo "raw-provider-retry: unexpected status 401 Unauthorized: Missing bearer authentication" >&2
exit 1
`
	writeExecutable(t, script, body)
	t.Setenv("HOME", home)
	t.Setenv("LINER_AGENT", "codex")
	t.Setenv("LINER_CODEX_BIN", script)
	t.Setenv("LINER_CODEX_HOME", filepath.Join(home, "codex-home"))

	_, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err == nil {
		t.Fatal("expected authentication failure")
	}
	message := err.Error()
	if strings.Contains(message, "raw-provider-retry") {
		t.Fatalf("primary error leaked raw provider output: %s", message)
	}
	if !strings.Contains(strings.ToLower(message), "authentication") {
		t.Fatalf("primary error is not actionable: %s", message)
	}
	match := regexp.MustCompile(`Full logs: ([^\n]+)`).FindStringSubmatch(message)
	if len(match) != 2 {
		t.Fatalf("primary error does not link the raw log: %s", message)
	}
	logData := readClarifyLogs(t, strings.TrimSpace(match[1]))
	if !strings.Contains(string(logData), "raw-provider-retry") {
		t.Fatalf("raw log omitted provider output:\n%s", logData)
	}
	for _, marker := range []string{`"type":"_liner_meta"`, `"taskLabel":"jtbd-clarify"`, `"type":"_liner_close"`, `"exitCode":1`} {
		if !strings.Contains(string(logData), marker) {
			t.Fatalf("JSONL log omitted %q:\n%s", marker, logData)
		}
	}
	scanner := bufio.NewScanner(bytes.NewReader(logData))
	for scanner.Scan() {
		if !json.Valid(scanner.Bytes()) {
			t.Fatalf("run log contains invalid JSONL line %q:\n%s", scanner.Text(), logData)
		}
	}
}

func TestGenerateClarifyingQuestionsLogsMalformedSuccessfulOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	script := filepath.Join(home, "codex")
	writeExecutable(t, script, "#!/bin/sh\necho 'raw-success-but-not-json'\n")
	t.Setenv("HOME", home)
	t.Setenv("LINER_AGENT", "codex")
	t.Setenv("LINER_CODEX_BIN", script)

	_, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err == nil {
		t.Fatal("expected malformed-output failure")
	}
	if strings.Contains(err.Error(), "raw-success-but-not-json") {
		t.Fatalf("primary error leaked raw provider output: %s", err)
	}
	if log := clarifyErrorLog(err); !strings.Contains(log, "raw-success-but-not-json") {
		t.Fatalf("raw log omitted malformed successful output:\n%s", log)
	}
}

func TestGenerateClarifyingQuestionsExplainsInvocationFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}
	home := t.TempDir()
	project := t.TempDir()
	script := filepath.Join(home, "codex")
	writeExecutable(t, script, "#!/bin/sh\nprintf '{\"type\":\"turn.started\"}\\n'\necho 'warning: authentication docs unavailable' >&2\necho 'raw unknown option detail' >&2\nexit 64\n")
	t.Setenv("HOME", home)
	t.Setenv("LINER_AGENT", "codex")
	t.Setenv("LINER_CODEX_BIN", script)

	_, err := GenerateClarifyingQuestions(project, "Help my AI agent read references.", 5*time.Second)
	if err == nil {
		t.Fatal("expected invocation failure")
	}
	message := err.Error()
	for _, expected := range []string{"invocation failed", "exit status 64", "Settings"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("primary error missing %q: %s", expected, message)
		}
	}
	if strings.Contains(message, "raw unknown option detail") {
		t.Fatalf("primary error leaked raw invocation output: %s", message)
	}
	logData := clarifyErrorLog(err)
	closeCount := 0
	scanner := bufio.NewScanner(strings.NewReader(logData))
	for scanner.Scan() {
		var event map[string]any
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event["type"] != "_liner_close" {
			continue
		}
		closeCount++
		wantStderrBytes := len("warning: authentication docs unavailable\nraw unknown option detail\n")
		if event["stderrBytes"] != float64(wantStderrBytes) {
			t.Fatalf("stderrBytes = %v, want %d in log:\n%s", event["stderrBytes"], wantStderrBytes, logData)
		}
	}
	if closeCount != 2 {
		t.Fatalf("close markers = %d, want 2 in log:\n%s", closeCount, logData)
	}
}

func TestClarifyPrimaryFailureRecognizesExpiredAuthenticationToken(t *testing.T) {
	agent := Agent{ID: "codex", Name: "OpenAI Codex", Bin: "/opt/codex"}
	failure := clarifyAttemptError{
		kind:   clarifyFailureInvocation,
		cause:  errors.New("exit status 1"),
		output: "Authentication token expired; run codex login",
	}

	message := clarifyPrimaryFailure(agent, []error{failure})
	if !strings.Contains(message, "authentication is not ready") || !strings.Contains(message, "codex login") {
		t.Fatalf("expired-token message = %q", message)
	}
}

func writeExecutable(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeRunnerConfig(t *testing.T, home string, agent string, executable string, configHome string, preferences ...string) {
	t.Helper()
	path := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent: " + agent + "\nrunner:\n  agent: " + agent + "\n  executable: " + executable + "\n  config_home: " + configHome + "\n"
	if len(preferences) > 0 && strings.TrimSpace(preferences[0]) != "" {
		body += "provider_preferences:\n  " + agent + ":\n    model: " + strings.TrimSpace(preferences[0]) + "\n"
		if agent == "codex" && len(preferences) > 1 && strings.TrimSpace(preferences[1]) != "" {
			body += "    reasoning_effort: " + strings.TrimSpace(preferences[1]) + "\n"
		}
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func clarifyErrorLog(err error) string {
	match := regexp.MustCompile(`Full logs: ([^\n]+)`).FindStringSubmatch(err.Error())
	if len(match) != 2 {
		return ""
	}
	entries, _ := os.ReadDir(strings.TrimSpace(match[1]))
	var logs strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(strings.TrimSpace(match[1]), entry.Name()))
		logs.Write(data)
	}
	return logs.String()
}

func readClarifyLogs(t *testing.T, dir string) []byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read clarify log directory: %v", err)
	}
	var logs []byte
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			t.Fatalf("read clarify log: %v", readErr)
		}
		logs = append(logs, data...)
	}
	return logs
}

func evaluatedPath(t *testing.T, path string) string {
	t.Helper()
	evaluated, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return evaluated
}
