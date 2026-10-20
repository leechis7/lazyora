package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"testing"
)

func selectMenuCommand(t *testing.T, s *screen, id string) {
	t.Helper()
	label := s.tr(commandFor(id).label)
	for i := 0; i < s.menuList.GetItemCount(); i++ {
		text, _ := s.menuList.GetItemText(i)
		if text == label || strings.HasPrefix(text, label+"  ") || strings.HasPrefix(text, "[gray]"+label) {
			s.menuList.SetCurrentItem(i)
			s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
			return
		}
	}
	t.Fatalf("command %s missing from menu", id)
}

func TestSQLTabMenuPreservesDocumentsAndProtectsDirtyClose(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		t.Run(language, func(t *testing.T) {
			s := fileTestScreen(t)
			s.prefs.Language = language
			s.initTabs()
			first := s.editor
			s.key(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModAlt))
			if s.menuIndex != menuTabs || s.menuList.GetItemCount() != 4 {
				t.Fatal("SQL tabs menu not available")
			}
			selectMenuCommand(t, s, "file_new")
			if len(s.tabs) != 2 || s.editor == first {
				t.Fatal("menu did not open an independent SQL tab")
			}
			s.editor.SetText("SELECT 'keep' FROM dual;", false)
			s.showMenu(menuTabs)
			selectMenuCommand(t, s, "tab_previous")
			if s.editor != first {
				t.Fatal("previous tab did not restore first editor")
			}
			s.showMenu(menuTabs)
			selectMenuCommand(t, s, "tab_next")
			s.showMenu(menuTabs)
			selectMenuCommand(t, s, "file_close")
			if page, _ := s.pages.GetFrontPage(); page != "file_unsaved" || len(s.tabs) != 2 {
				t.Fatal("menu close bypassed unsaved SQL protection")
			}
		})
	}
}

func TestEditMenuRetainsSelectionClipboardAndUndo(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetClipboard(s.sendClipboard, func() string { return s.clipboardText })
	s.editor.SetText("SELECT 한글 FROM dual;", false).Select(7, 13)
	s.showMenu(menuEdit)
	selectMenuCommand(t, s, "cut")
	if s.clipboardText != "한글" || s.editor.GetText() != "SELECT  FROM dual;" {
		t.Fatalf("cut lost the editor selection or clipboard: %q, %q", s.clipboardText, s.editor.GetText())
	}
	s.showMenu(menuEdit)
	selectMenuCommand(t, s, "undo")
	if s.editor.GetText() != "SELECT 한글 FROM dual;" {
		t.Fatal("menu undo did not restore cut text")
	}
	s.showMenu(menuEdit)
	selectMenuCommand(t, s, "redo")
	s.showMenu(menuEdit)
	selectMenuCommand(t, s, "paste")
	if s.editor.GetText() != "SELECT 한글 FROM dual;" {
		t.Fatal("menu redo/paste failed")
	}
	s.showMenu(menuEdit)
	selectMenuCommand(t, s, "select_all")
	text, _, _ := s.editor.GetSelection()
	if text != s.editor.GetText() || !s.editor.HasFocus() || len(s.jobs) != 0 {
		t.Fatal("select all lost focus or submitted DB work")
	}
	s.app.SetFocus(s.table)
	s.showMenu(menuEdit)
	selectMenuCommand(t, s, "cut")
	if s.editor.GetText() != text || !s.table.HasFocus() {
		t.Fatal("disabled editor command modified SQL from result focus")
	}
}

func TestMenusBlockTabChangesAndEditsWhileRunning(t *testing.T) {
	s := fileTestScreen(t)
	s.initTabs()
	s.busy = true
	original := s.editor.GetText()
	for _, id := range []string{"file_new", "file_close", "tab_next", "tab_previous", "cut", "paste", "undo", "redo", "select_all", "browser_refresh"} {
		s.showMenu(commandFor(id).group)
		selectMenuCommand(t, s, id)
		if len(s.tabs) != 1 || s.editor.GetText() != original || len(s.jobs) != 0 {
			t.Fatalf("busy menu command %s changed active work", id)
		}
	}
}

func TestLongMenuScrollsWithinSmallTerminal(t *testing.T) {
	s, terminal := resultScreen(t)
	terminal.SetSize(80, 10)
	s.app.SetFocus(s.editor)
	s.showMenu(menuEdit)
	_, root := s.pages.GetFrontPage()
	root.SetRect(0, 0, 80, 10)
	root.Draw(terminal)
	_, y, _, height := s.menuList.GetRect()
	if y < 0 || y+height > 10 {
		t.Fatal("menu extends beyond terminal")
	}
	for i := 1; i < s.menuList.GetItemCount(); i++ {
		s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	}
	root.Draw(terminal)
	var visible strings.Builder
	for row := 0; row < 10; row++ {
		for col := 0; col < 80; col++ {
			main, _, _, _ := terminal.GetContent(col, row)
			visible.WriteRune(main)
		}
	}
	if !strings.Contains(visible.String(), s.tr("clear_sql")) {
		t.Fatal("last edit command is not reachable on a short terminal")
	}
}

func TestBrowserRefreshAvailableFromMenuAndContextMenu(t *testing.T) {
	for _, context := range []bool{false, true} {
		s := demoBrowserScreen(t)
		s.app.SetFocus(s.browser)
		s.browser.GetRoot().ClearChildren()
		if context {
			s.showContextMenu()
		} else {
			s.showMenu(menuView)
		}
		selectMenuCommand(t, s, "browser_refresh")
		if len(s.browser.GetRoot().GetChildren()) == 0 || len(s.jobs) != 0 {
			t.Fatal("menu refresh did not reload demo objects locally")
		}
	}
}

func TestMenuCommandsRespectContextAndNeverAliasScriptToCommit(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = true
	s.key(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt))
	if s.menuList == nil || s.menuList.GetItemCount() != 7 {
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
	button := s.menuButtons[menuEdit]
	x, y, _, _ := button.GetRect()
	focus := func(p tview.Primitive) { s.app.SetFocus(p) }
	button.MouseHandler()(tview.MouseLeftDown, tcell.NewEventMouse(x+1, y, tcell.Button1, 0), focus)
	button.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y, 0, 0), focus)
	if s.menuFocus != s.table {
		t.Fatal("mouse menu stole panel focus")
	}
	selectMenuCommand(t, s, "copy_row")
	if !strings.Contains(s.clipboardText, "한글 [red]") || !s.table.HasFocus() {
		t.Fatal("mouse-opened menu did not copy the focused row")
	}
}
