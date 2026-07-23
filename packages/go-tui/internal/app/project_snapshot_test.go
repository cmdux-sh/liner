package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func TestProjectSnapshotOwnsMilestoneNextAndCapabilities(t *testing.T) {
	project := t.TempDir()
	m := projectSnapshotModel(project, core.MaintenanceProjectLifecycle{
		Milestone:      "corpus_ready",
		Corpus:         core.MaintenanceLifecycleEvidence{State: "ready", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "pending", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "missing"},
	})

	if got := m.projectMilestone(); got != "corpus_ready" {
		t.Fatalf("snapshot milestone should be authoritative, got %q", got)
	}
	if got := m.projectPrimaryLabel(); got != "Create Operating Layer" {
		t.Fatalf("snapshot Next should route to Operating Layer, got %q", got)
	}
	if m.projectCapabilities().HasLiner {
		t.Fatal("corpus-ready snapshot must not fabricate an Operating Layer capability")
	}

	m.projectSnapshot.Lifecycle.Milestone = "project_complete"
	m.projectSnapshot.Lifecycle.OperatingLayer.State = "ready"
	m.projectSnapshot.Lifecycle.ProjectSkill.Status = "active"
	if got := m.projectPrimaryLabel(); got != "Open LINER.md" {
		t.Fatalf("complete snapshot Next should open LINER.md, got %q", got)
	}
	if !m.projectCapabilities().HasLiner {
		t.Fatal("complete snapshot should expose the verified Operating Layer capability")
	}
}

func TestProjectSnapshotDoesNotAdvanceFromContradictoryLocalFiles(t *testing.T) {
	project := t.TempDir()
	for _, name := range []string{"MIXTAPE.md", "LINER.md", "SKILL.md"} {
		if err := os.WriteFile(filepath.Join(project, name), []byte("local diagnostic only\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := projectSnapshotModel(project, core.MaintenanceProjectLifecycle{
		Milestone:      "started",
		Corpus:         core.MaintenanceLifecycleEvidence{State: "missing", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "missing", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "missing"},
	})

	if got := m.projectMilestone(); got != "started" {
		t.Fatalf("local files must not advance the Snapshot milestone, got %q", got)
	}
	if m.projectCapabilities().HasLiner {
		t.Fatal("local LINER.md must remain diagnostic when the Snapshot says it is missing")
	}
	got, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen == screenPreview || got.screen == screenLinerReview {
		t.Fatalf("Enter must not route from local-file heuristics, screen=%v", got.screen)
	}
}

func TestProjectSnapshotStaleStateRoutesRefreshThroughCore(t *testing.T) {
	m := projectSnapshotModel(t.TempDir(), core.MaintenanceProjectLifecycle{
		Milestone:      "project_complete",
		Stale:          true,
		Corpus:         core.MaintenanceLifecycleEvidence{State: "ready", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "ready", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "active"},
	})

	if got := m.projectPrimaryLabel(); got != "Refresh Status" {
		t.Fatalf("stale Snapshot should expose Core refresh, got %q", got)
	}
	if got := m.projectMilestoneNextAction(); got != "Refresh Status through Liner Core." {
		t.Fatalf("stale Next copy should agree with Enter, got %q", got)
	}
	got, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil || !got.projectSnapshotRefreshing {
		t.Fatalf("Enter should start Core status refresh, refreshing=%v cmd=%v", got.projectSnapshotRefreshing, cmd)
	}
}

func TestInitialCorpusProgressPrecedesPrematureRefreshReview(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "initial-corpus")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	paths := tape.ProjectAt(project)
	if err := os.WriteFile(filepath.Join(paths.Path, ".liner-progress.json"), []byte("{\"step\":5,\"lastTouched\":\"2026-07-22T02:01:03Z\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	current, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	m := projectSnapshotModel(project, core.MaintenanceProjectLifecycle{
		Milestone:      "started",
		Stale:          true,
		Corpus:         core.MaintenanceLifecycleEvidence{State: "stale", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "pending", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "missing"},
		Refresh: &core.MaintenanceProjectRefresh{
			State:          "required",
			Synthesis:      core.MaintenanceRefreshGate{State: "review_required"},
			Corpus:         core.MaintenanceRefreshGate{State: "compile_required"},
			OperatingLayer: core.MaintenanceRefreshGate{State: "approved"},
		},
	})
	m.currentTape = current

	if got := m.projectNextKind(); got != projectNextContinueCorpus {
		t.Fatalf("initial corpus cursor must precede refresh review, got %v", got)
	}
	if got := m.projectPrimaryLabel(); got != "Continue Corpus Creation" {
		t.Fatalf("initial corpus Next must continue the saved build, got %q", got)
	}
}

