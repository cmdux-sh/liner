package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	draftDetails := assemblyDraftDetails(sources)
	sources = preserveExistingAssemblySources(sources, m.currentTape.Sources)
	m.sourceItems = source.Stage(sources, true)
	fetchStatuses := readEvaluationFetchStatuses(m.currentPath)
	for index := range m.sourceItems {
		m.sourceItems[index].DraftNote = draftDetails[index].Note
		m.sourceItems[index].DraftSection = draftDetails[index].Section
		m.sourceItems[index].DraftKind = draftDetails[index].Kind
		m.sourceItems[index].FetchStatus = evaluationFetchStatusFor(fetchStatuses, m.sourceItems[index].Source)
	}
	m.applySourceItems(m.sourceItems)
	m.screen = screenAssemblyReview
	m.note = "Loaded proposed sources for review."
	return m, nil
}

// assemblyDraftDetail is what the agents wrote about a draft entry.
type assemblyDraftDetail struct {
	Note    string
	Section string
	Kind    string
}

// assemblyDraftDetails keeps each draft entry's note, section and kind by
// position. A Source the Project already holds is replaced by its saved copy
// so the draft cannot change it, but what the agents wrote about it still
// belongs on the review screen.
func assemblyDraftDetails(draft []tape.Source) []assemblyDraftDetail {
	details := make([]assemblyDraftDetail, len(draft))
	for index, item := range draft {
		details[index] = assemblyDraftDetail{
			Note:    strings.TrimSpace(firstSourceText(item.Note, "")),
			Section: strings.TrimSpace(firstSourceText(item.Section, "")),
			Kind:    strings.TrimSpace(firstSourceText(item.Kind, "")),
		}
	}
	return details
}

// readEvaluationFetchStatuses maps each evaluated URL to the evaluation's
// structured fetch_status field.
func readEvaluationFetchStatuses(project string) map[string]string {
	data, err := os.ReadFile(projectAbsPath(project, "working/03-evaluation.yaml"))
	if err != nil {
		return nil
	}
	var artifact methodologyEvaluationArtifact
	if err := yaml.Unmarshal(data, &artifact); err != nil {
		return nil
	}
	statuses := map[string]string{}
	for _, candidate := range artifact.Candidates {
		status := strings.ToLower(strings.TrimSpace(candidate.FetchStatus))
		if status == "" {
			continue
		}
		for _, key := range issueKeysForURL(candidate.URL) {
			statuses[key] = status
		}
	}
	return statuses
}

func evaluationFetchStatusFor(statuses map[string]string, item tape.Source) string {
	for _, key := range issueKeysForURL(item.URL) {
		if status, ok := statuses[key]; ok {
			return status
		}
	}
	return ""
}

// fetchStatusMarker is the short table marker for the evaluation's
// fetch_status; fetchStatusText is its plain-language form.
func fetchStatusMarker(status string) string {
	switch status {
	case "readable":
		return "full"
	case "partial":
		return "partial"
	case "metadata_only":
		return "metadata"
	case "unavailable":
		return "none"
	case "":
		return "–"
	default:
		return status
	}
}

func fetchStatusText(status string) string {
	switch status {
	case "readable":
		return "Full text"
	case "partial":
		return "Partial text"
	case "metadata_only":
		return "Metadata only"
	case "unavailable":
		return "Not readable"
	case "":
		return "Not recorded"
	default:
		return status
	}
}

// assemblyReviewNote is the note the review shows: the agents' draft note
// when there is one, otherwise the saved note.
func assemblyReviewNote(item source.StagedSource) string {
	if note := strings.TrimSpace(item.DraftNote); note != "" {
		return note
	}
	return strings.TrimSpace(firstSourceText(item.Source.Note, ""))
}

// assemblyReviewSection and assemblyReviewKind show the saved value when
// there is one, otherwise the draft's.
func assemblyReviewSection(item source.StagedSource) string {
	return fallbackText(firstSourceText(item.Source.Section, ""), item.DraftSection)
}

func assemblyReviewKind(item source.StagedSource) string {
	return fallbackText(firstSourceText(item.Source.Kind, ""), item.DraftKind)
}

// assemblyReviewDraftOnly names the section and kind the review shows from the
// draft because the saved Source has none. Accepting keeps the saved Source.
func assemblyReviewDraftOnly(item source.StagedSource) []string {
	fields := []string{}
	if strings.TrimSpace(firstSourceText(item.Source.Kind, "")) == "" && item.DraftKind != "" {
		fields = append(fields, "kind")
	}
	if strings.TrimSpace(firstSourceText(item.Source.Section, "")) == "" && item.DraftSection != "" {
		fields = append(fields, "section")
	}
	return fields
}

