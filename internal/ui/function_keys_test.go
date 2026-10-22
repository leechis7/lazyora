package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestLegacyShiftFunctionKeysMatchExplicitModifiers(t *testing.T) {
	for _, keymap := range []string{"", "vim", "notepad++"} {
		for number := 1; number <= 12; number++ {
			t.Run(fmt.Sprintf("%s/Shift+F%d", keymap, number), func(t *testing.T) {
				snapshot := func(event *tcell.EventKey) string {
					s, _ := resultScreen(t)
					s.prefs.EditorKeymap = keymap
					s.demo, s.connected = false, true
					text := "SELECT 1 FROM dual;\nSELECT 2 FROM dual;"
					s.editor.SetText(text, false).Select(len(text), len(text))
					s.findText = "dual"
					s.app.SetFocus(s.editor)
					forwarded := s.key(event)
					page, _ := s.pages.GetFrontPage()
					selected, start, end := s.editor.GetSelection()
					var jobs []string
					for len(s.jobs) > 0 {
						job := <-s.jobs
						jobs = append(jobs, job.action+":"+job.text)
					}
					forwardedName := "consumed"
					if forwarded != nil {
						forwardedName = forwarded.Name()
					}
					return fmt.Sprintf("%s|%s|%v|%s|%v|%v|%d|%q|%d:%d:%s|%s|%s", forwardedName, page,
						s.fullscreen, s.maximized, s.editor.HasFocus(), s.table.HasFocus(), s.menuIndex,
						s.editor.GetText(), start, end, selected, strings.Join(jobs, ","), s.status.GetText(false))
				}
				want := snapshot(tcell.NewEventKey(tcell.KeyF1+tcell.Key(number-1), 0, tcell.ModShift))
				got := snapshot(tcell.NewEventKey(tcell.KeyF13+tcell.Key(number-1), 0, 0))
				if got != want {
					t.Fatalf("legacy and modified function keys differ\nlegacy: %s\nmodern: %s", got, want)
				}
			})
		}
	}
}

func TestModifiedHigherFunctionKeysAreNotReinterpreted(t *testing.T) {
	s, _ := resultScreen(t)
	for key := tcell.KeyF13; key <= tcell.KeyF24; key++ {
		event := tcell.NewEventKey(key, 0, tcell.ModCtrl)
		forwarded := s.key(event)
		if forwarded == nil || forwarded.Key() != key || forwarded.Modifiers() != tcell.ModCtrl {
			t.Fatalf("modified higher key %v was reinterpreted", key)
		}
	}
}
