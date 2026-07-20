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
	screenMaintenance
	screenSynthesisReview
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

	importPicker          filepicker.Model
	importBusy            bool
	settings              settingsInfo
	settingsPane          settingsPane
	settingsMenuCursor    int
	settingsCursor        int
	settingsRow           int
	settingsModelCursor   int
	settingsEffortCursor  int
	settingsCustomEditing bool
	settingsInput         textinput.Model

	onboardingStep           int
	onboardingDirInput       textinput.Model
	onboardingEditingDir     bool
	onboardingProviderCursor int
	onboardingJSCursor       int
	jsSetupRunning           bool
	jsSetupRetryCompile      bool
	jsSetupFromOnboarding    bool

	currentPath               string
	currentTape               tape.Tape
	statusPath                string
	status                    *core.ProjectStatus
	statusErr                 string
	projectSnapshotPath       string
	projectSnapshot           *core.MaintenanceProjectSnapshot
	projectSnapshotErr        string
	projectSnapshotAttempted  bool
	projectSnapshotLoading    bool
	projectSnapshotRefreshing bool
	projectPane               int

	createStep          int
	createInput         textinput.Model
	createArea          textarea.Model
	createDraft         createDraft
	createRunning       bool
	createError         string
	createOpenRetryPath string

	clarifyStep      int
	clarifyArea      textarea.Model
	clarifyQuestions []clarifyQuestion
	clarifyAnswers   []string
	clarifyLoading   bool
	clarifyError     string

	sourceInput                          textinput.Model
	sourcePlan                           source.Preview
	sourceItems                          []source.StagedSource
	sourceWarnings                       []string
	sourceTable                          table.Model
	sourceMaintenancePlan                *core.ProjectChangeSet
	sourceBatchRunning                   bool
	sourceBatchPhase                     string
	sourceBatchTotal                     int
	sourceBatchPrepared                  int
	sourceBatchCancelRequested           bool
	sourceBatchRunID                     uint64
	sourceBatchPlanValidated             bool
	sourceBatchApprovalCaptured          bool
	assemblyAwaitingSnapshot             bool
	assemblyReceipt                      *core.ChangeReceipt
	maintenanceInput                     textinput.Model
	maintenanceSnapshot                  *core.MaintenanceProjectSnapshot
	maintenancePlan                      *core.ProjectChangeSet
	maintenanceReceipt                   *core.ChangeReceipt
	maintenanceLoading                   bool
	maintenanceSnapshotPending           bool
	maintenanceStage                     maintenanceStage
	maintenancePlanView                  viewport.Model
	maintenanceApplying                  bool
	maintenanceReconcile                 bool
	maintenanceReplayPath                string
	maintenanceOperation                 int
	maintenanceSourceCursor              int
	maintenanceFieldCursor               int
	maintenanceEditing                   bool
	maintenanceFieldValues               map[string]string
	maintenanceTouched                   map[string]bool
	synthesisReviewCurrent               viewport.Model
	synthesisReviewCurrentText           string
	synthesisReviewKind                  semanticReviewKind
	synthesisReviewPlanView              viewport.Model
	synthesisReviewArea                  textarea.Model
	operatingLayerReviewSkillArea        textarea.Model
	operatingLayerReviewSkillCurrentText string
	operatingLayerReviewSkillPath        string
	operatingLayerReviewArtifact         int
	synthesisReviewChoice                int
	synthesisReviewEditing               bool
	synthesisReviewPlan                  *core.ProjectChangeSet
	synthesisReviewLoading               bool
	synthesisReviewApplying              bool
	synthesisReviewReconcile             bool
	synthesisReviewAwaitingCompile       bool
	skillItems                           []skillFile
	skillTable                           table.Model
	auditItems                           []auditFile
	auditTable                           table.Model
	evalItems                            []evalFile
	evalTable                            table.Model
	compositionItems                     []compositionFile
	compositionTable                     table.Model

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
	methodologyCancelled      bool
	methodologyLastErr        string
	methodologyFailureKind    string
	methodologyPrimaryFailure string
	methodologyRecovery       string
	methodologyDiagnostics    []string
	methodologyRawLog         []string
	methodologyLogPath        string
	methodologyRunID          uint64
	boardIndex                int
	clarifySpin               spinner.Model

	compileEvents                           <-chan core.CompileEvent
	compileDone                             <-chan error
	compileLines                            []string
	compileSpin                             spinner.Model
	compileBar                              progress.Model
	compiling                               bool
	compileTotal                            int
	compileDoneN                            int
	compileFailed                           int
	compileResult                           *core.CompileResultPayload
	compileWarningIndex                     int
	compilePane                             int
	compileSourceIndex                      int
	compileSourcesReviewed                  bool
	compileRepairAttempted                  bool
	compileRepairRetryCompileAfterRecovery  bool
	compileRepairRebuildCorpusAfterRecovery bool
	compileErr                              string
	compileRows                             []compileSourceRow
	compileMaintenancePlan                  *core.ProjectChangeSet
	sourceEntryReturnScreen                 screen
	sourceEntryReturnSet                    bool
	sourceRecovery                          *sourceRecoveryResult
	sourceRecoveryError                     string
	sourceRecoveryRunning                   bool
	sourceRecoveryReview                    bool
	improvementCursor                       int
	improvementDelta                        *improvementDelta
	improvementBaseline                     *core.MaintenanceProjectSnapshot
	improvementPlan                         *core.ProjectChangeSet
	improvementReceipt                      *core.ChangeReceipt
	improvementLoading                      bool
	improvementApplying                     bool
	improvementReconcile                    bool

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
	path    string
	tape    tape.Tape
	created bool
	err     error
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
	plan    *core.ProjectChangeSet
	receipt *core.ChangeReceipt
	err     error
	batch   bool
	runID   uint64
}

type sourceBatchPlannedMsg struct {
	preview source.Preview
	plan    core.ProjectChangeSet
	err     error
	runID   uint64
}

type sourceBatchValidatedMsg struct {
	preview source.Preview
	plan    core.ProjectChangeSet
	err     error
	runID   uint64
}

type sourceIngestedMsg struct {
	items    []source.StagedSource
	warnings []string
	err      error
}

type sourceManifestSavedMsg struct {
	err error
}

type maintenanceSnapshotMsg struct {
	snapshot core.MaintenanceProjectSnapshot
	err      error
}

type projectSnapshotMsg struct {
	path     string
	snapshot core.MaintenanceProjectSnapshot
	err      error
}

type projectStatusRefreshedMsg struct {
	path   string
	status core.ProjectStatus
	err    error
}

type maintenancePlanMsg struct {
	plan core.ProjectChangeSet
	err  error
}

type maintenanceAppliedMsg struct {
	receipt core.ChangeReceipt
	path    string
	err     error
}

type maintenanceReconciledMsg struct {
	snapshot core.MaintenanceProjectSnapshot
	path     string
	err      error
}

type synthesisReviewPlannedMsg struct {
	plan core.ProjectChangeSet
	err  error
}

type synthesisReviewAppliedMsg struct {
	receipt core.ChangeReceipt
	err     error
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
	runID uint64
}

type methodologyDoneMsg struct {
	err   error
	runID uint64
}

type improvementDeltaPlannedMsg struct {
	delta improvementDelta
	plan  core.ProjectChangeSet
	err   error
}

type improvementAppliedMsg struct {
	receipt     core.ChangeReceipt
	snapshot    core.MaintenanceProjectSnapshot
	snapshotErr error
	err         error
}

type improvementSnapshotMsg struct {
	snapshot core.MaintenanceProjectSnapshot
	err      error
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
