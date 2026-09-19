package terminal

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Column defines one column in a Table.
type Column struct {
	Header string
	Width  int
	Align  lipgloss.Position // lipgloss.Left, lipgloss.Center, lipgloss.Right
	Style  Style             // optional: applies a Style to the column cells
}

// Table renders rows of strings as a styled ASCII table using lipgloss.
type Table struct {
	w    io.Writer
	cols []Column
	rows [][]string
}

// NewTable constructs a Table that writes to w with the given columns.
func NewTable(w io.Writer, cols ...Column) *Table {
	if w == nil {
		w = io.Discard
	}
	// Apply default alignment (left) where unset.
	for i := range cols {
		if cols[i].Align == 0 {
			cols[i].Align = lipgloss.Left
		}
		if cols[i].Width == 0 {
			cols[i].Width = 12
		}
	}
	return &Table{w: w, cols: cols}
}

// SetRows replaces the table body.
func (t *Table) SetRows(rows [][]string) *Table {
	t.rows = rows
	return t
}

// Render writes the table to the configured writer.
func (t *Table) Render() error {
	if t.w == nil {
		return nil
	}
	headers := make([]string, len(t.cols))
	for i, c := range t.cols {
		headers[i] = StyleText(StyleHeader, c.Header)
	}
	border := lipgloss.NormalBorder()
	cellStyles := make([]lipgloss.Style, len(t.cols))
	for i, c := range t.cols {
		// Cell styles use the column's alignment and padding. We intentionally
		// don't apply Width() here because lipgloss Width() would wrap or
		// truncate long cell values; the table auto-sizes to fit the widest
		// content in each column. The Width field on Column is kept as a hint
		// (defaults to 12) for callers that want it; not applied visually.
		s := lipgloss.NewStyle().Align(c.Align).Padding(0, 1)
		if c.Style != 0 {
			s = s.Inherit(styles[c.Style])
		}
		cellStyles[i] = s
	}
	headerCellStyles := make([]lipgloss.Style, len(t.cols))
	for i, c := range t.cols {
		headerCellStyles[i] = lipgloss.NewStyle().Align(c.Align).Padding(0, 1)
	}
	tbl := table.New().
		Border(border).
		BorderStyle(lipgloss.NewStyle()).
		Headers(headers...).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				if col >= 0 && col < len(headerCellStyles) {
					return headerCellStyles[col].Bold(true)
				}
				return lipgloss.NewStyle().Bold(true).Padding(0, 1)
			}
			if col >= 0 && col < len(cellStyles) {
				return cellStyles[col]
			}
			return lipgloss.NewStyle()
		})
	for _, row := range t.rows {
		styled := make([]string, len(row))
		for i, cell := range row {
			idx := i
			if idx >= len(t.cols) {
				idx = len(t.cols) - 1
			}
			styled[i] = cellStyles[idx].Render(cell)
		}
		tbl.Row(styled...)
	}
	_, _ = fmt.Fprintln(t.w, tbl.String())
	return nil
}
