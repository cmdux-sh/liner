package app

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	linerprogress "github.com/cmdux/liner/packages/go-tui/internal/progress"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
	"gopkg.in/yaml.v3"
)

const assemblyDraftRelPath = "working/07-tape-draft.yaml"
const initialAssemblyMarkerRelPath = "working/.liner-initial-assembly"

type assemblyDraft struct {
	Sources []tape.Source `yaml:"sources"`
}

func (m Model) startAssemblyReview() (Model, tea.Cmd) {
	sources, err := readAssemblyDraft(m.currentPath)
	if err != nil {
		m.researchDone = true
		m.err = err.Error()
		return m, nil
	}
	m.sourceItems = source.Stage(sources, true)
	m.applySourceItems(m.sourceItems)
	m.screen = screenAssemblyReview
	m.note = "Loaded proposed sources for review."
	return m, nil
}

func (m Model) startPreparedAssemblyReview() (Model, tea.Cmd) {
	m, _ = m.startAssemblyReview()
	if m.screen != screenAssemblyReview || m.err != "" {
		return m, nil
	}
	return m.startInitialSourceBatch()
}

func (m Model) handleAssemblyReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.sourceBatchRunning || m.assemblyAwaitingSnapshot {
		switch keyMsg.String() {
		case "esc", "q", "ctrl+c":
			if m.assemblyAwaitingSnapshot {
				m.note = "Sources are saved. Waiting for Core to determine the required next step."
				return m, nil
			}
			m.sourceBatchCancelRequested = true
			if m.sourceBatchPhase == sourceBatchPhaseApply {
				m.note = "Cancellation requested. Atomic apply cannot be interrupted; Liner will stop after Core finishes or rolls back."
			} else {
				m.note = "Cancellation requested. Liner will stop at the next safe boundary before atomic apply."
			}
		default:
			if m.assemblyAwaitingSnapshot {
				m.note = "Sources are saved. Waiting for Core to determine the required next step."
			} else {
				m.note = "The Source batch is still running. Press esc to cancel at the next safe boundary."
			}
		}
		return m, nil
	}
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case " ", "space":
		index := m.sourceTable.Cursor()
		if index >= 0 && index < len(m.sourceItems) {
			m.sourceItems[index].Active = !m.sourceItems[index].Active
			m.applySourceItems(m.sourceItems)
			m.sourceMaintenancePlan = nil
			m.sourceBatchPlanValidated = false
			return m.startInitialSourceBatch()
		}
		return m, nil
	case "o":
		m.note = "Opened assembly draft."
		return m, openPath(projectAbsPath(m.currentPath, assemblyDraftRelPath))
	case "d":
		return m.discardAssemblyDraft()
	case "enter":
		return m.acceptAssemblyDraft()
	}
	return m, nil
}

func (m Model) acceptAssemblyDraft() (Model, tea.Cmd) {
	active := source.ActiveSources(m.sourceItems)
	if len(active) == 0 {
		m.err = "Keep at least one source checked before saving the draft."
		return m, nil
	}
	current, err := tape.ReadProject(m.currentPath)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := requireInitialAssemblyBoundary(m.currentPath, current); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.sourceBatchApprovalCaptured = true
	return m.startInitialSourceBatch()
}

func (m Model) finalizeAssemblyAcceptance() (Model, error) {
	current, err := tape.ReadProject(m.currentPath)
	if err != nil {
		return m, fmt.Errorf("Core applied initial Sources, but the Project tape could not be reloaded: %w", err)
	}
	if err := os.Remove(filepath.Join(projectCorpusPath(m.currentPath), initialAssemblyMarkerRelPath)); err != nil {
		return m, fmt.Errorf("Core applied initial Sources, but the one-shot assembly marker could not be retired: %w", err)
	}
	if _, err := linerprogress.MarkPhaseComplete(projectCorpusPath(m.currentPath), linerprogress.PhaseAssembly); err != nil {
		return m, fmt.Errorf("Core applied initial Sources, but progress could not be updated: %w", err)
	}
	if err := os.Remove(projectAbsPath(m.currentPath, assemblyDraftRelPath)); err != nil && !os.IsNotExist(err) {
		return m, fmt.Errorf("Core applied initial Sources, but the assembly draft could not be retired: %w", err)
	}
	m.currentTape = current
	return m, nil
}

func requireInitialAssemblyBoundary(project string, current tape.Tape) error {
	marker := filepath.Join(projectCorpusPath(project), initialAssemblyMarkerRelPath)
	info, err := os.Lstat(marker)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return legacyCoreWriterError("replace Project Sources from an assembly draft")
	}
	if len(current.Sources) != 0 {
		return legacyCoreWriterError("replace accepted Project Sources from an assembly draft")
	}
	return nil
}

func (m Model) discardAssemblyDraft() (Model, tea.Cmd) {
	if err := os.Remove(projectAbsPath(m.currentPath, assemblyDraftRelPath)); err != nil && !os.IsNotExist(err) {
		m.err = "Could not discard assembly draft: " + err.Error()
		return m, nil
	}
	m.sourceItems = source.Stage(m.currentTape.Sources, true)
	m.applySourceItems(m.sourceItems)
	m.screen = screenProject
	m.note = "Discarded assembly draft. Build Corpus again to create a new draft."
	return m, nil
}

