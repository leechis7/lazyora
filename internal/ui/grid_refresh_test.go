package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"lazyora/internal/oracle"
)

func TestTransactionGridRefreshUsesResultRecipeAndPreservesView(t *testing.T) {
	s, _ := resultScreen(t)
	s.connected = true
	query, err := oracle.NewQuerySpec("SELECT t.ROWID rid,t.id key,t.title caption FROM items t WHERE t.id>=:n", map[string]oracle.BindValue{"N": {Kind: "number", Text: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	result := oracle.Result{EditID: 9, EditRowIDColumn: 1, Query: query, EditObject: oracle.Object{Owner: "APP", Name: "ITEMS", Kind: "TABLE"},
		Columns: []string{"RID", "KEY", "CAPTION"}, EditColumns: []bool{false, true, true}, EditColumnNames: []string{"", "ID", "TITLE"}, EditRowIDs: []string{"rid1", "rid2"},
		Rows:   [][]string{{"rid1", "1", "first"}, {"rid2", "2", "second changed"}},
		Values: [][]oracle.Cell{{{Text: "rid1", Type: "ROWID"}, {Text: "1", Type: "NUMBER"}, {Text: "first", Type: "VARCHAR2"}}, {{Text: "rid2", Type: "ROWID"}, {Text: "2", Type: "NUMBER"}, {Text: "second changed", Type: "VARCHAR2"}}}}
	s.render(result)
	s.resultBrowser.filter = "second"
	s.resultBrowser.sorted, s.resultBrowser.descending, s.resultBrowser.sortColumn = true, true, 0
	s.applyResultBrowser(false)
	s.grid.widths = []int{12, 35}
	s.table.Select(1, 1)
	s.editor.SetText("DELETE FROM unrelated", false)
	s.app.SetFocus(s.table)
	s.markEditedCell(s.editedCellIdentity(result, 1, 2))
	// Transaction completion clears pending markers before the refresh job.
	s.clearEditedCells()
	s.invalidateEditResults(0)
	if !s.refreshTransactionGrid("rollback") {
		t.Fatal("result refresh not queued")
	}
	j := <-s.jobs
	defer s.cancel()
	if j.action != "result_refresh" || j.query.Text != query.Text || j.query.Bindings["N"].Text != "1" || j.gridRefresh.rowID != "rid2" {
		t.Fatal("refresh did not snapshot original SELECT/binds/row")
	}
	query.Bindings["N"] = oracle.BindValue{Kind: "number", Text: "999"}
	if j.query.Bindings["N"].Text != "1" {
		t.Fatal("queued binds changed with the source recipe")
	}
	result.Query = j.query
	result.EditID = 10
	result.Rows = [][]string{{"rid2", "2", "second"}, {"rid1", "1", "first"}}
	result.EditRowIDs = []string{"rid2", "rid1"}
	result.Values = [][]oracle.Cell{{{Text: "rid2", Type: "ROWID"}, {Text: "2", Type: "NUMBER"}, {Text: "second", Type: "VARCHAR2"}}, {{Text: "rid1", Type: "ROWID"}, {Text: "1", Type: "NUMBER"}, {Text: "first", Type: "VARCHAR2"}}}
	s.busy = false
	s.acceptTransactionGridRefresh(j, result, nil)
	row, col := s.table.GetSelection()
	if !reflect.DeepEqual(s.resultHeaders, []string{"KEY", "CAPTION"}) || len(s.resultRows) != 1 || s.resultRows[0][1] != "second" || row != 1 || col != 1 {
		t.Fatal("refresh lost hidden ROWID, filter, new values or selected physical row")
	}
	if !reflect.DeepEqual(s.grid.widths, []int{12, 35}) || !s.resultBrowser.sorted || !s.resultBrowser.descending || !s.table.HasFocus() || s.isEditedCell(1, 1) || !s.commandEnabled("grid_edit") {
		t.Fatal("refresh lost widths/sort/focus or did not renew edit capability")
	}
	if s.editor.GetText() != "DELETE FROM unrelated" || s.statusKey != "rollback_refreshed" || s.pendingChanges {
		t.Fatal("refresh changed SQL, transaction state or completion status")
	}
}

func TestTransactionRefreshEligibilityAndFailureStatus(t *testing.T) {
	for _, blocked := range []string{"exit", "uncertain", "disconnected", "busy", "missing_recipe", "non_select"} {
		t.Run(blocked, func(t *testing.T) {
			s, _ := inlineFixture(t)
			s.resultBrowser.source.Query = &oracle.QuerySpec{Text: "SELECT value FROM items"}
			switch blocked {
			case "exit":
				s.quitting = true
			case "uncertain":
				s.transactionUncertain = true
			case "disconnected":
				s.connected = false
			case "busy":
				s.busy = true
			case "missing_recipe":
				s.resultBrowser.source.Query = nil
			case "non_select":
				s.resultBrowser.source.Query.Text = "DELETE FROM items"
			}
			if s.refreshTransactionGrid("commit") || len(s.jobs) != 0 {
				t.Fatal("ineligible refresh submitted SQL")
			}
		})
	}
	s, _ := resultScreen(t)
	s.connected = true
	s.resultBrowser.source.Query, _ = oracle.NewQuerySpec("SELECT 1 FROM dual", nil)
	before := s.resultRows[0][0]
	if !s.refreshTransactionGrid("commit") {
		t.Fatal("read-only result not refreshable")
	}
	j := <-s.jobs
	defer s.cancel()
	s.busy = false
	s.acceptTransactionGridRefresh(j, oracle.Result{}, errors.New("query timed out"))
	if s.resultRows[0][0] != before || s.pendingChanges || s.statusKey != "commit_refresh_failed" || !strings.Contains(s.status.GetText(false), "Commit completed") {
		t.Fatal("reload failure was reported as transaction failure or erased the old view")
	}
}

func TestExplicitTableReloadDoesNotBroadenDirectSelect(t *testing.T) {
	s, _ := resultScreen(t)
	s.connected = true
	s.resultBrowser.source.EditObject = oracle.Object{Owner: "APP", Name: "ITEMS", Kind: "TABLE"}
	if !s.refreshTransactionGrid("rollback") {
		t.Fatal("explicit browser table load cannot refresh")
	}
	j := <-s.jobs
	defer s.cancel()
	if j.query != nil || j.object.Name != "ITEMS" || j.action != "result_refresh" {
		t.Fatal("wrong explicit table refresh target")
	}
}
