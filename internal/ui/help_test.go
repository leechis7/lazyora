package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
	"lazyora/internal/oracle"
)

func TestClosingHelpPreservesRunningOperation(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		for _, key := range []struct {
			name  string
			event *tcell.EventKey
		}{
			{"Escape", tcell.NewEventKey(tcell.KeyEscape, 0, 0)},
			{"F1", tcell.NewEventKey(tcell.KeyF1, 0, 0)},
			{"AltX", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt)},
		} {
			t.Run(language+"/"+key.name, func(t *testing.T) {
				s := shortcutScreen(t)
				s.prefs.Language = language
				s.footer = tview.NewTextView()
				s.toolbar()
				s.app.SetFocus(s.table)
				ctx, cancel := context.WithCancel(context.Background())
				s.cancel, s.busy, s.pendingChanges = cancel, true, true
				s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
				if page, _ := s.pages.GetFrontPage(); page != "help" {
					t.Fatal("help did not open during running SQL")
				}
				s.refreshFooter()
				if strings.Contains(s.footer.GetText(false), s.tr("footer_busy")) {
					t.Fatal("help footer advertises Escape as operation cancellation")
				}
				s.key(key.event)
				if ctx.Err() != nil {
					t.Fatal("closing help cancelled the running operation")
				}
				if page, _ := s.pages.GetFrontPage(); page != "main" || !s.table.HasFocus() {
					t.Fatal("help did not close and restore focus")
				}
				if !s.busy || !s.pendingChanges || len(s.jobs) != 0 || s.editor.GetText() != "SELECT 1 FROM dual;" {
					t.Fatal("closing help changed SQL, transaction or worker state")
				}
				// Esc on the main screen must still explicitly cancel the operation.
				s.refreshFooter()
				if !strings.Contains(s.footer.GetText(false), s.tr("footer_busy")) {
					t.Fatal("main footer lost its operation cancellation hint")
				}
				s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
				if ctx.Err() != context.Canceled || !s.busy {
					t.Fatal("main Escape did not request worker cancellation")
				}
			})
		}
	}
}

func TestHelpEscapeRestoresBusyExportBeforeExplicitCancellation(t *testing.T) {
	s := shortcutScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}})
	s.showExport()
	if s.exportForm == nil {
		t.Fatal("export form did not open")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.busy = cancel, true
	previous := s.app.GetFocus()
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "export" || s.app.GetFocus() != previous || ctx.Err() != nil {
		t.Fatal("help Escape cancelled export or lost its form/focus")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if ctx.Err() != context.Canceled || !s.busy || s.exportForm == nil {
		t.Fatal("export Escape did not request cancellation while keeping progress")
	}
}

func TestHelpJumpsToSectionsAndBackInBothLanguages(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		t.Run(language, func(t *testing.T) {
			s := shortcutScreen(t)
			s.prefs.Language = language
			s.app.SetFocus(s.table)
			terminal := tcell.NewSimulationScreen("UTF-8")
			if err := terminal.Init(); err != nil {
				t.Fatal(err)
			}
			defer terminal.Fini()
			terminal.SetSize(80, 24)
			s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
			_, root := s.pages.GetFrontPage()
			layout := root.(*tview.Flex)
			help := layout.GetItem(0).(*tview.TextView)
			draw := func() { layout.SetRect(0, 0, 80, 22); layout.Draw(terminal) }
			draw()
			text := help.GetText(true)
			if strings.Index(text, s.tr("help_current")) > strings.Index(text, "1  SQL") || strings.Index(text, appinfo.Email) < strings.Index(text, s.tr("help_about")) {
				t.Fatal("help does not lead with context and keep app details last")
			}
			// Result focus should produce copying controls instead of SQL execution controls.
			top := strings.Split(text, s.tr("help_contents"))[0]
			if !strings.Contains(top, "Ctrl+C") || strings.Contains(top, "Ctrl+Enter") {
				t.Fatal("quick controls do not match result focus")
			}
			s.key(tcell.NewEventKey(tcell.KeyRune, '8', 0))
			draw()
			row, _ := help.GetScrollOffset()
			if row == 0 || len(help.GetHighlights()) != 1 || help.GetHighlights()[0] != "about" {
				t.Fatal("8 did not scroll to app information")
			}
			s.key(tcell.NewEventKey(tcell.KeyRune, '0', 0))
			draw()
			row, _ = help.GetScrollOffset()
			if row > 5 {
				t.Fatal("0 did not return to quick controls")
			}
			s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
			if !s.table.HasFocus() || len(s.jobs) != 0 {
				t.Fatal("help navigation changed panel focus or executed SQL")
			}
		})
	}
}

func TestHelpConnectionContextKeepsCredentialsOutOfText(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("private-password")
	previous := s.app.GetFocus()
	s.showHelp()
	_, root := s.pages.GetFrontPage()
	text := root.(*tview.Flex).GetItem(0).(*tview.TextView).GetText(true)
	if strings.Contains(text, "private-password") || !strings.Contains(strings.Split(text, s.tr("help_contents"))[0], "Alt+C") {
		t.Fatal("help exposed credentials or omitted connection controls")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, '4', 0))
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	if s.app.GetFocus() != previous || s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "private-password" {
		t.Fatal("help changed the connection form")
	}
}
