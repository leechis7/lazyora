package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
	"strings"
	"testing"
)

func TestAboutMenuSharedMetadataAndStatePreserved(t *testing.T) {
	for _, lang := range []string{"ko", "en"} {
		for _, limited := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", lang, limited), func(t *testing.T) {
				s, terminal := resultScreen(t)
				s.prefs.Language = lang
				s.limitedColors = limited
				s.editor.SetText("SELECT '한글' FROM dual", true)
				s.app.SetFocus(s.editor)
				s.pendingChanges = true
				s.connected = true
				s.showMenu(menuHelp)
				if s.menuList.GetItemCount() != 2 {
					t.Fatal("Help/About menu missing")
				}
				s.menuList.SetCurrentItem(1)
				s.menuList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
				if !s.pages.HasPage("about") {
					t.Fatal("menu did not open About")
				}
				for _, value := range []string{appinfo.Name, appinfo.Version, appinfo.Developer, appinfo.Email, appinfo.License, appinfo.LicenseURL, appinfo.ProjectURL} {
					if !strings.Contains(s.aboutText.GetText(false), value) || !strings.Contains(s.helpArticle(9), value) {
						t.Fatalf("inconsistent info: %s", value)
					}
				}
				_, root := s.pages.GetFrontPage()
				for _, size := range [][2]int{{120, 30}, {60, 20}, {40, 12}} {
					terminal.SetSize(size[0], size[1])
					root.SetRect(0, 0, size[0], size[1])
					root.Draw(terminal)
				}
				s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
				if s.clipboardText != s.aboutContent() {
					t.Fatal("metadata copy changed")
				}
				s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
				s.closeHelp()
				if s.app.GetFocus() != s.aboutText {
					t.Fatal("help did not restore About focus")
				}
				s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
				button, ok := s.app.GetFocus().(*tview.Button)
				if !ok {
					t.Fatal("close button not keyboard accessible")
				}
				button.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
				if s.pages.HasPage("about") || s.app.GetFocus() != s.editor || !s.pendingChanges || !s.connected || s.editor.GetText() != "SELECT '한글' FROM dual" {
					t.Fatal("About disturbed working state")
				}
			})
		}
	}
}
func TestAboutEscapeDoesNotCancelSQLAndMouseClose(t *testing.T) {
	s, terminal := resultScreen(t)
	s.app.SetFocus(s.table)
	s.busy = true
	cancelled := false
	s.cancel = func() { cancelled = true }
	s.invoke("about")
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if cancelled || !s.busy || s.app.GetFocus() != s.table {
		t.Fatal("About Escape cancelled background work")
	}
	s.busy = false
	s.showAbout()
	_, root := s.pages.GetFrontPage()
	root.SetRect(0, 0, 80, 20)
	root.Draw(terminal)
	button := root.(*tview.Flex).GetItem(1).(*tview.Flex).GetItem(1).(*tview.Button)
	x, y, _, _ := button.GetRect()
	button.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y, tcell.Button1, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.pages.HasPage("about") {
		t.Fatal("mouse close failed")
	}
}
