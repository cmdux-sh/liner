package app

import (
	"charm.land/bubbles/v2/table"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func newDataTable(columns []table.Column, rows []table.Row, width int, height int, focused bool) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(focused),
		table.WithWidth(width),
		table.WithHeight(height),
	)
	t.SetStyles(dataTableStyles(focused))
	return t
}

func newVisibleDataTable(columns []table.Column, rows []table.Row, width int, height int, focused bool, cursor int) table.Model {
	t := newDataTable(columns, rows, width, height, focused)
	t.SetCursor(cursor)
	// Bubble's table renders one row beyond the viewport around the cursor.
	// Re-applying the movement logic nudges the viewport so the cursor remains visible.
	t.MoveDown(0)
	return t
}

type actionTableRow struct {
	Key    string
	Action string
	Writes string
}

type metadataTableRow struct {
	Field string
	Value string
}

func newMetadataTable(width int, rows []metadataTableRow) table.Model {
	width = max(width, 60)
	usableWidth := max(40, width-8)
	fieldWidth := min(18, max(10, usableWidth/4))
	valueWidth := max(24, usableWidth-fieldWidth)
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		value := fallbackText(row.Value, "none")
		tableRows = append(tableRows, table.Row{
			truncateMiddle(row.Field, fieldWidth),
			truncateMiddle(value, valueWidth),
		})
	}
	if len(tableRows) == 0 {
		tableRows = append(tableRows, table.Row{"none", ""})
	}
	return newDataTable(
		[]table.Column{
			{Title: "Field", Width: fieldWidth},
			{Title: "Value", Width: valueWidth},
		},
		tableRows,
		width,
		len(tableRows)+1,
		false,
	)
}

func dedupeMetadataRows(rows []metadataTableRow) []metadataTableRow {
	seen := map[string]bool{}
	out := make([]metadataTableRow, 0, len(rows))
	for _, row := range rows {
		key := row.Field + "\x00" + row.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, row)
	}
	return out
}

func newActionTable(width int, rows []actionTableRow) table.Model {
	width = max(width, 60)
	usableWidth := max(40, width-8)
	keyWidth := min(12, max(8, usableWidth/5))
	writesWidth := min(24, max(18, usableWidth/3))
	actionWidth := max(18, usableWidth-keyWidth-writesWidth)
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, table.Row{
			truncateMiddle(row.Key, keyWidth),
			truncateMiddle(row.Action, actionWidth),
			truncateMiddle(row.Writes, writesWidth),
		})
	}
	return newDataTable(
		[]table.Column{
			{Title: "Key", Width: keyWidth},
			{Title: "Action", Width: actionWidth},
			{Title: "Writes", Width: writesWidth},
		},
		tableRows,
		width,
		len(tableRows)+1,
		false,
	)
}

func newReviewActionTable(width int, acceptAction string, acceptWrites string, returnTarget string) table.Model {
	return newActionTable(width, []actionTableRow{
		{Key: "enter", Action: acceptAction, Writes: acceptWrites},
		{Key: "o", Action: "Open draft", Writes: "read-only"},
		{Key: "d", Action: "Discard draft", Writes: "remove draft"},
		{Key: "esc", Action: "Return to " + returnTarget, Writes: "no changes"},
	})
}

func dataTableStyles(focused bool) table.Styles {
	s := table.DefaultStyles()
	s.Header = styles.TableHeader.PaddingRight(2)
	s.Cell = styles.PrimaryText.PaddingRight(2)
	s.Selected = styles.TableSelected
	if focused {
		s.Selected = styles.TableSelectedFocused
	}
	return s
}
