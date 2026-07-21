package app

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func TestMaintenanceGuidesSourceAddAndMetadataUpdateWithoutJSON(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "maintenance-screen")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}

	input := textinput.New()
	input.Focus()
	m := Model{runner: runner, currentPath: project, screen: screenProject, maintenanceInput: input, width: 88, height: 30}
	m, snapshotCmd := m.startMaintenance()
	snapshotMsg := snapshotCmd().(maintenanceSnapshotMsg)
	if snapshotMsg.err != nil {
		t.Fatalf("maintenance screen did not consume the Core snapshot: %#v", snapshotMsg)
	}
	m = applyMaintenanceSnapshotForTest(m, snapshotMsg)

	// Add Source is the default operation. Fill the locator field, preview the
	// exact Core plan, then apply it explicitly.
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceFieldCursor = maintenanceFieldIndex(m, "locator")
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceInput.SetValue("https://example.test/original")
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m = planAndApplyGuidedMaintenance(t, m, "source.add")
	if m.maintenanceReceipt == nil || m.maintenanceReceipt.ReceiptPath == "" {
		t.Fatal("guided Source add did not preserve the Core receipt")
	}
	refresh := inspectMaintenanceProject(runner, project)().(maintenanceSnapshotMsg)
	m = applyMaintenanceSnapshotForTest(m, refresh)
	sourceID := maintenanceTestSourceID(t, refresh.snapshot, "https://example.test/original")

	// Return to operation selection, choose Update Source, select by immutable
	// ID, and send only the touched metadata field.
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceOperation = maintenanceOperationUpdate
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	for index, source := range refresh.snapshot.Sources {
		if source.SourceID != nil && *source.SourceID == sourceID {
			m.maintenanceSourceCursor = index
			break
		}
	}
	if got := selectedMaintenanceSourceID(m); got != sourceID {
		t.Fatalf("guided picker selected %q, want %q", got, sourceID)
	}
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceFieldCursor = maintenanceFieldIndex(m, "note")
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceInput.SetValue("updated")
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	operation, err := m.guidedMaintenanceOperation()
	if err != nil {
		t.Fatal(err)
	}
	changes, _ := operation["changes"].(map[string]any)
	if len(changes) != 1 || changes["note"] != "updated" || operation["source_id"] != sourceID {
		t.Fatalf("guided update reconstructed untouched fields or identity: %#v", operation)
	}
	m = planAndApplyGuidedMaintenance(t, m, "source.update")
}

func TestMaintenanceGuidesProjectRenameAndMoveThroughRealCore(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "guided-project-maintenance")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(project, "notes", "keep.md")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("user-authored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := textinput.New()
	m := Model{runner: runner, currentPath: project, screen: screenProject, maintenanceInput: input, width: 100, height: 32}
	m, snapshotCmd := m.startMaintenance()
	snapshot := snapshotCmd().(maintenanceSnapshotMsg)
	if snapshot.err != nil || snapshot.snapshot.ProjectID == nil {
		t.Fatalf("initial maintenance inspection: snapshot=%#v err=%v", snapshot.snapshot, snapshot.err)
	}
	projectID := *snapshot.snapshot.ProjectID
	m = applyMaintenanceSnapshotForTest(m, snapshot)

	m.maintenanceOperation = maintenanceOperationRename
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceInput.SetValue("Renamed Guided Project")
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m = planAndApplyGuidedMaintenance(t, m, "project.rename")
	renameRefresh := inspectMaintenanceProject(runner, project)().(maintenanceSnapshotMsg)
	if renameRefresh.err != nil || renameRefresh.snapshot.Name != "Renamed Guided Project" || renameRefresh.snapshot.ProjectID == nil || *renameRefresh.snapshot.ProjectID != projectID {
		t.Fatalf("guided rename did not preserve Project identity: snapshot=%#v err=%v", renameRefresh.snapshot, renameRefresh.err)
	}

	m = applyMaintenanceSnapshotForTest(m, renameRefresh)
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceOperation = maintenanceOperationMove
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	destination := filepath.Join(filepath.Dir(project), "guided-project-maintenance-moved")
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceInput.SetValue(destination)
	m, _ = m.handleMaintenanceKey(keyPress("enter"))

	planning, planCmd := m.handleMaintenanceKey(keyPress("p"))
	if planCmd == nil {
		t.Fatal("guided Project move did not request a Core preview")
	}
	planned := planCmd().(maintenancePlanMsg)
	if planned.err != nil || !planned.plan.ApprovalRequired || planned.plan.Operations[0]["type"] != "project.move" {
		t.Fatalf("guided move preview lost Core risk or operation: plan=%#v err=%v", planned.plan, planned.err)
	}
	updated, _ := planning.Update(planned)
	applying, applyCmd := updated.(Model).handleMaintenanceKey(keyPress("enter"))
	if applying.currentPath != project {
		t.Fatalf("active root changed before a matching Core receipt: %q", applying.currentPath)
	}
	applied := applyCmd().(maintenanceAppliedMsg)
	if applied.err != nil {
		t.Fatal(applied.err)
	}
	updated, _ = applying.Update(applied)
	moved := updated.(Model)
	canonicalDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatal(err)
	}
	if moved.currentPath != canonicalDestination {
		t.Fatalf("matching move receipt did not activate destination: got %q want %q", moved.currentPath, canonicalDestination)
	}
	movedSnapshot := inspectMaintenanceProject(runner, moved.currentPath)().(maintenanceSnapshotMsg)
	if movedSnapshot.err != nil || movedSnapshot.snapshot.ProjectID == nil || *movedSnapshot.snapshot.ProjectID != projectID {
		t.Fatalf("guided move did not reopen immutable Project identity: snapshot=%#v err=%v", movedSnapshot.snapshot, movedSnapshot.err)
	}
	kept, err := os.ReadFile(filepath.Join(canonicalDestination, "notes", "keep.md"))
	if err != nil || string(kept) != "user-authored\n" {
		t.Fatalf("guided move did not preserve user-authored content: %q err=%v", kept, err)
	}
}

