package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"gopkg.in/yaml.v3"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func TestSynthesisReviewStillCurrentUsesOneExactApprovalAndStartsCompile(t *testing.T) {
	m, synthesisPath, mixtapePath, linerPath := synthesisReviewProject(t)
	synthesisBefore := mustReadTestFile(t, synthesisPath)
	mixtapeBefore := mustReadTestFile(t, mixtapePath)
	linerBefore := mustReadTestFile(t, linerPath)

	review, cmd := m.startSynthesisReview()
	if cmd != nil || review.screen != screenSynthesisReview {
		t.Fatalf("expected write-free synthesis review surface, screen=%v cmd=%v err=%q", review.screen, cmd, review.err)
	}
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if planCmd == nil || !planning.synthesisReviewLoading {
		t.Fatal("still-current selection should request a Core preview")
	}
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	if plannedMsg.err != nil {
		t.Fatal(plannedMsg.err)
	}
	if !plannedMsg.plan.ApprovalRequired || len(plannedMsg.plan.Operations) != 1 || plannedMsg.plan.Operations[0]["disposition"] != "still_current" {
		t.Fatalf("unexpected still-current Change Set: %#v", plannedMsg.plan)
	}
	assertTestFilesEqual(t, synthesisPath, synthesisBefore, mixtapePath, mixtapeBefore, linerPath, linerBefore)

	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	approvalView := stripANSICodesForTest(planned.viewSynthesisReview())
	applying, applyCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if applyCmd == nil || !applying.synthesisReviewLoading {
		t.Fatal("approved preview should apply exactly once through Core")
	}
	appliedMsg := applyCmd().(synthesisReviewAppliedMsg)
	if appliedMsg.err != nil {
		t.Fatal(appliedMsg.err)
	}
	if appliedMsg.receipt.SynthesisDisposition != "approved_still_current" {
		t.Fatalf("unexpected synthesis receipt: %#v", appliedMsg.receipt)
	}
	completedModel, _ := applying.Update(appliedMsg)
	completed := completedModel.(Model)
	if completed.screen != screenSynthesisReview || !completed.projectSnapshotLoading || !completed.synthesisReviewLoading || completed.synthesisReviewPlan == nil || !strings.Contains(completed.note, "Receipt:") {
		t.Fatalf("completion should keep one stable review surface while checking Compile: screen=%v loading=%v note=%q", completed.screen, completed.projectSnapshotLoading, completed.note)
	}
	if got := stripANSICodesForTest(completed.viewSynthesisReview()); got != approvalView {
		t.Fatalf("approval must remain visually frozen until Compile replaces it\nbefore:\n%s\nafter:\n%s", approvalView, got)
	}
	assertTestFilesEqual(t, synthesisPath, synthesisBefore, mixtapePath, mixtapeBefore, linerPath, linerBefore)

	refreshed, err := m.runner.InspectMaintenanceProject(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedModel, compileCmd := completed.Update(projectSnapshotMsg{path: m.currentPath, snapshot: refreshed})
	updated := updatedModel.(Model)
	if compileCmd == nil || updated.screen != screenCompile || !updated.compiling {
		t.Fatalf("approved synthesis should enter Compile automatically: screen=%v compiling=%v cmd=%v err=%q", updated.screen, updated.compiling, compileCmd, updated.err)
	}
}

func TestPreparedSynthesisReviewShowsExactPlanBeforeItsOnlyApproval(t *testing.T) {
	m, synthesisPath, mixtapePath, linerPath := synthesisReviewProject(t)
	synthesisBefore := mustReadTestFile(t, synthesisPath)
	mixtapeBefore := mustReadTestFile(t, mixtapePath)
	linerBefore := mustReadTestFile(t, linerPath)

	preparing, planCmd := m.startPreparedSynthesisReview()
	if planCmd == nil || !preparing.synthesisReviewLoading || preparing.synthesisReviewPlan != nil {
		t.Fatalf("prepared review should plan before accepting approval: %#v cmd=%v", preparing, planCmd)
	}
	if preparing.screen != screenProject {
		t.Fatalf("prepared review must keep the current surface stable until the plan is ready, got screen %v", preparing.screen)
	}
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	plannedModel, _ := preparing.Update(plannedMsg)
	planned := plannedModel.(Model)
	if planned.screen != screenSynthesisReview {
		t.Fatalf("complete approval should appear in one transition, got screen %v", planned.screen)
	}
	if planned.synthesisReviewPlan == nil || !planned.synthesisReviewPlan.ApprovalRequired {
		t.Fatalf("prepared review must expose the exact approval-required Core plan: %#v", planned.synthesisReviewPlan)
	}
	applying, applyCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if applyCmd == nil || !applying.synthesisReviewApplying {
		t.Fatal("the first Enter on a prepared review must apply the exact visible plan")
	}
	assertTestFilesEqual(t, synthesisPath, synthesisBefore, mixtapePath, mixtapeBefore, linerPath, linerBefore)
}

func TestSynthesisReviewPatchPreviewsWithoutWritingThenAppliesAtomically(t *testing.T) {
	m, synthesisPath, mixtapePath, linerPath := synthesisReviewProject(t)
	synthesisBefore := mustReadTestFile(t, synthesisPath)
	mixtapeBefore := mustReadTestFile(t, mixtapePath)
	linerBefore := mustReadTestFile(t, linerPath)
	revision := "# Revised synthesis\n\nUse the newly accepted evidence to compare the strongest examples, preserve the Curator's intent, explain material contradictions, and carry the complete reviewed reasoning into the next compile without clipping the final-token.\n"

	review, _ := m.startSynthesisReview()
	review.synthesisReviewChoice = synthesisReviewPatch
	review.synthesisReviewArea.SetValue(revision)
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	if plannedMsg.err != nil {
		t.Fatal(plannedMsg.err)
	}
	operation := plannedMsg.plan.Operations[0]
	if operation["disposition"] != "patch" || operation["content"] != revision || !plannedMsg.plan.ApprovalRequired {
		t.Fatalf("unexpected patch Change Set: %#v", plannedMsg.plan)
	}
	assertTestFilesEqual(t, synthesisPath, synthesisBefore, mixtapePath, mixtapeBefore, linerPath, linerBefore)

	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	planned.synthesisReviewPlanView.SetHeight(100)
	planned.synthesisReviewPlanView.SetWidth(56)
	planned.syncSynthesisReviewPlanView()
	preview := stripANSICodesForTest(planned.viewSynthesisReview())
	if !strings.Contains(preview, "Replacement synthesis") || !strings.Contains(preview, "# Revised synthesis") || !strings.Contains(preview, "final-token") {
		t.Fatalf("exact patch approval preview must include readable Core-returned content:\n%s", preview)
	}
	if strings.Contains(preview, `Operation payload: {"content":`) || strings.Contains(preview, `\n`) {
		t.Fatalf("semantic preview must not make the user review escaped artifact content as a JSON payload:\n%s", preview)
	}
	for _, internal := range []string{"Core Change Set details", "Operation payload", "Expected revision", "Change Set", "Hash"} {
		if strings.Contains(preview, internal) {
			t.Fatalf("human approval must not expose internal %q:\n%s", internal, preview)
		}
	}
	if !strings.Contains(preview, "Preview lines 1–") || !strings.Contains(preview, "complete") {
		t.Fatalf("approval preview must report its visible range and completion state:\n%s", preview)
	}
	assertViewLinesFit(t, planned.synthesisReviewPlanView.View(), 56)
	applying, applyCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	appliedMsg := applyCmd().(synthesisReviewAppliedMsg)
	if appliedMsg.err != nil {
		t.Fatal(appliedMsg.err)
	}
	if appliedMsg.receipt.SynthesisDisposition != "approved_patch" {
		t.Fatalf("unexpected patch receipt: %#v", appliedMsg.receipt)
	}
	completedModel, _ := applying.Update(appliedMsg)
	completed := completedModel.(Model)
	if completed.screen != screenSynthesisReview || !completed.synthesisReviewLoading || completed.synthesisReviewPlan == nil || !strings.Contains(completed.note, appliedMsg.receipt.ChangeSetID) {
		t.Fatalf("patch completion should keep the stable review surface with receipt evidence: screen=%v note=%q", completed.screen, completed.note)
	}
	if got := mustReadTestFile(t, synthesisPath); string(got) != revision {
		t.Fatalf("synthesis patch mismatch: %q", got)
	}
	assertTestFilesEqual(t, mixtapePath, mixtapeBefore, linerPath, linerBefore)
}

func TestSynthesisReviewPatchEnterStartsTheFirstEdit(t *testing.T) {
	m, _, _, _ := synthesisReviewProject(t)
	review, _ := m.startSynthesisReview()
	review.synthesisReviewChoice = synthesisReviewPatch

	before := stripANSICodesForTest(review.viewSynthesisReview())
	if !strings.Contains(before, "No local edit yet") || strings.Contains(before, "Proposed synthesis revision") {
		t.Fatalf("an untouched edit choice should show a concise invitation instead of duplicating the synthesis:\n%s", before)
	}
	if footer := stripANSICodesForTest(review.footerHelp()); !strings.Contains(footer, "enter edit synthesis") {
		t.Fatalf("the selected edit route must make Enter's behavior explicit: %q", footer)
	}

	editing, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if planCmd != nil || editing.synthesisReviewLoading || !editing.synthesisReviewEditing || editing.err != "" {
		t.Fatalf("Enter on Edit before approval should open the editor: loading=%v editing=%v err=%q cmd=%v", editing.synthesisReviewLoading, editing.synthesisReviewEditing, editing.err, planCmd)
	}
	finished, _ := editing.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: 'd', Mod: tea.ModCtrl}))
	reopened, retryCmd := finished.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if retryCmd != nil || !reopened.synthesisReviewEditing || !strings.Contains(reopened.note, "No changes yet") {
		t.Fatalf("finishing without an edit should return Enter to the editor, not create a patch: editing=%v note=%q cmd=%v", reopened.synthesisReviewEditing, reopened.note, retryCmd)
	}
}

