package ui

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFullExportReplaysReviewedQueryAndDefaultsToCancel(t *testing.T) {
	s := fileTestScreen(t)
	s.connected = true
	values := map[string]oracle.BindValue{"N": {Kind: "text", Text: "secret original"}}
	query, err := oracle.NewQuerySpec("SELECT :n AS N FROM dual", values)
	if err != nil {
		t.Fatal(err)
	}
	s.render(oracle.Result{Query: query, Columns: []string{"N"}, Rows: [][]string{{"secret original"}}, Truncated: true})
	s.editor.SetText("DELETE FROM unrelated", false)
	values["N"] = oracle.BindValue{Kind: "text", Text: "later value"}
	s.showExport()
	path := filepath.Join(t.TempDir(), "all.csv")
	s.exportForm.GetFormItem(1).(*tview.InputField).SetText(path)
	s.exportForm.GetFormItem(2).(*choiceField).SetCurrentOption(exportFullScope)
	s.exportSubmit()
	if s.sqlConfirmText == nil || len(s.jobs) != 0 || !strings.Contains(s.sqlConfirmText.GetText(false), query.Text) || strings.Contains(s.sqlConfirmText.GetText(false), "secret original") {
		t.Fatal("requery wasn't reviewed or bind value leaked")
	}
	warningKey(s, tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if len(s.jobs) != 0 || s.sqlConfirmText != nil || s.exportForm == nil {
		t.Fatal("default requery action is not Cancel")
	}
	s.exportSubmit()
	warningKey(s, tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	warningKey(s, tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	j := <-s.jobs
	defer s.cancel()
	if j.query == nil || j.query.Text != query.Text || j.query.Bindings["N"].Text != "secret original" || len(j.dataset.Rows) != 0 || j.path != path {
		t.Fatal("export used editor/new binds or copied display rows")
	}
	if _, deadline := j.ctx.Deadline(); deadline {
		t.Fatal("full export retained 60-second deadline")
	}
	if !s.pendingChanges || len(s.sqlHistory) != 0 {
		t.Fatal("replay confirmation changed pending work/history")
	}
}

func TestFullExportTabAndClearLifecycle(t *testing.T) {
	s := fileTestScreen(t)
	s.connected = true
	query, _ := oracle.NewQuerySpec("SELECT :n FROM dual", map[string]oracle.BindValue{"N": {Text: "old", Kind: "text"}})
	s.render(oracle.Result{Query: query, Columns: []string{"N"}, Rows: [][]string{{"old"}}})
	s.initTabs()
	s.newTab()
	if s.resultQuery != nil {
		t.Fatal("new tab inherited replay query")
	}
	s.switchTab(0)
	if s.resultQuery == nil || s.resultQuery.Text != query.Text {
		t.Fatal("tab lost original query")
	}
	s.scriptReport = oracle.ScriptReport{Steps: []oracle.ScriptStep{{Result: oracle.Result{Query: query}}}}
	s.clearBindValues() // Also used on successful reconnect.
	s.showExport()
	s.exportForm.GetFormItem(1).(*tview.InputField).SetText(filepath.Join(t.TempDir(), "all.csv"))
	s.exportForm.GetFormItem(2).(*choiceField).SetCurrentOption(exportFullScope)
	s.exportSubmit()
	if s.resultQuery != nil || s.scriptReport.Steps[0].Result.Query != nil || len(s.jobs) != 0 || s.sqlConfirmText != nil {
		t.Fatal("replay survived bind clear/reconnect")
	}
	s.closeExport()
	s.switchTab(1)
	s.switchTab(0)
	if s.resultQuery != nil || len(s.resultRows) != 1 {
		t.Fatal("stale replay restored or display snapshot lost")
	}
	s.closeTab()
	if s.resultQuery != nil {
		t.Fatal("closed tab retained query")
	}
}

func TestExportBusyFreezesFieldsAndCancels(t *testing.T) {
	s := fileTestScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}})
	s.showExport()
	s.busy = true
	cancelled := false
	s.cancel = func() { cancelled = true }
	for _, event := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyRune, 'x', 0), tcell.NewEventKey(tcell.KeyCtrlS, 0, 0), tcell.NewEventKey(tcell.KeyEnter, 0, 0)} {
		if s.exportKey(event) != nil {
			t.Fatal("fields/buttons remain editable during export")
		}
	}
	s.exportKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if !cancelled || s.exportForm == nil {
		t.Fatal("Esc did not request cancellation or discarded progress before worker completed")
	}
}
