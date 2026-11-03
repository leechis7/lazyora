package ui

import (
	"testing"

	"github.com/leechis7/tuivory/go"
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
	delegate := func(p tuivory.Primitive) { first.app.SetFocus(p) }
	first.workspaceList.MouseHandler()(tuivory.MouseLeftClick, tuivory.NewEventMouse(x+1, y+2, 0, 0), delegate)
	if first.workspaceList.GetCurrentItem() != 1 || m.active != first || !first.pages.HasPage("workspaces") {
		t.Fatal("single click switched/closed")
	}
	first.workspaceList.MouseHandler()(tuivory.MouseLeftDoubleClick, tuivory.NewEventMouse(x+1, y+2, 0, 0), delegate)
	if m.active != first || !first.pages.HasPage("workspaces") {
		t.Fatal("double click switched/closed")
	}
	first.workspaceKey(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if m.active != first {
		t.Fatal("cancel applied selection")
	}
	first.showWorkspaces()
	first.workspaceList.SetCurrentItem(1)
	first.workspaceList.InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), delegate)
	if m.active != second || first.pages.HasPage("workspaces") {
		t.Fatal("explicit confirmation did not switch")
	}
}
