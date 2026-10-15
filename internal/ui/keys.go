package ui

import (
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func focusForm(s *screen, form *tview.Form, index int) {
	form.SetFocus(index)
	s.app.SetFocus(form)
}

func (s *screen) key(event *tcell.EventKey) *tcell.EventKey {
	page, _ := s.pages.GetFrontPage()
	ctrlEnter := event.Key() == tcell.KeyEnter && event.Modifiers()&tcell.ModCtrl != 0
	alt := event.Modifiers()&tcell.ModAlt != 0
	letter := unicode.ToLower(event.Rune())
	if event.Key() == tcell.KeyCtrlQ || event.Key() == tcell.KeyCtrlC {
		s.action("quit")
		return nil
	}
	if event.Key() == tcell.KeyF8 {
		s.action("cancel")
		return nil
	}
	if event.Key() == tcell.KeyF1 {
		if page == "help" {
			s.closeHelp()
		} else {
			s.showHelp()
		}
		return nil
	}
	if page == "help" {
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
			s.closeHelp()
			return nil
		}
		return event
	}
	if event.Key() == tcell.KeyF3 && page != "settings" {
		s.settingsForm()
		return nil
	}
	if page == "settings" {
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
			s.closeSettings()
			return nil
		}
		if event.Key() == tcell.KeyCtrlS || (alt && letter == 's') {
			s.savePrefs()
			return nil
		}
		if alt {
			switch letter {
			case 'l':
				focusForm(s, s.prefsForm, 0)
				return nil
			case 'm':
				focusForm(s, s.prefsForm, 1)
				return nil
			}
		}
		return event
	}
	if page == "connect" {
		if event.Key() == tcell.KeyCtrlS || (alt && letter == 's') {
			s.saveConnection()
			return nil
		}
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
			s.closeConnection()
			return nil
		}
		if ctrlEnter || (alt && letter == 'c') {
			s.connectAction()
			return nil
		}
		if alt {
			for i, key := range []rune{'h', 'p', 'u', 'w', 't', 'n', 'r', 'a'} {
				if letter == key {
					focusForm(s, s.connectForm, i)
					return nil
				}
			}
		}
		return event
	}
	if page != "main" {
		return event
	}
	if ctrlEnter {
		s.action("execute")
		return nil
	}
	switch event.Key() {
	case tcell.KeyF2:
		s.action("connect")
		return nil
	case tcell.KeyF4:
		s.action("execute")
		return nil
	case tcell.KeyF5:
		s.action("commit")
		return nil
	case tcell.KeyF6:
		s.action("rollback")
		return nil
	case tcell.KeyF9:
		s.app.SetFocus(s.editor)
		return nil
	case tcell.KeyF10:
		s.app.SetFocus(s.table)
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		if s.app.GetFocus() == s.editor {
			s.app.SetFocus(s.table)
		} else {
			s.app.SetFocus(s.editor)
		}
		return nil
	}
	return event
}
