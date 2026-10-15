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
		case *tview.DropDown:
			item.SetFocusedStyle(focusStyle()).SetListStyles(fieldStyle(), focusStyle())
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
