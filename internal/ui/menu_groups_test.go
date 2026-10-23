package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"testing"
)

func TestMenuSeparatorsNeverSelectOrExecute(t *testing.T) {
	s, terminal := resultScreen(t)
	s.app.SetFocus(s.editor)
	s.showMenu(menuHelp)
	delegate := func(p tview.Primitive) { s.app.SetFocus(p) }
	_, root := s.pages.GetFrontPage()
	root.SetRect(0, 0, 120, 35)
	root.Draw(terminal)
	separator := -1
	for i, row := range s.menuRows {
		if row.divider {
			separator = i
			break
		}
	}
	if separator < 1 {
		t.Fatal("missing separator")
	}
	s.menuList.SetCurrentItem(separator - 1)
	s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, 0), delegate)
	if s.menuRows[s.menuList.GetCurrentItem()].divider {
		t.Fatal("arrow selected separator")
	}
	x, y, _, _ := s.menuList.GetInnerRect()
	offset, _ := s.menuList.GetOffset()
	before := s.menuList.GetCurrentItem()
	s.menuList.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y+separator-offset, 0, 0), delegate)
	if s.menuList.GetCurrentItem() != before || !s.pages.HasPage("menu") || len(s.jobs) != 0 {
		t.Fatal("separator click changed selection or executed")
	}
}
func TestNestedMenuKeyboardMouseBackAndCommand(t *testing.T) {
	s, _ := resultScreen(t)
	s.app.SetFocus(s.editor)
	s.showMenu(menuEdit)
	if len(s.menuRows) > 12 {
		t.Fatal("long menu not grouped")
	}
	parent := s.menuList
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if s.menuParent == nil || s.menuList == parent {
		t.Fatal("right did not enter submenu")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.menuParent != nil || !s.pages.HasPage("menu") {
		t.Fatal("Escape did not return")
	}
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	s.menuRowAction(0)
	if s.menuParent != nil {
		t.Fatal("mouse Back item failed")
	}
	s.closeMenu()
	if !s.editor.HasFocus() {
		t.Fatal("lost panel focus")
	}
}
