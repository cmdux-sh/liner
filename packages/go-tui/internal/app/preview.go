package app

import (
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func (m Model) openPreview(rel string) (Model, tea.Cmd) {
	path := projectAbsPath(m.currentPath, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	return m.openPreviewContent(rel, string(data))
}

func (m Model) openPreviewContent(rel string, content string) (Model, tea.Cmd) {
	m.previewBack = m.screen
	m.hasPreviewBack = true
	m.setPreviewContent(rel, content)
	m.screen = screenPreview
	return m, nil
}

func (m Model) closePreview() Model {
	if m.hasPreviewBack {
		m.screen = m.previewBack
		m.hasPreviewBack = false
		return m
	}
	m.screen = screenProject
	if m.currentPath == "" {
		m.screen = screenHome
	}
	return m
}

func (m Model) viewPreview() string {
	title := "Preview"
	if strings.TrimSpace(m.previewRel) != "" {
		title = "Preview " + m.previewRel
	}
	preview := m.preview
	preview.SetWidth(max(1, styles.ClampWidth(m.width-4)))
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Title.Render(title),
		preview.View(),
	)
}
