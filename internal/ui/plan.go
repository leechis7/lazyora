package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) explainSQL() {
	if s.busy || !s.connected {
		return
	}
	text, offset, _ := s.editor.GetSelection()
	if text == "" {
		var err error
		text, err = oracle.AtStatement(s.editor.GetText(), offset)
		if err != nil {
			s.setStatus("error", err)
			return
		}
	}
	if _, err := oracle.PlanSQL(text); err != nil {
		s.setStatus("error", err)
		return
	}
	s.withBinds(text, func(values map[string]oracle.BindValue) { s.submit(job{action: "plan", text: text, bindings: values}) })
}

func (s *screen) showPlan(text string, steps []oracle.PlanStep) {
	s.planFocus = s.app.GetFocus()
	rootNode := tview.NewTreeNode(s.tr("plan_title")).SetSelectable(true).SetColor(palette.accent)
	tree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)
	tree.SetBorder(true)
	styleBox(tree.Box)
	detail := tview.NewTextView().SetWrap(false).SetScrollable(true).SetText(s.tr("plan_hint") + "\n\n" + text)
	detail.SetBorder(true)
	styleBox(detail.Box)
	detail.SetTextColor(palette.text)
	nodes := map[int]*tview.TreeNode{}
	for _, step := range steps {
		label := fmt.Sprintf("%d %s %s %s · cost %s / rows %s", step.ID, step.Operation, step.Options, step.Object, step.Cost, step.Rows)
		node := tview.NewTreeNode(label).SetReference(step).SetSelectable(true).SetColor(palette.text)
		nodes[step.ID] = node
	}
	for _, step := range steps {
		node := nodes[step.ID]
		parent := nodes[step.Parent]
		if !step.HasParent || parent == nil || step.ID == step.Parent {
			rootNode.AddChild(node)
		} else {
			parent.AddChild(node)
		}
	}
	tree.SetChangedFunc(func(node *tview.TreeNode) {
		if step, ok := node.GetReference().(oracle.PlanStep); ok {
			detail.SetText(step.Description() + "\n\n" + s.tr("plan_hint") + "\n\n" + text)
		}
	})
	tree.SetSelectedFunc(func(node *tview.TreeNode) { node.SetExpanded(!node.IsExpanded()) })
	s.planTree, s.planText = tree, detail
	close := clickable(tview.NewButton(s.tr("close")).SetSelectedFunc(s.closePlan))
	info := tview.NewTextView().SetWrap(true).SetText(s.tr("plan_hint"))
	info.SetTextColor(palette.muted).SetBackgroundColor(palette.background)
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(info, 3, 0, false).AddItem(tree, 0, 3, true).AddItem(detail, 0, 2, false).AddItem(close, 1, 0, false)
	s.pages.AddPage("plan", root, true, true)
	s.app.SetFocus(tree)
}

func (s *screen) closePlan() {
	s.pages.RemovePage("plan")
	s.planTree = nil
	s.planText = nil
	if s.planFocus != nil {
		s.app.SetFocus(s.planFocus)
	}
}
func (s *screen) planKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		s.closePlan()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(event, s.planTree, s.planText)
		return nil
	case tcell.KeyCtrlC:
		s.sendClipboard(s.planText.GetText(false))
		return nil
	}
	return event
}
