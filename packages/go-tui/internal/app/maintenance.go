package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func maintenanceApplyNeedsReconciliation(err error) bool {
	var maintenanceErr *core.MaintenanceError
	if errors.As(err, &maintenanceErr) && maintenanceErr.Report != nil && !maintenanceErr.Report.PartialSuccess {
		if maintenanceErr.Report.Code == "malformed_core_response" {
			return true
		}
		return false
	}
	return true
}

type maintenanceStage int

const (
	maintenanceStageOperation maintenanceStage = iota
	maintenanceStageSource
	maintenanceStageFields
	maintenanceStagePreview
	maintenanceStageReceipt
)

const (
	maintenanceOperationAdd = iota
	maintenanceOperationUpdate
	maintenanceOperationRename
	maintenanceOperationMove
	maintenanceOperationReplace
	maintenanceOperationRemove
	maintenanceOperationPurge
	maintenanceOperationCount
)

type maintenanceField struct {
	key   string
	label string
}

var maintenanceSourceFields = []maintenanceField{
	{key: "type", label: "Type"},
	{key: "locator", label: "URL or path"},
	{key: "note", label: "Note"},
	{key: "section", label: "Section"},
	{key: "priority", label: "Priority"},
	{key: "render", label: "Render"},
	{key: "citation", label: "Citation"},
	{key: "kind", label: "Kind"},
}

var maintenanceRenameFields = []maintenanceField{{key: "name", label: "Display name"}}
var maintenanceMoveFields = []maintenanceField{{key: "destination", label: "Destination"}}
var maintenanceReplacementFields = []maintenanceField{
	{key: "type", label: "Type"},
	{key: "locator", label: "URL or path"},
	{key: "note", label: "Note"},
	{key: "section", label: "Section"},
	{key: "priority", label: "Priority"},
	{key: "render", label: "Render"},
	{key: "citation", label: "Citation"},
	{key: "kind", label: "Kind"},
	{key: "content_hash", label: "Content hash"},
	{key: "provenance_intent", label: "Provenance"},
	{key: "provenance_reason", label: "Private reason"},
}
var maintenancePurgeFields = []maintenanceField{{key: "source_id", label: "Retained ID"}}

func maintenanceOperationUsesActiveSource(operation int) bool {
	return operation == maintenanceOperationUpdate || operation == maintenanceOperationReplace || operation == maintenanceOperationRemove
}

func (m Model) startMaintenance() (Model, tea.Cmd) {
	m.screen = screenMaintenance
	m.maintenanceSnapshot = nil
	m.maintenancePlan = nil
	m.maintenanceReceipt = nil
	m.maintenanceLoading = true
	m.maintenanceSnapshotPending = true
	m.maintenanceStage = maintenanceStageOperation
	m.maintenanceApplying = false
	m.maintenanceReconcile = false
	m.maintenanceReplayPath = ""
	m.maintenancePlanView.SetContent("")
	m.maintenancePlanView.GotoTop()
	m.maintenanceOperation = maintenanceOperationAdd
	m.maintenanceSourceCursor = 0
	m.maintenanceFieldCursor = 0
	m.maintenanceEditing = false
	m.maintenanceFieldValues = nil
	m.maintenanceTouched = nil
	m.maintenanceInput.SetValue("")
	m.maintenanceInput.Blur()
	m.note = "Inspecting the Core-owned Project snapshot."
	m.err = ""
	return m, inspectMaintenanceProject(m.runner, m.currentPath)
}

func inspectMaintenanceProject(runner core.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := runner.InspectMaintenanceProject(path)
		return maintenanceSnapshotMsg{snapshot: snapshot, err: err}
	}
}

