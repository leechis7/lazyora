package ui

import (
	"reflect"

	"github.com/leechis7/tuivory/go"
)

// Product adapters retain the existing field names; palette resolution lives in Tuivory.
type themeColors struct {
	background, surface, footer, field, text, muted, accent, button    tuivory.Color
	editor, editorText, fieldText, selectionText, placeholder, warning tuivory.Color
}

func (s *screen) fieldStyle() tuivory.Style {
	return s.tuivoryStyles().TextStyle(tuivory.Field, tuivory.Body)
}

func (s *screen) focusStyle() tuivory.Style {
	return s.tuivoryStyles().SelectionStyle()
}

// Standalone helpers share the same Tuivory defaults as application tuivory.
func buttonFocusStyle() tuivory.Style {
	return (&screen{}).buttonFocusStyle()
}

func (s *screen) monochrome() bool { return s.prefs.Theme == "blackwhite" }

func (s *screen) buttonFocusStyle() tuivory.Style {
	return s.tuivoryStyles().ButtonFocusStyle()
}

func (s *screen) buttonStyle() tuivory.Style {
	return s.tuivoryStyles().ButtonStyle()
}

func (s *screen) styleBox(box *tuivory.Box) {
	s.setSurface(box, s.surfaceRoles[box])
	s.tuivoryWidgets().Box(box, s.surfaceColor(s.surfaceRoles[box]), s.surfaceText(s.surfaceRoles[box]))
}

func (s *screen) styleForm(form *tuivory.WidgetForm) {
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
	seen := map[tuivory.Primitive]bool{}
	liveBoxes := map[*tuivory.Box]bool{}
	liveNodes := map[*tuivory.WidgetTreeNode]bool{}
	var visit func(tuivory.Primitive)
	visit = func(p tuivory.Primitive) {
		if p == nil || reflect.ValueOf(p).IsNil() || seen[p] {
			return
		}
		seen[p] = true
		if box := widgetBox(p); box != nil {
			liveBoxes[box] = true
		}
		if box, ok := p.(interface {
			GetBackgroundColor() tuivory.Color
			SetBorderStyle(tuivory.Style) *tuivory.Box
			SetTitleColor(tuivory.Color) *tuivory.Box
		}); ok {
			focused := p.HasFocus()
			bg := box.GetBackgroundColor()
			color := s.borderColor(focused, bg)
			box.SetBorderStyle(tuivory.StyleDefault.Foreground(color).Background(bg))
			title := s.colors().text
			if b, ok := p.(interface{ GetBackgroundColor() tuivory.Color }); ok {
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
		case *displayLayout:
			visit(w.Flex)
			return
		case *formPopupLayer:
			visit(w.Flex)
		case *tuivory.ShadowView:
			visit(w.Child)
		case *tuivory.PanelView:
			visit(w.Child)
		case *resultGrid:
			visit(w.Table)
		case *tuivory.Pages:
			for _, name := range w.GetPageNames(false) {
				visit(w.GetPage(name))
			}
		case *tuivory.Flex:
			for i := 0; i < w.GetItemCount(); i++ {
				visit(w.GetItem(i))
			}
		case *browserTree:
			if root := w.GetRoot(); root != nil {
				root.Walk(func(n, parent *tuivory.WidgetTreeNode) bool { liveNodes[n] = true; return true })
			}
		case *tuivory.WidgetTreeView:
			if root := w.GetRoot(); root != nil {
				root.Walk(func(n, parent *tuivory.WidgetTreeNode) bool { liveNodes[n] = true; return true })
			}
		case *tuivory.WidgetForm:
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

func (s *screen) borderColor(focused bool, background tuivory.Color) tuivory.Color {
	return tuivory.BorderColor(s.tuivoryStyles().Theme(), s.tuivoryProfile(), s.tuivoryStyles(), focused, background)
}

// Confirmation cards share the form surface while retaining their safe default.
func (s *screen) styleModal(modal *tuivory.Modal) {
	s.setSurface(modal.Box, surfacePanel)
	s.tuivoryWidgets().Modal(modal)
}

func (s *screen) colors() themeColors { return s.tokens().themeColors }

// Defaults retained for small standalone widgets and tests; application widgets
// always use their screen's theme, with no process-wide mutable theme state.
func styleBox(b *tuivory.Box)         { (&screen{}).styleBox(b) }
func styleForm(f *tuivory.WidgetForm) { (&screen{}).styleForm(f) }
func fieldStyle() tuivory.Style       { return (&screen{}).fieldStyle() }
func focusStyle() tuivory.Style       { return (&screen{}).focusStyle() }
func buttonStyle() tuivory.Style      { return (&screen{}).buttonStyle() }

func (s *screen) editorStyle() tuivory.Style {
	return s.tuivoryStyles().EditorStyle()
}
func (s *screen) styleEditor(editor *tuivory.TextArea) {
	s.setSurface(editor.Box, surfaceEditor)
	s.tuivoryWidgets().Editor(editor)
}