func (m Model) viewAssemblyReview() string {
	width := styles.ClampWidth(m.width - 4)
	reviewTable := newVisibleDataTable(
		assemblyColumns(width),
		assemblyRows(m.sourceItems, width, m.sourceTable.Cursor()),
		width,
		sourceReviewTableHeight(len(m.sourceItems)),
		true,
		m.sourceTable.Cursor(),
	)
	counts := fmt.Sprintf(
		"%d proposed  %d checked",
		len(m.sourceItems),
		len(source.ActiveSources(m.sourceItems)),
	)
	sections := []string{
		styles.Title.Render("Review Draft Sources"),
		styles.Subtitle.Render("Assembly proposed the source list for MIXTAPE.md."),
		styles.Section.Render(counts),
	}
	transitioning := m.assemblyApprovalTransitioning()
	if progress := m.sourceBatchProgressView(); progress != "" && !transitioning {
		sections = append(sections, progress)
	}
	sections = append(sections, reviewTable.View())
	if transitioning {
		sections = append(sections, "", assemblyApprovalTransitionView(width))
	} else if m.sourceMaintenancePlan != nil && !m.sourceBatchRunning {
		sections = append(sections, "", assemblyApprovalView(width, *m.sourceMaintenancePlan, len(source.ActiveSources(m.sourceItems))))
	} else {
		sections = append(sections,
			"",
			styles.ReportSection.Render("Selected"),
			m.assemblySelectedDetail(width),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) assemblyApprovalTransitioning() bool {
	return m.screen == screenAssemblyReview && ((m.sourceBatchRunning && m.sourceBatchPhase == sourceBatchPhaseApply) ||
		m.assemblyAwaitingSnapshot ||
		m.synthesisReviewLoading)
}

func assemblyApprovalTransitionView(width int) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Sources accepted"),
		styles.AccentText.Render("Preparing Synthesis approval…"),
		renderLabelValueBlock(width, []labelValueRow{
			{Label: "In progress", Value: "Save the approved Sources and verify the Project state"},
			{Label: "Next", Value: "Review Synthesis before Compile"},
			{Label: "Input", Value: "No additional action needed"},
		}, 0, 0),
	)
}

func assemblyApprovalView(width int, plan core.ProjectChangeSet, checked int) string {
	adds := 0
	unchanged := 0
	for _, operation := range plan.Operations {
		switch operation["type"] {
		case "source.add":
			adds++
		case "source.noop":
			unchanged++
		}
	}
	result := fmt.Sprintf("%d new Sources", adds)
	if unchanged > 0 {
		result = fmt.Sprintf("%d new · %d already present", adds, unchanged)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.ReportSection.Render("Ready to accept Sources"),
		styles.AccentText.Render(fmt.Sprintf("Press Enter to accept %d checked Sources for this Project.", checked)),
		renderLabelValueBlock(width, []labelValueRow{
			{Label: "Result", Value: result},
			{Label: "Change", Value: "Additive only; existing Sources stay unchanged"},
			{Label: "Next", Value: "Refresh the Project, then continue to its required next step"},
		}, 0, 0),
	)
}

func (m Model) assemblySelectedDetail(width int) string {
	index := m.sourceTable.Cursor()
	if index < 0 || index >= len(m.sourceItems) {
		return styles.SoftText.Render("No draft source selected.")
	}
	item := m.sourceItems[index]
	state := "checked"
	if !item.Active {
		state = "unchecked"
	}
	return sourceMetadataTable(width, item, state, "")
}

func assemblyColumns(width int) []table.Column {
	useWidth := 5
	kindWidth := 10
	sectionWidth := 14
	noteWidth := 28
	sourceWidth := width - useWidth - kindWidth - sectionWidth - noteWidth - 8
	if sourceWidth < 20 {
		noteWidth = max(14, noteWidth-(20-sourceWidth))
		sourceWidth = width - useWidth - kindWidth - sectionWidth - noteWidth - 8
	}
	sourceWidth = max(18, sourceWidth)
	return []table.Column{
		{Title: "Use", Width: useWidth},
		{Title: "Kind", Width: kindWidth},
		{Title: "Section", Width: sectionWidth},
		{Title: "Source", Width: sourceWidth},
		{Title: "Note", Width: noteWidth},
	}
}

func assemblyRows(items []source.StagedSource, width int, selected int) []table.Row {
	columns := assemblyColumns(width)
	sourceWidth := columns[3].Width
	noteWidth := columns[4].Width
	rows := make([]table.Row, 0, len(items))
	for index, item := range items {
		rows = append(rows, table.Row{
			sourceUseMarkForRow(item.Active, index == selected),
			truncateMiddle(firstSourceText(item.Source.Kind, item.Type), columns[1].Width),
			truncateMiddle(firstSourceText(item.Source.Section, "unsectioned"), columns[2].Width),
			truncateMiddle(item.Label, sourceWidth),
			truncateMiddle(firstSourceText(item.Source.Note, "no note"), noteWidth),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "", "", "No draft sources found", ""})
	}
	return rows
}

func firstSourceText(value *string, fallback string) string {
	if value != nil && *value != "" {
		return *value
	}
	return fallback
}

func readAssemblyDraft(project string) ([]tape.Source, error) {
	path := projectAbsPath(project, assemblyDraftRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no assembly draft found at %s", assemblyDraftRelPath)
	}
	var draft assemblyDraft
	if err := yaml.Unmarshal(data, &draft); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", assemblyDraftRelPath, err)
	}
	if len(draft.Sources) == 0 {
		return nil, fmt.Errorf("%s contains no sources", assemblyDraftRelPath)
	}
	for i := range draft.Sources {
		if draft.Sources[i].Priority == "" {
			draft.Sources[i].Priority = "required"
		}
	}
	return draft.Sources, nil
}
