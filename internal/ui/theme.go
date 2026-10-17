package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Nord-inspired neutral surfaces and cyan accents, with light input fields.
// https://www.nordtheme.com/docs/colors-and-palettes/
var palette = struct {
	background, surface, footer, field, text, muted, accent, button tcell.Color
}{
	background: tcell.GetColor("#2e3440"),
	surface:    tcell.GetColor("#3b4252"),
	footer:     tcell.GetColor("#434c5e"),
	field:      tcell.GetColor("#eceff4"),
	text:       tcell.GetColor("#eceff4"),
	muted:      tcell.GetColor("#d8dee9"),
	accent:     tcell.GetColor("#88c0d0"),
	button:     tcell.GetColor("#81a1c1"),
}

func fieldStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(palette.background).Background(palette.field)
}

func focusStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(palette.background).Background(palette.accent).Bold(true)
}

func buttonStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(palette.background).Background(palette.button)
}

func styleBox(box *tview.Box) {
	box.SetBackgroundColor(palette.background).SetBorderColor(palette.button).SetTitleColor(palette.text)
}

func styleForm(form *tview.Form) {
	styleBox(form.Box)
	form.SetLabelColor(palette.muted).SetFieldStyle(fieldStyle()).
		SetButtonStyle(buttonStyle()).SetButtonActivatedStyle(focusStyle())
	for index := 0; index < form.GetFormItemCount(); index++ {
		switch item := form.GetFormItem(index).(type) {
		case *tview.InputField:
			item.SetBackgroundColor(palette.background)
		case *choiceField:
			item.SetFocusedStyle(focusStyle())
		case *tview.Checkbox:
			item.SetActivatedStyle(focusStyle())
		}
	}
}

func (s *screen) styleScreen() {
	styleBox(s.editor.Box)
	s.editor.SetBackgroundColor(palette.field).SetTitleColor(palette.background)
	s.editor.SetTextStyle(fieldStyle()).SetSelectedStyle(focusStyle())
	styleBox(s.table.Box)
	s.table.SetSelectedStyle(focusStyle())
	s.status.SetBackgroundColor(palette.background)
	s.status.SetTextColor(palette.muted)
	s.footer.SetBackgroundColor(palette.surface)
	s.footer.SetTextColor(palette.accent)
}

// Pale RGB borders can collapse to white on an 8/16-color terminal. Use
// distinct ANSI colors there, including a dark border on the light editor.
func (s *screen) refreshBorders() {
	if !s.limitedColors {
		return
	}
	apply := func(box *tview.Box, focused, light bool) {
		color := tcell.ColorTeal
		background := palette.background
		if light {
			color = tcell.ColorNavy
			background = palette.field
		}
		box.SetBorderStyle(tcell.StyleDefault.Foreground(color).Background(background).Bold(focused))
	}
	apply(s.editor.Box, s.editor.HasFocus(), true)
	apply(s.table.Box, s.table.HasFocus(), false)
	if s.menuList != nil {
		apply(s.menuList.Box, true, false)
	}
	if s.describeForm != nil {
		apply(s.describeForm.Box, s.describeForm.HasFocus(), false)
	}
	if s.copyText != nil {
		apply(s.copyText.Box, s.copyText.HasFocus(), true)
	}
	if s.browser != nil {
		apply(s.browser.Box, s.browser.HasFocus(), false)
	}
	if s.objectTable != nil {
		apply(s.objectTable.Box, s.objectTable.HasFocus(), false)
	}
	if s.exitDialog != nil {
		apply(s.exitDialog.Box, s.exitDialog.HasFocus(), false)
	}
	if s.programModal != nil {
		apply(s.programModal.Box, s.programModal.HasFocus(), false)
	}
	if s.sourceEditor != nil {
		apply(s.sourceEditor.Box, s.sourceEditor.HasFocus(), true)
	}
	for _, table := range []*tview.Table{s.sourceDiagnostics, s.argumentTable} {
		if table != nil {
			apply(table.Box, table.HasFocus(), false)
		}
	}
	for _, form := range []*tview.Form{s.connectForm, s.connectProfiles, s.prefsForm} {
		if form != nil {
			apply(form.Box, form.HasFocus(), false)
		}
	}
	if page, root := s.pages.GetFrontPage(); page == "help" {
		help := root.(*tview.Flex).GetItem(0).(*tview.TextView)
		apply(help.Box, help.HasFocus(), false)
	}
}