func TestMaintenanceDeletesProjectRecoverablyThroughRealCore(t *testing.T) {
	runner := testCoreRunner(t)
	base := t.TempDir()
	project := filepath.Join(base, "mistaken-project")
	if err := runner.InitProjectWithMetadata(project, "Mistaken Project", "Deletion safety test", "Test Curator", "Safely remove mistaken projects"); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(project, "notes", "keep.md")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userFile, []byte("recoverable\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := textinput.New()
	m := Model{runner: runner, baseDir: base, currentPath: project, screen: screenProject, maintenanceInput: input, width: 100, height: 32}
	m, snapshotCmd := m.startMaintenance()
	snapshot := snapshotCmd().(maintenanceSnapshotMsg)
	if snapshot.err != nil || snapshot.snapshot.ProjectID == nil {
		t.Fatalf("initial maintenance inspection: snapshot=%#v err=%v", snapshot.snapshot, snapshot.err)
	}
	m = applyMaintenanceSnapshotForTest(m, snapshot)
	m.maintenanceOperation = maintenanceOperationDelete
	m, _ = m.handleMaintenanceKey(keyPress("enter"))

	m.maintenanceFieldValues["confirmation"] = "wrong name"
	if _, err := m.guidedMaintenanceOperation(); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%q", snapshot.snapshot.Name)) {
		t.Fatalf("delete accepted a mismatched Project name: %v", err)
	}
	m.maintenanceFieldValues["confirmation"] = snapshot.snapshot.Name
	operation, err := m.guidedMaintenanceOperation()
	if err != nil {
		t.Fatal(err)
	}
	destination, _ := operation["destination"].(string)
	if operation["type"] != "project.move" || filepath.Dir(destination) != filepath.Dir(snapshot.snapshot.Root) || !strings.HasPrefix(filepath.Base(destination), ".liner-trash-mistaken-project-") {
		t.Fatalf("delete did not map to a hidden recoverable sibling move: %#v", operation)
	}

	m = planAndApplyGuidedMaintenance(t, m, "project.move")
	if _, err := os.Stat(project); !os.IsNotExist(err) {
		t.Fatalf("original Project root still exists after receipt: %v", err)
	}
	kept, err := os.ReadFile(filepath.Join(destination, "notes", "keep.md"))
	if err != nil || string(kept) != "recoverable\n" {
		t.Fatalf("recoverable delete lost user content: %q err=%v", kept, err)
	}
	if m.maintenanceReceipt == nil || m.maintenanceReceipt.ReceiptPath == "" {
		t.Fatal("recoverable delete did not preserve the Core receipt")
	}

	m, cmd := m.handleMaintenanceKey(keyPress("enter"))
	if m.screen != screenProjects || m.currentPath != "" || cmd == nil {
		t.Fatalf("delete receipt did not return to Projects: screen=%v path=%q cmd=%v", m.screen, m.currentPath, cmd)
	}
	projects := cmd().(projectsLoadedMsg)
	if projects.err != nil {
		t.Fatal(projects.err)
	}
	for _, listed := range projects.projects {
		if listed.Path == destination {
			t.Fatalf("recoverable trash root remained in the active Projects list: %#v", listed)
		}
	}
}

