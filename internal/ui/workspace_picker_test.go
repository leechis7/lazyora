package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"testing"
)

func TestWorkspaceMouseSelectDoesNotApplyUntilConfirmed(t *testing.T) {
	m, terminal := workspaceTestSet(t)
	first := m.active
	second := secondWorkspace(m)
	m.activate(first)
	first.showWorkspaces()
	_, root := first.pages.GetFrontPage()
	root.SetRect(0, 0, 120, 36)
	root.Draw(terminal)
	for i := 0; i < first.workspaceList.GetItemCount(); i++ {
		_, note := first.workspaceList.GetItemText(i)
		if note == first.tr("workspace_shared_tabs") {
			t.Fatal("common note repeated")
		}
	}
	x, y, _, _ := first.workspaceList.GetInnerRect()
	delegate := func(p tview.Primitive) { first.app.SetFocus(p) }
	first.workspaceList.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y+2, 0, 0), delegate)
	if first.workspaceList.GetCurrentItem() != 1 || m.active != first || !first.pages.HasPage("workspaces") {
		t.Fatal("single click switched/closed")
	}
	first.workspaceList.MouseHandler()(tview.MouseLeftDoubleClick, tcell.NewEventMouse(x+1, y+2, 0, 0), delegate)
	if m.active != first || !first.pages.HasPage("workspaces") {
		t.Fatal("double click switched/closed")
	}
	first.workspaceKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if m.active != first {
		t.Fatal("cancel applied selection")
	}
	first.showWorkspaces()
	first.workspaceList.SetCurrentItem(1)
	first.workspaceList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), delegate)
	if m.active != second || first.pages.HasPage("workspaces") {
		t.Fatal("explicit confirmation did not switch")
	}
}
