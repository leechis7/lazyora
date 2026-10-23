package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"testing"
)

func TestTabRenamePreservesWorkAndFileNameWins(t *testing.T) {
	s, _ := resultScreen(t)
	s.initDocument()
	s.initTabs()
	before := s.editor.GetText()
	row := s.resultRows[0][1]
	s.showTabRename()
	s.renameForm.GetFormItem(0).(*tview.InputField).SetText("한글 [red] 조회")
	s.renameForm.GetButton(0).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(tview.Primitive) {})
	if s.tabs[0].name != "한글 [red] 조회" || s.editor.GetText() != before || s.resultRows[0][1] != row {
		t.Fatal("rename altered work")
	}
	s.newTab()
	s.switchTab(0)
	if !strings.Contains(s.editor.GetTitle(), "한글") {
		t.Fatal("tab lost name")
	}
	s.document.Path = "/tmp/query.sql"
	s.refreshDocumentTitle()
	if s.commandEnabled("tab_rename") || !strings.Contains(s.editor.GetTitle(), "query.sql") {
		t.Fatal("file name not protected")
	}
	for _, bad := range []string{"", "  ", "a\nb", "a\x1bb", strings.Repeat("가", 41), "a\u202eb"} {
		if validTabName(bad) {
			t.Fatalf("unsafe name %q", bad)
		}
	}
}
