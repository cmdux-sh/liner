package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func TestReadImprovementDeltaRejectsInferredDestructiveIntent(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(improvementWorkspacePath(project), filepath.FromSlash(improvementDeltaRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `contract: liner.improvement_delta
version: 1
summary: Fill the worked-example gap.
additions:
  - type: web
    url: https://example.com/new
    priority: required
    kind: example
removals:
  - source_id: src_existing
replacements: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := readImprovementDelta(project)
	if err == nil || !strings.Contains(err.Error(), "separate explicit maintenance") {
		t.Fatalf("expected destructive intent refusal, got %v", err)
	}
}

func TestImprovementDeltaPlansAddsAndExactDuplicatesWithoutReplacingCorpus(t *testing.T) {
	project := t.TempDir()
	existing := tape.Tape{
		Title:   "Existing corpus",
		Sources: []tape.Source{{Type: "web", URL: "https://example.com/existing", Priority: "required", Kind: stringPointer("principle")}},
	}
	if err := tape.WriteProject(project, existing); err != nil {
		t.Fatal(err)
	}
	runner := testCoreRunner(t)
	seed, err := runner.PlanMaintenance(project, core.SourceBatchOperation([]map[string]any{sourceMaintenancePayload(existing.Sources[0])}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, seed, seed.ApprovalRequired); err != nil {
		t.Fatal(err)
	}
	accepted, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	existingID := strings.TrimSpace(*accepted.Sources[0].ID)
	baseline, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}

	delta := improvementDelta{Contract: improvementDeltaContract, Version: 1, Summary: "Fill the gap", Additions: []tape.Source{
		accepted.Sources[0],
		{Type: "web", URL: "https://example.com/new", Priority: "required", Kind: stringPointer("example")},
	}}
	plan, err := planImprovementDelta(runner, project, delta, accepted, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 2 {
		t.Fatalf("expected two classified Source outcomes, got %#v", plan.Operations)
	}
	if plan.Operations[0]["type"] != "source.noop" || plan.Operations[0]["source_id"] != existingID {
		t.Fatalf("expected duplicate to retain immutable identity, got %#v", plan.Operations[0])
	}
	if plan.Operations[1]["type"] != "source.add" {
		t.Fatalf("expected focused addition, got %#v", plan.Operations[1])
	}
	for _, operation := range plan.Operations {
		if operation["type"] == "source.remove" || operation["type"] == "source.update" {
			t.Fatalf("improvement must not infer replacement/removal: %#v", operation)
		}
	}
}

func TestImprovementReviewDiscardLeavesCanonicalArtifactsUnchanged(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Stable", Sources: []tape.Source{{Type: "web", URL: "https://example.com/existing"}}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("# Stable synthesis\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeTape, _ := os.ReadFile(filepath.Join(project, "tape.yaml"))
	beforeSynthesis, _ := os.ReadFile(filepath.Join(project, "synthesis.md"))
	m := Model{
		screen:          screenImprovementReview,
		currentPath:     project,
		improvementPlan: &core.ProjectChangeSet{ChangeSetID: "cs_preview"},
	}

	got, cmd := m.handleImprovementReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd != nil || got.improvementPlan != nil {
		t.Fatalf("discard should clear only the staged preview: %#v", got.improvementPlan)
	}
	afterTape, _ := os.ReadFile(filepath.Join(project, "tape.yaml"))
	afterSynthesis, _ := os.ReadFile(filepath.Join(project, "synthesis.md"))
	if string(afterTape) != string(beforeTape) || string(afterSynthesis) != string(beforeSynthesis) {
		t.Fatal("discard changed canonical corpus artifacts")
	}
}

func TestImprovementReviewSummarizesSuggestionsWithoutCorePayload(t *testing.T) {
	note := "Role: Use as a decision-quality reference. Value: Detailed internal value. Limitation: Narrow setting."
	kind := "prescription"
	delta := &improvementDelta{Additions: []tape.Source{{
		Type: "web", URL: "https://example.com/decision-quality/guide.pdf", Kind: &kind, Note: &note,
	}}}
	plan := core.ProjectChangeSet{ChangeSetID: "internal-change-set", ChangeSetHash: "internal-hash", Operations: []map[string]any{{
		"type": "source.add", "source": map[string]any{"url": "https://example.com/decision-quality/guide.pdf", "note": note},
	}}}
	view := stripANSICodesForTest(improvementApprovalView(100, plan, delta))
	for _, expected := range []string{"Ready to add Sources", "accept 1 suggested Sources", "Existing", "Unchanged", "1 · prescription", "example.com/decision-quality/guide.pdf", "decision-quality reference"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("improvement summary missing %q:\n%s", expected, view)
		}
	}
	for _, internal := range []string{"Core Change Set", "Operation payload", "source.add", "internal-change-set", "internal-hash", "Detailed internal value", "Narrow setting"} {
		if strings.Contains(view, internal) {
			t.Fatalf("improvement summary exposed internal detail %q:\n%s", internal, view)
		}
	}
}

func TestImprovementWorkspaceIsolatesAgentWritesFromCanonicalArtifacts(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Canonical", Sources: []tape.Source{{Type: "web", URL: "https://example.com/existing"}}}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "working", "04-quality-checks.md"), []byte("canonical quality\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "synthesis.md"), []byte("canonical synthesis\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := core.MaintenanceProjectSnapshot{ProjectID: stringPointer("project_stable"), Revision: "sha256:revision", ContentHash: "sha256:content"}
	if err := prepareImprovementWorkspace(project, snapshot); err != nil {
		t.Fatal(err)
	}
	workspace := improvementWorkspacePath(project)
	stagedSnapshot, err := os.ReadFile(filepath.Join(workspace, improvementSnapshotFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stagedSnapshot), project) || !strings.Contains(string(stagedSnapshot), `"root": "."`) {
		t.Fatalf("staged Snapshot must not disclose the canonical root: %s", stagedSnapshot)
	}
	stagedDelta, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(improvementDeltaRelPath)))
	if err != nil || !stagedDelta.Mode().IsRegular() {
		t.Fatalf("improvement workspace must pre-create the scoped editable delta: info=%#v err=%v", stagedDelta, err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tape.yaml"), []byte("agent replacement\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "working", "04-quality-checks.md"), []byte("agent quality\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "synthesis.md"), []byte("agent synthesis\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	canonical, err := tape.ReadProject(project)
	if err != nil || canonical.Title != "Canonical" {
		t.Fatalf("agent workspace write escaped into canonical tape: %#v err=%v", canonical, err)
	}
	quality, _ := os.ReadFile(filepath.Join(project, "working", "04-quality-checks.md"))
	synthesis, _ := os.ReadFile(filepath.Join(project, "synthesis.md"))
	if string(quality) != "canonical quality\n" || string(synthesis) != "canonical synthesis\n" {
		t.Fatalf("agent workspace write escaped into canonical artifacts: quality=%q synthesis=%q", quality, synthesis)
	}
}

func TestImprovementAllDuplicateReviewReturnsToProjectWithoutApply(t *testing.T) {
	m := Model{
		screen:      screenImprovementReview,
		currentPath: t.TempDir(),
		improvementPlan: &core.ProjectChangeSet{Operations: []map[string]any{{
			"type": "source.noop", "source_id": "src_existing", "duplicate_classification": "exact_duplicate",
		}}},
	}

	got, cmd := m.handleImprovementReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got.screen != screenProject || got.improvementPlan != nil || cmd == nil {
		t.Fatalf("all-duplicate review should return through refreshed Project Flow without apply: screen=%v plan=%#v cmd=%v", got.screen, got.improvementPlan, cmd)
	}
	if !strings.Contains(got.note, "No canonical change was needed") {
		t.Fatalf("expected explicit no-op outcome, got %q", got.note)
	}
}

func TestImprovementReviewDisablesHomeWhilePlanIsProtected(t *testing.T) {
	m := Model{screen: screenImprovementReview, improvementPlan: &core.ProjectChangeSet{ChangeSetID: "cs_review"}}
	if m.supportsHomeShortcut() {
		t.Fatal("Home must not strand a reviewed improvement plan")
	}
	m.improvementPlan = nil
	if !m.supportsHomeShortcut() {
		t.Fatal("Home should be available on the initial improvement choice")
	}
}

func TestImprovementPassRequiresCurrentCoreSnapshot(t *testing.T) {
	m := Model{currentPath: t.TempDir(), currentTape: tape.Tape{Title: "No snapshot"}}
	got, cmd := m.startImprovementPass()
	if cmd != nil || !strings.Contains(got.err, "trustworthy current Core Project Snapshot") {
		t.Fatalf("improvement must fail closed without Core Snapshot evidence: err=%q cmd=%v", got.err, cmd)
	}
}

func TestImprovementRetryRebuildsWorkspaceFromFixedBaseline(t *testing.T) {
	project := t.TempDir()
	if err := tape.WriteProject(project, tape.Tape{Title: "Canonical retry"}); err != nil {
		t.Fatal(err)
	}
	baseline := core.MaintenanceProjectSnapshot{
		ProjectID: stringPointer("project_retry"), Revision: "sha256:revision", ContentHash: "sha256:content",
		Capabilities: map[string]bool{"plan": true, "apply": true},
	}
	if err := prepareImprovementWorkspace(project, baseline); err != nil {
		t.Fatal(err)
	}
	workspace := improvementWorkspacePath(project)
	if err := os.WriteFile(filepath.Join(workspace, "tape.yaml"), []byte("title: Corrupted staging\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, filepath.FromSlash(improvementDeltaRelPath)), []byte("contract: liner.improvement_delta\nversion: 1\nsummary: stale\nadditions:\n  - type: web\n    url: https://stale.test\n    priority: required\n    kind: example\nremovals: []\nreplacements: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "runner.cjs")
	if err := os.WriteFile(script, []byte(`process.stdout.write(JSON.stringify({ kind: "runner_start", phaseId: "improvement", agent: "codex" }) + "\n");`), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINER_HEADLESS_RUNNER", script)
	m := Model{
		currentPath: project, methodologyPhaseID: "improvement", methodologyFailed: true,
		improvementBaseline: &baseline,
	}

	got, cmd := m.retryMethodologyPhase()
	if cmd == nil || got.methodologyPhaseID != "improvement" {
		t.Fatalf("retry should start a fresh isolated run: phase=%q err=%q cmd=%v", got.methodologyPhaseID, got.err, cmd)
	}
	restored, err := tape.ReadProject(workspace)
	if err != nil || restored.Title != "Canonical retry" {
		t.Fatalf("retry reused corrupted staged tape: %#v err=%v", restored, err)
	}
	deltaBytes, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(improvementDeltaRelPath)))
	if err != nil || strings.Contains(string(deltaBytes), "stale.test") || !strings.Contains(string(deltaBytes), "additions: []") {
		t.Fatalf("retry reused stale delta: %q err=%v", deltaBytes, err)
	}
}

func TestImprovementAcceptAppliesOnceAndRoutesToSynthesisReview(t *testing.T) {
	m, _, _, _ := synthesisReviewProject(t)
	auditPath := projectAbsPath(m.currentPath, operatingFitAuditRelPath)
	audit := "# Operating-fit audit\n\nstatus: improvement_recommended\n\ngap: Missing outcome-bearing cases.\n"
	if err := os.MkdirAll(filepath.Dir(auditPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(auditPath, []byte(audit), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := *m.projectSnapshot
	originalID := strings.TrimSpace(*m.currentTape.Sources[0].ID)
	delta := improvementDelta{Contract: improvementDeltaContract, Version: 1, Summary: "Add a focused worked example", Additions: []tape.Source{{
		Type: "web", URL: "https://example.test/improvement", Priority: "required", Kind: stringPointer("example"),
	}}}
	plan, err := planImprovementDelta(m.runner, m.currentPath, delta, m.currentTape, baseline)
	if err != nil {
		t.Fatal(err)
	}
	m.screen = screenImprovementReview
	m.improvementDelta = &delta
	m.improvementBaseline = &baseline
	m.improvementPlan = &plan
	m.maintenancePlanView = newSynthesisReviewViewport(100, 12)
	m.syncImprovementPlanView()

	applying, applyCmd := m.handleImprovementReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if applyCmd == nil || !applying.improvementApplying {
		t.Fatal("reviewed improvement should start one protected Core apply")
	}
	applied := applyCmd().(improvementAppliedMsg)
	if applied.err != nil || applied.receipt.ReceiptID == "" {
		t.Fatalf("improvement apply should return one matching receipt: %#v", applied)
	}
	completedModel, planCmd := applying.Update(applied)
	completed := completedModel.(Model)
	if planCmd != nil && completed.synthesisReviewLoading {
		plannedMsg := commandMessage[synthesisReviewPlannedMsg](t, planCmd)
		plannedModel, _ := completed.Update(plannedMsg)
		completed = plannedModel.(Model)
	}
	if completed.screen != screenSynthesisReview || completed.improvementPlan != nil || completed.synthesisReviewPlan == nil || !strings.Contains(completed.note, "Sources are already approved") {
		t.Fatalf("successful improvement should route from refreshed Snapshot into Review Synthesis: screen=%v err=%q note=%q", completed.screen, completed.err, completed.note)
	}
	refreshed, err := tape.ReadProject(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Sources) != len(m.currentTape.Sources)+1 || strings.TrimSpace(*refreshed.Sources[0].ID) != originalID {
		t.Fatalf("atomic improvement must preserve accepted Source identity and add only the delta: %#v", refreshed.Sources)
	}
	resolvedAudit, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(resolvedAudit), "status: improvement_recommended") || !strings.Contains(string(resolvedAudit), "status: improvement_applied") {
		t.Fatalf("successful improvement must leave the human-readable audit visibly resolved:\n%s", resolvedAudit)
	}
	decision, err := readImprovementDecision(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Disposition != "applied" || decision.AuditSHA256 != improvementAuditSHA256(audit) {
		t.Fatalf("completion marker must retain the original audit identity: %#v", decision)
	}
	if operatingFitImprovementRecommended(m.currentPath) {
		t.Fatal("a visibly resolved audit must not reopen the improvement gate")
	}
}

func TestImprovementPassStartsDedicatedPhase(t *testing.T) {
	project := t.TempDir()
	script := filepath.Join(t.TempDir(), "runner.cjs")
	if err := os.WriteFile(script, []byte(`const value = (flag) => process.argv[process.argv.indexOf(flag) + 1] || "";
process.stdout.write(JSON.stringify({ kind: "runner_start", phaseId: value("--phase"), project: value("--project"), agent: "codex", resume: false }) + "\n");
`), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINER_HEADLESS_RUNNER", script)
	snapshot := &core.MaintenanceProjectSnapshot{
		ProjectID:    stringPointer("project_stable"),
		Root:         project,
		Revision:     "sha256:revision",
		ContentHash:  "sha256:content",
		Capabilities: map[string]bool{"plan": true, "apply": true},
	}
	m := Model{
		currentPath:              project,
		currentTape:              tape.Tape{Title: "Stable"},
		projectSnapshotPath:      project,
		projectSnapshot:          snapshot,
		projectSnapshotAttempted: true,
	}

	got, cmd := m.startImprovementPass()
	if cmd == nil {
		t.Fatal("expected improvement runner command")
	}
	msg := cmd()
	event, ok := msg.(methodologyEventMsg)
	if !ok || event.event.PhaseID != "improvement" {
		t.Fatalf("expected dedicated improvement phase, got %#v", msg)
	}
	if event.event.Project != improvementWorkspacePath(project) {
		t.Fatalf("improvement runner must receive only the isolated workspace, got %q", event.event.Project)
	}
	if got.methodologyPhaseID != "improvement" || got.methodologyPhaseIndex != -1 {
		t.Fatalf("improvement must stay outside initial methodology progress, got phase=%q index=%d", got.methodologyPhaseID, got.methodologyPhaseIndex)
	}
}

func TestImprovementProgressDoesNotMasqueradeAsBuildCorpus(t *testing.T) {
	m := Model{
		screen:             screenResearch,
		width:              100,
		height:             32,
		currentTape:        tape.Tape{Title: "Decision Support"},
		methodologyPhaseID: "improvement",
		researchLines:      []string{"Starting Improve Corpus."},
		researchSpin:       newLoadingSpinner(),
	}

	view := stripANSICodesForTest(m.viewResearch())
	for _, expected := range []string{"Improve Corpus", "Focused pass", "isolated workspace", "Canonical Project unchanged"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("focused improvement view missing %q:\n%s", expected, view)
		}
	}
	for _, misleading := range []string{"Build Corpus", "0/6 phases", "Framing", "Candidate discovery", "Assembly"} {
		if strings.Contains(view, misleading) {
			t.Fatalf("focused improvement view must not imply a six-phase rebuild with %q:\n%s", misleading, view)
		}
	}
	if next := m.nextAction(); next != "Let Liner finish the focused improvement pass." {
		t.Fatalf("focused improvement footer mismatch: %q", next)
	}
}

func TestRunningImprovementCanOpenLiveFullLog(t *testing.T) {
	m := Model{
		screen:             screenResearch,
		width:              100,
		height:             32,
		methodologyPhaseID: "improvement",
		methodologyRawLog:  []string{`{"kind":"tool_start","name":"search","query":"worked interface writing cases"}`},
	}

	opened, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'v', Text: "v"}))

	if opened.screen != screenPreview || opened.previewBack != screenResearch {
		t.Fatalf("v should open the live improvement log without stopping the pass: screen=%v back=%v", opened.screen, opened.previewBack)
	}
	if opened.methodologyPhaseID != "improvement" || opened.researchDone {
		t.Fatalf("opening the live log must preserve the running focused phase: phase=%q done=%v", opened.methodologyPhaseID, opened.researchDone)
	}
	if opened.previewRel != "Improve Corpus full log" {
		t.Fatalf("live improvement log should have a specific label, got %q", opened.previewRel)
	}
	if !strings.Contains(opened.preview.GetContent(), "worked interface writing cases") {
		t.Fatalf("live log should expose the current raw event, got %q", opened.preview.GetContent())
	}
	if !hasHelp(m.helpForScreen().ShortHelp(), "v") {
		t.Fatalf("running improvement help should advertise the live full log: %#v", m.helpForScreen().ShortHelp())
	}
}
