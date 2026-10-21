package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
)

func (s *screen) aboutContent() string {
	return fmt.Sprintf("%s v%s\n\n%s: %s\n\n%s: %s\n\n%s: %s\n%s\n\n%s: %s\n\nCopyright (C) 2026 Chaekang Lee\n", appinfo.Name, appinfo.Version, s.tr("developer"), appinfo.Developer, s.tr("email"), appinfo.Email, s.tr("license"), appinfo.License, appinfo.LicenseURL, s.tr("project"), appinfo.ProjectURL)
}
func (s *screen) showAbout() {
	s.aboutFocus = s.app.GetFocus()
	text := tview.NewTextView().SetWrap(true).SetScrollable(true).SetDynamicColors(false).SetText(s.aboutContent()).SetTextColor(s.colors().text)
	text.SetBackgroundColor(s.colors().background)
	text.SetBorder(true).SetTitle(s.tr("about"))
	s.styleBox(text.Box)
	s.aboutText = text
	close := tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeAbout).SetStyle(s.buttonStyle()).SetActivatedStyle(buttonFocusStyle())
	s.clickable(close)
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(text, 0, 1, true).AddItem(tview.NewFlex().AddItem(nil, 0, 1, false).AddItem(close, 20, 0, false).AddItem(nil, 0, 1, false), 1, 0, false)
	s.pages.AddPage("about", root, true, true)
	s.app.SetFocus(text)
}
func (s *screen) closeAbout() {
	s.pages.RemovePage("about")
	s.aboutText = nil
	if s.aboutFocus != nil {
		s.app.SetFocus(s.aboutFocus)
	}
	s.aboutFocus = nil
}
func (s *screen) aboutKey(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyEscape:
		s.closeAbout()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyCtrlC:
		s.sendClipboard(s.aboutContent())
		return nil
	case tcell.KeyCtrlQ:
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(e, s.aboutText)
		return nil
	}
	return e
}
