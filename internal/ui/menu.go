package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
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
	s.menuFocus = s.app.GetFocus()
	s.menuIndex = index
	list := tview.NewList().ShowSecondaryText(false).SetSelectedStyle(focusStyle())
	list.SetBorder(true).SetTitle(s.tr(menuGroups[index].label))
	styleBox(list.Box)
	s.menuList = list
	count := 0
	for _, cmd := range commands {
		if cmd.group != index {
			continue
		}
		enabled := s.commandEnabled(cmd.id)
		label := s.tr(cmd.label)
		if key := shortcut(cmd.id); key != "" {
			label += "  " + key
		}
		if !enabled {
			label = "[gray]" + label + "[-]"
		}
		id := cmd.id
		list.AddItem(label, "", 0, func() {
			// Recheck after closing, with the original panel focus restored.
			s.closeMenu()
			s.invoke(id)
		})
		count++
	}
	width := 80
	if s.terminal != nil {
		width, _ = s.terminal.Size()
	}
	left := index * width / len(menuGroups)
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
	case tcell.KeyEscape:
		s.closeMenu()
		return nil
	case tcell.KeyLeft:
		s.showMenu((s.menuIndex + len(menuGroups) - 1) % len(menuGroups))
		return nil
	case tcell.KeyRight:
		s.showMenu((s.menuIndex + 1) % len(menuGroups))
		return nil
	}
	return event
}

func (s *screen) mainHelp() string {
	var out strings.Builder
	for _, cmd := range commands {
		var keys []string
		for _, key := range cmd.bindings {
			keys = append(keys, key.label)
		}
		if len(keys) > 0 {
			fmt.Fprintf(&out, "%-30s %s\n", strings.Join(keys, " / "), s.tr(cmd.label))
		}
	}
	out.WriteString("\n" + s.tr("menu_hint") + "\n")
	return out.String()
}
