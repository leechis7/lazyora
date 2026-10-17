package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
)

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
