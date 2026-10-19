package ui

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/exportdata"
	"lazyora/internal/oracle"
	"lazyora/internal/sqlfile"
)

var exportFormats = []string{"csv", "tsv", "xlsx", "sql"}

func csvColumns(columns []string) string {
	var out strings.Builder
	w := csv.NewWriter(&out)
	_ = w.Write(columns)
	w.Flush()
	return strings.TrimSuffix(out.String(), "\n")
}

func (s *screen) exportDataset(scope int) (exportdata.Dataset, error) {
	data := exportdata.Dataset{}
	r1, c1, r2, c2 := 1, 0, len(s.resultRows), len(s.resultHeaders)-1
	if scope == 1 {
		if !s.resultSelection.active {
			return data, fmt.Errorf("select a result range first")
		}
		r1, c1, r2, c2 = s.resultBounds()
	} else if scope == 2 {
		r1, c1 = s.table.GetSelection()
		r2 = r1
		c1 = 0
	}
	if c1 < 0 || c2 >= len(s.resultHeaders) || r1 < 1 || r2 > len(s.resultRows) || r1 > r2+1 {
		return data, fmt.Errorf("invalid result selection")
	}
	data.Columns = append([]string(nil), s.resultHeaders[c1:c2+1]...)
	for row := r1; row <= r2; row++ {
		var values []oracle.Cell
		for col := c1; col <= c2; col++ {
			cell := oracle.Cell{Text: s.resultRows[row-1][col], Type: "VARCHAR2"}
			if row-1 < len(s.resultValues) && col < len(s.resultValues[row-1]) {
				cell = s.resultValues[row-1][col]
			}
			values = append(values, cell)
		}
		data.Rows = append(data.Rows, values)
	}
	return data, nil
}

func (s *screen) showExport() {
	if s.busy || s.exportForm != nil || len(s.resultHeaders) == 0 {
		return
	}
	s.exportFocus = s.app.GetFocus()
	directory, err := os.Getwd()
	if err != nil {
		s.setStatus("error", err)
		return
	}
	form := tview.NewForm().SetItemPadding(0)
	format := s.newChoice(s.tr("export_format"), []string{"CSV", "TSV", "Excel (.xlsx)", "Oracle INSERT (.sql)"}, 0, func(_ string, index int) {
		if form.GetFormItemCount() < 2 {
			return
		}
		field := form.GetFormItem(1).(*tview.InputField)
		path := field.GetText()
		ext := strings.ToLower(filepath.Ext(path))
		for _, known := range exportFormats {
			if ext == "."+known {
				field.SetText(strings.TrimSuffix(path, filepath.Ext(path)) + "." + exportFormats[index])
				break
			}
		}
	})
	form.AddFormItem(format).AddInputField(s.tr("file_path"), filepath.Join(directory, "results.csv"), 0, nil, nil)
	form.AddFormItem(s.newChoice(s.tr("export_scope"), []string{s.tr("export_all"), s.tr("export_selection"), s.tr("export_row")}, 0, func(_ string, index int) {
		if form.GetFormItemCount() < 9 {
			return
		}
		if data, err := s.exportDataset(index); err == nil {
			form.GetFormItem(8).(*tview.InputField).SetText(csvColumns(data.Columns))
		}
	}))
	form.AddCheckbox(s.tr("export_headers"), true, nil).AddCheckbox("UTF-8 BOM (CSV/TSV)", true, nil)
	form.AddInputField(s.tr("export_null"), `\N`, 20, nil, nil)
	form.AddInputField(s.tr("export_schema"), "", 30, nil, nil)
	form.AddInputField(s.tr("export_table"), "", 30, nil, nil)
	form.AddInputField(s.tr("export_columns"), csvColumns(s.resultHeaders), 0, nil, nil)
	info := tview.NewTextView().SetWrap(true).SetText(fmt.Sprintf(s.tr("export_hint"), len(s.resultRows), oracle.MaxRows))
	if s.resultTruncated {
		info.SetText(info.GetText(false) + "\n" + s.tr("export_truncated"))
	}
	info.SetTextColor(palette.muted).SetBackgroundColor(palette.background)
	s.exportInfo = info
	s.exportForm = form
	s.exportSubmit = func() {
		if s.busy {
			return
		}
		index, _ := format.GetCurrentOption()
		scope, _ := form.GetFormItem(2).(*choiceField).GetCurrentOption()
		data, err := s.exportDataset(scope)
		if err != nil {
			info.SetText(err.Error())
			return
		}
		path, err := sqlfile.Resolve(form.GetFormItem(1).(*tview.InputField).GetText(), directory)
		if err != nil {
			info.SetText(err.Error())
			return
		}
		options := exportdata.Options{Format: exportFormats[index], Headers: form.GetFormItem(3).(*tview.Checkbox).IsChecked(), BOM: form.GetFormItem(4).(*tview.Checkbox).IsChecked(), NullText: form.GetFormItem(5).(*tview.InputField).GetText(), Schema: form.GetFormItem(6).(*tview.InputField).GetText(), Table: form.GetFormItem(7).(*tview.InputField).GetText()}
		if options.Format == "sql" {
			if strings.TrimSpace(options.Table) == "" {
				info.SetText(s.tr("export_need_table"))
				return
			}
			reader := csv.NewReader(strings.NewReader(form.GetFormItem(8).(*tview.InputField).GetText()))
			options.Columns, err = reader.Read()
			if err != nil || len(options.Columns) != len(data.Columns) {
				info.SetText(s.tr("export_need_columns"))
				return
			}
			if _, err = reader.Read(); err != io.EOF {
				info.SetText(s.tr("export_need_columns"))
				return
			}
		}
		revision, err := exportdata.Inspect(path)
		if err != nil {
			info.SetText(err.Error())
			return
		}
		write := func() {
			s.submit(job{action: "export", path: path, dataset: data, exportOptions: options, revision: revision})
		}
		if revision.Exists {
			s.showProgramConfirm(fmt.Sprintf(s.tr("export_overwrite"), len(data.Rows), tview.Escape(path)), write)
		} else {
			write()
		}
	}
	form.AddButton(s.tr("export"), s.exportSubmit).AddButton(s.tr("cancel"), s.closeExport)
	form.SetBorder(true).SetTitle(s.tr("export"))
	styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		clickable(form.GetButton(i))
	}
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(info, 5, 0, false).AddItem(form, 0, 1, true)
	s.pages.AddPage("export", root, true, true)
	s.app.SetFocus(form)
}

func (s *screen) closeExport() {
	if s.busy {
		return
	}
	if page, _ := s.pages.GetFrontPage(); page == "help" {
		s.closeHelp()
	}
	s.pages.RemovePage("export")
	s.exportForm = nil
	s.exportInfo = nil
	s.exportSubmit = nil
	if s.exportFocus != nil {
		s.app.SetFocus(s.exportFocus)
	}
}

func (s *screen) exportKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if s.busy {
			if s.cancel != nil {
				s.cancel()
			}
		} else {
			s.closeExport()
		}
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyCtrlS:
		s.exportSubmit()
		return nil
	case tcell.KeyEnter:
		if event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			s.exportSubmit()
			return nil
		}
	}
	return event
}
