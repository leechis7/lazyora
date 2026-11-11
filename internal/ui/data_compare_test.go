package ui

import (
	"lazyora/internal/oracle"
	"slices"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestDataCompareUsesBothConnectionsAndCancelsBothReads(t *testing.T) {
	left, right := shortcutScreen(t), shortcutScreen(t)
	left.workspaceID, right.workspaceID = 1, 2
	left.workspaces = &workspaceSet{screens: []*screen{left, right}, active: left}
	left.invoke("data_compare")
	view := left.schemaCompare
	if view == nil {
		t.Fatal("table data comparison is unavailable")
	}
	form := view.form
	form.GetFormItem(1).(*tuivory.InputField).SetText("APP.LIVE")
	form.GetFormItem(3).(*tuivory.InputField).SetText("COPY.REPLICA")
	form.GetFormItem(4).(*tuivory.InputField).SetText(`id, "Region,Code"`)
	start := func() (job, job) {
		t.Helper()
		form.GetButton(0).InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(tuivory.Primitive) {})
		if len(left.jobs) != 1 || len(right.jobs) != 1 {
			t.Fatal("expected one read per workspace")
		}
		a, b := <-left.jobs, <-right.jobs
		if a.action != "data_compare" || b.action != "data_compare" || a.text != "APP.LIVE" || b.text != "COPY.REPLICA" ||
			!slices.Equal(a.compareKeys, []string{"ID", "Region,Code"}) || !slices.Equal(a.compareKeys, b.compareKeys) ||
			a.schemaCompare != b.schemaCompare {
			t.Fatalf("wrong read dispatch: %+v %+v", a, b)
		}
		return a, b
	}
	a, b := start()
	left.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if a.ctx.Err() == nil || b.ctx.Err() == nil {
		t.Fatal("cancel did not signal both workspaces")
	}
	a.schemaCompare.acceptData(a.compareSide, oracle.TableData{}, a.ctx.Err())
	b.schemaCompare.acceptData(b.compareSide, oracle.TableData{}, b.ctx.Err())
	if left.schemaCompare.run != nil || !strings.Contains(view.info.GetText(false), left.tr("schema_compare_cancelled")) {
		t.Fatal("cancel was not shown")
	}
	left.busy, right.busy = false, false
	a, b = start()
	defer left.cancel()
	defer right.cancel()
	data := oracle.TableData{Columns: []oracle.Column{{Name: "ID", DataType: "NUMBER"}, {Name: "Region,Code", DataType: "VARCHAR2"}, {Name: "VALUE", DataType: "VARCHAR2"}}, Key: []int{0, 1}}
	l, r := data, data
	l.Rows = map[string][]oracle.DataValue{"shared": {{Text: "1"}, {Text: "KR"}, {Null: true}}}
	r.Rows = map[string][]oracle.DataValue{"shared": {{Text: "1"}, {Text: "KR"}, {Text: "NULL"}}}
	a.schemaCompare.acceptData(a.compareSide, l, nil)
	if page, _ := left.pages.GetFrontPage(); page != "schema_compare" {
		t.Fatal("a partial comparison was shown")
	}
	b.schemaCompare.acceptData(b.compareSide, r, nil)
	if page, _ := left.pages.GetFrontPage(); page != "schema_compare_results" || view.list.GetItemCount() != 1 {
		t.Fatal("completed comparison did not show a changed row")
	}
	detail := view.detail.GetText(false)
	if !strings.Contains(detail, "VALUE (VARCHAR2)") || !strings.Contains(detail, ": NULL") || !strings.Contains(detail, `: "NULL"`) {
		t.Fatalf("detail does not distinguish the two values: %s", detail)
	}
}
