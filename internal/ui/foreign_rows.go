package ui

import (
	"errors"
	"fmt"
	"lazyora/internal/oracle"
	"strconv"

	"github.com/leechis7/tuivory/go"
)

type foreignRowView struct {
	info    *tuivory.WidgetTextView
	list    *tuivory.WidgetList
	result  *tuivory.RowsView
	headers []string
	row     []oracle.Cell
	links   []oracle.ForeignKeyLink
}

func (s *screen) showForeignKeys() {
	w := s.objectWorkspace
	if s.busy || w == nil || w.section != "data" || w.dataModel == nil || s.object.Kind != "TABLE" {
		return
	}
	row, err := strconv.Atoi(w.dataModel.RowID)
	if err != nil || row < 0 || row >= len(w.data.Values) {
		w.dataInfo.SetText(s.tr("foreign_pick_row"))
		return
	}
	v := &foreignRowView{
		info:    s.newTextView(textHint).SetText(s.tr("browser_loading")),
		list:    tuivory.NewWidgetList().ShowSecondaryText(false),
		headers: w.data.Columns, row: w.data.Values[row],
	}
	v.list.SetBorder(true).SetTitle(s.tr("foreign_keys"))
	s.styleList(v.list)
	v.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if !s.busy && index >= 0 && index < len(v.links) {
			v.info.SetText(s.tr("browser_loading"))
			s.submit(job{action: "related_rows", related: &relatedRowQuery{link: v.links[index], headers: v.headers, row: v.row}})
		}
	})
	s.foreignRows = v
	close := s.clickable(tuivory.NewWidgetButton(s.tr("close")).SetSelectedFunc(func() { s.closeForeignRows("foreign_keys") }))
	root := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(v.info, 2, 0, false).AddItem(v.list, 0, 1, true).AddItem(close, 1, 0, false)
	s.pages.AddPage("foreign_keys", s.formPopup(root, 90, 20, nil), true, true)
	s.app.SetFocus(v.list)
	if s.demo && !s.connected {
		s.acceptForeignKeys(oracle.ForeignKeyLinks{}, nil)
		return
	}
	s.submit(job{action: "foreign_keys", object: s.object})
}

type relatedRowQuery struct {
	link    oracle.ForeignKeyLink
	headers []string
	row     []oracle.Cell
}

func (s *screen) acceptForeignKeys(links oracle.ForeignKeyLinks, err error) {
	v := s.foreignRows
	if v == nil {
		return
	}
	if err != nil {
		v.info.SetText(fmt.Sprintf(s.tr("error"), err))
		return
	}
	v.links = links.Links
	hint := s.tr("foreign_choose")
	switch {
	case len(v.links) == 0:
		hint = s.tr("foreign_empty")
	case links.Limited:
		hint += " · " + s.tr("object_data_limit")
	}
	v.info.SetText(hint)
	for _, link := range v.links {
		direction := s.tr("foreign_child")
		if link.ToParent {
			direction = s.tr("foreign_parent")
		}
		v.list.AddItem(tuivory.Escape(direction+" · "+oracle.QualifiedObject(link.Target)+" · "+link.Name), "", 0, nil)
	}
}

func (s *screen) acceptRelatedRows(query *relatedRowQuery, result oracle.Result, err error) {
	v := s.foreignRows
	if v == nil {
		return
	}
	if errors.Is(err, oracle.ErrForeignKeyNull) {
		v.info.SetText(s.tr("foreign_null"))
		return
	}
	if err != nil {
		v.info.SetText(fmt.Sprintf(s.tr("error"), err))
		return
	}
	_, grid, err := s.objectPreviewGrid(result)
	if err != nil {
		v.info.SetText(fmt.Sprintf(s.tr("error"), err))
		return
	}
	v.info.SetText(s.tr("foreign_choose"))
	status := oracle.QualifiedObject(query.link.Target) + " · " + fmt.Sprintf(s.tr("object_data_rows"), len(result.Values))
	if result.Truncated {
		status += " · " + s.tr("object_data_limit")
	}
	info := s.newTextView(textHint).SetText(status)
	v.result = grid
	close := s.clickable(tuivory.NewWidgetButton(s.tr("close")).SetSelectedFunc(func() { s.closeForeignRows("related_rows") }))
	root := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(info, 2, 0, false).AddItem(grid, 0, 1, true).AddItem(close, 1, 0, false)
	s.pages.AddPage("related_rows", s.formPopup(root, 110, 24, nil), true, true)
	s.app.SetFocus(grid)
}

func (s *screen) foreignRowsKey(page string, event *tuivory.EventKey) *tuivory.EventKey {
	switch event.Key() {
	case tuivory.KeyEscape:
		s.closeForeignRows(page)
		return nil
	case tuivory.KeyTab, tuivory.KeyBacktab:
		if page == "related_rows" {
			s.dialogTab(event, s.foreignRows.result)
		} else {
			s.dialogTab(event, s.foreignRows.list)
		}
		return nil
	}
	return event
}

func (s *screen) closeForeignRows(page string) {
	if s.busy {
		s.action("cancel")
		return
	}
	s.pages.RemovePage(page)
	if page == "related_rows" {
		s.foreignRows.result = nil
		s.app.SetFocus(s.foreignRows.list)
	} else {
		s.foreignRows = nil
		s.app.SetFocus(s.objectWorkspace.dataGrid)
	}
}
