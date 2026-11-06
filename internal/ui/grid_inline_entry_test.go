package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

func inlineEntryScreen(t *testing.T, colors int, lang string) (*screen, tuivory.SimulationScreen) {
	t.Helper()
	raw := colorTestTerminal(t, 120, 36)
	terminal := auditTerminal{raw, colors}
	app := tuivory.NewApplication().SetScreen(terminal)
	raw.SetSize(120, 36)
	s := newWorkspaceScreen(app, terminal, settings.Config{Language: lang, DisplayMode: "unicode", Mouse: true}, filepath.Join(t.TempDir(), "settings.json"), true, nil)
	spaces := &workspaceSet{app: app, active: s}
	spaces.adopt(s)
	s.workerClosed = true
	t.Cleanup(func() { s.workerClosed = false; spaces.shutdown() })
	s.connected = true
	s.render(oracle.Result{EditID: 5, EditRowIDColumn: 1, EditObject: oracle.Object{Owner: "APP", Name: "ITEMS", Kind: "TABLE"}, EditColumns: []bool{false, true}, EditColumnNames: []string{"", "TITLE"}, EditRowIDs: []string{"physical-row-1"}, Columns: []string{"RID", "TITLE"}, Rows: [][]string{{"physical-row-1", "한글 값"}}, Values: [][]oracle.Cell{{{Text: "physical-row-1", Type: "ROWID"}, {Text: "한글 값", Type: "VARCHAR2"}}}})
	s.table.Select(1, 0)
	app.SetFocus(s.table)
	app.ForceDraw()
	if len(s.resultHeaders) != 1 || s.resultHeaders[0] != "TITLE" {
		t.Fatal("verified ROWID became visible")
	}
	return s, raw
}
func inlineEntryKey(s *screen, event *tuivory.EventKey) {
	if capture := s.app.GetInputCapture(); capture != nil {
		event = capture(event)
	}
	if event != nil {
		s.app.GetFocus().InputHandler()(event, func(p tuivory.Primitive) { s.app.SetFocus(p) })
	}
	s.app.ForceDraw()
}
func inlineEntryDoubleClick(s *screen) {
	x, y, _ := s.resultCellRect(1, 0)
	for _, action := range []tuivory.MouseAction{tuivory.MouseLeftDown, tuivory.MouseLeftUp, tuivory.MouseLeftClick, tuivory.MouseLeftDoubleClick} {
		buttons := tuivory.ButtonNone
		if action == tuivory.MouseLeftDown {
			buttons = tuivory.Button1
		}
		event := tuivory.NewEventMouse(x+1, y, buttons, 0)
		if capture := s.app.GetMouseCapture(); capture != nil {
			event, action = capture(event, action)
		}
		if event != nil {
			s.rootLayout.MouseHandler()(action, event, func(p tuivory.Primitive) { s.app.SetFocus(p) })
		}
		s.app.ForceDraw()
	}
}
func TestResultEnterAndDoubleClickKeepCursorInsideCell(t *testing.T) {
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, mode := range []string{"unicode", "ascii"} {
			t.Run(theme+"/"+mode, func(t *testing.T) { checkSharedInlineEntry(t, theme, mode) })
		}
	}
}
func checkSharedInlineEntry(t *testing.T, theme, mode string) {
	for _, key := range []string{"LAZYORA_HOST", "LAZYORA_USER", "LAZYORA_PASSWORD", "LAZYORA_NAME"} {
		t.Setenv(key, "")
	}
	for _, lang := range []string{"ko", "en"} {
		for _, colors := range []int{8, 16, 256, 1 << 24} {
			for _, entry := range []string{"enter", "double"} {
				t.Run(fmt.Sprintf("%s/%d/%s", lang, colors, entry), func(t *testing.T) {
					s, raw := inlineEntryScreen(t, colors, lang)
					s.prefs.Theme, s.prefs.DisplayMode = theme, mode
					s.app.ForceDraw()
					if entry == "enter" {
						inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
					} else {
						inlineEntryDoubleClick(s)
					}
					page, _ := s.pages.GetFrontPage()
					if page != "main" || s.inlineEdit == nil || s.editForm != nil || s.cellForm != nil || s.programModal != nil {
						t.Fatal("activation opened a popup instead of in-cell input")
					}
					if s.inlineEdit.GetText() != "한글 값" || s.inlineEdit.change.Column != 1 {
						t.Fatal("hidden ROWID changed the edit target")
					}
					for _, size := range [][2]int{{120, 36}, {80, 24}, {40, 16}, {120, 36}} {
						raw.SetSize(size[0], size[1])
						s.app.ForceDraw()
						x, y, w := s.resultCellRect(s.table.GetSelection())
						h := 1
						cx, cy, visible := raw.GetCursor()
						gx, gy, gw := s.resultCellRect(1, 0)
						if x != gx || y != gy || w > gw || h != 1 || !visible || cx < x || cx >= x+w || cy != y {
							t.Fatalf("%v: input=%d,%d %dx%d cursor=%d,%d visible=%v", size, x, y, w, h, cx, cy, visible)
						}
					}
					inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyRune, 'X', 0))
					if !strings.Contains(s.inlineEdit.GetText(), "X") || s.resultBrowser.source.Rows[0][1] != "한글 값" || s.pendingChanges {
						t.Fatal("typing bypassed the in-cell draft")
					}
					inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
					if s.inlineEdit != nil || !s.table.HasFocus() {
						t.Fatal("Esc failed to return to the grid")
					}
					_, _, visible := raw.GetCursor()
					if visible {
						t.Fatal("draft cursor remained visible after cancel")
					}
					inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyRune, 'd', tuivory.ModAlt))
					page, _ = s.pages.GetFrontPage()
					if page != "grid_edit" || s.editForm == nil {
						t.Fatal("Alt+D separate editor was removed")
					}
				})
			}
		}
	}
}
func TestUnavailableGridEditStaysInPlace(t *testing.T) {
	guards := []struct {
		name  string
		block func(*screen)
	}{
		{"snapshot", func(s *screen) { s.resultBrowser.source.EditID = 0 }},
		{"column", func(s *screen) { s.resultBrowser.source.EditColumns[1] = false }},
		{"disconnected", func(s *screen) { s.connected = false }},
		{"uncertain", func(s *screen) { s.transactionUncertain = true }},
		{"busy", func(s *screen) { s.busy = true }},
	}
	for _, guard := range guards {
		for _, entry := range []string{"enter", "double"} {
			t.Run(guard.name+"/"+entry, func(t *testing.T) {
				s, _ := inlineEntryScreen(t, 256, "ko")
				guard.block(s)
				if entry == "enter" {
					inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
				} else {
					inlineEntryDoubleClick(s)
				}
				page, _ := s.pages.GetFrontPage()
				if page != "main" || s.inlineEdit != nil || s.cellForm != nil || s.editForm != nil || s.programModal != nil || !s.table.HasFocus() {
					t.Fatal("blocked edit opened a popup or changed focus")
				}
				if !s.busy && !strings.Contains(s.status.GetText(true), s.tr("grid_inline_unavailable")) {
					t.Fatal("blocked edit has no explanation")
				}
				if len(s.resultHeaders) != 1 || s.resultHeaders[0] != "TITLE" {
					t.Fatal("read-only state revealed physical ROWID")
				}
			})
		}
	}
}

