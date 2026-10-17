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
		if s.browser.HasFocus() {
			s.app.SetFocus(s.editor)
		}
		s.browser.SetRect(0, 0, 0, 0)
	} else {
		s.app.SetFocus(s.browser)
	}
}

func (s *screen) newBrowser() *tview.TreeView {
	s.browser = tview.NewTreeView().SetTopLevel(1).SetGraphicsColor(palette.muted)
	s.browser.SetBorder(true)
	styleBox(s.browser.Box)
	s.browser.SetSelectedFunc(s.activateBrowser)
	s.browser.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	s.resetBrowser()
	return s.browser
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
}

func (s *screen) refreshBrowser() {
	if s.busy {
		return
	}
	if node := s.browser.GetCurrentNode(); node != nil {
		if data, ok := node.GetReference().(*browserNode); ok && data.object.Owner != "" {
			s.browserOwner = data.object.Owner
		}
	}
	if s.demo && !s.connected {
		s.showDemoBrowser()
		return
	}
	s.submit(job{action: "schemas"})
}

func (s *screen) populateSchemas(catalog oracle.Catalog) {
	root := s.treeNode(s.tr("schemas"), &browserNode{kind: "root", label: "schemas"}).SetSelectable(false)
	preferred := s.browserOwner
	if preferred == "" {
		preferred = catalog.Current
	}
	var selected *tview.TreeNode
	for _, owner := range catalog.Schemas {
		node := s.treeNode(owner, &browserNode{kind: "schema", object: oracle.Object{Owner: owner}}).SetExpanded(false)
		node.AddChild(s.treeMessage("browser_expand"))
		root.AddChild(node)
		if owner == preferred {
			selected = node
		}
	}
	if selected == nil && len(root.GetChildren()) > 0 {
		selected = root.GetChildren()[0]
	}
	if selected == nil {
		root.AddChild(s.treeMessage("browser_empty"))
	}
	s.browser.SetRoot(root).SetCurrentNode(selected)
	if selected != nil {
		s.loadSchema(selected)
	}
}

func (s *screen) loadSchema(node *tview.TreeNode) {
	if s.busy {
		return
	}
	data, ok := node.GetReference().(*browserNode)
	if !ok || data.kind != "schema" {
		return
	}
	s.browserOwner = data.object.Owner
	if s.demo && !s.connected {
		s.populateObjects(node, demoObjects(data.object.Owner))
		return
	}
	if !s.connected {
		s.setStatus("connect_first")
		return
	}
	s.submit(job{action: "objects", object: data.object, node: node})
}

func (s *screen) populateObjects(node *tview.TreeNode, objects []oracle.Object) {
	node.ClearChildren().SetExpanded(true)
	data := node.GetReference().(*browserNode)
	data.loaded = true
	for _, category := range objectGroups {
		kind, label := category.kind, category.label
		group := s.treeNode(s.tr(label), &browserNode{kind: "group", label: label, object: data.object, loaded: true})
		for _, object := range objects {
			if object.Kind == kind {
				group.AddChild(s.treeNode(object.Name, &browserNode{kind: "object", object: object}))
			}
		}
		if len(group.GetChildren()) == 0 {
			group.AddChild(s.treeMessage("browser_empty"))
		}
		node.AddChild(group)
	}
}

func (s *screen) activateBrowser(node *tview.TreeNode) {
	data, ok := node.GetReference().(*browserNode)
	if !ok {
		return
	}
	switch data.kind {
	case "schema":
		if !data.loaded {
			s.loadSchema(node)
			return
		}
		s.browserOwner = data.object.Owner
		node.SetExpanded(!node.IsExpanded())
	case "group":
		node.SetExpanded(!node.IsExpanded())
	case "object":
		s.openObject(data.object)
	}
}

func (s *screen) browserArrow(key tcell.Key) {
	node := s.browser.GetCurrentNode()
	if node == nil {
		return
	}
	data, ok := node.GetReference().(*browserNode)
	if !ok {
		return
	}
	if key == tcell.KeyRight {
		if data.kind == "schema" && !data.loaded {
			s.loadSchema(node)
		} else {
			node.SetExpanded(true)
		}
		return
	}
	if node.IsExpanded() && len(node.GetChildren()) > 0 {
		node.SetExpanded(false)
	} else {
		path := s.browser.GetPath(node)
		if len(path) > 2 {
			s.browser.SetCurrentNode(path[len(path)-2])
		}
	}
}

func (s *screen) showDemoBrowser() {
	s.populateSchemas(oracle.Catalog{Current: "LAZYORA", Schemas: []string{"LAZYORA"}})
}

func (s *screen) localizeBrowser() {
	if s.browser == nil {
		return
	}
	s.browser.SetTitle(s.tr("browser_title"))
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
