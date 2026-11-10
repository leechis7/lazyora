package ui

import (
	"lazyora/internal/oracle"

	"github.com/leechis7/tuivory/go"
)

func (s *screen) displayedGridCell(row, col int) oracle.Cell {
	source := s.resultBrowser.source
	cell := browserCell(source, row, col)
	if change, ok := s.gridDrafts[s.editedCellIdentity(source, row, col)]; ok {
		cell.Text, cell.Null = change.Text, change.Null
		if cell.Null {
			cell.Text = "NULL"
		}
	}
	return cell
}

func (s *screen) displayedGridText(row, col int, cell oracle.Cell) string {
	source := s.resultBrowser.source
	if _, draft := s.gridDrafts[s.editedCellIdentity(source, row, col)]; !draft && row < len(source.Rows) && col < len(source.Rows[row]) {
		return source.Rows[row][col]
	}
	return cell.Text
}

func (s *screen) stageGridChange(change oracle.EditChange) {
	source := s.resultBrowser.source
	key := s.editedCellIdentity(source, change.Row, change.Column)
	if !s.connected || s.busy || s.transactionUncertain || source.EditID != change.ID ||
		!source.CanEditColumn(change.Column) || key.rowID == "" || key.column == "" {
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
	s.setStatus("grid_drafts_count", len(s.gridDrafts))
}

func (s *screen) clearGridDrafts() {
	if len(s.gridDrafts) == 0 {
		return
	}
	s.gridDrafts = nil
	s.updateGridDraftDisplay("")
}

func (s *screen) updateGridDraftDisplay(rowID string) {
	rows := make(map[int]bool)
	for row, index := range s.resultBrowser.visible {
		if rowID != "" && (index >= len(s.resultBrowser.source.EditRowIDs) || s.resultBrowser.source.EditRowIDs[index] != rowID) {
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
	return s.pendingChanges || len(s.gridDrafts) != 0
}
