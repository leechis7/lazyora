package ui

import (
	"reflect"

	"github.com/leechis7/tuivory/go"
)

// Recolor existing widgets, including hidden pages and tabs. Rebuilding pages
// would lose focus, selections, unfinished SQL, credentials and pending jobs.
func (s *screen) applyTheme(previous string) {
	to := s.colors()
	seen := map[tuivory.Primitive]bool{}
	var visit func(tuivory.Primitive)
	visit = func(p tuivory.Primitive) {
		if p == nil || reflect.ValueOf(p).IsNil() || seen[p] {
			return
		}
		seen[p] = true
		var box *tuivory.Box
		switch w := p.(type) {
		case *displayLayout:
			visit(w.Flex)
			return
		case *formPopupLayer:
			visit(w.Flex)
			return
		case *resultGrid:
			visit(w.Table)
			return
		case *tuivory.ShadowView:
			_ = w.SetPalette(s.componentPalette())
			visit(w.Child)
			return
		case *tuivory.MenuView:
			_ = w.SetPalette(s.componentPalette())
			w.ASCII = s.asciiDisplay()
			return
		case *tuivory.PanelView:
			w.ASCII = s.asciiDisplay()
			_ = w.SetPalette(s.componentPalette())
			visit(w.Child)
			return
		case *commandMenuView:
			_ = w.SetPalette(s.componentPalette())
			return
		case *tuivory.RowsView:
			_ = w.SetPalette(s.componentPalette())
			return
		case *tuivory.Pages:
			box = w.Box
			for _, name := range w.GetPageNames(false) {
				visit(w.GetPage(name))
			}
		case *tuivory.Flex:
			box = w.Box
			for i := 0; i < w.GetItemCount(); i++ {
				visit(w.GetItem(i))
			}
		case *tuivory.WidgetForm:
			box = w.Box
			s.styleForm(w)
			for i := 0; i < w.GetFormItemCount(); i++ {
				visit(w.GetFormItem(i))
			}
		case *choiceField:
			box = w.Box
			s.styleChoice(w)
		case *tuivory.InputField:
			box = w.Box
			w.SetFieldStyle(s.fieldStyle()).SetLabelStyle(tuivory.StyleDefault.Foreground(to.text).Background(s.surfaceColor(s.surfaceRoles[w.Box])))
		case *tuivory.Checkbox:
			box = w.Box
			s.styleCheckbox(w)
		case *tuivory.TextArea:
			box = w.Box
			s.styleEditor(w)
		case *tuivory.WidgetTextView:
			box = w.Box
			s.textSurface(w, s.surfaceRoles[w.Box])
		case *tuivory.WidgetList:
			box = w.Box
			s.styleList(w)
		case *tuivory.WidgetTreeView:
			box = w.Box
			roles := make(map[*tuivory.WidgetTreeNode]tuivory.TextRole, len(s.treeRoles))
			for node, role := range s.treeRoles {
				roles[node] = tuivory.TextRole(role)
			}
			s.tuivoryWidgets().Tree(w, roles)
		case *tuivory.Table:
			box = w.Box
			s.styleDataTable(w)
		case *tuivory.WidgetButton:
			box = w.Box
			s.tuivoryWidgets().Button(w)
		case *tuivory.Modal:
			box = w.Box
			s.styleModal(w)
			w.SetTextColor(to.text).SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(s.buttonFocusStyle())
		case *tuivory.Box:
			box = w
		}
		if box != nil {
			role := s.surfaceRoles[box]
			switch p.(type) {
			case *tuivory.TextArea:
				role = surfaceEditor
			case *tuivory.WidgetForm, *tuivory.Modal, *tuivory.WidgetList, *tuivory.Checkbox:
				role = surfacePanel
			}
			s.setSurface(box, role)
			box.SetTitleColor(s.surfaceText(role))
		}
	}
	visit(s.rootLayout)
	visit(s.pages)
	visit(s.tabBar)
	visit(s.browserPanel)
	visit(s.editor)
	visit(s.table)
	visit(s.status)
	visit(s.footer)
	for _, tab := range s.tabs {
		visit(tab.editor)
		visit(tab.table)
		if tab.table != nil && tab.table != s.table {
			inactive := screen{prefs: s.prefs, terminal: s.terminal, limitedColors: s.limitedColors, table: tab.table, resultRows: tab.rows, resultHeaders: tab.headers, resultValues: tab.values, resultSelection: tab.selection, resultBrowser: tab.browser, gridDrafts: s.gridDrafts}
			inactive.repaintGridTheme()
		}
	}
	for _, button := range s.buttons {
		s.tuivoryWidgets().FooterButton(button)
	}
	if s.footerVersion != nil {
		s.footerVersion.SetTextColor(to.text).SetBackgroundColor(to.footer)
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
	if s.reference != nil {
		r := s.reference
		row, col := r.text.GetScrollOffset()
		r.text.SetBackgroundColor(to.background)
		r.text.SetText(s.referenceContent(r.articles[r.topics.GetCurrentItem()])).ScrollTo(row, col)
	}
	if s.aboutText != nil {
		s.aboutText.SetBackgroundColor(to.surface)
	}
	if s.connectionHeader != nil && s.connectionIdentity.Environment == "prod" {
		s.connectionHeader.SetTextColor(to.warning)
	}
	// ANSI grid colors have shared roles, so regenerate their decorative tuivory.
	if s.table != nil {
		s.repaintGridTheme()
	}
	s.styleTabs()
	s.styleKeymapComparison()
	s.refreshBorders()
}

func (s *screen) repaintGridTheme() {
	rows := map[int]bool{}
	for i := 1; i < s.table.GetRowCount(); i++ {
		rows[i] = true
	}
	s.paintResultRows(rows)
	bg := s.gridColors().background
	s.table.SetBackgroundColor(bg)
	for col := 0; col < s.table.GetColumnCount(); col++ {
		s.table.GetCell(0, col).SetStyle(s.gridHeaderStyle(false))
	}
}
