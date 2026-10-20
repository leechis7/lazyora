package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"lazyora/internal/oracle"
)

func TestOutputViewerCopiesRawTextWithoutChangingResultsOrTransaction(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		s, terminal := resultScreen(t)
		s.prefs.Language = lang
		s.pendingChanges = true
		before := s.resultRows[0][1]
		s.addOutput(oracle.Output{Lines: []string{"한글 [red]", "", "tail"}})
		s.key(tcell.NewEventKey(tcell.KeyRune, 'p', tcell.ModAlt))
		if s.outputText == nil || s.outputText.GetText(false) != "한글 [red]\n\ntail\n" {
			t.Fatal("viewer changed raw output")
		}
		_, root := s.pages.GetFrontPage()
		root.SetRect(0, 0, 120, 30)
		root.Draw(terminal)
		var rendered strings.Builder
		for row := 0; row < 30; row++ {
			for col := 0; col < 120; col++ {
				char, _, _, width := terminal.GetContent(col, row)
				rendered.WriteRune(char)
				if width > 1 {
					col += width - 1
				}
			}
		}
		if !strings.Contains(rendered.String(), "한글 [red]") {
			t.Fatal("drawing interpreted output as color tags")
		}
		s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
		if s.clipboardText != "한글 [red]\n\ntail\n" {
			t.Fatal("copy changed raw output")
		}
		s.key(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModAlt))
		if s.outputText.GetText(false) != "" || s.outputSize != 0 {
			t.Fatal("clear left retained output")
		}
		s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
		if !s.table.HasFocus() || !s.pendingChanges || s.resultRows[0][1] != before || len(s.jobs) != 0 {
			t.Fatal("viewer changed query/transaction/focus")
		}
	}
}

func TestClosingOutputDuringBusySQLDoesNotCancelIt(t *testing.T) {
	s := demoBrowserScreen(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.busy, s.cancel = true, cancel
	s.key(tcell.NewEventKey(tcell.KeyRune, 'p', tcell.ModAlt))
	s.refreshFooter()
	if strings.Contains(s.footer.GetText(false), s.tr("footer_busy")) || !s.outputToggle.IsDisabled() {
		t.Fatal("output viewer advertised SQL cancellation or allowed busy toggle")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if ctx.Err() != nil || !s.busy || !s.editor.HasFocus() {
		t.Fatal("closing output cancelled SQL or lost focus")
	}
}

func TestOutputHistoryKeepsRecentLinesWithinBudget(t *testing.T) {
	s := shortcutScreen(t)
	line := strings.Repeat("한", 1000)
	for i := 0; i < 20; i++ {
		s.addOutput(oracle.Output{Lines: []string{strings.Repeat(line, 20)}})
	}
	s.addOutput(oracle.Output{Lines: []string{"latest"}, Error: "collection warning"})
	if s.outputSize > outputHistoryBytes || len(s.outputLines) > outputHistoryLines || !s.outputTruncated || s.outputLines[len(s.outputLines)-1] != "latest" || s.outputError != "collection warning" {
		t.Fatal("history did not retain bounded recent output")
	}
	s.clearOutput()
	if s.outputTruncated || s.outputError != "" || s.outputSize != 0 {
		t.Fatal("clear left history warnings")
	}
}

func TestOutputToggleUsesWorkerAndIsDisabledWithoutConnection(t *testing.T) {
	s := shortcutScreen(t)
	s.showOutput()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModAlt))
	if len(s.jobs) != 1 || !s.busy || !s.outputToggle.IsDisabled() {
		t.Fatal("toggle did not lock and submit to worker")
	}
	if j := <-s.jobs; j.action != "output_toggle" {
		t.Fatal("toggle submitted another action")
	}
	s.closeOutput()
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.connected = false
	s.showOutput()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModAlt))
	if len(s.jobs) != 0 || !s.outputToggle.IsDisabled() {
		t.Fatal("disconnected toggle submitted DB work")
	}
}
