package ui

import (
	"slices"

	"lazyora/internal/oracle"
)

type gridRefreshState struct {
	action                            string
	row, column, offsetRow, offsetCol int
	rowID                             string
	headers                           []string
	widths                            []int
	filter                            string
	sortColumn                        int
	sorted, descending                bool
}

// Refresh only the current result's saved recipe. Editor text may already
// contain another statement, and must never be executed by commit/rollback.
func (s *screen) refreshTransactionGrid(action string) bool {
	if s.quitting || !s.connected || s.busy || s.transactionUncertain || len(s.resultHeaders) == 0 {
		return false
	}
	source := s.resultBrowser.source
	j := job{action: "result_refresh"}
	if source.Query != nil {
		var err error
		j.query, err = oracle.NewQuerySpec(source.Query.Text, source.Query.Bindings)
		if err != nil {
			return false
		}
	} else if source.EditObject.Kind == "TABLE" && source.EditColumns == nil {
		// Explicit browser table loads have no SELECT replay recipe. An expired
		// direct SELECT must not turn into a broader whole-table query.
		j.object = source.EditObject
	} else {
		return false
	}
	row, col := s.table.GetSelection()
	offsetRow, offsetCol := s.table.GetOffset()
	view := s.resultBrowser
	state := &gridRefreshState{action: action, row: row, column: col, offsetRow: offsetRow, offsetCol: offsetCol,
		headers: append([]string(nil), s.resultHeaders...), widths: append([]int(nil), s.grid.widths...),
		filter: view.filter, sortColumn: view.sortColumn, sorted: view.sorted, descending: view.descending}
	if row > 0 && row <= len(view.visible) {
		index := view.visible[row-1]
		if index < len(source.EditRowIDs) {
			state.rowID = source.EditRowIDs[index]
		}
	}
	j.gridRefresh = state
	s.submit(j)
	s.setStatus(action + "_refreshing")
	return true
}

func (s *screen) acceptTransactionGridRefresh(j job, result oracle.Result, err error) {
	state := j.gridRefresh
	if err != nil {
		s.setStatus(state.action+"_refresh_failed", err)
		return
	}
	s.resultBrowser = resultBrowser{source: result, filter: state.filter, sortColumn: state.sortColumn,
		sorted: state.sorted, descending: state.descending}
	s.applyResultBrowser(false)
	if slices.Equal(state.headers, s.resultHeaders) && len(state.widths) == len(s.grid.widths) {
		for col, width := range state.widths {
			s.grid.widths[col] = width
			for row := range s.table.GetRowCount() {
				s.table.GetCell(row, col).SetMaxWidth(width)
			}
			s.refreshGridHeader(col)
		}
	}
	if len(s.resultRows) > 0 && len(s.resultHeaders) > 0 {
		row := min(max(1, state.row), len(s.resultRows))
		if state.rowID != "" {
			for index, sourceRow := range s.resultBrowser.visible {
				if sourceRow < len(result.EditRowIDs) && result.EditRowIDs[sourceRow] == state.rowID {
					row = index + 1
					break
				}
			}
		}
		s.table.Select(row, min(max(0, state.column), len(s.resultHeaders)-1))
		s.table.SetOffset(min(state.offsetRow, max(0, len(s.resultRows)-1)), min(state.offsetCol, max(0, len(s.resultHeaders)-1)))
	}
	s.repaintEditedCells()
	s.setStatus(state.action + "_refreshed")
}
