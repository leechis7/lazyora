package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
)

// tview may report a rapid click on a different button as a double click.
// Buttons should activate on either kind of click; text selection stays unchanged.
func clickable(button *tview.Button) *tview.Button {
	button.SetStyle(buttonStyle()).SetActivatedStyle(focusStyle())
	button.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	return button
}

type footerKey struct {
	label, action string
	key           tcell.Key
	mod           tcell.ModMask
}

func (s *screen) toolbar() *tview.Flex {
	bar := tview.NewFlex()
	s.footerBar = bar
	for index := range 6 {
		button := clickable(tview.NewButton("").SetSelectedFunc(func() {
			if index >= len(s.footerKeys) {
				return
			}
			key := s.footerKeys[index]
			if page, _ := s.pages.GetFrontPage(); page == "exit" && (key.action == "exit_commit" || key.action == "exit_rollback") {
				s.confirmExit(key.action)
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "main" && commandFor(key.action).id != "" {
				s.invoke(key.action)
				return
			}
			if key.action == "source" {
				s.showSource()
				return
			}
			if key.action == "arguments" {
				s.showArguments()
				return
			}
			if page, root := s.pages.GetFrontPage(); page == "help" {
				s.app.SetFocus(root.(*tview.Flex).GetItem(0))
			} else if page == "object" {
				s.app.SetFocus(s.objectTable)
			} else if page == "arguments" {
				s.app.SetFocus(s.argumentTable)
			}
			s.app.QueueEvent(tcell.NewEventKey(key.key, 0, key.mod))
		}))
		button.SetStyle(tcell.StyleDefault.Foreground(palette.text).Background(palette.footer))
		s.buttons = append(s.buttons, button)
	}
	version := tview.NewTextView().SetText("v" + appinfo.Version).SetTextAlign(tview.AlignRight).SetWrap(false)
	version.SetTextColor(palette.accent).SetBackgroundColor(palette.footer)
	s.footerVersion = version
	return bar
}

func (s *screen) refreshFooter() {
	if s.footer == nil {
		return
	}
	page, _ := s.pages.GetFrontPage()
	keys := []footerKey{
		{"F1", "help", tcell.KeyF1, tcell.ModNone},
		{"F2", "connect", tcell.KeyF2, tcell.ModNone},
		{"F3", "settings", tcell.KeyF3, tcell.ModNone},
		{"Ctrl+Q", "quit", tcell.KeyCtrlQ, tcell.ModNone},
	}
	hint := "footer_sql"
	switch page {
	case "main":
		keys = nil
		for _, id := range []string{"help", "connect", "settings", "commit", "rollback", "quit"} {
			key := commandFor(id).bindings[0]
			keys = append(keys, footerKey{key.label, id, key.key, key.mod})
		}
		if s.table.HasFocus() {
			s.resultsFocused = true
		} else if s.editor.HasFocus() {
			s.resultsFocused = false
		}
		if s.resultsFocused {
			hint = "footer_results"
		}
		if s.browser != nil && s.browser.HasFocus() {
			hint = "footer_browser"
		}
	case "menu":
		hint = "menu_hint"
		keys = []footerKey{{"Esc", "close", tcell.KeyEscape, 0}}
	case "copy":
		hint = "copy_preview_hint"
		keys = []footerKey{{"Esc", "close", tcell.KeyEscape, 0}}
	case "describe":
		hint = "footer_describe"
		keys = []footerKey{{"Esc", "close", tcell.KeyEscape, 0}}
	case "connect":
		hint = "footer_connect"
		keys[1] = footerKey{"Ctrl+Enter", "connect", tcell.KeyEnter, tcell.ModCtrl}
		keys[2] = footerKey{"Ctrl+S", "save", tcell.KeyCtrlS, tcell.ModNone}
		keys[3] = footerKey{"Esc", "cancel", tcell.KeyEscape, tcell.ModNone}
	case "settings":
		hint = "footer_settings"
		keys[1] = footerKey{"Ctrl+S", "save", tcell.KeyCtrlS, tcell.ModNone}
		keys[2] = footerKey{"Esc", "cancel", tcell.KeyEscape, tcell.ModNone}
	case "help":
		hint = "footer_help"
		keys[0].action = "close"
		keys[1] = footerKey{"PgUp", "previous_page", tcell.KeyPgUp, tcell.ModNone}
		keys[2] = footerKey{"PgDn", "next_page", tcell.KeyPgDn, tcell.ModNone}
		keys[3] = footerKey{"Esc", "close", tcell.KeyEscape, tcell.ModNone}
	case "object":
		hint = "footer_object"
		keys[0].action = "help"
		keys[1] = footerKey{"PgUp", "previous_page", tcell.KeyPgUp, tcell.ModNone}
		keys[2] = footerKey{"PgDn", "next_page", tcell.KeyPgDn, tcell.ModNone}
		keys[3] = footerKey{"Esc", "close", tcell.KeyEscape, tcell.ModNone}
		if s.object.HasSource() {
			keys[1] = footerKey{"Alt+S", "source", tcell.KeyRune, tcell.ModAlt}
		}
		if s.object.HasArguments() {
			keys[2] = footerKey{"Alt+A", "arguments", tcell.KeyRune, tcell.ModAlt}
		}
	case "source":
		hint = "footer_source"
		keys = []footerKey{{"Ctrl+F9", "compile", tcell.KeyF9, tcell.ModCtrl}, {"Ctrl+R", "reload", tcell.KeyCtrlR, tcell.ModNone}, {"Esc", "close", tcell.KeyEscape, tcell.ModNone}}
	case "arguments":
		hint = "footer_arguments"
		keys = []footerKey{{"F1", "help", tcell.KeyF1, tcell.ModNone}, {"PgUp", "previous_page", tcell.KeyPgUp, tcell.ModNone}, {"PgDn", "next_page", tcell.KeyPgDn, tcell.ModNone}, {"Esc", "close", tcell.KeyEscape, tcell.ModNone}}
	case "program_confirm":
		hint = "footer_confirm"
		keys = []footerKey{{"Esc", "cancel", tcell.KeyEscape, tcell.ModNone}}
	case "exit":
		hint = "footer_exit"
		keys = []footerKey{{"Esc", "exit_cancel", tcell.KeyEscape, tcell.ModNone}}
		if s.pendingChanges {
			keys = append(keys, footerKey{shortcut("commit"), "exit_commit", tcell.KeyRune, tcell.ModAlt}, footerKey{shortcut("rollback"), "exit_rollback", tcell.KeyRune, tcell.ModAlt})
		}
	}
	text := s.tr(hint)
	viewKeys := shortcut("maximize") + " " + s.tr("maximize_short") + " / " + shortcut("fullscreen") + " " + s.tr("fullscreen_short")
	if hint == "footer_sql" {
		text = fmt.Sprintf(text, bindingLabels("execute"), viewKeys)
	}
	if hint == "footer_results" {
		text = fmt.Sprintf(text, shortcut("copy"), shortcut("copy_row"), viewKeys)
	}
	if page == "main" {
		if s.maximized != "" {
			if s.fullscreen {
				text += " | " + s.tr("fullscreen_short")
			} else {
				text += " | " + s.tr("maximized")
			}
		}

	}
	if s.busy {
		text = s.tr("footer_busy") + " | " + text
	}
	s.footer.SetText(text)
	s.footerKeys = keys
	s.footerBar.Clear()
	for i, key := range keys {
		s.buttons[i].SetLabel(key.label + " " + s.tr(key.action))
		s.footerBar.AddItem(s.buttons[i], 0, 1, false)
	}
	s.footerBar.AddItem(s.footerVersion, len(appinfo.Version)+3, 0, false)
}

func (s *screen) action(action string) {
	switch action {
	case "help":
		s.showHelp()
	case "connect":
		if !s.busy {
			s.connectionForm()
		}
	case "settings":
		s.settingsForm()
	case "execute":
		s.submit(job{action: "execute", text: s.editor.GetText()})
	case "commit", "rollback":
		s.submit(job{action: action})
	case "cancel":
		if s.cancel != nil {
			s.cancel()
		}
	case "quit":
		s.requestQuit()
	}
}
