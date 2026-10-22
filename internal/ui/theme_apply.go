package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"reflect"
)

// Recolor existing widgets, including hidden pages and tabs. Rebuilding pages
// would lose focus, selections, unfinished SQL, credentials and pending jobs.
func (s *screen) applyTheme(previous string) {
	old := screen{prefs: s.prefs, limitedColors: s.limitedColors}
	old.prefs.Theme = previous
	from, to := old.colors(), s.colors()
	oldGrid, newGrid := old.gridColors(), s.gridColors()
	pairs := [][2]tcell.Color{
		{from.editor, to.editor}, {from.editorText, to.editorText},
		{from.background, to.background}, {from.surface, to.surface}, {from.footer, to.footer},
		{from.field, to.field}, {from.text, to.text}, {from.muted, to.muted}, {from.accent, to.accent}, {from.button, to.button},
		{from.placeholder, to.placeholder}, {from.warning, to.warning},
		{oldGrid.background, newGrid.background}, {oldGrid.stripe, newGrid.stripe}, {oldGrid.header, newGrid.header},
		{oldGrid.current, newGrid.current}, {oldGrid.rangeBG, newGrid.rangeBG}, {oldGrid.divider, newGrid.divider}, {oldGrid.muted, newGrid.muted},
	}
	color := func(c tcell.Color) tcell.Color {
		for _, pair := range pairs {
			if c == pair[0] {
				return pair[1]
			}
		}
		return c
	}
	style := func(st tcell.Style) tcell.Style {
		fg, bg, _ := st.Decompose()
		return st.Foreground(color(fg)).Background(color(bg))
	}
	seen := map[tview.Primitive]bool{}
	var visit func(tview.Primitive)
	visit = func(p tview.Primitive) {
		if p == nil || reflect.ValueOf(p).IsNil() || seen[p] {
			return
		}
		seen[p] = true
		previousBG, previousBorder := tcell.ColorDefault, tcell.ColorDefault
		if b, ok := p.(interface {
			GetBackgroundColor() tcell.Color
			GetBorderColor() tcell.Color
		}); ok {
			previousBG, previousBorder = b.GetBackgroundColor(), b.GetBorderColor()
		}
		var box *tview.Box
		switch w := p.(type) {
		case *formPopupLayer:
			visit(w.Flex)
			return
		case *resultGrid:
			visit(w.Table)
			return
		case *tview.Pages:
			box = w.Box
			for _, name := range w.GetPageNames(false) {
				visit(w.GetPage(name))
			}
		case *tview.Flex:
			box = w.Box
			for i := 0; i < w.GetItemCount(); i++ {
				visit(w.GetItem(i))
			}
		case *tview.Form:
			box = w.Box
			s.styleForm(w)
			for i := 0; i < w.GetFormItemCount(); i++ {
				visit(w.GetFormItem(i))
			}
		case *choiceField:
			box = w.Box
			s.styleChoice(w)
		case *tview.InputField:
			box = w.Box
			w.SetFieldStyle(s.fieldStyle()).SetLabelStyle(tcell.StyleDefault.Foreground(to.text).Background(to.surface))
		case *tview.Checkbox:
			box = w.Box
			w.SetActivatedStyle(s.focusStyle())
		case *tview.TextArea:
			box = w.Box
			s.styleEditor(w)
		case *tview.TextView:
			box = w.Box
			fg := to.text
			if w.GetBackgroundColor() == from.field {
				fg = to.fieldText
			}
			w.SetTextColor(fg)
		case *tview.List:
			box = w.Box
			w.SetMainTextColor(to.text).SetSecondaryTextColor(to.muted).SetShortcutColor(to.accent).SetSelectedStyle(s.focusStyle())
			if w == s.menuList || w == s.workspaceList {
				w.SetMainTextStyle(tcell.StyleDefault.Foreground(to.text).Background(to.surface))
			}
		case *tview.TreeView:
			box = w.Box
			w.SetGraphicsColor(to.muted)
			if root := w.GetRoot(); root != nil {
				root.Walk(func(n, parent *tview.TreeNode) bool {
					n.SetTextStyle(style(n.GetTextStyle())).SetSelectedTextStyle(s.focusStyle())
					return true
				})
			}
		case *tview.Table:
			box = w.Box
			w.SetSelectedStyle(s.focusStyle()).SetBordersColor(to.muted)
			for row := 0; row < w.GetRowCount(); row++ {
				for col := 0; col < w.GetColumnCount(); col++ {
					cell := w.GetCell(row, col)
					cell.Color = color(cell.Color)
					cell.BackgroundColor = color(cell.BackgroundColor)
					cell.Style = style(cell.Style)
					cell.SelectedStyle = style(cell.SelectedStyle)
				}
			}
		case *tview.Button:
			box = w.Box
			w.SetStyle(s.buttonStyle()).SetActivatedStyle(buttonFocusStyle())
		case *tview.Modal:
			box = w.Box
			s.styleModal(w)
			w.SetTextColor(to.text).SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(buttonFocusStyle())
		case *tview.Box:
			box = w
		}
		if box != nil {
			box.SetBackgroundColor(color(previousBG)).SetBorderColor(color(previousBorder)).SetTitleColor(to.text)
			switch p.(type) {
			case *tview.TextArea:
				box.SetBackgroundColor(to.editor).SetTitleColor(to.editorText)
			case *tview.Form, *tview.Modal:
				box.SetBackgroundColor(to.surface)
			case *tview.TextView:
				if previousBG == from.field {
					box.SetBackgroundColor(to.field)
				}
			}
		}
	}
	visit(s.rootLayout)
	visit(s.pages)
	visit(s.editor)
	visit(s.table)
	visit(s.status)
	visit(s.footer)
	for _, tab := range s.tabs {
		visit(tab.editor)
		visit(tab.table)
		if s.limitedColors && tab.table != nil && tab.table != s.table {
			inactive := screen{prefs: s.prefs, limitedColors: s.limitedColors, table: tab.table, resultRows: tab.rows, resultHeaders: tab.headers, resultValues: tab.values, resultSelection: tab.selection}
			inactive.repaintGridTheme()
		}
	}
	for _, button := range s.menuButtons {
		visit(button)
	}
	for _, button := range s.buttons {
		button.SetStyle(tcell.StyleDefault.Foreground(to.text).Background(to.footer)).SetActivatedStyle(buttonFocusStyle())
	}
	if s.footerVersion != nil {
		s.footerVersion.SetTextColor(to.accent).SetBackgroundColor(to.footer)
	}
	if s.footer != nil {
		s.footer.SetTextColor(to.accent).SetBackgroundColor(to.surface)
	}
	if s.status != nil {
		s.status.SetTextColor(to.muted).SetBackgroundColor(to.background)
	}
	if s.helpText != nil && s.helpTopics != nil {
		row, col := s.helpText.GetScrollOffset()
		s.helpText.SetText(s.helpArticle(s.helpTopics.GetCurrentItem())).ScrollTo(row, col)
	}
	// ANSI grid colors have shared roles, so regenerate their decorative styles.
	if s.limitedColors && s.table != nil {
		s.repaintGridTheme()
	}
	s.refreshBorders()
}

func (s *screen) repaintGridTheme() {
	rows := map[int]bool{}
	for i := 1; i < s.table.GetRowCount(); i++ {
		rows[i] = true
	}
	s.paintResultRows(rows)
	bg := tcell.ColorBlack
	if s.prefs.Theme == "light" {
		bg = tcell.ColorWhite
	}
	s.table.SetBackgroundColor(bg)
}
