package ui

import (
	"fmt"
	"lazyora/internal/appinfo"

	"github.com/leechis7/tuivory/go"
)

func (s *screen) aboutContent() string {
	return fmt.Sprintf("%s v%s\n\n%s: %s\n\n%s: %s\n\n%s: %s\n%s\n\n%s: %s\n\nCopyright (C) 2026 Chaekang Lee\n", appinfo.Name, appinfo.Version, s.tr("developer"), appinfo.Developer, s.tr("email"), appinfo.Email, s.tr("license"), appinfo.License, appinfo.LicenseURL, s.tr("project"), appinfo.ProjectURL)
}
func (s *screen) showAbout() {
	s.aboutFocus = s.app.GetFocus()
	text := s.newTextView().SetWrap(true).SetScrollable(true).SetDynamicColors(false).SetText(s.aboutContent()).SetTextColor(s.colors().text)
	text.SetWordWrap(true)
	s.styleBox(text.Box)
	s.textSurface(text, surfacePanel)
	s.aboutText = text
	close := tuivory.NewWidgetButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeAbout).SetStyle(s.buttonStyle()).SetActivatedStyle(s.buttonFocusStyle())
	s.clickable(close)
	buttons := s.newFlex().AddItem(nil, 0, 1, false).AddItem(close, 20, 0, false).AddItem(nil, 0, 1, false)
	s.setSurface(buttons.Box, surfacePanel)
	content := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(text, 0, 1, true).AddItem(buttons, 1, 0, false)
	s.styleBox(content.Box)
	s.setSurface(content.Box, surfacePanel)
	content.SetBorder(true).SetTitle(s.tr("about"))
	root := s.formPopup(content, 76, 24, func(width, height int) {
		if height < 18 {
			content.SetBorderPadding(0, 0, 1, 1)
		} else {
			content.SetBorderPadding(1, 1, 2, 2)
		}
	})
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
func (s *screen) aboutKey(e *tuivory.EventKey) *tuivory.EventKey {
	if s.buttonArrow(e) {
		return nil
	}
	switch e.Key() {
	case tuivory.KeyEscape:
		s.closeAbout()
		return nil
	case tuivory.KeyF1:
		s.showHelp()
		return nil
	case tuivory.KeyCtrlC:
		s.sendClipboard(s.aboutContent())
		return nil
	case tuivory.KeyCtrlQ:
		return nil
	case tuivory.KeyTab, tuivory.KeyBacktab:
		s.dialogTab(e, s.aboutText)
		return nil
	}
	return e
}
