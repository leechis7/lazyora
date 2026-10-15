package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/settings"
)

func fillConnection(s *screen, alias, mode string) {
	for i, value := range map[int]string{0: "db.example", 1: "1522", 2: "scott", 3: "secret-only-in-memory", 5: "ORCL", 7: alias} {
		s.connectForm.GetFormItem(i).(*tview.InputField).SetText(value)
	}
	index := 0
	if mode == "sid" {
		index = 1
	}
	s.connectForm.GetFormItem(4).(*tview.DropDown).SetCurrentOption(index)
}

func TestConnectionProfilesPersistUpdateAndClearPassword(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	fillConnection(s, " 개발 DB ", "sid")
	s.key(tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModCtrl))
	if len(s.jobs) != 0 {
		t.Fatal("saving a profile attempted a database connection")
	}
	path := settings.ConnectionsPath(s.prefsPath)
	profiles, err := settings.LoadConnections(path)
	if err != nil || len(profiles) != 1 || profiles[0].Alias != "개발 DB" || profiles[0].Mode != "sid" || profiles[0].Port != 1522 {
		t.Fatalf("profile was not saved: %+v %v", profiles, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "secret-only-in-memory") || strings.Contains(strings.ToLower(string(data)), "password") {
		t.Fatal("password was written to disk")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("connection file permissions: %v", info.Mode())
	}
	if s.connectForm.GetFormItem(3).(*tview.InputField).GetText() != "secret-only-in-memory" {
		t.Fatal("saving discarded the password needed for connecting")
	}
	s.connectForm.GetFormItem(0).(*tview.InputField).SetText("updated.example")
	s.key(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt))
	profiles, _ = settings.LoadConnections(path)
	if len(profiles) != 1 || profiles[0].Host != "updated.example" {
		t.Fatal("saving the same alias did not update the profile")
	}
	fillConnection(s, "운영 DB", "service")
	s.connectForm.GetButton(2).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	profiles, _ = settings.LoadConnections(path)
	if len(profiles) != 2 {
		t.Fatal("second alias did not create another profile")
	}
	s.closeConnection()
	// Reopen from disk with a password supplied in the environment. Selecting a
	// saved profile must not reuse that password for a different database.
	t.Setenv("LAZYORA_PASSWORD", "environment-secret")
	s.connectionForm()
	s.connectForm.GetFormItem(6).(*tview.DropDown).SetCurrentOption(1)
	if s.connectForm.GetFormItem(3).(*tview.InputField).GetText() != "" || s.connectForm.GetFormItem(0).(*tview.InputField).GetText() != "updated.example" {
		t.Fatal("saved profile was not restored or password was reused")
	}
	index, _ := s.connectForm.GetFormItem(4).(*tview.DropDown).GetCurrentOption()
	if index != 1 {
		t.Fatal("saved SID mode was not restored")
	}
	s.connectForm.GetFormItem(3).(*tview.InputField).SetText("another-secret")
	s.settingsForm()
	s.prefsForm.GetFormItem(0).(*tview.DropDown).SetCurrentOption(1)
	s.savePrefs()
	if s.connectForm.GetFormItem(3).(*tview.InputField).GetText() != "another-secret" {
		t.Fatal("language change discarded connection inputs")
	}
	s.connectForm.GetFormItem(6).(*tview.DropDown).SetCurrentOption(2)
	if s.connectForm.GetFormItem(3).(*tview.InputField).GetText() != "" {
		t.Fatal("switching aliases reused a password")
	}
	index, _ = s.connectForm.GetFormItem(4).(*tview.DropDown).GetCurrentOption()
	if index != 0 {
		t.Fatal("saved Service Name mode was not restored")
	}
}

func TestConnectionProfileSaveFailuresKeepInputsAndFile(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	fillConnection(s, "", "service")
	s.saveConnection()
	if !strings.Contains(s.connectForm.GetTitle(), "alias") {
		t.Fatal("empty alias was accepted")
	}
	fillConnection(s, "dev", "service")
	s.saveConnection()
	path := settings.ConnectionsPath(s.prefsPath)
	before, _ := os.ReadFile(path)
	s.connectForm.GetFormItem(1).(*tview.InputField).SetText("0")
	s.saveConnection()
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid input overwrote the saved profile")
	}
	s.closeConnection()
	// Simulate a write failure after a valid profile file was loaded.
	s.connectionForm()
	fillConnection(s, "dev", "service")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	s.saveConnection()
	if !strings.Contains(s.connectForm.GetTitle(), "Could not save") || s.connectForm.GetFormItem(3).(*tview.InputField).GetText() != "secret-only-in-memory" {
		t.Fatal("write failure was not reported or discarded inputs")
	}
	s.closeConnection()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	s.connectionForm()
	fillConnection(s, "dev", "service")
	s.saveConnection()
	after, _ = os.ReadFile(path)
	if string(after) != "{broken" || !strings.Contains(s.connectForm.GetTitle(), "Could not save") {
		t.Fatal("corrupt profiles were silently overwritten")
	}
	s.closeConnection()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	s.connectionForm()
	fillConnection(s, "dev", "service")
	s.saveConnection()
	if s.connectForm.GetFormItem(3).(*tview.InputField).GetText() != "secret-only-in-memory" {
		t.Fatal("save error discarded inputs")
	}
	if page, _ := s.pages.GetFrontPage(); page != "connect" {
		t.Fatal("failed save closed the form")
	}
}

func TestConnectionProfilePathNeverOverwritesPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	if settings.ConnectionsPath(path) == path || settings.ConnectionsPath("") != "" {
		t.Fatal("invalid connection profile path")
	}
}