func TestMaintenanceGuidesReplaceRetainAndSeparateDestructivePurge(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "guided-source-lifecycle")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(project, "mixtape", "local-sources", "original.md")
	successor := filepath.Join(project, "mixtape", "local-sources", "successor.md")
	if err := os.MkdirAll(filepath.Dir(original), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(original, []byte("original evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(successor, []byte("successor evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	add, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", map[string]any{
		"type": "local_file", "path": "local-sources/original.md", "citation": "Original evidence",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, add, add.ApprovalRequired); err != nil {
		t.Fatal(err)
	}

	m := Model{runner: runner, currentPath: project, screen: screenProject, maintenanceInput: textinput.New(), width: 104, height: 34}
	m, snapshotCmd := m.startMaintenance()
	m = applyMaintenanceSnapshotForTest(m, snapshotCmd().(maintenanceSnapshotMsg))
	originalID := maintenanceTestSourceID(t, *m.maintenanceSnapshot, "local-sources/original.md")

	operationView := stripANSICodesForTest(m.maintenanceOperationView(100))
	for _, expected := range []string{"Replace Source", "Remove Source", "Retention Vault", "Purge Retained", "irreversibly delete"} {
		if !strings.Contains(operationView, expected) {
			t.Fatalf("guided lifecycle operations did not distinguish %q:\n%s", expected, operationView)
		}
	}

	m.maintenanceOperation = maintenanceOperationReplace
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	selectMaintenanceSourceForTest(t, &m, originalID)
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	setMaintenanceFieldForTest(t, &m, "locator", "local-sources/successor.md")
	setMaintenanceFieldForTest(t, &m, "citation", "Successor evidence")
	successorHash := sha256.Sum256([]byte("successor evidence\n"))
	setMaintenanceFieldForTest(t, &m, "content_hash", fmt.Sprintf("sha256:%x", successorHash))
	m = planAndApplyGuidedMaintenance(t, m, "source.replace")
	if m.maintenanceReceipt == nil || len(m.maintenanceReceipt.Lineage.Sources) != 1 || m.maintenanceReceipt.Lineage.Sources[0].Predecessor != originalID {
		t.Fatalf("replacement receipt did not expose predecessor lineage: %#v", m.maintenanceReceipt)
	}
	successorID := m.maintenanceReceipt.Lineage.Sources[0].Successor
	if view := stripANSICodesForTest(m.viewMaintenance()); !strings.Contains(view, "Lineage: "+originalID+" -> "+successorID) {
		t.Fatalf("replacement receipt view omitted Core lineage:\n%s", view)
	}

	replacedSnapshot := inspectMaintenanceProject(runner, project)().(maintenanceSnapshotMsg)
	if replacedSnapshot.err != nil {
		t.Fatal(replacedSnapshot.err)
	}
	m = applyMaintenanceSnapshotForTest(m, replacedSnapshot)
	refreshView := stripANSICodesForTest(m.viewMaintenance())
	for _, expected := range []string{"Lifecycle", "stale", "Refresh", "required", "Synthesis review"} {
		if !strings.Contains(refreshView, expected) {
			t.Fatalf("refreshed replacement Snapshot omitted lifecycle evidence %q:\n%s", expected, refreshView)
		}
	}
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceOperation = maintenanceOperationRemove
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	selectMaintenanceSourceForTest(t, &m, successorID)
	m, _ = m.handleMaintenanceKey(keyPress("enter"))

	planning, planCmd := m.handleMaintenanceKey(keyPress("p"))
	removePlan := planCmd().(maintenancePlanMsg)
	if removePlan.err != nil || removePlan.plan.Risk != "semantic" || !removePlan.plan.ApprovalRequired || len(removePlan.plan.FileEffects["retain"]) == 0 {
		t.Fatalf("remove preview did not preserve Core retention contract: plan=%#v err=%v", removePlan.plan, removePlan.err)
	}
	updated, _ := planning.Update(removePlan)
	applying, applyCmd := updated.(Model).handleMaintenanceKey(keyPress("enter"))
	removed := applyCmd().(maintenanceAppliedMsg)
	if removed.err != nil {
		t.Fatal(removed.err)
	}
	updated, _ = applying.Update(removed)
	m = updated.(Model)
	if len(removed.receipt.Lineage.RetainedSources) != 1 || removed.receipt.Lineage.RetainedSources[0] != successorID {
		t.Fatalf("remove receipt did not prove retained lineage: %#v", removed.receipt.Lineage)
	}
	if view := stripANSICodesForTest(m.viewMaintenance()); !strings.Contains(view, "Retained Source: "+successorID) {
		t.Fatalf("remove receipt view omitted retained lineage:\n%s", view)
	}
	if _, err := os.Stat(successor); err != nil {
		t.Fatalf("retention-first remove deleted successor capture: %v", err)
	}

	removedSnapshot := inspectMaintenanceProject(runner, project)().(maintenanceSnapshotMsg)
	if removedSnapshot.err != nil {
		t.Fatal(removedSnapshot.err)
	}
	m = applyMaintenanceSnapshotForTest(m, removedSnapshot)
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	m.maintenanceOperation = maintenanceOperationPurge
	m, _ = m.handleMaintenanceKey(keyPress("enter"))
	setMaintenanceFieldForTest(t, &m, "source_id", successorID)
	planning, planCmd = m.handleMaintenanceKey(keyPress("p"))
	purgePlan := planCmd().(maintenancePlanMsg)
	if purgePlan.err != nil || purgePlan.plan.Risk != "destructive" || !purgePlan.plan.ApprovalRequired || len(purgePlan.plan.FileEffects["delete"]) == 0 {
		t.Fatalf("purge preview did not preserve exact destructive contract: plan=%#v err=%v", purgePlan.plan, purgePlan.err)
	}
	updated, _ = planning.Update(purgePlan)
	applying, applyCmd = updated.(Model).handleMaintenanceKey(keyPress("enter"))
	purged := applyCmd().(maintenanceAppliedMsg)
	if purged.err != nil {
		t.Fatal(purged.err)
	}
	if len(purged.receipt.Lineage.PurgedSources) != 1 || purged.receipt.Lineage.PurgedSources[0] != successorID {
		t.Fatalf("purge receipt did not prove destructive lineage: %#v", purged.receipt.Lineage)
	}
	updated, _ = applying.Update(purged)
	m = updated.(Model)
	if view := stripANSICodesForTest(m.viewMaintenance()); !strings.Contains(view, "Purged Source: "+successorID) {
		t.Fatalf("purge receipt view omitted destructive lineage:\n%s", view)
	}
	if _, err := os.Stat(successor); !os.IsNotExist(err) {
		t.Fatalf("approved purge did not delete retained capture: %v effects=%#v", err, purgePlan.plan.FileEffects)
	}
	purgedSnapshot := inspectMaintenanceProject(runner, project)().(maintenanceSnapshotMsg)
	if purgedSnapshot.err != nil {
		t.Fatal(purgedSnapshot.err)
	}
	m = applyMaintenanceSnapshotForTest(m, purgedSnapshot)
	if m.maintenanceSnapshotPending || maintenanceTestHasSourceID(*m.maintenanceSnapshot, successorID) {
		t.Fatalf("refreshed Snapshot did not prove purged Source absence: %#v", m.maintenanceSnapshot)
	}
}

func TestInstalledMaintenanceSmokeRejectsNonAdditiveOperationsBeforeCoreResolution(t *testing.T) {
	for _, operation := range []string{
		`{"type":"source.remove","source_id":"11111111-1111-4111-8111-111111111111"}`,
		`{"type":"source.purge","source_id":"11111111-1111-4111-8111-111111111111"}`,
		`{"type":"project.move","destination":"/tmp/unsafe"}`,
	} {
		if _, err := RunInstalledMaintenanceSmoke("/tmp/project", operation); err == nil || !strings.Contains(err.Error(), "restricted to additive source.add") {
			t.Fatalf("smoke probe accepted consequential operation %s: %v", operation, err)
		}
	}
}

func TestSourceBatchDisclosesReceiptWhenLaterCorePlanFails(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "partial-source-batch")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	_, receipt, err := applySourceAdds(runner, project, []tape.Source{
		{Type: "web", URL: "https://example.test/accepted", Priority: "required"},
		{Type: "unsupported", URL: "https://example.test/rejected", Priority: "required"},
	}, nil, false)
	if err == nil || receipt == nil || receipt.ReceiptPath == "" {
		t.Fatalf("later failure must retain the earlier durable receipt: receipt=%#v err=%v", receipt, err)
	}
	snapshot, inspectErr := runner.InspectMaintenanceProject(project)
	if inspectErr != nil {
		t.Fatal(inspectErr)
	}
	if maintenanceTestSourceID(t, snapshot, "https://example.test/accepted") == "" {
		t.Fatal("accepted source was not visible after partial success")
	}
}

