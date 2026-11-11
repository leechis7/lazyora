package ui

import (
	"lazyora/internal/oracle"
	"time"

	"github.com/leechis7/tuivory/go"
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
	index := s.resultBrowser.visible[row-1]
	_, changed := s.gridDrafts[s.editedCellIdentity(s.resultBrowser.source, index, s.resultSourceColumn(col))]
	return changed && !s.gridRowPendingDelete(index)
}
func (s *screen) editedCellStyle() tuivory.Style {
	return s.tuivoryStyles().EditedStyle()
}

// Only Oracle target metadata lives here. Tuivory owns the draft buffer,
// rendering, cursor, clipping, and the edit session lifecycle.
type inlineCellEditor struct {
	model  *tuivory.Grid
	change oracle.EditChange
	key    editedCellKey
}

func (e *inlineCellEditor) GetText() string {
	session, _ := e.model.EditSession()
	return session.Value.Text
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

// Main-grid activation is an editing request. Keep unavailable cells in place
// instead of opening a details popup that resembles the separate editor.
func (s *screen) editSelectedResultCell() {
	if row, ok := s.selectedGridSourceRow(); ok && row < 0 && s.canChangeGridRows() {
		s.showGridInsert(row)
		return
	}
	if !s.startInlineEdit(nil) && !s.busy {
		s.setStatus("grid_inline_unavailable")
	}
}

func (s *screen) startInlineEdit(first *rune) bool {
	if s.inlineEdit != nil || !s.commandEnabled("grid_edit") {
		return false
	}
	s.prepareResultGrid()
	row, col := s.table.GetSelection()
	index := s.resultBrowser.visible[row-1]
	sourceCol := s.resultSourceColumn(col)
	model := s.grid.presentation.model
	text := ""
	if first != nil {
		text = string(*first)
	}
	if !model.BeginEdit(text) {
		return false
	}
	s.inlineEdit = &inlineCellEditor{model: model,
		change: oracle.EditChange{ID: s.resultBrowser.source.EditID, Row: index, Column: sourceCol},
		key:    s.editedCellIdentity(s.resultBrowser.source, index, sourceCol)}
	s.clearResultSelection()
	s.app.SetFocus(s.table)
	s.setStatus("grid_inline_hint")
	return true
}

func (s *screen) inlineTargetValid(e *inlineCellEditor) bool {
	return e != nil && s.inlineEdit == e && s.grid.presentation != nil &&
		s.grid.presentation.model == e.model && s.connected && !s.transactionUncertain &&
		s.resultBrowser.source.EditID == e.change.ID && s.canEditGridCell(e.change.Row, e.change.Column) &&
		s.editedCellIdentity(s.resultBrowser.source, e.change.Row, e.change.Column) == e.key
}

func (s *screen) closeInlineEdit() {
	if s.inlineEdit == nil {
		return
	}
	s.inlineEdit.model.CancelEdit()
	if p := s.grid.presentation; p != nil && p.model == s.inlineEdit.model {
		p.view.Blur()
	}
	s.inlineEdit = nil
	if s.programModal == nil {
		s.app.SetFocus(s.table)
	}
}

func (s *screen) inlineEditKey(event *tuivory.EventKey) *tuivory.EventKey {
	e := s.inlineEdit
	if !s.inlineTargetValid(e) {
		s.closeInlineEdit()
		s.setStatus("grid_inline_unavailable")
		return nil
	}
	if event.Key() == tuivory.KeyF9 || event.Key() == tuivory.KeyF5 ||
		event.Key() == tuivory.KeyEnter && event.Modifiers()&(tuivory.ModCtrl|tuivory.ModAlt) != 0 {
		s.inlineEditEvent(e.model.EditInput("submit", ""))
		if s.inlineEdit == nil {
			return s.key(event)
		}
		return nil
	}
	if event.Key() == tuivory.KeyRune && event.Modifiers()&tuivory.ModAlt != 0 {
		switch event.Rune() {
		case 'c':
			s.inlineEditEvent(e.model.EditInput("submit", ""))
			if s.inlineEdit == nil {
				s.action("commit")
			}
			return nil
		case 'r':
			s.closeInlineEdit()
			s.action("rollback")
			return nil
		}
	}
	if event.Key() == tuivory.KeyTab || event.Key() == tuivory.KeyBacktab {
		model := e.model
		s.inlineEditEvent(model.EditInput("submit", ""))
		if s.inlineEdit == nil && model.MoveEditable(event.Key() == tuivory.KeyBacktab || event.Modifiers()&tuivory.ModShift != 0) {
			s.resultGridEvent(tuivory.GridEvent{Kind: "select"})
			s.startInlineEdit(nil)
		}
		return nil
	}
	var result tuivory.GridEvent
	if event.Key() == tuivory.KeyDelete && event.Modifiers()&tuivory.ModAlt != 0 {
		e.model.EditInput("select_all", "")
		e.model.EditInput("delete", "")
		result = e.model.EditInput("null", "")
	} else if event.Key() == tuivory.KeyCtrlU {
		e.model.EditInput("select_all", "")
		result = e.model.EditInput("delete", "")
	} else {
		_, _, w, h := s.table.GetInnerRect()
		result = e.model.Input(tuivory.InputFromKey(event), w, h)
	}
	s.inlineEditEvent(result)
	return nil
}

func (s *screen) inlineEditEvent(event tuivory.GridEvent) {
	e := s.inlineEdit
	if e == nil {
		return
	}
	session, active := e.model.EditSession()
	if !active {
		s.closeInlineEdit()
		s.resultScopeStatus()
		return
	}
	if !s.inlineTargetValid(e) {
		s.closeInlineEdit()
		return
	}
	if event.Kind == "copy" || event.Kind == "cut" {
		s.sendClipboard(event.Text)
		return
	}
	if event.Kind != "edit_submit" {
		return
	}
	change := e.change
	change.Text, change.Null = session.Value.Text, session.Value.Null
	if !e.model.ResolveEdit(session.ID, "", nil) {
		s.closeInlineEdit()
		return
	}
	s.closeInlineEdit()
	s.stageGridChange(change)
}
