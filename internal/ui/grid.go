package ui

import (
	"fmt"
	"github.com/leechis7/tuivory/go"
	"strconv"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
	"lazyora/internal/oracle"
)

type gridState struct {
	widths  []int
	painted resultSelection
	row     int
}

// The gutter is decoration outside the Table's inner rectangle. Data column
// indices, hit testing, range selection and raw copying remain unchanged.
type resultGrid struct {
	*tview.Table
	owner *screen
}

func (g *resultGrid) Focus(delegate func(tview.Primitive)) { delegate(g.Table) }
func (s *screen) resultGrid() tview.Primitive              { return &resultGrid{Table: s.table, owner: s} }

// Grid-specific surfaces retain the app's muted blue/cyan palette while
// distinguishing header, row, range and active cell without bright row stripes.
type gridColors struct{ background, stripe, header, current, rangeBG, divider, muted tcell.Color }

func (s *screen) gridHeaderStyle(active bool) tcell.Style {
	return s.tuivoryStyles().GridHeaderStyle(active)
}
func (s *screen) gridActiveStyle(focused bool) tcell.Style {
	return s.tuivoryStyles().GridActiveStyle(focused, false)
}
func gridFill(terminal tcell.Screen, x, y, width int, style tcell.Style) {
	for px := x; px < x+width; px++ {
		terminal.SetContent(px, y, ' ', nil, style)
	}
}
func (g *resultGrid) Draw(terminal tcell.Screen) {
	s := g.owner
	rows := max(0, g.GetRowCount()-1)
	gutter := max(4, len(strconv.Itoa(rows))+3)
	g.SetBorderPadding(0, 1, gutter, 0)
	g.SetSelectedStyle(s.gridActiveStyle(g.HasFocus()))
	selectedRow, selectedCol := g.GetSelection()
	if s.isEditedCell(selectedRow, selectedCol) {
		g.SetSelectedStyle(s.editedCellStyle().Underline(g.HasFocus()))
	}
	for col := 0; col < g.GetColumnCount(); col++ {
		g.GetCell(0, col).SetStyle(s.gridHeaderStyle(col == selectedCol))
	}
	g.Table.Draw(terminal)
	x, y, w, h := g.GetInnerRect()
	if w <= 0 || h <= 0 {
		return
	}
	selected, _ := g.GetSelection()
	type edge struct{ x, col int }
	var edges []edge
	last := x
	for col := 0; col < g.GetColumnCount(); col++ {
		cx, cy, cw := g.GetCell(0, col).GetLastPosition()
		if cx < x || cx >= x+w || cy != y {
			continue
		}
		row, actual := g.CellAt(cx, cy)
		if row != 0 || actual != col {
			continue
		}
		last = max(last, cx+cw+1)
		if cx+cw < x+w {
			edges = append(edges, edge{cx + cw, col})
		}
	}
	for py := y; py < y+h; py++ {
		row, _ := g.CellAt(x, py)
		style := s.tuivoryStyles().GridGutterStyle()
		label := ""
		if row == 0 {
			label = "#"
			style = s.gridHeaderStyle(false)
		} else if row > 0 && row <= rows {
			label = strconv.Itoa(row)
			for col := 0; col < s.table.GetColumnCount(); col++ {
				if s.isEditedCell(row, col) {
					label = "*" + label
					break
				}
			}
			style = s.gridBaseStyle(row, 0)
		}
		if last < x+w {
			gridFill(terminal, last, py, x+w-last, style)
		}
		gridFill(terminal, x-gutter, py, gutter, style)
		fg, _, _ := style.Decompose()
		tview.Print(terminal, label, x-gutter+1, py, gutter-2, tview.AlignRight, fg)
		divider := s.tuivoryStyles().GridDividerColor()
		terminal.SetContent(x-1, py, '│', nil, style.Foreground(divider))
		for _, e := range edges {
			terminal.SetContent(e.x, py, '│', nil, style.Foreground(divider))
		}
		if row == selected && row > 0 && row <= rows {
			terminal.SetContent(x-gutter, py, '›', nil, s.gridActiveStyle(g.HasFocus()))
		}
	}
	bar := s.tuivoryStyles().GridStatusStyle()
	gridFill(terminal, x-gutter, y+h, w+gutter, bar)
	text := s.gridStatusText()
	fg, _, _ := bar.Decompose()
	tview.Print(terminal, tview.Escape(" "+text), x-gutter, y+h, w+gutter, tview.AlignLeft, fg)
}