func TestMaintenanceLocatorMatchingNormalizesSchemesAndRejectsNearMatches(t *testing.T) {
	for _, pair := range [][2]string{
		{"/tmp/source.md", "file:///tmp/source.md"},
		{"skills/example/SKILL.md", "skill://skills/example/SKILL.md"},
		{"https://example.test/source", "https://example.test/source"},
	} {
		if !maintenanceLocatorsMatch(pair[0], pair[1]) {
			t.Fatalf("expected locators to match: %#v", pair)
		}
	}
	if maintenanceLocatorsMatch("/tmp/source.md", "file:///tmp/other.md") || maintenanceLocatorsMatch("https://example.test/a", "https://example.test/A") {
		t.Fatal("locator matching accepted a different Source")
	}
}

func TestMaintenanceLocatorResolutionReturnsEveryAmbiguousImmutableID(t *testing.T) {
	first := "11111111-1111-4111-8111-111111111111"
	second := "22222222-2222-4222-8222-222222222222"
	snapshot := core.MaintenanceProjectSnapshot{Sources: []core.MaintenanceSourceSnapshot{
		{SourceID: &first, Locator: "/tmp/source.md"},
		{SourceID: &second, Locator: "file:///tmp/source.md"},
	}}
	ids := maintenanceSourceIDsForLocator(snapshot, "file:///tmp/source.md")
	if len(ids) != 2 || ids[0] != first || ids[1] != second {
		t.Fatalf("ambiguous locator resolution silently chose one Source: %#v", ids)
	}
}

func TestSourceNoteIDResolutionPreservesEarlierReceiptsOnMissingOrAmbiguousSource(t *testing.T) {
	first := "11111111-1111-4111-8111-111111111111"
	second := "22222222-2222-4222-8222-222222222222"
	receipts := []string{"/tmp/project/.liner-runs/maintenance/receipt-one.json"}
	snapshot := core.MaintenanceProjectSnapshot{Sources: []core.MaintenanceSourceSnapshot{
		{SourceID: &first, Locator: "https://example.test/duplicate"},
		{SourceID: &second, Locator: "https://example.test/duplicate"},
	}}
	for _, locator := range []string{"https://example.test/missing", "https://example.test/duplicate"} {
		_, err := sourceNoteCleanupSourceID(snapshot, locator, locator, receipts)
		if err == nil || !strings.Contains(err.Error(), "applied 1 source-note update") || !strings.Contains(err.Error(), receipts[0]) {
			t.Fatalf("%s resolution hid earlier Core receipt: %v", locator, err)
		}
	}
}

func TestMaintenanceSourcePickerKeepsTwentySevenImmutableIdentitiesUsableAtNarrowShortSize(t *testing.T) {
	sources := make([]core.MaintenanceSourceSnapshot, 0, 27)
	for index := 0; index < 27; index++ {
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", index)
		sources = append(sources, core.MaintenanceSourceSnapshot{SourceID: &id, Type: "web", Locator: fmt.Sprintf("https://example.test/source/%02d", index)})
	}
	input := textinput.New()
	m := Model{
		width: 72, height: 18, screen: screenMaintenance, maintenanceInput: input,
		maintenanceStage:        maintenanceStageSource,
		maintenanceOperation:    maintenanceOperationUpdate,
		maintenanceSourceCursor: 26,
		maintenanceSnapshot: &core.MaintenanceProjectSnapshot{
			Name: "Example", Root: "/tmp/example", Revision: "sha256:abc",
			Compatibility: core.MaintenanceCompatibility{State: "current"},
			Sources:       sources,
		},
	}
	view := m.viewMaintenance()
	for _, expected := range []string{"Source 27 of 27", "00000000-0000-4000-8000-000000000026", "https://example.test/source/26"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("maintenance view omitted %q:\n%s", expected, view)
		}
	}
	if strings.Contains(view, "source.add\"") || strings.Contains(view, "operation as JSON") {
		t.Fatalf("ordinary guided maintenance exposed raw JSON:\n%s", view)
	}
}

func TestMaintenancePlanAlwaysWaitsForExplicitReviewedApply(t *testing.T) {
	input := textinput.New()
	base := Model{screen: screenMaintenance, maintenanceInput: input, currentPath: "/tmp/project", maintenanceStage: maintenanceStageFields}
	nonDestructive := core.ProjectChangeSet{ApprovalRequired: false}
	updated, applyCmd := base.Update(maintenancePlanMsg{plan: nonDestructive})
	got := updated.(Model)
	if got.maintenanceLoading || applyCmd != nil || got.maintenanceStage != maintenanceStagePreview || !strings.Contains(got.note, "exact Core Change Set") {
		t.Fatalf("metadata Change Set skipped stable preview: loading=%t cmd=%v stage=%v note=%q", got.maintenanceLoading, applyCmd, got.maintenanceStage, got.note)
	}

	reviewed := base
	reviewedPlan := core.ProjectChangeSet{ApprovalRequired: true}
	updated, approvalCmd := reviewed.Update(maintenancePlanMsg{plan: reviewedPlan})
	got = updated.(Model)
	if got.maintenanceLoading || approvalCmd != nil || got.maintenanceStage != maintenanceStagePreview || !strings.Contains(got.note, "approval-required") {
		t.Fatalf("risky Change Set did not wait for explicit approval: loading=%t cmd=%v note=%q", got.maintenanceLoading, approvalCmd, got.note)
	}
}

func TestMaintenancePreviewShowsTheExactCoreSourcePayload(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	plan := core.ProjectChangeSet{
		Risk: "additive",
		Operations: []map[string]any{{
			"type":      "source.add",
			"source_id": id,
			"source": map[string]any{
				"type": "web",
				"url":  "https://example.test/source",
				"note": "keep  two spaces",
			},
		}},
	}
	view := maintenancePlanView(240, plan, "Enter")
	for _, exact := range []string{
		`"source_id":"11111111-1111-4111-8111-111111111111"`,
		`"url":"https://example.test/source"`,
		`"note":"keep  two spaces"`,
	} {
		if !strings.Contains(view, exact) {
			t.Fatalf("Core preview omitted or rewrote exact payload %q:\n%s", exact, view)
		}
	}
}

