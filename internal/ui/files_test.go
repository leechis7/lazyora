package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/sqlfile"
)

func fileTestScreen(t *testing.T) *screen {
	t.Helper()
	s := shortcutScreen(t)
	s.initDocument()
	s.pendingChanges = true
	s.table.SetCell(0, 0, tview.NewTableCell("keep result"))
	return s
}

func fileInput(s *screen, path string) {
	s.fileDialog.form.GetFormItem(0).(*tview.InputField).SetText(path)
	s.fileDialog.apply()
}

func modalChoice(s *screen, modal *tview.Modal, index int) {
	modal.SetFocus(index)
	modal.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
}

func TestSQLFilesSaveOpenAndDirtyTitle(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetText("-- 한글\nSELECT 42 FROM dual;", false)
	if !strings.Contains(s.editor.GetTitle(), "*") {
		t.Fatal("dirty marker missing")
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlS, 0, 0))
	if s.fileDialog == nil || s.fileDialog.mode != "save" {
		t.Fatal("unnamed Save did not request path")
	}
	path := filepath.Join(t.TempDir(), "한글 쿼리.sql")
	fileInput(s, path)
	if s.fileDialog != nil || s.document.Path != path || s.document.Dirty(s.editor.GetText()) || strings.Contains(s.editor.GetTitle(), "*") {
		t.Fatal("save did not update document and title")
	}
	s.editor.SetText("new SQL", false)
	s.key(tcell.NewEventKey(tcell.KeyCtrlS, 0, 0))
	raw, _ := os.ReadFile(path)
	if string(raw) != "new SQL" {
		t.Fatal("Ctrl+S did not save current file")
	}
	s.editor.SetText("changed", false)
	s.editor.SetText("new SQL", false)
	if s.document.Dirty(s.editor.GetText()) || strings.Contains(s.editor.GetTitle(), "*") {
		t.Fatal("restoring original text still dirty")
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlN, 0, 0))
	if s.document.Path != "" || s.editor.GetText() != "" {
		t.Fatal("New did not create empty document")
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlO, 0, 0))
	fileInput(s, path)
	if s.editor.GetText() != "new SQL" || s.document.Path != path {
		t.Fatal("open did not load saved document")
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModShift))
	if s.fileDialog == nil || s.fileDialog.mode != "save" {
		t.Fatal("Save As shortcut missing")
	}
	copyPath := filepath.Join(filepath.Dir(path), "copy.sql")
	fileInput(s, copyPath)
	if s.document.Path != copyPath {
		t.Fatal("Save As did not change document path")
	}
	if !s.pendingChanges || len(s.jobs) != 0 || s.table.GetCell(0, 0).Text != "keep result" {
		t.Fatal("file commands changed DB work or results")
	}
}

func TestSQLFilesDirtyGuardCancelDiscardAndSaveBeforeOpen(t *testing.T) {
	for _, choice := range []int{0, 1, 2} {
		t.Run([]string{"save", "discard", "cancel"}[choice], func(t *testing.T) {
			s := fileTestScreen(t)
			s.editor.SetText("unsaved SQL", false)
			dir := t.TempDir()
			target := filepath.Join(dir, "target.sql")
			if err := os.WriteFile(target, []byte("target SQL"), 0600); err != nil {
				t.Fatal(err)
			}
			s.showFileDialog("open", nil)
			fileInput(s, target)
			if s.documentModal == nil || s.fileDialog != nil {
				t.Fatal("dirty Open did not prompt outside picker")
			}
			modalChoice(s, s.documentModal, choice)
			if choice == 0 {
				if s.fileDialog == nil || s.editor.GetText() != "unsaved SQL" {
					t.Fatal("Save did not request path before opening target")
				}
				saved := filepath.Join(dir, "saved.sql")
				fileInput(s, saved)
				raw, _ := os.ReadFile(saved)
				if string(raw) != "unsaved SQL" {
					t.Fatal("original SQL not saved")
				}
			}
			if choice == 2 {
				if s.editor.GetText() != "unsaved SQL" || !s.document.Dirty(s.editor.GetText()) {
					t.Fatal("Cancel lost edits")
				}
			} else if s.editor.GetText() != "target SQL" || s.document.Path != target {
				t.Fatal("continuation did not open target")
			}
			if !s.pendingChanges || len(s.jobs) != 0 {
				t.Fatal("dirty guard touched DB transaction")
			}
		})
	}
}

