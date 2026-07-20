package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealCoreMaintenanceSmoke(t *testing.T) {
	runner, err := Resolve()
	if err != nil {
		t.Skipf("real Liner Core is unavailable: %v", err)
	}
	project := filepath.Join(t.TempDir(), "maintenance-smoke")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}

	addPlan, err := runner.PlanMaintenance(project, SourceOperation("source.add", "", map[string]any{
		"type": "web", "url": "https://example.test/one", "note": "Initial source",
	}))
	if err != nil {
		t.Fatal(err)
	}
	addReceipt, err := runner.ApplyMaintenance(project, addPlan, addPlan.ApprovalRequired)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptMatchesPlan(t, addReceipt, addPlan)
	if addReceipt.ReceiptPath == "" {
		t.Fatal("Core receipt did not expose a durable receipt path")
	}
	if _, err := os.Stat(addReceipt.ReceiptPath); err != nil {
		t.Fatalf("durable receipt path does not exist: %s: %v", addReceipt.ReceiptPath, err)
	}

	snapshot, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	sourceID := sourceIDForLocator(t, snapshot, "https://example.test/one")

	updatePlan, err := runner.PlanMaintenance(project, SourceOperation("source.update", sourceID, map[string]any{"note": "Updated source"}))
	if err != nil {
		t.Fatal(err)
	}
	updateReceipt, err := runner.ApplyMaintenance(project, updatePlan, updatePlan.ApprovalRequired)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptMatchesPlan(t, updateReceipt, updatePlan)

	replacePlan, err := runner.PlanMaintenance(project, SourceOperation("source.replace", sourceID, map[string]any{
		"type": "web", "url": "https://example.test/two", "note": "Replacement source",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if replacePlan.Risk != "semantic" || !replacePlan.ApprovalRequired {
		t.Fatalf("replacement did not preserve Core risk: %#v", replacePlan)
	}
	replaceReceipt, err := runner.ApplyMaintenance(project, replacePlan, true)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptMatchesPlan(t, replaceReceipt, replacePlan)

	snapshot, err = runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	replacementID := sourceIDForLocator(t, snapshot, "https://example.test/two")
	removePlan, err := runner.PlanMaintenance(project, SourceOperation("source.remove", replacementID, nil))
	if err != nil {
		t.Fatal(err)
	}
	removeReceipt, err := runner.ApplyMaintenance(project, removePlan, true)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptMatchesPlan(t, removeReceipt, removePlan)

	renamePlan, err := runner.PlanMaintenance(project, ProjectRenameOperation("Renamed Maintenance Smoke"))
	if err != nil {
		t.Fatal(err)
	}
	renameReceipt, err := runner.ApplyMaintenance(project, renamePlan, renamePlan.ApprovalRequired)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptMatchesPlan(t, renameReceipt, renamePlan)
	snapshot, err = runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Name != "Renamed Maintenance Smoke" {
		t.Fatalf("rename did not round-trip through Core: %#v", snapshot)
	}

	destination := filepath.Join(filepath.Dir(project), "maintenance-smoke-moved")
	movePlan, err := runner.PlanMaintenance(project, ProjectMoveOperation(destination))
	if err != nil {
		t.Fatal(err)
	}
	if !movePlan.ApprovalRequired {
		t.Fatal("Project move must preserve the Core approval requirement")
	}
	moveReceipt, err := runner.ApplyMaintenance(project, movePlan, true)
	if err != nil {
		t.Fatal(err)
	}
	assertReceiptMatchesPlan(t, moveReceipt, movePlan)
	if _, err := os.Stat(moveReceipt.ReceiptPath); err != nil {
		t.Fatalf("move receipt was not resolved at the destination root: %s: %v", moveReceipt.ReceiptPath, err)
	}
	project = destination
	snapshot, err = runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	resolvedDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Root != resolvedDestination || snapshot.ProjectID == nil || *snapshot.ProjectID != movePlan.ProjectID {
		t.Fatalf("move did not preserve Project identity at the approved root: %#v", snapshot)
	}

	purgePlan, err := runner.PlanMaintenance(project, SourceOperation("source.purge", replacementID, nil))
	if err != nil {
		t.Fatal(err)
	}
	if purgePlan.Risk != "destructive" || !purgePlan.ApprovalRequired {
		t.Fatalf("purge did not preserve destructive Core risk: %#v", purgePlan)
	}
	_, err = runner.ApplyMaintenance(project, purgePlan, false)
	var maintenanceErr *MaintenanceError
	if !errors.As(err, &maintenanceErr) || maintenanceErr.Report == nil || maintenanceErr.Report.Code != "approval_required" {
		t.Fatalf("destructive apply was not refused before mutation: %v", err)
	}
	if _, statErr := runner.InspectMaintenanceProject(project); statErr != nil && strings.TrimSpace(statErr.Error()) != "" {
		t.Fatalf("Project was not inspectable after destructive refusal: %v", statErr)
	}
}

func sourceIDForLocator(t *testing.T, snapshot MaintenanceProjectSnapshot, locator string) string {
	t.Helper()
	for _, source := range snapshot.Sources {
		if source.Locator == locator && source.SourceID != nil {
			return *source.SourceID
		}
	}
	t.Fatalf("snapshot has no Source ID for %s: %#v", locator, snapshot.Sources)
	return ""
}

func assertReceiptMatchesPlan(t *testing.T, receipt ChangeReceipt, plan ProjectChangeSet) {
	t.Helper()
	if receipt.ChangeSetID != plan.ChangeSetID || receipt.ChangeSetHash != plan.ChangeSetHash || receipt.ProjectID != plan.ProjectID {
		t.Fatalf("receipt does not match plan: receipt=%#v plan=%#v", receipt, plan)
	}
}
