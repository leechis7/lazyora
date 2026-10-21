package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestCompletionCancelAndInsertPreserveUndoAndFocus(t *testing.T) {
	for _, lang := range []string{"ko", "en"} {
		s := demoBrowserScreen(t)
		s.connected = false
		s.prefs.Language = lang
		s.editor.SetText("se", true)
		s.app.SetFocus(s.editor)
		s.key(tcell.NewEventKey(tcell.KeyCtrlT, 0, 0))
		if s.completionList == nil || len(s.jobs) != 0 {
			t.Fatal("offline completion used DB")
		}
		s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
		if s.editor.GetText() != "se" || !s.editor.HasFocus() {
			t.Fatal("cancel changed text/focus")
		}
		s.key(tcell.NewEventKey(tcell.KeyCtrlT, 0, 0))
		for i, candidate := range s.completionRows {
			if candidate.Name == "SELECT" {
				s.completionList.SetCurrentItem(i)
			}
		}
		s.applyCompletion()
		if s.editor.GetText() != "select" || !s.editor.HasFocus() {
			t.Fatalf("insert=%q", s.editor.GetText())
		}
		s.editor.InputHandler()(tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if s.editor.GetText() != "se" {
			t.Fatal("completion reset undo history")
		}
	}
}
func TestCompletionQuotesMetadataAndRefreshesWithFilter(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = true
	original := "SELECT s.na FROM sample_rows s"
	s.editor.SetText(original, false)
	cursor := len("SELECT s.na")
	s.editor.Select(cursor, cursor)
	s.app.SetFocus(s.editor)
	s.showCompletion(false)
	j := <-s.jobs
	s.cancel()
	s.cancel = nil
	s.busy = false
	if j.action != "complete" || j.completion.Prefix != "na" {
		t.Fatalf("%+v", j)
	}
	s.acceptCompletions(oracle.Completions{Candidates: []oracle.CompletionCandidate{{Text: `"Name"`, Name: "Name", Kind: "COLUMN", Detail: "APP.SAMPLE_ROWS"}}}, nil)
	s.applyCompletion()
	if got := s.editor.GetText(); got != `SELECT s."Name" FROM sample_rows s` {
		t.Fatal(got)
	}
	s.showCompletion(false)
	<-s.jobs
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.completionFilter.SetText("Na")
	s.key(tcell.NewEventKey(tcell.KeyCtrlR, 0, 0))
	j = <-s.jobs
	s.cancel()
	if !j.refresh || j.completion.Prefix != "Na" {
		t.Fatalf("refresh: %+v", j)
	}
}
func TestCompletionRejectsStaleCursorAndSkipsComments(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = false
	s.app.SetFocus(s.editor)
	s.editor.SetText("sele", true)
	s.showCompletion(false)
	s.editor.Select(2, 2)
	s.applyCompletion()
	if s.editor.GetText() != "sele" || s.completionList == nil || !strings.Contains(s.completionInfo.GetText(false), s.tr("completion_stale")) {
		t.Fatal("stale cursor replaced text")
	}
	s.closeCompletion()
	s.editor.SetText("-- SELECT something", true)
	s.showCompletion(false)
	if s.completionList != nil {
		t.Fatal("completed inside comment")
	}
}
func TestSourceCompletionPreservesDirtyStateAndEscapeCancelsMetadataOnly(t *testing.T) {
	s := programScreen(t)
	s.connected = false
	text := s.sourceEditor.GetText() + "\nSEL"
	s.sourceEditor.SetText(text, true)
	s.key(tcell.NewEventKey(tcell.KeyCtrlT, 0, 0))
	if s.completionEditor != s.sourceEditor {
		t.Fatal("wrong editor")
	}
	s.applyCompletion()
	if !strings.HasSuffix(s.sourceEditor.GetText(), "SELECT") || !s.sourceDirty() || !s.sourceEditor.HasFocus() {
		t.Fatal("source completion lost edit/focus")
	}
	s.connected = true
	s.showCompletion(false)
	<-s.jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.cancel()
	s.cancel = cancel
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if ctx.Err() == nil || s.completionEditor == nil || !s.busy {
		t.Fatal("busy Escape didn't cancel read safely")
	}
	s.busy = false
	s.closeCompletion()
	if !s.sourceDirty() {
		t.Fatal("metadata cancellation discarded source")
	}
}

func TestCompletionHelpKeepsMetadataRunningAndCancelButtonCancels(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = true
	s.editor.SetText("SEL", true)
	s.app.SetFocus(s.editor)
	s.showCompletion(false)
	j := <-s.jobs
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if j.ctx.Err() != nil || !s.busy || s.completionEditor == nil {
		t.Fatal("closing help cancelled completion read")
	}
	s.cancelCompletion()
	if j.ctx.Err() == nil || s.completionEditor == nil {
		t.Fatal("cancel button failed to cancel only metadata read")
	}
	s.busy = false
	s.closeCompletion()
	if s.editor.GetText() != "SEL" {
		t.Fatal("metadata cancellation edited SQL")
	}
}
func TestSourceDiagnosticFocusDoesNotCompleteSQL(t *testing.T) {
	s := programScreen(t)
	s.app.SetFocus(s.sourceDiagnostics)
	s.key(tcell.NewEventKey(tcell.KeyCtrlT, 0, 0))
	if s.completionEditor != nil {
		t.Fatal("non-editor focus opened completion")
	}
}
