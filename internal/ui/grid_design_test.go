package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestGridVisibleStatesAndColumnSeparators(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprint(limited), func(t *testing.T) {
			s, terminal := resultScreen(t)
			s.limitedColors = limited
			s.table.SetSelectable(true, true).SetFixed(1, 0)
			s.render(oracle.Result{Columns: []string{"ID", "이름", "금액", "상태"}, Rows: [][]string{{"1", "한글", "12345.67", "NULL"}, {"2", "second", "0", ""}, {"3", "third", "-1", "ready"}}, Values: [][]oracle.Cell{{{Type: "NUMBER(10)", Text: "1"}, {Type: "VARCHAR2", Text: "한글"}, {Type: "NUMBER", Text: "12345.67"}, {Null: true, Type: "VARCHAR2"}}, {{Type: "NUMBER", Text: "2"}, {Type: "VARCHAR2", Text: "second"}, {Type: "NUMBER", Text: "0"}, {Type: "VARCHAR2", Text: ""}}, {{Type: "NUMBER", Text: "3"}, {Type: "VARCHAR2", Text: "third"}, {Type: "NUMBER", Text: "-1"}, {Type: "VARCHAR2", Text: "ready"}}}})
			s.resultSelection = resultSelection{active: true, anchorRow: 1, anchorCol: 1, endRow: 2, endCol: 2}
			s.table.Select(2, 2)
			s.highlightResults()
			s.app.SetFocus(s.table)
			grid := s.resultGrid()
			grid.SetRect(0, 0, 100, 12)
			grid.Draw(terminal)
			background := func(row, col int) tcell.Color {
				x, y, _ := s.table.GetCell(row, col).GetLastPosition()
				_, _, style, _ := terminal.GetContent(x, y)
				_, bg, _ := style.Decompose()
				return bg
			}
			active, selection, current, regular := background(2, 2), background(1, 1), background(2, 0), background(3, 0)
			if active == selection || active == current || selection == regular || current == regular {
				t.Fatalf("active/range/current/regular colors: %v %v %v %v", active, selection, current, regular)
			}
			if s.table.GetCell(1, 0).Align != tview.AlignRight {
				t.Fatal("formatted NUMBER metadata was not right aligned")
			}
			x, y, width := s.table.GetCell(1, 1).GetLastPosition()
			r, _, _, _ := terminal.GetContent(x+width, y)
			if r != '│' {
				t.Fatal("column divider missing")
			}
			row, col := s.table.CellAt(x+1, y)
			if row != 1 || col != 1 {
				t.Fatal("padded cell shifted mouse hit testing")
			}
			s.copyResults("selection")
			if !strings.Contains(s.clipboardText, "한글\t12345.67") {
				t.Fatal("copy contains padding or separators")
			}
			s.app.SetFocus(s.editor)
			grid.Draw(terminal)
			if background(2, 2) == active {
				t.Fatal("inactive grid still looks focused")
			}
			for _, width := range []int{40, 60, 120} {
				grid.SetRect(0, 0, width, 12)
				grid.Draw(terminal)
				s.table.Select(3, 3)
				grid.Draw(terminal)
				x, y, _ := s.table.GetCell(3, 3).GetLastPosition()
				r, c := s.table.CellAt(x, y)
				if r != 3 || c != 3 {
					t.Fatal("resize/horizontal scroll changed data mapping")
				}
			}
		})
	}
}

// Include terminal drawing, not just selection bookkeeping, in this measure.
func BenchmarkGridDraw1000Rows(b *testing.B) {
	s := &screen{table: tview.NewTable().SetSelectable(true, true).SetFixed(1, 0)}
	data := oracle.Result{Columns: strings.Fields(strings.Repeat("COLUMN ", 20))}
	for i := 0; i < 1000; i++ {
		row := make([]string, 20)
		for j := range row {
			row[j] = "한글 [red] 123"
		}
		data.Rows = append(data.Rows, row)
	}
	s.render(data)
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		b.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(120, 28)
	grid := s.resultGrid()
	grid.SetRect(0, 0, 120, 28)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.table.Select(i%1000+1, i%20)
		grid.Draw(terminal)
	}
}
