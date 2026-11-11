package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"strconv"
	"strings"
)

func (s *screen) renderDataCompare(tables [2]oracle.TableData) {
	comparison, err := oracle.CompareTableData(tables[0], tables[1])
	if err != nil {
		s.schemaCompare.info.SetText(s.tr("data_compare_incompatible") + "\n" + err.Error())
		return
	}
	columns, key := tables[0].Columns, tables[0].Key
	kind := func(change oracle.DataDifference) string {
		switch {
		case change.Right == nil:
			return s.tr("schema_compare_left_only")
		case change.Left == nil:
			return s.tr("schema_compare_right_only")
		default:
			return s.tr("data_compare_changed")
		}
	}
	cell := func(row []oracle.DataValue, index int) string {
		if row == nil {
			return s.tr("schema_compare_missing")
		}
		value := row[index]
		if value.Null {
			return "NULL"
		}
		return strconv.Quote(value.Text)
	}
	keyText := func(change oracle.DataDifference, preview bool) string {
		row := change.Left
		if row == nil {
			row = change.Right
		}
		parts := make([]string, 0, len(key))
		for _, i := range key {
			text := cell(row, i)
			if preview {
				text = gridPreview(text)
			}
			parts = append(parts, columns[i].Name+"="+text)
		}
		return strings.Join(parts, ", ")
	}
	labels := make([]string, len(comparison.Rows))
	for i, change := range comparison.Rows {
		labels[i] = kind(change) + " · " + gridPreview(keyText(change, true))
	}
	summary := fmt.Sprintf(s.tr("data_compare_summary"), oracle.QualifiedObject(tables[0].Object), len(tables[0].Rows), oracle.QualifiedObject(tables[1].Object), len(tables[1].Rows), comparison.LeftOnly, comparison.RightOnly, comparison.Changed)
	s.showComparisonResults("data_compare", "data_compare_equal", summary, labels, func(index int) string {
		change := comparison.Rows[index]
		var text strings.Builder
		fmt.Fprintf(&text, "%s\n%s\n", kind(change), keyText(change, false))
		add := func(i int) {
			col := columns[i]
			fmt.Fprintf(&text, "\n%s (%s)\n%s: %s\n%s: %s\n", col.Name, col.DataType,
				s.tr("schema_compare_left"), cell(change.Left, i), s.tr("schema_compare_right"), cell(change.Right, i))
		}
		if change.Left == nil || change.Right == nil {
			for i := range columns {
				add(i)
			}
		} else {
			for _, i := range change.Columns {
				add(i)
			}
		}
		return text.String()
	})
}
