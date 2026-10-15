package ui

import (
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
	for index := range 4 {
		button := clickable(tview.NewButton("").SetSelectedFunc(func() {
			key := s.footerKeys[index]
			if page, root := s.pages.GetFrontPage(); page == "help" {
				s.app.SetFocus(root.(*tview.Flex).GetItem(0))
			}
			s.app.QueueEvent(tcell.NewEventKey(key.key, 0, key.mod))
		}))
		button.SetStyle(tcell.StyleDefault.Foreground(palette.text).Background(palette.footer))
		s.buttons = append(s.buttons, button)
		bar.AddItem(button, 0, 1, false)
	}
	version := tview.NewTextView().SetText("v" + appinfo.Version).SetTextAlign(tview.AlignRight).SetWrap(false)
	version.SetTextColor(palette.accent).SetBackgroundColor(palette.footer)
	bar.AddItem(version, len(appinfo.Version)+3, 0, false)
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
		if s.table.HasFocus() {
			s.resultsFocused = true
		} else if s.editor.HasFocus() {
			s.resultsFocused = false
		}
		if s.resultsFocused {
			hint = "footer_results"
		}
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
	}
	text := s.tr(hint)
	if s.busy {
		text = s.tr("footer_busy") + " | " + text
	}
	s.footer.SetText(text)
	s.footerKeys = keys
	for i, button := range s.buttons {
		button.SetLabel(keys[i].label + " " + s.tr(keys[i].action))
	}
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
		if s.busy {
			s.quitting = true
			s.cancel()
			s.setStatus("cancelling")
		} else {
			s.app.Stop()
		}
	}
}