func TestSynthesisReviewBackDiscardAndCompileGateAreSideEffectFree(t *testing.T) {
	m, synthesisPath, mixtapePath, linerPath := synthesisReviewProject(t)
	synthesisBefore := mustReadTestFile(t, synthesisPath)
	mixtapeBefore := mustReadTestFile(t, mixtapePath)
	linerBefore := mustReadTestFile(t, linerPath)

	blocked, cmd := m.startCompile()
	if cmd != nil || blocked.screen != screenProject || !strings.Contains(blocked.err, "Review Synthesis") {
		t.Fatalf("compile must fail closed at the synthesis gate: screen=%v err=%q cmd=%v", blocked.screen, blocked.err, cmd)
	}
	review, _ := m.startSynthesisReview()
	review.synthesisReviewChoice = synthesisReviewPatch
	review.synthesisReviewArea.SetValue("# Local proposal\n")
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	discarded, discardCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if discardCmd != nil || discarded.synthesisReviewPlan != nil || discarded.screen != screenSynthesisReview {
		t.Fatalf("discard should remove only the local plan: %#v", discarded)
	}
	closed, closeCmd := discarded.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if closeCmd != nil || closed.screen != screenProject {
		t.Fatalf("back should return to Project Flow without writes: %#v", closed)
	}
	assertTestFilesEqual(t, synthesisPath, synthesisBefore, mixtapePath, mixtapeBefore, linerPath, linerBefore)
}