func TestMaintenanceEditingRoutesHomeKeyIntoTypedURL(t *testing.T) {
	input := textinput.New()
	input.Focus()
	m := Model{
		screen:                 screenMaintenance,
		maintenanceInput:       input,
		maintenanceStage:       maintenanceStageFields,
		maintenanceEditing:     true,
		maintenanceFieldValues: map[string]string{},
	}
	for _, character := range "https://example.test/source" {
		updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: character, Text: string(character)}))
		m = updated.(Model)
	}
	if m.screen != screenMaintenance || m.maintenanceInput.Value() != "https://example.test/source" {
		t.Fatalf("typed URL was intercepted by global navigation: screen=%v value=%q", m.screen, m.maintenanceInput.Value())
	}
}

func TestMaintenanceApplyDefersQuitAndReplaysAmbiguousRealCoreReceipt(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "maintenance-replay")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	plan, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", map[string]any{
		"type": "web", "url": "https://example.test/replayed", "priority": "required",
	}))
	if err != nil {
		t.Fatal(err)
	}
	m := Model{
		runner: runner, currentPath: project, screen: screenMaintenance,
		maintenanceInput: textinput.New(), maintenanceStage: maintenanceStagePreview,
		maintenancePlan: &plan,
	}
	applying, applyCmd := m.handleMaintenanceKey(keyPress("enter"))
	if !applying.maintenanceApplying || applyCmd == nil {
		t.Fatalf("approved apply did not enter protected phase: %#v cmd=%v", applying, applyCmd)
	}
	deferred, quitCmd := applying.handleMaintenanceKey(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if quitCmd != nil || !deferred.maintenanceApplying || !strings.Contains(deferred.note, "cannot be interrupted") {
		t.Fatalf("Ctrl+C did not defer during Core apply: %#v cmd=%v", deferred, quitCmd)
	}
	committed := applyCmd().(maintenanceAppliedMsg)
	if committed.err != nil {
		t.Fatal(committed.err)
	}
	ambiguousModel, _ := applying.Update(maintenanceAppliedMsg{err: os.ErrDeadlineExceeded, path: project})
	ambiguous := ambiguousModel.(Model)
	if ambiguous.maintenancePlan == nil || !ambiguous.maintenanceReconcile || ambiguous.maintenanceApplying {
		t.Fatalf("ambiguous apply did not retain exact plan for replay: %#v", ambiguous)
	}
	reconciledMsg := reconcileMaintenanceProject(runner, project, plan)().(maintenanceReconciledMsg)
	if reconciledMsg.err != nil {
		t.Fatal(reconciledMsg.err)
	}
	refreshedModel, _ := ambiguous.Update(reconciledMsg)
	reconciling := refreshedModel.(Model)
	if reconciling.maintenancePlan == nil || !strings.Contains(reconciling.note, "receipt") {
		t.Fatalf("Snapshot reconciliation lost replay route: %#v", reconciling)
	}
	locked, abandonCmd := reconciling.handleMaintenanceKey(keyPress("esc"))
	if abandonCmd != nil || locked.maintenancePlan == nil || !locked.maintenanceReconcile {
		t.Fatalf("reconciliation allowed exact plan discard: %#v cmd=%v", locked, abandonCmd)
	}
	locked, quitCmd = locked.handleMaintenanceKey(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if quitCmd != nil || locked.maintenancePlan == nil {
		t.Fatalf("reconciliation allowed quit before receipt replay: %#v cmd=%v", locked, quitCmd)
	}
	replaying, replayCmd := locked.handleMaintenanceKey(keyPress("enter"))
	replayed := replayCmd().(maintenanceAppliedMsg)
	if replayed.err != nil || replayed.receipt.ReceiptID != committed.receipt.ReceiptID || !replayed.receipt.Replayed {
		t.Fatalf("exact-plan replay did not recover durable receipt: first=%#v replay=%#v", committed.receipt, replayed)
	}
	if !replaying.maintenanceApplying {
		t.Fatal("receipt replay did not use protected apply phase")
	}
}

func TestMaintenanceMoveRecoversReceiptFromDestinationWithoutPrematureRootActivation(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "maintenance-move-replay")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(filepath.Dir(project), "maintenance-move-replay-destination")
	plan, err := runner.PlanMaintenance(project, core.ProjectMoveOperation(destination))
	if err != nil {
		t.Fatal(err)
	}
	committed, err := runner.ApplyMaintenance(project, plan, true)
	if err != nil {
		t.Fatal(err)
	}

	m := Model{
		runner: runner, currentPath: project, screen: screenMaintenance,
		maintenanceInput: textinput.New(), maintenanceStage: maintenanceStagePreview,
		maintenancePlan: &plan, maintenanceApplying: true,
	}
	updated, _ := m.Update(maintenanceAppliedMsg{err: os.ErrDeadlineExceeded, path: project})
	ambiguous := updated.(Model)
	if ambiguous.currentPath != project || !ambiguous.maintenanceReconcile {
		t.Fatalf("ambiguous move activated a root before receipt recovery: %#v", ambiguous)
	}

	reconciledMsg := reconcileMaintenanceProject(runner, project, plan)()
	updated, _ = ambiguous.Update(reconciledMsg)
	reconciled := updated.(Model)
	canonicalDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.currentPath != project || reconciled.maintenanceReplayPath != canonicalDestination {
		t.Fatalf("move reconciliation did not isolate the discovered replay root: current=%q replay=%q", reconciled.currentPath, reconciled.maintenanceReplayPath)
	}

	replaying, replayCmd := reconciled.handleMaintenanceKey(keyPress("enter"))
	replayed := replayCmd().(maintenanceAppliedMsg)
	if replayed.err != nil || replayed.receipt.ReceiptID != committed.ReceiptID || !replayed.receipt.Replayed {
		t.Fatalf("destination replay did not recover matching receipt: committed=%#v replayed=%#v", committed, replayed)
	}
	if replaying.currentPath != project {
		t.Fatalf("replay activated destination before receipt delivery: %q", replaying.currentPath)
	}
	updated, _ = replaying.Update(replayed)
	if got := updated.(Model).currentPath; got != canonicalDestination {
		t.Fatalf("matching recovered receipt did not activate destination: got %q want %q", got, canonicalDestination)
	}
}