func reconcileMaintenanceProject(runner core.Runner, current string, plan core.ProjectChangeSet) tea.Cmd {
	return func() tea.Msg {
		candidates := []string{current}
		if destination := maintenanceMoveDestination(plan); destination != "" && destination != current {
			candidates = append(candidates, destination)
		}
		lastFailure := "Project could not be inspected at the original or reviewed destination root"
		for _, candidate := range candidates {
			snapshot, err := runner.InspectMaintenanceProject(candidate)
			if err != nil {
				lastFailure = err.Error()
				continue
			}
			if snapshot.ProjectID == nil || strings.TrimSpace(*snapshot.ProjectID) != plan.ProjectID {
				lastFailure = fmt.Sprintf("Project identity at %s does not match the reviewed Change Set", candidate)
				continue
			}
			replayPath := strings.TrimSpace(snapshot.Root)
			if replayPath == "" {
				replayPath = candidate
			}
			return maintenanceReconciledMsg{snapshot: snapshot, path: replayPath}
		}
		return maintenanceReconciledMsg{err: fmt.Errorf("receipt reconciliation failed: %s", lastFailure)}
	}
}

func maintenanceMoveDestination(plan core.ProjectChangeSet) string {
	for _, operation := range plan.Operations {
		if operation["type"] == "project.move" {
			if destination, ok := operation["new_root"].(string); ok {
				return strings.TrimSpace(destination)
			}
		}
	}
	return ""
}

func planMaintenanceOperation(runner core.Runner, path string, operation map[string]any) tea.Cmd {
	return func() tea.Msg {
		plan, err := runner.PlanMaintenance(path, operation)
		return maintenancePlanMsg{plan: plan, err: err}
	}
}

func applyMaintenanceOperation(runner core.Runner, path string, plan core.ProjectChangeSet) tea.Cmd {
	return func() tea.Msg {
		receipt, err := runner.ApplyMaintenance(path, plan, plan.ApprovalRequired)
		resultPath := path
		if err == nil {
			if err = validateMaintenanceReceipt(plan, receipt); err == nil {
				resultPath = maintenanceReceiptResultPath(path, receipt)
			}
		}
		return maintenanceAppliedMsg{receipt: receipt, path: resultPath, err: err}
	}
}

func validateMaintenanceReceipt(plan core.ProjectChangeSet, receipt core.ChangeReceipt) error {
	if err := core.ValidateMaintenanceReceipt(plan, receipt); err != nil {
		return fmt.Errorf("Liner Core returned a receipt whose Source operation does not match the reviewed Change Set: %w", err)
	}
	reviewedDestination := maintenanceMoveDestination(plan)
	receiptDestination := maintenanceReceiptMoveDestination(receipt)
	if reviewedDestination != receiptDestination {
		return fmt.Errorf("Liner Core receipt does not match the reviewed move destination")
	}
	return nil
}

func maintenanceReceiptResultPath(current string, receipt core.ChangeReceipt) string {
	if destination := maintenanceReceiptMoveDestination(receipt); destination != "" {
		return destination
	}
	return current
}

func maintenanceReceiptMoveDestination(receipt core.ChangeReceipt) string {
	for _, operation := range receipt.Operations {
		if operation["type"] == "project.move" {
			if destination, ok := operation["new_root"].(string); ok && strings.TrimSpace(destination) != "" {
				return strings.TrimSpace(destination)
			}
		}
	}
	return ""
}

type maintenanceSmokeEnvelope struct {
	Contract  string                          `json:"contract"`
	Version   int                             `json:"version"`
	Snapshot  core.MaintenanceProjectSnapshot `json:"snapshot"`
	ChangeSet core.ProjectChangeSet           `json:"change_set"`
	Receipt   core.ChangeReceipt              `json:"receipt"`
}

// RunInstalledMaintenanceSmoke is the non-interactive release probe for the
// installed Go adapter. It deliberately uses the same Runner methods as the
// Bubble Tea flow, so packaged acceptance can prove that the installed TUI
// reaches Core rather than only checking --version.
func RunInstalledMaintenanceSmoke(project string, operationJSON string) ([]byte, error) {
	var operation map[string]any
	if err := json.Unmarshal([]byte(operationJSON), &operation); err != nil || len(operation) == 0 {
		return nil, fmt.Errorf("maintenance smoke operation must be a non-empty JSON object")
	}
	if operation["type"] != "source.add" {
		return nil, fmt.Errorf("installed maintenance smoke is restricted to additive source.add operations")
	}
	runner, err := core.Resolve()
	if err != nil {
		return nil, err
	}
	snapshot, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		return nil, err
	}
	changeSet, err := runner.PlanMaintenance(project, operation)
	if err != nil {
		return nil, err
	}
	if changeSet.ApprovalRequired {
		return nil, fmt.Errorf("installed maintenance smoke refuses approval-required Change Sets")
	}
	receipt, err := runner.ApplyMaintenance(project, changeSet, false)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(maintenanceSmokeEnvelope{
		Contract:  "liner.tui_maintenance_smoke",
		Version:   core.MaintenanceContractVersion,
		Snapshot:  snapshot,
		ChangeSet: changeSet,
		Receipt:   receipt,
	}, "", "  ")
}

