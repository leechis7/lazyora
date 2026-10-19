package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestExportScopesKeepRawNULLAndLiteralValues(t *testing.T) {
	s := fileTestScreen(t)
	s.render(oracle.Result{Columns: []string{"A", "B"}, Rows: [][]string{{"NULL", "NULL"}, {"00123", "=1+1"}}, Values: [][]oracle.Cell{{{Text: "NULL", Null: true}, {Text: "NULL", Type: "VARCHAR2"}}, {{Text: "00123", Type: "VARCHAR2"}, {Text: "=1+1", Type: "VARCHAR2"}}}, Truncated: true})
	data, err := s.exportDataset(0)
	if err != nil || len(data.Rows) != 2 || !data.Rows[0][0].Null || data.Rows[0][1].Null {
		t.Fatal("loaded result metadata lost")
	}
	s.resultSelection = resultSelection{active: true, anchorRow: 2, endRow: 2, anchorCol: 1, endCol: 1}
	data, err = s.exportDataset(1)
	if err != nil || len(data.Columns) != 1 || data.Columns[0] != "B" || data.Rows[0][0].Text != "=1+1" {
		t.Fatal("rectangle scope wrong")
	}
	s.table.Select(2, 1)
	data, err = s.exportDataset(2)
	if err != nil || len(data.Rows) != 1 || len(data.Columns) != 2 || data.Rows[0][0].Text != "00123" {
		t.Fatal("row scope wrong")
	}
}

func TestExportUIOverwriteCancelAndJobSnapshot(t *testing.T) {
	s := fileTestScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}})
	s.showExport()
	form := s.exportForm
	path := filepath.Join(t.TempDir(), "existing.csv")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	form.GetFormItem(1).(*tview.InputField).SetText(path)
	s.exportSubmit()
	if s.programModal == nil || len(s.jobs) != 0 {
		t.Fatal("overwrite not confirmed")
	}
	modalChoice(s, s.programModal, 1)
	raw, _ := os.ReadFile(path)
	if string(raw) != "keep" || s.exportForm == nil {
		t.Fatal("cancel lost file/form")
	}
	s.exportSubmit()
	modalChoice(s, s.programModal, 0)
	j := <-s.jobs
	if j.action != "export" || j.dataset.Rows[0][0].Text != "1" || !j.revision.Exists || j.path != path || !s.pendingChanges {
		t.Fatal("export job lost snapshot or touched DB")
	}
	s.busy = false
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.exportForm != nil {
		t.Fatal("Esc did not close export")
	}
}

func TestExportKeyboardAndInsertTargetRequired(t *testing.T) {
	s := fileTestScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}})
	s.key(tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModAlt))
	if s.exportForm == nil {
		t.Fatal("export shortcut missing")
	}
	s.exportForm.GetFormItem(0).(*choiceField).SetCurrentOption(3)
	s.exportSubmit()
	if len(s.jobs) != 0 || s.exportForm == nil {
		t.Fatal("INSERT guessed target table")
	}
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "export" {
		t.Fatal("help lost export page")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("export cancel resolved DB work")
	}
}

func TestExportWorkerWithoutDBAndHelpDuringSave(t *testing.T) {
	s := runningVaultUI(t)
	path := filepath.Join(t.TempDir(), "result.xlsx")
	s.app.QueueUpdateDraw(func() {
		s.showExport()
		s.exportForm.GetFormItem(0).(*choiceField).SetCurrentOption(2)
		s.exportForm.GetFormItem(1).(*tview.InputField).SetText(path)
		s.exportSubmit()
		s.showHelp()
	})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		complete := false
		s.app.QueueUpdateDraw(func() {
			complete = !s.busy && s.exportForm == nil
			if complete {
				if page, _ := s.pages.GetFrontPage(); page != "main" {
					t.Errorf("export completion left page %s", page)
				}
				if !s.editor.HasFocus() && !s.table.HasFocus() {
					t.Error("export completion left detached focus")
				}
			}
		})
		if complete {
			raw, err := os.ReadFile(path)
			if err != nil || len(raw) < 2 || string(raw[:2]) != "PK" {
				t.Fatal("demo worker did not write XLSX")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("export worker did not finish")
}
