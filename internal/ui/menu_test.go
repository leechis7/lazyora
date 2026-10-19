package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"testing"
)

func TestMenuCommandsRespectContextAndNeverAliasScriptToCommit(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = true
	s.key(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt))
	if s.menuList == nil || s.menuList.GetItemCount() != 6 {
		t.Fatal("execute menu not opened")
	}
	text, _ := s.menuList.GetItemText(1)
	if !strings.Contains(text, "F5") {
		t.Fatal("script key not visible")
	}
	s.menuList.SetCurrentItem(1)
	s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if len(s.jobs) != 0 || s.sqlConfirmText == nil {
		t.Fatal("script bypassed confirmation or executed SQL/commit")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyF5, 0, 0))
	if len(s.jobs) != 0 {
		t.Fatal("F5 committed")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyF6, 0, 0))
	if len(s.jobs) != 0 || !s.table.HasFocus() {
		t.Fatal("F6 rolled back instead of toggling SQL/results")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt))
	s.menuList.SetCurrentItem(1)
	s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if j := <-s.jobs; j.action != "commit" {
		t.Fatal("session menu did not commit")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.key(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModAlt))
	if j := <-s.jobs; j.action != "rollback" {
		t.Fatal("Alt+R did not roll back")
	}
}

func TestMenusPreserveFocusAndStayOutOfDialogs(t *testing.T) {
	s := demoBrowserScreen(t)
	s.app.SetFocus(s.table)
	s.key(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModAlt))
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if s.menuIndex != 1 {
		t.Fatal("Right did not switch menu")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if !s.table.HasFocus() {
		t.Fatal("menu lost previous focus")
	}
	s.connectionForm()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModAlt))
	if s.menuList != nil || !s.connectForm.GetFormItem(connectionHost).HasFocus() {
		t.Fatal("menu stole connection mnemonic")
	}
	s.key(tcell.NewEventKey(tcell.KeyF9, 0, 0))
	if len(s.jobs) != 0 {
		t.Fatal("F9 executed behind connection form")
	}
	s.closeConnection()
	s.editor.SetText("SAMPLE_ROWS", false)
	s.key(tcell.NewEventKey(tcell.KeyF4, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "object" {
		t.Fatal("F4 did not describe demo object")
	}
	if len(s.jobs) != 0 {
		t.Fatal("F4 executed SQL")
	}
	s.prefs.Language = "ko"
	if !strings.Contains(s.mainHelp(), "전체화면") || !strings.Contains(s.mainHelp(), "Alt+R") {
		t.Fatal("generated help missed translated commands")
	}
}

func TestMouseMenuUsesTheFocusedResultPanel(t *testing.T) {
	s, _ := resultScreen(t)
	bar := s.menuBar()
	terminal := s.terminal
	bar.SetRect(0, 0, 120, 1)
	bar.Draw(terminal)
	button := s.menuButtons[1]
	x, y, _, _ := button.GetRect()
	focus := func(p tview.Primitive) { s.app.SetFocus(p) }
	button.MouseHandler()(tview.MouseLeftDown, tcell.NewEventMouse(x+1, y, tcell.Button1, 0), focus)
	button.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y, 0, 0), focus)
	if s.menuFocus != s.table {
		t.Fatal("mouse menu stole panel focus")
	}
	s.menuList.SetCurrentItem(1)
	s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), focus)
	if !strings.Contains(s.clipboardText, "한글 [red]") || !s.table.HasFocus() {
		t.Fatal("mouse-opened menu did not copy the focused row")
	}
}