func preserveExistingAssemblySources(reviewed []tape.Source, existing []tape.Source) []tape.Source {
	preserved := append([]tape.Source(nil), reviewed...)
	matched := make(map[int]bool, len(existing))
	for _, current := range existing {
		match := assemblySourceMatchIndex(current, reviewed, matched)
		if match < 0 {
			continue
		}
		preserved[match] = current
		matched[match] = true
	}
	return preserved
}

func assemblySourceMatchIndex(current tape.Source, reviewed []tape.Source, matched map[int]bool) int {
	currentID := sourceImmutableID(current)
	if currentID != "" {
		for index, candidate := range reviewed {
			if !matched[index] && sourceImmutableID(candidate) == currentID {
				return index
			}
		}
	}
	matches := []int{}
	for index, candidate := range reviewed {
		if matched[index] || sourceImmutableID(candidate) != "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(candidate.Type), strings.TrimSpace(current.Type)) &&
			strings.TrimSpace(sourceLocator(candidate)) != "" &&
			maintenanceLocatorsMatch(sourceLocator(candidate), sourceLocator(current)) {
			matches = append(matches, index)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return -1
}

func sourceImmutableID(item tape.Source) string {
	if item.ID == nil {
		return ""
	}
	return strings.TrimSpace(*item.ID)
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
	case "pgup":
		return m.scrollAssemblyDetail(-3), nil
	case "pgdown":
		return m.scrollAssemblyDetail(3), nil
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
	if err := requireInitialAssemblyBoundary(m.currentPath, current, active); err != nil {
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

func requireInitialAssemblyBoundary(project string, current tape.Tape, reviewed []tape.Source) error {
	marker := filepath.Join(projectCorpusPath(project), initialAssemblyMarkerRelPath)
	info, err := os.Lstat(marker)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return legacyCoreWriterError("replace Project Sources from an assembly draft")
	}
	reviewedIdentities := make(map[string]int, len(reviewed))
	for _, item := range reviewed {
		reviewedIdentities[sourceIdentityKey(item)]++
	}
	for _, item := range current.Sources {
		identity := sourceIdentityKey(item)
		if reviewedIdentities[identity] == 0 {
			return legacyCoreWriterError("replace accepted Project Sources from an assembly draft")
		}
		reviewedIdentities[identity]--
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
	layout := m.assemblyReviewLayout(width)
	return m.viewAssemblyReviewWithLayout(width, layout)
}

// assemblyLayout is how much of the review fits: the table's visible rows and,
// when the selected detail is longer than the rows left, the window onto it.
type assemblyLayout struct {
	tableHeight int
	detailRows  int // 0 shows the whole detail
	detailTotal int
	detailStart int
}

// assemblyReviewLayout keeps the selected Source's notes and the accept panel
// on screen: the table gives up rows first, then a long detail scrolls
// (pgup/pgdn) instead of letting the frame cut it off.
func (m Model) assemblyReviewLayout(width int) assemblyLayout {
	layout := assemblyLayout{tableHeight: sourceReviewTableHeight(len(m.sourceItems))}
	budget := m.bodyLineBudget()
	if budget <= 0 {
		return layout
	}
	overflow := lipgloss.Height(m.viewAssemblyReviewWithLayout(width, layout)) - budget
	if overflow <= 0 {
		return layout
	}
	shrink := min(overflow, max(0, layout.tableHeight-3))
	layout.tableHeight -= shrink
	overflow -= shrink
	if overflow <= 0 {
		return layout
	}
	total := lipgloss.Height(m.assemblySelectedDetail(width))
	// One row goes to the scroll position line.
	rows := max(2, total-overflow-1)
	if rows >= total {
		return layout
	}
	layout.detailRows = rows
	layout.detailTotal = total
	layout.detailStart = min(max(0, m.assemblyDetailOffset), total-rows)
	return layout
}

// scrollAssemblyDetail moves the selected detail's window by delta rows.
func (m Model) scrollAssemblyDetail(delta int) Model {
	layout := m.assemblyReviewLayout(styles.ClampWidth(m.width - 4))
	if layout.detailRows == 0 {
		m.assemblyDetailOffset = 0
		return m
	}
	m.assemblyDetailOffset = min(max(0, layout.detailStart+delta), layout.detailTotal-layout.detailRows)
	return m
}

func (m Model) assemblySelectedDetailView(width int, layout assemblyLayout) string {
	detail := m.assemblySelectedDetail(width)
	if layout.detailRows == 0 {
		return detail
	}
	lines := strings.Split(detail, "\n")
	end := min(len(lines), layout.detailStart+layout.detailRows)
	position := "more below"
	switch {
	case layout.detailStart > 0 && end < len(lines):
		position = "more above and below"
	case layout.detailStart > 0:
		position = "end; more above"
	}
	status := fmt.Sprintf("Detail lines %d–%d of %d · %s · pgup/pgdn scroll", layout.detailStart+1, end, len(lines), position)
	return lipgloss.JoinVertical(lipgloss.Left, strings.Join(lines[layout.detailStart:end], "\n"), styles.SoftText.Render(truncateMiddle(status, width)))
}

func (m Model) viewAssemblyReviewWithLayout(width int, layout assemblyLayout) string {
	tableHeight := layout.tableHeight
	reviewTable := newVisibleDataTable(
		assemblyColumns(width),
		assemblyRows(m.sourceItems, width, m.sourceTable.Cursor()),
		width,
		tableHeight,
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
		// Keep the selected Source's notes on screen while the person decides
		// whether to accept the planned Sources.
		sections = append(sections,
			"",
			styles.ReportSection.Render("Selected"),
			m.assemblySelectedDetailView(width, layout),
			"",
			assemblyApprovalView(width, *m.sourceMaintenancePlan, len(source.ActiveSources(m.sourceItems))),
		)
	} else {
		sections = append(sections,
			"",
			styles.ReportSection.Render("Selected"),
			m.assemblySelectedDetailView(width, layout),
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
	use := []string{state}
	for _, value := range []string{
		assemblyReviewKind(item),
		fallbackText(assemblyReviewSection(item), "unsectioned"),
		fallbackText(item.Source.Priority, "required"),
	} {
		if strings.TrimSpace(value) != "" {
			use = append(use, value)
		}
	}
	rows := []labelValueRow{
		{Label: "Source", Value: truncateMiddle(item.Label, max(24, width-12))},
		{Label: "Use", Value: strings.Join(use, " · ")},
		{Label: "Read", Value: fetchStatusText(item.FetchStatus)},
		{Label: "Note", Value: fallbackText(assemblyReviewNote(item), "No note")},
	}
	saved := strings.TrimSpace(firstSourceText(item.Source.Note, ""))
	if saved != "" && saved != assemblyReviewNote(item) {
		rows = append(rows, labelValueRow{Label: "Saved note", Value: saved})
	}
	if draftOnly := assemblyReviewDraftOnly(item); len(draftOnly) > 0 {
		verb := "comes"
		if len(draftOnly) > 1 {
			verb = "come"
		}
		rows = append(rows, labelValueRow{Label: "Saved", Value: "No " + strings.Join(draftOnly, " or ") + "; the " + strings.Join(draftOnly, " and ") + " shown " + verb + " from the draft. Accepting keeps the saved Source as it is."})
	}
	return renderLabelValueBlock(width, rows, 0, 0)
}

func assemblyColumns(width int) []table.Column {
	useWidth := 5
	kindWidth := 10
	sectionWidth := 14
	readWidth := 8
	noteWidth := 24
	gaps := 10
	sourceWidth := width - useWidth - kindWidth - sectionWidth - readWidth - noteWidth - gaps
	if sourceWidth < 20 {
		noteWidth = max(14, noteWidth-(20-sourceWidth))
		sourceWidth = width - useWidth - kindWidth - sectionWidth - readWidth - noteWidth - gaps
	}
	sourceWidth = max(18, sourceWidth)
	return []table.Column{
		{Title: "Use", Width: useWidth},
		{Title: "Kind", Width: kindWidth},
		{Title: "Section", Width: sectionWidth},
		{Title: "Source", Width: sourceWidth},
		{Title: "Read", Width: readWidth},
		{Title: "Note", Width: noteWidth},
	}
}

func assemblyRows(items []source.StagedSource, width int, selected int) []table.Row {
	columns := assemblyColumns(width)
	sourceWidth := columns[3].Width
	noteWidth := columns[5].Width
	rows := make([]table.Row, 0, len(items))
	for index, item := range items {
		rows = append(rows, table.Row{
			sourceUseMarkForRow(item.Active, index == selected),
			truncateMiddle(fallbackText(assemblyReviewKind(item), item.Type), columns[1].Width),
			truncateMiddle(fallbackText(assemblyReviewSection(item), "unsectioned"), columns[2].Width),
			truncateMiddle(item.Label, sourceWidth),
			fetchStatusMarker(item.FetchStatus),
			truncateMiddle(fallbackText(assemblyReviewNote(item), "no note"), noteWidth),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "", "", "No draft sources found", "", ""})
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
