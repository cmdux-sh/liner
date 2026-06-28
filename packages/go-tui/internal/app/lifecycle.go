package app

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func New(opts Options) (Model, error) {
	runner, err := core.Resolve()
	if err != nil {
		return Model{}, err
	}
	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = core.DefaultBaseDir()
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return Model{}, err
	}
	settings := readSettingsInfo()
	screen := screenHome
	onboardingStep := onboardingStepLibrary
	if !settings.OnboardingCompleted {
		screen = screenOnboarding
	} else if !settings.JSSetupPrompted {
		screen = screenOnboarding
		onboardingStep = onboardingStepJS
	}

	commands := newCommandList(70, 16)

	helpModel := help.New()

	input := textinput.New()
	input.Placeholder = "Paste one URL, article, file path, repo, or local document..."
	input.SetWidth(80)
	input.Focus()

	createInput := textinput.New()
	createInput.Prompt = ""
	createInput.Placeholder = "..."
	createInputStyles := textinput.DefaultDarkStyles()
	createInputStyles.Focused.Text = styles.InputFocused
	createInputStyles.Focused.Placeholder = styles.InputPlaceholder
	createInputStyles.Focused.Prompt = styles.InputFocused
	createInputStyles.Blurred.Text = styles.InputBlurred
	createInputStyles.Blurred.Placeholder = styles.InputPlaceholder
	createInputStyles.Cursor.Color = styles.InputCursor
	createInputStyles.Cursor.Shape = tea.CursorBar
	createInput.SetStyles(createInputStyles)
	createInput.SetWidth(64)
	createInput.Focus()

	createArea := newCreateArea(64)
	clarifyArea := newClarifyArea(64)

	spin := newLoadingSpinner()
	compileBar := newCompileProgress(48)
	researchSpin := newLoadingSpinner()
	researchLog := viewport.New(viewport.WithWidth(80), viewport.WithHeight(8))
	configureMethodologyLogViewport(&researchLog, 80, 8)
	clarifySpin := spinner.New(
		spinner.WithSpinner(spinner.Meter),
		spinner.WithStyle(styles.AccentText),
	)
	operatingLayerSpin := spinner.New(
		spinner.WithSpinner(spinner.Meter),
		spinner.WithStyle(styles.AccentText),
	)

	m := Model{
		baseDir:                  baseDir,
		runner:                   runner,
		screen:                   screen,
		projectTable:             newProjectTable(80, 10),
		commands:                 commands,
		help:                     helpModel,
		importPicker:             newImportPicker(".", 12),
		settings:                 settings,
		sourceInput:              input,
		sourceTable:              newSourceTable(80, 8),
		skillTable:               newSkillTable(80, 8),
		auditTable:               newAuditTable(80, 8),
		evalTable:                newEvalTable(80, 8),
		compositionTable:         newCompositionTable(80, 8),
		createInput:              createInput,
		createArea:               createArea,
		clarifyArea:              clarifyArea,
		clarifySpin:              clarifySpin,
		operatingLayerSpin:       operatingLayerSpin,
		compileSpin:              spin,
		compileBar:               compileBar,
		researchSpin:             researchSpin,
		researchLog:              researchLog,
		preview:                  viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
		onboardingDirInput:       newOnboardingDirInput(baseDir, 80),
		onboardingStep:           onboardingStep,
		onboardingProviderCursor: settingsProviderIndex(onboardingDefaultProvider(settings)),
	}
	m.commands.SetItems(m.commandItems())
	return m, nil
}

func newLoadingSpinner() spinner.Model {
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	return spin
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadProjects(m.runner, m.baseDir), m.compileSpin.Tick, m.researchSpin.Tick, m.clarifySpin.Tick, m.operatingLayerSpin.Tick, m.compileBar.Init())
}

func (m Model) View() tea.View {
	var body string
	switch m.screen {
	case screenHome:
		body = m.viewHome()
	case screenProjects:
		body = m.viewProjects()
	case screenProject:
		body = m.viewProject()
	case screenSources:
		body = m.viewSources()
	case screenSourceReview:
		body = m.viewSourceReview()
	case screenResearch:
		body = m.viewResearch()
	case screenAssemblyReview:
		body = m.viewAssemblyReview()
	case screenLinerReview:
		body = m.viewLinerReview()
	case screenSkills:
		body = m.viewSkills()
	case screenSkillReview:
		body = m.viewSkillReview()
	case screenAudits:
		body = m.viewAudits()
	case screenContradictionCleanupReview:
		body = m.viewContradictionCleanupReview()
	case screenSourceNoteCleanupReview:
		body = m.viewSourceNoteCleanupReview()
	case screenEvals:
		body = m.viewEvals()
	case screenComposition:
		body = m.viewComposition()
	case screenCompositionReview:
		body = m.viewCompositionReview()
	case screenReport:
		body = m.viewReport()
	case screenBoard:
		body = m.viewBoard()
	case screenCompile:
		body = m.viewCompile()
	case screenImprovementReview:
		body = m.viewImprovementReview()
	case screenPreview:
		body = m.viewPreview()
	case screenCreate:
		body = m.viewCreate()
	case screenClarify:
		body = m.viewClarify()
	case screenImport:
		body = m.viewImport()
	case screenSettings:
		body = m.viewSettings()
	case screenOnboarding:
		body = m.viewOnboarding()
	}
	banner := m.viewBanner()
	activity := m.viewActivity()
	footer := styles.Help.Render(m.footerHelp())
	if m.note != "" {
		footer += "\n" + styles.SuccessText.Render(m.note)
	}
	if m.err != "" {
		footer += "\n" + styles.ErrorText.Render(m.err)
	}
	body = fitBodyAboveFooter(body, m.height, m.width, banner, activity, footer)

	parts := []string{banner, "", body}
	if activity != "" {
		parts = append(parts, "", activity)
	}
	parts = append(parts, footer)
	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	view.AltScreen = true
	return view
}

func fitBodyAboveFooter(body string, terminalHeight int, terminalWidth int, banner string, activity string, footer string) string {
	if terminalHeight <= 0 || body == "" {
		return body
	}
	reserved := lipgloss.Height(banner) + 1 + lipgloss.Height(footer)
	if strings.TrimSpace(activity) != "" {
		reserved += 1 + lipgloss.Height(activity)
	}
	available := terminalHeight - reserved
	if available <= 0 {
		return ""
	}
	if lipgloss.Height(body) <= available {
		return body
	}
	return truncateViewLines(body, available, terminalWidth)
}

func truncateViewLines(body string, maxLines int, terminalWidth int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= maxLines {
		return body
	}
	if maxLines == 1 {
		return styles.Subtitle.Render(truncateMiddle("… more content hidden above the footer", styles.ClampWidth(terminalWidth-4)))
	}
	notice := truncateMiddle("… more content hidden; use this screen's controls or expand the terminal", styles.ClampWidth(terminalWidth-4))
	out := append([]string{}, lines[:maxLines-1]...)
	out = append(out, styles.Subtitle.Render(notice))
	return strings.Join(out, "\n")
}