func TestMaintenanceMoveReceiptMustMatchTheReviewedDestination(t *testing.T) {
	plan := core.ProjectChangeSet{
		ChangeSetID: "change-one", ChangeSetHash: "sha256:plan", ProjectID: "project-one",
		Operations: []map[string]any{{"type": "project.move", "new_root": "/reviewed/destination"}},
	}
	receipt := core.ChangeReceipt{
		ChangeSetID: "change-one", ChangeSetHash: "sha256:plan", ProjectID: "project-one",
		Operations: []map[string]any{{"type": "project.move", "new_root": "/different/destination"}},
	}
	if err := validateMaintenanceReceipt(plan, receipt); err == nil || !strings.Contains(err.Error(), "reviewed move destination") {
		t.Fatalf("contract-inconsistent move receipt was accepted: %v", err)
	}
	receipt.Operations = nil
	if err := validateMaintenanceReceipt(plan, receipt); err == nil || !strings.Contains(err.Error(), "reviewed move destination") {
		t.Fatalf("move receipt without the reviewed operation was accepted: %v", err)
	}
}

func TestMaintenanceSourceLifecycleReceiptMustMatchTheReviewedSource(t *testing.T) {
	plan := core.ProjectChangeSet{
		ChangeSetID: "change-one", ChangeSetHash: "sha256:plan", ProjectID: "project-one",
		Operations: []map[string]any{{"type": "source.purge", "source_id": "reviewed-source"}},
	}
	receipt := core.ChangeReceipt{
		ChangeSetID: "change-one", ChangeSetHash: "sha256:plan", ProjectID: "project-one",
		Operations: []map[string]any{{"type": "source.purge", "source_id": "different-source"}},
		Lineage:    core.MaintenanceLineage{ChangeSetID: "change-one", Sources: []core.MaintenanceSourceLineage{}, RetainedSources: []string{}, PurgedSources: []string{"different-source"}},
	}
	if err := validateMaintenanceReceipt(plan, receipt); err == nil || !strings.Contains(err.Error(), "Source operation") {
		t.Fatalf("self-consistent receipt for an unreviewed destructive Source was accepted: %v", err)
	}
}

func TestMaintenanceMoveRetriesIdentityDiscoveryBeforeAnyReceiptReplay(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "maintenance-move-discovery-retry")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(filepath.Dir(project), "maintenance-move-discovery-retry-destination")
	plan, err := runner.PlanMaintenance(project, core.ProjectMoveOperation(destination))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, plan, true); err != nil {
		t.Fatal(err)
	}
	parked := destination + "-temporarily-unavailable"
	if err := os.Rename(destination, parked); err != nil {
		t.Fatal(err)
	}

	m := Model{
		runner: runner, currentPath: project, screen: screenMaintenance,
		maintenanceInput: textinput.New(), maintenanceStage: maintenanceStagePreview,
		maintenancePlan: &plan, maintenanceReconcile: true, maintenanceLoading: true,
	}
	failedDiscovery := reconcileMaintenanceProject(runner, project, plan)().(maintenanceReconciledMsg)
	if failedDiscovery.err == nil {
		t.Fatal("identity discovery unexpectedly succeeded while both reviewed roots were unavailable")
	}
	updated, _ := m.Update(failedDiscovery)
	locked := updated.(Model)
	if locked.maintenanceReplayPath != "" || !locked.maintenanceReconcile || locked.maintenancePlan == nil {
		t.Fatalf("failed discovery did not retain a discovery-only locked plan: %#v", locked)
	}
	if err := os.Rename(parked, destination); err != nil {
		t.Fatal(err)
	}

	discovering, retryCmd := locked.handleMaintenanceKey(keyPress("enter"))
	if retryCmd == nil || !discovering.maintenanceReconcile || !discovering.maintenanceLoading {
		t.Fatalf("Enter did not retry identity discovery: %#v cmd=%v", discovering, retryCmd)
	}
	retried, ok := retryCmd().(maintenanceReconciledMsg)
	if !ok || retried.err != nil {
		t.Fatalf("discovery retry applied at the old root or failed: %#v", retried)
	}
	updated, _ = discovering.Update(retried)
	ready := updated.(Model)
	if ready.maintenanceReplayPath == "" || ready.currentPath != project {
		t.Fatalf("successful retry did not isolate the verified replay root: %#v", ready)
	}
}

func TestMaintenanceStructuredStalePlanRefusalUnlocksForReplan(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "maintenance-stale")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	stalePlan, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", map[string]any{
		"type": "web", "url": "https://example.test/stale", "priority": "required",
	}))
	if err != nil {
		t.Fatal(err)
	}
	newerPlan, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", map[string]any{
		"type": "web", "url": "https://example.test/newer", "priority": "required",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, newerPlan, newerPlan.ApprovalRequired); err != nil {
		t.Fatal(err)
	}
	m := Model{
		runner: runner, currentPath: project, screen: screenMaintenance,
		maintenanceInput: textinput.New(), maintenanceStage: maintenanceStagePreview,
		maintenancePlan: &stalePlan,
	}
	applying, applyCmd := m.handleMaintenanceKey(keyPress("enter"))
	refused := applyCmd().(maintenanceAppliedMsg)
	var maintenanceErr *core.MaintenanceError
	if refused.err == nil || !errors.As(refused.err, &maintenanceErr) || maintenanceErr.Report == nil || maintenanceErr.Report.Code != "stale_project" {
		t.Fatalf("real Core did not return structured stale_project: %v", refused.err)
	}
	updated, _ := applying.Update(refused)
	unlocked := updated.(Model)
	if unlocked.maintenanceReconcile || unlocked.maintenancePlan != nil || unlocked.maintenanceStage != maintenanceStageFields {
		t.Fatalf("definitive stale refusal entered endless receipt replay: %#v", unlocked)
	}
	if !strings.Contains(unlocked.note, "unchanged") || !unlocked.maintenanceLoading {
		t.Fatalf("definitive refusal did not explain safe refresh: %#v", unlocked)
	}
	refreshed, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	refreshedModel, _ := unlocked.Update(maintenanceSnapshotMsg{snapshot: refreshed})
	ready := refreshedModel.(Model)
	if ready.maintenanceLoading || !ready.supportsHomeShortcut() {
		t.Fatalf("refreshed field-selection state stayed navigation-locked: %#v", ready)
	}
}

