package ui

import (
	"encoding/csv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type resultSelection struct {
	anchorRow, anchorCol, endRow, endCol int
	active, dragging                     bool
}

func (s *screen) resultBounds() (int, int, int, int) {
	a := s.resultSelection
	return min(a.anchorRow, a.endRow), min(a.anchorCol, a.endCol), max(a.anchorRow, a.endRow), max(a.anchorCol, a.endCol)
}

func (s *screen) highlightResults() {
	current, _ := s.table.GetSelection()
	rows := map[int]bool{s.grid.row: true, current: true}
	for _, selection := range []resultSelection{s.grid.painted, s.resultSelection} {
		if selection.active {
			for row := min(selection.anchorRow, selection.endRow); row <= max(selection.anchorRow, selection.endRow); row++ {
				rows[row] = true
			}
		}
	}
	s.paintResultRows(rows)
	s.grid.painted, s.grid.row = s.resultSelection, current
}

func (s *screen) clearResultSelection() {
	s.resultSelection = resultSelection{}
	s.highlightResults()
}

func (s *screen) resultMouse(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
	if !s.prefs.Mouse {
		return action, nil
	}
	page, _ := s.pages.GetFrontPage()
	if page != "main" && page != "routine_grid" {
		return action, event
	}
	x, y := event.Position()
	row, col := s.table.CellAt(x, y)
	innerX, _, _, _ := s.table.GetInnerRect()
	if x < innerX && s.table.InRect(x, y) && row > 0 && row < s.table.GetRowCount() && (action == tview.MouseLeftDown || action == tview.MouseLeftClick) {
		if action == tview.MouseLeftDown {
			s.resultSelection = resultSelection{anchorRow: row, endRow: row, anchorCol: 0, endCol: len(s.resultHeaders) - 1, active: true}
			s.table.Select(row, 0)
			s.app.SetFocus(s.table)
			s.highlightResults()
		}
		return tview.MouseConsumed, nil
	}
	valid := s.table.InRect(x, y) && row > 0 && row < s.table.GetRowCount() && col >= 0 && col < s.table.GetColumnCount()
	switch action {
	case tview.MouseRightClick:
		if page == "routine_grid" {
			return action, event
		}
		s.app.SetFocus(s.table)
		s.showContextMenu()
		return tview.MouseConsumed, nil
	case tview.MouseLeftDown:
		s.clearResultSelection()
		if valid {
			s.resultSelection = resultSelection{anchorRow: row, anchorCol: col, endRow: row, endCol: col, active: true, dragging: true}
			s.table.Select(row, col)
			s.app.SetFocus(s.table)
		}
	case tview.MouseMove:
		if s.resultSelection.dragging && event.Buttons()&tcell.Button1 != 0 && valid {
			s.resultSelection.endRow, s.resultSelection.endCol = row, col
			s.table.Select(row, col)
			s.highlightResults()
		}
	case tview.MouseLeftUp:
		s.resultSelection.dragging = false
	case tview.MouseLeftClick, tview.MouseLeftDoubleClick:
		// Preserve the drag end; Table's default click would collapse the rectangle.
		if s.resultSelection.active {
			return tview.MouseConsumed, nil
		}
	}
	s.highlightResults()
	return action, event
}

func (s *screen) extendResultSelection(key tcell.Key) {
	row, col := s.table.GetSelection()
	if row <= 0 || s.table.GetRowCount() < 2 || s.table.GetColumnCount() == 0 {
		return
	}
	if !s.resultSelection.active {
		s.resultSelection = resultSelection{anchorRow: row, anchorCol: col, active: true}
	}
	switch key {
	case tcell.KeyUp:
		row--
	case tcell.KeyDown:
		row++
	case tcell.KeyLeft:
		col--
	case tcell.KeyRight:
		col++
	}
	row = max(1, min(row, s.table.GetRowCount()-1))
	col = max(0, min(col, s.table.GetColumnCount()-1))
	s.resultSelection.endRow, s.resultSelection.endCol = row, col
	s.table.Select(row, col)
	s.highlightResults()
}

func (s *screen) copyResults(mode string) {
	if len(s.resultRows) == 0 {
		s.setStatus("copy_empty")
		return
	}
	row, col := s.table.GetSelection()
	if row <= 0 || row > len(s.resultRows) {
		s.setStatus("copy_empty")
		return
	}
	r1, r2, c1, c2 := row, row, col, col
	if mode == "row" {
		c1, c2 = 0, len(s.resultHeaders)-1
	} else if s.resultSelection.active {
		r1, c1, r2, c2 = s.resultBounds()
	}
	var data strings.Builder
	writer := csv.NewWriter(&data)
	writer.Comma = '\t'
	for r := r1; r <= r2; r++ {
		values := make([]string, 0, c2-c1+1)
		for c := c1; c <= c2; c++ {
			value := ""
			if c < len(s.resultRows[r-1]) {
				value = s.resultRows[r-1][c]
			}
			values = append(values, value)
		}
		if err := writer.Write(values); err != nil {
			s.setStatus("error", err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		s.setStatus("error", err)
		return
	}
	s.sendClipboard(strings.TrimSuffix(data.String(), "\n"))
}

func (s *screen) sendClipboard(text string) {
	if text == "" {
		s.setStatus("copy_empty")
		return
	}
	s.clipboardText = text
	if s.terminal != nil {
		s.terminal.SetClipboard([]byte(text))
	}
	// OSC 52 has no delivery acknowledgement; never claim the PC received it.
	s.setStatus("copy_requested", len(text))
}

func (s *screen) copyPreview() {
	if s.clipboardText == "" {
		s.setStatus("copy_empty")
		return
	}
	s.copyFocus = s.app.GetFocus()
	text := tview.NewTextView().SetText(s.clipboardText).SetWrap(false)
	text.SetBorder(true).SetTitle(s.tr("copy_preview"))
	styleBox(text.Box)
	text.SetBackgroundColor(palette.field)
	text.SetTextColor(palette.background)

	s.copyText = text
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(tview.NewTextView().SetText(s.tr("copy_preview_hint")).SetWrap(true), 2, 0, false).AddItem(text, 0, 1, true)
	s.pages.AddPage("copy", root, true, true)
	s.app.SetFocus(text)
	// Terminal selection works on clients without OSC 52.
	s.app.EnableMouse(false)
}

func (s *screen) closeCopyPreview() {
	s.pages.RemovePage("copy")
	s.app.EnableMouse(s.prefs.Mouse)
	if s.copyFocus != nil {
		s.app.SetFocus(s.copyFocus)
	}
	s.copyText = nil
}
