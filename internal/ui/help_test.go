package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
	"lazyora/internal/oracle"
)

func TestHelpResponsiveTopicsKeyboardAndResize(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		for _, limited := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/limited=%v", language, limited), func(t *testing.T) {
				s := shortcutScreen(t)
				s.prefs.Language, s.limitedColors = language, limited
				s.app.SetFocus(s.editor)
				s.showHelp()
				_, root := s.pages.GetFrontPage()
				terminal := tcell.NewSimulationScreen("UTF-8")
				if err := terminal.Init(); err != nil {
					t.Fatal(err)
				}
				defer terminal.Fini()
				deliver := func(event *tcell.EventKey) {
					if event = s.key(event); event != nil {
						s.app.GetFocus().InputHandler()(event, func(p tview.Primitive) { s.app.SetFocus(p) })
					}
				}
				for _, size := range [][2]int{{100, 30}, {60, 20}, {40, 12}, {80, 24}} {
					terminal.SetSize(size[0], size[1])
					root.SetRect(0, 0, size[0], size[1]-2)
					root.Draw(terminal)
					x, y, w, h := s.helpText.GetRect()
					_, ty, tw, th := s.helpTopics.GetRect()
					if w < 35 || h < 4 || tw <= 0 || th <= 0 {
						t.Fatalf("unreadable layout at %v: article=%d,%d %dx%d contents=%dx%d", size, x, y, w, h, tw, th)
					}
					if size[0] < 76 && y <= ty || size[0] >= 76 && (y != ty || x < tw) {
						t.Fatalf("contents did not adapt to terminal width %v", size)
					}
				}
				// Shift+Tab from the article reaches the contents; arrows change
				// topics without sending any application command to the worker.
				deliver(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
				if !s.helpTopics.HasFocus() {
					t.Fatal("cannot reach contents with the keyboard")
				}
				deliver(tcell.NewEventKey(tcell.KeyDown, 0, 0))
				if s.helpTopics.GetCurrentItem() != 1 || !strings.Contains(s.helpText.GetText(true), "Ctrl+Enter") {
					t.Fatal("contents navigation did not load SQL commands")
				}
				deliver(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
				if !s.helpText.HasFocus() {
					t.Fatal("Enter did not move to the selected article")
				}
				deliver(tcell.NewEventKey(tcell.KeyRune, '8', 0))
				if !strings.Contains(s.helpText.GetText(true), "Notepad++") || strings.Contains(s.helpText.GetText(true), appinfo.Email) {
					t.Fatal("keymap topic was missing or mixed with app information")
				}
				s.refreshBorders()
				root.Draw(terminal)
				for py := 0; py < 22; py++ {
					for px := 0; px < 80; px++ {
						_, _, style, _ := terminal.GetContent(px, py)
						_, _, attributes := style.Decompose()
						if attributes&tcell.AttrUnderline != 0 {
							t.Fatal("help reintroduced underlined article or topic decoration")
						}
					}
				}
				deliver(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
				if s.helpText != nil || s.helpTopics != nil || !s.editor.HasFocus() || len(s.jobs) != 0 {
					t.Fatal("help did not release its state and preserve the editor")
				}
			})
		}
	}
}

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
			help := s.helpText
			draw := func() { layout.SetRect(0, 0, 80, 22); layout.Draw(terminal) }
			draw()
			text := help.GetText(true)
			if !strings.Contains(text, s.tr("help_current")) || strings.Contains(text, appinfo.Email) || strings.Contains(text, s.tr("vim_commands")) {
				t.Fatal("help does not isolate current-screen guidance from other topics")
			}
			// Result focus should produce copying controls instead of SQL execution controls.
			top := text
			if !strings.Contains(top, "Ctrl+C") || strings.Contains(top, "Ctrl+Enter") {
				t.Fatal("quick controls do not match result focus")
			}
			s.key(tcell.NewEventKey(tcell.KeyRune, '9', 0))
			draw()
			row, _ := help.GetScrollOffset()
			if row != 0 || !strings.Contains(help.GetText(true), appinfo.Email) || s.helpTopics.GetCurrentItem() != 9 {
				t.Fatal("9 did not show a separate app information topic from its beginning")
			}
			s.key(tcell.NewEventKey(tcell.KeyRune, '0', 0))
			draw()
			row, _ = help.GetScrollOffset()
			if row != 0 || strings.Contains(help.GetText(true), appinfo.Email) {
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
	text := s.helpText.GetText(true)
	if strings.Contains(text, "private-password") || !strings.Contains(text, "Alt+C") {
		t.Fatal("help exposed credentials or omitted connection controls")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, '4', 0))
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	if s.app.GetFocus() != previous || s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "private-password" {
		t.Fatal("help changed the connection form")
	}
}
