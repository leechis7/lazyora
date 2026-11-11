package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"maps"
	"slices"

	"github.com/leechis7/tuivory/go"
)

type gridInsertDialog struct {
	close, stage func()
}

func (s *screen) selectedGridSourceRow() (int, bool) {
	row, _ := s.table.GetSelection()
	if row < 1 || row > len(s.resultBrowser.visible) {
		return 0, false
	}
	return s.resultBrowser.visible[row-1], true
}

func (s *screen) gridRowPendingDelete(row int) bool {
	draft, found := s.gridDeletes[row]
	return found && draft.ID == s.resultBrowser.source.EditID
}

func (s *screen) canEditGridCell(row, col int) bool {
	return row >= 0 && !s.gridRowPendingDelete(row) && s.resultBrowser.source.CanEditColumn(col)
}

func (s *screen) canChangeGridRows() bool {
	return s.connected && !s.busy && !s.workerClosed && !s.transactionUncertain &&
		s.inlineEdit == nil && s.table.HasFocus()
}

func (s *screen) toggleGridDelete() {
	row, ok := s.selectedGridSourceRow()
	if !ok {
		return
	}
	source := s.resultBrowser.source
	if row < 0 {
		delete(s.gridInserts, row)
		s.reprojectGridDrafts()
	} else if source.EditID != 0 && source.EditCanDelete && row < len(source.EditRowIDs) {
		if s.gridRowPendingDelete(row) {
			delete(s.gridDeletes, row)
		} else {
			if s.gridDeletes == nil {
				s.gridDeletes = make(map[int]oracle.EditDelete)
			}
			s.gridDeletes[row] = oracle.EditDelete{ID: source.EditID, Row: row}
		}
		s.updateGridDraftDisplay(source.EditRowIDs[row])
	}
	s.setGridDraftStatus()
}

func (s *screen) showGridInsert(row int) {
	source := s.resultBrowser.source
	if !source.CanInsertRow() || s.gridInsert != nil {
		return
	}
	existing := row < 0
	draft := oracle.EditInsert{ID: source.EditID, Fields: map[int]oracle.EditInsertField{}}
	if existing {
		old, ok := s.gridInserts[row]
		if !ok || old.ID != source.EditID {
			return
		}
		draft.Fields = maps.Clone(old.Fields)
	} else if len(s.gridInserts) >= 1000 {
		s.setStatus("grid_insert_limit")
		return
	}
	var names []string
	var columns []int
	for col, name := range source.EditColumnNames {
		if name != "" && source.CanEditColumn(col) {
			columns = append(columns, col)
			names = append(names, name+" ("+source.EditColumnTypes[col]+")")
		}
	}
	if len(columns) == 0 {
		return
	}

	info := s.newTextView(textHint).SetDynamicColors(false).SetWrap(true)
	form := tuivory.NewWidgetForm()
	field := tuivory.NewInputField().SetLabel(s.tr("grid_row_value")).SetFieldWidth(42)
	mode := s.newChoice(s.tr("grid_row_mode"), []string{s.tr("grid_row_default"), "NULL", s.tr("grid_row_value")}, 0, nil)
	current := 0
	updateInfo := func(hint string) {
		info.SetText(oracle.QualifiedObject(source.EditObject) + "\n" + hint)
	}
	load := func() {
		value, set := draft.Fields[columns[current]]
		i := 0
		if set {
			i = 2
			if value.Null {
				i = 1
			}
		}
		mode.SetCurrentOption(i)
		field.SetText(value.Text)
	}
	store := func() bool {
		i, _ := mode.GetCurrentOption()
		col := columns[current]
		switch i {
		case 0:
			delete(draft.Fields, col)
		case 1:
			draft.Fields[col] = oracle.EditInsertField{Null: true}
		case 2:
			if field.GetText() == "" {
				updateInfo(s.tr("grid_row_need_value"))
				return false
			}
			draft.Fields[col] = oracle.EditInsertField{Text: field.GetText()}
		}
		return true
	}
	var choice *choiceField
	choice = s.newChoice(s.tr("grid_row_column"), names, current, func(_ string, next int) {
		if next < 0 || next == current {
			return
		}
		if !store() {
			choice.SetCurrentOption(current)
			return
		}
		current = next
		load()
		updateInfo(s.tr("grid_row_hint"))
	})
	close := func() {
		s.pages.RemovePage("grid_insert")
		s.gridInsert = nil
		s.app.SetFocus(s.table)
	}
	stage := func() {
		if s.busy || s.transactionUncertain || s.resultBrowser.source.EditID != draft.ID || !store() {
			return
		}
		if s.gridInserts == nil {
			s.gridInserts = make(map[int]oracle.EditInsert)
		}
		if !existing {
			s.gridInsertSeq++
			row = -s.gridInsertSeq
		}
		s.gridInserts[row] = draft
		close()
		s.applyResultBrowser(true)
		for i, logical := range s.resultBrowser.visible {
			if logical == row {
				s.table.Select(i+1, 0)
				break
			}
		}
		s.setGridDraftStatus()
	}
	s.gridInsert = &gridInsertDialog{close, stage}
	form.AddFormItem(choice).AddFormItem(mode).AddFormItem(field)
	form.AddButton(s.tr("edit_stage"), stage).AddButton(s.tr("cancel"), close)
	form.SetTitle(s.tr("grid_insert"))
	s.styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	updateInfo(s.tr("grid_row_hint"))
	load()
	s.pages.AddPage("grid_insert", s.formDialog(form, info, 92), true, true)
	focusForm(s, form, 0)
}

func (s *screen) gridInsertKey(event *tuivory.EventKey) *tuivory.EventKey {
	if s.buttonArrow(event) {
		return nil
	}
	switch event.Key() {
	case tuivory.KeyEscape:
		s.gridInsert.close()
	case tuivory.KeyCtrlS:
		s.gridInsert.stage()
	default:
		return event
	}
	return nil
}

func (s *screen) gridDraftCount() int {
	return len(s.gridDrafts) + len(s.gridDeletes) + len(s.gridInserts)
}

func (s *screen) gridDraftSummary() string {
	if len(s.gridDeletes)+len(s.gridInserts) == 0 {
		return fmt.Sprintf(s.tr("grid_drafts_count"), len(s.gridDrafts))
	}
	return fmt.Sprintf(s.tr("grid_row_counts"), len(s.gridDrafts), len(s.gridInserts), len(s.gridDeletes))
}

func (s *screen) setGridDraftStatus() {
	if len(s.gridDeletes)+len(s.gridInserts) == 0 {
		s.setStatus("grid_drafts_count", len(s.gridDrafts))
	} else {
		s.setStatus("grid_row_counts", len(s.gridDrafts), len(s.gridInserts), len(s.gridDeletes))
	}
}

func (s *screen) addedGridIndices() []int {
	indices := slices.Sorted(maps.Keys(s.gridInserts))
	slices.Reverse(indices) // negative sequence; oldest first
	return indices
}
