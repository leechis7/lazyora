package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

func workspaceTestSet(t *testing.T) (*workspaceSet, tcell.SimulationScreen) {
	t.Helper()
	terminal := colorTestTerminal(t, 120, 40)
	app := tview.NewApplication().SetScreen(terminal)
	path := filepath.Join(t.TempDir(), "settings.json")
	factory := func(p settings.Config) *screen { return newWorkspaceScreen(app, terminal, p, path, true, nil) }
	first := factory(settings.Config{Language: "en", Mouse: true})
	m := &workspaceSet{app: app, active: first, factory: factory}
	m.adopt(first)
	t.Cleanup(m.shutdown)
	return m, terminal
}
func secondWorkspace(m *workspaceSet) *screen {
	m.addNew()
	s := m.active
	s.closeConnection()
	return s
}
func exitChoice(s *screen, index int) {
	s.exitDialog.SetFocus(index)
	s.exitDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
}

func TestWorkspacesKeepSQLTabsResultsAndTransactionsSeparate(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	first.connected = true
	first.connectionIdentity = settings.Connection{Alias: "First", Environment: "prod", User: "A", Host: "host-a", Port: 1521, Name: "DB_A"}
	first.editor.SetText("SELECT 'first' FROM dual", false).Select(7, 14)
	first.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"first result"}}})
	first.pendingChanges = true
	first.outputLines = []string{"first output"}
	first.newTab()
	first.editor.SetText("SELECT 'second tab' FROM dual", false)
	first.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"second tab result"}}})
	second := secondWorkspace(m)
	second.connected = true
	second.connectionIdentity = settings.Connection{Alias: "Second", Environment: "dev", User: "B", Host: "host-b", Port: 1522, Name: "DB_B"}
	second.editor.SetText("SELECT 'other DB' FROM dual", false)
	second.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"other result"}}})
	if first.jobs == second.jobs || first.editor == second.editor || first.table == second.table {
		t.Fatal("workspace widgets/worker shared")
	}
	if !m.activate(first) {
		t.Fatal("first workspace not activated")
	}
	if len(first.tabs) != 2 || first.activeTab != 1 || first.editor.GetText() != "SELECT 'second tab' FROM dual" || first.resultRows[0][0] != "second tab result" || !first.pendingChanges || first.outputLines[0] != "first output" {
		t.Fatal("workspace state lost")
	}
	first.switchTab(0)
	selected, _, _ := first.editor.GetSelection()
	if selected != "'first'" || first.resultRows[0][0] != "first result" || second.resultRows[0][0] != "other result" || second.pendingChanges {
		t.Fatal("tab/result/transaction crossed workspaces")
	}
	first.refreshConnectionTitles()
	if !strings.Contains(first.editor.GetTitle(), "DB_A") || !strings.Contains(first.table.GetTitle(), "PROD") || strings.Contains(first.editor.GetTitle(), "DB_B") {
		t.Fatal("wrong connection identity")
	}
	if !m.activate(second) {
		t.Fatal("second workspace not activated")
	}
	second.settingsForm()
	second.prefsForm.GetFormItem(3).(*choiceField).SetCurrentOption(1)
	second.savePrefs()
	if first.prefs.Theme != "light" || first.editor.GetBackgroundColor() != first.colors().editor || first.resultRows[0][0] != "first result" {
		t.Fatal("shared preferences damaged inactive workspace")
	}
}

func TestWorkspaceCloseDefaultsToCancelAndProtectsOtherConnection(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	second := secondWorkspace(m)
	first.pendingChanges = true
	m.closeOne(first)
	if first.exitDialog == nil || m.active != first {
		t.Fatal("close did not open target transaction confirmation")
	}
	exitChoice(first, 3)
	if len(m.screens) != 2 || first.workerClosed || !first.pendingChanges || len(first.jobs) != 0 || second.workerClosed {
		t.Fatal("cancel closed a connection or submitted SQL")
	}
	first.pendingChanges = false
	m.closeOne(first)
	exitChoice(first, 0)
	if len(m.screens) != 1 || !first.workerClosed || second.workerClosed || m.active != second {
		t.Fatal("close affected another workspace")
	}
}

func TestGlobalQuitConfirmsEachWorkspaceAndCanCancel(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	second := secondWorkspace(m)
	if !m.activate(first) {
		t.Fatal("activate")
	}
	first.requestQuit()
	if !m.closingAll || first.exitDialog == nil {
		t.Fatal("global quit did not begin")
	}
	exitChoice(first, 0)
	if second.exitDialog == nil || !first.workerClosed {
		t.Fatal("global quit skipped a connection")
	}
	exitChoice(second, 1)
	if m.closingAll || second.workerClosed || len(m.screens) != 1 {
		t.Fatal("quit cancel not honored")
	}
}

func TestWorkspaceQuitCancellationPreservesUnsavedSQL(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	secondWorkspace(m)
	m.activate(first)
	first.editor.SetText("unsaved SQL", false)
	first.requestQuit()
	if first.documentModal == nil {
		t.Fatal("unsaved SQL was not checked")
	}
	modalChoice(first, first.documentModal, 2)
	if m.closingAll || first.editor.GetText() != "unsaved SQL" || len(m.screens) != 2 {
		t.Fatal("unsaved cancel lost state or kept quit chain")
	}
}

