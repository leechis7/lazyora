package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

type browserNode struct {
	kind, label string
	object      oracle.Object
	loaded      bool
}

var objectGroups = []struct{ kind, label string }{
	{"TABLE", "tables"}, {"VIEW", "views"}, {"MATERIALIZED VIEW", "materialized_views"},
	{"FUNCTION", "functions"}, {"PROCEDURE", "procedures"},
	{"PACKAGE", "packages"}, {"PACKAGE BODY", "package_bodies"},
	{"SEQUENCE", "sequences"}, {"SYNONYM", "synonyms"}, {"INDEX", "indexes"},
	{"TRIGGER", "triggers"}, {"TYPE", "types"}, {"TYPE BODY", "type_bodies"},
}

func demoObjects(owner string) []oracle.Object {
	var objects []oracle.Object
	for _, group := range objectGroups {
		name := "SAMPLE_" + strings.ReplaceAll(group.kind, " ", "_")
		if group.kind == "TABLE" {
			name = "SAMPLE_ROWS"
		}
		objects = append(objects, oracle.Object{Owner: owner, Name: name, Kind: group.kind, Status: "VALID"})
	}
	return objects
}

func (s *screen) toggleBrowser() {
	if s.browser == nil {
		return
	}
	s.browserHidden = !s.browserHidden
	wasMaximized := s.maximized != ""
	s.maximized = ""
	s.fullscreen = false
	s.rebuildPanels()
	if wasMaximized {
		s.restorePanelOffsets()
	}
	if s.browserHidden {
		if s.browserHasFocus() {
			s.app.SetFocus(s.editor)
		}
		s.hideBrowserRect()
	} else {
		s.app.SetFocus(s.browser)
	}
}

func (s *screen) newBrowser() *tview.Flex {
	s.browser = tview.NewTreeView().SetTopLevel(1).SetGraphics(false)
	s.browser.SetBorder(true)
	styleBox(s.browser.Box)
	s.browser.SetSelectedFunc(s.activateBrowser)
	s.browser.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	s.browserUsers = s.newChoice(s.tr("browser_user_label"), nil, -1, func(_ string, index int) { s.selectBrowserUser(index) })
	s.browserTypes = s.newChoice(s.tr("browser_type_label"), nil, -1, func(_ string, index int) { s.selectBrowserType(index) })
	s.browserUsers.SetFieldWidth(0)
	s.browserTypes.SetFieldWidth(0)
	s.browserForm = tview.NewForm().SetItemPadding(0).AddFormItem(s.browserUsers).AddFormItem(s.browserTypes)
	styleForm(s.browserForm)
	s.browserForm.SetBorder(true).SetTitle(s.tr("browser_title"))
	s.browserPanel = tview.NewFlex().SetDirection(tview.FlexRow).AddItem(s.browserForm, 6, 0, false).AddItem(s.browser, 0, 1, true)
	s.resetBrowser()
	return s.browserPanel
}

func (s *screen) browserHasFocus() bool {
	return s.browser != nil && (s.browser.HasFocus() || (s.browserForm != nil && s.browserForm.HasFocus()))
}

func (s *screen) showBrowserSelector(objectType bool) {
	if s.browserHidden || s.maximized != "" {
		if !s.browserHidden {
			s.browserHidden = true
		}
		s.toggleBrowser()
	}
	if objectType {
		s.app.SetFocus(s.browserTypes)
	} else {
		s.app.SetFocus(s.browserUsers)
	}
}

func (s *screen) selectBrowserType(index int) {
	if index < 0 || index >= len(objectGroups) {
		return
	}
	s.browserKind = index
	if data, ok := s.browser.GetRoot().GetReference().(*browserNode); ok && data.loaded && s.browserOwner != "" {
		s.renderBrowserObjects()
	}
}

func (s *screen) selectBrowserUser(index int) {
	if index < 0 || index >= len(s.browserSchemas) {
		return
	}
	owner := s.browserSchemas[index]
	if owner == s.browserOwner {
		return
	}
	if s.busy {
		for old, name := range s.browserSchemas {
			if name == s.browserOwner {
				s.browserUsers.SetSelectedFunc(nil).SetCurrentOption(old).SetSelectedFunc(func(_ string, i int) { s.selectBrowserUser(i) })
				break
			}
		}
		return
	}
	s.loadBrowserUser(owner)
}

func (s *screen) loadBrowserUser(owner string) {
	s.browserOwner = owner
	s.browserObjects = nil
	root := s.treeNode(owner, &browserNode{kind: "schema", object: oracle.Object{Owner: owner}}).SetSelectable(false)
	root.AddChild(s.treeMessage("browser_loading"))
	s.browser.SetRoot(root).SetCurrentNode(nil)
	if s.demo && !s.connected {
		s.populateObjects(root, demoObjects(owner))
		return
	}
	if !s.connected {
		s.resetBrowser()
		return
	}
	s.submit(job{action: "objects", object: oracle.Object{Owner: owner}, node: root})
}

