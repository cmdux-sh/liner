package app

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/agent"
	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type Options struct {
	BaseDir string
}

type screen int

const (
	screenHome screen = iota
	screenProjects
	screenProject
	screenSources
	screenSourceReview
	screenResearch
	screenAssemblyReview
	screenLinerReview
	screenSkills
	screenSkillReview
	screenAudits
	screenContradictionCleanupReview
	screenSourceNoteCleanupReview
	screenEvals
	screenComposition
	screenCompositionReview
	screenReport
	screenBoard
	screenCompile
	screenImprovementReview
	screenPreview
	screenCreate
	screenClarify
	screenImport
	screenSettings
	screenOnboarding
)

type Model struct {
	baseDir string
	runner  core.Runner

	screen        screen
	previous      screen
	width         int
	height        int
	researchReady bool
	fxFrame       int

	projects      []core.ProjectSummary
	projectItems  []projectItem
	projectShown  []projectItem
	projectTable  table.Model
	homeFilter    string
	homeFiltering bool
	commands      list.Model
	help          help.Model

	importPicker   filepicker.Model
	importBusy     bool
	settings       settingsInfo
	settingsCursor int

	onboardingStep           int
	onboardingDirInput       textinput.Model
	onboardingEditingDir     bool
	onboardingProviderCursor int
	onboardingJSCursor       int
	jsSetupRunning           bool
	jsSetupRetryCompile      bool
	jsSetupFromOnboarding    bool

	currentPath string
	currentTape tape.Tape
	statusPath  string
	status      *core.ProjectStatus
	statusErr   string
	projectPane int

	createStep  int
	createInput textinput.Model
	createArea  textarea.Model
	createDraft createDraft

	clarifyStep      int
	clarifyArea      textarea.Model
	clarifyQuestions []clarifyQuestion
	clarifyAnswers   []string
	clarifyLoading   bool
	clarifyError     string

	sourceInput      textinput.Model
	sourcePlan       source.Preview
	sourceItems      []source.StagedSource
	sourceWarnings   []string
	sourceTable      table.Model
	skillItems       []skillFile
	skillTable       table.Model
	auditItems       []auditFile
	auditTable       table.Model
	evalItems        []evalFile
	evalTable        table.Model
	compositionItems []compositionFile
	compositionTable table.Model

	researchLines             []string
	researchLog               viewport.Model
	researchStep              int
	researchSpin              spinner.Model
	researchDone              bool
	methodologyCancel         context.CancelFunc
	methodologyEvents         <-chan agent.Event
	methodologyDone           <-chan error
	methodologyPhaseIndex     int
	methodologyPhaseID        string
	methodologyEventCount     int
	methodologyLastEventFrame int
	methodologyFailed         bool
	methodologyLastErr        string
	boardIndex                int
	clarifySpin               spinner.Model

	compileEvents         <-chan core.CompileEvent
	compileDone           <-chan error
	compileLines          []string
	compileSpin           spinner.Model
	compileBar            progress.Model
	compiling             bool
	compileTotal          int
	compileDoneN          int
	compileFailed         int
	compileResult         *core.CompileResultPayload
	compileWarningIndex   int
	compilePane           int
	compileSourceIndex    int
	compileErr            string
	compileRows           []compileSourceRow
	sourceRecovery        *sourceRecoveryResult
	sourceRecoveryError   string
	sourceRecoveryRunning bool
	sourceRecoveryReview  bool
	improvementCursor     int

	operatingLayerRunning   bool
	operatingLayerComplete  bool
	operatingLayerStep      int
	operatingLayerContent   string
	operatingLayerSkillName string
	operatingLayerSkillPath string
	operatingLayerSpin      spinner.Model

	preview        viewport.Model
	previewRel     string
	previewBack    screen
	hasPreviewBack bool
	note           string
	err            string
}

type compileSourceRow struct {
	Status string
	Type   string
	Source string
	Detail string
}

type createDraft struct {
	Slug        string
	Title       string
	Description string
	Curator     string
	JTBD        string
	AddSources  bool
}

type clarifyQuestion struct {
	question string
}

type projectsLoadedMsg struct {
	projects []core.ProjectSummary
	err      error
}

type projectOpenedMsg struct {
	path string
	tape tape.Tape
	err  error
}

type projectCreatedMsg struct {
	path string
	tape tape.Tape
	err  error
}

type projectStatusLoadedMsg struct {
	path   string
	status core.ProjectStatus
	err    error
}

type archiveImportedMsg struct {
	path string
	tape tape.Tape
	err  error
}

type clarificationSavedMsg struct {
	tape tape.Tape
	err  error
}

type jsSetupFinishedMsg struct {
	err error
}

type clarificationQuestionsMsg struct {
	questions []string
	err       error
}

type sourcePreviewMsg struct {
	preview source.Preview
	err     error
}

type sourceSavedMsg struct {
	preview source.Preview
	err     error
}

type sourceIngestedMsg struct {
	items    []source.StagedSource
	warnings []string
	err      error
}

type sourceManifestSavedMsg struct {
	err error
}

type pathOpenedMsg struct {
	err error
}

type mixtapeCopiedMsg struct {
	err error
}

type projectSharedMsg struct {
	path string
	err  error
}

type operatingLayerStepMsg struct {
	step      int
	content   string
	skillName string
	skillPath string
	err       error
}

type methodologyEventMsg struct {
	event agent.Event
}

type methodologyDoneMsg struct {
	err error
}

type compileEventMsg struct {
	event core.CompileEvent
}

type compileDoneMsg struct {
	err error
}

type projectItem struct {
	project      core.ProjectSummary
	capabilities capabilitySummary
}

func projectItemsFromSummaries(projects []core.ProjectSummary) []projectItem {
	items := make([]projectItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectItem{
			project:      project,
			capabilities: capabilitiesForProject(project.Path),
		})
	}
	return items
}

func (p projectItem) Title() string { return p.project.Title }
func (p projectItem) Description() string {
	parts := []string{
		intLabel(p.project.SourceCount, "source"),
		homeProjectStatus(p.project, p.capabilities),
	}
	parts = append(parts, p.project.Path)
	return strings.Join(parts, " • ")
}
func (p projectItem) FilterValue() string {
	return p.project.Title + " " + p.project.Description + " " + p.project.Path
}

type commandItem struct {
	title string
	desc  string
	run   func(Model) (Model, tea.Cmd)
}

func (c commandItem) Title() string       { return c.title }
func (c commandItem) Description() string { return c.desc }
func (c commandItem) FilterValue() string { return c.title + " " + c.desc }
