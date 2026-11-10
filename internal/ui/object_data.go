package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"strconv"

	"github.com/leechis7/tuivory/go"
)

const objectPreviewStep = 100

type objectWorkspace struct {
	right, body   *tuivory.Flex
	tabs          *tuivory.TabView
	tabState      *tuivory.TabState
	split         *tuivory.SplitView
	dataInfo, ddl *tuivory.WidgetTextView
	dataGrid      *tuivory.RowsView
	buttons       *tuivory.Flex
	data          oracle.Result
	rows          int
	section       string
	ddlLoaded     bool
}

func (s *screen) objectTabs() *tuivory.Tabs {
	tabs, err := tuivory.NewTabs([]tuivory.TabItem{
		{ID: "columns", Title: s.tr("object_tab_columns")},
		{ID: "data", Title: s.tr("object_tab_data")},
		{ID: "ddl", Title: s.tr("ddl")},
	})
	if err != nil {
		panic(err)
	}
	return tabs
}

func (s *screen) makeObjectWorkspace(buttons *tuivory.Flex) tuivory.Primitive {
	w := &objectWorkspace{
		tabState: &tuivory.TabState{Active: "columns", Focused: "columns"},
		body:     s.newFlex().SetDirection(tuivory.FlexRow),
		dataInfo: s.newTextView(textHint).SetDynamicColors(false),
		ddl:      s.newTextView().SetDynamicColors(false).SetScrollable(true).SetWrap(false),
		buttons:  buttons,
		section:  "columns",
	}
	s.objectWorkspace = w
	w.ddl.SetBorder(true).SetTitle(s.tr("ddl"))
	s.styleBox(w.ddl.Box)
	w.tabs, _ = tuivory.NewTabView(s.objectTabs(), w.tabState, s.componentPalette())
	w.tabs.ASCII = s.asciiDisplay()
	w.tabs.OnEvent = func(e tuivory.TabEvent) {
		if e.Kind == "activate" {
			s.selectObjectTab(e.ID)
		}
	}
	w.right = s.newFlex().SetDirection(tuivory.FlexRow).AddItem(s.objectInfo, 2, 0, false).
		AddItem(w.tabs, 1, 0, false).AddItem(w.body, 0, 1, true).AddItem(buttons, 1, 0, false)
	w.body.AddItem(s.objectTable, 0, 1, true)
	if s.browserPanel == nil {
		return w.right
	}
	if s.objectSplit.Preferred == 0 {
		s.objectSplit.Preferred = 30
	}
	w.split, _ = tuivory.NewSplitView(s.browserPanel, w.right, &s.objectSplit, tuivory.SplitSpec{FirstMin: 16, SecondMin: 24}, s.componentPalette())
	return w.split
}

func (s *screen) selectObjectTab(section string) {
	w := s.objectWorkspace
	if w == nil {
		return
	}
	if s.busy {
		w.tabState.Active, w.tabState.Focused = w.section, w.section
		return
	}
	w.section = section
	w.tabState.Active, w.tabState.Focused = section, section
	w.body.Clear()
	s.app.SetFocus(w.tabs)
	switch section {
	case "data":
		w.body.AddItem(w.dataInfo, 2, 0, false)
		if w.dataGrid != nil {
			w.body.AddItem(w.dataGrid, 0, 1, false)
		}
		if w.rows == 0 {
			s.loadObjectData(objectPreviewStep)
		}
	case "ddl":
		w.body.AddItem(w.ddl, 0, 1, false)
		if !w.ddlLoaded {
			s.loadObjectDDL()
		}
	default:
		w.body.AddItem(s.objectTable, 0, 1, false)
	}
}

func (s *screen) loadObjectDDL() {
	w := s.objectWorkspace
	if w == nil || s.busy {
		return
	}
	w.ddl.SetText(s.tr("browser_loading"))
	if s.demo && !s.connected {
		if s.object.Kind == "TABLE" {
			s.acceptObjectDDL("CREATE TABLE "+oracle.QualifiedObject(s.object)+" (\"ID\" NUMBER);", nil)
		} else {
			s.acceptObjectDDL("CREATE "+s.object.Kind+" "+oracle.QualifiedObject(s.object)+" AS SELECT 1 AS \"ID\" FROM DUAL;", nil)
		}
		return
	}
	s.submit(job{action: "object_ddl", object: s.object})
}

func (s *screen) acceptObjectDDL(ddl string, err error) {
	w := s.objectWorkspace
	if w == nil {
		return
	}
	w.ddlLoaded = err == nil
	if err != nil {
		w.ddl.SetText(fmt.Sprintf(s.tr("error"), err))
		return
	}
	w.ddl.SetText(ddl).ScrollToBeginning()
}

