package app

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func longSemanticReviewText(lines int) string {
	out := make([]string, 0, lines)
	for index := range lines {
		out = append(out, fmt.Sprintf("%d. Rule %02d keeps the person in charge of the decision it names.", index+1, index+1))
	}
	return strings.Join(out, "\n")
}

func typeIntoModel(t *testing.T, m Model, keys ...tea.KeyPressMsg) Model {
	t.Helper()
	for _, keyMsg := range keys {
		updated, _ := m.Update(keyMsg)
		m = updated.(Model)
	}
	return m
}

func textKeys(text string) []tea.KeyPressMsg {
	keys := make([]tea.KeyPressMsg, 0, len(text))
	for _, r := range text {
		keys = append(keys, tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
	return keys
}

func TestSemanticReviewEditorAddsLinesToLongTextAndKeepsCursorVisibleAt100x30(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Model, string)
	}{
		{
			name: "synthesis",
			setup: func(m *Model, text string) {
				m.synthesisReviewKind = semanticReviewSynthesis
				m.synthesisReviewCurrentText = text
				m.synthesisReviewArea.SetValue(text)
			},
		},
		{
			name: "project skill",
			setup: func(m *Model, text string) {
				m.synthesisReviewKind = semanticReviewOperatingLayer
				m.operatingLayerReviewArtifact = operatingLayerReviewSkill
				m.operatingLayerReviewSkillPath = "SKILL.md"
				m.operatingLayerReviewSkillCurrentText = text
				m.operatingLayerReviewSkillArea.SetValue(text)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := longSemanticReviewText(51)
			m := Model{
				screen:                        screenSynthesisReview,
				width:                         100,
				height:                        30,
				help:                          help.New(),
				currentTape:                   tape.Tape{Title: "Synthetic"},
				synthesisReviewChoice:         synthesisReviewPatch,
				synthesisReviewCurrent:        newSynthesisReviewViewport(92, 10),
				synthesisReviewArea:           newSynthesisReviewArea(92),
				operatingLayerReviewSkillArea: newOperatingLayerReviewSkillArea(92),
			}
			m.help.SetWidth(96)
			tc.setup(&m, text)
			m.beginSemanticReviewEditing(true)

			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			m = typeIntoModel(t, m, textKeys("6. A new rule typed at the end.")...)
			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			m = typeIntoModel(t, m, textKeys("7. And another one.")...)

			if strings.Contains(m.note, "No changes yet") {
				t.Fatalf("the note should stop saying there are no changes once text changes, got %q", m.note)
			}
			area := m.activeSemanticReviewArea()
			if got := area.LineCount(); got != 53 {
				t.Fatalf("Enter should add lines past the visible height: got %d lines, want 53", got)
			}
			if !strings.HasSuffix(area.Value(), "\n6. A new rule typed at the end.\n7. And another one.") {
				t.Fatalf("typed lines should land at the cursor, got tail %q", area.Value()[max(0, len(area.Value())-80):])
			}

			rendered := m.View().Content
			if got := lipgloss.Height(rendered); got > 30 {
				t.Fatalf("editing view should fit 100 x 30, got %d lines", got)
			}
			plain := stripANSICodesForTest(rendered)
			for _, expected := range []string{"7. And another one.", "6. A new rule typed at the end.", "editing · line 53 of 53", "ctrl+d finish revision"} {
				if !strings.Contains(plain, expected) {
					t.Fatalf("100 x 30 editing view should show %q:\n%s", expected, plain)
				}
			}
			if strings.Contains(plain, "more content hidden") {
				t.Fatalf("the editor should not be cut off by the frame:\n%s", plain)
			}

			// Move to the top: the editor scrolls with the cursor.
			for range 60 {
				m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
			}
			plain = stripANSICodesForTest(m.View().Content)
			if !strings.Contains(plain, "1. Rule 01") || !strings.Contains(plain, "editing · line 1 of 53") {
				t.Fatalf("moving up should scroll the editor to the cursor:\n%s", plain)
			}
		})
	}
}

