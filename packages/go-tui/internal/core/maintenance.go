package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

const (
	ProjectSnapshotContract    = "liner.project_snapshot"
	MaintenanceRequestContract = "liner.maintenance_request"
	ProjectChangeSetContract   = "liner.project_change_set"
	ChangeReceiptContract      = "liner.change_receipt"
	FailureReportContract      = "liner.failure_report"
	MaintenanceContractVersion = 1
)

// MaintenanceProjectSnapshot is the Core-owned, read-only view used by the TUI.
// The TUI deliberately does not parse authoritative Project files for maintenance.
type MaintenanceProjectSnapshot struct {
	Contract      string                      `json:"contract"`
	Version       int                         `json:"version"`
	ProjectID     *string                     `json:"project_id"`
	Name          string                      `json:"name"`
	Root          string                      `json:"root"`
	Revision      string                      `json:"revision"`
	ContentHash   string                      `json:"content_hash"`
	Compatibility MaintenanceCompatibility    `json:"compatibility"`
	Lifecycle     MaintenanceProjectLifecycle `json:"lifecycle"`
	Capabilities  map[string]bool             `json:"capabilities"`
	Sources       []MaintenanceSourceSnapshot `json:"sources"`
}

type MaintenanceProjectLifecycle struct {
	Milestone      string                       `json:"milestone"`
	Stale          bool                         `json:"stale"`
	Updated        string                       `json:"updated"`
	Corpus         MaintenanceLifecycleEvidence `json:"corpus"`
	OperatingLayer MaintenanceLifecycleEvidence `json:"operating_layer"`
	ProjectSkill   MaintenanceProjectSkill      `json:"project_skill"`
	Refresh        *MaintenanceProjectRefresh   `json:"refresh,omitempty"`
}

type MaintenanceLifecycleEvidence struct {
	State    string `json:"state"`
	Evidence string `json:"evidence"`
}

type MaintenanceProjectSkill struct {
	Status string  `json:"status"`
	Name   *string `json:"name,omitempty"`
	Path   *string `json:"path,omitempty"`
}

type MaintenanceProjectRefresh struct {
	State          string                 `json:"state"`
	Synthesis      MaintenanceRefreshGate `json:"synthesis"`
	Corpus         MaintenanceRefreshGate `json:"corpus"`
	OperatingLayer MaintenanceRefreshGate `json:"operating_layer"`
	Remaining      []string               `json:"remaining_artifacts"`
}

type MaintenanceRefreshGate struct {
	State string `json:"state"`
}

type MaintenanceCompatibility struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

type MaintenanceSourceSnapshot struct {
	SourceID          *string `json:"source_id"`
	Type              string  `json:"type"`
	Locator           string  `json:"locator"`
	Note              *string `json:"note"`
	Section           *string `json:"section"`
	Role              string  `json:"role"`
	ActiveInstruction bool    `json:"active_instruction"`
}

type ProjectChangeSet struct {
	Contract            string              `json:"contract"`
	Version             int                 `json:"version"`
	ChangeSetID         string              `json:"change_set_id"`
	ChangeSetHash       string              `json:"change_set_hash"`
	ProjectID           string              `json:"project_id"`
	ExpectedRevision    string              `json:"expected_revision"`
	ExpectedContentHash string              `json:"expected_content_hash"`
	Risk                string              `json:"risk"`
	ApprovalRequired    bool                `json:"approval_required"`
	Operations          []map[string]any    `json:"operations"`
	FileEffects         map[string][]string `json:"file_effects"`
	Validation          []string            `json:"validation"`
	Lifecycle           map[string]any      `json:"lifecycle,omitempty"`
}

type ChangeReceipt struct {
	Contract             string              `json:"contract"`
	Version              int                 `json:"version"`
	ReceiptID            string              `json:"receipt_id"`
	ReceiptHash          string              `json:"receipt_hash"`
	ChangeSetID          string              `json:"change_set_id"`
	ChangeSetHash        string              `json:"change_set_hash"`
	ProjectID            string              `json:"project_id"`
	Before               map[string]string   `json:"before"`
	After                map[string]string   `json:"after"`
	Risk                 string              `json:"risk"`
	Operations           []map[string]any    `json:"operations"`
	FileEffects          map[string][]string `json:"file_effects"`
	Validation           []string            `json:"validation"`
	SynthesisDisposition string              `json:"synthesis_disposition"`
	StaleArtifacts       []string            `json:"stale_artifacts"`
	NextActions          []string            `json:"next_actions"`
	AppliedAt            string              `json:"applied_at"`
	Replayed             bool                `json:"replayed"`
	Lineage              MaintenanceLineage  `json:"lineage"`
	ReceiptPath          string              `json:"-"`
}

