package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Nord-inspired neutral surfaces and cyan accents, with light input fields.
// https://www.nordtheme.com/docs/colors-and-palettes/
type themeColors struct {
	background, surface, footer, field, text, muted, accent, button    tcell.Color
	editor, editorText, fieldText, selectionText, placeholder, warning tcell.Color
}

var palette = themeColors{
	editor: tcell.GetColor("#242d3a"), editorText: tcell.GetColor("#e5edf5"),
	background: tcell.GetColor("#2e3440"),
	surface:    tcell.GetColor("#3b4252"),
	footer:     tcell.GetColor("#434c5e"),
	field:      tcell.GetColor("#eceff4"),
	text:       tcell.GetColor("#eceff4"),
	muted:      tcell.GetColor("#d8dee9"),
	accent:     tcell.GetColor("#88c0d0"),
	button:     tcell.GetColor("#81a1c1"),
	fieldText:  tcell.GetColor("#2e3440"), selectionText: tcell.GetColor("#2e3440"), placeholder: tcell.GetColor("#66758a"), warning: tcell.ColorYellow,
}

func (s *screen) fieldStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().fieldText).Background(s.colors().field)
}

func (s *screen) focusStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().selectionText).Background(s.colors().accent).Bold(true)
}

// A separate button focus palette stays distinct even when pale RGB colors
// map to the same color on an 8/16-color terminal.
func buttonFocusStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy).Bold(true).Underline(true)
}

func (s *screen) buttonStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().fieldText).Background(s.colors().button)
}

func (s *screen) styleBox(box *tview.Box) {
	box.SetBackgroundColor(s.colors().background).SetBorderColor(s.colors().button).SetTitleColor(s.colors().text)
}

func (s *screen) styleForm(form *tview.Form) {
	s.styleBox(form.Box)
	form.SetBackgroundColor(s.colors().surface).SetBorderPadding(1, 1, 2, 2)
	singleLineBorder(form.Box)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetLabelColor(s.colors().muted).SetFieldStyle(s.fieldStyle()).
		SetButtonStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().footer).Bold(true)).SetButtonActivatedStyle(buttonFocusStyle())
	for index := 0; index < form.GetFormItemCount(); index++ {
		switch item := form.GetFormItem(index).(type) {
		case *tview.InputField:
			item.SetBackgroundColor(s.colors().surface)
			item.SetPlaceholderStyle(tcell.StyleDefault.Foreground(s.colors().placeholder).Background(s.colors().field))
			// Form reapplies field attributes before Draw. Set the focused input's
			// contrast here without replacing InputField or disturbing its caret.
			item.SetDrawFunc(func(_ tcell.Screen, x, y, width, height int) (int, int, int, int) {
				label := tcell.StyleDefault.Foreground(s.colors().muted).Background(s.colors().surface)
				input := s.fieldStyle()
				if item.HasFocus() {
					label = label.Foreground(s.colors().accent).Bold(true)
					input = input.Background(tcell.ColorWhite).Bold(true)
				}
				item.SetLabelStyle(label).SetFieldStyle(input)
				return x, y, width, height
			})
		case *choiceField:
			item.SetFocusedStyle(s.focusStyle())
		case *tview.Checkbox:
			item.SetActivatedStyle(s.focusStyle()).SetCheckedString(tview.Escape("[x]")).SetUncheckedString(tview.Escape("[ ]"))
		}
	}
}

func (s *screen) styleScreen() {
	s.styleBox(s.editor.Box)
	s.styleEditor(s.editor)
	s.styleBox(s.table.Box)
	s.table.SetSelectedStyle(s.focusStyle())
	s.status.SetBackgroundColor(s.colors().background)
	s.status.SetTextColor(s.colors().muted)
	s.footer.SetBackgroundColor(s.colors().surface)
	s.footer.SetTextColor(s.colors().accent)
}

