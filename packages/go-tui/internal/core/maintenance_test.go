package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fakeProjectID = "11111111-1111-4111-8111-111111111111"
const fakeChangeSetID = "22222222-2222-4222-8222-222222222222"

func TestMaintenanceAdapterMapsInspectPlanAndApprovedApply(t *testing.T) {
	t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
	t.Setenv("LINER_FAKE_CORE_SCENARIO", "success")
	runner := maintenanceTestRunner()
	project := t.TempDir()
	marker := filepath.Join(project, "tape.yaml")
	if err := os.WriteFile(marker, []byte("unchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ProjectID == nil || *snapshot.ProjectID != fakeProjectID || snapshot.Sources[0].Locator != "https://example.test/source" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	plan, err := runner.PlanMaintenance(project, SourceOperation("source.replace", "source-one", map[string]any{"type": "web", "url": "https://example.test/new"}))
	if err != nil {
		t.Fatal(err)
	}
	if !plan.ApprovalRequired || plan.Risk != "semantic" || plan.ChangeSetID != fakeChangeSetID {
		t.Fatalf("unexpected plan: %#v", plan)
	}

	receipt, err := runner.ApplyMaintenance(project, plan, true)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ChangeSetID != plan.ChangeSetID || receipt.ChangeSetHash != plan.ChangeSetHash || receipt.ProjectID != plan.ProjectID {
		t.Fatalf("receipt does not match plan: %#v", receipt)
	}
	lines := strings.Join(ReceiptSummaryLines(receipt), "\n")
	if !strings.Contains(lines, "maintenance/receipt.json") || !strings.Contains(lines, "Refresh synthesis") {
		t.Fatalf("receipt summary omitted durable location or next action:\n%s", lines)
	}
	if contents, err := os.ReadFile(marker); err != nil || string(contents) != "unchanged\n" {
		t.Fatalf("fake Core adapter test mutated the Project outside Core: %q, %v", contents, err)
	}
}

func TestMaintenanceAdapterRejectsSnapshotWithoutSemanticLifecycle(t *testing.T) {
	err := validateMaintenanceProjectSnapshot(MaintenanceProjectSnapshot{
		Root:         "/tmp/project",
		Revision:     "sha256:test",
		Capabilities: map[string]bool{"inspect": true},
	})
	if err == nil || !strings.Contains(err.Error(), "lifecycle milestone") {
		t.Fatalf("expected malformed lifecycle failure, got %v", err)
	}
}

func TestMaintenanceAdapterRejectsContradictoryLifecycle(t *testing.T) {
	err := validateMaintenanceProjectSnapshot(MaintenanceProjectSnapshot{
		Root:     "/tmp/project",
		Revision: "sha256:test",
		Lifecycle: MaintenanceProjectLifecycle{
			Milestone:      "project_complete",
			Corpus:         MaintenanceLifecycleEvidence{State: "missing", Evidence: "mixtape/MIXTAPE.md"},
			OperatingLayer: MaintenanceLifecycleEvidence{State: "missing", Evidence: "LINER.md"},
			ProjectSkill:   MaintenanceProjectSkill{Status: "missing"},
		},
		Capabilities: map[string]bool{"inspect": true},
	})
	if err == nil || !strings.Contains(err.Error(), "project-complete milestone contradicts") {
		t.Fatalf("expected contradictory lifecycle failure, got %v", err)
	}
}

func TestMaintenanceAdapterAcceptsStartedCorpusRefreshWithoutOperatingArtifacts(t *testing.T) {
	err := validateMaintenanceProjectSnapshot(MaintenanceProjectSnapshot{
		Root:     "/tmp/project",
		Revision: "sha256:test",
		Lifecycle: MaintenanceProjectLifecycle{
			Milestone: "started",
			Stale:     true,
			Corpus:    MaintenanceLifecycleEvidence{State: "stale", Evidence: "mixtape/MIXTAPE.md"},
			OperatingLayer: MaintenanceLifecycleEvidence{
				State: "pending", Evidence: "LINER.md",
			},
			ProjectSkill: MaintenanceProjectSkill{Status: "missing"},
			Refresh: &MaintenanceProjectRefresh{
				State:          "required",
				Synthesis:      MaintenanceRefreshGate{State: "review_required"},
				Corpus:         MaintenanceRefreshGate{State: "compile_required"},
				OperatingLayer: MaintenanceRefreshGate{State: "approved"},
			},
		},
		Capabilities: map[string]bool{"inspect": true, "plan": true, "apply": true},
	})
	if err != nil {
		t.Fatalf("started corpus refresh without an Operating Layer should be valid: %v", err)
	}
}

func TestMaintenanceAdapterCannotBypassCoreApproval(t *testing.T) {
	t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
	t.Setenv("LINER_FAKE_CORE_SCENARIO", "must-not-run")
	_, err := maintenanceTestRunner().ApplyMaintenance("/tmp/project", fakeChangeSet(), false)
	if err == nil || !strings.Contains(err.Error(), "explicit approval") {
		t.Fatalf("expected local approval gate, got %v", err)
	}
}

func TestMaintenanceAdapterPreservesCoreFailureReport(t *testing.T) {
	t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
	t.Setenv("LINER_FAKE_CORE_SCENARIO", "failure")
	_, err := maintenanceTestRunner().ApplyMaintenance("/tmp/project", fakeChangeSet(), true)
	var maintenanceErr *MaintenanceError
	if err == nil || !strings.Contains(err.Error(), "revision is stale") || !asMaintenanceError(err, &maintenanceErr) {
		t.Fatalf("expected structured Core failure, got %v", err)
	}
	if maintenanceErr.Report == nil || maintenanceErr.Report.Code != "stale_plan" || maintenanceErr.Report.PartialSuccess {
		t.Fatalf("unexpected failure report: %#v", maintenanceErr.Report)
	}
}

func TestMaintenanceAdapterRejectsMalformedCoreResponse(t *testing.T) {
	t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
	t.Setenv("LINER_FAKE_CORE_SCENARIO", "malformed")
	_, err := maintenanceTestRunner().PlanMaintenance("/tmp/project", SourceOperation("source.add", "", map[string]any{"type": "web", "url": "https://example.test"}))
	if err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("expected malformed response refusal, got %v", err)
	}
}

func TestMaintenanceAdapterPlansSourceBatchThroughOneCoreRequest(t *testing.T) {
	t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
	t.Setenv("LINER_FAKE_CORE_SCENARIO", "batch-plan")
	runner := maintenanceTestRunner()
	sources := []map[string]any{
		{"type": "web", "url": "https://example.test/one"},
		{"type": "web", "url": "https://example.test/two"},
	}

	plan, err := runner.PlanMaintenance("/tmp/project", SourceBatchOperation(sources))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 2 || plan.Operations[0]["type"] != "source.add" || plan.Operations[1]["type"] != "source.add" {
		t.Fatalf("batch plan did not preserve both Source additions: %#v", plan.Operations)
	}
}

func TestMaintenanceAdapterRejectsIncompleteAndTamperedStructuredResponses(t *testing.T) {
	for _, scenario := range []string{"bad-plan-shape", "bad-plan-hash"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
			t.Setenv("LINER_FAKE_CORE_SCENARIO", scenario)
			_, err := maintenanceTestRunner().PlanMaintenance("/tmp/project", SourceOperation("source.add", "", map[string]any{"type": "web", "url": "https://example.test"}))
			if err == nil || !strings.Contains(err.Error(), "malformed") {
				t.Fatalf("expected %s refusal, got %v", scenario, err)
			}
		})
	}
	t.Run("bad-receipt-hash", func(t *testing.T) {
		t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
		t.Setenv("LINER_FAKE_CORE_SCENARIO", "bad-receipt-hash")
		_, err := maintenanceTestRunner().ApplyMaintenance("/tmp/project", fakeChangeSet(), true)
		if err == nil || !strings.Contains(err.Error(), "receipt_hash") {
			t.Fatalf("expected tampered receipt refusal, got %v", err)
		}
	})
	for _, scenario := range []string{"missing-receipt-lineage", "mismatched-receipt-lineage", "mismatched-receipt-operation"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("LINER_FAKE_CORE_PROCESS", "1")
			t.Setenv("LINER_FAKE_CORE_SCENARIO", scenario)
			_, err := maintenanceTestRunner().ApplyMaintenance("/tmp/project", fakeChangeSet(), true)
			expected := "lineage"
			if scenario == "mismatched-receipt-operation" {
				expected = "Source operations"
			}
			if err == nil || !strings.Contains(err.Error(), expected) {
				t.Fatalf("expected %s refusal, got %v", scenario, err)
			}
		})
	}
}

