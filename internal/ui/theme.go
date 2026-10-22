package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"reflect"
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

func (s *screen) monochrome() bool { return s.prefs.Theme == "blackwhite" }

func (s *screen) buttonFocusStyle() tcell.Style {
	if s.monochrome() {
		return tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite).Bold(true).Underline(true)
	}
	return buttonFocusStyle()
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
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetLabelColor(s.colors().muted).SetFieldStyle(s.fieldStyle()).
		SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(s.buttonFocusStyle())
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
			s.styleCheckbox(item)
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

// Refresh every visible widget family, including nested popup cards. Focused
// boxes use tview's native double border; inactive boxes retain a single border.
func (s *screen) refreshBorders() {
	seen := map[tview.Primitive]bool{}
	var visit func(tview.Primitive)
	visit = func(p tview.Primitive) {
		if p == nil || reflect.ValueOf(p).IsNil() || seen[p] {
			return
		}
		seen[p] = true
		if box, ok := p.(interface {
			GetBackgroundColor() tcell.Color
			SetBorderStyle(tcell.Style) *tview.Box
			SetTitleColor(tcell.Color) *tview.Box
		}); ok {
			focused := p.HasFocus()
			bg := box.GetBackgroundColor()
			color := s.borderColor(focused, bg)
			box.SetBorderStyle(tcell.StyleDefault.Foreground(color).Background(bg).Bold(focused))
			title := s.colors().text
			if focused {
				title = color
			}
			box.SetTitleColor(title)
		}
		switch w := p.(type) {
		case *formPopupLayer:
			visit(w.Flex)
		case *resultGrid:
			visit(w.Table)
		case *tview.Pages:
			for _, name := range w.GetPageNames(false) {
				visit(w.GetPage(name))
			}
		case *tview.Flex:
			for i := 0; i < w.GetItemCount(); i++ {
				visit(w.GetItem(i))
			}
		case *tview.Form:
			for i := 0; i < w.GetFormItemCount(); i++ {
				visit(w.GetFormItem(i))
			}
			for i := 0; i < w.GetButtonCount(); i++ {
				visit(w.GetButton(i))
			}
		}
	}
	visit(s.rootLayout)
	visit(s.pages)
	visit(s.editor)
	visit(s.table)
	// Closed pages must not be retained by semantic text-style bookkeeping.
	for view := range s.textRoles {
		if !seen[view] {
			delete(s.textRoles, view)
		}
	}
}

func (s *screen) borderColor(focused bool, background tcell.Color) tcell.Color {
	if s.monochrome() {
		if focused {
			return tcell.ColorWhite
		}
		return tcell.ColorSilver
	}
	light := s.prefs.Theme == "light" || background == s.colors().field || background == tcell.ColorWhite
	if s.limitedColors {
		if focused {
			if light {
				return tcell.ColorNavy
			}
			return tcell.ColorYellow
		}
		if light {
			return tcell.ColorBlack
		}
		return tcell.ColorSilver
	}
	if focused {
		if light {
			return tcell.GetColor("#005fcc")
		}
		return tcell.GetColor("#5ee4f3")
	}
	if light {
		return tcell.GetColor("#8190a4")
	}
	return tcell.GetColor("#66758a")
}

// Confirmation cards share the form surface while retaining their safe default.
func (s *screen) styleModal(modal *tview.Modal) {
	s.styleBox(modal.Box)
	modal.SetBackgroundColor(s.colors().surface).SetTextColor(s.colors().text).
		SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(s.buttonFocusStyle())
}

var lightPalette = themeColors{
	editor: tcell.GetColor("#ffffff"), editorText: tcell.GetColor("#202a36"),
	background: tcell.GetColor("#f3f5f7"), surface: tcell.GetColor("#e4eaf0"), footer: tcell.GetColor("#ccd7e2"),
	field: tcell.GetColor("#ffffff"), text: tcell.GetColor("#202a36"), muted: tcell.GetColor("#485869"),
	accent: tcell.GetColor("#276a8e"), button: tcell.GetColor("#bbd4e5"), fieldText: tcell.GetColor("#202a36"),
	selectionText: tcell.ColorWhite, placeholder: tcell.GetColor("#66758a"), warning: tcell.GetColor("#8b4f00"),
}

func (s *screen) colors() themeColors {
	if s.monochrome() {
		return blackWhitePalette
	}
	if s.prefs.Theme == "light" {
		if s.limitedColors {
			return lightANSI
		}
		return lightPalette
	}
	if s.limitedColors {
		return darkANSI
	}
	return palette
}

var blackWhitePalette = themeColors{
	editor: tcell.ColorBlack, editorText: tcell.ColorWhite,
	background: tcell.ColorBlack, surface: tcell.ColorBlack, footer: tcell.ColorBlack,
	field: tcell.ColorWhite, text: tcell.ColorWhite, muted: tcell.ColorSilver,
	accent: tcell.ColorWhite, button: tcell.ColorSilver, fieldText: tcell.ColorBlack,
	selectionText: tcell.ColorBlack, placeholder: tcell.ColorGray, warning: tcell.ColorWhite,
}

// Neutral ANSI surfaces avoid RGB grays being approximated as cyan.
var darkANSI = themeColors{
	editor: tcell.ColorBlack, editorText: tcell.ColorSilver,
	background: tcell.ColorBlack, surface: tcell.ColorBlack, footer: tcell.ColorBlack,
	field: tcell.ColorSilver, text: tcell.ColorSilver, muted: tcell.ColorSilver,
	accent: tcell.ColorYellow, button: tcell.ColorSilver, fieldText: tcell.ColorBlack,
	selectionText: tcell.ColorBlack, placeholder: tcell.ColorNavy, warning: tcell.ColorYellow,
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
	text: tcell.ColorBlack, muted: tcell.ColorNavy, accent: tcell.ColorNavy, button: tcell.ColorSilver,
	fieldText: tcell.ColorBlack, selectionText: tcell.ColorWhite, placeholder: tcell.ColorGray, warning: tcell.ColorMaroon,
}

func (s *screen) editorStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().editorText).Background(s.colors().editor)
}
func (s *screen) styleEditor(editor *tview.TextArea) {
	editor.SetBackgroundColor(s.colors().editor).SetTitleColor(s.colors().editorText)
	editor.SetTextStyle(s.editorStyle()).SetSelectedStyle(s.focusStyle())
}
