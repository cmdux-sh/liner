package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"gopkg.in/yaml.v3"

	"github.com/cmdux/liner/packages/go-tui/internal/agent"
	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const (
	improvementRunRelPath       = ".liner-runs/improvement"
	improvementWorkspaceRelPath = ".liner-runs/improvement/workspace"
	improvementDeltaRelPath     = ".liner-runs/improvement/delta.yaml"
	improvementSnapshotFile     = ".liner-current-snapshot.json"
	improvementDeltaContract    = "liner.improvement_delta"
	improvementDecisionRelPath  = ".liner-runs/improvement-decision.json"
	improvementDecisionContract = "liner.improvement_decision"
)

type improvementDelta struct {
	Contract     string        `yaml:"contract"`
	Version      int           `yaml:"version"`
	Summary      string        `yaml:"summary"`
	Additions    []tape.Source `yaml:"additions"`
	Removals     []any         `yaml:"removals"`
	Replacements []any         `yaml:"replacements"`
}

type improvementDecision struct {
	Contract    string `json:"contract"`
	Version     int    `json:"version"`
	AuditSHA256 string `json:"audit_sha256"`
	Disposition string `json:"disposition"`
}

const (
	operatingFitAuditRelPath = "working/05-operating-fit-audit.md"
	qualityChecksRelPath     = "working/04-quality-checks.md"
)

const (
	improvementOptionRun int = iota
	improvementOptionSkip
)

var improvementOptions = []choiceOption{
	{
		Label:  "Improve now",
		Detail: "Run a focused improvement pass. Liner will use the quality notes to search for missing source roles, evaluate the new sources, and refresh the corpus before the Operating Layer.",
	},
	{
		Label:  "Skip",
		Detail: "Skip for now. Liner keeps the improvement notes, continues to the Operating Layer, and will offer this pass again if you run Compile for this project.",
	},
}

type operatingFitAudit struct {
	RelPath string
	Body    string
	Summary string
}

func improvementOptionAt(cursor int) int {
	if cursor == improvementOptionSkip {
		return improvementOptionSkip
	}
	return improvementOptionRun
}

func (m Model) startImprovementReview() Model {
	m.screen = screenImprovementReview
	m.improvementCursor = improvementOptionRun
	m.improvementDelta = nil
	m.improvementBaseline = nil
	m.improvementPlan = nil
	m.improvementReceipt = nil
	m.improvementLoading = false
	m.improvementApplying = false
	m.improvementReconcile = false
	m.note = ""
	m.err = ""
	return m
}