func TestSemanticReviewKeepsApproveActionOnScreenAfterEditingAt100x30(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Model, string)
	}{
		{
			name: "synthesis",
			setup: func(m *Model, text string) {
				m.synthesisReviewKind = semanticReviewSynthesis
				m.synthesisReviewCurrentText = text
				m.synthesisReviewArea.SetValue(text)
			},
		},
		{
			name: "project skill",
			setup: func(m *Model, text string) {
				m.synthesisReviewKind = semanticReviewOperatingLayer
				m.operatingLayerReviewArtifact = operatingLayerReviewSkill
				m.operatingLayerReviewSkillPath = "SKILL.md"
				m.synthesisReviewCurrentText = "# LINER\n"
				m.synthesisReviewArea.SetValue("# LINER\n")
				m.operatingLayerReviewSkillCurrentText = text
				m.operatingLayerReviewSkillArea.SetValue(text)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := longSemanticReviewText(51)
			m := Model{
				screen:                        screenSynthesisReview,
				width:                         100,
				height:                        30,
				help:                          help.New(),
				currentTape:                   tape.Tape{Title: "Synthetic"},
				synthesisReviewCurrent:        newSynthesisReviewViewport(92, semanticReviewCurrentRows(30)),
				synthesisReviewPlanView:       newSynthesisReviewViewport(92, semanticReviewPlanRows(30)),
				synthesisReviewArea:           newSynthesisReviewArea(92),
				operatingLayerReviewSkillArea: newOperatingLayerReviewSkillArea(92),
			}
			m.help.SetWidth(96)
			tc.setup(&m, text)
			m.syncSemanticReviewCurrent(true)

			assertFits := func(stage string, m Model, expected ...string) string {
				t.Helper()
				rendered := m.View().Content
				if got := lipgloss.Height(rendered); got > 30 {
					t.Fatalf("%s: review should fit 100 x 30, got %d lines", stage, got)
				}
				plain := stripANSICodesForTest(rendered)
				if strings.Contains(plain, "more content hidden") {
					t.Fatalf("%s: the review should not be cut off:\n%s", stage, plain)
				}
				flattened := strings.Join(strings.Fields(plain), " ")
				for _, want := range expected {
					if !strings.Contains(flattened, want) {
						t.Fatalf("%s: 100 x 30 review missing %q:\n%s", stage, want, plain)
					}
				}
				return plain
			}

			// Before an edit, both decisions and their keys are on screen.
			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
			assertFits("before edit", m, "● Approve unchanged", "○ Edit before approval", "enter preview approval")

			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
			assertFits("edit chosen, no edit yet", m, "● Edit before approval", "Press Enter or e to edit")

			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			m = typeIntoModel(t, m, textKeys("6. A new rule typed at the end.")...)
			m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: 'd', Mod: tea.ModCtrl}))
			if m.synthesisReviewEditing {
				t.Fatal("ctrl+d should finish editing")
			}

			plain := assertFits("after ctrl+d", m,
				"● Edit before approval",
				"52 lines, was 51: 1 line added, none removed.",
				"> Press Enter to preview the exact Core Change Set, then Enter again to approve it.",
				"enter preview Change Set",
			)
			if strings.Contains(plain, "Rule 51") && !strings.Contains(plain, "Lines") {
				t.Fatalf("the current text should stay in its viewport:\n%s", plain)
			}

			// The whole current text stays reachable in the smaller viewport.
			for range 60 {
				m = typeIntoModel(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
			}
			assertFits("scrolled to end", m, "51. Rule 51", "end of synthesis; more above", "> Press Enter to preview")

			// Enter is usable: it asks Core for the write-free preview.
			updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			next := updated.(Model)
			if cmd == nil || !next.synthesisReviewLoading || next.err != "" {
				t.Fatalf("enter after editing should request the Core preview: loading=%t err=%q", next.synthesisReviewLoading, next.err)
			}

			// Core's preview uses the rows the terminal has, even after a
			// shorter plan was shown first.
			for _, operation := range []map[string]any{
				{"type": "synthesis.review", "disposition": "still_current"},
				{"type": "synthesis.review", "disposition": "patch", "content": next.activeSemanticReviewArea().Value()},
			} {
				updated, _ = next.Update(synthesisReviewPlannedMsg{plan: core.ProjectChangeSet{Operations: []map[string]any{operation}}})
				next = updated.(Model)
			}
			if got := next.synthesisReviewPlanView.VisibleLineCount(); got != semanticReviewPlanRows(30) {
				t.Fatalf("the replacement preview should fill its %d rows, got %d", semanticReviewPlanRows(30), got)
			}
			assertFits("core preview", next, "Press Enter to record this approval", "enter record approval")
		})
	}
}
