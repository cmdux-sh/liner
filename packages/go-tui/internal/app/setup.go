package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func (m *Model) currentCreateFieldValid() bool {
	switch m.createStep {
	case 0:
		if strings.TrimSpace(m.createDraft.Title) == "" {
			m.err = "Name is required."
			return false
		}
	case 1:
		if strings.TrimSpace(m.createDraft.JTBD) == "" {
			m.err = "Job to Be Done is required."
			return false
		}
	case 2:
		if strings.TrimSpace(m.createDraft.Curator) == "" {
			m.err = "Curator is required."
			return false
		}
	}
	return true
}

func (m Model) canEditClarifyText() bool {
	return !m.clarifyLoading && len(m.clarifyQuestions) > 0
}

func (m *Model) startCreate() {
	m.screen = screenCreate
	m.createStep = 0
	m.createDraft = createDraft{AddSources: true}
	m.createRunning = false
	m.createError = ""
	m.createOpenRetryPath = ""
	m.createInput.SetValue("")
	m.createInput.Placeholder = "..."
	m.createInput.Focus()
	m.createArea.SetValue("")
	m.createArea.Placeholder = "..."
	m.createArea.Focus()
	m.note = ""
}

func (m *Model) startClarification() {
	m.screen = screenClarify
	m.clarifyStep = 0
	m.clarifyQuestions = nil
	m.clarifyAnswers = nil
	m.clarifyLoading = true
	m.clarifyError = ""
	m.clarifyArea.SetValue("")
	m.clarifyArea.Placeholder = "your answer (or leave blank to skip)"
	m.note = ""
}

func (m Model) startClarificationFlow() (Model, tea.Cmd) {
	m.startClarification()
	if draft, ok, err := loadClarificationDraft(m.currentPath, tapeJTBD(m.currentTape)); err == nil && ok {
		m.applyClarificationDraft(draft)
		return m, nil
	} else if err != nil {
		m.clarifyError = err.Error()
	}
	return m, generateClarifyingQuestions(m.currentPath, tapeJTBD(m.currentTape))
}

func (m *Model) commitCreateInput() {
	switch m.createStep {
	case 0:
		value := strings.TrimSpace(m.createInput.Value())
		if value != "" {
			m.createDraft.Title = value
			m.createDraft.Slug = slug(value)
		}
	case 1:
		m.createDraft.JTBD = strings.TrimSpace(m.createArea.Value())
	case 2:
		m.createDraft.Curator = strings.TrimSpace(m.createInput.Value())
	case 3:
		return
	}
}

func (m *Model) commitClarifyInput() {
	if m.clarifyStep >= 0 && m.clarifyStep < len(m.clarifyAnswers) {
		m.clarifyAnswers[m.clarifyStep] = strings.TrimSpace(m.clarifyArea.Value())
	}
}

func (m *Model) setClarifyQuestions(questions []string) {
	m.clarifyQuestions = make([]clarifyQuestion, 0, len(questions))
	for _, question := range questions {
		question = strings.TrimSpace(question)
		if question != "" {
			m.clarifyQuestions = append(m.clarifyQuestions, clarifyQuestion{question: question})
		}
	}
	m.clarifyAnswers = make([]string, len(m.clarifyQuestions))
	m.clarifyStep = 0
	if len(m.clarifyQuestions) > 0 {
		m.setClarifyField(0)
	}
}

func (m *Model) setCreateField(step int) {
	m.createStep = step
	switch step {
	case 0:
		m.createInput.SetValue(m.createDraft.Title)
		m.createInput.Placeholder = "..."
	case 1:
		m.createArea.SetValue(m.createDraft.JTBD)
		m.createArea.Placeholder = "..."
	case 2:
		m.createInput.SetValue(m.createDraft.Curator)
		m.createInput.Placeholder = "..."
	case 3:
		m.createInput.SetValue("")
		m.createInput.Placeholder = ""
	}
}

func (m *Model) setClarifyField(step int) {
	if len(m.clarifyQuestions) == 0 {
		m.clarifyStep = 0
		m.clarifyArea.SetValue("")
		return
	}
	if step < 0 {
		step = 0
	}
	if step >= len(m.clarifyQuestions) {
		step = len(m.clarifyQuestions) - 1
	}
	m.clarifyStep = step
	m.clarifyArea.SetValue(m.clarifyAnswers[step])
	m.clarifyArea.Placeholder = "your answer (or leave blank to skip)"
	m.clarifyArea.Focus()
}

