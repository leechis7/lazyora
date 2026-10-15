package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// tview may report a rapid click on a different button as a double click.
// Buttons should activate on either kind of click; text selection stays unchanged.
func clickable(button *tview.Button) *tview.Button {
	button.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	return button
}

var toolbarActions = []struct{ key, action string }{
	{"F1", "help"}, {"F2", "connect"}, {"F3", "settings"}, {"F4", "execute"},
	{"F5", "commit"}, {"F6", "rollback"}, {"F8", "cancel"}, {"Ctrl+Q", "quit"},
}

func (s *screen) toolbar() *tview.Flex {
	bar := tview.NewFlex().SetDirection(tview.FlexRow)
	for row := 0; row < 2; row++ {
		line := tview.NewFlex()
		for col := 0; col < 4; col++ {
			action := toolbarActions[row*4+col].action
			button := clickable(tview.NewButton("").SetSelectedFunc(func() { s.action(action) }))
			s.buttons = append(s.buttons, button)
			line.AddItem(button, 0, 1, false)
		}
		bar.AddItem(line, 1, 0, false)
	}
	s.hint = tview.NewTextView()
	bar.AddItem(s.hint, 1, 0, false)
	return bar
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
