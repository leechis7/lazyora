package ui

import (
	"encoding/csv"
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestGridDecorationPreservesRawCopyHitTestingAndRowNumbers(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprint(limited), func(t *testing.T) {
			s, terminal := resultScreen(t)
			s.limitedColors = limited
			long := strings.Repeat("한글👩‍💻e\u0301", 100) + "\t\n[red]"
			s.render(oracle.Result{Columns: []string{"N", "TEXT", "EMPTY"}, Rows: [][]string{{"123456789012345678901234567890", long, "NULL"}}, Values: [][]oracle.Cell{{{Text: "123456789012345678901234567890", Type: "NUMBER"}, {Text: long, Type: "VARCHAR2"}, {Type: "VARCHAR2", Null: true}}}})
			grid := s.resultGrid()
			for _, width := range []int{40, 80, 120} {
				grid.SetRect(0, 0, width, 12)
				grid.Draw(terminal)
			}
			if s.table.GetCell(1, 0).Align != tview.AlignRight || strings.TrimSpace(s.table.GetCell(1, 2).Text) != "∅ NULL" || !strings.Contains(s.table.GetCell(1, 1).Text, "…") {
				t.Fatal("typed preview/alignment missing")
			}
			x, y, _ := s.table.GetCell(1, 0).GetLastPosition()
			row, col := s.table.CellAt(x, y)
			if row != 1 || col != 0 {
				t.Fatal("gutter shifted data hit testing")
			}
			ix, _, _, _ := s.table.GetInnerRect()
			r, _, _, _ := terminal.GetContent(ix-2, y)
			if r != '1' {
				t.Fatalf("row number missing: %q", r)
			}
			s.resultMouse(tview.MouseLeftDown, tcell.NewEventMouse(ix-2, y, tcell.Button1, 0))
			s.copyResults("selection")
			reader := csv.NewReader(strings.NewReader(s.clipboardText))
			reader.Comma = '\t'
			record, err := reader.Read()
			if err != nil || len(record) != 3 || record[1] != long || record[2] != "NULL" {
				t.Fatalf("gutter selection copied decoration/truncated data: %v %v", record, err)
			}
			s.table.Select(1, 1)
			before := s.grid.widths[1]
			s.invoke("grid_widen")
			if s.grid.widths[1] != before+4 {
				t.Fatal("column width menu not connected")
			}
			s.resizeResultColumn(0)
			if s.grid.widths[1] > 40 || s.resultRows[0][1] != long || len(s.jobs) != 0 {
				t.Fatal("auto-fit changed data or executed SQL")
			}
		})
	}
}

func TestGridScrolledGutterTracksActualDataAndTabWidths(t *testing.T) {
	s, terminal := resultScreen(t)
	rows := make([][]string, 1000)
	for i := range rows {
		rows[i] = []string{fmt.Sprint(i), "text\tline\n"}
	}
	s.render(oracle.Result{Columns: []string{"ID", "TEXT"}, Rows: rows})
	s.table.Select(501, 1).SetOffset(500, 0)
	grid := s.resultGrid()
	grid.SetRect(0, 0, 80, 12)
	grid.Draw(terminal)
	_, y, _ := s.table.GetCell(501, 1).GetLastPosition()
	x, _, _, _ := s.table.GetInnerRect()
	var number strings.Builder
	for px := x - 5; px < x-1; px++ {
		r, _, _, _ := terminal.GetContent(px, y)
		number.WriteRune(r)
	}
	if strings.TrimSpace(number.String()) != "501" {
		t.Fatalf("scrolled row gutter: %q", number.String())
	}
	if !strings.Contains(s.table.GetCell(501, 1).Text, "⇥") || !strings.Contains(s.table.GetCell(501, 1).Text, "↵") {
		t.Fatal("line/tab preview missing")
	}
	s.resizeResultColumn(12)
	width := s.grid.widths[1]
	s.initDocument()
	s.initTabs()
	s.newTab()
	s.switchTab(0)
	if s.grid.widths[1] != width {
		t.Fatal("switching tabs lost manual widths")
	}
}

func BenchmarkGridMoveAcross1000Rows(b *testing.B) {
	s := &screen{table: tview.NewTable()}
	rows := make([][]string, 1000)
	for i := range rows {
		rows[i] = make([]string, 20)
		for col := range rows[i] {
			rows[i][col] = "한글 [red] 123"
		}
	}
	s.render(oracle.Result{Columns: strings.Fields(strings.Repeat("C ", 20)), Rows: rows})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.table.Select(i%1000+1, i%20)
	}
}