type MaintenanceLineage struct {
	ChangeSetID     string                     `json:"change_set_id"`
	Sources         []MaintenanceSourceLineage `json:"sources"`
	RetainedSources []string                   `json:"retained_sources"`
	PurgedSources   []string                   `json:"purged_sources"`
}

type MaintenanceSourceLineage struct {
	Predecessor string `json:"predecessor"`
	Successor   string `json:"successor"`
}

type FailureReport struct {
	Contract       string   `json:"contract"`
	Version        int      `json:"version"`
	Code           string   `json:"code"`
	Message        string   `json:"message"`
	PartialSuccess bool     `json:"partial_success"`
	Recovery       []string `json:"recovery"`
}

type MaintenanceError struct {
	Action string
	Report *FailureReport
	Output string
	Err    error
}

func (e *MaintenanceError) Error() string {
	if e.Report != nil && strings.TrimSpace(e.Report.Message) != "" {
		message := e.Report.Message
		if len(e.Report.Recovery) > 0 {
			message += " Recovery: " + strings.Join(e.Report.Recovery, " ")
		}
		return message
	}
	if strings.TrimSpace(e.Output) != "" {
		return strings.TrimSpace(e.Output)
	}
	if e.Err != nil {
		return fmt.Sprintf("Liner Core %s failed: %v", e.Action, e.Err)
	}
	return "Liner Core " + e.Action + " failed"
}

func (e *MaintenanceError) Unwrap() error { return e.Err }

func (r Runner) InspectMaintenanceProject(path string) (MaintenanceProjectSnapshot, error) {
	out, err := r.Run("project", "inspect", path, "--json")
	if err != nil {
		return MaintenanceProjectSnapshot{}, maintenanceCommandError("inspect", out, err)
	}
	var snapshot MaintenanceProjectSnapshot
	if err := decodeMaintenanceContract(out, ProjectSnapshotContract, &snapshot); err != nil {
		return MaintenanceProjectSnapshot{}, err
	}
	if err := validateMaintenanceProjectSnapshot(snapshot); err != nil {
		return MaintenanceProjectSnapshot{}, err
	}
	return snapshot, nil
}

