package core

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
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