func (m Model) handleMaintenanceKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "ctrl+c":
		if m.maintenanceApplying || m.maintenanceReconcile {
			m.note = "Maintenance apply reconciliation cannot be interrupted. Wait for Core or replay the exact Change Set to recover its receipt."
			return m, nil
		}
		return m, tea.Quit
	case "esc":
		if m.maintenanceLoading {
			return m, nil
		}
		if m.maintenancePlan != nil {
			if m.maintenanceReconcile {
				m.note = "Receipt reconciliation is still required. Press Enter to replay this exact Change Set; Core will not duplicate committed work."
				return m, nil
			}
			m.maintenancePlan = nil
			m.maintenanceStage = maintenanceStageFields
			m.maintenancePlanView.SetContent("")
			m.note = "Discarded the unapplied Core Change Set."
			m.err = ""
			return m, nil
		}
		if m.maintenanceEditing {
			m.maintenanceEditing = false
			m.maintenanceInput.SetValue("")
			m.maintenanceInput.Blur()
			m.note = "Discarded the uncommitted field edit."
			return m, nil
		}
		switch m.maintenanceStage {
		case maintenanceStageSource:
			m.maintenanceStage = maintenanceStageOperation
		case maintenanceStageFields:
			if maintenanceOperationUsesActiveSource(m.maintenanceOperation) {
				m.maintenanceStage = maintenanceStageSource
			} else {
				m.maintenanceStage = maintenanceStageOperation
			}
		case maintenanceStageReceipt:
			m.screen = screenProject
		default:
			m.screen = screenProject
		}
		return m, nil
	case "up", "k":
		if m.maintenancePlan != nil {
			m.maintenancePlanView.ScrollUp(1)
			return m, nil
		}
		if m.maintenanceLoading || m.maintenanceEditing {
			return m, nil
		}
		m.moveMaintenanceCursor(-1)
		return m, nil
	case "down", "j":
		if m.maintenancePlan != nil {
			m.maintenancePlanView.ScrollDown(1)
			return m, nil
		}
		if m.maintenanceLoading || m.maintenanceEditing {
			return m, nil
		}
		m.moveMaintenanceCursor(1)
		return m, nil
	case "pgup":
		if m.maintenancePlan != nil {
			m.maintenancePlanView.HalfPageUp()
		}
		return m, nil
	case "pgdown":
		if m.maintenancePlan != nil {
			m.maintenancePlanView.HalfPageDown()
		}
		return m, nil
	case "p":
		if m.maintenanceLoading || m.maintenanceEditing || m.maintenanceStage != maintenanceStageFields {
			return m, nil
		}
		operation, err := m.guidedMaintenanceOperation()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.maintenanceLoading = true
		m.maintenanceReceipt = nil
		m.note = "Asking Liner Core for a write-free Change Set preview."
		m.err = ""
		return m, planMaintenanceOperation(m.runner, m.currentPath, operation)
	case "enter":
		if m.maintenanceLoading {
			return m, nil
		}
		if m.maintenancePlan != nil {
			if m.maintenanceReconcile && strings.TrimSpace(m.maintenanceReplayPath) == "" {
				m.maintenanceLoading = true
				m.maintenanceSnapshotPending = true
				m.note = "Retrying identity discovery at the original and reviewed destination roots before any receipt replay."
				m.err = ""
				return m, reconcileMaintenanceProject(m.runner, m.currentPath, *m.maintenancePlan)
			}
			applyPath := m.currentPath
			if m.maintenanceReconcile && strings.TrimSpace(m.maintenanceReplayPath) != "" {
				applyPath = m.maintenanceReplayPath
			}
			m.maintenanceLoading = true
			m.maintenanceApplying = true
			m.maintenanceReconcile = false
			m.note = "Applying the exact reviewed Core Change Set."
			m.err = ""
			return m, applyMaintenanceOperation(m.runner, applyPath, *m.maintenancePlan)
		}
		switch m.maintenanceStage {
		case maintenanceStageOperation:
			if maintenanceOperationUsesActiveSource(m.maintenanceOperation) {
				if m.maintenanceSnapshot == nil || len(m.maintenanceSnapshot.Sources) == 0 {
					m.err = "This Project has no active Sources available for the selected operation."
					return m, nil
				}
				m.maintenanceStage = maintenanceStageSource
				m.maintenanceSourceCursor = min(m.maintenanceSourceCursor, len(m.maintenanceSnapshot.Sources)-1)
				m.note = "Choose the Source by readable locator and immutable Source ID."
				return m, nil
			}
			m.beginMaintenanceFields(nil)
			return m, nil
		case maintenanceStageSource:
			source := m.selectedMaintenanceSource()
			if source == nil || source.SourceID == nil || strings.TrimSpace(*source.SourceID) == "" {
				m.err = "The selected Source has no usable immutable Source ID; refresh before editing."
				return m, nil
			}
			m.beginMaintenanceFields(source)
			return m, nil
		case maintenanceStageFields:
			if m.maintenanceEditing {
				m.commitMaintenanceField()
				return m, nil
			}
			m.beginMaintenanceFieldEdit()
			return m, nil
		case maintenanceStageReceipt:
			m.maintenanceReceipt = nil
			m.maintenanceStage = maintenanceStageOperation
			m.maintenanceOperation = maintenanceOperationAdd
			m.maintenanceFieldValues = nil
			m.maintenanceTouched = nil
			m.note = "Choose another guided maintenance operation, or press Esc to return to Project Flow."
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) syncMaintenancePlanView() {
	if m.maintenancePlan == nil {
		m.maintenancePlanView.SetContent("")
		return
	}
	width := m.maintenancePlanView.Width()
	if width <= 0 {
		width = max(20, styles.ClampWidth(m.width-8))
		m.maintenancePlanView.SetWidth(width)
	}
	m.maintenancePlanView.SetContent(maintenancePlanView(width, *m.maintenancePlan, "Enter"))
	m.maintenancePlanView.GotoTop()
}

