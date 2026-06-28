package app

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func (m Model) canOpenSourceBoard() bool {
	return len(m.sourceItems) > 0 || len(m.currentTape.Sources) > 0
}

func (m Model) startSourceBoard() (Model, tea.Cmd) {
	if !m.canOpenSourceBoard() {
		m.err = "Add or save sources before opening Review Sources."
		return m, nil
	}
	if len(m.sourceItems) == 0 {
		m.sourceItems = source.Stage(m.currentTape.Sources, true)
	}
	m.boardIndex = min(max(0, m.boardIndex), max(0, len(m.sourceItems)-1))
	m.screen = screenBoard
	m.note = "Review Sources is an advanced source table. Toggle only when you need to adjust sources before compile."
	return m, nil
}

func (m Model) handleBoardKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case "up", "k":
		if m.boardIndex > 0 {
			m.boardIndex--
		}
	case "down", "j":
		if m.boardIndex < len(m.sourceItems)-1 {
			m.boardIndex++
		}
	case " ", "space":
		if m.boardIndex >= 0 && m.boardIndex < len(m.sourceItems) {
			m.toggleBoardItem(m.sourceItems[m.boardIndex].ID)
			return m, writeSourceManifest(m.currentPath, m.sourceItems)
		}
	case "enter":
		return m.startCompile()
	}
	m.boardIndex = min(max(0, m.boardIndex), max(0, len(m.sourceItems)-1))
	return m, nil
}

func (m Model) viewBoard() string {
	width := styles.ClampWidth(m.width - 4)
	boardTable := m.boardTable(width)
	detail := m.boardDetail(width)
	counts := fmt.Sprintf(
		"%d active  %d needs review  %d inactive",
		len(m.boardItems(0)),
		len(m.boardItems(1)),
		len(m.boardItems(2)),
	)
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.Title.Render("Personal Sources"),
		styles.Subtitle.Render("Review and toggle sources before compiling."),
		styles.Section.Render(counts),
		boardTable.View(),
		"",
		styles.ReportSection.Render("Selected"),
		detail,
	)
}

func (m Model) boardItems(column int) []source.StagedSource {
	var out []source.StagedSource
	for _, item := range m.sourceItems {
		switch column {
		case 0:
			if item.Active {
				out = append(out, item)
			}
		case 1:
			if item.Status == "needs review" {
				out = append(out, item)
			}
		case 2:
			if !item.Active {
				out = append(out, item)
			}
		}
	}
	return out
}

func (m *Model) toggleBoardItem(id string) {
	for i := range m.sourceItems {
		if m.sourceItems[i].ID == id {
			m.sourceItems[i].Active = !m.sourceItems[i].Active
			return
		}
	}
}

func (m Model) boardDetail(width int) string {
	if m.boardIndex < 0 || m.boardIndex >= len(m.sourceItems) {
		return styles.SoftText.Render("No source selected.")
	}
	item := m.sourceItems[m.boardIndex]
	state := boardState(item)
	reason := "This personal source is active and will be included when you compile."
	if !item.Active {
		reason = "This personal source is inactive. It remains stored locally and can be reactivated."
	}
	if item.Status == "needs review" {
		reason = "Liner could not confidently classify this personal source. Review it before compiling."
	}
	return sourceMetadataTable(width, item, state, reason)
}

func (m Model) boardTable(width int) table.Model {
	columns := boardColumns(width)
	rows := make([]table.Row, 0, len(m.sourceItems))
	for _, item := range m.sourceItems {
		rows = append(rows, table.Row{
			sourceUseMark(item.Active),
			boardState(item),
			truncateMiddle(item.Type, columns[2].Width),
			truncateMiddle(item.Label, columns[3].Width),
			truncateMiddle(item.Destination, columns[4].Width),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, table.Row{"", "empty", "", truncateMiddle("No sources available", columns[3].Width), ""})
	}
	t := newVisibleDataTable(
		columns,
		rows,
		width,
		min(max(4, len(rows)+1), max(4, m.height-14)),
		true,
		min(max(0, m.boardIndex), max(0, len(rows)-1)),
	)
	return t
}

func boardColumns(width int) []table.Column {
	width = max(width, 60)
	usableWidth := max(36, width-12)
	useWidth := 5
	stateWidth := 12
	typeWidth := 8
	remaining := max(20, usableWidth-useWidth-stateWidth-typeWidth)
	savedWidth := min(24, max(10, remaining/3))
	sourceWidth := max(10, remaining-savedWidth)
	if sourceWidth+savedWidth > remaining {
		savedWidth = max(8, remaining-sourceWidth)
	}
	return []table.Column{
		{Title: "Use", Width: useWidth},
		{Title: "Status", Width: stateWidth},
		{Title: "Type", Width: typeWidth},
		{Title: "Source", Width: sourceWidth},
		{Title: "Saved as", Width: savedWidth},
	}
}

func boardState(item source.StagedSource) string {
	if item.Status == "needs review" {
		return "needs review"
	}
	if item.Active {
		return "active"
	}
	return "inactive"
}
