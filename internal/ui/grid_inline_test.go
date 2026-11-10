package ui

import (
	"lazyora/internal/oracle"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func setInlineText(s *screen, text string) {
	s.inlineEdit.model.EditInput("select_all", "")
	s.inlineEdit.model.EditInput("insert", text)
}

func inlineFixture(t *testing.T) (*screen, tuivory.SimulationScreen) {
	t.Helper()
	s, terminal := resultScreen(t)
	s.connected = true
	s.render(oracle.Result{EditID: 5, EditObject: oracle.Object{Owner: "APP", Name: "ITEMS", Kind: "TABLE"}, EditColumns: []bool{false, true}, EditColumnNames: []string{"", "TITLE"}, EditRowIDs: []string{"rid1", "rid2"},
		Columns: []string{"RID", "CAPTION"}, Rows: [][]string{{"rid1", "first"}, {"rid2", "second"}}, Values: [][]oracle.Cell{{{Text: "rid1", Type: "ROWID"}, {Text: "first", Type: "VARCHAR2"}}, {{Text: "rid2", Type: "ROWID"}, {Text: "second", Type: "VARCHAR2"}}}})
	s.table.Select(1, 1)
	s.app.SetFocus(s.table)
	return s, terminal
}

// Set the same draft state used by the rendering audits, retaining the fixture value.
func (s *screen) markEditedCell(key editedCellKey) {
	if s.gridDrafts == nil {
		s.gridDrafts = make(map[editedCellKey]oracle.EditChange)
	}
	for row := range s.resultBrowser.source.Rows {
		for col := range s.resultBrowser.source.Columns {
			if key == s.editedCellIdentity(s.resultBrowser.source, row, col) {
				cell := browserCell(s.resultBrowser.source, row, col)
				s.gridDrafts[key] = oracle.EditChange{ID: s.resultBrowser.source.EditID, Row: row, Column: col, Text: cell.Text, Null: cell.Null}
			}
		}
	}
	s.repaintEditedCells()
}
func TestInlineCellStagesWithoutConfirmation(t *testing.T) {
	s, _ := inlineFixture(t)
	s.table.Select(1, 0)
	if s.startInlineEdit(nil) {
		t.Fatal("ROWID editable")
	}
	s.table.Select(1, 1)
	s.sortResult(true)
	s.table.Select(1, 1)
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	if s.inlineEdit == nil || s.inlineEdit.GetText() != "second" || s.inlineEdit.change.Row != 1 {
		t.Fatal("inline input lost sorted row mapping")
	}
	setInlineText(s, "new")
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	if s.programModal != nil || s.inlineEdit != nil || len(s.jobs) != 0 || s.resultRows[0][1] != "new" {
		t.Fatal("Enter did not stage the edit locally")
	}
	s.startInlineEdit(nil)
	setInlineText(s, "discard")
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if s.inlineEdit != nil || len(s.jobs) != 0 || s.resultBrowser.source.Rows[1][1] != "second" {
		t.Fatal("Esc changed data")
	}
	// The original popup is still available.
	s.showGridEdit()
	if s.editForm == nil {
		t.Fatal("separate editor removed")
	}
}
func TestInlineReplaceNullAndModalInputIsolation(t *testing.T) {
	s, terminal := inlineFixture(t)
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'X', 0))
	if s.inlineEdit == nil || s.inlineEdit.GetText() != "X" {
		t.Fatal("typing did not start replacement")
	}
	for _, width := range []int{40, 80, 120} {
		terminal.SetSize(width, 30)
		s.table.SetRect(0, 0, width, 25)
		s.resultGrid().Draw(terminal)
		s.resultGrid().Draw(terminal)
		x, y, w := s.resultCellRect(s.table.GetSelection())
		h := 1
		if x < 0 || x+w > width || y < 0 || h != 1 {
			t.Fatal("inline widget outside grid")
		}
	}
	s.key(tuivory.NewEventKey(tuivory.KeyDelete, 0, tuivory.ModAlt))
	session, _ := s.inlineEdit.model.EditSession()
	if !session.Value.Null {
		t.Fatal("NULL action missing")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	if len(s.jobs) != 0 || !s.resultValues[0][1].Null {
		t.Fatal("NULL draft was not staged")
	}
	s.startInlineEdit(nil)
	s.key(tuivory.NewEventKey(tuivory.KeyTab, 0, 0))
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'j', tuivory.ModAlt))
	if s.inlineEdit == nil {
		t.Fatal("application shortcut escaped editor")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	s.startInlineEdit(nil)
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	if s.inlineEdit != nil || s.programModal != nil || len(s.jobs) != 0 {
		t.Fatal("unchanged value generated UPDATE")
	}
}
func TestEditedCellMarkersFollowPhysicalIdentityAndTransaction(t *testing.T) {
	s, _ := inlineFixture(t)
	key := s.editedCellIdentity(s.resultBrowser.source, 1, 1)
	s.markEditedCell(key)
	if !s.isEditedCell(2, 1) || s.isEditedCell(1, 1) || s.isEditedCell(2, 0) {
		t.Fatal("wrong modified cell")
	}
	s.table.Select(1, 1)
	s.sortResult(true)
	if !s.isEditedCell(1, 1) || s.isEditedCell(2, 1) {
		t.Fatal("marker lost sorting")
	}
	s.resultBrowser.filter = "second"
	s.applyResultBrowser(true)
	if !s.isEditedCell(1, 1) {
		t.Fatal("marker lost filtering")
	}
	result := s.resultBrowser.source
	result.Columns[1] = "NEW_ALIAS"
	s.render(result)
	if !s.isEditedCell(2, 1) {
		t.Fatal("alias/reload lost marker")
	}
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, limited := range []bool{false, true} {
			s.prefs.Theme = theme
			s.limitedColors = limited
			s.repaintEditedCells()
			_, changed, _ := s.table.GetCell(2, 1).Style.Decompose()
			_, normal, _ := s.table.GetCell(2, 0).Style.Decompose()
			if changed == normal {
				t.Fatalf("marker invisible: %s limited=%v", theme, limited)
			}
		}
	}
	s.clearGridDrafts()
	if s.isEditedCell(2, 1) {
		t.Fatal("transaction end retained marker")
	}
}

