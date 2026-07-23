package app

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	skipComponentUpdate := false
	wasOnboardingDirEditing := m.screen == screenOnboarding && m.onboardingEditingDir
	wasSynthesisReviewEditing := m.screen == screenSynthesisReview && m.synthesisReviewEditing
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.err = ""
		if m.synthesisReviewLoading && m.screen != screenSynthesisReview {
			m.note = "Preparing the Synthesis approval…"
			return m, nil
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(max(0, msg.Width-4))
		m.projectTable.SetWidth(projectBrowserListWidth(msg.Width))
		m.projectTable.SetHeight(projectBrowserHeight(msg.Height, len(m.projectShown)))
		m.projectTable.SetColumns(projectBrowserColumns(projectBrowserListWidth(msg.Width)))
		m.applyHomeProjectFilter()
		m.commands.SetSize(max(0, min(msg.Width-8, 74)), max(0, min(msg.Height-8, 16)))
		m.createInput.SetWidth(createInputWidth(msg.Width))
		m.createArea.SetWidth(createAreaWidth(msg.Width))
		m.clarifyArea.SetWidth(clarifyAreaWidth(msg.Width))
		m.onboardingDirInput.SetWidth(onboardingDirInputWidth(msg.Width))
		m.settingsInput.SetWidth(max(20, styles.ClampWidth(msg.Width-12)))
		m.sourceInput.SetWidth(sourceInputWidth(msg.Width))
		m.maintenanceInput.SetWidth(sourceInputWidth(msg.Width))
		m.maintenancePlanView.SetWidth(max(20, styles.ClampWidth(msg.Width-8)))
		m.maintenancePlanView.SetHeight(max(3, msg.Height-14))
		if m.maintenancePlan != nil {
			m.syncMaintenancePlanView()
		}
		if m.improvementPlan != nil {
			m.syncImprovementPlanView()
		}
		m.synthesisReviewArea.SetWidth(max(20, styles.ClampWidth(msg.Width-8)))
		m.operatingLayerReviewSkillArea.SetWidth(max(20, styles.ClampWidth(msg.Width-8)))
		m.synthesisReviewCurrent.SetWidth(max(20, styles.ClampWidth(msg.Width-8)))
		m.synthesisReviewCurrent.SetHeight(max(3, min(10, msg.Height-20)))
		if m.screen == screenSynthesisReview && m.synthesisReviewPlan == nil {
			m.syncSemanticReviewCurrent(false)
		}
		m.synthesisReviewPlanView.SetWidth(max(20, styles.ClampWidth(msg.Width-8)))
		m.synthesisReviewPlanView.SetHeight(max(3, msg.Height-14))
		if m.synthesisReviewPlan != nil {
			m.syncSynthesisReviewPlanView()
		}
		m.importPicker.SetHeight(importPickerHeight(msg.Height))
		m.sourceTable.SetWidth(styles.ClampWidth(msg.Width - 8))
		m.sourceTable.SetHeight(max(5, msg.Height-14))
		m.sourceTable.SetColumns(sourceColumns(styles.ClampWidth(msg.Width - 8)))
		m.skillTable.SetWidth(styles.ClampWidth(msg.Width - 8))
		m.skillTable.SetHeight(max(5, msg.Height-14))
		m.skillTable.SetColumns(skillColumns(styles.ClampWidth(msg.Width - 8)))
		m.auditTable.SetWidth(styles.ClampWidth(msg.Width - 8))
		m.auditTable.SetHeight(max(5, msg.Height-14))
		m.auditTable.SetColumns(auditColumns(styles.ClampWidth(msg.Width - 8)))
		m.evalTable.SetWidth(styles.ClampWidth(msg.Width - 8))
		m.evalTable.SetHeight(max(5, msg.Height-14))
		m.evalTable.SetColumns(evalColumns(styles.ClampWidth(msg.Width - 8)))
		m.compositionTable.SetWidth(styles.ClampWidth(msg.Width - 8))
		m.compositionTable.SetHeight(max(5, msg.Height-14))
		m.compositionTable.SetColumns(compositionColumns(styles.ClampWidth(msg.Width - 8)))
		m.compileBar.SetWidth(compileProgressWidth(msg.Width))
		researchLogWasAtBottom := m.researchLog.AtBottom()
		configureMethodologyLogViewport(&m.researchLog, styles.ClampWidth(msg.Width-4), methodologyLogHeight(msg.Height))
		m.syncMethodologyLog(researchLogWasAtBottom)
		m.preview.SetWidth(max(1, styles.ClampWidth(msg.Width-4)))
		m.preview.SetHeight(max(1, msg.Height-10))
	case projectsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.projects = msg.projects
		m.projectItems = projectItemsFromSummaries(msg.projects)
		m.applyHomeProjectFilter()
	case projectOpenedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.currentPath = msg.path
		m.currentTape = msg.tape
		m.clearProjectStatus()
		m.beginProjectSnapshotLoad(msg.path)
		m.researchReady = researchReportExists(msg.path)
		m.screen = screenProject
		m.note = "Opened " + msg.tape.Title
		cmds = append(cmds, loadProjectStatus(m.runner, msg.path), loadProjectSnapshot(m.runner, msg.path))
	case projectStatusLoadedMsg:
		if msg.path != m.currentPath {
			break
		}
		m.statusPath = msg.path
		if msg.err != nil {
			m.status = nil
			m.statusErr = msg.err.Error()
			break
		}
		m.status = &msg.status
		m.statusErr = ""
	case projectSnapshotMsg:
		if msg.path != m.currentPath {
			break
		}
		m.projectSnapshotPath = msg.path
		m.projectSnapshotAttempted = true
		m.projectSnapshotLoading = false
		if msg.err != nil {
			m.projectSnapshot = nil
			m.projectSnapshotErr = msg.err.Error()
			if m.synthesisReviewAwaitingCompile {
				m.synthesisReviewAwaitingCompile = false
				m.synthesisReviewLoading = false
				m.synthesisReviewPlan = nil
				m.screen = screenProject
				m.err = "Core approved the synthesis, but Liner could not verify the Compile gate: " + msg.err.Error()
				m.note = "The synthesis receipt is durable. Refresh Project Flow before starting Compile."
			} else if m.assemblyAwaitingSnapshot {
				receipt := m.assemblyReceipt
				m.assemblyAwaitingSnapshot = false
				m.assemblyReceipt = nil
				m.screen = screenProject
				m.err = "Core saved the reviewed Sources, but Liner could not determine the required next step: " + msg.err.Error()
				m.note = maintenanceReceiptNote(receipt, "The atomic Source batch is durable. Refresh Project Flow before continuing.")
			} else if m.screen == screenSynthesisReview && m.synthesisReviewReconcile {
				m.err = "Receipt reconciliation could not refresh the Core Project Snapshot: " + msg.err.Error()
				m.note = "The exact Change Set remains available for receipt replay."
			} else {
				m.err = ""
			}
			break
		}
		m.projectSnapshot = &msg.snapshot
		m.projectSnapshotErr = ""
		if m.synthesisReviewAwaitingCompile {
			m.synthesisReviewAwaitingCompile = false
			m.synthesisReviewLoading = false
			m.synthesisReviewPlan = nil
			next, cmd := m.startCompile()
			m = next
			if cmd == nil {
				m.screen = screenProject
				if m.err == "" {
					m.err = "Compile did not start after synthesis approval."
				}
			} else {
				m.note = "Synthesis approved through the exact Core Change Set. Compiling MIXTAPE.md."
				cmds = append(cmds, cmd)
			}
		} else if m.assemblyAwaitingSnapshot {
			receipt := m.assemblyReceipt
			m.assemblyAwaitingSnapshot = false
			m.assemblyReceipt = nil
			sourceCount := len(m.currentTape.Sources)
			switch m.projectNextKind() {
			case projectNextReviewSynthesis:
				next, cmd := m.startPreparedSynthesisReview()
				m = next
				m.note = maintenanceReceiptNote(receipt, fmt.Sprintf("Saved %d Source(s) atomically. Review the current synthesis before Compile.", sourceCount))
				cmds = append(cmds, cmd)
			case projectNextCompileRefresh, projectNextContinueCorpus:
				next, cmd := m.startCompile()
				m = next
				m.note = maintenanceReceiptNote(receipt, fmt.Sprintf("Saved %d Source(s) atomically. Compiling MIXTAPE.md.", sourceCount))
				cmds = append(cmds, cmd)
			default:
				m.screen = screenProject
				m.err = ""
				m.note = maintenanceReceiptNote(receipt, fmt.Sprintf("Saved %d Source(s) atomically. Continue from the refreshed Project Flow.", sourceCount))
			}
		} else if m.screen == screenSynthesisReview && m.synthesisReviewReconcile {
			m.note = "Core Snapshot refreshed after an ambiguous apply. Press Enter to replay this exact Change Set and recover its durable receipt without duplicating work."
		} else {
			m.err = ""
		}
	case projectStatusRefreshedMsg:
		if msg.path != m.currentPath {
			break
		}
		m.projectSnapshotRefreshing = false
		if msg.err != nil {
			m.err = "Liner Core could not refresh Project status: " + msg.err.Error()
			break
		}
		m.statusPath = msg.path
		m.status = &msg.status
		m.statusErr = ""
		m.projectSnapshotLoading = true
		m.note = "Project status refreshed. Reloading the Core Project Snapshot."
		m.err = ""
		cmds = append(cmds, loadProjectSnapshot(m.runner, msg.path))
	case projectCreatedMsg:
		m.createRunning = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.createError = msg.err.Error()
			m.createOpenRetryPath = ""
			if msg.created {
				m.createOpenRetryPath = msg.path
				m.note = "Project created, but Liner could not open it. Retry will only reopen the created Project."
			} else {
				m.note = "Project creation failed. The reviewed draft is preserved."
			}
			m.screen = screenCreate
			break
		}
		m.createError = ""
		m.createOpenRetryPath = ""
		m.currentPath = msg.path
		m.currentTape = msg.tape
		m.clearProjectStatus()
		m.beginProjectSnapshotLoad(msg.path)
		m.researchReady = false
		m.sourceInput.SetValue("")
		m.sourceItems = []source.StagedSource{}
		m.sourceWarnings = []string{}
		m.applySourcePreview(source.Preview{})
		if m.createDraft.AddSources {
			m.startSourceEntry()
		} else {
			next, cmd := m.startClarificationFlow()
			m = next
			cmds = append(cmds, cmd)
		}
		m.note = "Created and opened " + msg.tape.Title + "."
		cmds = append(cmds, loadProjects(m.runner, m.baseDir), loadProjectStatus(m.runner, msg.path), loadProjectSnapshot(m.runner, msg.path))
	case archiveImportedMsg:
		m.importBusy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.screen = screenImport
			break
		}
		m.err = ""
		m.currentPath = msg.path
		m.currentTape = msg.tape
		m.clearProjectStatus()
		m.beginProjectSnapshotLoad(msg.path)
		m.researchReady = researchReportExists(msg.path)
		m.screen = screenProject
		m.note = "Imported " + msg.tape.Title
		cmds = append(cmds, loadProjects(m.runner, m.baseDir), loadProjectStatus(m.runner, msg.path), loadProjectSnapshot(m.runner, msg.path))
	case clarificationQuestionsMsg:
		m.clarifyLoading = false
		if msg.err != nil {
			m.clarifyError = msg.err.Error()
			m.clarifyQuestions = nil
			m.clarifyAnswers = nil
			break
		}
		m.clarifyError = ""
		m.setClarifyQuestions(msg.questions)
		m.persistClarificationDraft()
	case clarificationSavedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.screen = screenClarify
			break
		}
		m.currentTape = msg.tape
		m.note = ""
		next, cmd := m.startResearch()
		m = next
		cmds = append(cmds, cmd)
	case sourcePreviewMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.applySourcePreview(msg.preview)
		m.note = fmt.Sprintf("Preview: %d sources, %d warnings", len(msg.preview.Sources), len(msg.preview.Warnings))
	case sourceBatchPlannedMsg:
		if msg.runID != m.sourceBatchRunID {
			break
		}
		if msg.err != nil {
			m.sourceBatchRunning = false
			m.sourceBatchPhase = sourceBatchPhaseFailed
			m.sourceMaintenancePlan = nil
			m.sourceBatchPlanValidated = false
			m.err = msg.err.Error()
			m.note = "The Project is unchanged. Press Enter to plan the same reviewed Source set again."
			break
		}
		m.sourceMaintenancePlan = &msg.plan
		m.sourceBatchPlanValidated = false
		if m.sourceBatchCancelRequested {
			m.sourceBatchRunning = false
			m.sourceBatchPhase = sourceBatchPhaseCancelled
			m.err = ""
			m.note = "Cancelled before atomic apply. The Project is unchanged; press Enter to retry the reviewed Core plan."
			break
		}
		m.sourceBatchPhase = sourceBatchPhaseValidation
		m.note = "Validating that the Core Change Set matches every reviewed Source."
		cmds = append(cmds, validateInitialSourceBatch(
			m.sourceItems,
			m.currentTape.Sources,
			m.currentProjectID(),
			msg.plan,
			msg.runID,
		))
	case sourceBatchValidatedMsg:
		if msg.runID != m.sourceBatchRunID {
			break
		}
		if msg.err != nil {
			m.sourceBatchRunning = false
			m.sourceBatchPhase = sourceBatchPhaseFailed
			m.sourceMaintenancePlan = nil
			m.sourceBatchPlanValidated = false
			m.err = msg.err.Error()
			m.note = "The Project is unchanged. Press Enter to plan the reviewed Source set again."
			break
		}
		m.sourceMaintenancePlan = &msg.plan
		m.sourceBatchPlanValidated = true
		if m.sourceBatchCancelRequested {
			m.sourceBatchRunning = false
			m.sourceBatchPhase = sourceBatchPhaseCancelled
			m.err = ""
			m.note = "Cancelled after validation and before atomic apply. The Project is unchanged; press Enter to retry."
			break
		}
		if msg.plan.ApprovalRequired || !m.sourceBatchApprovalCaptured {
			m.sourceBatchRunning = false
			m.sourceBatchPhase = sourceBatchPhaseValidation
			m.err = ""
			m.note = "Review the checked Sources. Press Enter to accept this selection."
			break
		}
		m.sourceBatchPhase = sourceBatchPhaseApply
		m.note = "Applying the validated Source batch atomically through Liner Core."
		cmds = append(cmds, applyInitialSourceBatch(m.runner, m.currentPath, m.sourceItems, msg.plan, false, msg.runID))
	case sourceSavedMsg:
		if msg.batch && msg.runID != m.sourceBatchRunID {
			break
		}
		if msg.batch {
			m.sourceBatchRunning = false
		}
		if msg.err != nil {
			if msg.batch {
				m.sourceBatchPhase = sourceBatchPhaseFailed
			}
			if msg.receipt != nil {
				if refreshed, readErr := tape.ReadProject(m.currentPath); readErr == nil {
					m.currentTape = refreshed
				}
				m.clearProjectStatus()
				m.beginProjectSnapshotLoad(m.currentPath)
				m.sourceMaintenancePlan = nil
				m.sourceBatchPlanValidated = false
				m.note = maintenanceReceiptNote(msg.receipt, "Liner Core committed the atomic Source batch, but the local review manifest could not refresh. The receipt is durable; refresh before retrying.")
				cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
			} else if msg.batch && sourceApplyNeedsFreshPlan(msg.err) {
				m.sourceMaintenancePlan = nil
				m.sourceBatchPlanValidated = false
				m.note = "Core safely refused a stale Source plan. Preparing a fresh exact plan for the same reviewed Source set."
				m.err = msg.err.Error()
				next, retryCmd := m.startInitialSourceBatch()
				m = next
				cmds = append(cmds, retryCmd)
				break
			} else if msg.batch {
				m.note = "Atomic apply did not return a receipt. The exact Change Set is retained only for receipt reconciliation."
			}
			m.err = msg.err.Error()
			break
		}
		if msg.plan != nil {
			m.sourceMaintenancePlan = msg.plan
			m.err = ""
			m.note = "Review the Core preview below. Press Enter again to approve this exact Change Set."
			break
		}
		m.sourceMaintenancePlan = nil
		m.sourceBatchPlanValidated = false
		m.sourceBatchApprovalCaptured = false
		if msg.batch {
			m.sourceBatchPhase = sourceBatchPhaseComplete
		}
		m.applySourcePreview(msg.preview)
		refreshed, err := tape.ReadProject(m.currentPath)
		if err == nil {
			m.currentTape = refreshed
		}
		if msg.batch && m.screen == screenAssemblyReview {
			var finalizeErr error
			m, finalizeErr = m.finalizeAssemblyAcceptance()
			if finalizeErr != nil {
				m.screen = screenProject
				m.err = finalizeErr.Error()
				m.note = maintenanceReceiptNote(msg.receipt, "The atomic Source batch was committed, but assembly completion needs recovery.")
				m.clearProjectStatus()
				m.beginProjectSnapshotLoad(m.currentPath)
				cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
				break
			}
			m.sourceBatchCancelRequested = false
			m.assemblyAwaitingSnapshot = true
			m.assemblyReceipt = msg.receipt
			m.clearProjectStatus()
			m.beginProjectSnapshotLoad(m.currentPath)
			m.err = ""
			m.note = maintenanceReceiptNote(msg.receipt, fmt.Sprintf("Saved %d Source(s) atomically. Checking the required next step.", len(msg.preview.Sources)))
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
			break
		}
		m.clearProjectStatus()
		m.beginProjectSnapshotLoad(m.currentPath)
		m.sourceInput.SetValue("")
		m.applySourcePreview(source.Preview{})
		if msg.batch && m.sourceBatchCancelRequested {
			m.sourceBatchCancelRequested = false
			if m.sourceEntryReturnsToCompile() {
				m.screen = screenCompile
				m.compilePane = compilePaneSources
				m.sourceEntryReturnSet = false
			} else {
				m.screen = screenProject
			}
			m.note = maintenanceReceiptNote(msg.receipt, "Atomic Source apply completed before cancellation. The Project is consistent; continue when ready.")
			m.err = ""
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
			break
		}
		if m.sourceEntryReturnsToCompile() {
			m.screen = screenCompile
			m.compilePane = compilePaneSources
			m.sourceEntryReturnSet = false
			m.compileSourceIndex = clampCompileSourceIndex(m.compileSourceIndex, len(m.compileSourceListItems()))
			m.note = maintenanceReceiptNote(msg.receipt, "Saved source through Liner Core. Retry compile to include it in MIXTAPE.md.")
			m.err = ""
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
			break
		}
		cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
		next, cmd := m.startClarificationFlow()
		m = next
		cmds = append(cmds, cmd)
	case sourceIngestedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.sourceItems = append(m.sourceItems, msg.items...)
		m.sourceWarnings = append(m.sourceWarnings, msg.warnings...)
		m.applySourceItems(m.sourceItems)
		if len(m.sourceItems) > 0 {
			m.sourceTable.SetCursor(len(m.sourceItems) - 1)
		}
		m.sourceInput.SetValue("")
		m.sourcePlan = source.Preview{}
		if len(msg.items) == 0 {
			m.note = "No source was added. Check the warning below, or paste another source."
		} else {
			m.note = fmt.Sprintf("Added %d source(s). Paste another, or finish.", len(msg.items))
		}
		cmds = append(cmds, writeSourceManifest(m.currentPath, m.sourceItems))
	case sourceManifestSavedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
	case maintenanceSnapshotMsg:
		m.maintenanceLoading = false
		m.maintenanceSnapshotPending = false
		if msg.err != nil {
			m.err = msg.err.Error()
			if m.maintenanceReconcile {
				m.note = "Receipt reconciliation could not refresh the Core Snapshot. The exact Change Set remains locked for safe replay."
			}
			break
		}
		m.maintenanceSnapshot = &msg.snapshot
		if m.maintenanceReconcile {
			m.note = "Core Snapshot refreshed after an ambiguous apply. Press Enter to replay this exact Change Set and recover its durable receipt without duplicating work."
		} else {
			m.err = ""
		}
	case maintenanceReconciledMsg:
		m.maintenanceLoading = false
		m.maintenanceSnapshotPending = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.note = "Receipt reconciliation could not locate the reviewed Project identity. The exact Change Set remains locked; retry after the original or destination root is inspectable."
			break
		}
		m.maintenanceSnapshot = &msg.snapshot
		m.maintenanceReplayPath = msg.path
		m.note = "The reviewed Project identity was found without changing the active root. Press Enter to replay the exact Change Set there and recover its durable receipt."
		m.err = ""
	case maintenancePlanMsg:
		m.maintenanceLoading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.note = "Core refused the typed operation. The Project is unchanged; edit fields and preview again."
			break
		}
		m.maintenancePlan = &msg.plan
		m.maintenanceReceipt = nil
		m.maintenanceStage = maintenanceStagePreview
		m.maintenanceReconcile = false
		m.maintenanceReplayPath = ""
		m.syncMaintenancePlanView()
		m.note = "Review the exact Core Change Set below, then press Enter to apply it."
		m.err = ""
		if msg.plan.ApprovalRequired {
			m.note = "Core classified this Change Set as approval-required. Review the exact Core Change Set, then press Enter to confirm and apply it."
		}
	case maintenanceAppliedMsg:
		m.maintenanceLoading = false
		m.maintenanceApplying = false
		if msg.err != nil {
			m.err = msg.err.Error()
			if !maintenanceApplyNeedsReconciliation(msg.err) {
				m.maintenancePlan = nil
				m.maintenanceReconcile = false
				m.maintenanceReplayPath = ""
				if maintenanceOperationUsesActiveSource(m.maintenanceOperation) {
					m.maintenanceStage = maintenanceStageSource
					m.maintenanceSourceCursor = 0
					m.maintenanceFieldValues = nil
					m.maintenanceTouched = nil
				} else {
					m.maintenanceStage = maintenanceStageFields
				}
				m.maintenancePlanView.SetContent("")
				m.maintenanceLoading = true
				m.maintenanceSnapshotPending = true
				m.note = "Core definitively refused the Change Set; the Project is unchanged. Refreshing the Snapshot so you can edit and plan again."
				cmds = append(cmds, inspectMaintenanceProject(m.runner, m.currentPath))
				break
			}
			m.maintenanceReconcile = true
			m.maintenanceReplayPath = ""
			m.maintenanceLoading = true
			m.maintenanceSnapshotPending = true
			m.note = "Core did not return a receipt. Locating the reviewed Project identity at the original or approved destination root; the exact Change Set remains locked for idempotent replay."
			cmds = append(cmds, reconcileMaintenanceProject(m.runner, m.currentPath, *m.maintenancePlan))
			break
		}
		m.currentPath = msg.path
		m.maintenancePlan = nil
		m.maintenanceReceipt = &msg.receipt
		m.maintenanceStage = maintenanceStageReceipt
		m.maintenanceReconcile = false
		m.maintenanceReplayPath = ""
		m.maintenanceSnapshot = nil
		m.maintenanceSnapshotPending = true
		m.maintenanceEditing = false
		m.maintenanceInput.SetValue("")
		m.maintenanceInput.Blur()
		if m.maintenanceOperation == maintenanceOperationDelete {
			m.maintenanceSnapshotPending = false
			m.clearProjectStatus()
			m.note = "Project moved to recoverable Liner Trash. No files were erased. Press Enter to return to Projects."
			m.err = ""
			cmds = append(cmds, loadProjects(m.runner, m.baseDir))
			break
		}
		var refreshErr error
		if refreshed, err := tape.ReadProject(msg.path); err == nil {
			m.currentTape = refreshed
		} else {
			refreshErr = err
		}
		m.clearProjectStatus()
		m.beginProjectSnapshotLoad(msg.path)
		m.note = strings.Join(core.ReceiptSummaryLines(msg.receipt), " · ")
		m.err = ""
		if refreshErr != nil {
			m.err = "Liner Core applied the Change Set and wrote the receipt, but the TUI could not refresh the Project: " + refreshErr.Error()
		}
		cmds = append(cmds, inspectMaintenanceProject(m.runner, msg.path), loadProjectStatus(m.runner, msg.path), loadProjectSnapshot(m.runner, msg.path), loadProjects(m.runner, m.baseDir))
	case synthesisReviewPlannedMsg:
		m.synthesisReviewLoading = false
		m.synthesisReviewApplying = false
		if msg.err != nil {
			m.synthesisReviewPlan = nil
			m.synthesisReviewReconcile = false
			m.err = msg.err.Error()
			m.note = "The " + m.semanticReviewName() + " and Project are unchanged. Review the disposition, then plan again."
			break
		}
		m.synthesisReviewPlan = &msg.plan
		m.screen = screenSynthesisReview
		m.synthesisReviewReconcile = false
		m.syncSynthesisReviewPlanView()
		m.err = ""
		if m.synthesisReviewKind == semanticReviewSynthesis {
			m.note = "Sources are already approved. Press Enter to record Synthesis approval and start Compile."
		} else {
			m.note = "Press Enter to record Operating Layer approval and finish the Project refresh."
		}
	case synthesisReviewAppliedMsg:
		m.synthesisReviewLoading = false
		m.synthesisReviewApplying = false
		if msg.err != nil {
			m.synthesisReviewReconcile = true
			m.err = msg.err.Error()
			m.note = "Core did not return a receipt. Refreshing the Snapshot; the exact Change Set is retained for idempotent receipt replay."
			cmds = append(cmds, loadProjectSnapshot(m.runner, m.currentPath))
			break
		}
		m.synthesisReviewReconcile = false
		m.synthesisReviewEditing = false
		m.synthesisReviewAwaitingCompile = m.synthesisReviewKind == semanticReviewSynthesis
		if m.synthesisReviewAwaitingCompile {
			// Keep the approved card intact and block input while Snapshot verifies
			// the Compile gate. The next visible state should be Compile itself.
			m.synthesisReviewLoading = true
		} else {
			m.synthesisReviewPlan = nil
			m.screen = screenProject
		}
		m.clearProjectStatus()
		m.beginProjectSnapshotLoad(m.currentPath)
		if m.synthesisReviewAwaitingCompile {
			m.note = "Synthesis approval recorded. Checking Compile…"
		} else {
			m.note = "Operating Layer review recorded."
		}
		m.err = ""
		cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath), loadProjects(m.runner, m.baseDir))
	case pathOpenedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
	case mixtapeCopiedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.note = "Copied MIXTAPE.md to clipboard."
	case projectSharedMsg:
		if msg.err != nil {
			m.err = "Archive export failed: " + msg.err.Error()
			break
		}
		m.note = "Created local archive: " + msg.path
	case operatingLayerStepMsg:
		next, cmd := m.applyOperatingLayerStep(msg)
		m = next
		cmds = append(cmds, cmd)
	case methodologyEventMsg:
		if msg.runID != m.methodologyRunID {
			break
		}
		m.applyMethodologyEvent(msg.event)
		cmds = append(cmds, waitMethodologyEvent(m.methodologyEvents, m.methodologyDone, msg.runID))
	case methodologyDoneMsg:
		if msg.runID != m.methodologyRunID {
			break
		}
		next, cmd := m.finishMethodologyPhase(msg.err)
		m = next
		cmds = append(cmds, cmd)
	case improvementDeltaPlannedMsg:
		m.improvementLoading = false
		m.screen = screenImprovementReview
		if msg.err != nil {
			m.improvementDelta = nil
			m.improvementBaseline = nil
			m.improvementPlan = nil
			_ = os.RemoveAll(projectAbsPath(m.currentPath, improvementRunRelPath))
			m.err = msg.err.Error()
			m.note = "The staged delta could not be classified. Canonical Project artifacts are unchanged."
			break
		}
		m.improvementDelta = &msg.delta
		m.improvementPlan = &msg.plan
		m.improvementReceipt = nil
		m.improvementReconcile = false
		m.syncImprovementPlanView()
		m.err = ""
		m.note = "Core classified every proposal as an addition or exact duplicate. Review the atomic Change Set, then press Enter to apply."
	case improvementAppliedMsg:
		m.improvementLoading = false
		m.improvementApplying = false
		if msg.err != nil {
			m.err = msg.err.Error()
			if !maintenanceApplyNeedsReconciliation(msg.err) {
				m.improvementPlan = nil
				m.improvementBaseline = nil
				m.improvementReconcile = false
				_ = os.RemoveAll(projectAbsPath(m.currentPath, improvementRunRelPath))
				m.note = "Core definitively refused the improvement delta. Canonical Project artifacts are unchanged; run or review the pass again."
				break
			}
			m.improvementReconcile = true
			m.improvementLoading = true
			m.note = "Core did not return a receipt. Refreshing the Snapshot; the exact reviewed delta remains locked for idempotent replay."
			cmds = append(cmds, inspectImprovementSnapshot(m.runner, m.currentPath))
			break
		}
		m.improvementReceipt = &msg.receipt
		m.improvementPlan = nil
		m.improvementBaseline = nil
		m.improvementReconcile = false
		if refreshed, err := tape.ReadProject(m.currentPath); err == nil {
			m.currentTape = refreshed
		}
		m.clearProjectStatus()
		if msg.snapshotErr != nil {
			m.screen = screenSynthesisReview
			m.beginProjectSnapshotLoad(m.currentPath)
			m.err = msg.snapshotErr.Error()
			m.note = maintenanceReceiptNote(&msg.receipt, "Core committed the improvement delta, but the refreshed Snapshot is unavailable. Refresh Project Flow before continuing.")
			cmds = append(cmds, loadProjectSnapshot(m.runner, m.currentPath), loadProjectStatus(m.runner, m.currentPath))
			break
		}
		m.projectSnapshotPath = m.currentPath
		m.projectSnapshot = &msg.snapshot
		m.projectSnapshotErr = ""
		m.projectSnapshotAttempted = true
		m.projectSnapshotLoading = false
		if err := recordImprovementDecision(m.currentPath, "applied"); err != nil {
			m.err = "Improvement was applied, but Liner could not save its completion marker: " + err.Error()
		}
		_ = os.RemoveAll(projectAbsPath(m.currentPath, improvementRunRelPath))
		var cmd tea.Cmd
		if m.projectNextKind() == projectNextReviewSynthesis {
			next, reviewCmd := m.startPreparedSynthesisReview()
			m = next
			cmd = reviewCmd
			m.note = maintenanceReceiptNote(&msg.receipt, "Improvement applied atomically. Review the current synthesis against the refreshed corpus.")
		} else {
			m.screen = screenProject
			m.note = maintenanceReceiptNote(&msg.receipt, "Improvement applied atomically. Continuing from the refreshed Core Project Flow state.")
			m.err = ""
		}
		cmds = append(cmds, cmd, loadProjectStatus(m.runner, m.currentPath))
	case improvementSnapshotMsg:
		m.improvementLoading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.note = "Receipt reconciliation could not refresh the Core Snapshot. The exact improvement Change Set remains locked for safe replay."
			break
		}
		m.projectSnapshotPath = m.currentPath
		m.projectSnapshot = &msg.snapshot
		m.projectSnapshotErr = ""
		m.projectSnapshotAttempted = true
		m.projectSnapshotLoading = false
		m.err = ""
		m.note = "Core Snapshot refreshed after an ambiguous apply. Press Enter to replay this exact Change Set and recover its durable receipt without duplicating work."
	case compileEventMsg:
		cmd := m.applyCompileEvent(msg.event)
		cmds = append(cmds, cmd)
		cmds = append(cmds, waitCompileEvent(m.compileEvents, m.compileDone))
	case compileDoneMsg:
		m.compiling = false
		if msg.err != nil {
			m.compileErr = m.friendlyCompileError(msg.err)
			m.compileLines = append(m.compileLines, "× "+m.compileErr)
			m.note = ""
			if !m.compileHasUsableResult() {
				m.err = m.compileErr
			} else {
				m.note = "Compile finished with an error. Review the result before continuing."
			}
		} else {
			m.compileErr = ""
			if len(m.compileAttentionItems()) > 0 {
				m.compileLines = append(m.compileLines, "Compile finished, but the mixtape needs attention.")
				m.note = "Compile finished with source issues. Review sources before continuing."
			} else {
				m.compileLines = append(m.compileLines, "Compile finished.")
				if recovered := m.recoveredCompileWarningCount(); recovered > 0 {
					m.note = fmt.Sprintf("JS rendering recovered %s. Review sources when ready.", intLabel(recovered, "source"))
				} else {
					m.note = "Compile finished. MIXTAPE.md is ready."
				}
			}
		}
		m.recordCompileProgress()
		m.clearProjectStatus()
		if strings.TrimSpace(m.currentPath) != "" {
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
		}
	case sourceRecoveryDoneMsg:
		m.sourceRecoveryRunning = false
		m.sourceRecovery = &msg.result
		m.sourceRecoveryReview = true
		if msg.err != nil {
			m.sourceRecoveryError = msg.err.Error()
			m.err = ""
			m.note = "Unavailable source retry finished with an error. Continue when ready."
			m.compileLines = append(m.compileLines, "× Unavailable source retry failed: "+msg.err.Error())
			break
		}
		m.sourceRecoveryError = ""
		summary := fmt.Sprintf("Unavailable source retry checked %d retryable source(s): %d recovered, %d still unavailable.", msg.result.Attempted, msg.result.Succeeded, msg.result.Failed)
		m.compileLines = append(m.compileLines, summary)
		if msg.result.Succeeded > 0 {
			m.note = "Recovered source content saved. Continue when ready."
			m.compileLines = append(m.compileLines, "Saved recovered source copies under local-sources/recovered/. Run Build Corpus when ready.")
		} else {
			m.note = "Retryable sources are still unavailable. Continue when ready."
		}
		if strings.TrimSpace(m.currentPath) != "" {
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath), loadProjectSnapshot(m.runner, m.currentPath))
		}
		if m.compileRepairRetryCompileAfterRecovery {
			m.compileRepairRetryCompileAfterRecovery = false
			m.compileRepairAttempted = true
			m.compileSourcesReviewed = true
			if msg.result.Succeeded > 0 {
				m.compileRepairRebuildCorpusAfterRecovery = true
				m.note = "Recovered custom source content. Press enter to refresh source evaluation."
			} else {
				m.compileRepairRebuildCorpusAfterRecovery = false
				m.note = "No custom sources recovered. Review sources, repair again, or add replacements."
			}
		}
	case jsSetupFinishedMsg:
		m.jsSetupRunning = false
		if msg.err != nil {
			m.err = "JS rendering setup failed: " + msg.err.Error()
			m.note = ""
			m.jsSetupRetryCompile = false
			m.jsSetupFromOnboarding = false
			break
		}
		if err := writeJSSetupConfig(true, true); err != nil {
			m.err = "JS rendering is ready, but config could not be saved: " + err.Error()
			break
		}
		m.settings = readSettingsInfo()
		if m.jsSetupRetryCompile {
			m.jsSetupRetryCompile = false
			m.jsSetupFromOnboarding = false
			m.note = "JS rendering is ready. Retrying compile."
			next, cmd := m.startCompile()
			m = next
			m.compileRepairAttempted = true
			m.compileSourcesReviewed = false
			cmds = append(cmds, cmd)
			break
		}
		if m.jsSetupFromOnboarding {
			m.jsSetupFromOnboarding = false
			next, cmd := m.finishOnboarding()
			m = next
			m.note = "Onboarding saved. JS rendering is ready."
			cmds = append(cmds, cmd)
			break
		}
		m.note = "JS rendering is ready."
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.fxFrame++
		m.compileSpin, cmd = m.compileSpin.Update(msg)
		cmds = append(cmds, cmd)
		m.researchSpin, cmd = m.researchSpin.Update(msg)
		cmds = append(cmds, cmd)
		m.clarifySpin, cmd = m.clarifySpin.Update(msg)
		cmds = append(cmds, cmd)
		m.operatingLayerSpin, cmd = m.operatingLayerSpin.Update(msg)
		cmds = append(cmds, cmd)
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.compileBar, cmd = m.compileBar.Update(msg)
		cmds = append(cmds, cmd)
	case tea.KeyPressMsg:
		previousScreen := m.screen
		importKey := m.screen == screenImport
		next, cmd := m.handleKey(msg)
		m = next
		if m.screen != previousScreen {
			skipComponentUpdate = true
		}
		if importKey {
			skipComponentUpdate = true
		}
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	if !skipComponentUpdate {
		switch m.screen {
		case screenHome:
			m.commands, cmd = m.commands.Update(msg)
		case screenProjects:
			m.projectTable, cmd = m.projectTable.Update(msg)
		case screenSources:
			if keyMsg, ok := msg.(tea.KeyPressMsg); ok && sourceEntryShouldUpdateTable(m, keyMsg) {
				m.sourceTable, cmd = m.sourceTable.Update(msg)
			} else if keyMsg, ok := msg.(tea.KeyPressMsg); !ok || shouldEditSourceText(m, keyMsg) {
				before := m.sourceInput.Value()
				m.sourceInput, cmd = m.sourceInput.Update(msg)
				if m.sourceInput.Value() != before {
					preview, err := source.Import(m.sourceInput.Value(), m.currentPath, false)
					if err != nil {
						m.err = err.Error()
					} else {
						m.applySourcePreview(preview)
					}
				}
			}
		case screenSourceReview:
			m.sourceTable, cmd = m.sourceTable.Update(msg)
		case screenAssemblyReview:
			m.sourceTable, cmd = m.sourceTable.Update(msg)
		case screenSkills:
			m.skillTable, cmd = m.skillTable.Update(msg)
		case screenSkillReview:
			m.preview, cmd = m.preview.Update(msg)
		case screenAudits:
			m.auditTable, cmd = m.auditTable.Update(msg)
		case screenSourceNoteCleanupReview:
			m.preview, cmd = m.preview.Update(msg)
		case screenEvals:
			m.evalTable, cmd = m.evalTable.Update(msg)
		case screenComposition:
			m.compositionTable, cmd = m.compositionTable.Update(msg)
		case screenCompositionReview:
			m.preview, cmd = m.preview.Update(msg)
		case screenResearch:
			m.researchLog, cmd = m.researchLog.Update(msg)
		case screenImport:
			m.importPicker, cmd = m.importPicker.Update(msg)
		case screenReport:
			m.preview, cmd = m.preview.Update(msg)
		case screenPreview:
			m.preview, cmd = m.preview.Update(msg)
		case screenCreate:
			if !m.createRunning {
				if keyMsg, ok := msg.(tea.KeyPressMsg); !ok || shouldEditCreateText(m, keyMsg) {
					if m.createStep == 1 {
						m.createArea, cmd = m.createArea.Update(msg)
					} else {
						m.createInput, cmd = m.createInput.Update(msg)
					}
				}
			}
		case screenClarify:
			if keyMsg, ok := msg.(tea.KeyPressMsg); m.canEditClarifyText() && (!ok || shouldEditClarifyText(keyMsg)) {
				m.clarifyArea, cmd = m.clarifyArea.Update(msg)
				m.commitClarifyInput()
				m.persistClarificationDraft()
			}
		case screenOnboarding:
			if keyMsg, ok := msg.(tea.KeyPressMsg); wasOnboardingDirEditing && (!ok || shouldEditOnboardingDir(keyMsg)) {
				m.onboardingDirInput, cmd = m.onboardingDirInput.Update(msg)
			}
		case screenSettings:
			if m.settingsCustomEditing || m.settingsPane == settingsPaneProjectsFolder {
				m.settingsInput, cmd = m.settingsInput.Update(msg)
			}
		case screenMaintenance:
			if !m.maintenanceLoading && m.maintenancePlan == nil && m.maintenanceEditing {
				m.maintenanceInput, cmd = m.maintenanceInput.Update(msg)
			}
		case screenSynthesisReview:
			if wasSynthesisReviewEditing && m.synthesisReviewEditing && !m.synthesisReviewLoading && m.synthesisReviewPlan == nil {
				if keyMsg, ok := msg.(tea.KeyPressMsg); !ok || shouldEditSemanticReviewText(keyMsg) {
					area := m.activeSemanticReviewArea()
					updated, nextCmd := area.Update(msg)
					*area = updated
					cmd = nextCmd
				}
			}
		}
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}