func (m *Model) moveMaintenanceCursor(delta int) {
	switch m.maintenanceStage {
	case maintenanceStageOperation:
		m.maintenanceOperation = (m.maintenanceOperation + delta + maintenanceOperationCount) % maintenanceOperationCount
	case maintenanceStageSource:
		count := 0
		if m.maintenanceSnapshot != nil {
			count = len(m.maintenanceSnapshot.Sources)
		}
		if count > 0 {
			m.maintenanceSourceCursor = (m.maintenanceSourceCursor + delta + count) % count
		}
	case maintenanceStageFields:
		fields := m.maintenanceFields()
		if len(fields) > 0 {
			m.maintenanceFieldCursor = (m.maintenanceFieldCursor + delta + len(fields)) % len(fields)
		}
	}
}

func (m *Model) beginMaintenanceFields(source *core.MaintenanceSourceSnapshot) {
	m.maintenanceStage = maintenanceStageFields
	m.maintenanceFieldCursor = 0
	m.maintenanceFieldValues = map[string]string{}
	m.maintenanceTouched = map[string]bool{}
	if m.maintenanceOperation == maintenanceOperationRename {
		if m.maintenanceSnapshot != nil {
			m.maintenanceFieldValues["name"] = m.maintenanceSnapshot.Name
		}
		m.note = "Type the Project display name, then press p to review identity effects from Core."
		return
	}
	if m.maintenanceOperation == maintenanceOperationMove {
		m.note = "Type the destination root, then press p to review validation, collision checks, and identity effects from Core."
		return
	}
	if m.maintenanceOperation == maintenanceOperationPurge {
		m.note = "Type the immutable ID of a previously retained Source, then press p to review Core's irreversible delete effects."
		return
	}
	if m.maintenanceOperation == maintenanceOperationRemove {
		m.note = "The selected Source will be detached into the Retention Vault, not deleted. Press p to review Core's exact retained and moved effects."
		return
	}
	if m.maintenanceOperation == maintenanceOperationReplace && source != nil {
		m.maintenanceFieldValues["type"] = source.Type
		m.maintenanceFieldValues["locator"] = source.Locator
		if source.Note != nil {
			m.maintenanceFieldValues["note"] = *source.Note
		}
		if source.Section != nil {
			m.maintenanceFieldValues["section"] = *source.Section
		}
		m.note = "Type successor material and optional provenance intent, then press p to review Core's predecessor-to-successor lineage."
		return
	}
	if source == nil {
		m.maintenanceFieldValues["type"] = "web"
		m.maintenanceFieldValues["priority"] = "required"
		m.note = "Edit typed Source fields, then press p to review the exact Core Change Set."
		return
	}
	m.maintenanceFieldValues["locator"] = source.Locator
	if source.Note != nil {
		m.maintenanceFieldValues["note"] = *source.Note
	}
	if source.Section != nil {
		m.maintenanceFieldValues["section"] = *source.Section
	}
	m.note = "Only touched fields will be sent to Core; Source identity and unshown metadata stay intact."
}

