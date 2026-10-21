package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) confirmSQL(text string, risk oracle.SQLRisk) {
	s.confirmExecution(text, string(risk), s.submitSQL)
}

func (s *screen) confirmExecution(text, risk string, submit func(string)) {
	s.confirmActionPreview(text, s.tr("sql_risk_"+risk)+"\n\n"+s.tr("sql_risk_limits"), s.tr("sql_confirm_title"), s.tr("execute"), func() { submit(text) })
}

func (s *screen) confirmActionPreview(text, message, title, button string, submit func()) {
	if s.sqlConfirmText != nil || s.programModal != nil || s.busy {
		return
	}
	s.sqlConfirmFocus = s.app.GetFocus()
	reason := tview.NewTextView().SetWrap(true).SetText(message)
	reason.SetBackgroundColor(palette.background)
	reason.SetTextColor(tcell.ColorYellow)
	preview := tview.NewTextView().SetText(text).SetDynamicColors(false).SetWrap(false).SetScrollable(true)
	preview.SetBorder(true).SetTitle(title)
	styleBox(preview.Box)
	preview.SetTextColor(palette.text)
	s.sqlConfirmText = preview
	execute := clickable(tview.NewButton(button).SetSelectedFunc(func() {
		if s.busy {
			return
		}
		s.closeSQLConfirmation()
		submit() // Submit the reviewed snapshot, never a later editor value.
	}))
	cancel := clickable(tview.NewButton(s.tr("cancel")).SetSelectedFunc(s.closeSQLConfirmation))
	buttons := tview.NewFlex().AddItem(execute, 0, 1, false).AddItem(cancel, 0, 1, true)
	reason.SetBackgroundColor(palette.surface)
	content := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(reason, 8, 0, false).AddItem(preview, 0, 1, false).AddItem(buttons, 1, 0, true)
	root := s.formPopup(content, 104, 32, func(w, h int) {
		content.ResizeItem(reason, min(8, max(3, h/3)), 0)
	})
	s.pages.AddPage("sql_confirm", root, true, true)
	s.app.SetFocus(cancel)
}
func (s *screen) closeSQLConfirmation() {
	s.pages.RemovePage("sql_confirm")
	s.sqlConfirmText = nil
	if s.sqlConfirmFocus != nil {
		s.app.SetFocus(s.sqlConfirmFocus)
	} else {
		s.app.SetFocus(s.editor)
	}
	s.sqlConfirmFocus = nil
}
func (s *screen) sqlConfirmationKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		return nil
	case tcell.KeyEscape:
		s.closeSQLConfirmation()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(event, s.sqlConfirmText)
		return nil
	}
	return event
}
