package ui

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

type resultBrowser struct {
	source             oracle.Result
	filter             string
	sortColumn         int
	sorted, descending bool
	visible            []int
}

func (s *screen) invalidateResultCursors(active uint64) {
	if s.resultBrowser.source.PageID != active {
		s.resultBrowser.source.PageID = 0
	}
	for _, tab := range s.tabs {
		if tab.browser.source.PageID != active {
			tab.browser.source.PageID = 0
		}
	}
}

func browserCell(result oracle.Result, row, col int) oracle.Cell {
	if row < len(result.Values) && col < len(result.Values[row]) {
		return result.Values[row][col]
	}
	if row < len(result.Rows) && col < len(result.Rows[row]) {
		return oracle.Cell{Text: result.Rows[row][col], Type: "VARCHAR2"}
	}
	return oracle.Cell{Null: true}
}

func gridNumber(cell oracle.Cell) *big.Rat {
	if !gridNumeric(cell) || cell.Null || len(cell.Text) > 256 {
		return nil
	}
	if at := strings.IndexAny(cell.Text, "eE"); at >= 0 {
		exponent, err := strconv.Atoi(cell.Text[at+1:])
		if err != nil || exponent > 1000 || exponent < -1000 {
			return nil
		}
	}
	value, _ := new(big.Rat).SetString(cell.Text)
	return value
}

func (s *screen) applyResultBrowser(preserveWidths bool) {
	view := &s.resultBrowser
	indices := make([]int, 0, len(view.source.Rows))
	filter := strings.ToLower(view.filter)
	for i, row := range view.source.Rows {
		match := filter == ""
		for _, value := range row {
			if !match && strings.Contains(strings.ToLower(value), filter) {
				match = true
				break
			}
		}
		if match {
			indices = append(indices, i)
		}
	}
	if view.sorted && view.sortColumn >= 0 && view.sortColumn < len(view.source.Columns) {
		col := view.sortColumn
		numbers := make(map[int]*big.Rat, len(indices))
		numeric := true
		for _, row := range indices {
			cell := browserCell(view.source, row, col)
			numbers[row] = gridNumber(cell)
			if !cell.Null && numbers[row] == nil {
				numeric = false
			}
		}
		sort.SliceStable(indices, func(i, j int) bool {
			a, b := browserCell(view.source, indices[i], col), browserCell(view.source, indices[j], col)
			if a.Null || b.Null {
				return !a.Null && b.Null
			}
			comparison := strings.Compare(a.Text, b.Text)
			if numeric {
				comparison = numbers[indices[i]].Cmp(numbers[indices[j]])
			}
			if view.descending {
				return comparison > 0
			}
			return comparison < 0
		})
	}
	result := view.source
	view.visible = indices
	result.Rows = nil
	result.Values = nil
	for _, row := range indices {
		result.Rows = append(result.Rows, view.source.Rows[row])
		cells := make([]oracle.Cell, len(view.source.Columns))
		for col := range cells {
			cells[col] = browserCell(view.source, row, col)
		}
		result.Values = append(result.Values, cells)
	}
	widths := append([]int(nil), s.grid.widths...)
	s.renderGrid(result)
	if preserveWidths && len(widths) == len(s.grid.widths) {
		for col, width := range widths {
			s.grid.widths[col] = width
			for row := 0; row < s.table.GetRowCount(); row++ {
				s.table.GetCell(row, col).SetMaxWidth(width)
			}
			s.refreshGridHeader(col)
		}
	}
}

func (s *screen) sortResult(descending bool) {
	_, col := s.table.GetSelection()
	s.resultBrowser.sortColumn, s.resultBrowser.sorted, s.resultBrowser.descending = col, true, descending
	s.applyResultBrowser(true)
	s.resultScopeStatus()
}

func (s *screen) resetResultView() {
	s.resultBrowser.filter, s.resultBrowser.sorted = "", false
	s.applyResultBrowser(true)
	s.resultScopeStatus()
}

func (s *screen) resultScopeStatus() {
	s.setStatus("result_scope", len(s.resultRows), len(s.resultBrowser.source.Rows))
}

func (s *screen) fetchMoreResults() {
	if s.busy || s.resultBrowser.source.PageID == 0 {
		return
	}
	s.submit(job{action: "fetch_more", pageID: s.resultBrowser.source.PageID})
}

func (s *screen) acceptResultPage(page oracle.Result, err error) {
	selected, col := s.table.GetSelection()
	logical := -1
	if selected > 0 && selected <= len(s.resultBrowser.visible) {
		logical = s.resultBrowser.visible[selected-1]
	}
	offsetRow, offsetCol := s.table.GetOffset()
	source := &s.resultBrowser.source
	for len(source.Values) < len(source.Rows) {
		row := len(source.Values)
		cells := make([]oracle.Cell, len(source.Columns))
		for col := range cells {
			cells[col] = browserCell(*source, row, col)
		}
		source.Values = append(source.Values, cells)
	}
	source.Rows = append(source.Rows, page.Rows...)
	source.Values = append(source.Values, page.Values...)
	source.PageID, source.PageLimited, source.Truncated = page.PageID, page.PageLimited, page.Truncated
	if err != nil {
		source.Truncated = true
	}
	s.applyResultBrowser(true)
	for row, index := range s.resultBrowser.visible {
		if index == logical {
			s.table.Select(row+1, col).SetOffset(max(0, offsetRow+row+1-selected), offsetCol)
			break
		}
	}
	s.resultScopeStatus()
	if page.PageLimited {
		s.setStatus("result_page_limit", oracle.MaxGridRows, oracle.MaxGridBytes/(1024*1024))
	}
	if err != nil {
		source.PageID = 0
		s.setStatus("error", err)
	}
}

func (s *screen) showResultFilter() {
	if s.busy {
		return
	}
	form := tview.NewForm().AddInputField(s.tr("result_filter_label"), s.resultBrowser.filter, 30, nil, nil)
	close := func() { s.pages.RemovePage("result_filter"); s.app.SetFocus(s.table) }
	form.AddButton(s.tr("result_filter_apply"), func() {
		s.resultBrowser.filter = form.GetFormItem(0).(*tview.InputField).GetText()
		s.applyResultBrowser(true)
		s.resultScopeStatus()
		close()
	})
	form.AddButton(s.tr("cancel"), close)
	form.SetBorder(true).SetTitle(s.tr("result_filter") + " · " + fmt.Sprintf(s.tr("result_local_rows"), len(s.resultBrowser.source.Rows)))
	s.styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	s.pages.AddPage("result_filter", s.formDialog(form, nil, 80), true, true)
	s.app.SetFocus(form)
}
