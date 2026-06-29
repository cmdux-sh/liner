package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	skipComponentUpdate := false
	wasOnboardingDirEditing := m.screen == screenOnboarding && m.onboardingEditingDir
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.err = ""
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
		m.sourceInput.SetWidth(sourceInputWidth(msg.Width))
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
		m.researchReady = researchReportExists(msg.path)
		m.screen = screenProject
		m.note = "Opened " + msg.tape.Title
		cmds = append(cmds, loadProjectStatus(m.runner, msg.path))
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
	case projectCreatedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.screen = screenCreate
			break
		}
		m.currentPath = msg.path
		m.currentTape = msg.tape
		m.clearProjectStatus()
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
		cmds = append(cmds, loadProjects(m.runner, m.baseDir))
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
		m.researchReady = researchReportExists(msg.path)
		m.screen = screenProject
		m.note = "Imported " + msg.tape.Title
		cmds = append(cmds, loadProjects(m.runner, m.baseDir), loadProjectStatus(m.runner, msg.path))
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
	case sourceSavedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.applySourcePreview(msg.preview)
		refreshed, err := tape.ReadProject(m.currentPath)
		if err == nil {
			m.currentTape = refreshed
		}
		m.clearProjectStatus()
		m.sourceInput.SetValue("")
		m.applySourcePreview(source.Preview{})
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
		m.applyMethodologyEvent(msg.event)
		cmds = append(cmds, waitMethodologyEvent(m.methodologyEvents, m.methodologyDone))
	case methodologyDoneMsg:
		next, cmd := m.finishMethodologyPhase(msg.err)
		m = next
		cmds = append(cmds, cmd)
	case compileEventMsg:
		cmd := m.applyCompileEvent(msg.event)
		cmds = append(cmds, cmd)
		cmds = append(cmds, waitCompileEvent(m.compileEvents, m.compileDone))
	case compileDoneMsg:
		m.compiling = false
		if msg.err != nil {
			m.compileErr = m.friendlyCompileError(msg.err)
			m.compileLines = append(m.compileLines, "× "+m.compileErr)
			if !m.compileHasUsableResult() {
				m.err = m.compileErr
			}
		} else {
			m.compileErr = ""
			if len(m.compileAttentionItems()) > 0 {
				m.compileLines = append(m.compileLines, "Compile finished, but the mixtape needs attention.")
			} else {
				m.compileLines = append(m.compileLines, "Compile finished.")
			}
		}
		m.recordCompileProgress()
		m.clearProjectStatus()
		if strings.TrimSpace(m.currentPath) != "" {
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath))
		}
	case sourceRecoveryDoneMsg:
		m.sourceRecoveryRunning = false
		m.sourceRecovery = &msg.result
		m.sourceRecoveryReview = true
		if msg.err != nil {
			m.sourceRecoveryError = msg.err.Error()
			m.err = ""
			m.note = "Excluded local source retry finished with an error. Continue when ready."
			m.compileLines = append(m.compileLines, "× Excluded local source retry failed: "+msg.err.Error())
			break
		}
		m.sourceRecoveryError = ""
		summary := fmt.Sprintf("Excluded local source retry checked %d excluded local source(s): %d recovered, %d still unavailable.", msg.result.Attempted, msg.result.Succeeded, msg.result.Failed)
		m.compileLines = append(m.compileLines, summary)
		if msg.result.Succeeded > 0 {
			m.note = "Recovered source content saved. Continue when ready."
			m.compileLines = append(m.compileLines, "Saved recovered source copies under local-sources/recovered/. Run Build Corpus when ready.")
		} else {
			m.note = "Excluded local sources are still unavailable. Continue when ready."
		}
		if strings.TrimSpace(m.currentPath) != "" {
			cmds = append(cmds, loadProjectStatus(m.runner, m.currentPath))
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
			if keyMsg, ok := msg.(tea.KeyPressMsg); !ok || shouldEditCreateText(m, keyMsg) {
				if m.createStep == 1 {
					m.createArea, cmd = m.createArea.Update(msg)
				} else {
					m.createInput, cmd = m.createInput.Update(msg)
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
		}
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}
