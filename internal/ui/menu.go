package ui

import (
	"fmt"
	"strings"

	"github.com/leechis7/tuivory/go"
)

const (
	menuFile = iota
	menuTabs
	menuEdit
	menuSession
	menuExecute
	menuView
	menuTools
	menuHelp
)

var menuGroups = []struct {
	label string
	key   rune
}{
	{"menu_file", 'f'}, {"menu_tabs", 'g'}, {"menu_edit", 'e'}, {"menu_session", 's'}, {"menu_execute", 'x'}, {"menu_view", 'v'}, {"menu_tools", 't'}, {"menu_help", 'h'},
}

type commandMenuView struct {
	*tuivory.MenuView
	model *tuivory.Menu
}

// Menu data and command eligibility belong to the application. Navigation,
// scrolling, disabled items, pointer handling and painting belong to Tuivory.
func (s *screen) menuItems(entries []command, parent string, index int) []tuivory.MenuItem {
	var items []tuivory.MenuItem
	for n, row := range s.commandMenuRows(entries, index) {
		id := fmt.Sprintf("%s/section-%d", parent, n)
		if row.divider {
			items = append(items, tuivory.MenuItem{ID: id, Parent: parent, Separator: true})
		} else if len(row.children) > 0 {
			items = append(items, tuivory.MenuItem{ID: id, Parent: parent, Label: row.label})
			items = append(items, s.menuItems(row.children, id, -1)...)
		} else {
			cmd := commandFor(row.id)
			items = append(items, tuivory.MenuItem{ID: parent + "/" + cmd.id, Parent: parent, Label: s.tr(cmd.label), Shortcut: s.editorBindingLabels(cmd.id, 1), Disabled: !s.commandEnabled(cmd.id)})
		}
	}
	return items
}
func menuRoot(index int) string { return fmt.Sprintf("group-%d", index) }
func (s *screen) refreshMenuLabels() {
	var items []tuivory.MenuItem
	for index, group := range menuGroups {
		root := menuRoot(index)
		items = append(items, tuivory.MenuItem{ID: root, Label: s.tr(group.label), Mnemonic: string(group.key)})
		var entries []command
		for _, cmd := range commands {
			if cmd.group == index {
				entries = append(entries, cmd)
			}
		}
		items = append(items, s.menuItems(entries, root, index)...)
	}
	if s.mainMenu == nil {
		s.mainMenu, _ = tuivory.NewMenu(items)
	} else {
		_, err := s.mainMenu.SetItems(items)
		if err != nil {
			panic(err)
		}
	}
	s.mainMenu.BackLabel = strings.TrimPrefix(s.tr("mg_back"), "← ")
	if s.menuBarView != nil {
		_ = s.menuBarView.SetPalette(s.componentPalette())
	}
}
func (s *screen) showMenu(index int) {
	page, _ := s.pages.GetFrontPage()
	if page != "main" && page != "menu" {
		return
	}
	if index < 0 || index >= len(menuGroups) {
		return
	}
	if page == "menu" {
		s.closeMenu()
	}
	s.menuFocus = s.app.GetFocus()
	s.refreshMenuLabels()
	s.menuIndex = index
	s.openMenuModel(s.mainMenu, menuRoot(index))
}
func (s *screen) showContextMenu() {
	page, _ := s.pages.GetFrontPage()
	if page != "main" {
		return
	}
	ids := []string{"file_new", "file_close", "execute", "script", "plan", "undo", "redo", "cut", "copy", "paste", "select_all", "find", "replace", "clear_sql", "history", "describe", "complete", "format_code", "duplicate_line", "line_up", "line_down", "fullscreen_sql", "maximize"}
	if s.table.HasFocus() {
		ids = []string{"copy", "copy_row", "export", "copy_preview", "grid_widen", "grid_narrow", "grid_auto", "fetch_more", "result_sort_asc", "result_sort_desc", "result_filter", "result_view_reset", "fullscreen_results", "maximize", "focus_toggle", "cell_detail", "grid_edit", "table_edit_load", "grid_inline"}
	}
	if s.browserHasFocus() {
		ids = []string{"describe", "browser_refresh", "browser_user", "browser_type", "browser", "focus_toggle", "table_edit_load"}
	}
	var entries []command
	for _, id := range ids {
		entries = append(entries, commandFor(id))
	}
	s.openCommandMenu(s.tr("context_menu"), entries, -1)
}

