package app

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	linerprogress "github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type projectPipelinePhase struct {
	ID       linerprogress.PhaseID
	Label    string
	Artifact string
}

type projectPipelineRow struct {
	Phase    string
	State    string
	Evidence string
	Current  bool
}

var projectPipelinePhases = []projectPipelinePhase{
	{ID: linerprogress.PhaseFraming, Label: "Framing", Artifact: "working/01-jtbd-and-knowledge-map.md"},
	{ID: linerprogress.PhaseGate0, Label: "Confirm framing"},
	{ID: linerprogress.PhaseCandidates, Label: "Candidate discovery", Artifact: "working/02-candidate-longlist.md"},
	{ID: linerprogress.PhaseGate1, Label: "Confirm candidates"},
	{ID: linerprogress.PhaseEvaluation, Label: "Evaluation", Artifact: "working/03-evaluation.yaml"},
	{ID: linerprogress.PhaseQuality, Label: "Quality checks", Artifact: "working/04-quality-checks.md"},
	{ID: linerprogress.PhaseGate2, Label: "Confirm evaluation"},
	{ID: linerprogress.PhaseSynthesis, Label: "Synthesis", Artifact: "synthesis.md"},
	{ID: linerprogress.PhaseAssembly, Label: "Assembly", Artifact: assemblyDraftRelPath},
	{ID: linerprogress.PhaseCompile, Label: "Compile", Artifact: "MIXTAPE.md"},
}

func (m Model) projectPipelineView(width int, sourceEntryPrimary bool) string {
	status := m.currentProjectStatus()
	if status == nil {
		if sourceEntryPrimary || !projectPipelineHasSignal(m.currentPath) {
			return ""
		}
	}
	rows := projectPipelineRows(m.currentPath, m.currentTape, status)
	rows = projectPipelineWindow(rows)
	if len(rows) == 0 {
		return ""
	}
	phaseWidth := min(24, max(18, width/3))
	stateWidth := 10
	evidenceWidth := max(22, width-phaseWidth-stateWidth-8)
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, table.Row{
			truncateMiddle(row.Phase, phaseWidth),
			truncateMiddle(row.State, stateWidth),
			truncateMiddle(row.Evidence, evidenceWidth),
		})
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Corpus build progress"),
		newDataTable(
			[]table.Column{
				{Title: "Phase", Width: phaseWidth},
				{Title: "Status", Width: stateWidth},
				{Title: "Evidence", Width: evidenceWidth},
			},
			tableRows,
			width,
			len(tableRows)+1,
			false,
		).View(),
	)
}

func (m Model) currentProjectStatus() *core.ProjectStatus {
	if m.status == nil || m.statusPath != m.currentPath {
		return nil
	}
	return m.status
}

func (m *Model) clearProjectStatus() {
	m.statusPath = ""
	m.status = nil
	m.statusErr = ""
}

func projectPipelineRows(project string, t tape.Tape, status *core.ProjectStatus) []projectPipelineRow {
	if status != nil && len(status.Phases) > 0 && !projectCompileArtifactsNeedAttention(project, t.Sources, len(t.Sources)) {
		return projectPipelineRowsFromStatus(*status)
	}
	step := projectPipelineStep(project, t)
	corpusPath := projectCorpusPath(project)
	gates := linerprogress.ReadGateState(corpusPath)
	rows := make([]projectPipelineRow, 0, len(projectPipelinePhases))
	for index, phase := range projectPipelinePhases {
		state := "queued"
		if index < step {
			state = "done"
		} else if index == step {
			state = "current"
		}
		if step >= len(projectPipelinePhases) {
			state = "done"
		}
		rows = append(rows, projectPipelineRow{
			Phase:    phase.Label,
			State:    state,
			Evidence: projectPipelineEvidence(project, phase, gates, t),
			Current:  index == step && step < len(projectPipelinePhases),
		})
	}
	return rows
}

func projectPipelineRowsFromStatus(status core.ProjectStatus) []projectPipelineRow {
	rows := make([]projectPipelineRow, 0, len(status.Phases))
	for _, phase := range status.Phases {
		rows = append(rows, projectPipelineRow{
			Phase:    phase.Label,
			State:    projectPipelineStateFromStatus(phase.Status),
			Evidence: projectPipelineEvidenceFromStatus(phase),
			Current:  phase.Status == "in_progress",
		})
	}
	return rows
}

func projectPipelineStateFromStatus(status string) string {
	switch status {
	case "complete":
		return "done"
	case "in_progress":
		return "current"
	case "not_started":
		return "queued"
	default:
		if strings.TrimSpace(status) == "" {
			return "queued"
		}
		return status
	}
}

func projectPipelineEvidenceFromStatus(phase core.StatusPhase) string {
	if phase.Gate != nil {
		return gateEvidence(phase.Gate.Accepted)
	}
	if phase.Artifact != nil {
		if phase.Artifact.Exists {
			if phase.Artifact.HasRealContent {
				return phase.Artifact.Path
			}
			return phase.Artifact.Path + " placeholder"
		}
		return "waiting for " + phase.Artifact.Path
	}
	if phase.Runs.Count > 0 {
		return fmt.Sprintf("%d run(s)", phase.Runs.Count)
	}
	return ""
}