func (m Model) maintenanceFields() []maintenanceField {
	switch m.maintenanceOperation {
	case maintenanceOperationUpdate:
		return maintenanceSourceFields[1:]
	case maintenanceOperationRename:
		return maintenanceRenameFields
	case maintenanceOperationMove:
		return maintenanceMoveFields
	case maintenanceOperationReplace:
		return maintenanceReplacementFields
	case maintenanceOperationRemove:
		return nil
	case maintenanceOperationPurge:
		return maintenancePurgeFields
	default:
		return maintenanceSourceFields
	}
}

func maintenanceFieldIndex(m Model, key string) int {
	for index, field := range m.maintenanceFields() {
		if field.key == key {
			return index
		}
	}
	return 0
}

func (m *Model) beginMaintenanceFieldEdit() {
	fields := m.maintenanceFields()
	if len(fields) == 0 {
		return
	}
	m.maintenanceFieldCursor = max(0, min(m.maintenanceFieldCursor, len(fields)-1))
	field := fields[m.maintenanceFieldCursor]
	m.maintenanceInput.SetValue(m.maintenanceFieldValues[field.key])
	m.maintenanceInput.Focus()
	m.maintenanceEditing = true
	m.note = "Edit " + strings.ToLower(field.label) + ", then press Enter to keep it or Esc to discard the edit."
}

func (m *Model) commitMaintenanceField() {
	fields := m.maintenanceFields()
	if len(fields) == 0 {
		return
	}
	field := fields[max(0, min(m.maintenanceFieldCursor, len(fields)-1))]
	if m.maintenanceFieldValues == nil {
		m.maintenanceFieldValues = map[string]string{}
	}
	if m.maintenanceTouched == nil {
		m.maintenanceTouched = map[string]bool{}
	}
	m.maintenanceFieldValues[field.key] = strings.TrimSpace(m.maintenanceInput.Value())
	m.maintenanceTouched[field.key] = true
	m.maintenanceEditing = false
	m.maintenanceInput.SetValue("")
	m.maintenanceInput.Blur()
	m.note = "Field saved locally. Press p when the typed operation is ready for Core preview."
}

func (m Model) selectedMaintenanceSource() *core.MaintenanceSourceSnapshot {
	if m.maintenanceSnapshot == nil || len(m.maintenanceSnapshot.Sources) == 0 {
		return nil
	}
	index := max(0, min(m.maintenanceSourceCursor, len(m.maintenanceSnapshot.Sources)-1))
	return &m.maintenanceSnapshot.Sources[index]
}

func selectedMaintenanceSourceID(m Model) string {
	source := m.selectedMaintenanceSource()
	if source == nil || source.SourceID == nil {
		return ""
	}
	return strings.TrimSpace(*source.SourceID)
}

func sourceLocatorField(sourceType string, locator string) string {
	if strings.EqualFold(strings.TrimSpace(sourceType), "web") ||
		strings.EqualFold(strings.TrimSpace(sourceType), "youtube") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(locator)), "http://") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(locator)), "https://") {
		return "url"
	}
	return "path"
}

