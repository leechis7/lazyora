package ui

import (
	"lazyora/internal/oracle"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestRelatedRowsUseSelectedPreviewRowAndReturnToIt(t *testing.T) {
	s := demoBrowserScreen(t)
	s.openObjectDetails(oracle.Object{Owner: "APP", Name: "ORDERS", Kind: "TABLE"}, true)
	s.selectObjectTab("data")
	w := s.objectWorkspace
	s.connected = true
	w.dataModel.Focus("1", "0")
	s.app.SetFocus(w.dataGrid)
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'f', tuivory.ModAlt))
	load := <-s.jobs
	if load.action != "foreign_keys" || load.object != s.object {
		t.Fatalf("wrong relationship job: %v", load.action)
	}
	s.cancel()
	s.busy = false
	link := oracle.ForeignKeyLink{Target: oracle.Object{Owner: "APP", Name: "DETAILS", Kind: "TABLE"}, Columns: []oracle.ForeignKeyColumn{{Source: "ID", Target: "ORDER_ID"}}}
	s.acceptForeignKeys(oracle.ForeignKeyLinks{Links: []oracle.ForeignKeyLink{link}}, nil)
	connectionKey(s, tuivory.KeyEnter)
	query := <-s.jobs
	if query.action != "related_rows" || query.related.link.Target != link.Target || query.related.row[0].Text != "2" {
		t.Fatal("related query did not preserve the selected row")
	}
	s.cancel()
	s.busy = false
	s.acceptRelatedRows(query.related, oracle.Result{Columns: []string{"ORDER_ID"}, Values: [][]oracle.Cell{{{Text: "2"}}}}, nil)
	if page, _ := s.pages.GetFrontPage(); page != "related_rows" {
		t.Fatal("related result did not open")
	}
	connectionKey(s, tuivory.KeyEscape)
	connectionKey(s, tuivory.KeyEscape)
	if page, _ := s.pages.GetFrontPage(); page != "object" || !w.dataGrid.HasFocus() || w.dataModel.RowID != "1" {
		t.Fatal("related navigation did not restore the original row")
	}
}