func (s *screen) renderBrowserObjects() {
	root := s.browser.GetRoot()
	root.ClearChildren().SetExpanded(true)
	var first *tview.TreeNode
	for _, object := range s.browserObjects {
		if object.Kind != objectGroups[s.browserKind].kind {
			continue
		}
		node := s.treeNode(object.Name, &browserNode{kind: "object", object: object})
		root.AddChild(node)
		if first == nil {
			first = node
		}
	}
	if first == nil {
		root.AddChild(s.treeMessage("browser_empty"))
	}
	s.browser.SetCurrentNode(first)
	s.browser.SetTitle(" " + s.tr(objectGroups[s.browserKind].label) + " ")
}

func (s *screen) treeNode(text string, data *browserNode) *tview.TreeNode {
	return tview.NewTreeNode(tview.Escape(text)).SetReference(data).
		SetTextStyle(tcell.StyleDefault.Foreground(palette.text).Background(palette.background)).
		SetSelectedTextStyle(focusStyle())
}

func (s *screen) treeMessage(key string) *tview.TreeNode {
	return s.treeNode(s.tr(key), &browserNode{kind: "message", label: key}).SetSelectable(false)
}

func (s *screen) resetBrowser() {
	root := s.treeNode(s.tr("schemas"), &browserNode{kind: "root", label: "schemas"}).SetSelectable(false)
	root.AddChild(s.treeMessage("browser_connect"))
	if s.connected {
		root.SetChildren([]*tview.TreeNode{s.treeMessage("browser_retry")})
	}
	s.browser.SetRoot(root).SetCurrentNode(nil)
	s.browserOwner = ""
	s.browserSchemas = nil
	s.browserObjects = nil
	s.browserKind = 0
	s.browserUsers.SetSelectedFunc(nil).SetOptions(nil, nil).SetCurrentOption(-1).SetSelectedFunc(func(_ string, i int) { s.selectBrowserUser(i) })
	s.localizeBrowser()
}

func (s *screen) refreshBrowser() {
	if s.busy {
		return
	}
	if s.demo && !s.connected {
		s.showDemoBrowser()
		return
	}
	s.submit(job{action: "schemas"})
}

func (s *screen) populateSchemas(catalog oracle.Catalog) {
	preferred := s.browserOwner
	if preferred == "" {
		preferred = catalog.Current
	}
	s.browserSchemas = append([]string(nil), catalog.Schemas...)
	index := -1
	var labels []string
	for i, owner := range s.browserSchemas {
		labels = append(labels, tview.Escape(owner))
		if owner == preferred {
			index = i
		}
	}
	if index < 0 && len(labels) > 0 {
		index = 0
	}
	s.browserUsers.SetSelectedFunc(nil).SetOptions(labels, nil).SetCurrentOption(index).SetSelectedFunc(func(_ string, i int) { s.selectBrowserUser(i) })
	if index >= 0 {
		s.loadBrowserUser(s.browserSchemas[index])
	} else {
		s.browserOwner = ""
		s.browserObjects = nil
		s.browser.GetRoot().ClearChildren().AddChild(s.treeMessage("browser_empty"))
		s.browser.SetCurrentNode(nil)
	}
}

func (s *screen) populateObjects(node *tview.TreeNode, objects []oracle.Object) {
	if node.GetReference().(*browserNode).object.Owner != s.browserOwner {
		return
	}
	node.GetReference().(*browserNode).loaded = true
	s.browserObjects = objects
	s.renderBrowserObjects()
}

func (s *screen) activateBrowser(node *tview.TreeNode) {
	if data, ok := node.GetReference().(*browserNode); ok && data.kind == "object" {
		s.openObject(data.object)
	}
}

func (s *screen) showDemoBrowser() {
	s.populateSchemas(oracle.Catalog{Current: "LAZYORA", Schemas: []string{"LAZYORA"}})
}

func (s *screen) localizeBrowser() {
	if s.browser == nil {
		return
	}
	s.browserForm.SetTitle(s.tr("browser_title"))
	s.browserUsers.SetLabel(s.tr("browser_user_label"))
	s.browserTypes.SetLabel(s.tr("browser_type_label"))
	var types []string
	for _, group := range objectGroups {
		types = append(types, s.tr(group.label))
	}
	s.browserTypes.SetSelectedFunc(nil).SetOptions(types, nil).SetCurrentOption(s.browserKind).SetSelectedFunc(func(_ string, i int) { s.selectBrowserType(i) })
	s.browser.SetTitle(" " + s.tr(objectGroups[s.browserKind].label) + " ")
	s.browser.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {
		if data, ok := node.GetReference().(*browserNode); ok && data.label != "" {
			node.SetText(tview.Escape(s.tr(data.label)))
		}
		return true
	})
	if s.objectTable != nil {
		s.renderObject(s.objectColumns)
	}
}

