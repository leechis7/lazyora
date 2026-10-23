package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"unicode"
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

func (s *screen) refreshMenuLabels() {
	for i, button := range s.menuButtons {
		button.SetLabel(fmt.Sprintf("%s(%c)", s.tr(menuGroups[i].label), unicode.ToUpper(menuGroups[i].key)))
	}
}

func (s *screen) showMenu(index int) {
	page, _ := s.pages.GetFrontPage()
	if page != "main" && page != "menu" {
		return
	}
	if page == "menu" {
		s.closeMenu()
	}
	var entries []command
	for _, cmd := range commands {
		if cmd.group == index {
			entries = append(entries, cmd)
		}
	}
	s.openCommandMenu(s.tr(menuGroups[index].label), entries, index)
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
	s.menuParent = nil
	s.renderCommandMenu(title, entries, index)
}
func (s *screen) renderCommandMenu(title string, entries []command, index int) {
	s.pages.RemovePage("menu")
	s.menuIndex = index
	s.menuTitle = title
	s.menuEntries = entries
	list := tview.NewList().SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetShortcutColor(s.colors().accent).ShowSecondaryText(false).SetSelectedStyle(s.focusStyle())
	list.SetBorder(true).SetTitle(title)
	s.styleList(list)
	list.SetBackgroundColor(s.colors().surface)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface))
	list.SetBorderStyle(tcell.StyleDefault.Foreground(s.colors().button).Background(s.colors().surface))
	s.menuList = list
	s.menuRows = s.commandMenuRows(entries, index)
	for i, row := range s.menuRows {
		label := row.label
		if row.divider {
			label = "────────────────────────────────────────────"
		}
		if row.id != "" {
			cmd := commandFor(row.id)
			label = s.tr(cmd.label)
			if key := s.editorBindingLabels(cmd.id, 1); key != "" {
				label += "  " + key
			}
			// Evaluate against the original panel focus, not the menu widget.
			current := s.app.GetFocus()
			s.app.SetFocus(s.menuFocus)
			enabled := s.commandEnabled(cmd.id)
			s.app.SetFocus(current)
			if !enabled {
				label = "[" + s.tokens().disabled.String() + "]" + label + "[-:-:-]"
			}
		}
		at := i
		list.AddItem(label, "", 0, func() { s.menuRowAction(at) })
	}
	count := len(s.menuRows)
	list.SetInputCapture(s.menuListKey).SetMouseCapture(s.menuListMouse)
	// Flex leaves the underlying page untouched. Restrict its inner layout to
	// the menu instead of painting spacer Boxes over SQL and result panels.
	root := s.newFlex().AddItem(list, 0, 1, true)
	root.SetDrawFunc(func(_ tcell.Screen, x, y, width, height int) (int, int, int, int) {
		popupWidth := min(48, width)
		left := min(max(0, index)*width/len(menuGroups), max(0, width-popupWidth))
		// Recalculate on every draw so open menus also follow terminal resizing.
		return x + left, y, popupWidth, min(count+2, height)
	})
	s.pages.AddPage("menu", root, true, true)
	s.app.SetFocus(list)
}

func (s *screen) closeMenu() {
	s.pages.RemovePage("menu")
	s.menuList = nil
	s.menuRows = nil
	s.menuParent = nil
	s.menuEntries = nil
	if s.menuFocus != nil {
		s.app.SetFocus(s.menuFocus)
	}
}

func (s *screen) menuKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF10:
		if event.Modifiers() == 0 {
			if s.menuIndex < 0 {
				s.showMenu(menuFile)
			} else {
				s.closeMenu()
			}
			return nil
		}
		if event.Modifiers() == tcell.ModShift {
			context := s.menuIndex < 0
			s.closeMenu()
			if !context {
				s.showContextMenu()
			}
			return nil
		}
		return event
	case tcell.KeyEscape:
		if s.backCommandMenu() {
			return nil
		}
		s.closeMenu()
		return nil
	case tcell.KeyLeft:
		if s.backCommandMenu() {
			return nil
		}
		if s.menuIndex < 0 {
			return nil
		}
		s.showMenu((s.menuIndex + len(menuGroups) - 1) % len(menuGroups))
		return nil
	case tcell.KeyRight:
		if s.menuIndex < 0 {
			return nil
		}
		s.showMenu((s.menuIndex + 1) % len(menuGroups))
		return nil
	}
	return event
}
