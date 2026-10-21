package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

func editorEvent(s *screen, event *tcell.EventKey) {
	if forwarded := s.key(event); forwarded != nil {
		s.app.GetFocus().InputHandler()(forwarded, func(p tview.Primitive) { s.app.SetFocus(p) })
	}
}
func editorText(s *screen, text string) {
	for _, r := range text {
		editorEvent(s, tcell.NewEventKey(tcell.KeyRune, r, 0))
	}
}
func mappedScreen(t *testing.T, keymap, text string) *screen {
	s := demoBrowserScreen(t)
	s.prefs.EditorKeymap = keymap
	s.editor.SetText(text, false).Select(0, 0)
	s.app.SetFocus(s.editor)
	return s
}
func TestDefaultMapAndSettingsSwitchKeepTextAndUndo(t *testing.T) {
	s := mappedScreen(t, "", "한글 SELECT")
	editorText(s, "i")
	if s.editor.GetText() != "i한글 SELECT" {
		t.Fatal("default mode swallowed Vim letter")
	}
	for index, keymap := range []string{"", "vim", "notepad++"} {
		original := s.editor.GetText()
		s.settingsForm()
		s.prefsForm.GetFormItem(2).(*choiceField).SetCurrentOption(index)
		s.savePrefs()
		saved, err := settings.Load(s.prefsPath)
		if err != nil || saved.EditorKeymap != keymap || s.editor.GetText() != original {
			t.Fatal("keymap selection lost document or was not persisted")
		}
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
	if s.editor.GetText() != "한글 SELECT" {
		t.Fatal("settings switch cleared undo")
	}
}
func TestNotepadKeysDuplicateMoveAndUndoUnicode(t *testing.T) {
	s := mappedScreen(t, "notepad++", "첫째\n둘째\n셋째")
	s.editor.Select(len("첫째\n"), len("첫째\n"))
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlD, 0, 0))
	if s.editor.GetText() != "첫째\n둘째\n둘째\n셋째" {
		t.Fatalf("duplicate: %q", s.editor.GetText())
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
	s.editor.Select(len("첫째\n"), len("첫째\n둘째"))
	editorEvent(s, tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModCtrl|tcell.ModShift))
	selected, _, _ := s.editor.GetSelection()
	if s.editor.GetText() != "첫째\n셋째\n둘째" || selected != "둘째" {
		t.Fatalf("move: %q selected %q", s.editor.GetText(), selected)
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
	s.editor.Select(len("첫째\n"), len("첫째\n"))
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlT, 0, 0))
	if s.editor.GetText() != "둘째\n첫째\n셋째" {
		t.Fatal("Ctrl+T did not transpose")
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlA, 0, 0))
	selected, _, _ = s.editor.GetSelection()
	if selected != s.editor.GetText() {
		t.Fatal("Ctrl+A did not select all")
	}
}
func TestNotepadClipboardAndReadOnlyPanels(t *testing.T) {
	s := mappedScreen(t, "notepad++", "한글\nlast")
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if s.clipboardText != "한글\n" {
		t.Fatal("line copy missing")
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlL, 0, 0))
	if s.editor.GetText() != "last" {
		t.Fatal("line cut failed")
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
	before := s.editor.GetText()
	s.app.SetFocus(s.table)
	if s.editorKey(tcell.NewEventKey(tcell.KeyCtrlD, 0, 0)) || s.editor.GetText() != before {
		t.Fatal("map leaked into results")
	}
	s.connectionForm()
	if s.editorKey(tcell.NewEventKey(tcell.KeyCtrlA, 0, 0)) {
		t.Fatal("map leaked into connection form")
	}
}
func TestVimModesGraphemesSelectionAndUndo(t *testing.T) {
	s := mappedScreen(t, "vim", "가e\u0301👩‍💻Z")
	editorText(s, "llvhy") // Select emoji and preceding combining cluster backwards.
	if s.clipboardText != "e\u0301👩‍💻" || s.vim.mode != "NORMAL" {
		t.Fatalf("visual copy: %q mode %s", s.clipboardText, s.vim.mode)
	}
	editorText(s, "0x")
	if s.editor.GetText() != "e\u0301👩‍💻Z" || !utf8.ValidString(s.editor.GetText()) {
		t.Fatal("delete split Unicode")
	}
	editorText(s, "u")
	if s.editor.GetText() != "가e\u0301👩‍💻Z" {
		t.Fatal("undo failed")
	}
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlR, 0, 0))
	if s.editor.GetText() != "e\u0301👩‍💻Z" {
		t.Fatal("redo failed")
	}
	editorText(s, "i한글")
	editorEvent(s, tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.vim.mode != "NORMAL" || !strings.HasPrefix(s.editor.GetText(), "한글") {
		t.Fatal("insert/escape failed")
	}
	before := s.editor.GetText()
	editorText(s, ":")
	if before != s.editor.GetText() {
		t.Fatal("unsupported normal command inserted text")
	}
}
func TestVimWordsCountsLineYankAndPaste(t *testing.T) {
	s := mappedScreen(t, "vim", "one two three\n둘째\nlast")
	editorText(s, "2dw")
	if s.editor.GetText() != "three\n둘째\nlast" {
		t.Fatalf("count delete: %q", s.editor.GetText())
	}
	editorText(s, "ujyyGp")
	if s.editor.GetText() != "one two three\n둘째\nlast\n둘째" {
		t.Fatalf("line paste: %q", s.editor.GetText())
	}
	editorText(s, "ggdd")
	if s.editor.GetText() != "둘째\nlast\n둘째" {
		t.Fatalf("dd: %q", s.editor.GetText())
	}
}
func TestVimDoesNotConsumeExecutionHelpOrBusyCancel(t *testing.T) {
	for _, event := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl), tcell.NewEventKey(tcell.KeyF9, 0, 0)} {
		s := mappedScreen(t, "vim", "SELECT 1 FROM dual")
		s.connected = true
		s.key(event)
		if len(s.jobs) != 1 {
			t.Fatal("Vim swallowed SQL execution")
		}
	}
	s := mappedScreen(t, "vim", "SELECT 1")
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "help" {
		t.Fatal("Vim swallowed help")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "main" {
		t.Fatal("help escape entered editor map")
	}
}
func TestVimSourceEscapeAndFindStayInSource(t *testing.T) {
	s := programScreen(t)
	s.prefs.EditorKeymap = "vim"
	sql := s.editor.GetText()
	original := s.sourceEditor.GetText()
	editorText(s, "i-- 한글")
	editorEvent(s, tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "source" || s.programModal != nil {
		t.Fatal("Vim Escape closed source")
	}
	s.findText = "BEGIN"
	s.findEditor(s.sourceEditor, false)
	_, a, _ := s.sourceEditor.GetSelection()
	if a != strings.Index(s.sourceEditor.GetText(), "BEGIN") || !s.sourceEditor.HasFocus() || s.editor.GetText() != sql {
		t.Fatal("source find changed hidden SQL")
	}
	s.sourceEditor.SetText(original, false)
	s.key(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt))
	if s.sourceEditor != nil {
		t.Fatal("Alt+X did not close source")
	}
}