func TestMaintenancePreviewRendersCoreRiskEffectsAndValidation(t *testing.T) {
	lines := strings.Join(MaintenancePreviewLines(fakeChangeSet()), "\n")
	for _, expected := range []string{"Risk: semantic", "Approval required: true", "Operation: source.remove", `Operation payload: {"source_id":"source-one","type":"source.remove"}`, "write: mixtape/tape.yaml", "Validation: revision must match"} {
		if !strings.Contains(lines, expected) {
			t.Fatalf("preview missing %q:\n%s", expected, lines)
		}
	}
}

func TestMaintenanceCanonicalJSONMatchesPythonUnicodeAndLargeIntegerVector(t *testing.T) {
	value := map[string]any{
		"large":     json.Number("9007199254740993"),
		"literal":   `\u2028`,
		"separator": "a\u2028b\u2029c",
	}
	encoded, err := canonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"large":9007199254740993,"literal":"\\u2028","separator":"a` + "\u2028" + `b` + "\u2029" + `c"}`
	if string(encoded) != expected {
		t.Fatalf("canonical JSON differs from Python ensure_ascii=False:\n got: %q\nwant: %q", encoded, expected)
	}
	if digest := fmt.Sprintf("%x", sha256.Sum256(encoded)); digest != "2ef2e8b13f1f9a330176f6eaee88c0edc161f3892ce947e64bd8a35882d0ab5b" {
		t.Fatalf("cross-runtime hash vector = %s", digest)
	}
	var decoded map[string]any
	if err := decodeMaintenanceJSON(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if number, ok := decoded["large"].(json.Number); !ok || number.String() != "9007199254740993" {
		t.Fatalf("large integer lost precision: %#v", decoded["large"])
	}
}

func TestMaintenanceHelperProcess(t *testing.T) {
	if os.Getenv("LINER_FAKE_CORE_PROCESS") != "1" {
		return
	}
	args := helperArgs(os.Args)
	scenario := os.Getenv("LINER_FAKE_CORE_SCENARIO")
	if scenario == "must-not-run" {
		os.Exit(91)
	}
	if scenario == "malformed" {
		fmt.Print(`{"contract":"wrong","version":99}`)
		os.Exit(0)
	}
	if scenario == "failure" {
		writeJSON(FailureReport{Contract: FailureReportContract, Version: 1, Code: "stale_plan", Message: "Project revision is stale.", Recovery: []string{"Inspect and plan again."}})
		os.Exit(1)
	}
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "project inspect"):
		projectID := fakeProjectID
		sourceID := "33333333-3333-4333-8333-333333333333"
		writeJSONAndExit(MaintenanceProjectSnapshot{
			Contract: ProjectSnapshotContract, Version: 1, ProjectID: &projectID,
			Name: "Example", Root: "/tmp/project", Revision: "rev-1", ContentHash: "hash-1",
			Compatibility: MaintenanceCompatibility{State: "compatible", Message: "ready"},
			Lifecycle: MaintenanceProjectLifecycle{
				Milestone:      "started",
				Corpus:         MaintenanceLifecycleEvidence{State: "missing", Evidence: "mixtape/MIXTAPE.md"},
				OperatingLayer: MaintenanceLifecycleEvidence{State: "missing", Evidence: "LINER.md"},
				ProjectSkill:   MaintenanceProjectSkill{Status: "missing"},
			},
			Capabilities: map[string]bool{"inspect": true, "plan": true, "apply": true},
			Sources:      []MaintenanceSourceSnapshot{{SourceID: &sourceID, Type: "web", Locator: "https://example.test/source"}},
		})
	case strings.Contains(joined, "project plan"):
		if scenario == "bad-plan-shape" {
			payload := hashedPayload(fakeChangeSet(), "change_set_hash")
			delete(payload, "validation")
			writeJSONAndExit(payload)
		}
		if scenario == "bad-plan-hash" {
			payload := hashedPayload(fakeChangeSet(), "change_set_hash")
			payload["risk"] = "tampered"
			writeJSONAndExit(payload)
		}
		if scenario == "success" {
			var request struct {
				Contract  string         `json:"contract"`
				Version   int            `json:"version"`
				Operation map[string]any `json:"operation"`
			}
			_ = json.Unmarshal([]byte(argValue(args, "--request-json")), &request)
			if request.Contract != MaintenanceRequestContract || request.Version != 1 || request.Operation["type"] != "source.replace" || request.Operation["source_id"] != "source-one" {
				os.Exit(94)
			}
		}
		if scenario == "batch-plan" {
			var request struct {
				Contract  string         `json:"contract"`
				Version   int            `json:"version"`
				Operation map[string]any `json:"operation"`
			}
			_ = json.Unmarshal([]byte(argValue(args, "--request-json")), &request)
			sources, ok := request.Operation["sources"].([]any)
			if request.Contract != MaintenanceRequestContract || request.Version != 1 || request.Operation["type"] != "source.add" || !ok || len(sources) != 2 {
				os.Exit(95)
			}
			writeJSONAndExit(hashedPayload(fakeBatchChangeSet(), "change_set_hash"))
		}
		if scenario == "success" {
			writeJSONAndExit(hashedPayload(fakeReplaceChangeSet(), "change_set_hash"))
		}
		writeJSONAndExit(hashedPayload(fakeChangeSet(), "change_set_hash"))
	case strings.Contains(joined, "project apply"):
		if !containsArg(args, "--approve") {
			os.Exit(92)
		}
		var applied map[string]any
		_ = json.Unmarshal([]byte(argValue(args, "--change-set-json")), &applied)
		var operations []map[string]any
		encodedOperations, _ := json.Marshal(applied["operations"])
		_ = json.Unmarshal(encodedOperations, &operations)
		payload := hashedPayload(ChangeReceipt{
			Contract: ChangeReceiptContract, Version: 1, ReceiptID: "receipt-one",
			ChangeSetID: stringValue(applied["change_set_id"]), ChangeSetHash: stringValue(applied["change_set_hash"]), ProjectID: stringValue(applied["project_id"]),
			Before: map[string]string{"revision": "rev-1"}, After: map[string]string{"revision": "rev-2"}, Risk: "semantic",
			Operations: operations,
			FileEffects: map[string][]string{
				"create": {"mixtape/.liner-runs/maintenance/receipt.json"}, "write": {"mixtape/tape.yaml"}, "delete": {},
			},
			Validation: []string{"revision matched"}, SynthesisDisposition: "review_required",
			StaleArtifacts: []string{"mixtape/MIXTAPE.md"}, NextActions: []string{"Refresh synthesis"}, AppliedAt: "2026-07-14T00:00:00Z",
			Lineage: fakeReceiptLineage(stringValue(applied["change_set_id"]), operations),
		}, "receipt_hash")
		if scenario == "bad-receipt-hash" {
			payload["risk"] = "tampered"
		}
		if scenario == "missing-receipt-lineage" {
			delete(payload, "lineage")
			rehashPayload(payload, "receipt_hash")
		}
		if scenario == "mismatched-receipt-lineage" {
			payload["lineage"].(map[string]any)["retained_sources"] = []any{"different-source"}
			rehashPayload(payload, "receipt_hash")
		}
		if scenario == "mismatched-receipt-operation" {
			payload["operations"].([]any)[0].(map[string]any)["source_id"] = "different-source"
			payload["lineage"].(map[string]any)["retained_sources"] = []any{"different-source"}
			rehashPayload(payload, "receipt_hash")
		}
		writeJSONAndExit(payload)
	default:
		os.Exit(93)
	}
}

func maintenanceTestRunner() Runner {
	return Runner{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestMaintenanceHelperProcess", "--"},
	}
}

func fakeChangeSet() ProjectChangeSet {
	return ProjectChangeSet{
		Contract: ProjectChangeSetContract, Version: 1, ChangeSetID: fakeChangeSetID,
		ChangeSetHash: "change-hash", ProjectID: fakeProjectID, ExpectedRevision: "rev-1", ExpectedContentHash: "hash-1",
		Risk: "semantic", ApprovalRequired: true,
		Operations:  []map[string]any{{"type": "source.remove", "source_id": "source-one"}},
		FileEffects: map[string][]string{"create": {"mixtape/.liner-runs/maintenance/<receipt-id>.json"}, "write": {"mixtape/tape.yaml"}, "delete": {}},
		Validation:  []string{"revision must match"},
	}
}

func fakeReplaceChangeSet() ProjectChangeSet {
	plan := fakeChangeSet()
	plan.Operations = []map[string]any{{
		"type": "source.replace", "predecessor_source_id": "source-one", "successor_source_id": "source-two",
		"source": map[string]any{"type": "web", "url": "https://example.test/new"},
	}}
	return plan
}

func fakeBatchChangeSet() ProjectChangeSet {
	plan := fakeChangeSet()
	plan.Risk = "additive"
	plan.ApprovalRequired = false
	plan.Operations = []map[string]any{
		{"type": "source.add", "source_id": "33333333-3333-4333-8333-333333333333", "source": map[string]any{"type": "web", "url": "https://example.test/one"}},
		{"type": "source.add", "source_id": "44444444-4444-4444-8444-444444444444", "source": map[string]any{"type": "web", "url": "https://example.test/two"}},
	}
	return plan
}

func hashedPayload(value any, field string) map[string]any {
	encoded, _ := json.Marshal(value)
	var raw map[string]any
	_ = json.Unmarshal(encoded, &raw)
	rehashPayload(raw, field)
	return raw
}

func rehashPayload(raw map[string]any, field string) {
	delete(raw, field)
	canonical, _ := canonicalJSON(raw)
	raw[field] = fmt.Sprintf("%x", sha256.Sum256(canonical))
}

func fakeReceiptLineage(changeSetID string, operations []map[string]any) MaintenanceLineage {
	lineage := MaintenanceLineage{ChangeSetID: changeSetID, Sources: []MaintenanceSourceLineage{}, RetainedSources: []string{}, PurgedSources: []string{}}
	for _, operation := range operations {
		switch operation["type"] {
		case "source.replace":
			lineage.Sources = append(lineage.Sources, MaintenanceSourceLineage{
				Predecessor: stringValue(operation["predecessor_source_id"]),
				Successor:   stringValue(operation["successor_source_id"]),
			})
		case "source.remove":
			lineage.RetainedSources = append(lineage.RetainedSources, stringValue(operation["source_id"]))
		case "source.purge":
			lineage.PurgedSources = append(lineage.PurgedSources, stringValue(operation["source_id"]))
		}
	}
	return lineage
}

func helperArgs(args []string) []string {
	for index, arg := range args {
		if arg == "--" && index+1 < len(args) {
			return args[index+1:]
		}
	}
	return nil
}

func containsArg(args []string, expected string) bool {
	for _, arg := range args {
		if arg == expected {
			return true
		}
	}
	return false
}

func argValue(args []string, expected string) string {
	for index, arg := range args {
		if arg == expected && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}

func writeJSON(value any) {
	_ = json.NewEncoder(os.Stdout).Encode(value)
}

func writeJSONAndExit(value any) {
	writeJSON(value)
	os.Exit(0)
}

func asMaintenanceError(err error, target **MaintenanceError) bool {
	maintenanceErr, ok := err.(*MaintenanceError)
	if ok {
		*target = maintenanceErr
	}
	return ok
}