func TestSharedInlineApprovalCursorAndLiteralDraft(t *testing.T) {
	s, raw := inlineEntryScreen(t, 256, "ko")
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	draft := "한글|😀e\u0301" + strings.Repeat("긴 값", 30)
	setInlineText(s, draft)
	s.app.ForceDraw()
	_, _, visible := raw.GetCursor()
	if !visible || s.inlineEdit.GetText() != draft {
		t.Fatal("SDK draft/cursor lost Unicode or literal pipe")
	}
	editor := s.inlineEdit
	x, y, _, _ := s.editor.GetInnerRect()
	if mouse, _ := s.app.GetMouseCapture()(tuivory.NewEventMouse(x, y, tuivory.Button1, 0), tuivory.MouseLeftDown); mouse != nil {
		t.Fatal("click outside grid escaped the draft workflow")
	}
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	session, active := editor.model.EditSession()
	_, _, visible = raw.GetCursor()
	if !active || !session.Pending || s.programModal == nil || visible {
		t.Fatal("host approval did not preserve a pending draft with hidden cursor")
	}
	confirmInline(t, s, false)
	s.app.ForceDraw()
	session, active = editor.model.EditSession()
	_, _, visible = raw.GetCursor()
	if !active || session.Pending || !visible || session.Value.Text != draft {
		t.Fatal("rejection failed to restore the same editable draft/cursor")
	}
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyDelete, 0, tuivory.ModAlt))
	session, _ = editor.model.EditSession()
	_, _, visible = raw.GetCursor()
	if !session.Value.Null || visible {
		t.Fatal("NULL must hide the shared cursor")
	}
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyRune, 'X', 0))
	session, _ = editor.model.EditSession()
	if session.Value.Null || session.Value.Text != "X" {
		t.Fatal("typing after NULL did not replace the empty draft")
	}
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyCtrlA, 0, 0))
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyCtrlC, 0, 0))
	if s.clipboardText != "X" {
		t.Fatal("shared edit copy was not connected to the host clipboard")
	}
	inlineEntryKey(s, tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if s.pendingChanges || s.resultBrowser.source.Rows[0][1] != "한글 값" {
		t.Fatal("cancel mutated source")
	}
	s.app.SetFocus(s.editor)
	s.app.ForceDraw()
	cx, cy, visible := raw.GetCursor()
	x, y, w, h := s.editor.GetInnerRect()
	if !visible || cx < x || cx >= x+w || cy < y || cy >= y+h {
		t.Fatal("closed grid editor stole the SQL editor cursor")
	}
}

func TestSharedInlineRejectsStaleApproval(t *testing.T) {
	s, _ := inlineFixture(t)
	s.startInlineEdit(nil)
	setInlineText(s, "stale")
	editor := s.inlineEdit
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	session, active := editor.model.EditSession()
	if !active || !session.Pending {
		t.Fatal("approval not pending")
	}
	replacement := s.resultBrowser.source
	replacement.EditID++
	s.render(replacement)
	confirmInline(t, s, true)
	if len(s.jobs) != 0 || s.inlineEdit != nil || editor.model.ResolveEdit(session.ID, "", nil) {
		t.Fatal("stale approval submitted an UPDATE or revived an invalidated editor")
	}
}