func projectPipelineWindow(rows []projectPipelineRow) []projectPipelineRow {
	if len(rows) <= 4 {
		return rows
	}
	current := -1
	for index, row := range rows {
		if row.Current {
			current = index
			break
		}
	}
	if current < 0 {
		current = len(rows) - 1
	}
	start := max(0, current-1)
	end := min(len(rows), current+2)
	for end-start < 3 && start > 0 {
		start--
	}
	for end-start < 3 && end < len(rows) {
		end++
	}
	return rows[start:end]
}

func projectPipelineStep(project string, t tape.Tape) int {
	corpusPath := projectCorpusPath(project)
	if projectCompileArtifactsNeedAttention(project, t.Sources, len(t.Sources)) {
		return projectPipelinePhaseIndex(linerprogress.PhaseCompile)
	}
	if projectFileExists(project, ".liner-progress.json") {
		return linerprogress.Read(corpusPath).Step
	}
	step := 0
	gates := linerprogress.ReadGateState(corpusPath)
	for _, phase := range projectPipelinePhases {
		if !projectPipelinePhaseComplete(project, phase, gates, t) {
			break
		}
		step++
	}
	return step
}

func projectPipelinePhaseIndex(id linerprogress.PhaseID) int {
	for index, phase := range projectPipelinePhases {
		if phase.ID == id {
			return index
		}
	}
	return 0
}

func projectPipelineHasSignal(project string) bool {
	if strings.TrimSpace(project) == "" {
		return false
	}
	if projectFileExists(project, ".liner-progress.json") || projectFileExists(project, ".liner-gates.json") {
		return true
	}
	for _, phase := range projectPipelinePhases {
		switch phase.ID {
		case linerprogress.PhaseFraming, linerprogress.PhaseCandidates, linerprogress.PhaseEvaluation, linerprogress.PhaseQuality, linerprogress.PhaseAssembly:
		default:
			continue
		}
		if phase.Artifact != "" && projectFileExists(project, phase.Artifact) {
			return true
		}
	}
	return false
}

func projectPipelinePhaseComplete(project string, phase projectPipelinePhase, gates linerprogress.GateState, t tape.Tape) bool {
	switch phase.ID {
	case linerprogress.PhaseGate0:
		return gates.Gate0Accepted
	case linerprogress.PhaseGate1:
		return gates.Gate1Accepted
	case linerprogress.PhaseGate2:
		return gates.Gate2Accepted
	case linerprogress.PhaseFraming:
		return t.JTBD != nil && strings.TrimSpace(*t.JTBD) != "" && projectArtifactHasRealContent(projectAbsPath(project, phase.Artifact))
	case linerprogress.PhaseSynthesis:
		return projectSynthesisHasRealContent(projectAbsPath(project, phase.Artifact))
	case linerprogress.PhaseAssembly:
		return len(t.Sources) > 0
	case linerprogress.PhaseCompile:
		return projectFileExists(project, phase.Artifact) && !projectCompileArtifactsNeedAttention(project, t.Sources, len(t.Sources))
	default:
		return phase.Artifact != "" && projectArtifactHasRealContent(projectAbsPath(project, phase.Artifact))
	}
}

func projectPipelineEvidence(project string, phase projectPipelinePhase, gates linerprogress.GateState, t tape.Tape) string {
	switch phase.ID {
	case linerprogress.PhaseGate0:
		return gateEvidence(gates.Gate0Accepted)
	case linerprogress.PhaseGate1:
		return gateEvidence(gates.Gate1Accepted)
	case linerprogress.PhaseGate2:
		return gateEvidence(gates.Gate2Accepted)
	case linerprogress.PhaseAssembly:
		if len(t.Sources) > 0 {
			return fmt.Sprintf("%d source(s) accepted", len(t.Sources))
		}
		if projectFileExists(project, phase.Artifact) {
			return phase.Artifact + " ready"
		}
		return phase.Artifact
	case linerprogress.PhaseCompile:
		if projectFileExists(project, phase.Artifact) {
			if projectCompileArtifactsNeedAttention(project, t.Sources, len(t.Sources)) {
				return phase.Artifact + " needs attention"
			}
			return phase.Artifact + " ready"
		}
		return phase.Artifact
	default:
		if phase.Artifact == "" {
			return ""
		}
		if projectFileExists(project, phase.Artifact) {
			return phase.Artifact
		}
		return "waiting for " + phase.Artifact
	}
}

func gateEvidence(accepted bool) string {
	if accepted {
		return "accepted"
	}
	return "pending"
}

func projectArtifactHasRealContent(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := string(data)
	if strings.TrimSpace(text) == "" {
		return false
	}
	for _, marker := range []string{
		"Quantity over precision",
		"candidates: []",
		"Run each test deliberately",
		"Replace this placeholder",
		"TODO —",
		"example bullets below are placeholders",
	} {
		if strings.Contains(text, marker) {
			return false
		}
	}
	return true
}

func projectSynthesisHasRealContent(path string) bool {
	return projectArtifactHasRealContent(path)
}