func validateMaintenanceProjectSnapshot(snapshot MaintenanceProjectSnapshot) error {
	if strings.TrimSpace(snapshot.Root) == "" || strings.TrimSpace(snapshot.Revision) == "" {
		return malformedMaintenanceResponse("inspect", "snapshot is missing root or revision")
	}
	if snapshot.Lifecycle.Milestone != "started" && snapshot.Lifecycle.Milestone != "corpus_ready" && snapshot.Lifecycle.Milestone != "project_complete" {
		return malformedMaintenanceResponse("inspect", "snapshot has an unsupported lifecycle milestone")
	}
	if strings.TrimSpace(snapshot.Lifecycle.Corpus.State) == "" || strings.TrimSpace(snapshot.Lifecycle.OperatingLayer.State) == "" || strings.TrimSpace(snapshot.Lifecycle.ProjectSkill.Status) == "" {
		return malformedMaintenanceResponse("inspect", "snapshot is missing lifecycle state")
	}
	if !oneOf(snapshot.Lifecycle.Corpus.State, "missing", "ready", "stale") || !oneOf(snapshot.Lifecycle.OperatingLayer.State, "missing", "pending", "ready", "stale") || !oneOf(snapshot.Lifecycle.ProjectSkill.Status, "missing", "active") {
		return malformedMaintenanceResponse("inspect", "snapshot has unsupported lifecycle state")
	}
	if strings.TrimSpace(snapshot.Lifecycle.Corpus.Evidence) == "" || strings.TrimSpace(snapshot.Lifecycle.OperatingLayer.Evidence) == "" {
		return malformedMaintenanceResponse("inspect", "snapshot is missing lifecycle evidence")
	}
	switch snapshot.Lifecycle.Milestone {
	case "started":
		validStarted := snapshot.Lifecycle.Corpus.State == "missing" && snapshot.Lifecycle.OperatingLayer.State == "missing"
		refresh := snapshot.Lifecycle.Refresh
		validPendingOperatingRefresh := refresh != nil &&
			refresh.State == "required" &&
			refresh.OperatingLayer.State == "approved" &&
			snapshot.Lifecycle.OperatingLayer.State == "pending"
		validStaleStarted := snapshot.Lifecycle.Stale &&
			snapshot.Lifecycle.Corpus.State == "stale" &&
			(snapshot.Lifecycle.OperatingLayer.State == "stale" || validPendingOperatingRefresh)
		if !validStarted && !validStaleStarted {
			return malformedMaintenanceResponse("inspect", "started milestone contradicts lifecycle readiness")
		}
	case "corpus_ready":
		if !oneOf(snapshot.Lifecycle.Corpus.State, "ready", "stale") || !oneOf(snapshot.Lifecycle.OperatingLayer.State, "pending", "stale") {
			return malformedMaintenanceResponse("inspect", "corpus-ready milestone contradicts lifecycle readiness")
		}
	case "project_complete":
		if !oneOf(snapshot.Lifecycle.Corpus.State, "ready", "stale") || !oneOf(snapshot.Lifecycle.OperatingLayer.State, "ready", "stale") || snapshot.Lifecycle.ProjectSkill.Status != "active" {
			return malformedMaintenanceResponse("inspect", "project-complete milestone contradicts lifecycle readiness")
		}
	}
	if !snapshot.Lifecycle.Stale && (snapshot.Lifecycle.Corpus.State == "stale" || snapshot.Lifecycle.OperatingLayer.State == "stale") {
		return malformedMaintenanceResponse("inspect", "current lifecycle contains stale artifact state")
	}
	if refresh := snapshot.Lifecycle.Refresh; refresh != nil {
		if strings.TrimSpace(refresh.State) == "" || strings.TrimSpace(refresh.Synthesis.State) == "" || strings.TrimSpace(refresh.Corpus.State) == "" || strings.TrimSpace(refresh.OperatingLayer.State) == "" {
			return malformedMaintenanceResponse("inspect", "snapshot has an incomplete refresh lifecycle")
		}
		if !oneOf(refresh.State, "required", "current") || !oneOf(refresh.Synthesis.State, "review_required", "approved") || !oneOf(refresh.Corpus.State, "compile_required", "current") || !oneOf(refresh.OperatingLayer.State, "review_required", "approved") {
			return malformedMaintenanceResponse("inspect", "snapshot has unsupported refresh lifecycle state")
		}
		if (refresh.State == "required") != snapshot.Lifecycle.Stale {
			return malformedMaintenanceResponse("inspect", "refresh lifecycle contradicts stale state")
		}
		if refresh.State == "current" && (refresh.Synthesis.State != "approved" || refresh.Corpus.State != "current" || refresh.OperatingLayer.State != "approved") {
			return malformedMaintenanceResponse("inspect", "current refresh lifecycle contains pending gates")
		}
	}
	if !snapshot.Capabilities["inspect"] {
		return malformedMaintenanceResponse("inspect", "snapshot does not declare inspect capability")
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func (r Runner) PlanMaintenance(path string, operation map[string]any) (ProjectChangeSet, error) {
	request := map[string]any{
		"contract":  MaintenanceRequestContract,
		"version":   MaintenanceContractVersion,
		"operation": operation,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return ProjectChangeSet{}, fmt.Errorf("encode maintenance request: %w", err)
	}
	out, runErr := r.Run("project", "plan", path, "--request-json", string(payload), "--json")
	if runErr != nil {
		return ProjectChangeSet{}, maintenanceCommandError("plan", out, runErr)
	}
	var changeSet ProjectChangeSet
	if err := decodeMaintenanceContract(out, ProjectChangeSetContract, &changeSet); err != nil {
		return ProjectChangeSet{}, err
	}
	if err := validateChangeSetPayload(out, changeSet); err != nil {
		return ProjectChangeSet{}, err
	}
	return changeSet, nil
}

func (r Runner) ApplyMaintenance(path string, changeSet ProjectChangeSet, approved bool) (ChangeReceipt, error) {
	if changeSet.Contract != ProjectChangeSetContract || changeSet.Version != MaintenanceContractVersion {
		return ChangeReceipt{}, malformedMaintenanceResponse("apply", "unsupported Change Set contract or version")
	}
	if changeSet.ApprovalRequired && !approved {
		return ChangeReceipt{}, &MaintenanceError{
			Action: "apply",
			Report: &FailureReport{
				Contract: FailureReportContract,
				Version:  MaintenanceContractVersion,
				Code:     "approval_required",
				Message:  "This Core Change Set requires explicit approval before apply.",
				Recovery: []string{"Review the Core preview, then approve that exact Change Set."},
			},
		}
	}
	payload, err := json.Marshal(changeSet)
	if err != nil {
		return ChangeReceipt{}, fmt.Errorf("encode Change Set: %w", err)
	}
	args := []string{"project", "apply", path, "--change-set-json", string(payload), "--json"}
	if approved {
		args = append(args, "--approve")
	}
	if destination := approvedMoveDestination(changeSet); destination != "" {
		args = append(args, "--approved-destination", destination)
	}
	out, runErr := r.Run(args...)
	if runErr != nil {
		return ChangeReceipt{}, maintenanceCommandError("apply", out, runErr)
	}
	var receipt ChangeReceipt
	if err := decodeMaintenanceContract(out, ChangeReceiptContract, &receipt); err != nil {
		return ChangeReceipt{}, err
	}
	if err := validateReceiptPayload(out, receipt); err != nil {
		return ChangeReceipt{}, err
	}
	if err := ValidateMaintenanceReceipt(changeSet, receipt); err != nil {
		return ChangeReceipt{}, malformedMaintenanceResponse("apply", err.Error())
	}
	receiptRoot := path
	if destination := approvedMoveDestination(changeSet); destination != "" {
		receiptRoot = destination
	}
	receipt.ReceiptPath = durableReceiptPath(receiptRoot, receipt)
	return receipt, nil
}

// ValidateMaintenanceReceipt binds receipt identity and consequential Source
// summaries to the exact reviewed Change Set. Receipt operations are summaries,
// so only stable lifecycle identity fields are compared here.
func ValidateMaintenanceReceipt(changeSet ProjectChangeSet, receipt ChangeReceipt) error {
	if receipt.ChangeSetID != changeSet.ChangeSetID || receipt.ChangeSetHash != changeSet.ChangeSetHash || receipt.ProjectID != changeSet.ProjectID {
		return fmt.Errorf("receipt does not match the approved Change Set")
	}
	reviewed, err := sourceLifecycleOperationIdentities(changeSet.Operations)
	if err != nil {
		return err
	}
	received, err := sourceLifecycleOperationIdentities(receipt.Operations)
	if err != nil {
		return err
	}
	if !slices.Equal(reviewed, received) {
		return fmt.Errorf("receipt Source operations do not match the approved Change Set")
	}
	return nil
}

func sourceLifecycleOperationIdentities(operations []map[string]any) ([]string, error) {
	identities := []string{}
	for _, operation := range operations {
		operationType := stringValue(operation["type"])
		switch operationType {
		case "source.replace":
			predecessor := stringValue(operation["predecessor_source_id"])
			successor := stringValue(operation["successor_source_id"])
			if predecessor == "" || successor == "" {
				return nil, fmt.Errorf("replacement Source operation is missing lineage identity")
			}
			identities = append(identities, operationType+"|"+predecessor+"|"+successor)
		case "source.remove", "source.purge":
			sourceID := stringValue(operation["source_id"])
			if sourceID == "" {
				return nil, fmt.Errorf("%s Source operation is missing source_id", operationType)
			}
			identities = append(identities, operationType+"|"+sourceID)
		}
	}
	return identities, nil
}

func SourceOperation(kind string, sourceID string, source map[string]any) map[string]any {
	operation := map[string]any{"type": kind}
	if strings.TrimSpace(sourceID) != "" {
		operation["source_id"] = strings.TrimSpace(sourceID)
	}
	if source != nil {
		key := "source"
		if kind == "source.update" {
			key = "changes"
		}
		operation[key] = source
	}
	return operation
}

func SourceBatchOperation(sources []map[string]any) map[string]any {
	return map[string]any{
		"type":    "source.add",
		"sources": sources,
	}
}

func ProjectRenameOperation(name string) map[string]any {
	return map[string]any{"type": "project.rename", "name": name}
}

func ProjectMoveOperation(destination string) map[string]any {
	return map[string]any{"type": "project.move", "destination": destination}
}

func MaintenancePreviewLines(changeSet ProjectChangeSet) []string {
	lines := []string{
		"Risk: " + fallbackMaintenanceText(changeSet.Risk, "unknown"),
		fmt.Sprintf("Approval required: %t", changeSet.ApprovalRequired),
	}
	for _, operation := range changeSet.Operations {
		lines = append(lines, "Operation: "+fallbackMaintenanceText(stringValue(operation["type"]), "unknown"))
		if payload, err := json.Marshal(operation); err == nil {
			lines = append(lines, "Operation payload: "+string(payload))
		}
	}
	effects := make([]string, 0, len(changeSet.FileEffects))
	for effect := range changeSet.FileEffects {
		effects = append(effects, effect)
	}
	sort.Strings(effects)
	for _, effect := range effects {
		for _, path := range changeSet.FileEffects[effect] {
			lines = append(lines, fmt.Sprintf("%s: %s", effect, path))
		}
	}
	for _, validation := range changeSet.Validation {
		lines = append(lines, "Validation: "+validation)
	}
	return lines
}

func ReceiptSummaryLines(receipt ChangeReceipt) []string {
	lines := []string{
		"Applied Change Set " + receipt.ChangeSetID,
		"Receipt: " + receiptPath(receipt),
	}
	if receipt.SynthesisDisposition != "" {
		lines = append(lines, "Synthesis: "+receipt.SynthesisDisposition)
	}
	for _, lineage := range receipt.Lineage.Sources {
		lines = append(lines, fmt.Sprintf("Lineage: %s -> %s", lineage.Predecessor, lineage.Successor))
	}
	for _, sourceID := range receipt.Lineage.RetainedSources {
		lines = append(lines, "Retained Source: "+sourceID)
	}
	for _, sourceID := range receipt.Lineage.PurgedSources {
		lines = append(lines, "Purged Source: "+sourceID)
	}
	for _, artifact := range receipt.StaleArtifacts {
		lines = append(lines, "Stale: "+artifact)
	}
	for _, action := range receipt.NextActions {
		lines = append(lines, "Next: "+action)
	}
	return lines
}

func receiptPath(receipt ChangeReceipt) string {
	if strings.TrimSpace(receipt.ReceiptPath) != "" {
		return receipt.ReceiptPath
	}
	for _, path := range receipt.FileEffects["create"] {
		if strings.Contains(filepath.ToSlash(path), "/maintenance/") && strings.HasSuffix(path, ".json") {
			return path
		}
	}
	return "maintenance receipt " + receipt.ReceiptID
}

func durableReceiptPath(project string, receipt ChangeReceipt) string {
	for _, path := range receipt.FileEffects["create"] {
		if !strings.Contains(filepath.ToSlash(path), "/maintenance/") || !strings.HasSuffix(path, ".json") {
			continue
		}
		path = strings.ReplaceAll(path, "<receipt-id>", receipt.ReceiptID)
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return filepath.Join(project, filepath.FromSlash(path))
	}
	return ""
}

func approvedMoveDestination(changeSet ProjectChangeSet) string {
	if !changeSet.ApprovalRequired {
		return ""
	}
	for _, operation := range changeSet.Operations {
		if stringValue(operation["type"]) == "project.move" {
			return stringValue(operation["new_root"])
		}
	}
	return ""
}

func decodeMaintenanceContract(data []byte, expected string, target any) error {
	var envelope struct {
		Contract string `json:"contract"`
		Version  int    `json:"version"`
	}
	if err := decodeMaintenanceJSON(data, &envelope); err != nil {
		return malformedMaintenanceResponse(expected, "response is not valid JSON")
	}
	if envelope.Contract != expected || envelope.Version != MaintenanceContractVersion {
		return malformedMaintenanceResponse(expected, fmt.Sprintf("got contract %q version %d", envelope.Contract, envelope.Version))
	}
	if err := decodeMaintenanceJSON(data, target); err != nil {
		return malformedMaintenanceResponse(expected, err.Error())
	}
	return nil
}

func validateChangeSetPayload(data []byte, changeSet ProjectChangeSet) error {
	var raw map[string]any
	if err := decodeMaintenanceJSON(data, &raw); err != nil {
		return malformedMaintenanceResponse("plan", "Change Set is not valid JSON")
	}
	requiredText := []string{"change_set_id", "change_set_hash", "project_id", "expected_revision", "expected_content_hash", "risk"}
	for _, field := range requiredText {
		if stringValue(raw[field]) == "" {
			return malformedMaintenanceResponse("plan", "Change Set is missing "+field)
		}
	}
	if _, ok := raw["approval_required"].(bool); !ok {
		return malformedMaintenanceResponse("plan", "Change Set approval_required must be a boolean")
	}
	if len(changeSet.Operations) == 0 {
		return malformedMaintenanceResponse("plan", "Change Set has no operations")
	}
	if len(changeSet.Validation) == 0 {
		return malformedMaintenanceResponse("plan", "Change Set has no validation steps")
	}
	if err := validateFileEffects(raw["file_effects"], "plan"); err != nil {
		return err
	}
	if err := validatePayloadHash(raw, "change_set_hash", "plan"); err != nil {
		return err
	}
	return nil
}

func validateReceiptPayload(data []byte, receipt ChangeReceipt) error {
	var raw map[string]any
	if err := decodeMaintenanceJSON(data, &raw); err != nil {
		return malformedMaintenanceResponse("apply", "receipt is not valid JSON")
	}
	requiredText := []string{"receipt_id", "receipt_hash", "change_set_id", "change_set_hash", "project_id", "risk", "applied_at"}
	for _, field := range requiredText {
		if stringValue(raw[field]) == "" {
			return malformedMaintenanceResponse("apply", "receipt is missing "+field)
		}
	}
	if _, ok := raw["before"].(map[string]any); !ok {
		return malformedMaintenanceResponse("apply", "receipt before state is missing")
	}
	if _, ok := raw["after"].(map[string]any); !ok {
		return malformedMaintenanceResponse("apply", "receipt after state is missing")
	}
	if len(receipt.Operations) == 0 || len(receipt.Validation) == 0 {
		return malformedMaintenanceResponse("apply", "receipt operations or validation are incomplete")
	}
	if err := validateReceiptLineage(raw["lineage"], receipt); err != nil {
		return err
	}
	if err := validateFileEffects(raw["file_effects"], "apply"); err != nil {
		return err
	}
	if err := validatePayloadHash(raw, "receipt_hash", "apply"); err != nil {
		return err
	}
	return nil
}

func validateReceiptLineage(value any, receipt ChangeReceipt) error {
	raw, ok := value.(map[string]any)
	if !ok {
		return malformedMaintenanceResponse("apply", "receipt lineage must be an object")
	}
	if stringValue(raw["change_set_id"]) == "" || stringValue(raw["change_set_id"]) != receipt.ChangeSetID {
		return malformedMaintenanceResponse("apply", "receipt lineage change_set_id does not match the receipt")
	}
	for _, field := range []string{"sources", "retained_sources", "purged_sources"} {
		if _, ok := raw[field].([]any); !ok {
			return malformedMaintenanceResponse("apply", "receipt lineage is missing "+field+" list")
		}
	}
	expectedSources := []MaintenanceSourceLineage{}
	expectedRetained := []string{}
	expectedPurged := []string{}
	for _, operation := range receipt.Operations {
		switch stringValue(operation["type"]) {
		case "source.replace":
			lineage := MaintenanceSourceLineage{
				Predecessor: stringValue(operation["predecessor_source_id"]),
				Successor:   stringValue(operation["successor_source_id"]),
			}
			if lineage.Predecessor == "" || lineage.Successor == "" {
				return malformedMaintenanceResponse("apply", "replacement receipt operation is missing Source lineage IDs")
			}
			expectedSources = append(expectedSources, lineage)
		case "source.remove":
			sourceID := stringValue(operation["source_id"])
			if sourceID == "" {
				return malformedMaintenanceResponse("apply", "remove receipt operation is missing source_id")
			}
			expectedRetained = append(expectedRetained, sourceID)
		case "source.purge":
			sourceID := stringValue(operation["source_id"])
			if sourceID == "" {
				return malformedMaintenanceResponse("apply", "purge receipt operation is missing source_id")
			}
			expectedPurged = append(expectedPurged, sourceID)
		}
	}
	if !slices.Equal(receipt.Lineage.Sources, expectedSources) ||
		!slices.Equal(receipt.Lineage.RetainedSources, expectedRetained) ||
		!slices.Equal(receipt.Lineage.PurgedSources, expectedPurged) {
		return malformedMaintenanceResponse("apply", "receipt lineage does not match its Source operations")
	}
	return nil
}

func validateFileEffects(value any, action string) error {
	effects, ok := value.(map[string]any)
	if !ok {
		return malformedMaintenanceResponse(action, "file_effects must be an object")
	}
	for _, field := range []string{"create", "write", "delete"} {
		if _, ok := effects[field].([]any); !ok {
			return malformedMaintenanceResponse(action, "file_effects is missing "+field+" list")
		}
	}
	return nil
}

func validatePayloadHash(raw map[string]any, field string, action string) error {
	supplied := stringValue(raw[field])
	unsigned := make(map[string]any, len(raw)-1)
	for key, value := range raw {
		if key != field {
			unsigned[key] = value
		}
	}
	encoded, err := canonicalJSON(unsigned)
	if err != nil {
		return malformedMaintenanceResponse(action, "could not canonicalize "+field)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
	if supplied != digest {
		return malformedMaintenanceResponse(action, field+" does not match the payload")
	}
	return nil
}

func canonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return restorePythonUnicodeSeparators(bytes.TrimSpace(buffer.Bytes())), nil
}

func decodeMaintenanceJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("response contains multiple JSON values")
		}
		return err
	}
	return nil
}