func (m Model) submitCreate() (Model, tea.Cmd) {
	if m.createRunning {
		return m, nil
	}
	if m.createOpenRetryPath != "" {
		m.createRunning = true
		m.createError = ""
		m.err = ""
		m.note = "Retrying the created Project open."
		return m, readCreatedProject(m.createOpenRetryPath)
	}
	m.commitCreateInput()
	m.createError = ""
	if strings.TrimSpace(m.createDraft.Title) == "" {
		m.err = "Title is required."
		m.setCreateField(0)
		return m, nil
	}
	if strings.TrimSpace(m.createDraft.JTBD) == "" {
		m.err = "Job to Be Done is required."
		m.setCreateField(1)
		return m, nil
	}
	if strings.TrimSpace(m.createDraft.Curator) == "" {
		m.err = "Curator is required."
		m.setCreateField(2)
		return m, nil
	}
	if m.createDraft.Slug == "" {
		m.createDraft.Slug = slug(m.createDraft.Title)
	}
	path := filepath.Join(m.baseDir, m.createDraft.Slug)
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			m.err = fmt.Sprintf("A project named %q already exists. Choose another name or open it from Projects.", m.createDraft.Slug)
		} else {
			m.err = fmt.Sprintf("%s already exists and is not a project folder.", path)
		}
		return m, nil
	} else if !os.IsNotExist(err) {
		m.err = fmt.Sprintf("Could not inspect %s: %s", path, err)
		return m, nil
	}
	m.createRunning = true
	m.createError = ""
	m.err = ""
	m.note = "Project creation accepted."
	return m, createProject(m.runner, m.baseDir, m.createDraft)
}

func (m Model) submitClarification() (Model, tea.Cmd) {
	m.commitClarifyInput()
	nextTape := applyClarificationAnswers(m.currentTape, m.clarifyQuestions, m.clarifyAnswers)
	nextTape = markTapeClarificationComplete(nextTape)
	return m, saveClarification(m.currentPath, nextTape)
}

func createFieldCount() int {
	return 4
}

func applyClarificationAnswers(current tape.Tape, questions []clarifyQuestion, answers []string) tape.Tape {
	for i, answer := range answers {
		answer = strings.TrimSpace(answer)
		if answer == "" {
			continue
		}
		if i >= len(questions) {
			continue
		}
		current.JTBDClarifications = append(current.JTBDClarifications, tape.Clarification{
			Question: questions[i].question,
			Answer:   answer,
		})
	}
	return current
}

