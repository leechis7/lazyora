package ui

import (
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestSchemaCompareReadsSeparateWorkspacesAndWaitsForBoth(t *testing.T) {
	left, right := shortcutScreen(t), shortcutScreen(t)
	left.workspaceID, right.workspaceID = 1, 2
	left.workspaces = &workspaceSet{screens: []*screen{left, right}, active: left}
	left.setStatus("connected", "user", "host", 1521, "svc", "service")
	right.setStatus("disconnected")
	sql := left.editor.GetText()
	left.invoke("schema_compare")
	v := left.schemaCompare
	if v == nil {
		t.Fatal("comparison unavailable with two connections")
	}
	v.form.GetFormItem(1).(*tuivory.InputField).SetText("\"MixedCase\"")
	v.form.GetFormItem(3).(*tuivory.InputField).SetText("schema_b")
	v.form.GetButton(0).InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(tuivory.Primitive) {})
	a, b := <-left.jobs, <-right.jobs
	if a.text != "MixedCase" || b.text != "SCHEMA_B" || a.schemaCompare != b.schemaCompare || a.compareSide == b.compareSide {
		t.Fatal("wrong comparison targets")
	}
	a.schemaCompare.accept(a.compareSide, oracle.SchemaStructure{Owner: a.text, Tables: map[string]oracle.SchemaTable{"A": {}, "SHARED": {"C": {Type: "NUMBER"}}}}, nil)
	if page, _ := left.pages.GetFrontPage(); page != "schema_compare" {
		t.Fatal("partial comparison was shown")
	}
	b.schemaCompare.accept(b.compareSide, oracle.SchemaStructure{Owner: b.text, Tables: map[string]oracle.SchemaTable{"B": {}, "SHARED": {"C": {Type: "VARCHAR2"}}}}, nil)
	if page, _ := left.pages.GetFrontPage(); page != "schema_compare_results" || v.list.GetItemCount() != 3 || left.editor.GetText() != sql {
		t.Fatal("missing complete comparison or SQL changed")
	}
	v.list.SetCurrentItem(2)
	if got := v.detail.GetText(false); !strings.Contains(got, "NUMBER") || !strings.Contains(got, "VARCHAR2") {
		t.Fatalf("column details missing: %s", got)
	}
}
