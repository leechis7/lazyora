package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"testing"
)

func inlineFixture(t *testing.T) (*screen, tcell.SimulationScreen) {
	t.Helper()
	s, terminal := resultScreen(t)
	s.connected = true
	s.render(oracle.Result{EditID: 5, EditObject: oracle.Object{Owner: "APP", Name: "ITEMS", Kind: "TABLE"}, EditColumns: []bool{false, true}, EditColumnNames: []string{"", "TITLE"}, EditRowIDs: []string{"rid1", "rid2"},
		Columns: []string{"RID", "CAPTION"}, Rows: [][]string{{"rid1", "first"}, {"rid2", "second"}}, Values: [][]oracle.Cell{{{Text: "rid1", Type: "ROWID"}, {Text: "first", Type: "VARCHAR2"}}, {{Text: "rid2", Type: "ROWID"}, {Text: "second", Type: "VARCHAR2"}}}})
	s.table.Select(1, 1)
	s.app.SetFocus(s.table)
	return s, terminal
}
func confirmInline(t *testing.T, s *screen, proceed bool) {
	t.Helper()
	if s.programModal == nil || len(s.jobs) != 0 {
		t.Fatal("missing confirmation or premature UPDATE")
	}
	delegate := func(p tview.Primitive) { s.app.SetFocus(p) }
	if !proceed {
		s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyRight, 0, 0), delegate)
	}
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), delegate)
}
func TestInlineCellEditConfirmationAndCancel(t *testing.T) {
	s, _ := inlineFixture(t)
	s.table.Select(1, 0)
	if s.startInlineEdit(nil) {
		t.Fatal("ROWID editable")
	}
	s.table.Select(1, 1)
	s.sortResult(true)
	s.table.Select(1, 1)
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if s.inlineEdit == nil || s.inlineEdit.GetText() != "second" || s.inlineEdit.change.Row != 1 {
		t.Fatal("inline input lost sorted row mapping")
	}
	s.inlineEdit.SetText("new")
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	confirmInline(t, s, false)
	if s.inlineEdit == nil || s.inlineEdit.GetText() != "new" {
		t.Fatal("cancel confirmation lost draft")
	}
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	confirmInline(t, s, true)
	j := <-s.jobs
	if j.edit.Row != 1 || j.edit.Text != "new" || j.editKey.rowID != "rid2" || j.editKey.column != "TITLE" || s.inlineEdit != nil {
		t.Fatalf("wrong inline edit job: %+v", j)
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	if len(s.editedCells) != 0 {
		t.Fatal("unapplied job marked edited")
	}
	s.startInlineEdit(nil)
	s.inlineEdit.SetText("discard")
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
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
	s.key(tcell.NewEventKey(tcell.KeyRune, 'X', 0))
	if s.inlineEdit == nil || s.inlineEdit.GetText() != "X" {
		t.Fatal("typing did not start replacement")
	}
	for _, width := range []int{40, 80, 120} {
		terminal.SetSize(width, 30)
		s.table.SetRect(0, 0, width, 25)
		s.resultGrid().Draw(terminal)
		s.inlineEdit.Draw(terminal)
		x, y, w, h := s.inlineEdit.GetRect()
		if x < 0 || x+w > width || y < 0 || h != 1 {
			t.Fatal("inline widget outside grid")
		}
	}
	s.key(tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModAlt))
	if !s.inlineEdit.change.Null {
		t.Fatal("NULL action missing")
	}
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	confirmInline(t, s, true)
	j := <-s.jobs
	if !j.edit.Null {
		t.Fatal("NULL lost")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.startInlineEdit(nil)
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModAlt))
	if s.inlineEdit == nil {
		t.Fatal("application shortcut escaped editor")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	s.startInlineEdit(nil)
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
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
	s.clearEditedCells()
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
	action, _ := s.resultMouse(tview.MouseLeftDoubleClick, tcell.NewEventMouse(x, y, 0, 0))
	if action != tview.MouseConsumed || s.inlineEdit == nil || s.inlineEdit.GetText() != "first" {
		t.Fatal("double-click did not open inline editor")
	}
	s.closeInlineEdit()
	s.markEditedCell(s.editedCellIdentity(s.resultBrowser.source, 0, 1))
	s.resultGrid().Draw(terminal)
	x, y, _ = s.resultCellRect(1, 1)
	_, _, style, _ := terminal.GetContent(x, y)
	_, bg, _ := style.Decompose()
	if bg != tcell.ColorOlive {
		t.Fatalf("selected modified cell lost background: %v", bg)
	}
	// Other tabs in the same workspace observe transaction-wide markers, and
	// markers painted into an inactive tab disappear when it becomes active.
	s.initDocument()
	s.initTabs()
	s.newTab()
	s.switchTab(0)
	if !s.isEditedCell(1, 1) {
		t.Fatal("tab switching lost marker")
	}
	s.newTab()
	s.clearEditedCells()
	s.switchTab(0)
	_, bg, _ = s.table.GetCell(1, 1).Style.Decompose()
	if s.isEditedCell(1, 1) || bg == tcell.ColorOlive {
		t.Fatal("inactive tab retained transaction marker")
	}
}
