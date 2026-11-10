package ui

import (
	"lazyora/internal/settings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestFavoritesFromHistoryOpenNewTabWithoutExecuting(t *testing.T) {
	m, _ := workspaceTestSet(t)
	s := m.active
	s.sqlHistory = []string{"SELECT :account FROM dual;"}
	s.editor.SetText("SELECT 42 FROM dual;", false)
	s.showHistory()
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 's', tuivory.ModAlt))
	if page, _ := s.pages.GetFrontPage(); page != "favorite_edit" {
		t.Fatal("history save did not open the favorite form")
	}
	s.app.GetFocus().(*tuivory.InputField).SetText("조회 SQL")
	connectionKey(s, tuivory.KeyTab)
	s.app.GetFocus().(*tuivory.InputField).SetText("월간")
	connectionKey(s, tuivory.KeyCtrlS)
	items, err := settings.LoadFavorites(settings.FavoritesPath(s.prefsPath))
	if err != nil || len(items) != 1 || items[0].Tags != "월간" || items[0].SQL != s.sqlHistory[0] {
		t.Fatalf("favorite was not saved: %v, %v", items, err)
	}
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'f', tuivory.ModAlt))
	filter := s.app.GetFocus().(*tuivory.InputField)
	filter.SetText("없는 항목")
	connectionKey(s, tuivory.KeyDown)
	connectionKey(s, tuivory.KeyEnter)
	if len(s.tabs) != 1 {
		t.Fatal("an empty search opened a tab")
	}
	filter.SetText("월간")
	connectionKey(s, tuivory.KeyEnter)
	if len(s.tabs) != 2 || s.editor.GetText() != items[0].SQL || s.tabs[0].editor.GetText() != "SELECT 42 FROM dual;" {
		t.Fatal("favorite did not open in an independent tab")
	}
	if s.busy || len(s.jobs) != 0 {
		t.Fatal("opening a favorite submitted SQL")
	}
}