func TestVimSearchDirectionAndChangeWord(t *testing.T) {
	s := mappedScreen(t, "vim", "one two one")
	editorText(s, "cwX")
	editorEvent(s, tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.editor.GetText() != "X two one" {
		t.Fatalf("cw ate separator: %q", s.editor.GetText())
	}
	s = mappedScreen(t, "vim", "one two one")
	editorText(s, "?")
	s.searchForm.GetFormItem(0).(*tview.InputField).SetText("one")
	s.searchAction("submit")
	_, at, _ := s.editor.GetSelection()
	if at != 8 {
		t.Fatalf("backward search: %d", at)
	}
	editorText(s, "n")
	_, at, _ = s.editor.GetSelection()
	if at != 0 {
		t.Fatalf("repeat search: %d", at)
	}
}

func TestKeymapBusyAndEditorTabsAreIsolated(t *testing.T) {
	s := fileTestScreen(t)
	s.prefs.EditorKeymap = "vim"
	s.editor.SetText("original", false).Select(0, 0)
	s.app.SetFocus(s.editor)
	editorText(s, "i")
	s.initTabs()
	s.newTab()
	first := s.tabs[0].editor
	editorText(s, "x")
	if s.editor.GetText() != "" || first.GetText() != "original" {
		t.Fatal("new tab inherited insert mode or edited old document")
	}
	s.switchTab(0)
	s.busy = true
	editorText(s, "x")
	if s.editor.GetText() != "original" {
		t.Fatal("busy normal command inserted literal text")
	}
	s.busy = false
	for _, keymap := range []string{"", "vim", "notepad++"} {
		s.prefs.EditorKeymap = keymap
		s.render(oracle.Result{Columns: []string{"C"}, Rows: [][]string{{"unchanged"}}})
		s.app.SetFocus(s.table)
		if s.editorKey(tcell.NewEventKey(tcell.KeyRune, 'j', 0)) {
			t.Fatal("editor map consumed result-grid key")
		}
	}
}

func TestSourceEditMenuAlternativeAndCompletionBinding(t *testing.T) {
	s := programScreen(t)
	s.prefs.EditorKeymap = "notepad++"
	sql := s.editor.GetText()
	before := s.sourceEditor.GetText()
	s.key(tcell.NewEventKey(tcell.KeyF10, 0, 0))
	page, root := s.pages.GetFrontPage()
	if page != "editor_actions" {
		t.Fatal("source menu did not open")
	}
	list := root.(*tview.List)
	for i := 0; i < list.GetItemCount(); i++ {
		label, _ := list.GetItemText(i)
		if strings.Contains(label, s.tr("duplicate_line")) {
			list.SetCurrentItem(i)
			list.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
			break
		}
	}
	if s.sourceEditor.GetText() == before || s.editor.GetText() != sql || !s.sourceEditor.HasFocus() {
		t.Fatal("source edit menu targeted wrong editor")
	}
	s.sourceEditor.SetText("SEL", false).Select(3, 3)
	s.key(tcell.NewEventKey(tcell.KeyCtrlSpace, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "completion" {
		t.Fatal("NPP completion unavailable")
	}
	s.cancelCompletion()
	for _, lang := range []string{"en", "ko"} {
		s.prefs.Language = lang
		if !strings.Contains(s.mainHelp(), "Ctrl+Space") || s.editorBindingLabels("select_all", 1) != "Ctrl+A" {
			t.Fatal("help keys do not reflect map")
		}
	}
}

func TestLineEditsAtDocumentEdgesAndMultilineSelection(t *testing.T) {
	for _, tc := range []struct {
		text, action, want string
		pos                int
	}{
		{"", "duplicate_line", "\n", 0},
		{"a", "duplicate_line", "a\na", 0},
		{"a\nb", "cut_line", "a", 2},
		{"a\n", "delete_line", "a", 2},
		{"a\nb", "line_up", "b\na", 2},
		{"a\nb\n", "line_down", "b\na\n", 0},
	} {
		s := mappedScreen(t, "notepad++", tc.text)
		s.editor.Select(tc.pos, tc.pos)
		s.editorLineAction(s.editor, tc.action)
		if s.editor.GetText() != tc.want {
			t.Fatalf("%s %q: %q", tc.action, tc.text, s.editor.GetText())
		}
		s.nativeEditorKey(s.editor, tcell.KeyCtrlZ)
		if s.editor.GetText() != tc.text {
			t.Fatalf("undo %s: %q", tc.action, s.editor.GetText())
		}
	}
	s := mappedScreen(t, "notepad++", "top\na\nb\nend\n")
	s.editor.Select(4, 8)
	s.editorLineAction(s.editor, "line_up")
	selected, _, _ := s.editor.GetSelection()
	if s.editor.GetText() != "a\nb\ntop\nend\n" || selected != "a\nb\n" {
		t.Fatalf("block move: %q selection %q", s.editor.GetText(), selected)
	}
}

func TestVimReplaceMenuUsesCursorAndDoesNotSkipFollowingMatch(t *testing.T) {
	s := mappedScreen(t, "vim", "aa")
	s.showSearch(true)
	s.searchForm.GetFormItem(0).(*tview.InputField).SetText("a")
	s.searchForm.GetFormItem(1).(*tview.InputField).SetText("b")
	s.searchAction("replace")
	_, at, _ := s.editor.GetSelection()
	if s.editor.GetText() != "ba" || at != 1 {
		t.Fatalf("replace next match: %q %d", s.editor.GetText(), at)
	}
	s.showSearch(true)
	s.searchForm.GetFormItem(0).(*tview.InputField).SetText("a")
	s.searchForm.GetFormItem(1).(*tview.InputField).SetText("")
	s.searchAction("replace")
	if s.editor.GetText() != "b" {
		t.Fatal("empty replacement failed")
	}
}
