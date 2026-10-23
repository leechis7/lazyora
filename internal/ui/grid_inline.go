package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"time"
)

// Physical identities keep pending-change markers attached to the right cell
// through sorting, filtering, paging, tab switching and SELECT aliases.
type editedCellKey struct{ owner, table, rowID, column string }

func (s *screen) editedCellIdentity(result oracle.Result, row, col int) editedCellKey {
	if row < 0 || row >= len(result.EditRowIDs) || col < 0 || col >= len(result.EditColumnNames) {
		return editedCellKey{}
	}
	return editedCellKey{result.EditObject.Owner, result.EditObject.Name, result.EditRowIDs[row], result.EditColumnNames[col]}
}
func (s *screen) markEditedCell(key editedCellKey) {
	if key.rowID == "" || key.column == "" {
		return
	}
	if s.editedCells == nil {
		s.editedCells = make(map[editedCellKey]bool)
	}
	s.editedCells[key] = true
	s.repaintEditedCells()
}
func (s *screen) clearEditedCells() {
	if len(s.editedCells) == 0 {
		return
	}
	s.editedCells = nil
	s.repaintEditedCells()
}
func (s *screen) repaintEditedCells() {
	rows := make(map[int]bool)
	for row := 1; row < s.table.GetRowCount(); row++ {
		rows[row] = true
	}
	s.paintResultRows(rows)
}
func (s *screen) isEditedCell(row, col int) bool {
	if row < 1 || row > len(s.resultBrowser.visible) {
		return false
	}
	return s.editedCells[s.editedCellIdentity(s.resultBrowser.source, s.resultBrowser.visible[row-1], s.resultSourceColumn(col))]
}
func (s *screen) editedCellStyle() tcell.Style {
	fg, bg := tcell.ColorBlack, tcell.GetColor("#e8c56a")
	if s.limitedColors {
		bg = tcell.ColorOlive
	}
	if s.monochrome() {
		bg = tcell.ColorSilver
	}
	return tcell.StyleDefault.Foreground(fg).Background(bg).Bold(true)
}

// The page covers the input routing area, while Draw places the real input
// widget over just the selected cell. It never paints an opaque full-screen box.
type inlineCellEditor struct {
	*tview.InputField
	owner    *screen
	change   oracle.EditChange
	row, col int
}

func (e *inlineCellEditor) Draw(terminal tcell.Screen) {
	table := e.owner.table
	x, y, width := table.GetCell(e.row, e.col).GetLastPosition()
	left, top, w, h := table.GetInnerRect()
	actualRow, actualCol := table.CellAt(x, y)
	if actualRow != e.row || actualCol != e.col || y < top || y >= top+h || x < left || x >= left+w {
		return
	}
	width = min(max(1, width), left+w-x)
	e.InputField.SetRect(x, y, width, 1)
	e.InputField.Draw(terminal)
}
func (e *inlineCellEditor) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, focus func(tview.Primitive)) (bool, tview.Primitive) {
		e.InputField.MouseHandler()(action, event, focus)
		return true, nil // Do not change the source cell or dismiss unconfirmed text.
	}
}

func editInputText(cell oracle.Cell) string {
	if cell.Null {
		return ""
	}
	if cell.Type == "DATE" || cell.Type == "TIMESTAMP" {
		if date, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", cell.Text); err == nil {
			layout := "2006-01-02 15:04:05.999999999"
			if cell.Type == "DATE" {
				layout = "2006-01-02 15:04:05"
			}
			return date.Format(layout)
		}
	}
	return cell.Text
}
func (s *screen) startInlineEdit(first *rune) bool {
	if !s.commandEnabled("grid_edit") {
		return false
	}
	row, col := s.table.GetSelection()
	index := s.resultBrowser.visible[row-1]
	sourceCol := s.resultSourceColumn(col)
	old := browserCell(s.resultBrowser.source, index, sourceCol)
	value := editInputText(old)
	change := oracle.EditChange{ID: s.resultBrowser.source.EditID, Row: index, Column: sourceCol, Text: value, Null: old.Null}
	if first != nil {
		value = string(*first)
		change.Null = false
	}
	field := tview.NewInputField().SetText(value).SetFieldWidth(0).SetFieldStyle(s.editedCellStyle())
	field.SetPlaceholderStyle(s.editedCellStyle().Dim(true))
	editor := &inlineCellEditor{InputField: field, owner: s, change: change, row: row, col: col}
	if change.Null {
		field.SetPlaceholder("NULL")
	}
	field.SetChangedFunc(func(string) { editor.change.Null = false; field.SetPlaceholder("") })
	s.inlineEdit = editor
	s.clearResultSelection()
	s.pages.AddPage("grid_inline", editor, true, true)
	s.app.SetFocus(editor)
	s.setStatus("grid_inline_hint")
	return true
}
func (s *screen) closeInlineEdit() {
	if s.inlineEdit == nil {
		return
	}
	s.pages.RemovePage("grid_inline")
	s.inlineEdit = nil
	s.app.SetFocus(s.table)
}
func (s *screen) inlineEditKey(event *tcell.EventKey) *tcell.EventKey {
	editor := s.inlineEdit
	if editor == nil {
		return nil
	}
	switch event.Key() {
	case tcell.KeyEscape:
		s.closeInlineEdit()
		s.resultScopeStatus()
		return nil
	case tcell.KeyEnter:
		change := editor.change
		change.Text = editor.GetText()
		old := browserCell(s.resultBrowser.source, change.Row, change.Column)
		if change.Null == old.Null && (change.Null || change.Text == editInputText(old)) {
			s.closeInlineEdit()
			return nil
		}
		s.previewCellChange(change)
		return nil
	case tcell.KeyDelete:
		if event.Modifiers()&tcell.ModAlt != 0 {
			editor.SetText("")
			editor.change.Null = true
			editor.SetPlaceholder("NULL")
			return nil
		}
	}
	// Keep all application commands inside this editor until Enter/Esc. Ctrl+U,
	// cursor movement, backspace and paste retain InputField's native behavior.
	if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
		return nil
	}
	return event
}