func (m Model) viewCreate() string {
	width := setupContentWidth(m.width)
	if m.createRunning {
		title := "Creating Liner Project"
		section := "Accepted submission"
		status := "Liner Core is creating this Project. Additional submit input is disabled until the result returns."
		rows := []labelValueRow{
			{Label: "Name", Value: m.createDraft.Title},
			{Label: "Job to Be Done", Value: m.createDraft.JTBD},
			{Label: "Curator", Value: m.createDraft.Curator},
			{Label: "Source Inbox", Value: customSourcesValue(m.createDraft.AddSources)},
		}
		if m.createOpenRetryPath != "" {
			title = "Opening Created Liner Project"
			section = "Created Project"
			status = "Liner is reading the already-created Project. Core creation is not running. Additional input is disabled until the open result returns."
			rows = append([]labelValueRow{{Label: "Path", Value: m.createOpenRetryPath}}, rows...)
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			m.renderLoadingTitle(title, true),
			styles.Subtitle.Render("Working"),
			"",
			styles.Section.Render(section),
			renderLabelValueBlock(width, rows, 1, 1),
			styles.MutedText.Render(strings.Join(wrapWords(status, width), "\n")),
		)
	}
	title := m.createDraft.Title
	jtbd := m.createDraft.JTBD
	curator := m.createDraft.Curator
	switch m.createStep {
	case 0:
		title = strings.TrimSpace(m.createInput.Value())
	case 1:
		jtbd = strings.TrimSpace(m.createArea.Value())
	case 2:
		curator = strings.TrimSpace(m.createInput.Value())
	}
	prompt := strings.Join(wrapWords(createStepPrompt(m.createStep), width), "\n")
	rows := strings.Join([]string{
		m.setupInlineRow("Name", title, m.createStep == 0, width),
		m.setupInlineRow("Job to Be Done", jtbd, m.createStep == 1, width),
		m.setupInlineRow("Curator", curator, m.createStep == 2, width),
		m.setupInlineRow("Source Inbox", customSourcesValue(m.createDraft.AddSources), m.createStep == 3, width),
	}, "\n")
	parts := []string{
		styles.Title.Render("Setup"),
		styles.Subtitle.Render(createStepSubtitle(m.createStep)),
	}
	if m.createError != "" {
		guidance := "The reviewed draft is preserved. Press enter to retry."
		if m.createOpenRetryPath != "" {
			guidance = "The Project was created at " + m.createOpenRetryPath + ". Press enter to retry opening it; Core creation will not run again."
		}
		parts = append(parts,
			"",
			styles.ReportSection.Render("Creation failed"),
			styles.ErrorText.Render(strings.Join(wrapWords(m.createError, width), "\n")),
			styles.MutedText.Render(strings.Join(wrapWords(guidance, width), "\n")),
		)
	}
	parts = append(parts,
		"",
		styles.Section.Render(prompt),
		"",
		rows,
	)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) viewClarify() string {
	width := styles.ClampWidth(m.width - 4)
	jtbd := "Not set."
	if m.currentTape.JTBD != nil && strings.TrimSpace(*m.currentTape.JTBD) != "" {
		jtbd = strings.TrimSpace(*m.currentTape.JTBD)
	}
	job := strings.Join(wrapWords(jtbd, width), "\n")
	var body string
	switch {
	case m.clarifyLoading:
		body = lipgloss.JoinVertical(lipgloss.Left,
			styles.Section.Render("Job to Be Done"),
			job,
			"",
			styles.Section.Render("AI is working on it."),
			m.clarifySpin.View()+" "+clarifyLoaderView(m.fxFrame),
		)
	case len(m.clarifyQuestions) == 0:
		message := "No Clarify Job questions were generated."
		if m.clarifyError != "" {
			message = "Could not generate Clarify Job questions: " + m.clarifyError
		}
		body = lipgloss.JoinVertical(lipgloss.Left,
			styles.Section.Render("Job to Be Done"),
			job,
			"",
			styles.ErrorText.Render(message),
			styles.Subtitle.Render("Press Enter to retry question generation, or go back."),
		)
	default:
		question := m.clarifyQuestions[m.clarifyStep].question
		progress := fmt.Sprintf("Question %d of %d", m.clarifyStep+1, len(m.clarifyQuestions))
		body = lipgloss.JoinVertical(lipgloss.Left,
			styles.Section.Render("Job to Be Done"),
			job,
			"",
			styles.Subtitle.Render(progress),
			styles.Section.Render(strings.Join(wrapWords(question, width), "\n")),
			"",
			m.clarifyArea.View(),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderLoadingTitle("Clarify Job", m.clarifyLoading),
		styles.Subtitle.Render("Refine the Job Story before corpus creation."),
		"",
		lipgloss.NewStyle().Width(width).Render(body),
	)
}

func (m Model) setupInlineRow(label string, value string, active bool, width int) string {
	labelWidth := 18
	valueWidth := max(18, width-labelWidth-4)
	prefix := "  "
	labelStyle := styles.Subtitle
	if active {
		prefix = "> "
		labelStyle = styles.Section
	}
	labelText := fmt.Sprintf("%s%-*s ", prefix, labelWidth, label)
	continuation := strings.Repeat(" ", lipgloss.Width(labelText))
	var lines []string
	if active {
		lines = strings.Split(m.createInlineControl(valueWidth), "\n")
	} else {
		lines = wrapWords(valueOr(value, "..."), valueWidth)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	for i := range lines {
		if i == 0 {
			lines[i] = labelStyle.Render(labelText) + styleSetupValue(lines[i], active)
		} else {
			lines[i] = continuation + styleSetupValue(lines[i], active)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) createInlineControl(width int) string {
	if m.createStep == 1 {
		return m.createArea.View()
	}
	if m.createStep == 3 {
		return renderCustomSourceSelector(m.createDraft.AddSources, width)
	}
	return m.createInput.View()
}

func styleSetupValue(value string, active bool) string {
	if active {
		return value
	}
	return styles.Subtitle.Render(value)
}

func valueOr(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func customSourcesValue(addSources bool) string {
	if addSources {
		return "Yes"
	}
	return "No"
}

func renderCustomSourceSelector(addSources bool, _ int) string {
	yes := selectorOption("Yes", addSources)
	no := selectorOption("No", !addSources)
	return yes + "  " + no
}

const clarifyLoaderCopy = "Reading the Job to Be Done and preparing Clarify Job questions."

func clarifyLoaderMessage(frame int) string {
	return clarifyLoaderCopy
}

func clarifyLoaderView(frame int) string {
	if (frame/18)%2 == 1 {
		return styles.SoftText.Render(clarifyLoaderMessage(frame))
	}
	return styles.Subtitle.Render(clarifyLoaderMessage(frame))
}

func selectorOption(label string, active bool) string {
	if active {
		return styles.NextActionTitle.Render("☑ " + label)
	}
	return styles.SoftText.Render("☐ " + label)
}

func newCreateArea(width int) textarea.Model {
	createArea := textarea.New()
	createArea.Prompt = ""
	createArea.Placeholder = "..."
	createArea.ShowLineNumbers = false
	createArea.DynamicHeight = true
	createArea.MinHeight = 1
	createArea.MaxHeight = 6
	createArea.SetWidth(width)
	createArea.SetHeight(1)
	areaStyles := textarea.DefaultDarkStyles()
	areaStyles.Focused.CursorLine = lipgloss.NewStyle()
	areaStyles.Focused.Text = styles.InputFocused
	areaStyles.Focused.Prompt = styles.InputFocused
	areaStyles.Focused.Placeholder = styles.InputPlaceholder
	areaStyles.Blurred.CursorLine = lipgloss.NewStyle()
	areaStyles.Blurred.Text = styles.InputBlurred
	areaStyles.Blurred.Prompt = styles.InputPlaceholder
	areaStyles.Blurred.Placeholder = styles.InputPlaceholder
	areaStyles.Cursor.Color = styles.InputCursor
	areaStyles.Cursor.Shape = tea.CursorBar
	createArea.SetStyles(areaStyles)
	createArea.Focus()
	return createArea
}

func newClarifyArea(width int) textarea.Model {
	clarifyArea := textarea.New()
	clarifyArea.Prompt = "> "
	clarifyArea.Placeholder = "your answer (or leave blank to skip)"
	clarifyArea.ShowLineNumbers = false
	clarifyArea.DynamicHeight = true
	clarifyArea.MinHeight = 1
	clarifyArea.MaxHeight = 6
	clarifyArea.SetWidth(width)
	clarifyArea.SetHeight(1)
	areaStyles := textarea.DefaultDarkStyles()
	areaStyles.Focused.CursorLine = lipgloss.NewStyle()
	areaStyles.Focused.Text = styles.InputFocused
	areaStyles.Focused.Prompt = styles.InputFocused
	areaStyles.Focused.Placeholder = styles.InputPlaceholder
	areaStyles.Blurred.CursorLine = lipgloss.NewStyle()
	areaStyles.Blurred.Text = styles.InputBlurred
	areaStyles.Blurred.Prompt = styles.InputPlaceholder
	areaStyles.Blurred.Placeholder = styles.InputPlaceholder
	areaStyles.Cursor.Color = styles.InputCursor
	areaStyles.Cursor.Shape = tea.CursorBar
	clarifyArea.SetStyles(areaStyles)
	clarifyArea.Focus()
	return clarifyArea
}

func createStepPrompt(step int) string {
	switch step {
	case 0:
		return "Name this Liner Project."
	case 1:
		return "Describe the Job to Be Done. A useful Job Story names the situation, motivation, and outcome; Liner will infer the research lanes."
	case 2:
		return "Who is the Curator for this Liner Project?"
	case 3:
		return "Would you like to add Sources before research starts?"
	default:
		return ""
	}
}

func createStepSubtitle(step int) string {
	return fmt.Sprintf("Setup %d of %d - %s", min(max(step, 0), createFieldCount()-1)+1, createFieldCount(), createStepName(step))
}

func createStepName(step int) string {
	switch step {
	case 0:
		return "Name the Liner Project."
	case 1:
		return "Define the Job to Be Done."
	case 2:
		return "Name the Curator."
	case 3:
		return "Choose source capture."
	default:
		return "Set up the Liner Project."
	}
}

func createInputWidth(width int) int {
	return setupInlineValueWidth(width)
}

func createAreaWidth(width int) int {
	return setupInlineValueWidth(width)
}

func setupContentWidth(width int) int {
	if width <= 0 {
		return 92
	}
	return styles.ClampWidth(width - 4)
}

func setupInlineValueWidth(width int) int {
	labelWidth := 18
	return max(24, setupContentWidth(width)-labelWidth-4)
}

func clarifyAreaWidth(width int) int {
	if width <= 0 {
		return 76
	}
	return max(18, styles.ClampWidth(width-4)-lipgloss.Width("> "))
}

func createProject(r core.Runner, baseDir string, draft createDraft) tea.Cmd {
	return func() tea.Msg {
		path := filepath.Join(baseDir, draft.Slug)
		if err := r.InitProjectWithMetadata(
			path,
			draft.Title,
			createDraftDescription(draft),
			draft.Curator,
			draft.JTBD,
		); err != nil {
			return projectCreatedMsg{path: path, err: err}
		}
		t, err := tape.ReadProject(path)
		return projectCreatedMsg{path: path, tape: t, created: true, err: err}
	}
}

func readCreatedProject(path string) tea.Cmd {
	return func() tea.Msg {
		t, err := tape.ReadProject(path)
		return projectCreatedMsg{path: path, tape: t, created: true, err: err}
	}
}

const pendingProjectDescription = "AI-generated description pending."

func createDraftDescription(draft createDraft) string {
	if description := descriptionFromJob(draft.JTBD); description != "" {
		return description
	}
	if titleDescription := descriptionFromTitle(draft.Title); titleDescription != "" {
		return titleDescription
	}
	return "Liner project."
}

func projectSummaryDescription(project core.ProjectSummary) string {
	description := strings.TrimSpace(project.Description)
	if description != "" && description != pendingProjectDescription {
		return description
	}
	if project.JTBD != nil {
		if description := descriptionFromJob(*project.JTBD); description != "" {
			return description
		}
	}
	if titleDescription := descriptionFromTitle(fallbackText(project.Title, project.Name)); titleDescription != "" {
		return titleDescription
	}
	return "No description yet."
}

func projectTapeDescription(current tape.Tape) string {
	description := strings.TrimSpace(current.Description)
	if description != "" && description != pendingProjectDescription {
		return description
	}
	if current.JTBD != nil {
		if description := descriptionFromJob(*current.JTBD); description != "" {
			return description
		}
	}
	if titleDescription := descriptionFromTitle(current.Title); titleDescription != "" {
		return titleDescription
	}
	return "No description yet."
}

func descriptionFromJob(job string) string {
	job = strings.Join(strings.Fields(job), " ")
	if job == "" {
		return ""
	}
	first := firstSentence(job)
	if len([]rune(first)) > 120 {
		return ""
	}
	return ensureSentencePeriod(first)
}

func descriptionFromTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return "Guidance for " + ensureNoTrailingPeriod(title) + "."
}

func firstSentence(value string) string {
	value = strings.TrimSpace(value)
	for index, r := range value {
		if r == '.' || r == '!' || r == '?' {
			return strings.TrimSpace(value[:index])
		}
	}
	return value
}

func ensureSentencePeriod(value string) string {
	value = ensureNoTrailingPeriod(value)
	if value == "" {
		return ""
	}
	return value + "."
}

func ensureNoTrailingPeriod(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), ".!?")
}

func saveClarification(project string, current tape.Tape) tea.Cmd {
	return func() tea.Msg {
		err := tape.WriteProject(project, current)
		if err == nil {
			err = removeClarificationDraft(project)
		}
		return clarificationSavedMsg{tape: current, err: err}
	}
}

func generateClarifyingQuestions(project string, jtbd string) tea.Cmd {
	return func() tea.Msg {
		questions, err := core.GenerateClarifyingQuestions(project, jtbd, 60*time.Second)
		return clarificationQuestionsMsg{questions: questions, err: err}
	}
}

func tapeJTBD(current tape.Tape) string {
	if current.JTBD == nil {
		return ""
	}
	return strings.TrimSpace(*current.JTBD)
}
