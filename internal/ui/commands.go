package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

type binding struct {
	label string
	key   tcell.Key
	mod   tcell.ModMask
	char  rune
}
type command struct {
	id, label string
	group     int
	bindings  []binding
}

var commands = []command{
	{"quit", "quit", 0, []binding{{"Ctrl+Q", tcell.KeyCtrlQ, 0, 0}}},
	{"copy", "copy_selection", 1, []binding{{"Ctrl+C", tcell.KeyCtrlC, 0, 0}}},
	{"copy_row", "copy_row", 1, []binding{{"Alt+Y", tcell.KeyRune, tcell.ModAlt, 'y'}}},
	{"copy_preview", "copy_preview", 1, nil},
	{"connect", "connect", 2, []binding{{"F2", tcell.KeyF2, 0, 0}}},
	{"commit", "commit", 2, []binding{{"Alt+C", tcell.KeyRune, tcell.ModAlt, 'c'}}},
	{"rollback", "rollback", 2, []binding{{"Alt+R", tcell.KeyRune, tcell.ModAlt, 'r'}}},
	{"execute", "execute", 3, []binding{{"F9", tcell.KeyF9, 0, 0}, {"Ctrl+Enter", tcell.KeyEnter, tcell.ModCtrl, 0}, {"Alt+Enter", tcell.KeyEnter, tcell.ModAlt, 0}}},
	{"script", "script_unavailable", 3, []binding{{"F5", tcell.KeyF5, 0, 0}}},
	{"cancel", "cancel_execution", 3, []binding{{"F8", tcell.KeyF8, 0, 0}}},
	{"maximize", "maximize", 4, []binding{{"F11", tcell.KeyF11, 0, 0}}},
	{"fullscreen", "fullscreen", 4, []binding{{"Ctrl+F11", tcell.KeyF11, tcell.ModCtrl, 0}}},
	{"browser", "schemas", 4, []binding{{"F7", tcell.KeyF7, 0, 0}}},
	{"focus_sql", "focus_sql", 4, []binding{{"F6", tcell.KeyF6, 0, 0}}},
	{"focus_results", "focus_results", 4, []binding{{"F10", tcell.KeyF10, 0, 0}}},
	{"describe", "describe", 4, []binding{{"F4", tcell.KeyF4, 0, 0}}},
	{"settings", "settings", 5, []binding{{"F3", tcell.KeyF3, 0, 0}}},
	{"help", "help", 6, []binding{{"F1", tcell.KeyF1, 0, 0}}},
}

func commandFor(id string) command {
	for _, cmd := range commands {
		if cmd.id == id {
			return cmd
		}
	}
	return command{}
}
func bindingLabels(id string) string {
	var labels []string
	for _, key := range commandFor(id).bindings {
		labels = append(labels, key.label)
	}
	return strings.Join(labels, " / ")
}

func shortcut(id string) string {
	cmd := commandFor(id)
	if len(cmd.bindings) > 0 {
		return cmd.bindings[0].label
	}
	return ""
}

func (s *screen) commandEnabled(id string) bool {
	page, _ := s.pages.GetFrontPage()
	if page != "main" {
		return false
	}
	switch id {
	case "quit", "help", "settings", "focus_sql", "focus_results":
		return true
	case "maximize", "fullscreen":
		return s.editor.HasFocus() || s.table.HasFocus()
	case "browser":
		return s.browser != nil
	case "connect":
		return !s.busy
	case "execute", "commit", "rollback":
		return s.connected && !s.busy
	case "cancel":
		return s.busy
	case "describe":
		return !s.busy && (s.connected || s.demo)
	case "copy":
		return (s.table.HasFocus() && len(s.resultRows) > 0) || (s.editor.HasFocus() && s.editor.HasSelection())
	case "copy_row":
		return s.table.HasFocus() && len(s.resultRows) > 0
	case "copy_preview":
		return s.clipboardText != ""
	}
	return false
}

func (s *screen) invoke(id string) {
	if !s.commandEnabled(id) {
		if id == "script" {
			s.setStatus("script_unavailable")
		}
		if id == "execute" || id == "commit" || id == "rollback" {
			if !s.connected {
				s.setStatus("connect_first")
			}
		}
		return
	}
	switch id {
	case "maximize":
		s.toggleMaximize()
	case "fullscreen":
		s.toggleFullscreen()
	case "browser":
		s.toggleBrowser()
	case "focus_sql":
		s.focusPanel("sql")
	case "focus_results":
		s.focusPanel("results")
	case "describe":
		s.describeObject()
	case "copy_row":
		s.copyResults("row")
	case "copy":
		if s.table.HasFocus() {
			s.copyResults("selection")
		} else {
			text, _, _ := s.editor.GetSelection()
			s.sendClipboard(text)
		}
	case "copy_preview":
		s.copyPreview()
	default:
		s.action(id)
	}
}

func (s *screen) commandKey(event *tcell.EventKey) bool {
	for _, cmd := range commands {
		for _, key := range cmd.bindings {
			// tcell normalizes C0 control keys to include ModCtrl.
			mod := event.Modifiers()
			if event.Key() >= tcell.KeyCtrlA && event.Key() <= tcell.KeyCtrlZ && event.Key() != tcell.KeyEnter && event.Key() != tcell.KeyTab {
				mod &^= tcell.ModCtrl
			}
			if event.Key() == key.key && mod == key.mod && (key.key != tcell.KeyRune || event.Rune() == key.char) {
				s.invoke(cmd.id)
				return true
			}
		}
	}
	return false
}

func (s *screen) menuBar() *tview.Flex {
	bar := tview.NewFlex()
	for index := range menuGroups {
		button := clickable(tview.NewButton("").SetSelectedFunc(func() { s.showMenu(index) }))
		button.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			// Menu clicks act on the current panel; do not transfer focus to the bar.
			if action == tview.MouseLeftDown {
				return tview.MouseConsumed, nil
			}
			if action == tview.MouseLeftDoubleClick {
				action = tview.MouseLeftClick
			}
			return action, event
		})
		s.menuButtons = append(s.menuButtons, button)
		bar.AddItem(button, 0, 1, false)
	}
	s.refreshMenuLabels()
	return bar
}