func TestSQLFilesQuitKeepsDocumentAndDBDecisionsSeparate(t *testing.T) {
	for _, choice := range []int{0, 1, 2} {
		s := fileTestScreen(t)
		s.editor.SetText("unsaved", false)
		s.requestQuit()
		if s.documentModal == nil || s.exitDialog != nil {
			t.Fatal("quit did not ask about SQL first")
		}
		if event := s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0)); event != nil {
			t.Fatal("Ctrl+C could bypass guard")
		}
		modalChoice(s, s.documentModal, choice)
		if choice == 0 {
			if s.exitDialog != nil || s.fileDialog == nil {
				t.Fatal("quit advanced before save")
			}
			fileInput(s, filepath.Join(t.TempDir(), "saved.sql"))
		}
		if choice != 2 {
			if s.exitDialog == nil || !s.pendingChanges || len(s.jobs) != 0 {
				t.Fatal("file decision resolved DB transaction")
			}
			s.cancelExit()
			if choice == 1 && !s.document.Dirty(s.editor.GetText()) {
				t.Fatal("canceling DB exit silently cleared dirty state")
			}
		} else if s.exitDialog != nil || s.editor.GetText() != "unsaved" {
			t.Fatal("cancel quit changed content")
		}
	}
	s := fileTestScreen(t)
	s.editor.SetText("unsaved", false)
	s.requestQuit()
	modalChoice(s, s.documentModal, 0)
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.fileDialog != nil || s.exitDialog != nil || s.editor.GetText() != "unsaved" {
		t.Fatal("cancel Save As did not abort quit")
	}
	// Esc and default Enter both cancel the unsaved prompt.
	for _, key := range []tcell.Key{tcell.KeyEscape, tcell.KeyEnter} {
		s.requestQuit()
		s.documentModal.InputHandler()(tcell.NewEventKey(key, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if s.documentModal != nil || s.exitDialog != nil {
			t.Fatal("default guard action did not cancel")
		}
	}
}

func TestSQLFilesOverwriteAndExternalEditConfirmation(t *testing.T) {
	s := fileTestScreen(t)
	path := filepath.Join(t.TempDir(), "query.sql")
	if err := os.WriteFile(path, []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	s.editor.SetText("mine", false)
	s.saveDocument(true, nil)
	fileInput(s, path)
	if s.programModal == nil {
		t.Fatal("Save As overwrote without confirmation")
	}
	modalChoice(s, s.programModal, 1)
	raw, _ := os.ReadFile(path)
	if string(raw) != "external" || s.fileDialog == nil {
		t.Fatal("overwrite cancel changed file or closed picker")
	}
	fileInput(s, path)
	modalChoice(s, s.programModal, 0)
	if err := os.WriteFile(path, []byte("external again"), 0600); err != nil {
		t.Fatal(err)
	}
	// Even a clean editor must detect an external edit rather than claim success.
	s.saveDocument(false, nil)
	if s.programModal == nil {
		t.Fatal("external edit not detected")
	}
	modalChoice(s, s.programModal, 1)
	raw, _ = os.ReadFile(path)
	if string(raw) != "external again" {
		t.Fatal("external edit lost on cancel")
	}
	s.saveDocument(false, nil)
	modalChoice(s, s.programModal, 0)
	raw, _ = os.ReadFile(path)
	if string(raw) != "mine" || s.document.Dirty("mine") {
		t.Fatal("confirmed overwrite failed")
	}
}

func TestSQLFilesFailuresPreserveEditorAndAbortContinuation(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetText("keep unsaved", false)
	s.showFileDialog("open", nil)
	fileInput(s, filepath.Join(t.TempDir(), "missing.sql"))
	modalChoice(s, s.documentModal, 1)
	if s.editor.GetText() != "keep unsaved" || s.document.Path != "" {
		t.Fatal("failed Open lost text")
	}
	continued := false
	s.saveDocument(true, func() { continued = true })
	fileInput(s, filepath.Join(t.TempDir(), "missing-dir", "query.sql"))
	if continued || s.fileDialog == nil || !s.document.Dirty(s.editor.GetText()) {
		t.Fatal("save failure lost document or continued")
	}
	s.fileDialog.close()
	s.key(tcell.NewEventKey(tcell.KeyCtrlW, 0, 0))
	if s.documentModal == nil {
		t.Fatal("Close did not protect edits")
	}
	modalChoice(s, s.documentModal, 1)
	if s.editor.GetText() != "" || s.document.Path != "" {
		t.Fatal("Close did not reset buffer")
	}
}

func TestSQLFilesUnchangedSavePreservesMixedLineEndings(t *testing.T) {
	s := fileTestScreen(t)
	path := filepath.Join(t.TempDir(), "mixed.sql")
	raw := []byte("-- comment\r\nSELECT 1 FROM dual;\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s.openDocument(path)
	s.saveDocument(false, nil)
	got, _ := os.ReadFile(path)
	if string(got) != string(raw) {
		t.Fatal("unchanged Save rewrote original bytes")
	}
	if _, err := sqlfile.Read(path); err != nil {
		t.Fatal(err)
	}
}

func TestSQLFilesPickerKeyboardNavigationAndHelp(t *testing.T) {
	s := fileTestScreen(t)
	s.showFileDialog("open", nil)
	dialog := s.fileDialog
	dir := t.TempDir()
	fileInput(s, dir)
	if dialog.directory != dir {
		t.Fatal("directory path did not navigate")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if !dialog.list.HasFocus() {
		t.Fatal("Tab did not reach list")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if _, ok := s.app.GetFocus().(*tview.Button); !ok {
		t.Fatal("Tab did not reach buttons")
	}
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "file" || s.fileDialog != dialog {
		t.Fatal("help did not restore picker")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.fileDialog != nil || !s.editor.HasFocus() {
		t.Fatal("cancel did not restore editor focus")
	}
}

func TestSQLFilesPickerMouseAtSmallTerminal(t *testing.T) {
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(80, 24)
	for _, language := range []string{"en", "ko"} {
		s := fileTestScreen(t)
		s.prefs.Language = language
		dir := t.TempDir()
		path := filepath.Join(dir, "한글.sql")
		if err := os.WriteFile(path, []byte("SELECT 10 FROM dual;"), 0600); err != nil {
			t.Fatal(err)
		}
		s.showFileDialog("open", nil)
		fileInput(s, dir)
		_, page := s.pages.GetFrontPage()
		root := page.(*formPopupLayer)
		root.SetRect(0, 0, 80, 21)
		root.Draw(terminal)
		list := s.fileDialog.list
		x, y, _, _ := list.GetInnerRect()
		// The parent entry is followed by the SQL file. Click the file to
		// populate Path, then click Open; selecting a file alone never executes it.
		root.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y+1, tcell.ButtonNone, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if got := s.fileDialog.form.GetFormItem(0).(*tview.InputField).GetText(); got != path {
			t.Fatalf("file click selected %q", got)
		}
		content := root.content.(*tview.Flex)
		buttons := content.GetItem(content.GetItemCount() - 1).(*tview.Flex)
		open := buttons.GetItem(0).(*tview.Button)
		x, y, width, height := open.GetRect()
		if x < 0 || x+width > 80 || height != 1 || y >= 21 {
			t.Fatal("file action button off screen")
		}
		root.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+width/2, y, tcell.ButtonNone, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if s.fileDialog != nil || s.document.Path != path || len(s.jobs) != 0 || !s.pendingChanges {
			t.Fatal("mouse open failed or affected DB state")
		}
	}
}

func TestSQLFilesCtrlCInHelpCannotBypassExitGuard(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetText("unsaved SQL", false)
	s.showHelp()
	original := tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
	forwarded := s.key(original)
	if forwarded == original {
		t.Fatal("tview would stop immediately and bypass unsaved SQL confirmation")
	}
	if forwarded != nil && forwarded.Key() != tcell.KeyCtrlC {
		t.Fatal("copy key changed")
	}
	if s.editor.GetText() != "unsaved SQL" || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("copy touched SQL or DB work")
	}
}
