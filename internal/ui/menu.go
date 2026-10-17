package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"unicode"
)

var menuGroups = []struct {
	label string
	key   rune
}{
	{"menu_file", 'f'}, {"menu_edit", 'e'}, {"menu_session", 's'}, {"menu_execute", 'x'}, {"menu_view", 'v'}, {"menu_tools", 't'}, {"menu_help", 'h'},
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
	ids := []string{"execute", "find", "replace", "clear_sql", "history", "describe", "copy", "fullscreen_sql", "maximize"}
	if s.table.HasFocus() {
		ids = []string{"copy", "copy_row", "copy_preview", "fullscreen_results", "maximize", "focus_toggle"}
	}
	if s.browserHasFocus() {
		ids = []string{"describe", "browser", "focus_toggle"}
	}
	var entries []command
	for _, id := range ids {
		entries = append(entries, commandFor(id))
	}
	s.openCommandMenu(s.tr("context_menu"), entries, -1)
}

func (s *screen) openCommandMenu(title string, entries []command, index int) {
	s.menuFocus = s.app.GetFocus()
	s.menuIndex = index
	list := tview.NewList().ShowSecondaryText(false).SetSelectedStyle(focusStyle())
	list.SetBorder(true).SetTitle(title)
	styleBox(list.Box)
	list.SetBackgroundColor(palette.surface)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(palette.text).Background(palette.surface))
	list.SetBorderStyle(tcell.StyleDefault.Foreground(palette.button).Background(palette.surface))
	singleLineBorder(list.Box)
	s.menuList = list
	count := 0
	for _, cmd := range entries {
		enabled := s.commandEnabled(cmd.id)
		label := s.tr(cmd.label)
		if key := shortcut(cmd.id); key != "" {
			label += "  " + key
		}
		if !enabled {
			label = "[gray]" + label + "[-]"
		}
		id := cmd.id
		list.AddItem(label, "", 0, func() { s.closeMenu(); s.invoke(id) })
		count++
	}
	width := 80
	if s.terminal != nil {
		width, _ = s.terminal.Size()
	}
	left := max(0, index) * width / len(menuGroups)
	popupWidth := min(48, width)
	left = min(left, max(0, width-popupWidth))
	row := tview.NewFlex()
	if left > 0 {
		row.AddItem(tview.NewBox(), left, 0, false)
	}
	row.AddItem(list, popupWidth, 0, true).AddItem(tview.NewBox(), 0, 1, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(row, count+2, 0, true).AddItem(tview.NewBox(), 0, 1, false)
	s.pages.AddPage("menu", root, true, true)
	s.app.SetFocus(list)
}

func (s *screen) closeMenu() {
	s.pages.RemovePage("menu")
	s.menuList = nil
	if s.menuFocus != nil {
		s.app.SetFocus(s.menuFocus)
	}
}

func (s *screen) menuKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF10, tcell.KeyEscape:
		s.closeMenu()
		return nil
	case tcell.KeyLeft:
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
