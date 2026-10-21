package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestLocalResultSortFilterPreservesTypesRawCopyAndReplay(t *testing.T) {
	s, _ := resultScreen(t)
	numbers := []string{"99999999999999999999999999999999999998", "2", "10", "NULL", "99999999999999999999999999999999999999"}
	data := oracle.Result{Columns: []string{"N", "TEXT"}, Query: &oracle.QuerySpec{Text: "SELECT n,text FROM t"}, PageID: 7}
	for i, n := range numbers {
		text := "Keep\t한글\n[red]"
		if i == 2 {
			text = "Other"
		}
		data.Rows = append(data.Rows, []string{n, text})
		data.Values = append(data.Values, []oracle.Cell{{Text: n, Type: "NUMBER", Null: i == 3}, {Text: text, Type: "VARCHAR2"}})
	}
	s.render(data)
	s.table.Select(1, 0)
	s.sortResult(false)
	if s.resultRows[0][0] != "2" || s.resultRows[1][0] != "10" || s.resultRows[2][0] != numbers[0] || !s.resultValues[4][0].Null {
		t.Fatalf("exact numeric/NULL sort: %v", s.resultRows)
	}
	s.sortResult(true)
	if s.resultRows[0][0] != numbers[4] || !s.resultValues[4][0].Null {
		t.Fatal("descending order lost precision or NULL position")
	}
	s.resultBrowser.filter = "keep"
	s.applyResultBrowser(true)
	if len(s.resultRows) != 4 || s.resultQuery != data.Query {
		t.Fatal("local filter changed replay recipe or row count")
	}
	s.table.Select(1, 1)
	s.copyResults("selection")
	if !strings.Contains(s.clipboardText, "Keep\t한글\n[red]") {
		t.Fatal("filtered copy used rendered preview")
	}
	export, err := s.exportDataset(0)
	if err != nil || len(export.Rows) != 4 || export.Rows[0][0].Type != "NUMBER" || export.Rows[0][0].Text != numbers[4] {
		t.Fatal("local export lost view order or types")
	}
	s.clearBindValues()
	s.resetResultView()
	if s.resultQuery != nil || s.resultBrowser.source.Query != nil || s.resultRows[0][0] != numbers[0] {
		t.Fatal("reset view restored replay bindings or lost original order")
	}
}

func TestResultPageAppendKeepsLocalFilterAndClosesStaleCursor(t *testing.T) {
	s, _ := resultScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}, {"2"}}, PageID: 3, Truncated: true})
	s.resultBrowser.filter = "2"
	s.applyResultBrowser(true)
	s.acceptResultPage(oracle.Result{Rows: [][]string{{"20"}}, Values: [][]oracle.Cell{{{Text: "20", Type: "NUMBER"}}}, PageID: 3, Truncated: true}, nil)
	if len(s.resultBrowser.source.Rows) != 3 || len(s.resultRows) != 2 || s.resultRows[1][0] != "20" || s.resultValues[1][0].Type != "NUMBER" {
		t.Fatal("fetch lost filter/base rows/typed page")
	}
	s.initDocument()
	s.initTabs()
	s.newTab()
	s.switchTab(0)
	if s.resultBrowser.filter != "2" || len(s.resultBrowser.source.Rows) != 3 {
		t.Fatal("tabs lost browser state")
	}
	s.invalidateResultCursors(99)
	if s.resultBrowser.source.PageID != 0 {
		t.Fatal("another query's cursor remained fetchable")
	}
	s.acceptResultPage(oracle.Result{}, errors.New("fetch cancelled"))
	if len(s.resultRows) != 2 || s.resultBrowser.source.PageID != 0 || !s.resultTruncated {
		t.Fatal("cancel discarded already loaded rows")
	}
}

func TestResultFilterCancelAndHelpKeepPendingWork(t *testing.T) {
	s, _ := resultScreen(t)
	s.pendingChanges = true
	s.showResultFilter()
	_, root := s.pages.GetFrontPage()
	form := root.(*tview.Form)
	previous := s.app.GetFocus()
	form.GetFormItem(0).(*tview.InputField).SetText("not applied")
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.app.GetFocus() != previous {
		t.Fatal("help lost filter form focus")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.resultBrowser.filter != "" || !s.pendingChanges || !s.table.HasFocus() || len(s.jobs) != 0 {
		t.Fatal("cancel applied filter or changed pending work")
	}
}

func TestFetchKeepsLogicalCellAfterSortingAndNewSQLResetsScroll(t *testing.T) {
	s, _ := resultScreen(t)
	result := oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"2"}, {"10"}}, Values: [][]oracle.Cell{{{Text: "2", Type: "NUMBER"}}, {{Text: "10", Type: "NUMBER"}}}, PageID: 4}
	s.render(result)
	s.sortResult(true)
	s.table.Select(2, 0)
	s.acceptResultPage(oracle.Result{Rows: [][]string{{"20"}}, Values: [][]oracle.Cell{{{Text: "20", Type: "NUMBER"}}}}, nil)
	row, _ := s.table.GetSelection()
	if row != 3 || s.resultRows[row-1][0] != "2" {
		t.Fatal("fetch moved selection to a different logical row")
	}
	s.table.ScrollToEnd()
	s.render(result)
	offset, _ := s.table.GetOffset()
	if offset != 0 {
		t.Fatal("new SQL retained previous end tracking/scroll")
	}
}
