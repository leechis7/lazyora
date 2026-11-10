package ui

import (
	"strconv"

	"github.com/leechis7/tuivory/go"
)

// The compatibility Table stores existing application metadata/selection only;
// it is never used to draw the result body. GridSource reads the original rows
// lazily, so switching to GridView does not duplicate a full result dataset.
type resultPresentation struct {
	model   *tuivory.Grid
	view    *tuivory.RowsView
	pointer bool
}
type resultSource struct{ owner *screen }

func (r resultSource) Len() int { return len(r.owner.resultRows) }
func (r resultSource) IndexOf(id string) int {
	n, err := strconv.Atoi(id)
	if err != nil || n < 1 || n > r.Len() {
		return -1
	}
	return n - 1
}
func (r resultSource) Row(n int) tuivory.GridRecord {
	if n < 0 || n >= r.Len() {
		return tuivory.GridRecord{}
	}
	cells := make(map[string]string, len(r.owner.resultHeaders))
	for c, v := range r.owner.resultRows[n] {
		cells[strconv.Itoa(c)] = v
	}
	return tuivory.GridRecord{ID: strconv.Itoa(n + 1), Cells: cells}
}
func (s *screen) prepareResultGrid() {
	p := s.grid.presentation
	if p == nil {
		columns := make([]tuivory.GridColumn, len(s.resultHeaders))
		for c, label := range s.resultHeaders {
			if s.resultBrowser.sorted && s.resultBrowser.sortColumn == c {
				arrow := "↑"
				if s.resultBrowser.descending {
					arrow = "↓"
				}
				label = arrow + " " + label
			}
			columns[c] = tuivory.GridColumn{ID: strconv.Itoa(c), Label: label, Width: max(1, s.grid.widths[c]-2), Editable: true}
		}
		model, err := tuivory.NewGrid(columns, resultSource{s})
		if err != nil {
			panic(err)
		}
		view, err := tuivory.NewGridView(model, s.resultPalette(), s.resultGridEvent)
		if err != nil {
			panic(err)
		}
		model.SetEditOptions(tuivory.GridEditOptions{Enabled: true, AllowNull: true,
			Prepare: func(row tuivory.GridRecord, col tuivory.GridColumn) (tuivory.GridValue, bool) {
				r, errR := strconv.Atoi(row.ID)
				c, errC := strconv.Atoi(col.ID)
				if errR != nil || errC != nil || !s.connected || s.busy || s.transactionUncertain ||
					r < 1 || r > len(s.resultBrowser.visible) || c < 0 || c >= len(s.resultHeaders) {
					return tuivory.GridValue{}, false
				}
				index, sourceCol := s.resultBrowser.visible[r-1], s.resultSourceColumn(c)
				if !s.resultBrowser.source.CanEditColumn(sourceCol) {
					return tuivory.GridValue{}, false
				}
				cell := s.displayedGridCell(index, sourceCol)
				return tuivory.GridValue{Text: editInputText(cell), Null: cell.Null}, true
			}})
		p = &resultPresentation{model: model, view: view}
		s.grid.presentation = p
		model.SetCellInfo(func(row tuivory.GridRecord, col tuivory.GridColumn) tuivory.GridCellInfo {
			r, _ := strconv.Atoi(row.ID)
			c, _ := strconv.Atoi(col.ID)
			info := tuivory.GridCellInfo{Display: gridPreview(row.Cells[col.ID]), HasDisplay: true}
			if r > 0 && r <= len(s.resultValues) && c < len(s.resultValues[r-1]) {
				cell := s.resultValues[r-1][c]
				info.Null = cell.Null
				info.Numeric = gridNumeric(cell)
				if cell.Null {
					info.Display = "∅ NULL"
				}
			}
			info.State.Edited = s.isEditedCell(r, c)
			return info
		})
		// Published v0.11.0 separates the aggregate gutter marker from cell
		// colors. Supply identities; the SDK owns marker rendering and styling.
		model.SetRowEdited(func(row tuivory.GridRecord) bool {
			r, _ := strconv.Atoi(row.ID)
			for c := range s.resultHeaders {
				if s.isEditedCell(r, c) {
					return true
				}
			}
			return false
		})
	}
	model := p.model
	options, _ := tuivory.GridPreset("data")
	options.Striped = true
	options.ShowCellMarkers = false
	options.ASCII = s.asciiDisplay()
	// Host keymaps and transaction actions take priority over generic demo keys.
	options.DisableHeaderSort = true
	options.DisableEditing = false
	// Preserve the product's richer localized status text, rendered by StatusLine.
	options.StatusFields = []string{"meta"}
	_ = model.SetOptions(options)
	model.Status.Metadata = s.gridStatusText()
	model.Labels = map[string]string{"editing": s.tr("grid_inline_editing")}
	_ = p.view.SetPalette(s.resultPalette())
	row, col := s.table.GetSelection()
	model.Focus(strconv.Itoa(row), strconv.Itoa(col))
	model.Top, model.Left = s.table.GetOffset()
	for c, width := range s.grid.widths {
		id := strconv.Itoa(c)
		model.ResizeColumn(id, max(1, width-2))
		_ = model.SetColumnOptions(id, tuivory.GridColumnOptions{Align: "numeric", Overflow: "ellipsis", MaxWidth: 160})
	}
	if s.resultSelection.active {
		a := s.resultSelection
		model.SelectRange(tuivory.GridRange{AnchorRow: strconv.Itoa(a.anchorRow), AnchorColumn: strconv.Itoa(a.anchorCol), EndRow: strconv.Itoa(a.endRow), EndColumn: strconv.Itoa(a.endCol)})
	} else {
		if _, editing := model.EditSession(); !editing {
			model.Handle("cancel", 1)
		}
		_ = model.SetSelectionMode("cell")
	}
}
func (s *screen) resultGridEvent(e tuivory.GridEvent) {
	if e.Kind == "" {
		return
	}
	p := s.grid.presentation
	model := p.model
	s.table.Select(model.Row+1, model.Column)
	_, _, w, h := s.table.GetInnerRect()
	geo := model.Geometry(w, h)
	s.table.SetOffset(geo.Top, geo.Left)
	if model.SelectionMode() == "range" {
		a := model.SelectedRange()
		ar, _ := strconv.Atoi(a.AnchorRow)
		ac, _ := strconv.Atoi(a.AnchorColumn)
		er, _ := strconv.Atoi(a.EndRow)
		ec, _ := strconv.Atoi(a.EndColumn)
		s.resultSelection = resultSelection{anchorRow: ar, anchorCol: ac, endRow: er, endCol: ec, active: ar > 0 && er > 0}
	} else if model.SelectionMode() == "row" {
		s.resultSelection = resultSelection{anchorRow: model.Row + 1, endRow: model.Row + 1, anchorCol: 0, endCol: len(s.resultHeaders) - 1, active: true}
	} else {
		s.resultSelection = resultSelection{}
	}
	if e.Kind == "resize" {
		for c, column := range model.Columns() {
			s.grid.widths[c] = column.Width + 2
		}
	}
	s.highlightResults()
	s.refreshResultPosition()
}
func (s *screen) resultGridKey(e *tuivory.EventKey) *tuivory.EventKey {
	action := ""
	switch e.Key() {
	case tuivory.KeyUp, tuivory.KeyDown, tuivory.KeyLeft, tuivory.KeyRight, tuivory.KeyHome, tuivory.KeyEnd, tuivory.KeyPgUp, tuivory.KeyPgDn:
		action = tuivory.GridKey(e)
	}
	if action == "" {
		return nil
	} // Global keys/actions are handled by screen.key.
	s.prepareResultGrid()
	_, _, _, h := s.table.GetInnerRect()
	s.resultGridEvent(s.grid.presentation.model.Handle(action, h))
	return nil
}