func (s *screen) openCommandMenu(title string, entries []command, index int) {
	s.menuFocus = s.app.GetFocus()
	model, err := tuivory.NewMenu(s.menuItems(entries, "", -1))
	if err != nil {
		panic(err)
	}
	model.BackLabel = strings.TrimPrefix(s.tr("mg_back"), "← ")
	s.menuIndex = index
	s.menuTitle = title
	s.openMenuModel(model, "")
}
func (s *screen) openMenuModel(model *tuivory.Menu, root string) {
	s.menuModel = model
	model.Open(root, "panel")
	view, err := tuivory.NewMenuView(model, s.componentPalette(), s.asciiDisplay(), s.menuEvent)
	if err != nil {
		panic(err)
	}
	view.SetBackgroundColor(s.colors().surface)
	s.menuList = &commandMenuView{MenuView: view, model: model}
	// The published PanelView recognizes the raw MenuView child and owns the
	// single border, literal title clipping, opaque surface and wide-glyph repair.
	card, err := tuivory.NewPanelView(view, s.componentPalette())
	if err != nil {
		panic(err)
	}
	card.SetBorderPadding(0, 0, 0, 0)
	s.menuCard = card
	rootView := s.newFlex().AddItem(card, 0, 1, true)
	rootView.SetDrawFunc(func(_ tuivory.Screen, x, y, w, h int) (int, int, int, int) {
		cw, ch := model.ContentSize()
		viewport := tuivory.CellRect{X: x, Y: y, Width: w, Height: h}
		anchor := tuivory.CellRect{X: x, Y: y}
		if s.menuIndex >= 0 {
			barX, barY, barWidth, barHeight := x, y, w, 0
			if s.menuBarView != nil {
				bx, by, bw, bh := s.menuBarView.GetInnerRect()
				if bw > 0 && bh == 1 { // Ignore a not-yet-laid-out widget's default rectangle.
					barX, barY, barWidth, barHeight = bx, by, bw, bh
				}
			}
			for _, cell := range model.BarLayout(barWidth).Cells {
				if cell.ID == menuRoot(s.menuIndex) {
					anchor = tuivory.CellRect{X: barX + cell.Start, Y: barY, Width: cell.Width, Height: barHeight}
					break
				}
			}
		}
		placement := tuivory.MenuPlacement(anchor, viewport, min(64, cw+2), ch+2)
		title := model.Title()
		if title == "" {
			title = s.menuTitle
		}
		_ = card.SetPalette(s.componentPalette())
		card.ASCII = s.asciiDisplay()
		card.SetTitle(title)
		view.ASCII = s.asciiDisplay()
		_ = view.SetPalette(s.componentPalette())
		return placement.X, placement.Y, placement.Width, placement.Height
	})
	s.pages.AddPage("menu", rootView, true, true)
	s.app.SetFocus(s.menuList)
}
func (s *screen) menuEvent(e tuivory.MenuEvent) {
	if e.Kind == "activate" {
		id := e.ID
		for _, cmd := range commands {
			if strings.HasSuffix(id, "/"+cmd.id) {
				s.closeMenu()
				s.invoke(cmd.id)
				return
			}
		}
	}
	if e.Kind == "close" {
		s.closeMenu()
		return
	}
	if s.menuModel != nil && s.menuModel.Opened() {
		if path := s.menuModel.Path(); len(path) > 0 && s.menuIndex >= 0 {
			for i := range menuGroups {
				if path[0] == menuRoot(i) {
					s.menuIndex = i
				}
			}
		}
		s.app.SetFocus(s.menuList)
	}
}
func (s *screen) closeMenu() {
	if s.menuModel != nil {
		s.menuModel.Close()
	}
	s.pages.RemovePage("menu")
	s.menuList = nil
	s.menuModel = nil
	s.menuCard = nil
	if s.menuFocus != nil {
		s.app.SetFocus(s.menuFocus)
	}
}
func (s *screen) menuKey(event *tuivory.EventKey) *tuivory.EventKey {
	switch event.Key() {
	case tuivory.KeyF10:
		if event.Modifiers() == 0 {
			if s.menuIndex < 0 {
				s.showMenu(menuFile)
			} else {
				s.closeMenu()
			}
			return nil
		}
		if event.Modifiers() == tuivory.ModShift {
			context := s.menuIndex < 0
			s.closeMenu()
			if !context {
				s.showContextMenu()
			}
			return nil
		}
	case tuivory.KeyLeft:
		s.menuEvent(s.menuModel.Handle("left", "", 1))
		return nil
	case tuivory.KeyRight:
		// Preserve LazyOra's adjacent top-level navigation, including grouped menus.
		// Enter opens children; context menus never jump into the main menu bar.
		if s.menuIndex >= 0 {
			e := s.menuModel.HandleBar("next", "")
			s.menuEvent(s.menuModel.Open(e.ID, s.menuModel.ReturnID()))
		}
		return nil
	case tuivory.KeyEscape:
		s.menuEvent(s.menuModel.Back())
		return nil
	}
	return event
}
