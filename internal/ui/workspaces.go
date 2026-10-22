package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

const maxWorkspaces = 8

// Every screen owns its widgets, SQL tabs and worker. Only the terminal and
// application event loop are shared; Oracle sessions never cross this boundary.
type workspaceSet struct {
	app        *tview.Application
	screens    []*screen
	active     *screen
	factory    func(settings.Config) *screen
	sequence   int
	closingAll bool
	retired    []<-chan struct{}
}

func (m *workspaceSet) adopt(s *screen) {
	m.sequence++
	s.workspaceID = m.sequence
	s.workspaces = m
	m.screens = append(m.screens, s)
	m.refresh()
}
func (m *workspaceSet) refresh() {
	for _, s := range m.screens {
		s.refreshConnectionTitles()
		s.rebuildChrome()
	}
}
func (m *workspaceSet) activate(s *screen) bool {
	if s == nil || s.workerClosed {
		return false
	}
	if current := m.active; current != nil && !current.workerClosed {
		if !current.canSwitchWorkspace() {
			current.setStatus("workspace_busy")
			return false
		}
		current.workspaceFocus = m.app.GetFocus()
	}
	m.active = s
	if s.installHandlers != nil {
		s.installHandlers()
	}
	m.app.SetRoot(s.rootLayout, true)
	focus := s.workspaceFocus
	if focus == nil {
		focus = s.editor
	}
	m.app.SetFocus(focus)
	s.refreshConnectionTitles()
	return true
}
func (m *workspaceSet) addNew() {
	current := m.active
	if current == nil || !current.canSwitchWorkspace() {
		return
	}
	if len(m.screens) >= maxWorkspaces {
		current.setStatus("workspace_limit", maxWorkspaces)
		return
	}
	current.workspaceFocus = m.app.GetFocus()
	next := m.factory(current.prefs)
	m.active = nil // The constructor installed its focus; preserve the previous one.
	m.adopt(next)
	m.activate(next)
	if next.connectForm == nil {
		next.connectionForm()
	}
	// Factory opens the connection popup. Restore its intended focus after activate.
	if next.connectForm != nil {
		m.app.SetFocus(next.connectForm)
	}
}
func (m *workspaceSet) requestQuit(s *screen) {
	if m.closingAll || !s.canExitWorkspace() {
		return
	}
	m.closingAll = true
	s.requestQuitLocal()
}
func (s *screen) canExitWorkspace() bool {
	return s.exitDialog == nil && s.reconnectDialog == nil && s.programModal == nil && !s.quitting && s.documentModal == nil && s.fileDialog == nil && s.bindForm == nil && s.exportForm == nil && !s.vaultBusy
}
func (m *workspaceSet) closeOne(s *screen) {
	if !s.canExitWorkspace() {
		return
	}
	if m.active != s && !m.activate(s) {
		return
	}
	m.closingAll = false
	s.requestQuitLocal()
}
func (m *workspaceSet) finished(s *screen) {
	index := -1
	for i, item := range m.screens {
		if item == s {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	s.stopWorker()
	m.retired = append(m.retired, s.done)
	m.screens = append(m.screens[:index], m.screens[index+1:]...)
	if len(m.screens) == 0 {
		m.active = nil
		m.app.Stop()
		return
	}
	m.active = nil
	next := m.screens[min(index, len(m.screens)-1)]
	m.activate(next)
	m.refresh()
	if m.closingAll {
		next.requestQuitLocal()
	}
}
func (s *screen) stopWorker() {
	if s.workerClosed {
		return
	}
	s.workerClosed = true
	if s.cancel != nil {
		s.cancel()
	}
	s.lockWorkspacePasswords()
	close(s.jobs)
}
func (m *workspaceSet) shutdown() {
	for _, s := range m.screens {
		if s.vaultWorkDone != nil {
			<-s.vaultWorkDone
		}
		s.stopWorker()
	}
	for _, s := range m.screens {
		<-s.done
	}
	for _, done := range m.retired {
		<-done
	}
}
func (s *screen) sharePreferences() {
	if s.workspaces == nil {
		return
	}
	for _, other := range s.workspaces.screens {
		if other == s {
			continue
		}
		old := other.prefs.Theme
		other.prefs = s.prefs
		other.vim = vimState{}
		other.applyTheme(old)
		other.refreshLabels()
		other.refreshConnectionTitles()
	}
}
func safeIdentity(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	r := []rune(strings.TrimSpace(text))
	if len(r) > 120 {
		r = append(r[:119], '…')
	}
	return string(r)
}
func (s *screen) connectionBadge() string {
	p := s.connectionIdentity
	if s.workspaces == nil && p.User == "" && p.Host == "" {
		return ""
	}
	if !s.connected {
		if s.workspaces != nil {
			return fmt.Sprintf("#%d %s", s.workspaceID, s.tr("workspace_disconnected"))
		}
		return ""
	}
	target := safeIdentity(p.User) + "@" + safeIdentity(p.Host) + ":" + fmt.Sprint(p.Port) + "/" + safeIdentity(p.Name)
	if p.Alias != "" {
		target = safeIdentity(p.Alias) + " · " + target
	}
	if p.Environment != "" {
		target = "[" + strings.ToUpper(p.Environment) + "] " + target
	}
	if s.pendingChanges {
		target += " [TX]"
	}
	if s.transactionUncertain {
		target += " !"
	}
	return target
}
func (s *screen) refreshConnectionTitles() {
	if s.editor == nil {
		return
	}
	s.refreshDocumentTitle()
	if s.sourceEditor != nil {
		s.refreshSourceTitle()
	}
	if s.connectionHeader != nil {
		s.connectionHeader.SetText(fmt.Sprintf("#%d · %s  | Alt+J %s", s.workspaceID, s.connectionBadge(), s.tr("workspaces")))
		color := s.colors().accent
		if s.connectionIdentity.Environment == "prod" {
			color = s.colors().warning
		}
		s.connectionHeader.SetTextColor(color).SetBackgroundColor(s.colors().surface)
	}
	if s.table != nil {
		s.refreshResultPosition()
	}
}

func (s *screen) showWorkspaces() {
	if !s.canSwitchWorkspace() || s.workspaceList != nil {
		return
	}
	s.workspaceFocusBefore = s.app.GetFocus()
	list := tview.NewList().ShowSecondaryText(true).SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetSelectedStyle(s.focusStyle())
	s.workspaceList = list
	s.styleList(list)
	list.SetBackgroundColor(s.colors().surface)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface)).SetSecondaryTextStyle(tcell.StyleDefault.Foreground(s.colors().muted).Background(s.colors().surface))
	list.SetBorder(true).SetTitle(s.tr("workspaces"))
	snapshot := append([]*screen(nil), s.workspaces.screens...)
	selected := 0
	for i, item := range snapshot {
		label := fmt.Sprintf("#%d · %s", item.workspaceID, item.connectionBadge())
		if item == s {
			label = "› " + label
			selected = i
		}
		note := s.tr("workspace_shared_tabs")
		if item.pendingChanges {
			note = s.tr("workspace_pending")
		}
		if item.transactionUncertain {
			note = s.tr("workspace_uncertain")
		}
		list.AddItem(tview.Escape(label), note, 0, nil)
	}
	list.SetCurrentItem(selected)
	activate := func() {
		index := list.GetCurrentItem()
		if index < 0 || index >= len(snapshot) {
			return
		}
		s.closeWorkspaces()
		s.workspaces.activate(snapshot[index])
	}
	closeSelected := func() {
		index := list.GetCurrentItem()
		if index < 0 || index >= len(snapshot) {
			return
		}
		s.closeWorkspaces()
		s.workspaces.closeOne(snapshot[index])
	}
	list.SetSelectedFunc(func(int, string, string, rune) { activate() })
	controls := tview.NewForm().AddButton(s.tr("workspace_activate"), activate).
		AddButton(s.tr("workspace_new"), func() { s.closeWorkspaces(); s.workspaces.addNew() }).
		AddButton(s.tr("workspace_close"), closeSelected).AddButton(s.tr("cancel"), s.closeWorkspaces)
	s.styleForm(controls)
	for i := 0; i < controls.GetButtonCount(); i++ {
		s.clickable(controls.GetButton(i))
	}
	fit := formLayout(controls)
	content := s.newFlex().SetDirection(tview.FlexRow).AddItem(list, 0, 1, true).AddItem(controls, 5, 0, false)
	root := s.formPopup(content, 104, 24, func(w, h int) { fit(w, 5) })
	s.pages.AddPage("workspaces", root, true, true)
	s.app.SetFocus(list)
}
func (s *screen) closeWorkspaces() {
	s.pages.RemovePage("workspaces")
	s.workspaceList = nil
	s.app.SetFocus(s.workspaceFocusBefore)
	s.workspaceFocusBefore = nil
}
func (s *screen) workspaceKey(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyEscape:
		s.closeWorkspaces()
		return nil
	case tcell.KeyCtrlQ:
		s.closeWorkspaces()
		s.requestQuit()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		_, root := s.pages.GetFrontPage()
		popup := root.(*formPopupLayer)
		content := popup.content.(*tview.Flex)
		form := content.GetItem(1).(*tview.Form)
		if s.workspaceList.HasFocus() {
			index := 0
			if e.Key() == tcell.KeyBacktab {
				index = form.GetButtonCount() - 1
			}
			focusForm(s, form, index)
			return nil
		}
		if e.Key() == tcell.KeyTab && form.GetButton(form.GetButtonCount()-1).HasFocus() || e.Key() == tcell.KeyBacktab && form.GetButton(0).HasFocus() {
			s.app.SetFocus(s.workspaceList)
			return nil
		}
		return e

	case tcell.KeyDelete:
		index := s.workspaceList.GetCurrentItem()
		if index >= 0 && index < len(s.workspaces.screens) {
			target := s.workspaces.screens[index]
			s.closeWorkspaces()
			s.workspaces.closeOne(target)
		}
		return nil
	case tcell.KeyRune:
		if e.Rune() == 'n' && e.Modifiers() == 0 && s.workspaceList.HasFocus() {
			s.closeWorkspaces()
			s.workspaces.addNew()
			return nil
		}
	}
	if s.buttonArrow(e) {
		return nil
	}
	return e
}

