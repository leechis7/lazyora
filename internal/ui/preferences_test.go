package ui

import (
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestLanguageChangePreservesSQLResultsAndConnectionInputs(t *testing.T) {
	s := shortcutScreen(t)
	s.render(oracle.Result{Columns: []string{"Original"}, Rows: [][]string{{"한글 SQL data"}}})
	s.connectionForm()
	s.connectForm.GetFormItem(connectionUser).(*tuivory.InputField).SetText("my_user")
	s.connectForm.GetFormItem(connectionPassword).(*tuivory.InputField).SetText("my_password")
	s.settingsForm()
	s.prefsForm.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	s.prefsForm.GetFormItem(1).(*tuivory.Checkbox).SetChecked(false)
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlS, 0, tuivory.ModCtrl))
	if s.prefs.Language != "ko" || s.prefs.Mouse {
		t.Fatal("preferences were not applied")
	}
	if page, _ := s.pages.GetFrontPage(); page != "connect" {
		t.Fatal("connection form was discarded")
	}
	if !strings.Contains(s.connectForm.GetButton(0).GetLabel(), "접속") {
		t.Fatal("connection button was not translated")
	}
	if s.connectForm.GetFormItem(connectionUser).(*tuivory.InputField).GetText() != "my_user" || s.connectForm.GetFormItem(connectionPassword).(*tuivory.InputField).GetText() != "my_password" {
		t.Fatal("connection inputs were lost")
	}
	if s.editor.GetText() != "SELECT 1 FROM dual;" || s.resultRows[0][0] != "한글 SQL data" {
		t.Fatal("SQL or data was changed by translation")
	}
	loaded, err := settings.Load(s.prefsPath)
	if err != nil || loaded != s.prefs {
		t.Fatal("saved preferences differ from the UI")
	}
	data, _ := os.ReadFile(s.prefsPath)
	if strings.Contains(string(data), "my_user") || strings.Contains(string(data), "my_password") {
		t.Fatal("connection credentials were saved")
	}
}

func TestFailedSaveAndCancelKeepPreferences(t *testing.T) {
	s := shortcutScreen(t)
	parent := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(parent, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	s.prefsPath = filepath.Join(parent, "settings.json")
	s.settingsForm()
	s.prefsForm.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	s.savePrefs()
	if s.prefs.Language != "en" {
		t.Fatal("failed save changed active language")
	}
	if page, _ := s.pages.GetFrontPage(); page != "settings" {
		t.Fatal("failed save closed settings")
	}
	if !strings.Contains(s.prefsForm.GetTitle(), "Could not save") {
		t.Fatal("save error was not shown")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, tuivory.ModNone))
	if s.prefs.Language != "en" {
		t.Fatal("cancel changed preferences")
	}
}

func TestConnectionAndPaneShortcuts(t *testing.T) {
	for _, shortcut := range []*tuivory.EventKey{
		tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModCtrl),
		tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModAlt),
		tuivory.NewEventKey(tuivory.KeyRune, 'c', tuivory.ModAlt),
	} {
		s := shortcutScreen(t)
		s.connectionForm()
		s.connectForm.GetFormItem(connectionUser).(*tuivory.InputField).SetText("test_user")
		s.connectForm.GetFormItem(connectionName).(*tuivory.InputField).SetText("FREEPDB1")
		for i, letter := range []rune{'a', 'h', 'p', 'u', 'w', 't', 'n'} {
			s.key(tuivory.NewEventKey(tuivory.KeyRune, letter, tuivory.ModAlt))
			if s.app.GetFocus() != s.connectForm.GetFormItem(i) {
				t.Fatalf("Alt+%c did not focus its field", letter)
			}
		}
		s.key(shortcut)
		select {
		case j := <-s.jobs:
			if j.action != "connect" || j.config.User != "test_user" {
				t.Fatal("shortcut did not submit a connection")
			}
		default:
			t.Fatal("connection shortcut was ignored")
		}
	}
	s := shortcutScreen(t)
	s.key(tuivory.NewEventKey(tuivory.KeyF6, 0, tuivory.ModNone))
	if s.app.GetFocus() != s.table {
		t.Fatal("F6 did not focus results")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF6, 0, tuivory.ModNone))
	if s.app.GetFocus() != s.editor {
		t.Fatal("F6 did not focus editor")
	}
}