func TestMaintenanceMalformedApplyReceiptStaysLockedForExactPlanReplay(t *testing.T) {
	plan := core.ProjectChangeSet{ChangeSetID: "exact-plan"}
	m := Model{
		screen: screenMaintenance, maintenanceStage: maintenanceStagePreview,
		maintenancePlan: &plan, maintenanceApplying: true,
	}
	malformed := &core.MaintenanceError{
		Action: "apply",
		Report: &core.FailureReport{
			Contract: core.FailureReportContract, Version: core.MaintenanceContractVersion,
			Code: "malformed_core_response", Message: "receipt could not be verified",
		},
		Err: errors.New("receipt hash mismatch"),
	}
	updated, _ := m.Update(maintenanceAppliedMsg{err: malformed, path: "/tmp/project"})
	locked := updated.(Model)
	if !locked.maintenanceReconcile || locked.maintenancePlan == nil || locked.maintenancePlan.ChangeSetID != "exact-plan" {
		t.Fatalf("malformed post-apply receipt discarded ambiguous exact plan: %#v", locked)
	}
}

func TestMaintenanceStaleUpdateWhoseTargetDisappearsRequiresSourceReselection(t *testing.T) {
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "maintenance-stale-update")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	add, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", map[string]any{
		"type": "web", "url": "https://example.test/target", "priority": "required",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, add, add.ApprovalRequired); err != nil {
		t.Fatal(err)
	}
	snapshot, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	targetID := maintenanceTestSourceID(t, snapshot, "https://example.test/target")
	staleUpdate, err := runner.PlanMaintenance(project, core.SourceOperation("source.update", targetID, map[string]any{"note": "stale note"}))
	if err != nil {
		t.Fatal(err)
	}
	remove, err := runner.PlanMaintenance(project, core.SourceOperation("source.remove", targetID, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, remove, remove.ApprovalRequired); err != nil {
		t.Fatal(err)
	}
	m := Model{
		runner: runner, currentPath: project, screen: screenMaintenance,
		maintenanceInput: textinput.New(), maintenanceStage: maintenanceStagePreview,
		maintenanceOperation: maintenanceOperationUpdate, maintenancePlan: &staleUpdate,
		maintenanceFieldValues: map[string]string{"note": "stale note"}, maintenanceTouched: map[string]bool{"note": true},
	}
	applying, applyCmd := m.handleMaintenanceKey(keyPress("enter"))
	refused := applyCmd().(maintenanceAppliedMsg)
	if refused.err == nil {
		t.Fatal("stale update unexpectedly applied after target removal")
	}
	updated, _ := applying.Update(refused)
	unlocked := updated.(Model)
	if unlocked.maintenanceStage != maintenanceStageSource || unlocked.maintenanceFieldValues != nil || unlocked.maintenanceTouched != nil {
		t.Fatalf("stale update retained cursor-bound target fields: stage=%v values=%#v touched=%#v", unlocked.maintenanceStage, unlocked.maintenanceFieldValues, unlocked.maintenanceTouched)
	}
	refreshed, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	refreshedModel, _ := unlocked.Update(maintenanceSnapshotMsg{snapshot: refreshed})
	ready := refreshedModel.(Model)
	if ready.maintenanceStage != maintenanceStageSource || selectedMaintenanceSourceID(ready) == targetID {
		t.Fatalf("refresh silently retargeted stale update to removed Source %s: %#v", targetID, ready.maintenanceSnapshot)
	}
}

func TestMaintenanceHomeShortcutOnlyLocksUnsafeOrEditingStates(t *testing.T) {
	for _, stage := range []maintenanceStage{maintenanceStageOperation, maintenanceStageSource, maintenanceStageFields, maintenanceStageReceipt} {
		m := Model{screen: screenMaintenance, maintenanceStage: stage}
		if !m.supportsHomeShortcut() {
			t.Fatalf("stable maintenance stage %v lost Home navigation", stage)
		}
	}
	for name, m := range map[string]Model{
		"editing":       {screen: screenMaintenance, maintenanceEditing: true},
		"loading":       {screen: screenMaintenance, maintenanceLoading: true},
		"applying":      {screen: screenMaintenance, maintenanceApplying: true},
		"reconciling":   {screen: screenMaintenance, maintenanceReconcile: true},
		"exact preview": {screen: screenMaintenance, maintenancePlan: &core.ProjectChangeSet{}},
	} {
		if m.supportsHomeShortcut() {
			t.Fatalf("%s maintenance state exposed unsafe Home navigation", name)
		}
	}
}

