package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

func sqlSummary(sql string, limit int) string {
	text := []rune(strings.Join(strings.Fields(sql), " "))
	if len(text) > limit {
		return string(text[:limit]) + "…"
	}
	return string(text)
}

func (s *screen) exitMessage(key string) string {
	message := s.tr(key)
	if badge := s.connectionBadge(); badge != "" {
		message = tview.Escape(badge) + "\n\n" + message
	}
	if key != "exit_pending" && key != "workspace_exit_pending" {
		return message
	}
	records := s.pendingWork.Statements
	message += "\n\n" + fmt.Sprintf(s.tr("pending_count"), len(records)+s.pendingWork.Omitted)
	if len(records) > 0 {
		message += "\n" + s.tr("pending_latest") + ": " + tview.Escape(sqlSummary(records[len(records)-1].SQL, 80))
	}
	return message + "\n" + s.tr("pending_view_hint")
}

func (s *screen) showPendingWork() {
	if s.busy || s.pendingTable != nil {
		return
	}
	s.pendingFocus = s.app.GetFocus()
	s.pendingSQL = s.newTextView().SetWrap(true).SetWordWrap(true).SetScrollable(true)
	s.pendingSQL.SetBorder(true).SetTitle(s.tr("pending_sql"))
	s.styleBox(s.pendingSQL.Box)
	s.pendingSQL.SetTextColor(s.colors().text)
	s.pendingTable = s.newDataTable().SetFixed(1, 0).SetSelectable(true, false).SetSelectedStyle(s.focusStyle())
	s.pendingTable.SetBorder(true).SetTitle(s.tr("pending_work"))
	s.styleBox(s.pendingTable.Box)
	for col, key := range []string{"pending_number", "pending_time", "pending_outcome", "pending_rows", "pending_sql"} {
		s.pendingTable.SetCell(0, col, tview.NewTableCell(s.tr(key)).SetSelectable(false).SetTextColor(s.colors().accent))
	}
	for i, record := range s.pendingWork.Statements {
		outcome := s.tr("pending_success")
		if record.Error != "" {
			outcome = s.tr("pending_error")
		}
		rows := "—"
		if record.Affected >= 0 {
			rows = fmt.Sprint(record.Affected)
		}
		for col, text := range []string{fmt.Sprint(i + 1 + s.pendingWork.Omitted), record.At.Format("15:04:05"), outcome, rows, sqlSummary(record.SQL, 80)} {
			s.pendingTable.SetCell(i+1, col, tview.NewTableCell(tview.Escape(text)).SetTextColor(s.colors().text))
		}
	}
	s.pendingTable.SetSelectionChangedFunc(func(row, col int) { s.selectPendingStatement(row) })
	info := s.newTextView(textHint).SetWrap(true).SetWordWrap(true).SetTextColor(s.colors().muted)
	info.SetBackgroundColor(s.colors().background)
	text := fmt.Sprintf(s.tr("pending_count"), len(s.pendingWork.Statements)+s.pendingWork.Omitted) + "\n" + s.tr("pending_note")
	if s.pendingWork.Omitted > 0 {
		text += "\n" + fmt.Sprintf(s.tr("pending_omitted"), s.pendingWork.Omitted)
	}
	info.SetText(text)
	close := s.clickable(tview.NewButton(s.tr("back") + " (Esc)").SetSelectedFunc(s.closePendingWork))
	root := s.newFlex().SetDirection(tview.FlexRow).AddItem(info, 5, 0, false).
		AddItem(s.pendingTable, 0, 2, true).AddItem(s.pendingSQL, 0, 3, false).AddItem(close, 1, 0, false)
	s.pages.AddPage("pending", root, true, true)
	if len(s.pendingWork.Statements) > 0 {
		s.pendingTable.Select(1, 0)
		s.app.SetFocus(s.pendingTable)
	} else {
		s.pendingSQL.SetText(s.tr("pending_empty"))
		s.app.SetFocus(close)
	}
}

func (s *screen) selectPendingStatement(row int) {
	if row < 1 || row > len(s.pendingWork.Statements) {
		return
	}
	record := s.pendingWork.Statements[row-1]
	text := record.SQL
	if record.Truncated {
		text += "\n\n" + s.tr("pending_truncated")
	}
	if record.Error != "" {
		text += "\n\n" + s.tr("pending_error") + ": " + record.Error
	}
	s.pendingSQL.SetText(text).ScrollToBeginning()
}

func (s *screen) closePendingWork() {
	s.pages.RemovePage("pending")
	s.pendingTable = nil
	s.pendingSQL = nil
	if s.exitDialog != nil {
		s.exitDialog.SetFocus(s.exitCancelIndex())
		s.app.SetFocus(s.exitDialog)
	} else if s.pendingFocus != nil {
		s.app.SetFocus(s.pendingFocus)
	}
	s.pendingFocus = nil
}