func TestInlineDoubleClickAndFocusedModifiedCellColor(t *testing.T) {
	s, terminal := inlineFixture(t)
	s.prefs.Mouse = true
	s.limitedColors = true
	s.table.SetRect(0, 0, 120, 25)
	s.resultGrid().Draw(terminal)
	x, y, _ := s.resultCellRect(1, 1)
	action, _ := s.resultMouse(tuivory.MouseLeftDoubleClick, tuivory.NewEventMouse(x, y, 0, 0))
	if action != tuivory.MouseConsumed || s.inlineEdit == nil || s.inlineEdit.GetText() != "first" {
		t.Fatal("double-click did not open inline editor")
	}
	s.closeInlineEdit()
	s.markEditedCell(s.editedCellIdentity(s.resultBrowser.source, 0, 1))
	s.resultGrid().Draw(terminal)
	x, y, _ = s.resultCellRect(1, 1)
	_, _, style, _ := terminal.GetContent(x, y)
	_, bg, _ := style.Decompose()
	if bg != tuivory.ColorOlive {
		t.Fatalf("selected modified cell lost background: %v", bg)
	}
	s.initDocument()
	s.initTabs()
	s.newTab()
	if len(s.tabs) != 1 || !s.isEditedCell(1, 1) {
		t.Fatal("opening another tab hid an unresolved draft")
	}
	s.clearGridDrafts()
	s.newTab()
	if len(s.tabs) != 2 {
		t.Fatal("tab remained blocked after clearing draft")
	}
}

func TestGridDraftCommitIncludesAllRowsAndColumns(t *testing.T) {
	s, _ := inlineFixture(t)
	source := s.resultBrowser.source
	source.Columns = append(source.Columns, "AMOUNT")
	source.EditColumns = append(source.EditColumns, true)
	source.EditColumnNames = append(source.EditColumnNames, "AMOUNT")
	for i := range source.Rows {
		source.Rows[i] = append(source.Rows[i], "10")
		source.Values[i] = append(source.Values[i], oracle.Cell{Type: "NUMBER", Text: "10"})
	}
	s.render(source)
	s.table.Select(1, 1)
	s.startInlineEdit(nil)
	setInlineText(s, "title")
	s.key(tuivory.NewEventKey(tuivory.KeyTab, 0, 0))
	if s.inlineEdit == nil || s.inlineEdit.change.Row != 0 || s.inlineEdit.change.Column != 2 {
		t.Fatal("Tab did not move to next editable column")
	}
	setInlineText(s, "11")
	s.key(tuivory.NewEventKey(tuivory.KeyTab, 0, 0))
	if s.inlineEdit == nil || s.inlineEdit.change.Row != 1 || s.inlineEdit.change.Column != 1 {
		t.Fatal("Tab did not skip the next row's ROWID")
	}
	setInlineText(s, "row two")
	s.key(tuivory.NewEventKey(tuivory.KeyBacktab, 0, 0))
	if s.inlineEdit == nil || s.inlineEdit.change.Row != 0 || s.inlineEdit.change.Column != 2 {
		t.Fatal("Shift+Tab did not return to previous editable cell")
	}
	if len(s.jobs) != 0 || s.programModal != nil || source.Rows[0][1] != "first" {
		t.Fatal("local staging executed SQL or changed the snapshot")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'c', tuivory.ModAlt))
	select {
	case j := <-s.jobs:
		if j.action != "commit" || len(j.edits) != 3 || j.edits[0].Text != "title" || j.edits[1].Text != "11" || j.edits[2].Text != "row two" {
			t.Fatalf("wrong commit batch: %+v", j.edits)
		}
	default:
		t.Fatal("commit was not submitted")
	}
	s.cancel()
}