func TestSynthesisReviewAtomicApplyDefersQuitAndReplaysAmbiguousReceipt(t *testing.T) {
	m, _, _, _ := synthesisReviewProject(t)
	review, _ := m.startSynthesisReview()
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	applying, applyCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if !applying.synthesisReviewApplying {
		t.Fatal("approved semantic apply should enter an uninterruptible Core phase")
	}
	for _, binding := range applying.helpForScreen().FullHelp()[0] {
		if binding.Help().Key == "ctrl+c" || binding.Help().Key == "esc" {
			t.Fatalf("protected apply help must not advertise blocked navigation: %#v", applying.helpForScreen().FullHelp())
		}
	}
	deferred, quitCmd := applying.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if quitCmd != nil || !deferred.synthesisReviewApplying || !strings.Contains(deferred.note, "cannot be interrupted") {
		t.Fatalf("Ctrl+C must defer until Core resolves apply: %#v cmd=%v", deferred, quitCmd)
	}
	committed := applyCmd().(synthesisReviewAppliedMsg)
	if committed.err != nil {
		t.Fatal(committed.err)
	}
	ambiguousModel, _ := applying.Update(synthesisReviewAppliedMsg{err: os.ErrDeadlineExceeded})
	ambiguous := ambiguousModel.(Model)
	if ambiguous.synthesisReviewPlan == nil || !ambiguous.synthesisReviewReconcile || ambiguous.synthesisReviewApplying {
		t.Fatalf("ambiguous completion must retain the exact plan for receipt replay: %#v", ambiguous)
	}
	refreshed, err := m.runner.InspectMaintenanceProject(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	refreshedModel, _ := ambiguous.Update(projectSnapshotMsg{path: m.currentPath, snapshot: refreshed})
	reconciling := refreshedModel.(Model)
	if !strings.Contains(reconciling.note, "receipt") || reconciling.synthesisReviewPlan == nil {
		t.Fatalf("Snapshot refresh must preserve the replay route: %#v", reconciling)
	}
	locked, abandonCmd := reconciling.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if abandonCmd != nil || locked.synthesisReviewPlan == nil || locked.screen != screenSynthesisReview || !strings.Contains(locked.note, "reconciliation") {
		t.Fatalf("ambiguous receipt reconciliation cannot discard the exact plan: %#v cmd=%v", locked, abandonCmd)
	}
	locked, quitCmd = locked.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if quitCmd != nil || locked.synthesisReviewPlan == nil {
		t.Fatalf("ambiguous receipt reconciliation cannot quit before replay: %#v cmd=%v", locked, quitCmd)
	}
	if locked.supportsHomeShortcut() {
		t.Fatal("Review Synthesis must not expose a Home escape while exact-plan reconciliation is pending")
	}
	for _, binding := range locked.helpForScreen().FullHelp()[0] {
		if binding.Help().Key == "ctrl+c" || binding.Help().Key == "esc" {
			t.Fatalf("reconciliation help must not advertise blocked navigation: %#v", locked.helpForScreen().FullHelp())
		}
	}
	replaying, replayCmd := locked.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	replayed := replayCmd().(synthesisReviewAppliedMsg)
	if replayed.err != nil || replayed.receipt.ReceiptID != committed.receipt.ReceiptID || !replayed.receipt.Replayed {
		t.Fatalf("exact-plan replay should recover the durable receipt: first=%#v replay=%#v", committed.receipt, replayed)
	}
	if !replaying.synthesisReviewApplying {
		t.Fatal("receipt replay should use the same protected apply phase")
	}
}

func TestCompileFailsClosedWhileProjectSnapshotReloads(t *testing.T) {
	m := Model{
		screen:                   screenProject,
		currentPath:              t.TempDir(),
		projectSnapshotPath:      "",
		projectSnapshotAttempted: true,
		projectSnapshotLoading:   true,
	}
	blocked, cmd := m.startCompile()
	if cmd != nil || blocked.compiling || blocked.screen != screenProject || !strings.Contains(blocked.err, "trustworthy Project Snapshot") {
		t.Fatalf("compile must fail closed during Snapshot reload: %#v cmd=%v", blocked, cmd)
	}
}

func TestSynthesisReviewRemainsUsableInNarrowShortTerminal(t *testing.T) {
	m, _, _, _ := synthesisReviewProject(t)
	m.help = help.New()
	m.width = 64
	m.height = 22
	m.help.SetWidth(60)
	m.synthesisReviewArea.SetWidth(56)
	m.synthesisReviewCurrent.SetWidth(56)
	m.synthesisReviewCurrent.SetHeight(3)
	review, _ := m.startSynthesisReview()
	if !review.supportsHomeShortcut() {
		t.Fatal("ordinary write-free Review Synthesis should preserve global Home navigation")
	}
	view := review.viewSynthesisReview()
	plain := stripANSICodesForTest(view)
	for _, expected := range []string{"Review Synthesis", "Synthesis awaiting approval", "Approve unchanged", "Edit before approval", "MIXTAPE.md"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("narrow synthesis review missing %q:\n%s", expected, plain)
		}
	}
	assertViewLinesFit(t, view, 60)
	footer := review.footerHelp()
	if !strings.Contains(stripANSICodesForTest(footer), "enter preview approval") {
		t.Fatalf("narrow footer lost the primary action: %q", footer)
	}
	assertViewLinesFit(t, footer, 60)
}

