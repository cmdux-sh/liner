package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

const (
	methodologyVisibleLogLines = 3
	methodologyQuietFrameDelay = 120
)

func (m Model) viewResearch() string {
	width := styles.ClampWidth(m.width - 4)
	phaseTable := m.methodologyPhaseTable(width)
	bar := newTaskProgressBar(taskProgressWidth(m.width)).ViewAs(m.researchPercent())
	logView := m.methodologyLogViewport(width, methodologyLogHeight(m.height))
	parts := []string{
		m.renderLoadingTitle("Build Corpus", !m.researchDone),
		styles.Subtitle.Render(m.currentTape.Title),
		"",
		styles.ReportSection.Render("Progress"),
		bar + "  " + styles.Subtitle.Render(fmt.Sprintf("%d/%d phases", min(m.researchStep, len(methodologyPhaseOrder)), len(methodologyPhaseOrder))),
		"",
		styles.ReportSection.Render("Phases"),
		phaseTable.View(),
		"",
		styles.ReportSection.Render("Log"),
		logView.View(),
	}
	if quiet := m.methodologyQuietStatus(); quiet != "" {
		parts = append(parts, quiet)
	}
	if cue := m.methodologyCue(); cue != "" {
		parts = append(parts, "", cue)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func configureMethodologyLogViewport(log *viewport.Model, width int, height int) {
	log.SetWidth(max(1, width))
	log.SetHeight(max(1, height))
	log.SoftWrap = false
	log.FillHeight = true
	log.Style = styles.MutedText
}

func (m *Model) syncMethodologyLog(forceBottom bool) {
	configureMethodologyLogViewport(&m.researchLog, styles.ClampWidth(m.width-4), methodologyLogHeight(m.height))
	wasAtBottom := m.researchLog.AtBottom()
	m.researchLog.SetContent(m.methodologyLogContent(styles.ClampWidth(m.width - 4)))
	if forceBottom || wasAtBottom {
		m.researchLog.GotoBottom()
	}
}

func (m Model) methodologyLogViewport(width int, height int) viewport.Model {
	log := m.researchLog
	configureMethodologyLogViewport(&log, width, height)
	if log.GetContent() == "" && len(m.researchLines) > 0 {
		log.SetContent(m.methodologyLogContent(width))
		log.GotoBottom()
	}
	return log
}

func (m Model) methodologyLogContent(width int) string {
	lines := make([]string, 0, len(m.researchLines))
	for _, line := range m.researchLines {
		lines = append(lines, truncateMiddle(strings.Join(strings.Fields(line), " "), max(1, width)))
	}
	return strings.Join(lines, "\n")
}

func methodologyLogHeight(height int) int {
	return methodologyVisibleLogLines
}

func (m Model) methodologyLogScrollable() bool {
	log := m.methodologyLogViewport(styles.ClampWidth(m.width-4), methodologyLogHeight(m.height))
	return log.TotalLineCount() > log.VisibleLineCount()
}

func (m Model) methodologyPhaseTable(width int) table.Model {
	labels := m.researchPhaseLabels()
	artifacts := m.researchPhaseArtifacts()
	rows := make([]table.Row, 0, len(labels))
	for i, phase := range labels {
		status := "queued"
		if i < m.researchStep {
			status = "done"
		}
		if i == m.researchStep && !m.researchDone {
			status = "working"
		}
		if m.methodologyFailed && i == m.methodologyPhaseIndex {
			status = "failed"
		}
		if m.researchDone && m.researchStep >= len(labels) {
			status = "done"
		}
		artifact := ""
		if i < len(artifacts) {
			artifact = artifacts[i]
		}
		rows = append(rows, table.Row{phase, status, artifact})
	}
	return newDataTable(
		[]table.Column{
			{Title: "Phase", Width: max(18, width/3)},
			{Title: "Status", Width: 10},
			{Title: "Artifact", Width: max(24, width-width/3-14)},
		},
		rows,
		width,
		min(max(4, len(rows)+1), max(4, m.height-16)),
		false,
	)
}

func (m Model) methodologyCue() string {
	if m.methodologyFailed {
		return styles.ErrorText.Render("Paused on error.") + " " + styles.NextActionText.Render("Retry this phase, or return to the project.")
	}
	if m.researchDone && strings.TrimSpace(m.note) != "" {
		return styles.NextCueTitle.Render("Next: ") + styles.NextActionText.Render(m.note)
	}
	return ""
}

func (m Model) methodologyQuietStatus() string {
	if m.researchDone || m.methodologyFailed {
		return ""
	}
	if m.methodologyEvents == nil && m.methodologyDone == nil {
		return ""
	}
	if m.fxFrame-m.methodologyLastEventFrame < methodologyQuietFrameDelay {
		return ""
	}
	phase := methodologyPhaseLabel(m.methodologyPhaseID)
	if strings.TrimSpace(phase) == "" {
		phase = "this phase"
	}
	return styles.MutedText.Render(fmt.Sprintf("Still running %s. Some agent tool calls stay quiet until they finish.", phase))
}
