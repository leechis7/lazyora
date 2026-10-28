package ui

import (
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) browserObjectCount() int {
	if s.browser == nil || s.browser.GetRoot() == nil {
		return 0
	}
	count := 0
	for _, n := range s.browser.GetRoot().GetChildren() {
		if d, ok := n.GetReference().(*browserNode); ok && d.kind == "object" {
			count++
		}
	}
	return count
}

func (s *screen) expandBrowser(n *tview.TreeNode) {
	d, ok := n.GetReference().(*browserNode)
	if !ok || d.kind != "object" || !d.object.HasColumns() || d.loaded {
		return
	}
	if s.busy {
		n.SetExpanded(false)
		s.browser.Refresh()
		return
	}
	n.ClearChildren().AddChild(s.treeMessage("browser_loading"))
	s.browser.Refresh()
	if s.demo && !s.connected {
		s.acceptBrowserColumns(n, []oracle.Column{
			{Name: "ID", DataType: "NUMBER(10,0)", Comment: "식별자 / Identifier"},
			{Name: "TITLE", DataType: "VARCHAR2(100 CHAR)", Nullable: true},
			{Name: "AMOUNT", DataType: "NUMBER(18,4)", Nullable: true},
		}, nil)
		return
	}
	if !s.connected {
		n.SetExpanded(false)
		s.browser.Refresh()
		return
	}
	s.submit(job{action: "browser_columns", object: d.object, node: n})
}

func (s *screen) acceptBrowserColumns(n *tview.TreeNode, columns []oracle.Column, err error) {
	if n == nil {
		return
	}
	d, ok := n.GetReference().(*browserNode)
	// Cache membership uses pointer identity as well as the Oracle identity, so
	// refresh/reconnect responses cannot populate a new incarnation of a node.
	if !ok || d.object.Owner != s.browserOwner || s.browserNodes[d.object] != n {
		return
	}
	n.ClearChildren()
	if err != nil {
		n.AddChild(s.treeMessage("browser_error"))
		// Collapsing and reopening retries; failed requests are not cached.
		s.setStatus("error", err)
	} else {
		d.loaded = true
		for _, c := range columns {
			n.AddChild(s.treeNode(c.Name, &browserNode{kind: "column", object: d.object, column: c}))
		}
		if len(columns) == 0 {
			n.AddChild(s.treeMessage("columns_empty"))
		}
		s.setStatus("columns_loaded", len(columns))
	}
	s.browser.Refresh()
}
