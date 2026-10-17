package ui

import "github.com/rivo/tview"

type panelOffsets struct{ editorRow, editorCol, resultRow, resultCol int }

func (s *screen) restorePanelOffsets() {
	s.editor.SetOffset(s.splitOffsets.editorRow, s.splitOffsets.editorCol)
	s.table.SetOffset(s.splitOffsets.resultRow, s.splitOffsets.resultCol)
}

func (s *screen) rebuildPanels() {
	if s.mainLayout == nil || s.queryLayout == nil {
		return
	}
	s.mainLayout.Clear()
	s.queryLayout.Clear()
	s.editor.SetRect(0, 0, 0, 0)
	s.table.SetRect(0, 0, 0, 0)
	if s.browser != nil {
		s.hideBrowserRect()
	}
	switch s.maximized {
	case "sql":
		s.queryLayout.AddItem(s.editor, 0, 1, true)
	case "results":
		s.queryLayout.AddItem(s.table, 0, 1, true)
	default:
		s.queryLayout.AddItem(s.editor, 0, 2, true).AddItem(s.table, 0, 3, false)
		if s.browser != nil && !s.browserHidden {
			s.mainLayout.AddItem(s.browserPanel, 0, 1, false)
		}
	}
	if !s.fullscreen {
		s.queryLayout.AddItem(s.status, 2, 0, false)
	} else {
		s.status.SetRect(0, 0, 0, 0)
	}
	s.mainLayout.AddItem(s.queryLayout, 0, 3, true)
}

func (s *screen) toggleMaximize()   { s.togglePanelMode(false) }
func (s *screen) toggleFullscreen() { s.togglePanelMode(true) }

func (s *screen) togglePanelMode(fullscreen bool) {
	if s.maximized != "" && s.fullscreen == fullscreen {
		s.maximized = ""
		s.fullscreen = false
		defer s.restorePanelOffsets()
	} else {
		if s.maximized == "" {
			s.splitOffsets.editorRow, s.splitOffsets.editorCol = s.editor.GetOffset()
			s.splitOffsets.resultRow, s.splitOffsets.resultCol = s.table.GetOffset()
			s.maximized = "sql"
			if s.table.HasFocus() {
				s.maximized = "results"
			}
		}
		s.fullscreen = fullscreen
	}
	s.rebuildPanels()
	if s.maximized == "results" {
		s.app.SetFocus(s.table)
	} else if s.maximized == "sql" {
		s.app.SetFocus(s.editor)
	}
}

func (s *screen) focusPanel(panel string) {
	if s.maximized != "" {
		s.maximized = panel
		s.rebuildPanels()
	}
	if panel == "results" {
		s.app.SetFocus(s.table)
	} else {
		s.app.SetFocus(s.editor)
	}
}

func (s *screen) cyclePanel(reverse bool) {
	if s.maximized != "" {
		if s.editor.HasFocus() {
			s.focusPanel("results")
		} else {
			s.focusPanel("sql")
		}
		return
	}
	panels := []tview.Primitive{s.editor, s.table}
	if s.browser != nil && !s.browserHidden {
		panels = append(panels, s.browserUsers, s.browserTypes, s.browser)
	}
	step := 1
	if reverse {
		step = len(panels) - 1
	}
	for i, panel := range panels {
		if panel.HasFocus() {
			s.app.SetFocus(panels[(i+step)%len(panels)])
			return
		}
	}
	s.app.SetFocus(s.editor)
}

// Only the selected panel occupies the terminal in fullscreen. Temporary pages
// retain normal navigation; closing them automatically restores fullscreen.
func (s *screen) syncChrome() {
	if s.rootLayout == nil {
		return
	}
	page, _ := s.pages.GetFrontPage()
	hidden := s.fullscreen && s.maximized != "" && page == "main"
	if hidden == s.chromeHidden {
		return
	}
	s.chromeHidden = hidden
	s.rootLayout.Clear()
	if hidden {
		s.menuLayout.SetRect(0, 0, 0, 0)
		s.footer.SetRect(0, 0, 0, 0)
		s.footerBar.SetRect(0, 0, 0, 0)
		s.rootLayout.AddItem(s.pages, 0, 1, true)
	} else {
		s.rootLayout.AddItem(s.menuLayout, 1, 0, false).AddItem(s.pages, 0, 1, true).
			AddItem(s.footer, 1, 0, false).AddItem(s.footerBar, 1, 0, false)
	}
}

func (s *screen) togglePanelFullscreen(panel string) {
	if s.fullscreen && s.maximized != "" && s.maximized != panel {
		s.focusPanel(panel)
		return
	}
	s.focusPanel(panel)
	s.toggleFullscreen()
}

func (s *screen) toggleQueryResults() {
	if s.editor.HasFocus() {
		s.focusPanel("results")
	} else {
		s.focusPanel("sql")
	}
}

func (s *screen) hideBrowserRect() {
	s.browser.SetRect(0, 0, 0, 0)
	if s.browserPanel != nil {
		s.browserPanel.SetRect(0, 0, 0, 0)
		s.browserForm.SetRect(0, 0, 0, 0)
		s.browserUsers.SetRect(0, 0, 0, 0)
		s.browserTypes.SetRect(0, 0, 0, 0)
	}
}
