package agent

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCommandArgsBuildsHeadlessRunnerInvocation(t *testing.T) {
	runner := Runner{ScriptPath: "/repo/packages/tui/dist/agents/headless-runner.js"}

	got, err := runner.CommandArgs(RunArgs{
		Project:   "/tmp/mixtape",
		PhaseID:   "framing",
		Agent:     "codex",
		SkillPath: "/repo/docs/curation-skill",
		Resume:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"/repo/packages/tui/dist/agents/headless-runner.js",
		"--project", "/tmp/mixtape",
		"--phase", "framing",
		"--agent", "codex",
		"--skill-path", "/repo/docs/curation-skill",
		"--resume",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command args mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestCommandArgsDefaultsAgentToAuto(t *testing.T) {
	runner := Runner{ScriptPath: "/runner.js"}

	got, err := runner.CommandArgs(RunArgs{Project: "/tmp/mixtape", PhaseID: "quality"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"/runner.js", "--project", "/tmp/mixtape", "--phase", "quality", "--agent", "auto"}) {
		t.Fatalf("unexpected default args: %#v", got)
	}
}

func TestDecodeEventKeepsRawAndKnownFields(t *testing.T) {
	ok := true
	line := []byte(`{"kind":"tool_done","id":"call_1","ok":true,"preview":"wrote file"}`)

	got, err := DecodeEvent(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "tool_done" || got.ID != "call_1" || got.Preview != "wrote file" {
		t.Fatalf("decoded wrong fields: %#v", got)
	}
	if got.OK == nil || *got.OK != ok {
		t.Fatalf("expected ok pointer to be true: %#v", got.OK)
	}
	if string(got.Raw) != string(line) {
		t.Fatalf("raw event mismatch: %s", got.Raw)
	}
}

func TestScanEventsStreamsNDJSON(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		`{"kind":"runner_start","phaseId":"framing","agent":"codex"}`,
		`{"kind":"text","text":"working"}`,
		`{"kind":"runner_done","code":0}`,
		"",
	}, "\n"))
	events := make(chan Event)
	errs := make(chan error, 1)

	go func() {
		errs <- scanEvents(context.Background(), input, events)
		close(events)
	}()

	var kinds []string
	for event := range events {
		kinds = append(kinds, event.Kind)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(kinds, []string{"runner_start", "text", "runner_done"}) {
		t.Fatalf("unexpected event kinds: %#v", kinds)
	}
}

func TestStartInterruptsRunnerOnContextCancel(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh is required for this cancellation regression")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "interrupted")
	script := filepath.Join(dir, "runner.sh")
	body := strings.Join([]string{
		"#!/bin/sh",
		"trap 'echo interrupted > " + shellQuote(marker) + "; echo \"{\\\"kind\\\":\\\"runner_error\\\",\\\"message\\\":\\\"cancelled\\\"}\"; exit 130' INT TERM",
		"echo '{\"kind\":\"runner_start\",\"phaseId\":\"framing\",\"agent\":\"fake\"}'",
		"while :; do sleep 1; done",
		"",
	}, "\n")
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	run, err := Runner{NodeBin: "/bin/sh", ScriptPath: script}.Start(ctx, RunArgs{
		Project: "/tmp/mixtape",
		PhaseID: "framing",
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-run.Events:
		if event.Kind != "runner_start" {
			t.Fatalf("expected runner_start, got %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not start")
	}

	cancel()

	select {
	case <-run.Done:
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not stop after context cancellation")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("runner was not interrupted before hard kill: %v", err)
	}
}

func TestDecodeEventRejectsMissingKind(t *testing.T) {
	if _, err := DecodeEvent([]byte(`{"text":"missing"}`)); err == nil {
		t.Fatal("expected missing kind to fail")
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