func (m Model) handleImprovementReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.improvementLoading {
		return m, nil
	}
	if m.improvementPlan != nil {
		switch keyMsg.String() {
		case "ctrl+c":
			if m.improvementApplying || m.improvementReconcile {
				m.note = "Improvement receipt reconciliation cannot be interrupted. Replay the exact Change Set to recover its receipt."
				return m, nil
			}
			return m, tea.Quit
		case "enter":
			if !improvementPlanHasAdditions(*m.improvementPlan) {
				m.improvementPlan = nil
				m.improvementDelta = nil
				m.improvementBaseline = nil
				m.maintenancePlanView.SetContent("")
				_ = os.RemoveAll(projectAbsPath(m.currentPath, improvementRunRelPath))
				m.screen = screenProject
				m.beginProjectSnapshotLoad(m.currentPath)
				m.note = "Every proposed Source was an exact duplicate. No canonical change was needed; returning to the refreshed Project Flow."
				m.err = ""
				return m, tea.Batch(loadProjectSnapshot(m.runner, m.currentPath), loadProjectStatus(m.runner, m.currentPath))
			}
			m.improvementLoading = true
			m.improvementApplying = true
			m.note = "Applying the exact reviewed improvement Change Set through Liner Core."
			m.err = ""
			return m, applyImprovementDelta(m.runner, m.currentPath, *m.improvementPlan)
		case "esc", "d":
			if m.improvementReconcile {
				m.note = "Receipt reconciliation is still required. Press Enter to replay this exact Change Set; Core will not duplicate committed work."
				return m, nil
			}
			m.improvementPlan = nil
			m.improvementDelta = nil
			m.improvementBaseline = nil
			m.maintenancePlanView.SetContent("")
			_ = os.RemoveAll(projectAbsPath(m.currentPath, improvementRunRelPath))
			m.note = "Discarded the staged improvement delta. Canonical Project artifacts are unchanged."
			m.err = ""
			return m, nil
		case "up", "k":
			m.maintenancePlanView.ScrollUp(1)
		case "down", "j":
			m.maintenancePlanView.ScrollDown(1)
		case "pgup":
			m.maintenancePlanView.HalfPageUp()
		case "pgdown":
			m.maintenancePlanView.HalfPageDown()
		}
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.screen = screenCompile
		m.err = ""
		return m, nil
	case "enter":
		return m.applyImprovementOption()
	case "shift+tab", "up", "left":
		return m.moveImprovementOption(-1), nil
	case "tab", "down", "right":
		return m.moveImprovementOption(1), nil
	case "p":
		if audit, ok := operatingFitImprovementAudit(m.currentPath); ok {
			return m.openPreview(audit.RelPath)
		}
		m.err = "No improvement notes are available to preview."
		return m, nil
	}
	switch keyMsg.Key().Code {
	case tea.KeyEsc:
		m.screen = screenCompile
		m.err = ""
		return m, nil
	case tea.KeyLeft, tea.KeyUp:
		return m.moveImprovementOption(-1), nil
	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		return m.moveImprovementOption(1), nil
	case tea.KeyEnter:
		return m.applyImprovementOption()
	}
	return m, nil
}

func (m Model) moveImprovementOption(delta int) Model {
	if len(improvementOptions) == 0 {
		return m
	}
	m.improvementCursor = (selectedChoiceIndex(improvementOptions, m.improvementCursor) + delta + len(improvementOptions)) % len(improvementOptions)
	m.err = ""
	return m
}

func (m Model) applyImprovementOption() (Model, tea.Cmd) {
	switch improvementOptionAt(m.improvementCursor) {
	case improvementOptionSkip:
		if err := recordImprovementDecision(m.currentPath, "skipped"); err != nil {
			m.err = "Could not save the Improve Corpus decision: " + err.Error()
			m.note = "The Operating Layer remains unchanged. Retry after Liner can save the decision."
			return m, nil
		}
		next, cmd := m.startLinerDraftReview()
		next.note = "Skipped improvement pass for now. Notes remain in " + operatingFitAuditRelPath + "."
		return next, cmd
	default:
		return m.startImprovementPass()
	}
}

func (m Model) startImprovementPass() (Model, tea.Cmd) {
	snapshot := m.currentProjectSnapshot()
	if snapshot == nil {
		m.err = "Improve Corpus requires a trustworthy current Core Project Snapshot. Refresh Project Flow, then retry."
		return m, nil
	}
	if !snapshot.Capabilities["plan"] || !snapshot.Capabilities["apply"] {
		m.err = "Liner Core reports this Project as read-only. Improve Corpus cannot stage an apply."
		return m, nil
	}
	m.stopMethodology("")
	m.screen = screenResearch
	m.researchDone = false
	m.methodologyFailed = false
	m.methodologyCancelled = false
	m.methodologyLastErr = ""
	m.methodologyFailureKind = ""
	m.methodologyPrimaryFailure = ""
	m.methodologyRecovery = ""
	m.methodologyDiagnostics = nil
	m.methodologyRawLog = nil
	m.methodologyLogPath = ""
	m.methodologyPhaseID = ""
	m.methodologyEventCount = 0
	m.methodologyLastEventFrame = m.fxFrame
	m.ensureBoardItems()
	m.note = ""
	m.err = ""
	m.researchStep = 0
	m.methodologyPhaseIndex = -1
	m.researchLines = []string{
		"Starting Improvement Pass...",
		"Using " + operatingFitAuditRelPath + " to target missing source roles.",
		"Queued one staged Source delta; initial methodology artifacts remain unchanged.",
	}
	baseline := *snapshot
	m.improvementBaseline = &baseline
	if err := prepareImprovementWorkspace(m.currentPath, baseline); err != nil {
		m.researchDone = true
		m.err = "Could not prepare the isolated improvement workspace: " + err.Error()
		m.researchLines = append(m.researchLines, "Improvement staging stopped before the agent runner started.")
		m.syncMethodologyLog(true)
		return m, nil
	}
	m.syncMethodologyLog(true)
	return m.startImprovementAgent(false)
}

