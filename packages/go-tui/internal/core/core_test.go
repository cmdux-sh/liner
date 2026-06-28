package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestImportedProjectPathParsesCliOutput(t *testing.T) {
	got := importedProjectPath("Extracted /tmp/mixtapes/product-design\n", "/tmp/archive.mixtape", "/tmp/mixtapes")
	if got != "/tmp/mixtapes/product-design" {
		t.Fatalf("unexpected imported path: %q", got)
	}
}

func TestDefaultBaseDirUsesLinerDirOverride(t *testing.T) {
	home := t.TempDir()
	override := filepath.Join(home, "override-projects")
	t.Setenv("HOME", home)
	t.Setenv("LINER_DIR", override)

	if got := DefaultBaseDir(); got != override {
		t.Fatalf("expected LINER_DIR override, got %q want %q", got, override)
	}
}

func TestDefaultBaseDirUsesConfiguredProjectsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LINER_DIR", "")
	configPath := filepath.Join(home, ".liner", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("projects_dir: ~/custom-liner/projects\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(home, "custom-liner", "projects")
	if got := DefaultBaseDir(); got != want {
		t.Fatalf("expected configured projects_dir, got %q want %q", got, want)
	}
}

func TestDefaultBaseDirUsesHomeProjectLibrary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LINER_DIR", "")

	want := filepath.Join(home, "liner", "projects")
	if got := DefaultBaseDir(); got != want {
		t.Fatalf("expected home project library, got %q want %q", got, want)
	}
}

func TestImportedProjectPathStripsAnsiOutput(t *testing.T) {
	got := importedProjectPath("\x1b[32mExtracted\x1b[0m /tmp/mixtapes/product-design\n", "/tmp/archive.mixtape", "/tmp/mixtapes")
	if got != "/tmp/mixtapes/product-design" {
		t.Fatalf("unexpected imported path: %q", got)
	}
}

func TestImportedProjectPathFallsBackToArchiveName(t *testing.T) {
	got := importedProjectPath("", "/tmp/Product Design.mixtape", "/tmp/mixtapes")
	want := filepath.Join("/tmp/mixtapes", "Product Design")
	if got != want {
		t.Fatalf("unexpected fallback path: got %q want %q", got, want)
	}
}

func TestSharedArchivePathParsesCliOutput(t *testing.T) {
	got := sharedArchivePath("Wrote /tmp/mixtapes/product-design.mixtape (104 entries)\n", "/tmp/mixtapes/product-design")
	if got != "/tmp/mixtapes/product-design.mixtape" {
		t.Fatalf("unexpected archive path: %q", got)
	}
}

func TestSharedArchivePathStripsAnsiAndWrappedOutput(t *testing.T) {
	got := sharedArchivePath("\x1b[32mWrote\x1b[0m \n/tmp/mixtapes/product-design.mixtape\n(104 entries)\n", "/tmp/mixtapes/product-design")
	if got != "/tmp/mixtapes/product-design.mixtape" {
		t.Fatalf("unexpected archive path: %q", got)
	}
}

func TestSharedArchivePathFallsBackToProjectName(t *testing.T) {
	got := sharedArchivePath("", "/tmp/mixtapes/Product Design")
	want := filepath.Join("/tmp/mixtapes", "Product Design.mixtape")
	if got != want {
		t.Fatalf("unexpected fallback path: got %q want %q", got, want)
	}
}

func TestParseProjectStatus(t *testing.T) {
	raw := map[string]any{
		"progress": map[string]any{
			"step":          2,
			"total":         10,
			"current_phase": "candidates",
			"source":        "file",
		},
		"status_snapshot": map[string]any{
			"milestone": "corpus_ready",
			"stale":     true,
			"updated":   "2026-06-18T12:00:00Z",
			"corpus": map[string]any{
				"state":    "ready",
				"evidence": "mixtape/MIXTAPE.md",
			},
			"operating_layer": map[string]any{
				"state":    "pending",
				"evidence": "LINER.md",
				"audit":    "working/audits/2026-06-18-operating-layer.md",
			},
		},
		"project_skill": map[string]any{
			"status": "active",
			"name":   "UI Design",
			"path":   "skills/ui-design.md",
		},
		"phases": []map[string]any{
			{
				"id":     "framing",
				"label":  "Framing",
				"index":  0,
				"status": "complete",
				"artifact": map[string]any{
					"path":             "working/01-jtbd-and-knowledge-map.md",
					"exists":           true,
					"bytes":            120,
					"has_real_content": true,
				},
				"runs": map[string]any{
					"count":            1,
					"latest_exit_code": 0,
					"latest_log_path":  ".liner-runs/framing/run.jsonl",
				},
			},
			{
				"id":       "gate0",
				"label":    "Confirm framing",
				"index":    1,
				"status":   "complete",
				"artifact": nil,
				"gate": map[string]any{
					"key":      "gate0Accepted",
					"accepted": true,
				},
				"runs": map[string]any{"count": 0},
			},
		},
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	got, err := parseProjectStatus(payload)
	if err != nil {
		t.Fatal(err)
	}

	if got.Progress.Step != 2 || got.Progress.CurrentPhase == nil || *got.Progress.CurrentPhase != "candidates" {
		t.Fatalf("unexpected progress: %#v", got.Progress)
	}
	if got.Snapshot.Milestone != "corpus_ready" || !got.Snapshot.Stale {
		t.Fatalf("unexpected status snapshot: %#v", got.Snapshot)
	}
	if got.Snapshot.Corpus.Evidence != "mixtape/MIXTAPE.md" {
		t.Fatalf("unexpected corpus evidence: %#v", got.Snapshot.Corpus)
	}
	if got.Snapshot.OperatingLayer.Audit == nil || *got.Snapshot.OperatingLayer.Audit != "working/audits/2026-06-18-operating-layer.md" {
		t.Fatalf("unexpected operating-layer evidence: %#v", got.Snapshot.OperatingLayer)
	}
	if got.ProjectSkill.Status != "active" || got.ProjectSkill.Name == nil || *got.ProjectSkill.Name != "UI Design" {
		t.Fatalf("unexpected project skill status: %#v", got.ProjectSkill)
	}
	if len(got.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %#v", got.Phases)
	}
	if got.Phases[0].Artifact == nil || !got.Phases[0].Artifact.HasRealContent {
		t.Fatalf("expected artifact content in first phase: %#v", got.Phases[0].Artifact)
	}
	if got.Phases[0].Runs.LatestExitCode == nil || *got.Phases[0].Runs.LatestExitCode != 0 {
		t.Fatalf("expected latest exit code, got %#v", got.Phases[0].Runs)
	}
	if got.Phases[1].Gate == nil || !got.Phases[1].Gate.Accepted {
		t.Fatalf("expected accepted gate, got %#v", got.Phases[1].Gate)
	}
}

func TestProjectStatusFallsBackWhenNoWriteUnsupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script runner fixture is unix-only")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "liner")
	body := `#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = "--no-write" ]; then
    echo "No such option: --no-write" >&2
    exit 2
  fi
done
printf '{"progress":{"step":10,"total":10,"source":"fallback"},"phases":[]}\n'
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := (Runner{Command: script}).ProjectStatus("/tmp/project")
	if err != nil {
		t.Fatal(err)
	}
	if got.Progress.Source != "fallback" || got.Progress.Step != 10 {
		t.Fatalf("unexpected fallback status: %#v", got.Progress)
	}
}
