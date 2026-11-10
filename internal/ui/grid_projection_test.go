package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestHiddenRowIDKeepsOriginalEditColumnAndRowIdentity(t *testing.T) {
	for hidden := range 3 {
		t.Run(fmt.Sprint(hidden), func(t *testing.T) {
			s, _ := resultScreen(t)
			s.connected = true
			query := &oracle.QuerySpec{Text: "original SELECT with ROWID and binds"}
			r := oracle.Result{EditID: 42, EditRowIDColumn: hidden + 1, EditObject: oracle.Object{Owner: "APP", Name: "ITEMS", Kind: "TABLE"}, EditRowIDs: []string{"rid1", "rid2"}, Query: query}
			visible := 0
			for col := range 3 {
				name, original, editable := "KEY", "ID", true
				if col == hidden {
					name, original, editable = "ROW_LOCATOR", "", false
				} else {
					if visible == 1 {
						name, original = "CAPTION", "TITLE"
					}
					visible++
				}
				r.Columns = append(r.Columns, name)
				r.EditColumnNames = append(r.EditColumnNames, original)
				r.EditColumns = append(r.EditColumns, editable)
			}
			for row := range 2 {
				var values []string
				var cells []oracle.Cell
				for col := range 3 {
					value, kind := fmt.Sprint(row+1), "NUMBER"
					if col == hidden {
						value, kind = r.EditRowIDs[row], "ROWID"
					} else if r.EditColumnNames[col] == "TITLE" {
						value, kind = []string{"first", "second"}[row], "VARCHAR2"
					}
					values = append(values, value)
					cells = append(cells, oracle.Cell{Text: value, Type: kind})
				}
				r.Rows = append(r.Rows, values)
				r.Values = append(r.Values, cells)
			}
			s.render(r)
			s.app.SetFocus(s.table)
			if !reflect.DeepEqual(s.resultHeaders, []string{"KEY", "CAPTION"}) || s.table.GetColumnCount() != 2 {
				t.Fatalf("ROWID not hidden: %v", s.resultHeaders)
			}
			s.table.Select(1, 0)
			s.sortResult(true)
			s.table.Select(1, 1)
			s.copyResults("row")
			if s.clipboardText != "2\tsecond" {
				t.Fatalf("hidden column leaked into row copy: %q", s.clipboardText)
			}
			data, err := s.exportDataset(0)
			if err != nil || len(data.Columns) != 2 || data.Rows[0][1].Text != "second" {
				t.Fatal("loaded-result export lost visible projection")
			}
			s.refreshFooter()
			if !strings.Contains(s.footer.GetText(false), "Alt+D") {
				t.Fatal("dialog shortcut missing from footer")
			}
			s.key(tuivory.NewEventKey(tuivory.KeyRune, 'd', tuivory.ModAlt))
			if s.editForm == nil || s.editChange.Row != 1 || s.editChange.Column != s.resultSourceColumn(1) {
				t.Fatal("Alt+D mapped to the wrong original cell")
			}
			s.editForm.GetFormItem(0).(*tuivory.InputField).SetText("new")
			s.stageGridEdit()
			change := s.gridDrafts[editedCellKey{"APP", "ITEMS", "rid2", "TITLE"}]
			if len(s.jobs) != 0 || change.Row != 1 || change.Column != s.resultSourceColumn(1) || change.Text != "new" {
				t.Fatalf("wrong draft identity: %+v", change)
			}
			if !s.isEditedCell(1, 1) || s.isEditedCell(2, 1) {
				t.Fatal("edited marker moved to another row")
			}
			s.resultBrowser.filter = "rid2"
			s.applyResultBrowser(true)
			if len(s.resultRows) != 0 {
				t.Fatal("filter matched hidden ROWID")
			}
			s.resultBrowser.filter = "new"
			s.applyResultBrowser(true)
			s.table.Select(1, 1)
			if !s.startInlineEdit(nil) || s.inlineEdit.change.Column != s.resultSourceColumn(1) || s.inlineEdit.change.Row != 1 || !s.isEditedCell(1, 1) {
				t.Fatal("filtered inline edit or marker lost original mapping")
			}
			s.closeInlineEdit()
			if len(s.resultBrowser.source.Columns) != 3 || s.resultBrowser.source.Rows[1][hidden] != "rid2" {
				t.Fatal("hiding modified the source snapshot")
			}
			s.resultBrowser.filter = ""
			s.applyResultBrowser(true)
			s.table.Select(1, 1)
			page := oracle.Result{EditRowIDs: []string{"rid3"}}
			values := make([]string, 3)
			cells := append([]oracle.Cell(nil), r.Values[1]...)
			for col := range cells {
				value := "3"
				if col == hidden {
					value = "rid3"
				} else if r.EditColumnNames[col] == "TITLE" {
					value = "third"
				}
				values[col], cells[col].Text = value, value
			}
			page.Rows, page.Values = [][]string{values}, [][]oracle.Cell{cells}
			s.acceptResultPage(page, nil)
			if len(s.resultHeaders) != 2 || s.resultRows[0][1] != "third" || !s.isEditedCell(2, 1) {
				t.Fatal("paging lost hidden projection or physical marker")
			}
			s.initDocument()
			s.initTabs()
			s.newTab()
			s.switchTab(0)
			if len(s.resultHeaders) != 2 || s.resultSourceColumn(1) != slices.Index(r.EditColumnNames, "TITLE") || !s.isEditedCell(2, 1) {
				t.Fatal("tab switch lost original column mapping")
			}
			s.invalidateEditResults(0)
			s.applyResultBrowser(true)
			if len(s.resultHeaders) != 2 || s.commandEnabled("grid_edit") {
				t.Fatal("expired snapshot exposed ROWID or stayed editable")
			}
		})
	}
}

func TestUnverifiedRowIDLabelRemainsVisible(t *testing.T) {
	s, _ := resultScreen(t)
	s.render(oracle.Result{Columns: []string{"ROWID"}, Rows: [][]string{{"ordinary value"}}})
	if len(s.resultHeaders) != 1 || s.resultRows[0][0] != "ordinary value" {
		t.Fatal("an unverified label was treated as physical identity")
	}
}
