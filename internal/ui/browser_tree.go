package ui

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
	"github.com/rivo/tview"
)

// browserTree retains Oracle node references for asynchronous jobs and delegates
// rendering and navigation to the published Tuivory component.
type browserTree struct {
	*tuivory.TreeView
	screen *screen
	root   *tview.TreeNode
	state  tuivory.TreeState
	nodes  map[string]*tview.TreeNode
	model  *tuivory.Tree
}

func (s *screen) newBrowserTree() *browserTree {
	b := &browserTree{screen: s}
	b.model, _ = tuivory.NewTree(nil)
	var err error
	b.TreeView, err = tuivory.NewTreeView(b.model, &b.state, s.browserPalette())
	if err != nil {
		panic(err)
	}
	b.OnEvent = b.event
	s.styleBox(b.Box)
	return b
}

// Adapt public style values, rather than mixing the draft candidate palette
// into the application's established theme.
func (s *screen) browserPalette() tuivory.Palette {
	p := s.tuivoryStyles()
	c := p.Colors
	return tuivory.Palette{
		"body":         {FG: c.Text, BG: c.Background},
		"hint":         {FG: c.Muted, BG: c.Background},
		"surface":      {FG: c.Text, BG: c.Background},
		"selection":    {FG: p.SelectionFG, BG: p.Selection},
		"focus.border": {FG: s.borderColor(true, c.Background), BG: c.Background},
	}
}

func browserID(n *tview.TreeNode) string {
	d, _ := n.GetReference().(*browserNode)
	if d == nil {
		return fmt.Sprintf("node:%p", n)
	}
	return d.kind + ":" + strconv.Quote(d.object.Owner) + ":" + strconv.Quote(d.object.Kind) + ":" + strconv.Quote(d.object.Name) + ":" + strconv.Quote(d.column.Name) + ":" + strconv.Quote(d.label)
}

func (b *browserTree) GetRoot() *tview.TreeNode { return b.root }
func (b *browserTree) SetRoot(root *tview.TreeNode) *browserTree {
	b.root = root
	b.state = tuivory.TreeState{Expanded: map[string]bool{}}
	b.Refresh()
	return b
}
func (b *browserTree) GetCurrentNode() *tview.TreeNode {
	n := b.nodes[b.state.Selected]
	if n != nil {
		if d, ok := n.GetReference().(*browserNode); ok && (d.kind == "object" || d.kind == "column") {
			return n
		}
	}
	return nil
}
func (b *browserTree) SetCurrentNode(n *tview.TreeNode) *browserTree {
	b.state.Selected = ""
	if n != nil {
		b.state.Selected = browserID(n)
	}
	b.Refresh()
	return b
}

func (b *browserTree) Refresh() {
	var nodes []tuivory.TreeNode
	b.nodes = map[string]*tview.TreeNode{}
	var visit func(*tview.TreeNode, string)
	visit = func(n *tview.TreeNode, parent string) {
		id := browserID(n)
		d, _ := n.GetReference().(*browserNode)
		if d != nil && d.kind == "message" && parent != "" {
			id = parent + "/" + id
		}
		row := tuivory.TreeNode{ID: id, Parent: parent, Name: tview.Unescape(n.GetText())}
		if d != nil {
			switch d.kind {
			case "object":
				row.Name = d.object.Name
				row.Icon, row.ASCIIIcon = "◇", "O"
				row.Branch = d.object.HasColumns()
				if row.Branch {
					row.Icon, row.ASCIIIcon = "▦", "T"
					if d.object.Kind == "VIEW" {
						row.Icon, row.ASCIIIcon = "◇", "V"
					}
					if d.object.Kind == "MATERIALIZED VIEW" {
						row.Icon, row.ASCIIIcon = "◇", "M"
					}
				}
				if d.object.Status != "VALID" {
					row.Secondary = d.object.Status
				}
				if d.loaded && row.Branch {
					count := uint64(len(n.GetChildren()))
					if len(n.GetChildren()) == 1 {
						if x, ok := n.GetChildren()[0].GetReference().(*browserNode); ok && x.kind == "message" {
							count = 0
						}
					}
					row.Count = &count
				}
			case "column":
				row.Name = d.column.Name
				row.Secondary = d.column.DataType
				if d.column.Nullable {
					row.Secondary += " · NULL"
				}
				// TreeView supplies the leaf marker; a second icon wastes columns.
			case "message":
				row.Name = b.screen.tr(d.label)
			}
		}
		b.nodes[id] = n
		nodes = append(nodes, row)
		if n.IsExpanded() {
			b.state.Expanded[id] = true
		} else {
			delete(b.state.Expanded, id)
		}
		for _, child := range n.GetChildren() {
			visit(child, id)
		}
	}
	if b.state.Expanded == nil {
		b.state.Expanded = map[string]bool{}
	}
	if b.root != nil {
		for _, n := range b.root.GetChildren() {
			visit(n, "")
		}
	}
	var err error
	b.model, err = tuivory.NewTree(nodes)
	if err != nil {
		panic(err)
	} // Node identities are generated from Oracle identities.
	_ = b.SetTree(b.model)
}

func (b *browserTree) event(e tuivory.TreeEvent) {
	n := b.nodes[e.ID]
	if n == nil {
		return
	}
	switch e.Kind {
	case "expand":
		n.SetExpanded(true)
		b.screen.expandBrowser(n)
	case "collapse":
		n.SetExpanded(false)
	case "activate":
		b.screen.activateBrowser(n)
	}
}
func (b *browserTree) Draw(screen tcell.Screen) {
	_ = b.SetPalette(b.screen.browserPalette())
	b.ASCII = b.screen.asciiDisplay()
	b.TreeView.Draw(screen)
}
func (b *browserTree) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return b.WrapMouseHandler(func(action tview.MouseAction, e *tcell.EventMouse, setFocus func(tview.Primitive)) (bool, tview.Primitive) {
		x, y := e.Position()
		left, top, w, h := b.GetInnerRect()
		if x < left || x >= left+w || y < top || y >= top+h {
			return false, nil
		}
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		if action == tview.MouseScrollUp || action == tview.MouseScrollDown {
			a := tuivory.TreeUp
			if action == tview.MouseScrollDown {
				a = tuivory.TreeDown
			}
			b.event(b.model.Handle(&b.state, a, h))
			return true, nil
		}
		if action != tview.MouseLeftClick {
			return false, nil
		}
		setFocus(b)
		rows := b.model.Reconcile(&b.state, h)
		index := b.state.Scroll + y - top
		if index >= len(rows) {
			return true, nil
		}
		event := b.model.Click(&b.state, x-left, y-top, h)
		b.event(event)
		// Keep the existing single-click details action; the disclosure arrow is
		// reserved for expansion; the leading selection margin does not activate.
		if event.Kind != "expand" && event.Kind != "collapse" && x-left >= 4 {
			b.screen.activateBrowser(b.nodes[rows[index].Node.ID])
		}
		return true, nil
	})
}
