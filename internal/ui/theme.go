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
	fieldText:  tcell.GetColor("#2e3440"), selectionText: tcell.GetColor("#2e3440"), placeholder: tcell.GetColor("#485869"), warning: tcell.ColorYellow,
}

func (s *screen) fieldStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().fieldText).Background(s.colors().field)
}

func (s *screen) focusStyle() tcell.Style {
	t := s.tokens()
	return tcell.StyleDefault.Foreground(t.selectionFG).Background(t.selection)
}

// A separate button focus palette stays distinct even when pale RGB colors
// map to the same color on an 8/16-color terminal.
func buttonFocusStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy).Bold(true).Underline(true)
}

func (s *screen) monochrome() bool { return s.prefs.Theme == "blackwhite" }

func (s *screen) buttonFocusStyle() tcell.Style { return s.focusStyle().Underline(true) }

func (s *screen) buttonStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.colors().fieldText).Background(s.colors().button)
}

func (s *screen) styleBox(box *tview.Box) {
	s.setSurface(box, s.surfaceRoles[box])
	box.SetBorderColor(s.colors().button).SetTitleColor(s.surfaceText(s.surfaceRoles[box]))
}

func (s *screen) styleForm(form *tview.Form) {
	s.styleBox(form.Box)
	s.setSurface(form.Box, surfacePanel)
	form.SetBorderPadding(1, 1, 2, 2)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetLabelColor(s.colors().muted).SetFieldStyle(s.fieldStyle()).
		SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(s.buttonFocusStyle()).SetButtonDisabledStyle(s.disabledStyle())
	for index := 0; index < form.GetFormItemCount(); index++ {
		switch item := form.GetFormItem(index).(type) {
		case *tview.InputField:
			s.setSurface(item.Box, surfacePanel)
			item.SetPlaceholderStyle(tcell.StyleDefault.Foreground(s.colors().placeholder).Background(s.colors().field))
			// Form reapplies field attributes before Draw. Set the focused input's
			// contrast here without replacing InputField or disturbing its caret.
			item.SetDrawFunc(func(_ tcell.Screen, x, y, width, height int) (int, int, int, int) {
				label := tcell.StyleDefault.Foreground(s.colors().muted).Background(s.colors().surface)
				input := s.fieldStyle()
				if item.HasFocus() {
					label = label.Foreground(s.colors().accent).Bold(true)
					input = input.Underline(true)
				}
				item.SetLabelStyle(label).SetFieldStyle(input)
				return x, y, width, height
			})
		case *tview.TextView:
			// Form supplies input text colors even for read-only items before Draw.
			// Reapply the description role after those attributes, without changing data.
			s.textSurface(item, surfacePanel)
			item.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
				item.SetTextColor(s.colors().text)
				return x, y, w, h
			})
		case *choiceField:
			item.SetFocusedStyle(s.focusStyle().Underline(true))
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
	liveBoxes := map[*tview.Box]bool{}
	liveNodes := map[*tview.TreeNode]bool{}
	var visit func(tview.Primitive)
	visit = func(p tview.Primitive) {
		if p == nil || reflect.ValueOf(p).IsNil() || seen[p] {
			return
		}
		seen[p] = true
		if box := widgetBox(p); box != nil {
			liveBoxes[box] = true
		}
		if box, ok := p.(interface {
			GetBackgroundColor() tcell.Color
			SetBorderStyle(tcell.Style) *tview.Box
			SetTitleColor(tcell.Color) *tview.Box
		}); ok {
			focused := p.HasFocus()
			bg := box.GetBackgroundColor()
			color := s.borderColor(focused, bg)
			box.SetBorderStyle(tcell.StyleDefault.Foreground(color).Background(bg))
			title := s.colors().text
			if b, ok := p.(interface{ GetBackgroundColor() tcell.Color }); ok {
				if b.GetBackgroundColor() == s.colors().field {
					title = s.colors().fieldText
				}
				if b.GetBackgroundColor() == s.colors().editor {
					title = s.colors().editorText
				}
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
		case *tview.TreeView:
			if root := w.GetRoot(); root != nil {
				root.Walk(func(n, parent *tview.TreeNode) bool { liveNodes[n] = true; return true })
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
	for _, tab := range s.tabs {
		visit(tab.editor)
		visit(tab.table)
	}
	visit(s.browserPanel)
	visit(s.tabBar)
	visit(s.footerBar)
	visit(s.connectionHeader)
	visit(s.menuLayout)
	visit(s.footer)
	for box := range s.surfaceRoles {
		if !liveBoxes[box] {
			delete(s.surfaceRoles, box)
		}
	}
	for node := range s.treeRoles {
		if !liveNodes[node] {
			delete(s.treeRoles, node)
		}
	}
	// Closed pages must not be retained by semantic text-style bookkeeping.
	for view := range s.textRoles {
		if !seen[view] {
			delete(s.textRoles, view)
		}
	}
}

func (s *screen) borderColor(focused bool, background tcell.Color) tcell.Color {
	r, g, b := background.RGB()
	light := r >= 0 && 299*r+587*g+114*b >= 128000
	if s.monochrome() {
		if light {
			return tcell.ColorBlack
		}
		return tcell.ColorSilver
	}
	if focused {
		if light {
			if s.ansiColors() {
				return tcell.ColorNavy
			}
			if s.colorProfile() == profile256 {
				return indexed(25)
			}
			return tcell.GetColor("#005fcc")
		}
		if s.ansiColors() {
			if s.colorProfile() == profileANSI8 {
				return indexed(3)
			}
			return tcell.ColorWhite
		}
		return s.tokens().focus
	}
	if s.ansiColors() {
		if light {
			return tcell.ColorBlack
		}
		return tcell.ColorSilver
	}
	if s.colorProfile() == profile256 {
		return indexed(244)
	}
	if light {
		return tcell.GetColor("#66758a")
	}
	return tcell.GetColor("#8190a4")
}

// Confirmation cards share the form surface while retaining their safe default.
func (s *screen) styleModal(modal *tview.Modal) {
	s.styleBox(modal.Box)
	s.setSurface(modal.Box, surfacePanel)
	modal.SetBackgroundColor(s.colors().surface).SetTextColor(s.colors().text).
		SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(s.buttonFocusStyle())
}

var lightPalette = themeColors{
	editor: tcell.GetColor("#ffffff"), editorText: tcell.GetColor("#202a36"),
	background: tcell.GetColor("#f3f5f7"), surface: tcell.GetColor("#e4eaf0"), footer: tcell.GetColor("#ccd7e2"),
	field: tcell.GetColor("#ffffff"), text: tcell.GetColor("#202a36"), muted: tcell.GetColor("#485869"),
	accent: tcell.GetColor("#276a8e"), button: tcell.GetColor("#bbd4e5"), fieldText: tcell.GetColor("#202a36"),
	selectionText: tcell.ColorWhite, placeholder: tcell.GetColor("#485869"), warning: tcell.GetColor("#8b4f00"),
}

func (s *screen) colors() themeColors { return s.tokens().themeColors }

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
	s.setSurface(editor.Box, surfaceEditor)
	editor.SetTitleColor(s.colors().editorText)
	editor.SetTextStyle(s.editorStyle()).SetSelectedStyle(s.focusStyle())
}