func (m Model) guidedMaintenanceOperation() (map[string]any, error) {
	values := m.maintenanceFieldValues
	switch m.maintenanceOperation {
	case maintenanceOperationAdd:
		sourceType := strings.TrimSpace(values["type"])
		locator := strings.TrimSpace(values["locator"])
		if sourceType == "" || locator == "" {
			return nil, fmt.Errorf("Source type and URL or path are required before preview")
		}
		payload := map[string]any{
			"type":                                  sourceType,
			sourceLocatorField(sourceType, locator): locator,
		}
		for _, key := range []string{"note", "section", "priority", "render", "citation", "kind"} {
			if value := strings.TrimSpace(values[key]); value != "" {
				payload[key] = value
			}
		}
		return core.SourceOperation("source.add", "", payload), nil
	case maintenanceOperationRename:
		name := strings.TrimSpace(values["name"])
		if name == "" {
			return nil, fmt.Errorf("Project display name is required before preview")
		}
		return core.ProjectRenameOperation(name), nil
	case maintenanceOperationMove:
		destination := strings.TrimSpace(values["destination"])
		if destination == "" {
			return nil, fmt.Errorf("Project destination is required before preview")
		}
		return core.ProjectMoveOperation(destination), nil
	case maintenanceOperationReplace:
		source := m.selectedMaintenanceSource()
		if source == nil || source.SourceID == nil || strings.TrimSpace(*source.SourceID) == "" {
			return nil, fmt.Errorf("Choose a predecessor Source with an immutable Source ID before preview")
		}
		sourceType := strings.TrimSpace(values["type"])
		locator := strings.TrimSpace(values["locator"])
		if sourceType == "" || locator == "" {
			return nil, fmt.Errorf("Successor Source type and URL or path are required before preview")
		}
		payload := map[string]any{
			"type":                                  sourceType,
			sourceLocatorField(sourceType, locator): locator,
		}
		for _, key := range []string{"note", "section", "priority", "render", "citation", "kind", "content_hash"} {
			if value := strings.TrimSpace(values[key]); value != "" {
				payload[key] = value
			}
		}
		operation := core.SourceOperation("source.replace", *source.SourceID, payload)
		if intent := strings.TrimSpace(values["provenance_intent"]); intent != "" {
			operation["provenance_intent"] = intent
		}
		if reason := strings.TrimSpace(values["provenance_reason"]); reason != "" {
			operation["provenance_reason"] = reason
		}
		return operation, nil
	case maintenanceOperationRemove:
		sourceID := selectedMaintenanceSourceID(m)
		if sourceID == "" {
			return nil, fmt.Errorf("Choose a Source with an immutable Source ID before preview")
		}
		return core.SourceOperation("source.remove", sourceID, nil), nil
	case maintenanceOperationPurge:
		sourceID := strings.TrimSpace(values["source_id"])
		if sourceID == "" {
			return nil, fmt.Errorf("A retained immutable Source ID is required before preview")
		}
		return core.SourceOperation("source.purge", sourceID, nil), nil
	}
	source := m.selectedMaintenanceSource()
	if source == nil || source.SourceID == nil || strings.TrimSpace(*source.SourceID) == "" {
		return nil, fmt.Errorf("Choose a Source with an immutable Source ID before preview")
	}
	changes := map[string]any{}
	for _, field := range m.maintenanceFields() {
		if !m.maintenanceTouched[field.key] {
			continue
		}
		key := field.key
		value := strings.TrimSpace(values[field.key])
		if key == "locator" {
			if value == "" {
				return nil, fmt.Errorf("A Source locator cannot be empty")
			}
			key = sourceLocatorField(source.Type, value)
		}
		if key == "priority" && value == "" {
			return nil, fmt.Errorf("Priority must be required or optional")
		}
		if value == "" {
			changes[key] = nil
		} else {
			changes[key] = value
		}
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("Edit at least one Source field before preview")
	}
	return core.SourceOperation("source.update", *source.SourceID, changes), nil
}