func (s *screen) loadObjectData(limit int) {
	w := s.objectWorkspace
	if w == nil || s.busy {
		return
	}
	w.dataInfo.SetText(s.tr("browser_loading"))
	if s.demo && !s.connected {
		s.acceptObjectData(oracle.Result{
			Columns: []string{"ID", "TITLE", "AMOUNT"},
			Values: [][]oracle.Cell{
				{{Text: "1", Type: "NUMBER"}, {Text: "First row"}, {Text: "1250", Type: "NUMBER"}},
				{{Text: "2", Type: "NUMBER"}, {Text: "Second row"}, {Null: true}},
			},
		}, limit, nil)
		return
	}
	s.submit(job{action: "object_data", object: s.object, previewLimit: limit})
}

func (s *screen) acceptObjectData(result oracle.Result, limit int, err error) {
	w := s.objectWorkspace
	if w == nil {
		return
	}
	if err != nil {
		w.dataInfo.SetText(fmt.Sprintf(s.tr("error"), err))
		return
	}
	columns := make([]tuivory.GridColumn, len(result.Columns))
	for i, label := range result.Columns {
		columns[i] = tuivory.GridColumn{ID: strconv.Itoa(i), Label: label, Width: 18}
	}
	records := make([]tuivory.GridRecord, len(result.Values))
	for i, values := range result.Values {
		record := tuivory.GridRecord{ID: strconv.Itoa(i), Cells: make(map[string]string, len(values))}
		for j, cell := range values {
			value := cell.Text
			if cell.Null {
				value = "∅ NULL"
			} else if cell.Partial {
				value = gridPreview(value) + " …"
			}
			record.Cells[columns[j].ID] = gridPreview(value)
		}
		records[i] = record
	}
	source, _ := tuivory.NewMemoryRows(records)
	model, err := tuivory.NewGrid(columns, source)
	if err != nil {
		w.dataInfo.SetText(fmt.Sprintf(s.tr("error"), err))
		return
	}
	options, _ := tuivory.GridPreset("data")
	options.DisableEditing, options.DisableHeaderSort = true, true
	options.ASCII, options.StatusBar = s.asciiDisplay(), false
	_ = model.SetOptions(options)
	view, _ := tuivory.NewGridView(model, s.resultPalette(), nil)
	focused := w.dataGrid != nil && w.dataGrid.HasFocus()
	w.data, w.rows, w.dataGrid = result, limit, view
	info := fmt.Sprintf(s.tr("object_data_rows"), len(result.Values))
	if result.Truncated {
		if result.PageLimited || limit >= oracle.MaxObjectPreviewRows {
			info += " · " + s.tr("object_data_limit")
		} else {
			info += " · " + s.tr("object_data_more")
		}
	}
	w.dataInfo.SetText(info)
	if w.tabState.Active == "data" {
		w.body.Clear().AddItem(w.dataInfo, 2, 0, false).AddItem(view, 0, 1, false)
		if focused {
			s.app.SetFocus(view)
		}
	}
}

func (s *screen) objectWorkspaceKey(event *tuivory.EventKey) bool {
	w := s.objectWorkspace
	if w == nil || s.busy {
		return false
	}
	if event.Modifiers() == tuivory.ModAlt && event.Key() == tuivory.KeyRune {
		switch event.Rune() {
		case 'v', 'V':
			s.selectObjectTab("data")
			return true
		}
	}
	switch event.Key() {
	case tuivory.KeyCtrlR:
		switch w.tabState.Active {
		case "data":
			s.loadObjectData(max(objectPreviewStep, w.rows))
			return true
		case "ddl":
			s.loadObjectDDL()
			return true
		default:
			s.openObjectDetails(s.object, true)
			return true
		}
	case tuivory.KeyCtrlN:
		if w.tabState.Active == "data" && w.data.Truncated && !w.data.PageLimited && w.rows < oracle.MaxObjectPreviewRows {
			s.loadObjectData(min(w.rows+objectPreviewStep, oracle.MaxObjectPreviewRows))
			return true
		}
	case tuivory.KeyCtrlC:
		if w.ddl.HasFocus() && w.ddlLoaded {
			s.sendClipboard(w.ddl.GetText(false))
			return true
		}
	}
	return false
}

func (s *screen) cycleObjectWorkspace(event *tuivory.EventKey) bool {
	w := s.objectWorkspace
	if w == nil {
		return false
	}
	var items []tuivory.Primitive
	if w.split != nil && s.browser != nil {
		_, _, width, _ := s.browserPanel.GetRect()
		if width > 0 {
			items = append(items, s.browserUsers, s.browserTypes, s.browserSearch, s.browser)
		}
	}
	items = append(items, w.tabs)
	switch w.tabState.Active {
	case "data":
		if w.dataGrid != nil {
			items = append(items, w.dataGrid)
		}
	case "ddl":
		items = append(items, w.ddl)
	default:
		items = append(items, s.objectTable)
	}
	for i := 0; i < w.buttons.GetItemCount(); i++ {
		items = append(items, w.buttons.GetItem(i))
	}
	step := 1
	if event.Key() == tuivory.KeyBacktab {
		step = len(items) - 1
	}
	for i, item := range items {
		if item.HasFocus() {
			s.app.SetFocus(items[(i+step)%len(items)])
			return true
		}
	}
	s.app.SetFocus(w.tabs)
	return true
}
