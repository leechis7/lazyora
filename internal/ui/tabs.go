package ui

import (
	"fmt"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/sqlfile"
)

// Tabs share one DB connection/transaction. Executions lock tab switching and
// closing, so a result always belongs to the editor which submitted the job.
type sqlTab struct {
	script     oracle.ScriptReport
	editor     *tview.TextArea
	table      *tview.Table
	document   *sqlfile.Document
	rows       [][]string
	headers    []string
	selection  resultSelection
	status     string
	statusArgs []any
}

func (s *screen) initTabs() {
	if len(s.tabs) > 0 {
		return
	}
	s.tabs = []*sqlTab{{}}
	s.tabBar = tview.NewFlex()
	s.stashTab()
	s.refreshTabs()
	s.rebuildPanels()
}

func (s *screen) stashTab() {
	if len(s.tabs) == 0 {
		return
	}
	tab := s.tabs[s.activeTab]
	tab.script = s.scriptReport
	tab.editor, tab.table, tab.document = s.editor, s.table, s.document
	tab.rows, tab.headers, tab.selection = s.resultRows, s.resultHeaders, s.resultSelection
	tab.status, tab.statusArgs = s.statusKey, append([]any(nil), s.statusArgs...)
}

func (s *screen) switchTab(index int) {
	if s.busy || index < 0 || index >= len(s.tabs) {
		return
	}
	s.stashTab()
	s.activeTab = index
	tab := s.tabs[index]
	s.scriptReport = tab.script
	s.editor, s.table, s.document = tab.editor, tab.table, tab.document
	s.resultRows, s.resultHeaders, s.resultSelection = tab.rows, tab.headers, tab.selection
	s.setStatus(tab.status, tab.statusArgs...)
	s.rebuildPanels()
	s.refreshDocumentTitle()
	s.focusPanel("sql")
}

func (s *screen) cycleTab(step int) {
	if len(s.tabs) > 0 {
		s.switchTab((s.activeTab + step + len(s.tabs)) % len(s.tabs))
	}
}

func (s *screen) newTab() {
	if s.busy {
		return
	}
	s.stashTab()
	editor := tview.NewTextArea()
	editor.SetBorder(true)
	styleBox(editor.Box)
	editor.SetBackgroundColor(palette.field).SetTitleColor(palette.background)
	editor.SetTextStyle(fieldStyle()).SetSelectedStyle(focusStyle())
	editor.SetClipboard(s.sendClipboard, func() string { return s.clipboardText })
	editor.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseRightClick {
			s.app.SetFocus(s.editor)
			s.showContextMenu()
			return tview.MouseConsumed, nil
		}
		return action, event
	})
	table := tview.NewTable().SetFixed(1, 0).SetSelectable(true, true).SetSelectedStyle(focusStyle())
	table.SetBorder(true)
	styleBox(table.Box)
	table.SetTitle(s.tr("results_title"))
	table.SetMouseCapture(s.resultMouse)
	s.tabs = append(s.tabs, &sqlTab{editor: editor, table: table, document: &sqlfile.Document{}, status: "tab_shared"})
	s.switchTab(len(s.tabs) - 1)
	s.initDocument()
}

func (s *screen) closeTab() {
	if s.busy {
		return
	}
	s.guardDocument(func() {
		index := s.activeTab
		// Store the latest pointers before unlinking the active tab.
		s.stashTab()
		s.tabs = append(s.tabs[:index], s.tabs[index+1:]...)
		if len(s.tabs) == 0 {
			s.tabs = []*sqlTab{{editor: s.editor, table: s.table, document: &sqlfile.Document{}}}
			s.activeTab = 0
			s.document = s.tabs[0].document
			s.editor.SetText("", false)
			s.table.Clear()
			s.resultRows = nil
			s.resultHeaders = nil
			s.resultSelection = resultSelection{}
			s.scriptReport = oracle.ScriptReport{}
			s.setStatus("tab_shared")
			s.stashTab()
			s.refreshDocumentTitle()
			s.focusPanel("sql")
			return
		}
		// Avoid overwriting the destination with the removed tab's state.
		s.activeTab = min(index, len(s.tabs)-1)
		tab := s.tabs[s.activeTab]
		s.scriptReport = tab.script
		s.editor, s.table, s.document = tab.editor, tab.table, tab.document
		s.resultRows, s.resultHeaders, s.resultSelection = tab.rows, tab.headers, tab.selection
		s.setStatus(tab.status, tab.statusArgs...)
		s.rebuildPanels()
		s.refreshDocumentTitle()
		s.focusPanel("sql")
	})
}

func (s *screen) guardDocuments(next func()) {
	if len(s.tabs) == 0 {
		s.guardDocument(next)
		return
	}
	s.stashTab()
	original := s.activeTab
	originalFocus := s.app.GetFocus()
	var check func(int)
	check = func(index int) {
		if index == len(s.tabs) {
			if s.activeTab != original {
				s.switchTab(original)
			}
			if originalFocus != nil {
				s.app.SetFocus(originalFocus)
			}
			next()
			return
		}
		tab := s.tabs[index]
		if !tab.document.Dirty(tab.editor.GetText()) {
			check(index + 1)
			return
		}
		s.switchTab(index)
		s.guardDocument(func() { check(index + 1) })
	}
	check(0)
}

func (s *screen) refreshTabs() {
	if s.tabBar == nil {
		return
	}
	s.tabBar.Clear()
	shared := tview.NewTextView().SetText(s.tr("tab_session"))
	shared.SetTextColor(palette.muted).SetBackgroundColor(palette.background)
	s.tabBar.AddItem(shared, 10, 0, false)
	// Keep the active tab visible even on a narrow terminal. All tabs remain
	// reachable with previous/next; at most four labels share the row.
	start := max(0, s.activeTab-3)
	end := min(len(s.tabs), start+4)
	if len(s.tabs) > 4 {
		s.tabBar.AddItem(clickable(tview.NewButton("<").SetSelectedFunc(func() { s.cycleTab(-1) })), 3, 0, false)
	}
	for i := start; i < end; i++ {
		tab := s.tabs[i]
		doc := tab.document
		if i == s.activeTab {
			doc = s.document
		}
		name := s.tr("file_untitled")
		if doc.Path != "" {
			name = filepath.Base(doc.Path)
		}
		mark := ""
		if doc.Dirty(tab.editor.GetText()) {
			mark = "*"
		}
		label := fmt.Sprintf("%d %s%s", i+1, name, mark)
		if i == s.activeTab {
			label = "> " + label
		}
		index := i
		button := clickable(tview.NewButton(label).SetSelectedFunc(func() { s.switchTab(index) }))
		if i == s.activeTab {
			button.SetStyle(buttonFocusStyle())
		}
		s.tabBar.AddItem(button, 0, 1, false)
	}
	if len(s.tabs) > 4 {
		s.tabBar.AddItem(clickable(tview.NewButton(">").SetSelectedFunc(func() { s.cycleTab(1) })), 3, 0, false)
	}
}
