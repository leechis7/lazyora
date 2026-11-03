package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestToadPanelAndReservedKeys(t *testing.T) {
	s := demoBrowserScreen(t)
	s.key(tuivory.NewEventKey(tuivory.KeyF2, 0, 0))
	if !s.fullscreen || s.maximized != "sql" || !s.editor.HasFocus() || s.connectForm != nil {
		t.Fatal("F2 did not fullscreen SQL")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF2, 0, tuivory.ModShift))
	if !s.fullscreen || s.maximized != "results" || !s.table.HasFocus() {
		t.Fatal("Shift+F2 did not switch fullscreen to results")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF2, 0, tuivory.ModShift))
	if s.fullscreen || s.maximized != "" {
		t.Fatal("Shift+F2 did not restore split")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF6, 0, 0))
	if !s.editor.HasFocus() {
		t.Fatal("F6 did not switch to SQL")
	}
	for _, key := range []*tuivory.EventKey{tuivory.NewEventKey(tuivory.KeyF5, 0, 0), tuivory.NewEventKey(tuivory.KeyF11, 0, 0), tuivory.NewEventKey(tuivory.KeyF9, 0, tuivory.ModCtrl)} {
		s.key(key)
		if len(s.jobs) != 0 || s.maximized != "" || s.fullscreen {
			t.Fatal("reserved key submitted DB work or changed layout")
		}
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF10, 0, tuivory.ModShift))
	if s.menuList == nil || s.menuIndex != -1 || s.menuFocus != s.editor {
		t.Fatal("Shift+F10 did not open SQL context menu")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if !s.editor.HasFocus() {
		t.Fatal("context menu lost focus")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlD, 0, tuivory.ModCtrl))
	if s.describeForm == nil || len(s.jobs) != 0 {
		t.Fatal("Ctrl+D did not prompt for an object name")
	}
}

func TestClearSQLConfirmsAndSupportsUndoWithoutChangingResults(t *testing.T) {
	s := shortcutScreen(t)
	s.pendingChanges = true
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
	original := s.editor.GetText()
	s.key(tuivory.NewEventKey(tuivory.KeyF7, 0, 0))
	if s.programModal == nil {
		t.Fatal("F7 did not ask before clearing SQL")
	}
	modalKey(s, tuivory.KeyEnter)
	if s.editor.GetText() != original {
		t.Fatal("default confirmation cleared SQL")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF7, 0, 0))
	modalKey(s, tuivory.KeyLeft)
	modalKey(s, tuivory.KeyEnter)
	if s.editor.GetText() != "" || s.resultRows[0][0] != "keep" || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("clear affected SQL/results/transaction incorrectly")
	}
	s.editor.InputHandler()(tuivory.NewEventKey(tuivory.KeyCtrlZ, 0, 0), func(tuivory.Primitive) {})
	if s.editor.GetText() != original {
		t.Fatal("cleared SQL could not be undone")
	}
}

func TestFindReplaceUnicodeWrapAndDialogIsolation(t *testing.T) {
	s := shortcutScreen(t)
	s.editor.SetText("한글 x 한글 X", false).Select(0, 0)
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlF, 0, tuivory.ModCtrl))
	if s.searchForm == nil {
		t.Fatal("Ctrl+F did not open Find")
	}
	s.searchForm.GetFormItem(0).(*tuivory.InputField).SetText("한글")
	s.key(tuivory.NewEventKey(tuivory.KeyF9, 0, 0))
	if len(s.jobs) != 0 {
		t.Fatal("search dialog executed SQL behind it")
	}
	s.searchAction("next")
	assertMatch := func(start int) {
		t.Helper()
		selected, from, _ := s.editor.GetSelection()
		if selected != "한글" || from != start {
			t.Fatalf("wrong Unicode match: %q at %d", selected, from)
		}
	}
	assertMatch(0)
	s.key(tuivory.NewEventKey(tuivory.KeyF3, 0, 0))
	assertMatch(9)
	s.key(tuivory.NewEventKey(tuivory.KeyF3, 0, 0))
	assertMatch(0)
	s.key(tuivory.NewEventKey(tuivory.KeyF3, 0, tuivory.ModShift))
	assertMatch(9)
	if s.prefsForm != nil {
		t.Fatal("F3 opened settings")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlR, 0, tuivory.ModCtrl))
	s.searchForm.GetFormItem(1).(*tuivory.InputField).SetText("가")
	s.searchAction("replace")
	if s.editor.GetText() != "한글 x 가 X" {
		t.Fatal("single replace changed wrong text")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlR, 0, tuivory.ModCtrl))
	s.searchForm.GetFormItem(0).(*tuivory.InputField).SetText("x")
	s.searchForm.GetFormItem(1).(*tuivory.InputField).SetText("끝")
	s.searchAction("replace_all")
	if s.editor.GetText() != "한글 끝 가 X" || len(s.jobs) != 0 {
		t.Fatal("literal case-sensitive replace-all failed")
	}
	s.findText = "missing"
	s.findSQL(false)
	if s.statusKey != "find_not_found" {
		t.Fatal("missing text was not reported")
	}
}

func TestSQLHistoryUsesSubmittedSelectionAndRequiresOverwriteConfirmation(t *testing.T) {
	s := shortcutScreen(t)
	for i := 0; i < 102; i++ {
		query := fmt.Sprintf("SELECT %d FROM dual", i)
		s.editor.SetText(query+"\n-- draft", false).Select(0, len(query))
		s.key(tuivory.NewEventKey(tuivory.KeyF9, 0, tuivory.ModShift))
		j := <-s.jobs
		if j.text != query || j.action != "execute" {
			t.Fatal("Shift+F9 did not execute selected SQL")
		}
		s.cancel()
		s.cancel = nil
		s.busy = false
	}
	if len(s.sqlHistory) != 100 || s.sqlHistory[0] != "SELECT 2 FROM dual" {
		t.Fatal("history did not enforce its memory limit")
	}
	s.editor.SetText("unfinished draft", false)
	s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, 0))
	if s.historyList == nil {
		t.Fatal("F8 did not open history")
	}
	text, _ := s.historyList.GetItemText(0)
	if !strings.Contains(text, "101") {
		t.Fatal("history is not newest first")
	}
	load := func() {
		s.historyList.InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
	}
	load()
	if s.programModal == nil {
		t.Fatal("history overwrite did not ask")
	}
	modalKey(s, tuivory.KeyEnter)
	if s.editor.GetText() != "unfinished draft" {
		t.Fatal("history overwrite default lost draft")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, 0))
	load()
	modalKey(s, tuivory.KeyLeft)
	modalKey(s, tuivory.KeyEnter)
	if s.editor.GetText() != "SELECT 101 FROM dual" || len(s.jobs) != 0 || !s.editor.HasFocus() {
		t.Fatal("confirmed history load executed or lost SQL")
	}
	cancelled := false
	s.busy = true
	s.cancel = func() { cancelled = true }
	s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, 0))
	if s.historyList != nil || cancelled {
		t.Fatal("F8 still cancels running work")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if !cancelled {
		t.Fatal("Esc did not cancel running work")
	}
}