func TestSynthesisReviewWrapsCurrentContentAndShowsDocumentPosition(t *testing.T) {
	m, synthesisPath, _, _ := synthesisReviewProject(t)
	content := "# Current synthesis\n\n" + strings.Repeat("This source-grounded synthesis must remain readable inside the terminal instead of continuing beyond the right edge. ", 6) + "\n\n" + strings.Repeat("A second paragraph makes the viewport scroll far enough to prove page navigation and a visible completion boundary. ", 5) + "\n"
	if err := os.WriteFile(synthesisPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m.width = 64
	m.height = 22
	m.synthesisReviewArea.SetWidth(56)
	m.synthesisReviewCurrent.SetWidth(56)
	m.synthesisReviewCurrent.SetHeight(3)
	review, _ := m.startSynthesisReview()

	if review.synthesisReviewCurrentText != content || review.synthesisReviewArea.Value() != content {
		t.Fatal("visual wrapping must not change the canonical synthesis submitted to Core")
	}
	assertViewLinesFit(t, review.synthesisReviewCurrent.View(), 56)
	plain := stripANSICodesForTest(review.viewSynthesisReview())
	for _, expected := range []string{"Lines 1–", "of ", "more below", "accepted Sources", "without", "rewriting synthesis.md"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("responsive synthesis review missing %q:\n%s", expected, plain)
		}
	}
	if review.synthesisReviewCurrent.AtBottom() {
		t.Fatal("long wrapped synthesis should not appear complete at the top of the viewport")
	}

	lineScrolled, _ := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if got := lineScrolled.synthesisReviewCurrent.YOffset() - review.synthesisReviewCurrent.YOffset(); got != 1 {
		t.Fatalf("Down should move one rendered line, got %d", got)
	}
	pageScrolled, _ := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	if got := pageScrolled.synthesisReviewCurrent.YOffset() - review.synthesisReviewCurrent.YOffset(); got <= 1 {
		t.Fatalf("Page Down should move by a viewport page, got %d", got)
	}
}

