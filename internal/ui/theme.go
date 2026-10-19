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

// A separate button focus palette stays distinct even when pale RGB colors
// map to the same color on an 8/16-color terminal.
func buttonFocusStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy).Bold(true).Underline(true)
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
		SetButtonStyle(buttonStyle()).SetButtonActivatedStyle(buttonFocusStyle())
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
	if s.sqlConfirmText != nil {
		apply(s.sqlConfirmText.Box, s.sqlConfirmText.HasFocus(), false)
	}
	if s.bindForm != nil {
		apply(s.bindForm.Box, s.bindForm.HasFocus(), false)
	}
	if s.scriptList != nil {
		apply(s.scriptList.Box, s.scriptList.HasFocus(), false)
		apply(s.scriptText.Box, s.scriptText.HasFocus(), false)
		apply(s.scriptResult.Box, s.scriptResult.HasFocus(), false)
	}
	if s.fileDialog != nil {
		apply(s.fileDialog.form.Box, s.fileDialog.form.HasFocus(), false)
		apply(s.fileDialog.list.Box, s.fileDialog.list.HasFocus(), false)
	}
	if s.documentModal != nil {
		apply(s.documentModal.Box, s.documentModal.HasFocus(), false)
	}
	apply(s.editor.Box, s.editor.HasFocus(), true)
	apply(s.table.Box, s.table.HasFocus(), false)
	if s.pendingTable != nil {
		apply(s.pendingTable.Box, s.pendingTable.HasFocus(), false)
	}
	if s.pendingSQL != nil {
		apply(s.pendingSQL.Box, s.pendingSQL.HasFocus(), false)
	}
	if s.searchForm != nil {
		apply(s.searchForm.Box, s.searchForm.HasFocus(), false)
	}
	if s.historyList != nil {
		apply(s.historyList.Box, s.historyList.HasFocus(), false)
	}
	if s.menuList != nil {
		s.menuList.SetBorderStyle(tcell.StyleDefault.Foreground(tcell.ColorTeal).Background(s.menuList.GetBackgroundColor()))
	}
	if s.describeForm != nil {
		apply(s.describeForm.Box, s.describeForm.HasFocus(), false)
	}
	if s.copyText != nil {
		apply(s.copyText.Box, s.copyText.HasFocus(), true)
	}
	if s.browserForm != nil {
		apply(s.browserForm.Box, s.browserForm.HasFocus(), false)
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
	for _, form := range []*tview.Form{s.connectForm, s.connectProfiles, s.prefsForm, s.masterForm, s.vaultForm} {
		if form != nil {
			apply(form.Box, form.HasFocus(), false)
		}
	}
	if page, root := s.pages.GetFrontPage(); page == "help" {
		help := root.(*tview.Flex).GetItem(0).(*tview.TextView)
		apply(help.Box, help.HasFocus(), false)
	}
}

// Box normally uses double lines while focused. Normalize only this box's
// border glyphs, preserving its title and styles without changing global borders.
func singleLineBorder(box *tview.Box) {
	box.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		replace := func(px, py int) {
			r, combining, style, _ := screen.GetContent(px, py)
			switch r {
			case tview.Borders.HorizontalFocus:
				r = tview.Borders.Horizontal
			case tview.Borders.VerticalFocus:
				r = tview.Borders.Vertical
			case tview.Borders.TopLeftFocus:
				r = tview.Borders.TopLeft
			case tview.Borders.TopRightFocus:
				r = tview.Borders.TopRight
			case tview.Borders.BottomLeftFocus:
				r = tview.Borders.BottomLeft
			case tview.Borders.BottomRightFocus:
				r = tview.Borders.BottomRight
			default:
				return
			}
			screen.SetContent(px, py, r, combining, style)
		}
		if width >= 2 && height >= 2 {
			for px := x; px < x+width; px++ {
				replace(px, y)
				replace(px, y+height-1)
			}
			for py := y + 1; py < y+height-1; py++ {
				replace(x, py)
				replace(x+width-1, py)
			}
		}
		return x + 1, y + 1, max(0, width-2), max(0, height-2)
	})
}
