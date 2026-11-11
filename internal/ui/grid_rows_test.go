package ui

import (
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestGridRowDraftsUseVerifiedSourceAndStayOnTheirRows(t *testing.T) {
	s := shortcutScreen(t)
	s.connected = true
	s.render(oracle.Result{
		EditID: 42, EditObject: oracle.Object{Owner: "APP", Name: "DATA", Kind: "TABLE"},
		EditCanDelete: true, EditRowIDColumn: 1, EditRowIDs: []string{"first", "second"},
		EditColumnNames: []string{"", "TITLE", "OTHER"}, EditColumnTypes: []string{"", "VARCHAR2", "VARCHAR2"}, EditColumns: []bool{false, true, true},
		Columns: []string{"HIDDEN", "TITLE", "OTHER"},
		Rows:    [][]string{{"first", "a", "1"}, {"second", "b", "2"}},
	})
	s.app.SetFocus(s.table)
	s.table.Select(1, 0)
	s.stageGridChange(oracle.EditChange{ID: 42, Row: 0, Column: 1, Text: "edited"})
	s.invoke("grid_delete")
	if s.commandEnabled("grid_inline") || !s.hasPendingWork() || len(s.jobs) != 0 {
		t.Fatal("deletion should stage without executing or allowing cell edits")
	}
	s.invoke("grid_delete")
	if !s.commandEnabled("grid_inline") || len(s.gridDrafts) != 1 {
		t.Fatal("undoing deletion should restore cell draft")
	}
	s.invoke("grid_delete")
	s.invoke("grid_insert")
	name, page := s.pages.GetFrontPage()
	if name != "grid_insert" {
		t.Fatal("verified result did not open insert form")
	}
	form := page.(*formPopupLayer).content.(*tuivory.Flex).GetItem(1).(*tuivory.WidgetForm)
	columns := form.GetFormItem(0).(*choiceField)
	mode := form.GetFormItem(1).(*choiceField)
	field := form.GetFormItem(2).(*tuivory.InputField)
	mode.SetCurrentOption(2)
	field.SetText("inserted")
	columns.SetCurrentOption(1)
	mode.SetCurrentOption(1)
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlS, 0, 0))
	if len(s.resultRows) != 3 || len(s.resultHeaders) != 2 || s.resultRows[2][0] != "inserted" || !s.resultValues[2][1].Null {
		t.Fatal("new row or hidden identity was projected incorrectly")
	}
	s.sortResult(true)
	s.prepareResultGrid()
	model := s.grid.presentation.model
	for _, tt := range []struct{ id, mark string }{{"row:first", "−"}, {"new:1", "+"}} {
		// Locate the displayed line using the SDK's source identity rather than
		// the pre-sort row number.
		source := resultSource{s}
		r := source.IndexOf(tt.id)
		if r < 0 || source.Row(r).ID != tt.id {
			t.Fatalf("missing stable ID after sort: %s", tt.id)
		}
		lines := model.Layout(90, 16)
		marked := false
		for _, span := range lines[r+1].Spans {
			marked = marked || span.Role == "grid.gutter" && strings.Contains(span.Text, tt.mark)
		}
		if !marked {
			t.Fatalf("wrong marker for %s: %+v", tt.id, lines[r+1])
		}
	}
	s.invoke("commit")
	j := <-s.jobs
	s.cancel()
	if len(j.deletes) != 1 || j.deletes[0].Row != 0 || len(j.edits) != 0 || len(j.inserts) != 1 ||
		j.inserts[0].Fields[1].Text != "inserted" || !j.inserts[0].Fields[2].Null || s.gridDraftCount() != 3 {
		t.Fatalf("wrong batch or drafts cleared before success: %+v", j)
	}
	s.clearGridDrafts()
	if s.gridDraftCount() != 0 || len(s.resultRows) != 2 {
		t.Fatal("clearing drafts left preview rows")
	}
}