func TestCompileCachedResultCannotSkipSynthesisAfterSourceRemoval(t *testing.T) {
	project := t.TempDir()
	synthesisPath := projectAbsPath(project, "synthesis.md")
	if err := os.MkdirAll(filepath.Dir(synthesisPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(synthesisPath, []byte("# Current synthesis\n\nUse the retained evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := projectSnapshotModel(project, core.MaintenanceProjectLifecycle{
		Milestone:      "corpus_ready",
		Stale:          true,
		Corpus:         core.MaintenanceLifecycleEvidence{State: "stale", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "pending", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "missing"},
		Refresh: &core.MaintenanceProjectRefresh{
			State:          "required",
			Synthesis:      core.MaintenanceRefreshGate{State: "review_required"},
			Corpus:         core.MaintenanceRefreshGate{State: "compile_required"},
			OperatingLayer: core.MaintenanceRefreshGate{State: "approved"},
		},
	})
	m.runner = testCoreRunner(t)
	m.screen = screenCompile
	m.compilePane = compilePaneSources
	m.compileRepairAttempted = true
	m.compileResult = &core.CompileResultPayload{
		Summary: core.CompileSummary{Total: 51, Succeeded: 50, Failed: 1},
	}
	m.synthesisReviewCurrent = newSynthesisReviewViewport(80, 8)
	m.synthesisReviewPlanView = newSynthesisReviewViewport(80, 12)
	m.synthesisReviewArea = newSynthesisReviewArea(80)

	got, cmd := m.continueFromCompile()

	if cmd == nil || !got.synthesisReviewLoading {
		t.Fatalf("stale Core Snapshot must prepare Synthesis review, loading=%v cmd=%v err=%q", got.synthesisReviewLoading, cmd, got.err)
	}
	if got.screen != screenCompile {
		t.Fatalf("prepared review should hold the Compile surface until planning returns, screen=%v", got.screen)
	}
	if got.synthesisReviewKind != semanticReviewSynthesis {
		t.Fatalf("expected Synthesis review, kind=%v", got.synthesisReviewKind)
	}
}

func TestReadOnlyProjectSnapshotBlocksEveryProjectWrite(t *testing.T) {
	m := projectSnapshotModel(t.TempDir(), core.MaintenanceProjectLifecycle{
		Milestone:      "corpus_ready",
		Corpus:         core.MaintenanceLifecycleEvidence{State: "ready", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "pending", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "missing"},
	})
	m.projectSnapshot.Capabilities["plan"] = false
	m.projectSnapshot.Capabilities["apply"] = false

	if got := m.nextAction(); got != "" {
		t.Fatalf("read-only Snapshot must not expose a write-backed Next, got %q", got)
	}
	help := m.helpForScreen().ShortHelp()
	for _, keyName := range []string{"enter", "a", "c", "m", "i"} {
		if hasHelp(help, keyName) {
			t.Fatalf("read-only Project footer must hide write key %q: %#v", keyName, help)
		}
	}
	for _, keyName := range []string{"enter", "a", "c", "l", "m", "i", "r"} {
		candidate := m
		keyPress := tea.KeyPressMsg(tea.Key{Code: rune(keyName[0]), Text: keyName})
		if keyName == "enter" {
			keyPress = tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
		}
		got, cmd := candidate.handleKey(keyPress)
		if cmd != nil || got.screen != screenProject || !strings.Contains(strings.ToLower(got.err), "read-only") {
			t.Fatalf("read-only key %q escaped capability gate: screen=%v err=%q cmd=%v", keyName, got.screen, got.err, cmd)
		}
	}
}

func TestOperatingLayerReviewStateOpensCoreSemanticReview(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte("# Current Operating Layer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "SKILL.md"), []byte("---\nname: launch\ndescription: Use Launch.\n---\n# Launch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillName := "launch"
	skillPath := "SKILL.md"
	m := projectSnapshotModel(project, core.MaintenanceProjectLifecycle{
		Milestone:      "project_complete",
		Stale:          true,
		Corpus:         core.MaintenanceLifecycleEvidence{State: "ready", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "stale", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "active", Name: &skillName, Path: &skillPath},
		Refresh: &core.MaintenanceProjectRefresh{
			State:          "required",
			Synthesis:      core.MaintenanceRefreshGate{State: "approved"},
			Corpus:         core.MaintenanceRefreshGate{State: "current"},
			OperatingLayer: core.MaintenanceRefreshGate{State: "review_required"},
		},
	})
	m.synthesisReviewCurrent = newSynthesisReviewViewport(100, 8)
	m.synthesisReviewPlanView = newSynthesisReviewViewport(100, 12)
	m.synthesisReviewArea = newSynthesisReviewArea(100)
	m.operatingLayerReviewSkillArea = newOperatingLayerReviewSkillArea(100)

	health := stripANSICodesForTest(m.projectHealthDetail(110))
	for _, expected := range []string{"Primary action", "Review Operating Layer", "Missing next", "Review Operating Layer"} {
		if !strings.Contains(health, expected) {
			t.Fatalf("stale Project Health missing %q:\n%s", expected, health)
		}
	}
	if strings.Contains(health, "Create Operating Layer") {
		t.Fatalf("stale Project Health should not contradict its review action:\n%s", health)
	}
	flow := projectFlowRowsFromSnapshot(*m.projectSnapshot)
	if flow[2].Evidence != "Review Operating Layer required" {
		t.Fatalf("Operating Layer flow should expose the review gate: %#v", flow[2])
	}

	if got := m.nextAction(); got != "Review Operating Layer." {
		t.Fatalf("Operating Layer review should be the current Next action, got %q", got)
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "enter") {
		t.Fatalf("Project footer should advertise the Core review surface: %#v", m.helpForScreen().ShortHelp())
	}
	got, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || got.screen != screenSynthesisReview || got.synthesisReviewKind != semanticReviewOperatingLayer || got.err != "" {
		t.Fatalf("Operating Layer review should open the Core semantic-review surface, screen=%v kind=%v err=%q cmd=%v", got.screen, got.synthesisReviewKind, got.err, cmd)
	}
	view := stripANSICodesForTest(got.viewSynthesisReview())
	for _, expected := range []string{"Review Operating Layer", "Current LINER.md", "Approve unchanged", "Edit before approval"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("Operating Layer review view missing %q:\n%s", expected, view)
		}
	}
}

func TestProjectSnapshotFailureIsReadOnlyWithRetry(t *testing.T) {
	m := Model{
		screen:                   screenProject,
		width:                    110,
		currentPath:              t.TempDir(),
		currentTape:              tape.Tape{Title: "Launch", Description: "Test Project"},
		projectSnapshotPath:      "wrong-path",
		projectSnapshotAttempted: true,
		projectSnapshotErr:       "malformed lifecycle state",
	}

	view := stripANSICodesForTest(m.viewProject())
	for _, expected := range []string{"Core Project Snapshot unavailable", "malformed lifecycle state", "Retry", "read-only"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("degraded Project view missing %q:\n%s", expected, view)
		}
	}
	if got := m.nextAction(); got != "" {
		t.Fatalf("degraded Project must not fabricate Next, got %q", got)
	}
	blocked, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))
	if cmd != nil || blocked.screen != screenProject || !strings.Contains(blocked.err, "read-only") {
		t.Fatalf("degraded mutation should fail closed, screen=%v err=%q cmd=%v", blocked.screen, blocked.err, cmd)
	}
	retrying, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))
	if cmd == nil || !retrying.projectSnapshotLoading {
		t.Fatalf("Retry should reload the Snapshot, loading=%v cmd=%v", retrying.projectSnapshotLoading, cmd)
	}
}