// Python's json.dumps(..., ensure_ascii=False) leaves U+2028 and U+2029 as
// UTF-8. encoding/json escapes them even when HTML escaping is disabled. Only
// odd-length backslash runs end in a real JSON Unicode escape; even runs encode
// a literal backslash and must remain untouched.
func restorePythonUnicodeSeparators(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for index := 0; index < len(data); {
		if data[index] != '\\' {
			result = append(result, data[index])
			index++
			continue
		}
		runEnd := index
		for runEnd < len(data) && data[runEnd] == '\\' {
			runEnd++
		}
		runLength := runEnd - index
		result = append(result, data[index:runEnd]...)
		if runLength%2 == 1 && runEnd+5 <= len(data) {
			escape := string(data[runEnd : runEnd+5])
			if escape == "u2028" || escape == "u2029" {
				result = result[:len(result)-1]
				if escape == "u2028" {
					result = append(result, []byte("\u2028")...)
				} else {
					result = append(result, []byte("\u2029")...)
				}
				index = runEnd + 5
				continue
			}
		}
		index = runEnd
	}
	return result
}

func maintenanceCommandError(action string, out []byte, err error) error {
	var report FailureReport
	if json.Unmarshal(out, &report) == nil && report.Contract == FailureReportContract && report.Version == MaintenanceContractVersion {
		return &MaintenanceError{Action: action, Report: &report, Output: string(out), Err: err}
	}
	return &MaintenanceError{Action: action, Output: string(out), Err: err}
}

func malformedMaintenanceResponse(action string, detail string) error {
	return &MaintenanceError{
		Action: action,
		Report: &FailureReport{
			Contract: FailureReportContract,
			Version:  MaintenanceContractVersion,
			Code:     "malformed_core_response",
			Message:  "Liner Core returned a malformed " + action + " response: " + detail + ".",
			Recovery: []string{"Leave the Project unchanged, verify the installed Core version, then inspect and plan again."},
		},
		Err: errors.New(detail),
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func fallbackMaintenanceText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