func (s *screen) openObject(object oracle.Object) {
	if page, _ := s.pages.GetFrontPage(); page != "object" {
		s.objectFocus = s.app.GetFocus()
	}

	if s.busy {
		return
	}
	s.object = object
	s.objectColumns = nil
	s.objectInfo = tview.NewTextView().SetWrap(true)
	s.objectInfo.SetBackgroundColor(palette.background)
	s.objectInfo.SetTextColor(palette.text)
	s.objectTable = tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).SetSelectedStyle(focusStyle())
	s.objectTable.SetBorder(true)
	styleBox(s.objectTable.Box)
	s.renderObject(nil)
	close := clickable(tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeObject))
	buttons := tview.NewFlex()
	if object.HasSource() {
		buttons.AddItem(clickable(tview.NewButton(s.tr("source_button")).SetSelectedFunc(s.showSource)), 0, 1, false)
	}
	if object.HasArguments() {
		buttons.AddItem(clickable(tview.NewButton(s.tr("arguments_button")).SetSelectedFunc(s.showArguments)), 0, 1, false)
		buttons.AddItem(clickable(tview.NewButton(s.tr("routine_button")).SetSelectedFunc(func() { s.showRoutine(s.object, "") })), 0, 1, false)
	}
	buttons.AddItem(close, 0, 1, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.objectInfo, 2, 0, false).AddItem(s.objectTable, 0, 1, true).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("object", root, true, true)
	s.app.SetFocus(s.objectTable)
	if !object.HasColumns() {
		if !s.demo || s.connected {
			s.submit(job{action: "object_info", object: object})
		}
		return
	}
	if s.demo && !s.connected {
		s.renderObject([]oracle.Column{{Name: "ID", DataType: "NUMBER(10,0)", Comment: "식별자 / Identifier"}, {Name: "TITLE", DataType: "VARCHAR2(100 CHAR)"}, {Name: "AMOUNT", DataType: "NUMBER(18,4)", Nullable: true}})
		return
	}
	s.objectInfo.SetText(s.objectInfo.GetText(false) + "\n" + s.tr("browser_loading"))
	s.submit(job{action: "columns", object: object})
}

func (s *screen) renderObject(columns []oracle.Column) {
	s.objectColumns = columns
	s.objectInfo.SetText(fmt.Sprintf("%s: %s    %s: %s    %s: %s", s.tr("owner"), s.object.Owner, s.tr("object_name"), s.object.Name, s.tr("object_type"), s.object.Kind))
	s.objectTable.SetTitle(s.tr("columns_title"))
	s.objectTable.Clear()
	if !s.object.HasColumns() {
		s.objectTable.SetTitle(s.tr("object_details"))
		for i, key := range []string{"property", "value"} {
			s.objectTable.SetCell(0, i, tview.NewTableCell(s.tr(key)).SetTextColor(palette.accent).SetSelectable(false))
		}
		for row, pair := range [][2]string{{s.tr("owner"), s.object.Owner}, {s.tr("object_name"), s.object.Name}, {s.tr("object_type"), s.object.Kind}, {s.tr("object_status"), s.object.Status}} {
			for col, value := range pair {
				s.objectTable.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(palette.text))
			}
		}
		s.objectInfo.SetText(s.objectInfo.GetText(false) + "\n" + s.tr("object_info_only"))
		return
	}
	for i, key := range []string{"column_position", "column_name", "data_type", "nullable", "column_comment"} {
		s.objectTable.SetCell(0, i, tview.NewTableCell(s.tr(key)).SetStyle(tcell.StyleDefault.Foreground(palette.accent).Background(palette.surface).Bold(true)).SetSelectable(false))
	}
	for row, column := range columns {
		nullable := "no"
		if column.Nullable {
			nullable = "yes"
		}
		for col, value := range []string{fmt.Sprint(row + 1), column.Name, column.DataType, s.tr(nullable), column.Comment} {
			s.objectTable.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(palette.text))
		}
	}
}

func (s *screen) closeObject() {
	s.pages.RemovePage("object")
	if s.objectFocus != nil {
		s.app.SetFocus(s.objectFocus)
	} else {
		s.app.SetFocus(s.browser)
	}
}