func TestProjectOwnsManagementActionsAndHomeDoesNotDuplicateThem(t *testing.T) {
	m := projectSnapshotModel(t.TempDir(), core.MaintenanceProjectLifecycle{
		Milestone:      "corpus_ready",
		Corpus:         core.MaintenanceLifecycleEvidence{State: "ready", Evidence: "mixtape/MIXTAPE.md"},
		OperatingLayer: core.MaintenanceLifecycleEvidence{State: "pending", Evidence: "LINER.md"},
		ProjectSkill:   core.MaintenanceProjectSkill{Status: "missing"},
	})

	for _, title := range []string{"Maintain project", "Improve Corpus", "Build Corpus", "Compile MIXTAPE.md"} {
		if hasCommandTitle(m.commandItems(), title) {
			t.Fatalf("Home must not duplicate Project action %q", title)
		}
	}
	help := m.helpForScreen().ShortHelp()
	if !hasHelp(help, "m") || !hasHelpDesc(help, "maintain") || !hasHelp(help, "i") || !hasHelpDesc(help, "improve corpus") {
		t.Fatalf("Project footer should own Maintain and Improve actions: %#v", help)
	}
	maintain, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'm', Text: "m"}))
	if maintain.screen != screenMaintenance || cmd == nil {
		t.Fatalf("m should open Project maintenance, screen=%v cmd=%v", maintain.screen, cmd)
	}
	improve, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	if improve.screen != screenImprovementReview {
		t.Fatalf("i should open Improve Corpus, screen=%v", improve.screen)
	}
}

func projectSnapshotModel(project string, lifecycle core.MaintenanceProjectLifecycle) Model {
	maintenanceInput := textinput.New()
	maintenanceInput.Focus()
	sourceInput := textinput.New()
	sourceInput.Focus()
	snapshot := &core.MaintenanceProjectSnapshot{
		Root:         project,
		Revision:     "sha256:test",
		Lifecycle:    lifecycle,
		Capabilities: map[string]bool{"inspect": true, "plan": true, "apply": true},
	}
	return Model{
		screen:                   screenProject,
		width:                    110,
		currentPath:              project,
		currentTape:              tape.Tape{Title: "Launch", Description: "Test Project"},
		projectSnapshotPath:      project,
		projectSnapshot:          snapshot,
		projectSnapshotAttempted: true,
		maintenanceInput:         maintenanceInput,
		sourceInput:              sourceInput,
	}
}
