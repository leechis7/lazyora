package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"unicode"
	"unicode/utf8"
)

func validTabName(name string) bool {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 40 {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			return false
		}
	}
	return true
}
func (s *screen) currentTabName() string {
	if len(s.tabs) > s.activeTab && s.tabs[s.activeTab].name != "" {
		return s.tabs[s.activeTab].name
	}
	return s.tr("file_untitled")
}
func (s *screen) showTabRename() {
	if !s.commandEnabled("tab_rename") {
		return
	}
	form := tview.NewForm().AddInputField(s.tr("tab_rename_label"), s.currentTabName(), 40, nil, nil)
	s.renameForm = form
	close := func() { s.pages.RemovePage("tab_rename"); s.renameForm = nil; s.app.SetFocus(s.editor) }
	apply := func() {
		name := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		if !validTabName(name) {
			s.setStatus("tab_rename_invalid")
			return
		}
		if s.document.Path != "" {
			return
		}
		s.tabs[s.activeTab].name = name
		s.refreshDocumentTitle()
		close()
	}
	form.AddButton(s.tr("save"), apply).AddButton(s.tr("cancel"), close)
	s.styleForm(form)
	form.SetTitle(s.tr("tab_rename"))
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	form.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			close()
			return nil
		}
		if e.Key() == tcell.KeyCtrlS {
			apply()
			return nil
		}
		return e
	})
	s.pages.AddPage("tab_rename", s.formDialog(form, s.newTextView(textHint).SetText(s.tr("tab_rename_hint")), 64), true, true)
	focusForm(s, form, 0)
}