// Cell coordinates come from the same public geometry that paints the grid.
func (s *screen) resultCellRect(row, col int) (int, int, int) {
	if s.grid.presentation == nil {
		return -1, -1, 0
	}
	x, y, w, h := s.table.GetInnerRect()
	geo := s.grid.presentation.model.Geometry(w, h)
	py := y
	if row > 0 {
		py = y + geo.HeaderRows + (row-1-geo.Top)*geo.Stride
		if row-1 < geo.Top || py >= y+h-geo.FooterRows {
			return -1, -1, 0
		}
	}
	for _, span := range geo.Columns {
		if span.ID == strconv.Itoa(col) {
			return x + span.Start, py, span.Width
		}
	}
	return -1, -1, 0
}
func (s *screen) resultCellAt(x, y int) (int, int) {
	if s.grid.presentation == nil {
		return -1, -1
	}
	left, top, w, h := s.table.GetInnerRect()
	geo := s.grid.presentation.model.Geometry(w, h)
	if x < left || y < top || x >= left+w || y >= top+h-geo.FooterRows {
		return -1, -1
	}
	row := 0
	if y >= top+geo.HeaderRows {
		row = geo.Top + (y-top-geo.HeaderRows)/geo.Stride + 1
		if row > len(s.resultRows) {
			return -1, -1
		}
	}
	for _, span := range geo.Columns {
		if x >= left+span.Start && x < left+span.Start+span.Width {
			c, _ := strconv.Atoi(span.ID)
			return row, c
		}
	}
	return row, -1
}