func TestSynthesisReviewPlannedStateScrollsExactPreviewAboveShortFooter(t *testing.T) {
	m, _, _, _ := synthesisReviewProject(t)
	m.help = help.New()
	m.width = 64
	m.height = 18
	m.help.SetWidth(60)
	m.synthesisReviewArea.SetWidth(56)
	m.synthesisReviewCurrent.SetWidth(56)
	m.synthesisReviewPlanView.SetWidth(56)
	m.synthesisReviewPlanView.SetHeight(4)
	review, _ := m.startSynthesisReview()
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	top := stripANSICodesForTest(planned.View().Content)
	for _, expected := range []string{"Sources approved", "Confirm Synthesis approval", "enter record approval", "pgup/pgdn review preview", "Preview lines 1–4", "more below"} {
		if !strings.Contains(top, expected) {
			t.Fatalf("short planned view missing %q:\n%s", expected, top)
		}
	}
	if planned.synthesisReviewPlanView.AtBottom() {
		t.Fatal("short planned preview should have scrollable Core details")
	}
	pageScrolled, _ := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	if got := pageScrolled.synthesisReviewPlanView.YOffset() - planned.synthesisReviewPlanView.YOffset(); got <= 1 {
		t.Fatalf("Page Down should move the approval preview by a viewport page, got %d", got)
	}
	for !planned.synthesisReviewPlanView.AtBottom() {
		planned, _ = planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	bottom := stripANSICodesForTest(planned.View().Content)
	if !strings.Contains(bottom, "Compile MIXTAPE.md") || !strings.Contains(bottom, "enter record approval") {
		t.Fatalf("scrolling must reveal the bottom of the exact Core preview while keeping approval visible:\n%s", bottom)
	}
}

func TestSynthesisReviewShortApprovalPreviewFitsContentWithoutTerminalHeightPadding(t *testing.T) {
	m, _, _, _ := synthesisReviewProject(t)
	m.width = 110
	m.height = 70
	m.synthesisReviewPlanView.SetWidth(100)
	m.synthesisReviewPlanView.SetHeight(56)
	review, _ := m.startSynthesisReview()
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)

	if total := planned.synthesisReviewPlanView.TotalLineCount(); total >= 56 {
		t.Fatalf("fixture must produce a short approval preview, got %d lines", total)
	} else if got := planned.synthesisReviewPlanView.Height(); got != total {
		t.Fatalf("short approval preview should fit its %d content lines instead of padding to terminal height; got height %d", total, got)
	}
	plain := stripANSICodesForTest(planned.viewSynthesisReview())
	if !strings.Contains(plain, "Confirm Synthesis approval") || !strings.Contains(plain, "Sources approved") || strings.Contains(plain, "Review required.") {
		t.Fatalf("expected approval should read as a neutral checkpoint, not an error:\n%s", plain)
	}
	for _, expected := range []string{"Synthesis text", "No changes", "This action", "Record curator approval", "After approval", "Compile MIXTAPE.md"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("still-current approval missing %q:\n%s", expected, plain)
		}
	}
	for _, internal := range []string{"Core Change Set details", "Operation payload", "Expected revision", "Change Set", "Hash"} {
		if strings.Contains(plain, internal) {
			t.Fatalf("still-current approval must not expose internal %q:\n%s", internal, plain)
		}
	}
}

func TestSynthesisReviewContentWrapPreservesWhitespaceAndLongTokens(t *testing.T) {
	line := "    indented  Markdown with a hard break  and-supercalifragilisticexpialidocious-final-token  "
	segments := wrapSynthesisReviewLine(line, 18)
	if got := strings.Join(segments, ""); got != line {
		t.Fatalf("visual wrapping changed exact Core content\nwant: %q\n got: %q", line, got)
	}
	for _, segment := range segments {
		if width := lipgloss.Width(segment); width > 18 {
			t.Fatalf("wrapped segment is still clipped: width=%d segment=%q", width, segment)
		}
	}
}