func TestMaintenanceShortPreviewScrollsCompletePayloadAboveFullViewFooter(t *testing.T) {
	input := textinput.New()
	plan := core.ProjectChangeSet{
		Risk: "additive", ApprovalRequired: false,
		Operations: []map[string]any{{
			"type": "source.add", "source_id": "11111111-1111-4111-8111-111111111111",
			"source": map[string]any{
				"type": "web", "url": "https://example.test/a/very/long/source/locator/that/requires/scrolling",
				"note": "A deliberately long typed note whose exact tail must remain reviewable before apply.",
			},
		}},
		FileEffects: map[string][]string{"write": {"mixtape/tape.yaml"}},
		Validation:  []string{"sentinel validation at the exact preview tail"},
	}
	m := Model{
		screen: screenMaintenance, width: 64, height: 18, help: help.New(),
		maintenanceInput: input, maintenanceStage: maintenanceStagePreview,
		maintenancePlan:     &plan,
		maintenancePlanView: viewport.New(viewport.WithWidth(56), viewport.WithHeight(4)),
	}
	m.help.SetWidth(60)
	m.syncMaintenancePlanView()
	top := stripANSICodesForTest(m.View().Content)
	if !strings.Contains(top, "Core Change Set preview") || !strings.Contains(top, "review preview") || !strings.Contains(top, "apply exact preview") {
		t.Fatalf("short preview lost stable review controls:\n%s", top)
	}
	if m.maintenancePlanView.AtBottom() {
		t.Fatal("long exact Core preview is not scrollable")
	}
	for !m.maintenancePlanView.AtBottom() {
		m, _ = m.handleMaintenanceKey(keyPress("down"))
	}
	bottom := stripANSICodesForTest(m.View().Content)
	normalizedBottom := strings.Join(strings.Fields(bottom), " ")
	if !strings.Contains(normalizedBottom, "sentinel validation at the exact preview tail") || !strings.Contains(bottom, "apply exact preview") {
		t.Fatalf("scrolling did not expose exact preview tail above stable footer:\n%s", bottom)
	}
}

func TestMaintenanceReceiptDoesNotLabelThePreApplySnapshotAsRefreshed(t *testing.T) {
	input := textinput.New()
	m := Model{
		screen:           screenMaintenance,
		maintenanceInput: input,
		maintenanceSnapshot: &core.MaintenanceProjectSnapshot{
			Revision: "before-revision",
		},
	}
	updated, _ := m.Update(maintenanceAppliedMsg{
		path: "/tmp/project",
		receipt: core.ChangeReceipt{
			ReceiptID: "receipt-one",
			After:     map[string]string{"revision": "after-revision"},
		},
	})
	got := updated.(Model)
	if got.maintenanceStage != maintenanceStageReceipt || got.maintenanceSnapshot != nil {
		t.Fatalf("receipt screen retained the pre-apply Snapshot as refreshed: stage=%v snapshot=%#v", got.maintenanceStage, got.maintenanceSnapshot)
	}
	view := got.viewMaintenance()
	if strings.Contains(view, "Refreshed Core Snapshot") || !strings.Contains(view, "Refreshing Core Snapshot") {
		t.Fatalf("receipt screen misstated refresh evidence:\n%s", view)
	}
}

func maintenanceTestSourceID(t *testing.T, snapshot core.MaintenanceProjectSnapshot, locator string) string {
	t.Helper()
	for _, source := range snapshot.Sources {
		if source.Locator == locator && source.SourceID != nil {
			return *source.SourceID
		}
	}
	t.Fatalf("snapshot has no immutable Source ID for %s: %#v", locator, snapshot.Sources)
	return ""
}

func maintenanceTestHasSourceID(snapshot core.MaintenanceProjectSnapshot, sourceID string) bool {
	for _, source := range snapshot.Sources {
		if source.SourceID != nil && *source.SourceID == sourceID {
			return true
		}
	}
	return false
}

func planAndApplyGuidedMaintenance(t *testing.T, m Model, expectedType string) Model {
	t.Helper()
	next, planCmd := m.handleMaintenanceKey(keyPress("p"))
	if planCmd == nil {
		t.Fatalf("%s did not start a Core plan", expectedType)
	}
	planMsg := planCmd().(maintenancePlanMsg)
	if planMsg.err != nil {
		t.Fatalf("%s plan: %v", expectedType, planMsg.err)
	}
	if len(planMsg.plan.Operations) == 0 || planMsg.plan.Operations[len(planMsg.plan.Operations)-1]["type"] != expectedType {
		t.Fatalf("%s mapped to unexpected Core operations: %#v", expectedType, planMsg.plan.Operations)
	}
	updated, _ := next.Update(planMsg)
	next = updated.(Model)
	next, applyCmd := next.handleMaintenanceKey(keyPress("enter"))
	if applyCmd == nil {
		t.Fatalf("%s did not apply the reviewed Core plan", expectedType)
	}
	applied := applyCmd().(maintenanceAppliedMsg)
	if applied.err != nil {
		t.Fatalf("%s apply: %v", expectedType, applied.err)
	}
	next.currentPath = applied.path
	next.maintenancePlan = nil
	next.maintenanceReceipt = &applied.receipt
	next.maintenanceLoading = false
	next.maintenanceStage = maintenanceStageReceipt
	return next
}

func applyMaintenanceSnapshotForTest(m Model, msg maintenanceSnapshotMsg) Model {
	m.maintenanceLoading = false
	m.maintenanceSnapshotPending = false
	m.maintenanceSnapshot = &msg.snapshot
	return m
}

func selectMaintenanceSourceForTest(t *testing.T, m *Model, sourceID string) {
	t.Helper()
	if m.maintenanceSnapshot == nil {
		t.Fatal("maintenance Source selection has no Core Snapshot")
	}
	for index, source := range m.maintenanceSnapshot.Sources {
		if source.SourceID != nil && *source.SourceID == sourceID {
			m.maintenanceSourceCursor = index
			return
		}
	}
	t.Fatalf("Core Snapshot has no Source %s: %#v", sourceID, m.maintenanceSnapshot.Sources)
}

func setMaintenanceFieldForTest(t *testing.T, m *Model, key string, value string) {
	t.Helper()
	m.maintenanceFieldCursor = maintenanceFieldIndex(*m, key)
	var cmd tea.Cmd
	*m, cmd = m.handleMaintenanceKey(keyPress("enter"))
	if cmd != nil || !m.maintenanceEditing {
		t.Fatalf("maintenance field %s did not enter typed editing", key)
	}
	m.maintenanceInput.SetValue(value)
	*m, _ = m.handleMaintenanceKey(keyPress("enter"))
}

func keyPress(value string) tea.KeyPressMsg {
	if value == "enter" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(value[0]), Text: value})
}
