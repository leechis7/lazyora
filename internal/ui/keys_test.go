package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/settings"
)

func shortcutScreen(t *testing.T) *screen {
	t.Helper()
	s := &screen{
		app: tview.NewApplication(), pages: tview.NewPages(),
		editor: tview.NewTextArea().SetText("SELECT 1 FROM dual;", false),
		table:  tview.NewTable(), status: tview.NewTextView(), jobs: make(chan job, 1),
		connected: true,
		prefs:     settings.Config{Language: "en", Mouse: true},
		prefsPath: filepath.Join(t.TempDir(), "settings.json"),
	}
	s.pages.AddPage("main", tview.NewFlex().AddItem(s.editor, 0, 1, true), true, true)
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
		key  tcell.Key
		mod  tcell.ModMask
	}{
		{"CtrlEnter", tcell.KeyEnter, tcell.ModCtrl},
		{"F4", tcell.KeyF4, tcell.ModNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := shortcutScreen(t)
			if s.key(tcell.NewEventKey(tc.key, 0, tc.mod)) != nil {
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
	event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	forwarded := s.key(event)
	if forwarded == nil {
		t.Fatal("plain Enter was intercepted")
	}
	s.editor.InputHandler()(forwarded, func(tview.Primitive) {})
	if !strings.Contains(s.editor.GetText(), "\n") {
		t.Fatal("plain Enter did not insert a newline")
	}
	if len(s.jobs) != 0 || s.busy {
		t.Fatal("plain Enter executed SQL")
	}
}

func TestHelpRestoresPreviousPageAndFocus(t *testing.T) {
	s := shortcutScreen(t)
	for _, closeKey := range []tcell.Key{tcell.KeyF1, tcell.KeyEscape} {
		s.app.SetFocus(s.table)
		s.key(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone))
		if page, _ := s.pages.GetFrontPage(); page != "help" {
			t.Fatal("F1 did not open help")
		}
		s.key(tcell.NewEventKey(tcell.KeyF4, 0, tcell.ModNone))
		s.key(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl))
		if len(s.jobs) != 0 {
			t.Fatal("help allowed SQL execution behind the overlay")
		}
		s.key(tcell.NewEventKey(closeKey, 0, tcell.ModNone))
		if page, _ := s.pages.GetFrontPage(); page != "main" {
			t.Fatal("help did not restore the main page")
		}
		if s.app.GetFocus() != s.table {
			t.Fatal("help did not restore focus")
		}
	}
	s.connectionForm()
	focus := s.app.GetFocus()
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if page, _ := s.pages.GetFrontPage(); page != "connect" {
		t.Fatal("help discarded the connection form")
	}
	if s.app.GetFocus() != focus {
		t.Fatal("help did not restore the connection input")
	}
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl))
	if len(s.jobs) != 0 {
		t.Fatal("Ctrl+Enter in the form executed SQL")
	}
}
