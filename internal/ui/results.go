package ui

import (
	"encoding/csv"
	"strings"

	"github.com/leechis7/tuivory/go"
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

// The app intercepts Oracle editing/context commands; all pointer selection,
// hit testing and range geometry come from the common Grid.
func (s *screen) resultMouse(action tuivory.MouseAction, event *tuivory.EventMouse) (tuivory.MouseAction, *tuivory.EventMouse) {
	if !s.prefs.Mouse {
		return tuivory.MouseConsumed, nil
	}
	page, _ := s.pages.GetFrontPage()
	if page != "main" && page != "routine_grid" {
		return action, event
	}
	s.prepareResultGrid()
	x, y := event.Position()
	left, top, w, h := s.table.GetInnerRect()
	if s.inlineEdit != nil && page == "main" {
		e := s.inlineEdit
		if !s.inlineTargetValid(e) {
			s.closeInlineEdit()
			return tuivory.MouseConsumed, nil
		}
		if handler := s.grid.presentation.view.MouseHandler(); handler != nil {
			handler(action, event, func(tuivory.Primitive) {})
		}
		s.inlineEditEvent(tuivory.GridEvent{})
		return tuivory.MouseConsumed, nil
	}
	if action == tuivory.MouseRightClick {
		if page == "main" {
			s.app.SetFocus(s.table)
			s.showContextMenu()
		}
		return tuivory.MouseConsumed, nil
	}
	pointer := map[tuivory.MouseAction]string{tuivory.MouseLeftDown: "down", tuivory.MouseLeftUp: "up", tuivory.MouseMove: "move", tuivory.MouseLeftClick: "click", tuivory.MouseLeftDoubleClick: "double", tuivory.MouseScrollUp: "wheel_up", tuivory.MouseScrollDown: "wheel_down", tuivory.MouseScrollLeft: "wheel_left", tuivory.MouseScrollRight: "wheel_right"}[action]
	if pointer == "" {
		return action, event
	}
	if !s.table.InRect(x, y) && pointer != "up" && pointer != "move" {
		return action, event
	}
	// Main-grid double-click shares Enter's in-cell edit action.
	// Routine results retain their read-only detail viewer.
	if pointer == "double" {
		row, col := s.resultCellAt(x, y)
		if row > 0 && col >= 0 {
			s.table.Select(row, col)
			s.app.SetFocus(s.table)
			if page == "main" {
				s.editSelectedResultCell()
			} else {
				s.showCellDetail()
			}
		}
		return tuivory.MouseConsumed, nil
	}
	p := s.grid.presentation
	if pointer != "move" && pointer != "up" || pointer == "move" && p.pointer {
		s.app.SetFocus(s.table)
	}
	eventResult := p.model.Pointer(pointer, x-left, y-top, w, h, event.Modifiers()&tuivory.ModShift != 0)
	if pointer == "down" && eventResult.Kind != "" {
		p.pointer = true
	}
	if pointer == "up" {
		p.pointer = false
	}
	s.resultGridEvent(eventResult)
	return tuivory.MouseConsumed, nil
}
func (s *screen) extendResultSelection(key tuivory.Key) {
	s.prepareResultGrid()
	action := tuivory.GridKey(tuivory.NewEventKey(key, 0, tuivory.ModShift))
	_, _, _, h := s.table.GetInnerRect()
	s.resultGridEvent(s.grid.presentation.model.Handle(action, h))
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
			if r <= len(s.resultValues) && c < len(s.resultValues[r-1]) && s.resultValues[r-1][c].Partial {
				s.setStatus("cell_copy_partial")
				return
			}
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
	text := s.newTextView().SetText(s.clipboardText).SetWrap(false)
	text.SetBorder(true).SetTitle(s.tr("copy_preview"))
	s.styleBox(text.Box)
	s.textSurface(text, surfaceField)

	s.copyText = text
	root := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(s.newTextView(textHint).SetText(s.tr("copy_preview_hint")).SetWrap(true), 2, 0, false).AddItem(text, 0, 1, true)
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