func (m Model) startImprovementAgent(resume bool) (Model, tea.Cmd) {
	if strings.TrimSpace(m.currentPath) == "" {
		m.researchDone = true
		m.err = "Cannot improve the corpus without a project path."
		return m, nil
	}
	script := resolveHeadlessRunnerScript()
	if script == "" {
		m.researchDone = true
		m.err = "Could not find the Corpus Builder runner. Run npm --prefix packages/tui run build, or set LINER_HEADLESS_RUNNER."
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	run, err := agent.Runner{ScriptPath: script}.Start(ctx, agent.RunArgs{
		Project: improvementWorkspacePath(m.currentPath), PhaseID: "improvement", Agent: "auto", Resume: resume,
	})
	if err != nil {
		cancel()
		m.researchDone = true
		m.err = err.Error()
		return m, nil
	}
	m.methodologyCancel = cancel
	m.methodologyEvents = run.Events
	m.methodologyDone = run.Done
	m.methodologyPhaseIndex = -1
	m.methodologyPhaseID = "improvement"
	m.methodologyEventCount = 0
	m.methodologyLastEventFrame = m.fxFrame
	m.methodologyRunID++
	m.researchDone = false
	verb := "Starting"
	if resume {
		verb = "Resuming"
	}
	m.researchLines = append(m.researchLines, verb+" Improve Corpus.")
	m.syncMethodologyLog(true)
	return m, waitMethodologyEvent(run.Events, run.Done, m.methodologyRunID)
}

func readImprovementDelta(project string) (improvementDelta, error) {
	data, err := os.ReadFile(filepath.Join(improvementWorkspacePath(project), filepath.FromSlash(improvementDeltaRelPath)))
	if err != nil {
		return improvementDelta{}, fmt.Errorf("read staged improvement delta: %w", err)
	}
	var delta improvementDelta
	if err := yaml.Unmarshal(data, &delta); err != nil {
		return improvementDelta{}, fmt.Errorf("parse staged improvement delta: %w", err)
	}
	if delta.Contract != improvementDeltaContract || delta.Version != 1 {
		return improvementDelta{}, fmt.Errorf("staged improvement delta has an unsupported contract or version")
	}
	if strings.TrimSpace(delta.Summary) == "" || len(delta.Additions) == 0 {
		return improvementDelta{}, fmt.Errorf("staged improvement delta needs a summary and at least one focused addition")
	}
	if len(delta.Removals) > 0 || len(delta.Replacements) > 0 {
		return improvementDelta{}, fmt.Errorf("removals and replacements require separate explicit maintenance; Improve Corpus cannot infer destructive intent")
	}
	for index, item := range delta.Additions {
		if err := validateImprovementSource(item); err != nil {
			return improvementDelta{}, fmt.Errorf("addition %d: %w", index+1, err)
		}
	}
	return delta, nil
}

func improvementWorkspacePath(project string) string {
	return projectAbsPath(project, improvementWorkspaceRelPath)
}

func prepareImprovementWorkspace(project string, snapshot core.MaintenanceProjectSnapshot) error {
	workspace := improvementWorkspacePath(project)
	if err := os.RemoveAll(workspace); err != nil {
		return err
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return err
	}
	stagedSnapshot := snapshot
	stagedSnapshot.Root = "."
	snapshotData, err := json.MarshalIndent(stagedSnapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workspace, improvementSnapshotFile), append(snapshotData, '\n'), 0o644); err != nil {
		return err
	}
	deltaPath := filepath.Join(workspace, filepath.FromSlash(improvementDeltaRelPath))
	if err := os.MkdirAll(filepath.Dir(deltaPath), 0o755); err != nil {
		return err
	}
	stagedDelta := "contract: liner.improvement_delta\nversion: 1\nsummary: TODO\nadditions: []\nremovals: []\nreplacements: []\n"
	if err := os.WriteFile(deltaPath, []byte(stagedDelta), 0o644); err != nil {
		return err
	}
	corpus := projectCorpusPath(project)
	for _, rel := range []string{"tape.yaml", "synthesis.md", "working"} {
		sourcePath := filepath.Join(corpus, filepath.FromSlash(rel))
		if _, err := os.Lstat(sourcePath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := copyImprovementArtifact(sourcePath, filepath.Join(workspace, filepath.FromSlash(rel))); err != nil {
			return err
		}
	}
	return nil
}

func copyImprovementArtifact(sourcePath string, destinationPath string) error {
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink in improvement staging: %s", sourcePath)
	}
	if info.IsDir() {
		if err := os.MkdirAll(destinationPath, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyImprovementArtifact(filepath.Join(sourcePath, entry.Name()), filepath.Join(destinationPath, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular improvement artifact: %s", sourcePath)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destinationPath, data, info.Mode().Perm())
}

func validateImprovementSource(item tape.Source) error {
	if strings.TrimSpace(item.Type) == "" || strings.TrimSpace(item.Priority) == "" || item.Kind == nil || strings.TrimSpace(*item.Kind) == "" {
		return fmt.Errorf("type, priority, and kind are required")
	}
	switch item.Type {
	case "web", "youtube":
		if strings.TrimSpace(item.URL) == "" {
			return fmt.Errorf("url is required for %s", item.Type)
		}
	case "local_file":
		if item.Path == nil || strings.TrimSpace(*item.Path) == "" {
			return fmt.Errorf("path is required for local_file")
		}
	case "skill":
		if strings.TrimSpace(item.URL) == "" && (item.Path == nil || strings.TrimSpace(*item.Path) == "") {
			return fmt.Errorf("path or url is required for skill")
		}
	default:
		return fmt.Errorf("unsupported Source type %q", item.Type)
	}
	return nil
}

func planImprovementDelta(runner core.Runner, project string, delta improvementDelta, current tape.Tape, baseline core.MaintenanceProjectSnapshot) (core.ProjectChangeSet, error) {
	if err := validateImprovementBaseline(current, baseline); err != nil {
		return core.ProjectChangeSet{}, err
	}
	payloads := make([]map[string]any, 0, len(delta.Additions))
	for _, item := range delta.Additions {
		payloads = append(payloads, sourceMaintenancePayload(item))
	}
	plan, err := runner.PlanMaintenance(project, core.SourceBatchOperation(payloads))
	if err != nil {
		return core.ProjectChangeSet{}, err
	}
	projectID := ""
	if baseline.ProjectID != nil {
		projectID = strings.TrimSpace(*baseline.ProjectID)
	}
	if projectID == "" || plan.ProjectID != projectID || plan.ExpectedRevision != baseline.Revision || plan.ExpectedContentHash != baseline.ContentHash {
		return core.ProjectChangeSet{}, fmt.Errorf("Core returned a plan that does not match the fixed improvement Snapshot")
	}
	if err := validateInitialSourceBatchPlan(plan, delta.Additions, current.Sources, nil); err != nil {
		return core.ProjectChangeSet{}, fmt.Errorf("Core returned an unsafe improvement plan: %w", err)
	}
	for _, operation := range plan.Operations {
		if kind, _ := operation["type"].(string); kind != "source.add" && kind != "source.noop" {
			return core.ProjectChangeSet{}, fmt.Errorf("Core returned unrelated improvement operation %q", kind)
		}
	}
	return plan, nil
}

func validateImprovementBaseline(current tape.Tape, baseline core.MaintenanceProjectSnapshot) error {
	if baseline.ProjectID == nil || strings.TrimSpace(*baseline.ProjectID) == "" || strings.TrimSpace(baseline.Revision) == "" || strings.TrimSpace(baseline.ContentHash) == "" {
		return fmt.Errorf("Improve Corpus baseline is missing Core identity or revision evidence")
	}
	currentByID := make(map[string]tape.Source, len(current.Sources))
	for _, item := range current.Sources {
		if item.ID != nil && strings.TrimSpace(*item.ID) != "" {
			currentByID[strings.TrimSpace(*item.ID)] = item
		}
	}
	for _, accepted := range baseline.Sources {
		if accepted.SourceID == nil || strings.TrimSpace(*accepted.SourceID) == "" {
			return fmt.Errorf("Improve Corpus baseline contains a Source without immutable identity")
		}
		item, ok := currentByID[strings.TrimSpace(*accepted.SourceID)]
		if !ok || sourceLocator(item) != strings.TrimSpace(accepted.Locator) {
			return fmt.Errorf("accepted Source %s no longer matches the fixed Core Snapshot", strings.TrimSpace(*accepted.SourceID))
		}
	}
	return nil
}

func sourceLocator(item tape.Source) string {
	if strings.TrimSpace(item.URL) != "" {
		return strings.TrimSpace(item.URL)
	}
	if item.Path != nil {
		return strings.TrimSpace(*item.Path)
	}
	return ""
}

func planImprovementDeltaFromBaselineCommand(runner core.Runner, project string, baseline core.MaintenanceProjectSnapshot) tea.Cmd {
	return func() tea.Msg {
		delta, err := readImprovementDelta(project)
		if err != nil {
			return improvementDeltaPlannedMsg{err: err}
		}
		current, err := tape.ReadProject(project)
		if err != nil {
			return improvementDeltaPlannedMsg{delta: delta, err: err}
		}
		plan, err := planImprovementDelta(runner, project, delta, current, baseline)
		return improvementDeltaPlannedMsg{delta: delta, plan: plan, err: err}
	}
}

func applyImprovementDelta(runner core.Runner, project string, plan core.ProjectChangeSet) tea.Cmd {
	return func() tea.Msg {
		receipt, err := runner.ApplyMaintenance(project, plan, plan.ApprovalRequired)
		if err != nil {
			return improvementAppliedMsg{err: err}
		}
		snapshot, snapshotErr := runner.InspectMaintenanceProject(project)
		return improvementAppliedMsg{receipt: receipt, snapshot: snapshot, snapshotErr: snapshotErr}
	}
}

func improvementPlanHasAdditions(plan core.ProjectChangeSet) bool {
	for _, operation := range plan.Operations {
		if operation["type"] == "source.add" {
			return true
		}
	}
	return false
}

func inspectImprovementSnapshot(runner core.Runner, project string) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := runner.InspectMaintenanceProject(project)
		return improvementSnapshotMsg{snapshot: snapshot, err: err}
	}
}

func (m *Model) syncImprovementPlanView() {
	if m.improvementPlan == nil {
		m.maintenancePlanView.SetContent("")
		return
	}
	width := m.maintenancePlanView.Width()
	if width <= 0 {
		width = max(20, styles.ClampWidth(m.width-8))
		m.maintenancePlanView.SetWidth(width)
	}
	m.maintenancePlanView.SetContent(improvementApprovalView(width, *m.improvementPlan, m.improvementDelta))
	m.maintenancePlanView.SetHeight(min(max(1, m.maintenancePlanView.Height()), max(1, m.maintenancePlanView.TotalLineCount())))
	m.maintenancePlanView.GotoTop()
}

func improvementApprovalView(width int, plan core.ProjectChangeSet, delta *improvementDelta) string {
	adds, duplicates := 0, 0
	for _, operation := range plan.Operations {
		switch operation["type"] {
		case "source.add":
			adds++
		case "source.noop":
			duplicates++
		}
	}
	rows := []labelValueRow{
		{Label: "Accept", Value: fmt.Sprintf("%d suggested Sources", adds)},
		{Label: "Existing", Value: "Unchanged"},
		{Label: "Duplicates", Value: fmt.Sprintf("%d already present", duplicates)},
		{Label: "Next", Value: "Review synthesis, then Compile"},
	}
	parts := []string{
		styles.ReportSection.Render("Ready to add Sources"),
		styles.AccentText.Render(fmt.Sprintf("Press Enter to accept %d suggested Sources, or d to discard them.", adds)),
		"",
		renderLabelValueBlock(width, rows, 0, 0),
	}
	if delta != nil && len(delta.Additions) > 0 {
		parts = append(parts, "", styles.ReportSection.Render("Suggestions"))
		for index, source := range delta.Additions {
			kind := "source"
			if source.Kind != nil && strings.TrimSpace(*source.Kind) != "" {
				kind = strings.TrimSpace(*source.Kind)
			}
			label := fmt.Sprintf("%d · %s", index+1, kind)
			parts = append(parts, renderLabelValueBlock(width, []labelValueRow{{Label: label, Value: improvementSourceSummary(source)}}, 0, 0))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func improvementSourceSummary(source tape.Source) string {
	locator := strings.TrimSpace(source.URL)
	if parsed, err := url.Parse(locator); err == nil && parsed.Host != "" {
		locator = parsed.Host + strings.TrimSuffix(parsed.EscapedPath(), "/")
	}
	if locator == "" && source.Path != nil {
		locator = strings.TrimSpace(*source.Path)
	}
	if locator == "" {
		locator = "Local source"
	}
	purpose := "Focused evidence for the identified gap."
	if source.Note != nil {
		if role := improvementNoteSection(*source.Note, "Role:", "Value:"); role != "" {
			purpose = role
		}
	}
	return truncateMiddle(locator, 72) + " — " + truncateMiddle(purpose, 120)
}

func improvementNoteSection(note string, start string, end string) string {
	value := strings.TrimSpace(note)
	startIndex := strings.Index(value, start)
	if startIndex < 0 {
		return ""
	}
	value = strings.TrimSpace(value[startIndex+len(start):])
	if endIndex := strings.Index(value, end); endIndex >= 0 {
		value = strings.TrimSpace(value[:endIndex])
	}
	return strings.Join(strings.Fields(value), " ")
}

func (m Model) viewImprovementReview() string {
	width := styles.ClampWidth(m.width - 4)
	if m.improvementPlan != nil {
		summary := "Focused Source delta"
		if m.improvementDelta != nil && strings.TrimSpace(m.improvementDelta.Summary) != "" {
			summary = m.improvementDelta.Summary
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			styles.Title.Render("Review improvement delta"),
			styles.Subtitle.Render(m.currentTape.Title), "",
			renderLabelValueBlock(width, []labelValueRow{{Label: "Gap", Value: summary}, {Label: "Outcome", Value: improvementOutcomeSummary(*m.improvementPlan)}}, 0, 0), "",
			m.maintenancePlanView.View(),
		)
	}
	body := "Quality checks found a source-role gap before this corpus becomes operating guidance. You can let Liner run a focused second pass now, or skip for now and continue to the Operating Layer."
	parts := []string{
		styles.Title.Render("Improve Corpus"),
		styles.Subtitle.Render(m.currentTape.Title),
		"",
		styles.PrimaryText.Render(strings.Join(wrapWords(body, width), "\n")),
	}
	if audit, ok := operatingFitImprovementAudit(m.currentPath); ok {
		parts = append(parts, "", renderImprovementAuditSummary(width, m.currentPath, audit))
	}
	parts = append(parts,
		"",
		renderChoiceSelector(improvementOptions, m.improvementCursor),
		renderChoiceDetail(width, improvementOptions, m.improvementCursor),
	)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func improvementOutcomeSummary(plan core.ProjectChangeSet) string {
	adds, duplicates := 0, 0
	for _, operation := range plan.Operations {
		switch operation["type"] {
		case "source.add":
			adds++
		case "source.noop":
			duplicates++
		}
	}
	return fmt.Sprintf("%d additions · %d exact duplicates · one atomic apply", adds, duplicates)
}

func renderImprovementAuditSummary(width int, project string, audit operatingFitAudit) string {
	rows := []labelValueRow{
		{Label: "Notes", Value: displayProjectPath(project, audit.RelPath)},
	}
	if audit.Summary != "" {
		rows = append(rows, labelValueRow{Label: "Gap", Value: audit.Summary})
	}
	return renderLabelValueBlock(width, rows, 0, 0)
}

func (m Model) improvementNextAction() string {
	if m.improvementLoading {
		return "Wait for Liner Core to finish the current improvement operation."
	}
	if m.improvementPlan != nil {
		if m.improvementReconcile {
			return "Replay the exact improvement Change Set to recover its receipt."
		}
		return "Accept the summarized Source suggestions, or discard them without changing the Project."
	}
	if improvementOptionAt(m.improvementCursor) == improvementOptionSkip {
		return "Skip for now and continue to the Operating Layer."
	}
	return "Run the improvement pass before creating the Operating Layer."
}

func operatingFitImprovementRecommended(project string) bool {
	audit, ok := operatingFitImprovementAudit(project)
	if !ok {
		return false
	}
	decision, err := readImprovementDecision(project)
	if err != nil || decision.AuditSHA256 != improvementAuditSHA256(audit.Body) {
		return true
	}
	return decision.Disposition != "skipped" && decision.Disposition != "applied"
}

func recordImprovementDecision(project string, disposition string) error {
	audit, ok := operatingFitImprovementAudit(project)
	if !ok {
		return fmt.Errorf("the improvement recommendation is no longer available")
	}
	decision := improvementDecision{
		Contract: improvementDecisionContract, Version: 1,
		AuditSHA256: improvementAuditSHA256(audit.Body), Disposition: disposition,
	}
	data, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		return err
	}
	path := projectAbsPath(project, improvementDecisionRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".improvement-decision-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func readImprovementDecision(project string) (improvementDecision, error) {
	data, err := os.ReadFile(projectAbsPath(project, improvementDecisionRelPath))
	if err != nil {
		return improvementDecision{}, err
	}
	var decision improvementDecision
	if err := json.Unmarshal(data, &decision); err != nil {
		return improvementDecision{}, err
	}
	if decision.Contract != improvementDecisionContract || decision.Version != 1 {
		return improvementDecision{}, fmt.Errorf("unsupported improvement decision")
	}
	return decision, nil
}

func improvementAuditSHA256(body string) string {
	sum := sha256.Sum256([]byte(body))
	return fmt.Sprintf("sha256:%x", sum)
}

func resetSkippedImprovementDecision(project string) {
	decision, err := readImprovementDecision(project)
	if err == nil && decision.Disposition == "skipped" {
		_ = os.Remove(projectAbsPath(project, improvementDecisionRelPath))
	}
}

func operatingFitImprovementAudit(project string) (operatingFitAudit, bool) {
	for _, rel := range []string{operatingFitAuditRelPath, qualityChecksRelPath} {
		body, err := os.ReadFile(projectAbsPath(project, rel))
		if err != nil {
			continue
		}
		text := string(body)
		if !improvementRecommendationMarker(text) {
			continue
		}
		return operatingFitAudit{
			RelPath: filepath.ToSlash(rel),
			Body:    text,
			Summary: improvementAuditSummary(text),
		}, true
	}
	return operatingFitAudit{}, false
}

func improvementRecommendationMarker(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "status: improvement_recommended") ||
		strings.Contains(lower, "status: improvement recommended") ||
		strings.Contains(lower, "improvement recommended") ||
		strings.Contains(lower, "recommended improvement pass")
}

func improvementAuditSummary(body string) string {
	for _, prefix := range []string{"gap:", "why_it_matters:", "why it matters:", "recommendation:", "recommended pass:"} {
		if value := firstMarkdownValue(body, prefix); value != "" {
			return value
		}
	}
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(strings.TrimLeft(line, "-* "))
		if strings.Contains(strings.ToLower(clean), "source-role") || strings.Contains(strings.ToLower(clean), "source role") {
			return clean
		}
	}
	return ""
}

func firstMarkdownValue(body string, prefix string) string {
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(strings.TrimLeft(line, "-* "))
		if !strings.HasPrefix(strings.ToLower(clean), prefix) {
			continue
		}
		value := strings.TrimSpace(clean[len(prefix):])
		value = strings.Trim(value, "`")
		if value != "" {
			return value
		}
	}
	return ""
}