func (m Model) maintenanceOperationView(width int) string {
	rows := []table.Row{
		{"Add Source", "Create one additive Source through Core"},
		{"Update Source", "Change metadata or locator; preserve immutable identity"},
		{"Rename Project", "Change display name; preserve immutable Project ID"},
		{"Move Project", "Validate and atomically activate a new root for the same Project ID"},
		{"Replace Source", "Create a reviewed successor with predecessor lineage"},
		{"Remove Source", "Detach into the Retention Vault; retain content and provenance"},
		{"Purge Retained", "Separately approve and irreversibly delete retained Source material"},
	}
	return newVisibleDataTable(
		[]table.Column{{Title: "Operation", Width: 16}, {Title: "Outcome", Width: max(30, width-20)}},
		rows, width, len(rows)+1, true, m.maintenanceOperation,
	).View()
}

func (m Model) maintenanceSourcePickerView(width int) string {
	if m.maintenanceSnapshot == nil || len(m.maintenanceSnapshot.Sources) == 0 {
		return styles.SoftText.Render("No active Sources are available.")
	}
	identityWidth := max(14, width-53)
	rows := make([]table.Row, 0, len(m.maintenanceSnapshot.Sources))
	for _, source := range m.maintenanceSnapshot.Sources {
		id := "missing immutable ID"
		if source.SourceID != nil && strings.TrimSpace(*source.SourceID) != "" {
			id = strings.TrimSpace(*source.SourceID)
		}
		rows = append(rows, table.Row{
			truncateMiddle(source.Locator, identityWidth),
			truncateMiddle(source.Type, 9),
			truncateMiddle(id, 36),
		})
	}
	height := max(3, min(7, m.height-14))
	t := newVisibleDataTable(
		[]table.Column{{Title: "Readable identity", Width: identityWidth}, {Title: "Type", Width: 9}, {Title: "Immutable Source ID", Width: 36}},
		rows, width, height, true, m.maintenanceSourceCursor,
	)
	selected := m.selectedMaintenanceSource()
	detail := ""
	if selected != nil {
		detail = renderLabelValueBlock(width, []labelValueRow{
			{Label: "Selection", Value: fmt.Sprintf("Source %d of %d", max(0, min(m.maintenanceSourceCursor, len(rows)-1))+1, len(rows))},
			{Label: "Source", Value: selected.Locator},
			{Label: "Immutable ID", Value: selectedMaintenanceSourceID(m)},
		}, 0, 0)
	}
	return lipgloss.JoinVertical(lipgloss.Left, t.View(), detail)
}

