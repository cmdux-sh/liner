package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

func (m *Model) startImport() tea.Cmd {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	m.importPicker = newImportPicker(cwd, m.height)
	m.importBusy = false
	m.screen = screenImport
	m.note = ""
	m.err = ""
	return m.importPicker.Init()
}

func newImportPicker(dir string, height int) filepicker.Model {
	picker := filepicker.New()
	picker.CurrentDirectory = fallbackText(dir, ".")
	picker.AllowedTypes = []string{".mixtape"}
	picker.DirAllowed = false
	picker.FileAllowed = true
	picker.ShowHidden = false
	picker.ShowPermissions = false
	picker.ShowSize = true
	picker.Cursor = ">"
	picker.KeyMap.Back = key.NewBinding(
		key.WithKeys("left", "backspace"),
		key.WithHelp("←", "folder"),
	)
	picker.KeyMap.Open = key.NewBinding(
		key.WithKeys("right", "enter"),
		key.WithHelp("enter/→", "open"),
	)
	picker.KeyMap.Select = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "import"),
	)
	picker.Styles = importPickerStyles()
	picker.SetHeight(importPickerHeight(height))
	return picker
}

func importPickerStyles() filepicker.Styles {
	pickerStyles := filepicker.DefaultStyles()
	pickerStyles.Cursor = styles.AccentText
	pickerStyles.Selected = styles.AccentText.Bold(false)
	pickerStyles.Directory = styles.DimText
	pickerStyles.File = styles.PrimaryText
	pickerStyles.DisabledFile = styles.MutedText
	pickerStyles.DisabledCursor = styles.MutedText
	pickerStyles.DisabledSelected = styles.MutedText
	pickerStyles.Permission = styles.MutedText
	pickerStyles.FileSize = lipgloss.NewStyle().
		Foreground(styles.Muted).
		Width(7).
		Align(lipgloss.Right)
	pickerStyles.EmptyDirectory = styles.MutedText.
		PaddingLeft(2).
		SetString("No files here.")
	return pickerStyles
}

func importPickerHeight(height int) int {
	if height <= 0 {
		return 10
	}
	return min(14, max(5, height-15))
}

func (m Model) handleImportKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(keyMsg, bindings.Quit, bindings.QuitKey):
		return m, tea.Quit
	case key.Matches(keyMsg, bindings.Back):
		m.importBusy = false
		m.screen = screenHome
		if m.currentPath != "" {
			m.screen = screenProject
		}
		return m, nil
	case key.Matches(keyMsg, bindings.Refresh):
		m.err = ""
		m.note = "Refreshing " + m.importPicker.CurrentDirectory + "..."
		return m, m.importPicker.Init()
	}
	if m.importBusy {
		return m, nil
	}

	var cmd tea.Cmd
	m.importPicker, cmd = m.importPicker.Update(keyMsg)
	if selected, path := m.importPicker.DidSelectFile(keyMsg); selected {
		if strings.TrimSpace(path) == "" {
			return m, cmd
		}
		return m.submitImport(path)
	}
	if selected, path := m.importPicker.DidSelectDisabledFile(keyMsg); selected {
		if strings.TrimSpace(path) == "" {
			return m, cmd
		}
		name := filepath.Base(path)
		if name == "." || name == string(filepath.Separator) || name == "" {
			name = "that file"
		}
		m.err = fmt.Sprintf("%s is not a .mixtape project file.", name)
		return m, nil
	}
	return m, cmd
}

func (m Model) submitImport(path string) (Model, tea.Cmd) {
	if m.importBusy {
		return m, nil
	}
	m.err = ""
	path = strings.TrimSpace(path)
	if path == "" {
		m.err = "Choose a .mixtape project file."
		return m, nil
	}
	if !strings.EqualFold(filepath.Ext(path), ".mixtape") {
		m.err = filepath.Base(path) + " is not a .mixtape project file."
		return m, nil
	}
	m.importBusy = true
	m.note = "Importing " + filepath.Base(path) + "..."
	return m, importArchive(m.runner, path, m.baseDir)
}

func (m Model) viewImport() string {
	width := styles.ClampWidth(m.width - 4)
	picker := m.importPicker
	if strings.TrimSpace(picker.CurrentDirectory) == "" {
		picker = newImportPicker(".", m.height)
	}
	picker.SetHeight(importPickerHeight(m.height))
	detail := renderLabelValueBlock(width, []labelValueRow{
		{Label: "Folder", Value: picker.CurrentDirectory},
		{Label: "Destination", Value: m.baseDir},
		{Label: "Sources", Value: "Use archived source files"},
	}, 1, 1)
	pickerView := strings.TrimRight(picker.View(), "\n")
	if m.importBusy {
		pickerView = styles.Subtitle.Render("Importing...")
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderLoadingTitle("Import Project", m.importBusy),
		styles.Subtitle.Render("Choose a .mixtape file to import as a Liner project."),
		detail,
		lipgloss.NewStyle().Width(width).Render(pickerView),
	)
}

func importArchive(r core.Runner, archive string, destination string) tea.Cmd {
	return func() tea.Msg {
		projectPath, err := r.ImportArchive(archive, destination, true)
		if err != nil {
			return archiveImportedMsg{err: err}
		}
		imported, err := tape.ReadProject(projectPath)
		return archiveImportedMsg{path: projectPath, tape: imported, err: err}
	}
}