func TestWorkspaceBusyBlocksSwitchAndCreation(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	second := secondWorkspace(m)
	second.busy = true
	if m.activate(first) {
		t.Fatal("switched executing workspace")
	}
	m.addNew()
	if len(m.screens) != 2 || m.active != second {
		t.Fatal("new workspace bypassed active operation")
	}
	second.busy = false
	second.showWorkspaces()
	e := second.workspaceKey(tcell.NewEventKey(tcell.KeyRune, 'n', 0))
	if e != nil || len(m.screens) != 3 || m.active == second {
		t.Fatal("N new connection failed")
	}
	if m.active.connectForm == nil {
		t.Fatal("new workspace lacks login popup")
	}
}

func TestWorkspacePickerResizeButtonsAndCloseCancel(t *testing.T) {
	m, terminal := workspaceTestSet(t)
	first := m.active
	secondWorkspace(m)
	m.activate(first)
	first.showWorkspaces()
	_, root := first.pages.GetFrontPage()
	for _, size := range [][2]int{{120, 36}, {80, 24}, {40, 16}} {
		terminal.SetSize(size[0], size[1])
		root.SetRect(0, 0, size[0], size[1])
		root.Draw(terminal)
		first.workspaceKey(tcell.NewEventKey(tcell.KeyTab, 0, 0))
		focused := first.app.GetFocus()
		x, y, w, h := focused.GetRect()
		if x < 0 || y < 0 || x+w > size[0] || y+h > size[1] {
			t.Fatal("picker button clipped", size)
		}
		first.workspaceKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
		first.app.SetFocus(first.workspaceList)
	}
	first.workspaceKey(tcell.NewEventKey(tcell.KeyDelete, 0, 0))
	if first.exitDialog == nil {
		t.Fatal("Delete did not ask before close")
	}
	exitChoice(first, 1)
	if len(m.screens) != 2 {
		t.Fatal("Delete bypassed cancellation")
	}
}

func TestReconnectDropsOldServerHandlesAcrossTabs(t *testing.T) {
	m, _ := workspaceTestSet(t)
	s := m.active
	s.editor.SetText("keep SQL", false)
	s.render(oracle.Result{PageID: 7, EditID: 8, Columns: []string{"VALUE"}, Rows: [][]string{{"preview"}}, Values: [][]oracle.Cell{{{Type: "CLOB", LOBID: 9, Text: "preview"}}}})
	s.newTab()
	s.editor.SetText("keep second SQL", false)
	s.render(oracle.Result{PageID: 10, Columns: []string{"VALUE"}, Rows: [][]string{{"second"}}})
	s.clearConnectionResults()
	s.switchTab(0)
	if s.editor.GetText() != "keep SQL" || len(s.resultValues) != 0 || len(s.resultRows) != 0 || s.resultBrowser.source.PageID != 0 || s.resultBrowser.source.EditID != 0 {
		t.Fatal("old connection data/handles survived")
	}
	s.switchTab(1)
	if s.editor.GetText() != "keep second SQL" || len(s.resultRows) != 0 {
		t.Fatal("reconnect lost SQL or kept results")
	}
}

func TestUncertainWorkspaceOnlyExplicitlyDisconnects(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	second := secondWorkspace(m)
	first.pendingChanges, first.transactionUncertain = true, true
	m.closeOne(first)
	first.confirmExit("exit_commit")
	first.confirmExit("exit_rollback")
	if len(first.jobs) != 0 {
		t.Fatal("uncertain transaction retried resolution")
	}
	exitChoice(first, 1)
	if first.workerClosed || len(m.screens) != 2 {
		t.Fatal("uncertain cancel closed workspace")
	}
	m.closeOne(first)
	first.showPendingWork()
	first.closePendingWork()
	first.exitDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { first.app.SetFocus(p) })
	if first.workerClosed {
		t.Fatal("pending detail lost cancel default")
	}
	m.closeOne(first)
	exitChoice(first, 0)
	if !first.workerClosed || second.workerClosed || len(first.jobs) != 0 {
		t.Fatal("explicit uncertain disconnect touched another session")
	}
}

func TestWorkspaceSwitchPreservesUnappliedSourceAndMetadata(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	second := secondWorkspace(m)
	m.activate(first)
	first.object = oracle.Object{Owner: "A", Name: "P", Kind: "PROCEDURE"}
	first.sourceEditor = tview.NewTextArea().SetText("unapplied source", false)
	first.sourceLoaded = true
	first.sourceBase = "original source"
	first.pages.AddPage("source", first.sourceEditor, true, true)
	first.app.SetFocus(first.sourceEditor)
	first.key(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModAlt))
	if first.workspaceList == nil {
		t.Fatal("Alt+J unavailable in source editor")
	}
	first.workspaceList.SetCurrentItem(1)
	first.workspaceList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { first.app.SetFocus(p) })
	if m.active != second || first.sourceEditor.GetText() != "unapplied source" {
		t.Fatal("workspace switch lost source")
	}
	if !m.activate(first) || !first.sourceEditor.HasFocus() || !first.sourceDirty() || first.object.Owner != "A" {
		t.Fatal("source context was not restored")
	}
	first.requestQuit()
	if first.programModal == nil {
		t.Fatal("unapplied source was not protected on quit")
	}
	first.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { first.app.SetFocus(p) })
	if m.closingAll || len(m.screens) != 2 {
		t.Fatal("source cancel did not abort global quit")
	}
}