func (m Model) maintenanceFieldsView(width int) string {
	fields := m.maintenanceFields()
	if len(fields) == 0 && m.maintenanceOperation == maintenanceOperationRemove {
		source := m.selectedMaintenanceSource()
		if source == nil {
			return styles.SoftText.Render("No active Source is selected.")
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			renderLabelValueBlock(width, []labelValueRow{
				{Label: "Source", Value: source.Locator},
				{Label: "Immutable ID", Value: selectedMaintenanceSourceID(m)},
				{Label: "Disposition", Value: "Retention Vault — detach and preserve content/provenance"},
			}, 0, 0),
			"",
			styles.SoftText.Render("p preview Core's exact retention effects · Esc choose another Source"),
		)
	}
	rows := make([]table.Row, 0, len(fields))
	valueWidth := max(24, width-19)
	for _, field := range fields {
		value := m.maintenanceFieldValues[field.key]
		if m.maintenanceOperation == maintenanceOperationUpdate && !m.maintenanceTouched[field.key] && value == "" {
			value = "(unchanged)"
		}
		rows = append(rows, table.Row{field.label, truncateMiddle(fallbackText(value, "(empty)"), valueWidth)})
	}
	height := max(4, min(len(rows)+1, m.height-13))
	t := newVisibleDataTable(
		[]table.Column{{Title: "Field", Width: 14}, {Title: "Typed value", Width: valueWidth}},
		rows, width, height, !m.maintenanceEditing, m.maintenanceFieldCursor,
	)
	parts := []string{t.View()}
	if m.maintenanceEditing {
		parts = append(parts, "", styles.ReportSection.Render("Editing"), m.maintenanceInput.View())
	} else {
		parts = append(parts, "", styles.SoftText.Render("↑/↓ choose field · Enter edit · p preview exact Core Change Set · Esc back"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) viewMaintenance() string {
	width := styles.ClampWidth(m.width - 4)
	sections := []string{
		styles.Title.Render("Maintain Project through Liner Core"),
		styles.Subtitle.Render("Guided Project and Source maintenance keeps immutable identity and Core authority visible at every step."),
	}
	if snapshot := m.maintenanceSnapshot; snapshot != nil {
		sections = append(sections, "", renderLabelValueBlock(width, []labelValueRow{
			{Label: "Project", Value: snapshot.Name},
			{Label: "Revision", Value: snapshot.Revision},
		}, 0, 0))
	}
	if m.maintenanceLoading {
		sections = append(sections, "", renderWaitStatusBlock(width, "Core request in progress", "The Project remains unchanged until a validated apply completes.", "waiting for Liner Core"))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}
	switch m.maintenanceStage {
	case maintenanceStageSource:
		sections = append(sections, "", styles.ReportSection.Render("Choose Source"), m.maintenanceSourcePickerView(width))
	case maintenanceStageFields:
		sections = append(sections, "", styles.ReportSection.Render("Typed maintenance fields"), m.maintenanceFieldsView(width))
	case maintenanceStagePreview:
		if m.maintenancePlan != nil {
			sections = append(sections, "", m.maintenancePlanView.View())
		}
	case maintenanceStageReceipt:
		if m.maintenanceReceipt != nil {
			sections = append(sections, "", styles.ReportSection.Render("Core receipt"), styles.SuccessText.Render(strings.Join(core.ReceiptSummaryLines(*m.maintenanceReceipt), "\n")))
		}
		if m.maintenanceSnapshotPending {
			sections = append(sections, "", styles.ReportSection.Render("Refreshing Core Snapshot"), styles.SoftText.Render("Waiting for the post-receipt Core inspection before showing completion evidence."))
		} else if snapshot := m.maintenanceSnapshot; snapshot != nil {
			rows := []labelValueRow{
				{Label: "Project", Value: snapshot.Name},
				{Label: "Root", Value: snapshot.Root},
				{Label: "Revision", Value: snapshot.Revision},
				{Label: "Sources", Value: fmt.Sprintf("%d active", len(snapshot.Sources))},
				{Label: "Lifecycle", Value: maintenanceLifecycleState(snapshot.Lifecycle)},
				{Label: "Corpus", Value: snapshot.Lifecycle.Corpus.State},
				{Label: "Operating Layer", Value: snapshot.Lifecycle.OperatingLayer.State},
			}
			if refresh := snapshot.Lifecycle.Refresh; refresh != nil {
				rows = append(rows,
					labelValueRow{Label: "Refresh", Value: refresh.State},
					labelValueRow{Label: "Synthesis review", Value: refresh.Synthesis.State},
					labelValueRow{Label: "Corpus refresh", Value: refresh.Corpus.State},
					labelValueRow{Label: "Operating refresh", Value: refresh.OperatingLayer.State},
				)
				if len(refresh.Remaining) > 0 {
					rows = append(rows, labelValueRow{Label: "Remaining", Value: strings.Join(refresh.Remaining, ", ")})
				}
			}
			sections = append(sections, "", styles.ReportSection.Render("Refreshed Core Snapshot"), renderLabelValueBlock(width, rows, 0, 0))
		} else {
			sections = append(sections, "", styles.ReportSection.Render("Core Snapshot refresh unavailable"), styles.SoftText.Render("The durable receipt remains valid. Refresh Project Flow before starting another maintenance operation."))
		}
		sections = append(sections, "", styles.SoftText.Render("Enter maintain another Project element · Esc return to Project Flow"))
	default:
		sections = append(sections, "", styles.ReportSection.Render("Choose operation"), m.maintenanceOperationView(width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func maintenanceLifecycleState(lifecycle core.MaintenanceProjectLifecycle) string {
	state := "current"
	if lifecycle.Stale {
		state = "stale"
	}
	if strings.TrimSpace(lifecycle.Milestone) != "" {
		return state + " · milestone " + lifecycle.Milestone
	}
	return state
}
