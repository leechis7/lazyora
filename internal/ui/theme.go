package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
	"github.com/rivo/tview"
	"reflect"
)

// Product adapters retain the existing field names; palette resolution lives in Tuivory.
type themeColors struct {
	background, surface, footer, field, text, muted, accent, button    tcell.Color
	editor, editorText, fieldText, selectionText, placeholder, warning tcell.Color
}

func (s *screen) fieldStyle() tcell.Style {
	return s.tuivoryStyles().FieldStyle()
}

func (s *screen) focusStyle() tcell.Style {
	return s.tuivoryStyles().SelectionStyle()
}

// Standalone helpers share the same Tuivory defaults as application tuivory.
func buttonFocusStyle() tcell.Style {
	return (&screen{}).buttonFocusStyle()
}

func (s *screen) monochrome() bool { return s.prefs.Theme == "blackwhite" }

func (s *screen) buttonFocusStyle() tcell.Style {
	return s.tuivoryStyles().ButtonFocusStyle()
}

func (s *screen) buttonStyle() tcell.Style {
	return s.tuivoryStyles().ButtonStyle()
}

func (s *screen) styleBox(box *tview.Box) {
	s.setSurface(box, s.surfaceRoles[box])
	s.tuivoryWidgets().Box(box, s.surfaceColor(s.surfaceRoles[box]), s.surfaceText(s.surfaceRoles[box]))
}

func (s *screen) styleForm(form *tview.Form) {
	s.setSurface(form.Box, surfacePanel)
	s.tuivoryWidgets().Form(form)
	for i := 0; i < form.GetFormItemCount(); i++ {
		if box := widgetBox(form.GetFormItem(i)); box != nil {
			s.setSurface(box, surfacePanel)
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
	return tuivory.BorderColor(s.tuivoryStyles().Theme(), s.tuivoryProfile(), s.tuivoryStyles(), focused, background)
}

// Confirmation cards share the form surface while retaining their safe default.
func (s *screen) styleModal(modal *tview.Modal) {
	s.setSurface(modal.Box, surfacePanel)
	s.tuivoryWidgets().Modal(modal)
}

func (s *screen) colors() themeColors { return s.tokens().themeColors }

// Defaults retained for small standalone widgets and tests; application widgets
// always use their screen's theme, with no process-wide mutable theme state.
func styleBox(b *tview.Box)    { (&screen{}).styleBox(b) }
func styleForm(f *tview.Form)  { (&screen{}).styleForm(f) }
func fieldStyle() tcell.Style  { return (&screen{}).fieldStyle() }
func focusStyle() tcell.Style  { return (&screen{}).focusStyle() }
func buttonStyle() tcell.Style { return (&screen{}).buttonStyle() }

func (s *screen) editorStyle() tcell.Style {
	return s.tuivoryStyles().EditorStyle()
}
func (s *screen) styleEditor(editor *tview.TextArea) {
	s.setSurface(editor.Box, surfaceEditor)
	s.tuivoryWidgets().Editor(editor)
}