func TestOperatingLayerReviewStillCurrentUsesReceiptAndRefreshesProjectFlow(t *testing.T) {
	m, linerPath, skillPath := operatingLayerReviewProject(t)
	linerBefore := mustReadTestFile(t, linerPath)
	skillBefore := mustReadTestFile(t, skillPath)

	review, cmd := m.startOperatingLayerReview()
	if cmd != nil || review.screen != screenSynthesisReview || review.synthesisReviewKind != semanticReviewOperatingLayer {
		t.Fatalf("expected write-free Operating Layer review surface, screen=%v kind=%v cmd=%v err=%q", review.screen, review.synthesisReviewKind, cmd, review.err)
	}
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if planCmd == nil || !planning.synthesisReviewLoading {
		t.Fatal("still-current Operating Layer selection should request a Core preview")
	}
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	if plannedMsg.err != nil {
		t.Fatal(plannedMsg.err)
	}
	operation := plannedMsg.plan.Operations[0]
	if operation["type"] != "operating_layer.review" || operation["disposition"] != "still_current" || !plannedMsg.plan.ApprovalRequired {
		t.Fatalf("unexpected Operating Layer Change Set: %#v", plannedMsg.plan)
	}
	assertTestFilesEqual(t, linerPath, linerBefore, skillPath, skillBefore)

	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	applying, applyCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if applyCmd == nil || !applying.synthesisReviewApplying {
		t.Fatal("approved Operating Layer preview should enter protected Core apply")
	}
	appliedMsg := applyCmd().(synthesisReviewAppliedMsg)
	if appliedMsg.err != nil {
		t.Fatal(appliedMsg.err)
	}
	if len(appliedMsg.receipt.Operations) != 1 || appliedMsg.receipt.Operations[0]["type"] != "operating_layer.review" {
		t.Fatalf("receipt did not prove the Operating Layer disposition: %#v", appliedMsg.receipt)
	}
	completedModel, _ := applying.Update(appliedMsg)
	completed := completedModel.(Model)
	if completed.screen != screenProject || !completed.projectSnapshotLoading || !strings.Contains(completed.note, "Receipt:") {
		t.Fatalf("completion should return to Project Flow with receipt evidence: %#v", completed)
	}
	assertTestFilesEqual(t, linerPath, linerBefore, skillPath, skillBefore)

	refreshed, err := m.runner.InspectMaintenanceProject(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedModel, _ := completed.Update(projectSnapshotMsg{path: m.currentPath, snapshot: refreshed})
	updated := updatedModel.(Model)
	if updated.projectNextKind() != projectNextOpenLiner || refreshed.Lifecycle.Stale {
		t.Fatalf("receipt refresh should return to the current completed Project: %#v", refreshed.Lifecycle)
	}
}

func TestOperatingLayerReviewPatchPreviewsAndAppliesBothArtifacts(t *testing.T) {
	m, linerPath, skillPath := operatingLayerReviewProject(t)
	linerRevision := "# Reviewed Operating Layer\n\nUse the refreshed corpus.\n"
	skillRevision := "---\nname: synthesis-review\ndescription: Use the refreshed Project.\n---\n# Reviewed Project Skill\n"

	review, _ := m.startOperatingLayerReview()
	review.synthesisReviewChoice = synthesisReviewPatch
	review.synthesisReviewArea.SetValue(linerRevision)
	review.operatingLayerReviewSkillArea.SetValue(skillRevision)
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	if plannedMsg.err != nil {
		t.Fatal(plannedMsg.err)
	}
	operation := plannedMsg.plan.Operations[0]
	if operation["type"] != "operating_layer.review" || operation["disposition"] != "patch" || operation["liner_content"] != linerRevision || operation["skill_content"] != skillRevision {
		t.Fatalf("unexpected dual-artifact patch Change Set: %#v", plannedMsg.plan)
	}

	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	planned.synthesisReviewPlanView.SetHeight(100)
	planned.synthesisReviewPlanView.SetWidth(64)
	planned.syncSynthesisReviewPlanView()
	preview := stripANSICodesForTest(planned.viewSynthesisReview())
	for _, expected := range []string{"Review Operating Layer", "Replacement LINER.md", "# Reviewed Operating Layer", "Replacement Project Skill · SKILL.md", "# Reviewed Project Skill"} {
		if !strings.Contains(preview, expected) {
			t.Fatalf("exact Operating Layer preview missing %q:\n%s", expected, preview)
		}
	}
	assertViewLinesFit(t, planned.synthesisReviewPlanView.View(), 64)

	applying, applyCmd := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	appliedMsg := applyCmd().(synthesisReviewAppliedMsg)
	if appliedMsg.err != nil {
		t.Fatal(appliedMsg.err)
	}
	completedModel, _ := applying.Update(appliedMsg)
	completed := completedModel.(Model)
	if completed.screen != screenProject || !strings.Contains(completed.note, appliedMsg.receipt.ChangeSetID) {
		t.Fatalf("patch completion should show receipt evidence in Project Flow: %#v", completed)
	}
	if got := string(mustReadTestFile(t, linerPath)); got != linerRevision {
		t.Fatalf("LINER.md patch mismatch: %q", got)
	}
	if got := string(mustReadTestFile(t, skillPath)); got != skillRevision {
		t.Fatalf("Project Skill patch mismatch: %q", got)
	}
}

func TestOperatingLayerReviewTabSwitchesOnlyTheFocusedArtifactInput(t *testing.T) {
	m, _, _ := operatingLayerReviewProject(t)
	review, _ := m.startOperatingLayerReview()
	review.synthesisReviewChoice = synthesisReviewPatch
	linerBefore := review.synthesisReviewArea.Value()
	skillBefore := review.operatingLayerReviewSkillArea.Value()

	switched, _ := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if switched.operatingLayerReviewArtifact != operatingLayerReviewSkill {
		t.Fatalf("Tab should switch to the declared Project Skill: %#v", switched)
	}
	if !hasHelpDesc(switched.helpForScreen().ShortHelp(), "artifact") {
		t.Fatalf("Operating Layer review help should advertise artifact switching: %#v", switched.helpForScreen().ShortHelp())
	}
	editing, _ := switched.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	updatedModel, _ := editing.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	updated := updatedModel.(Model)
	if updated.synthesisReviewArea.Value() != linerBefore || updated.operatingLayerReviewSkillArea.Value() == skillBefore {
		t.Fatalf("typing should update only the focused Project Skill area: liner=%q skill=%q", updated.synthesisReviewArea.Value(), updated.operatingLayerReviewSkillArea.Value())
	}
	skillAfter := updated.operatingLayerReviewSkillArea.Value()

	linerModel, _ := updated.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	linerEditing := linerModel.(Model)
	if linerEditing.operatingLayerReviewArtifact != operatingLayerReviewLINER || !linerEditing.synthesisReviewEditing {
		t.Fatalf("Tab while editing should preserve editing and switch to LINER.md: %#v", linerEditing)
	}
	linerUpdatedModel, _ := linerEditing.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	linerUpdated := linerUpdatedModel.(Model)
	if linerUpdated.synthesisReviewArea.Value() == linerBefore || linerUpdated.operatingLayerReviewSkillArea.Value() != skillAfter {
		t.Fatalf("typing should update only the focused LINER.md area: liner=%q skill=%q", linerUpdated.synthesisReviewArea.Value(), linerUpdated.operatingLayerReviewSkillArea.Value())
	}
}

func TestOperatingLayerReviewEditorOwnsPrintableKeysFromFirstKeystroke(t *testing.T) {
	m, _, _ := operatingLayerReviewProject(t)
	review, _ := m.startOperatingLayerReview()
	review.synthesisReviewChoice = synthesisReviewPatch
	before := review.synthesisReviewArea.Value()

	editingModel, _ := review.Update(tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	editing := editingModel.(Model)
	if !editing.synthesisReviewEditing || editing.screen != screenSynthesisReview {
		t.Fatalf("e should enter semantic-review editing: %#v", editing)
	}
	if got := editing.synthesisReviewArea.Value(); got != before {
		t.Fatalf("the edit command must not be inserted into the artifact: got %q want %q", got, before)
	}

	for _, typed := range []string{"h", "?"} {
		updatedModel, _ := editing.Update(tea.KeyPressMsg(tea.Key{Code: rune(typed[0]), Text: typed}))
		updated := updatedModel.(Model)
		if updated.screen != screenSynthesisReview || !updated.synthesisReviewEditing {
			t.Fatalf("printable %q escaped the editor: screen=%v editing=%v", typed, updated.screen, updated.synthesisReviewEditing)
		}
		if !strings.HasSuffix(updated.synthesisReviewArea.Value(), typed) {
			t.Fatalf("printable %q should reach the artifact editor, got %q", typed, updated.synthesisReviewArea.Value())
		}
		editing = updated
	}
}

func TestOperatingLayerReviewUsesSpecificScreenLabel(t *testing.T) {
	m, _, _ := operatingLayerReviewProject(t)
	review, _ := m.startOperatingLayerReview()
	if got := review.screenLabel(); got != "review operating layer" {
		t.Fatalf("unexpected Operating Layer review label: %q", got)
	}
}

func TestOperatingLayerReviewAmbiguousApplyRetainsExactPlanForReceiptRecovery(t *testing.T) {
	m, _, _ := operatingLayerReviewProject(t)
	review, _ := m.startOperatingLayerReview()
	planning, planCmd := review.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	plannedMsg := planCmd().(synthesisReviewPlannedMsg)
	plannedModel, _ := planning.Update(plannedMsg)
	planned := plannedModel.(Model)
	applying, _ := planned.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	ambiguousModel, _ := applying.Update(synthesisReviewAppliedMsg{err: os.ErrDeadlineExceeded})
	ambiguous := ambiguousModel.(Model)
	if ambiguous.synthesisReviewPlan == nil || !ambiguous.synthesisReviewReconcile || ambiguous.synthesisReviewApplying {
		t.Fatalf("ambiguous Operating Layer completion must retain the exact plan: %#v", ambiguous)
	}
	refreshed, err := m.runner.InspectMaintenanceProject(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	reconcilingModel, _ := ambiguous.Update(projectSnapshotMsg{path: m.currentPath, snapshot: refreshed})
	reconciling := reconcilingModel.(Model)
	if reconciling.synthesisReviewPlan == nil || !strings.Contains(reconciling.note, "receipt") {
		t.Fatalf("Snapshot reconciliation must preserve Operating Layer receipt replay: %#v", reconciling)
	}
	locked, cmd := reconciling.handleSynthesisReviewKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd != nil || locked.synthesisReviewPlan == nil || locked.screen != screenSynthesisReview {
		t.Fatalf("receipt recovery must not expose an escape that discards the exact plan: %#v cmd=%v", locked, cmd)
	}
}

func synthesisReviewProject(t *testing.T) (Model, string, string, string) {
	t.Helper()
	runner := testCoreRunner(t)
	project := filepath.Join(t.TempDir(), "synthesis-review")
	if err := runner.InitProject(project); err != nil {
		t.Fatal(err)
	}
	paths := tape.ProjectAt(project)
	synthesisPath := filepath.Join(paths.Path, "synthesis.md")
	mixtapePath := filepath.Join(paths.Path, "MIXTAPE.md")
	linerPath := filepath.Join(project, "LINER.md")
	for path, content := range map[string]string{
		synthesisPath: "# Current synthesis\n\nKeep the strongest evidence.\n",
		mixtapePath:   "# Published MIXTAPE\n",
		linerPath:     "# Verified Operating Layer\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	metadataPath := filepath.Join(project, "liner.yaml")
	metadataBytes := mustReadTestFile(t, metadataPath)
	var metadata map[string]any
	if err := yaml.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["status"] = map[string]any{
		"milestone": "project_complete",
		"stale":     false,
		"updated":   "2026-01-01T00:00:00Z",
		"corpus": map[string]any{
			"state": "ready", "evidence": "mixtape/MIXTAPE.md",
		},
		"operating_layer": map[string]any{
			"state": "ready", "evidence": "LINER.md",
		},
	}
	metadata["project_skill"] = map[string]any{
		"status": "active", "name": "synthesis-review", "path": "SKILL.md",
	}
	encoded, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "SKILL.md"), []byte("---\nname: synthesis-review\ndescription: Test Project Skill.\n---\n# Synthesis Review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	addPlan, err := runner.PlanMaintenance(project, core.SourceOperation("source.add", "", map[string]any{
		"type": "web", "url": "https://example.test/refresh", "note": "Primary", "priority": "required",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyMaintenance(project, addPlan, addPlan.ApprovalRequired); err != nil {
		t.Fatal(err)
	}
	snapshot, err := runner.InspectMaintenanceProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Lifecycle.Refresh == nil || snapshot.Lifecycle.Refresh.Synthesis.State != "review_required" {
		t.Fatalf("fixture did not reach synthesis review gate: %#v", snapshot.Lifecycle)
	}
	current, err := tape.ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	return Model{
		runner:                   runner,
		screen:                   screenProject,
		width:                    110,
		height:                   36,
		currentPath:              project,
		currentTape:              current,
		projectSnapshotPath:      project,
		projectSnapshot:          &snapshot,
		projectSnapshotAttempted: true,
		synthesisReviewCurrent:   newSynthesisReviewViewport(100, 8),
		synthesisReviewPlanView:  newSynthesisReviewViewport(100, 12),
		synthesisReviewArea:      newSynthesisReviewArea(100),
	}, synthesisPath, mixtapePath, linerPath
}

func operatingLayerReviewProject(t *testing.T) (Model, string, string) {
	t.Helper()
	m, _, _, linerPath := synthesisReviewProject(t)
	metadataPath := filepath.Join(m.currentPath, "liner.yaml")
	metadataBytes := mustReadTestFile(t, metadataPath)
	var metadata map[string]any
	if err := yaml.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	status, _ := metadata["status"].(map[string]any)
	refresh, _ := status["refresh"].(map[string]any)
	triggerChangeSetID := refresh["trigger_change_set_id"]
	metadata["status"] = map[string]any{
		"milestone": "project_complete",
		"stale":     true,
		"updated":   "2026-01-02T00:00:00Z",
		"corpus": map[string]any{
			"state": "ready", "evidence": "mixtape/MIXTAPE.md",
		},
		"operating_layer": map[string]any{
			"state": "stale", "evidence": "LINER.md", "last_verified_state": "ready",
		},
		"refresh": map[string]any{
			"state":                 "required",
			"trigger_change_set_id": triggerChangeSetID,
			"affected_artifacts":    []string{"mixtape/synthesis.md", "mixtape/MIXTAPE.md", "LINER.md", "SKILL.md"},
			"remaining_artifacts":   []string{"LINER.md", "SKILL.md"},
			"synthesis":             map[string]any{"state": "approved", "disposition": "still_current"},
			"corpus":                map[string]any{"state": "current"},
			"operating_layer":       map[string]any{"state": "review_required"},
		},
	}
	encoded, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := m.runner.InspectMaintenanceProject(m.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Lifecycle.Refresh == nil || snapshot.Lifecycle.Refresh.OperatingLayer.State != "review_required" {
		t.Fatalf("fixture did not reach Operating Layer review gate: %#v", snapshot.Lifecycle)
	}
	m.projectSnapshot = &snapshot
	m.projectSnapshotPath = m.currentPath
	m.operatingLayerReviewSkillArea = newOperatingLayerReviewSkillArea(100)
	return m, linerPath, filepath.Join(m.currentPath, "SKILL.md")
}

func mustReadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertTestFilesEqual(t *testing.T, pairs ...any) {
	t.Helper()
	for index := 0; index < len(pairs); index += 2 {
		path := pairs[index].(string)
		want := pairs[index+1].([]byte)
		if got := mustReadTestFile(t, path); string(got) != string(want) {
			t.Fatalf("%s changed unexpectedly\nwant: %q\n got: %q", path, want, got)
		}
	}
}
