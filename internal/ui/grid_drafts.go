package ui

import (
	"lazyora/internal/oracle"

	"github.com/leechis7/tuivory/go"
)

func (s *screen) displayedGridCell(row, col int) oracle.Cell {
	source := s.resultBrowser.source
	if row < 0 {
		draft := s.gridInserts[row]
		if f, ok := draft.Fields[col]; ok {
			if f.Null {
				return oracle.Cell{Text: "NULL", Null: true, Type: source.EditColumnTypes[col]}
			}
			return oracle.Cell{Text: f.Text, Type: source.EditColumnTypes[col]}
		}
		return oracle.Cell{Text: "DEFAULT"}
	}
	cell := browserCell(source, row, col)
	if change, ok := s.gridDrafts[s.editedCellIdentity(source, row, col)]; ok && !s.gridRowPendingDelete(row) {
		cell.Text, cell.Null = change.Text, change.Null
		if cell.Null {
			cell.Text = "NULL"
		}
	}
	return cell
}

func (s *screen) displayedGridText(row, col int, cell oracle.Cell) string {
	source := s.resultBrowser.source
	if _, draft := s.gridDrafts[s.editedCellIdentity(source, row, col)]; (!draft || s.gridRowPendingDelete(row)) && row >= 0 && row < len(source.Rows) && col < len(source.Rows[row]) {
		return source.Rows[row][col]
	}
	return cell.Text
}

func (s *screen) stageGridChange(change oracle.EditChange) {
	source := s.resultBrowser.source
	key := s.editedCellIdentity(source, change.Row, change.Column)
	if !s.connected || s.busy || s.transactionUncertain || source.EditID != change.ID ||
		!s.canEditGridCell(change.Row, change.Column) || key.rowID == "" || key.column == "" {
		s.setStatus("grid_inline_unavailable")
		return
	}
	old := browserCell(source, change.Row, change.Column)
	change.Null = change.Null || change.Text == ""
	if change.Null == old.Null && (change.Null || change.Text == editInputText(old)) {
		delete(s.gridDrafts, key)
	} else {
		if s.gridDrafts == nil {
			s.gridDrafts = make(map[editedCellKey]oracle.EditChange)
		}
		s.gridDrafts[key] = change
	}
	s.updateGridDraftDisplay(key.rowID)
	s.setGridDraftStatus()
}

func (s *screen) clearGridDrafts() {
	if s.gridDraftCount() == 0 {
		return
	}
	hadRows := len(s.gridInserts)+len(s.gridDeletes) != 0
	s.gridDrafts, s.gridInserts, s.gridDeletes = nil, nil, nil
	if hadRows {
		s.reprojectGridDrafts()
	} else {
		s.updateGridDraftDisplay("")
	}
}

func (s *screen) updateGridDraftDisplay(rowID string) {
	rows := make(map[int]bool)
	for row, index := range s.resultBrowser.visible {
		if rowID != "" && (index < 0 || index >= len(s.resultBrowser.source.EditRowIDs) || s.resultBrowser.source.EditRowIDs[index] != rowID) {
			continue
		}
		rows[row+1] = true
		for col, sourceCol := range s.resultBrowser.columns {
			cell := s.displayedGridCell(index, sourceCol)
			s.resultValues[row][col], s.resultRows[row][col] = cell, s.displayedGridText(index, sourceCol, cell)
			label := gridPreview(cell.Text)
			if cell.Null {
				label = "∅ NULL"
			}
			s.table.GetCell(row+1, col).SetText(tuivory.Escape(" " + label + " "))
		}
	}
	s.paintResultRows(rows)
}

func (s *screen) hasPendingWork() bool {
	return s.pendingChanges || s.gridDraftCount() != 0
}

// Keep the user's selected source row and scroll when projection changes
// because a preview row is removed or all drafts are resolved.
func (s *screen) reprojectGridDrafts() {
	logical, selected := s.selectedGridSourceRow()
	row, col := s.table.GetSelection()
	top, left := s.table.GetOffset()
	s.applyResultBrowser(true)
	if len(s.resultRows) == 0 || len(s.resultHeaders) == 0 {
		return
	}
	row = min(max(1, row), len(s.resultRows))
	if selected {
		for i, sourceRow := range s.resultBrowser.visible {
			if sourceRow == logical {
				row = i + 1
				break
			}
		}
	}
	s.table.Select(row, min(col, len(s.resultHeaders)-1))
	s.table.SetOffset(min(top, len(s.resultRows)-1), min(left, len(s.resultHeaders)-1))
}
