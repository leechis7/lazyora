package ui

import (
	"lazyora/internal/settings"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func shortcutScreen(t *testing.T) *screen {
	t.Helper()
	s := &screen{
		app: tuivory.NewApplication(), pages: tuivory.NewPages(),
		editor: tuivory.NewTextArea().SetText("SELECT 1 FROM dual;", false),
		table:  tuivory.NewTable(), status: tuivory.NewWidgetTextView(), jobs: make(chan job, 1),
		connected: true,
		prefs:     settings.Config{Language: "en", Mouse: true},
		prefsPath: filepath.Join(t.TempDir(), "settings.json"),
	}
	s.pages.AddPage("main", tuivory.NewFlex().AddItem(s.editor, 0, 1, true), true, true)
	s.app.SetRoot(s.pages, true).SetFocus(s.editor)
	t.Cleanup(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
	return s
}

func TestSQLExecutionShortcuts(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  tuivory.Key
		mod  tuivory.ModMask
	}{
		{"CtrlEnter", tuivory.KeyEnter, tuivory.ModCtrl},
		{"AltEnter", tuivory.KeyEnter, tuivory.ModAlt},
		{"F9", tuivory.KeyF9, tuivory.ModNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := shortcutScreen(t)
			if s.key(tuivory.NewEventKey(tc.key, 0, tc.mod)) != nil {
				t.Fatal("execution shortcut reached the editor")
			}
			select {
			case j := <-s.jobs:
				if j.action != "execute" || j.text != "SELECT 1 FROM dual;" {
					t.Fatalf("unexpected job: %+v", j)
				}
			default:
				t.Fatal("no SQL execution was submitted")
			}
			if s.editor.GetText() != "SELECT 1 FROM dual;" {
				t.Fatal("shortcut changed SQL text")
			}
		})
	}
}

func TestPlainEnterStillInsertsNewline(t *testing.T) {
	s := shortcutScreen(t)
	event := tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModNone)
	forwarded := s.key(event)
	if forwarded == nil {
		t.Fatal("plain Enter was intercepted")
	}
	s.editor.InputHandler()(forwarded, func(tuivory.Primitive) {})
	if !strings.Contains(s.editor.GetText(), "\n") {
		t.Fatal("plain Enter did not insert a newline")
	}
	if len(s.jobs) != 0 || s.busy {
		t.Fatal("plain Enter executed SQL")
	}
}

func TestHelpRestoresPreviousPageAndFocus(t *testing.T) {
	s := shortcutScreen(t)
	for _, closeKey := range []tuivory.Key{tuivory.KeyF1, tuivory.KeyEscape} {
		s.app.SetFocus(s.table)
		s.key(tuivory.NewEventKey(tuivory.KeyF1, 0, tuivory.ModNone))
		if page, _ := s.pages.GetFrontPage(); page != "help" {
			t.Fatal("F1 did not open help")
		}
		s.key(tuivory.NewEventKey(tuivory.KeyF4, 0, tuivory.ModNone))
		s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModCtrl))
		if len(s.jobs) != 0 {
			t.Fatal("help allowed SQL execution behind the overlay")
		}
		s.key(tuivory.NewEventKey(closeKey, 0, tuivory.ModNone))
		if page, _ := s.pages.GetFrontPage(); page != "main" {
			t.Fatal("help did not restore the main page")
		}
		if s.app.GetFocus() != s.table {
			t.Fatal("help did not restore focus")
		}
	}
	s.connectionForm()
	focus := s.app.GetFocus()
	s.key(tuivory.NewEventKey(tuivory.KeyF1, 0, tuivory.ModNone))
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, tuivory.ModNone))
	if page, _ := s.pages.GetFrontPage(); page != "connect" {
		t.Fatal("help discarded the connection form")
	}
	if s.app.GetFocus() != focus {
		t.Fatal("help did not restore the connection input")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModCtrl))
	if len(s.jobs) != 0 {
		t.Fatal("Ctrl+Enter in the form executed SQL")
	}
}

func TestLegacyAltEnterExecutesWithoutOpeningMenu(t *testing.T) {
	s := shortcutScreen(t)
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'm', tuivory.ModAlt|tuivory.ModCtrl))
	if j := <-s.jobs; j.action != "execute" {
		t.Fatal("legacy Alt+Enter did not execute")
	}
	if s.menuList != nil {
		t.Fatal("legacy Alt+Enter opened Alt+M menu")
	}
}