func (s *screen) gridStatusText() string {
	row, col := s.table.GetSelection()
	text := fmt.Sprintf(s.tr("grid_summary"), len(s.resultRows), len(s.resultHeaders))
	if row > 0 && row <= len(s.resultRows) && col >= 0 && col < len(s.resultHeaders) {
		text = strings.TrimPrefix(strings.TrimSpace(fmt.Sprintf(s.tr("grid_position"), row, len(s.resultRows), col+1, len(s.resultHeaders))), "· ")
		text += " · " + s.resultHeaders[col] + " · " + s.resultValues[row-1][col].Type
		if s.isEditedCell(row, col) {
			text += " · * " + s.tr("grid_uncommitted_cell")
		}
	}
	if s.resultSelection.active {
		r1, c1, r2, c2 := s.resultBounds()
		text += fmt.Sprintf(s.tr("grid_range"), r2-r1+1, c2-c1+1)
	}
	if s.resultBrowser.sorted {
		text += " · " + s.tr("grid_sorted")
	}
	if s.resultBrowser.filter != "" {
		text += " · " + s.tr("grid_filtered")
	}
	if s.resultBrowser.source.PageID != 0 || s.resultTruncated {
		text += " · " + s.tr("grid_more")
	}
	return text
}
func (s *screen) refreshGridHeader(col int) {
	label := " " + s.resultHeaders[col] + " "
	if s.resultBrowser.sorted && s.resultBrowser.sortColumn == col {
		arrow := "↑"
		if s.resultBrowser.descending {
			arrow = "↓"
		}
		label = " " + arrow + " " + s.resultHeaders[col] + " "
	}
	width := s.grid.widths[col]
	label += strings.Repeat(" ", max(0, width-uniseg.StringWidth(label)))
	s.table.GetCell(0, col).SetText(tview.Escape(label)).SetMaxWidth(width)
}

// Limit rendered values, not retained/copy/export values. Iterate graphemes
// rather than bytes so a preview never splits Hangul, emoji or combining text.
func gridPreview(value string) string {
	var out strings.Builder
	parts := uniseg.NewGraphemes(value)
	width := 0
	for parts.Next() {
		part := parts.Str()
		if out.Len()+len(part) > 2048 {
			out.WriteRune('…')
			break
		}
		switch part {
		case "\r\n", "\n", "\r":
			part = "↵"
		case "\t":
			part = "⇥"
		}
		part = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return '�'
			}
			return r
		}, part)
		count := uniseg.StringWidth(part)
		if width+count > 160 {
			out.WriteRune('…')
			break
		}
		out.WriteString(part)
		width += count
	}
	return out.String()
}

func gridNumeric(cell oracle.Cell) bool {
	kind := strings.ToUpper(cell.Type)
	kind, _, _ = strings.Cut(kind, "(")
	switch kind {
	case "NUMBER", "FLOAT", "BINARY_FLOAT", "BINARY_DOUBLE", "INTEGER", "DECIMAL", "NUMERIC":
		return true
	}
	return false
}
func (s *screen) gridBaseStyle(row, col int) tcell.Style {
	current, _ := s.table.GetSelection()
	null := row > 0 && row <= len(s.resultValues) && col >= 0 && col < len(s.resultValues[row-1]) && s.resultValues[row-1][col].Null
	return s.tuivoryStyles().GridCellStyle(tuivory.CellState{Row: row, CurrentRow: row == current, Null: null})
}

func (s *screen) paintResultRows(rows map[int]bool) {
	r1, c1, r2, c2 := s.resultBounds()
	for row := range rows {
		if row < 1 || row >= s.table.GetRowCount() {
			continue
		}
		for col := 0; col < s.table.GetColumnCount(); col++ {
			style := s.gridBaseStyle(row, col)
			if s.isEditedCell(row, col) {
				style = s.editedCellStyle()
			}
			if s.resultSelection.active && row >= r1 && row <= r2 && col >= c1 && col <= c2 {
				style = s.fieldStyle().Underline(true)
			}
			s.table.GetCell(row, col).SetStyle(style).SetTransparency(false)
		}
	}
}

func (s *screen) resizeResultColumn(delta int) {
	_, col := s.table.GetSelection()
	if col < 0 || col >= len(s.grid.widths) {
		return
	}
	width := s.grid.widths[col]
	if width == 0 {
		width = s.gridAutoWidth(col)
	}
	if delta == 0 {
		width = s.gridAutoWidth(col)
	} else {
		width = max(4, min(160, width+delta))
	}
	s.grid.widths[col] = width
	for row := 0; row < s.table.GetRowCount(); row++ {
		s.table.GetCell(row, col).SetMaxWidth(width).SetExpansion(0)
	}
	s.refreshGridHeader(col)
	s.setStatus("grid_width", s.resultHeaders[col], width)
}

func (s *screen) gridAutoWidth(col int) int {
	width := uniseg.StringWidth(s.resultHeaders[col]) + 2
	if s.resultBrowser.sorted && s.resultBrowser.sortColumn == col {
		width += 2
	}
	for row := 1; row < s.table.GetRowCount(); row++ {
		width = max(width, tview.TaggedStringWidth(s.table.GetCell(row, col).Text))
	}
	return max(4, min(40, width))
}

func (s *screen) refreshResultPosition() {
	title := s.tr("results_title")
	if object := s.resultBrowser.source.EditObject; object.Kind == "TABLE" {
		title += " · " + tview.Escape(oracle.QualifiedObject(object))
		if s.resultBrowser.source.EditID != 0 {
			title += " · " + s.tr("grid_editable")
		} else {
			title += " · " + s.tr("grid_readonly")
		}
	}
	if badge := s.connectionTitleBadge(); badge != "" {
		title = " " + tview.Escape(badge) + " · " + title
	}
	s.table.SetTitle(title)
}

func (s *screen) gridColors() gridColors {
	p := s.tuivoryStyles().GridColors()
	return gridColors{p.Background, p.Stripe, p.Header, p.Current, p.RangeBG, p.Divider, p.Muted}
}
