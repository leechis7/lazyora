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
	s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.menuParent == nil || s.menuList == parent {
		t.Fatal("Enter did not enter submenu")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.menuParent != nil || !s.pages.HasPage("menu") {
		t.Fatal("Escape did not return")
	}
	s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	s.menuRowAction(0)
	if s.menuParent != nil {
		t.Fatal("mouse Back item failed")
	}
	s.closeMenu()
	if !s.editor.HasFocus() {
		t.Fatal("lost panel focus")
	}
}

func TestMenuArrowsTraverseAllTopLevelMenus(t *testing.T) {
	s, _ := resultScreen(t)
	s.app.SetFocus(s.editor)
	s.showMenu(menuFile)
	for _, direction := range []struct {
		key  tcell.Key
		step int
	}{{tcell.KeyRight, 1}, {tcell.KeyLeft, -1}} {
		for range len(menuGroups) {
			want := (s.menuIndex + direction.step + len(menuGroups)) % len(menuGroups)
			s.key(tcell.NewEventKey(direction.key, 0, 0))
			if s.menuIndex != want || s.menuParent != nil || !s.menuList.HasFocus() {
				t.Fatalf("arrow %v did not reach top-level menu %d", direction.key, want)
			}
		}
	}
	s.closeMenu()
	if !s.editor.HasFocus() || len(s.jobs) != 0 {
		t.Fatal("menu traversal lost focus or started work")
	}
}

func TestSubmenuArrowsReturnOrAdvanceAndContextStaysLocal(t *testing.T) {
	s, _ := resultScreen(t)
	s.app.SetFocus(s.editor)
	for _, menu := range []int{menuEdit, menuView} {
		s.showMenu(menu)
		selected := s.menuList.GetCurrentItem()
		s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if s.menuParent == nil {
			t.Fatal("Enter did not open submenu")
		}
		s.key(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
		if s.menuParent != nil || s.menuIndex != menu || s.menuList.GetCurrentItem() != selected {
			t.Fatal("Left did not restore parent selection")
		}
		s.menuRowAction(selected)
		s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
		if s.menuParent != nil || s.menuIndex != menu+1 {
			t.Fatal("Right did not advance from submenu to next top-level menu")
		}
	}
	s.closeMenu()
	s.showContextMenu()
	list := s.menuList
	for _, key := range []tcell.Key{tcell.KeyLeft, tcell.KeyRight} {
		s.key(tcell.NewEventKey(key, 0, 0))
		if s.menuIndex != -1 || s.menuList != list || s.menuParent != nil {
			t.Fatal("context menu arrow opened a top-level menu")
		}
	}
	s.closeMenu()
	if !s.editor.HasFocus() {
		t.Fatal("lost original editor focus")
	}
}
