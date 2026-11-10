package ui

import (
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestObjectWorkspaceUsesLazyDataTabAndPreservesMainResult(t *testing.T) {
	s := demoBrowserScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
	editor := s.editor.GetText()
	s.app.SetFocus(s.browser)
	s.openObjectDetails(oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_ROWS", Kind: "TABLE"}, true)
	w := s.objectWorkspace
	if w == nil || w.split == nil || w.dataGrid != nil {
		t.Fatal("workspace did not defer data loading")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'v', tuivory.ModAlt))
	if w.section != "data" || w.dataGrid == nil || len(w.data.Values) != 2 {
		t.Fatal("data preview missing")
	}
	connectionKey(s, tuivory.KeyRight)
	connectionKey(s, tuivory.KeyEnter)
	if w.section != "ddl" || !strings.Contains(w.ddl.GetText(false), "CREATE TABLE") {
		t.Fatal("DDL did not open in the same workspace")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "main" || s.editor.GetText() != editor || s.resultRows[0][0] != "keep" {
		t.Fatal("object workspace changed the SQL editor or result")
	}
}
