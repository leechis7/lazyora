package ui

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/exportdata"
	"lazyora/internal/oracle"
	"lazyora/internal/sqlfile"
)

func (s *screen) showCellDetail() {
	if s.busy || len(s.resultRows) == 0 {
		return
	}
	row, col := s.table.GetSelection()
	if row < 1 || row > len(s.resultValues) || col < 0 || col >= len(s.resultValues[row-1]) {
		return
	}
	s.cellValue = s.resultValues[row-1][col]
	s.cellFocus = s.app.GetFocus()
	s.cellInfo = tview.NewTextView().SetWrap(true).SetTextColor(s.colors().muted)
	s.cellInfo.SetBackgroundColor(s.colors().background)
	s.cellText = tview.NewTextView().SetWrap(false).SetScrollable(true).SetDynamicColors(false).SetTextColor(s.colors().text)
	s.cellText.SetBorder(true).SetTitle(fmt.Sprintf("%s · %s", s.tr("cell_detail"), tview.Escape(s.resultHeaders[col])))
	s.styleBox(s.cellText.Box)
	name := "cell.txt"
	if s.cellValue.Type == "RAW" || s.cellValue.Type == "BLOB" {
		name = "cell.bin"
	}
	form := tview.NewForm().AddInputField(s.tr("file_path"), name, 40, nil, nil)
	s.cellForm = form
	form.AddButton(s.tr("save")+" (Ctrl+S)", s.saveCellFile)
	form.AddButton(s.tr("cell_copy")+" (Ctrl+C)", s.copyCellDetail)
	form.AddButton(s.tr("cell_expand")+" (Alt+V)", s.expandCellPreview)
	form.AddButton(s.tr("close")+" (Esc)", s.closeCellDetail)
	s.styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	s.renderCellDetail()
	form.SetBorder(true)
	fit := formLayout(form)
	content := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(s.cellInfo, 3, 0, false).AddItem(s.cellText, 0, 1, true).AddItem(form, 8, 0, false)
	root := s.formPopup(content, 104, 32, func(w, h int) {
		controls := min(8, max(5, h/3))
		content.ResizeItem(form, controls, 0)
		content.ResizeItem(s.cellInfo, min(3, max(1, h/6)), 0)
		fit(w, controls)
	})
	s.pages.AddPage("cell", root, true, true)
	s.app.SetFocus(s.cellText)
}

func (s *screen) renderCellDetail() {
	cell := s.cellValue
	unit := s.tr("cell_bytes")
	if cell.Type == "CLOB" || cell.Type == "NCLOB" {
		unit = s.tr("cell_units")
	}
	size := cell.Size
	if size == 0 && !cell.Null {
		size = int64(len(cell.Text))
		if cell.Type == "RAW" || cell.Type == "BLOB" {
			size /= 2
		}
	}
	info := fmt.Sprintf("%s · %d %s", cell.Type, size, unit)
	if cell.Null {
		info += " · NULL"
	}
	if cell.Partial {
		info += "\n" + s.tr("cell_partial")
	} else {
		info += "\n" + s.tr("cell_complete")
	}
	s.cellInfo.SetText(info)
	text := cell.Text
	if cell.Type == "RAW" || cell.Type == "BLOB" {
		if cell.Null {
			text = "NULL"
		} else if data, err := hex.DecodeString(cell.Text[:min(len(cell.Text), 8192)]); err == nil {
			text = hex.Dump(data)
			if len(cell.Text) > 8192 {
				text += "\n…"
			}
		}
	} else {
		if len(text) > 256*1024 {
			end := 256 * 1024
			for end > 0 && !utf8.RuneStart(text[end]) {
				end--
			}
			text = text[:end] + "\n…"
		}
		text = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				return '�'
			}
			return r
		}, text)
	}
	s.cellText.SetText(text).ScrollToBeginning()
}

func (s *screen) copyCellDetail() {
	if s.cellValue.Partial {
		s.setStatus("cell_copy_partial")
		return
	}
	s.sendClipboard(s.cellValue.Text)
}
func (s *screen) expandCellPreview() {
	if s.busy || s.cellValue.LOBID == 0 {
		return
	}
	s.cellInfo.SetText(s.tr("browser_loading"))
	s.submit(job{action: "cell_preview", cell: s.cellValue})
}
func (s *screen) saveCellFile() {
	if s.busy || s.cellValue.Null {
		return
	}
	directory, err := os.Getwd()
	if err != nil {
		s.cellInfo.SetText(err.Error())
		return
	}
	path, err := sqlfile.Resolve(s.cellForm.GetFormItem(0).(*tview.InputField).GetText(), directory)
	if err != nil {
		s.cellInfo.SetText(err.Error())
		return
	}
	revision, err := exportdata.Inspect(path)
	if err != nil {
		s.cellInfo.SetText(err.Error())
		return
	}
	cell := s.cellValue
	write := func() {
		s.cellInfo.SetText(s.tr("cell_saving"))
		s.submit(job{action: "cell_save", cell: cell, path: path, revision: revision})
	}
	if revision.Exists {
		s.showProgramConfirm(fmt.Sprintf(s.tr("cell_overwrite"), tview.Escape(path)), write)
	} else {
		write()
	}
}
func (s *screen) closeCellDetail() {
	if s.busy {
		s.action("cancel")
		return
	}
	s.pages.RemovePage("cell")
	s.cellInfo, s.cellText, s.cellForm = nil, nil, nil
	s.cellValue = oracle.Cell{}
	if s.cellFocus != nil {
		s.app.SetFocus(s.cellFocus)
	}
	s.cellFocus = nil
}
func (s *screen) cellKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyF1 {
		s.showHelp()
		return nil
	}
	if event.Key() == tcell.KeyEscape {
		s.closeCellDetail()
		return nil
	}
	if s.busy {
		return nil
	}
	if s.buttonArrow(event) {
		return nil
	}
	switch event.Key() {
	case tcell.KeyCtrlS:
		s.saveCellFile()
		return nil
	case tcell.KeyCtrlC:
		s.copyCellDetail()
		return nil
	case tcell.KeyRune:
		if event.Modifiers()&tcell.ModAlt != 0 && (event.Rune() == 'v' || event.Rune() == 'V') {
			s.expandCellPreview()
			return nil
		}
	case tcell.KeyTab, tcell.KeyBacktab:
		index := 0
		if s.cellForm.GetFormItem(0).HasFocus() {
			index = 1
		}
		for i := 0; i < s.cellForm.GetButtonCount(); i++ {
			if s.cellForm.GetButton(i).HasFocus() {
				index = i + 2
			}
		}
		step := 1
		if event.Key() == tcell.KeyBacktab {
			step = s.cellForm.GetButtonCount() + 1
		}
		next := (index + step) % (s.cellForm.GetButtonCount() + 2)
		if next == 0 {
			s.app.SetFocus(s.cellText)
		} else {
			focusForm(s, s.cellForm, next-1)
		}
		return nil
	}
	return event
}