// Pale RGB borders can collapse to white on an 8/16-color terminal. Use
// distinct ANSI colors there, including a dark border on the light editor.
func (s *screen) refreshBorders() {
	if !s.limitedColors {
		return
	}
	apply := func(box *tview.Box, focused, light bool) {
		color := tcell.ColorTeal
		background := s.colors().background
		if light {
			color = tcell.ColorNavy
			background = s.colors().field
		}
		box.SetBorderStyle(tcell.StyleDefault.Foreground(color).Background(background).Bold(focused))
	}
	if s.sqlConfirmText != nil {
		apply(s.sqlConfirmText.Box, s.sqlConfirmText.HasFocus(), false)
	}
	if s.bindForm != nil {
		apply(s.bindForm.Box, s.bindForm.HasFocus(), false)
	}
	if s.exportForm != nil {
		apply(s.exportForm.Box, s.exportForm.HasFocus(), false)
	}
	if s.planTree != nil {
		apply(s.planTree.Box, s.planTree.HasFocus(), false)
		apply(s.planText.Box, s.planText.HasFocus(), false)
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
	apply(s.editor.Box, s.editor.HasFocus(), s.prefs.Theme == "light")
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
	if s.workspaceList != nil {
		apply(s.workspaceList.Box, s.workspaceList.HasFocus(), false)
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
	if s.reconnectDialog != nil {
		apply(s.reconnectDialog.Box, s.reconnectDialog.HasFocus(), false)
	}
	if s.programModal != nil {
		apply(s.programModal.Box, s.programModal.HasFocus(), false)
	}
	if s.sourceEditor != nil {
		apply(s.sourceEditor.Box, s.sourceEditor.HasFocus(), s.prefs.Theme == "light")
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
	if s.helpText != nil {
		apply(s.helpText.Box, s.helpText.HasFocus(), false)
		apply(s.helpTopics.Box, s.helpTopics.HasFocus(), false)
	}
	if s.aboutText != nil {
		apply(s.aboutText.Box, s.aboutText.HasFocus(), false)
	}
	if s.editForm != nil {
		apply(s.editForm.Box, s.editForm.HasFocus(), false)
		apply(s.editInfo.Box, s.editInfo.HasFocus(), false)
	}
	if s.cellText != nil {
		apply(s.cellText.Box, s.cellText.HasFocus(), false)
		apply(s.cellForm.Box, s.cellForm.HasFocus(), false)
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
		// Invalidate the cached custom rectangle, then preserve the actual
		// border and padding configured by each form/help/list.
		box.SetRect(x, y, width, height)
		return box.GetInnerRect()
	})
}

// Confirmation cards share the form surface while retaining their safe default.
func (s *screen) styleModal(modal *tview.Modal) {
	s.styleBox(modal.Box)
	modal.SetBackgroundColor(s.colors().surface)
	singleLineBorder(modal.Box)
}

var lightPalette = themeColors{
	editor: tcell.ColorWhite, editorText: tcell.GetColor("#202a36"),
	background: tcell.GetColor("#f3f5f7"), surface: tcell.GetColor("#e4eaf0"), footer: tcell.GetColor("#ccd7e2"),
	field: tcell.ColorWhite, text: tcell.GetColor("#202a36"), muted: tcell.GetColor("#485869"),
	accent: tcell.GetColor("#276a8e"), button: tcell.GetColor("#bbd4e5"), fieldText: tcell.GetColor("#202a36"),
	selectionText: tcell.ColorWhite, placeholder: tcell.GetColor("#66758a"), warning: tcell.GetColor("#8b4f00"),
}

func (s *screen) colors() themeColors {
	if s.prefs.Theme == "light" {
		if s.limitedColors {
			return lightANSI
		}
		return lightPalette
	}
	colors := palette
	if s.limitedColors {
		colors.editor = tcell.ColorBlack
		colors.editorText = tcell.ColorWhite
	}
	return colors
}

// Defaults retained for small standalone widgets and tests; application widgets
// always use their screen's theme, with no process-wide mutable theme state.
func styleBox(b *tview.Box)    { (&screen{}).styleBox(b) }
func styleForm(f *tview.Form)  { (&screen{}).styleForm(f) }
func fieldStyle() tcell.Style  { return (&screen{}).fieldStyle() }
func focusStyle() tcell.Style  { return (&screen{}).focusStyle() }
func buttonStyle() tcell.Style { return (&screen{}).buttonStyle() }

var lightANSI = themeColors{
	editor: tcell.ColorWhite, editorText: tcell.ColorBlack,
	background: tcell.ColorSilver, surface: tcell.ColorSilver, footer: tcell.ColorWhite, field: tcell.ColorWhite,
	text: tcell.ColorBlack, muted: tcell.ColorNavy, accent: tcell.ColorNavy, button: tcell.ColorTeal,
	fieldText: tcell.ColorBlack, selectionText: tcell.ColorWhite, placeholder: tcell.ColorGray, warning: tcell.ColorMaroon,
}

func (s *screen) editorStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().editorText).Background(s.colors().editor)
}
func (s *screen) styleEditor(editor *tview.TextArea) {
	editor.SetBackgroundColor(s.colors().editor).SetTitleColor(s.colors().editorText)
	editor.SetTextStyle(s.editorStyle()).SetSelectedStyle(s.focusStyle())
}