// A successful replacement connection must not reuse opaque page/LOB/edit IDs
// or cached metadata from the old server, including results in inactive tabs.
func (s *screen) clearConnectionResults() {
	s.clearBindValues()
	s.definitionCache = nil
	s.definitionTrail = nil
	s.cellValue = oracle.Cell{}
	for _, tab := range s.tabs {
		tab.table.Clear()
		tab.rows, tab.headers, tab.values = nil, nil, nil
		tab.query = nil
		tab.truncated = false
		tab.browser = resultBrowser{}
		tab.grid = gridState{}
		tab.selection = resultSelection{}
		tab.script = oracle.ScriptReport{}
	}
	s.render(oracle.Result{})
}

func (s *screen) canSwitchWorkspace() bool {
	if s.workspaces == nil || s.busy || s.vaultBusy || s.quitting || s.workspaces.closingAll {
		return false
	}
	page, _ := s.pages.GetFrontPage()
	switch page {
	case "main", "source", "object", "definition", "arguments", "output", "plan", "script", "routines", "routine_results", "routine_grid", "copy", "cell", "about", "help", "reference":
		return true
	}
	return false
}

func (s *screen) connectionTitleBadge() string {
	if !s.connected {
		return s.connectionBadge()
	}
	p := s.connectionIdentity
	if p.User == "" && p.Host == "" {
		return ""
	}
	label := safeIdentity(p.User) + "@" + safeIdentity(p.Name)
	if p.Alias != "" {
		label += " · " + popupLabel(safeIdentity(p.Alias), 18)
	}
	if p.Environment != "" {
		label = "[" + strings.ToUpper(p.Environment) + "] " + label
	}
	if s.pendingChanges {
		label += " [TX]"
	}
	if s.transactionUncertain {
		label += " !"
	}
	return label
}
