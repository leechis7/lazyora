package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/credentials"
	"lazyora/internal/settings"
)

func runningVaultUI(t *testing.T) *screen {
	t.Helper()
	t.Setenv("LAZYORA_CONFIG", filepath.Join(t.TempDir(), "settings.json"))
	t.Setenv("LC_ALL", "C")
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	terminal.SetSize(100, 30)
	ready := make(chan *screen, 1)
	finished := make(chan error, 1)
	go func() {
		finished <- run(true, terminal, func(s *screen) {
			var once sync.Once
			s.app.SetAfterDrawFunc(func(tcell.Screen) { once.Do(func() { ready <- s }) })
		})
	}()
	var s *screen
	select {
	case s = <-ready:
	case err := <-finished:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("app did not start")
	}
	t.Cleanup(func() {
		s.app.QueueUpdateDraw(func() { s.app.Stop() })
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("app did not stop")
		}
	})
	return s
}
func waitVaultUI(t *testing.T, s *screen) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		busy := false
		s.app.QueueUpdateDraw(func() { busy = s.vaultBusy })
		if !busy {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("password operation did not finish")
}
func inputMaster(s *screen, value string) {
	s.masterForm.GetFormItem(1).(*tview.InputField).SetText(value)
	if s.masterForm.GetFormItemCount() > 2 {
		s.masterForm.GetFormItem(2).(*tview.InputField).SetText(value)
	}
	s.masterSubmit()
}
func TestMasterPasswordUIRoundTripAndCancellation(t *testing.T) {
	s := runningVaultUI(t)
	s.app.QueueUpdateDraw(func() {
		s.connectionForm()
		fillConnection(s, "development", "service")
		s.connectForm.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(true)
		s.saveConnection()
		if s.masterForm == nil {
			t.Error("first save did not ask for master")
			return
		}
		inputMaster(s, "short")
		if !strings.Contains(s.masterForm.GetTitle(), "8") {
			t.Error("weak master accepted")
		}
		inputMaster(s, "long master password!")
	})
	waitVaultUI(t, s)
	var path string
	s.app.QueueUpdateDraw(func() {
		if s.masterForm != nil {
			t.Error("master setup failed")
		}
		path = credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath))
		profiles, err := settings.LoadConnections(settings.ConnectionsPath(s.prefsPath))
		if err != nil || len(profiles) != 1 || !profiles[0].SavedPassword {
			t.Error("save marker missing")
		}
		s.lockPasswords()
		if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "" {
			t.Error("lock retained password field")
		}
		s.closeConnection()
		s.connectionForm()
		s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(1)
		if s.masterForm == nil {
			t.Error("reopening saved profile did not ask to unlock")
			return
		}
		inputMaster(s, "wrong master")
	})
	waitVaultUI(t, s)
	s.app.QueueUpdateDraw(func() {
		if s.masterForm == nil {
			t.Error("wrong master accepted")
			return
		}
		if strings.Contains(s.masterForm.GetTitle(), "wrong master") {
			t.Error("master leaked into error")
		}
		s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
		if s.masterForm != nil || s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "" {
			t.Error("cancel loaded a secret")
		}
		// Saving after cancelled unlock must never replace the saved secret with empty text.
		s.saveConnection()
		if s.masterForm != nil || !strings.Contains(s.connectForm.GetTitle(), "Enter a DB password") {
			t.Error("empty password could overwrite the saved secret")
		}
		// Cancelled unlock still permits typing a password manually.
		s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("manual-entry")
		s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(1)
		if s.masterForm == nil {
			t.Error("retry did not ask for master")
			return
		}
		inputMaster(s, "long master password!")
	})
	waitVaultUI(t, s)
	s.app.QueueUpdateDraw(func() {
		if s.masterForm != nil {
			t.Error("correct master rejected")
			return
		}
		if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "secret-only-in-memory" {
			t.Error("saved password not restored")
		}
		s.connectForm.GetFormItem(connectionHost).(*tview.InputField).SetText("other-host")
		if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "" {
			t.Error("autofilled password followed a host edit")
		}
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"long master password!", "secret-only-in-memory"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("plaintext written")
		}
	}
}

func TestPasswordOptOutDeletesOnlySelectedSecretAndResetNeedsConfirmation(t *testing.T) {
	s := shortcutScreen(t)
	store, err := credentials.Open(credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath)))
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Create([]byte("master password for tests")); err != nil {
		t.Fatal(err)
	}
	s.vault = store
	t.Cleanup(store.Lock)
	s.connectionForm()
	fillConnection(s, "first", "service")
	s.connectForm.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(true)
	s.saveConnection()
	fillConnection(s, "second", "service")
	s.saveConnection()
	profiles, err := settings.LoadConnections(settings.ConnectionsPath(s.prefsPath))
	if err != nil || len(profiles) != 2 {
		t.Fatal("profiles not saved", err)
	}
	s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	s.connectForm.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(false)
	s.saveConnection()
	if _, ok, err := store.Get(profiles[0].ID, profiles[0].Binding()); err != nil || ok {
		t.Fatal("opt-out did not delete secret", err)
	}
	if _, ok, err := store.Get(profiles[1].ID, profiles[1].Binding()); err != nil || !ok {
		t.Fatal("other profile lost", err)
	}
	s.passwordManager()
	s.vaultForm.GetButton(3).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(tview.Primitive) {})
	if s.programModal == nil {
		t.Fatal("reset did not confirm")
	}
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyEscape, 0, 0), func(tview.Primitive) {})
	if !store.Exists() {
		t.Fatal("cancel reset deleted passwords")
	}
	s.vaultForm.GetButton(3).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(tview.Primitive) {})
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, 0), func(tview.Primitive) {})
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(tview.Primitive) {})
	path := credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath))
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("reset did not remove passwords", err)
	}
	profiles, err = settings.LoadConnections(settings.ConnectionsPath(s.prefsPath))
	if err != nil || len(profiles) != 2 {
		t.Fatal("reset removed profiles", err)
	}
	for _, p := range profiles {
		if p.SavedPassword {
			t.Fatal("reset left saved marker")
		}
	}
}

func TestMasterHelpPreservesInputWithoutDisplayingIt(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	s.ensurePasswords(func() {})
	if s.masterForm == nil {
		t.Fatal("master form missing")
	}
	s.masterForm.GetFormItem(1).(*tview.InputField).SetText("secret passphrase not for help")
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "help" {
		t.Fatal("F1 did not open help")
	}
	if strings.Contains(s.mainHelp(), "secret passphrase not for help") {
		t.Fatal("master leaked into help")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "master" {
		t.Fatal("help did not restore master form")
	}
	if s.masterForm.GetFormItem(1).(*tview.InputField).GetText() != "secret passphrase not for help" {
		t.Fatal("help discarded input")
	}
	s.masterCancel()
}

func TestForgottenMasterAllowsManualDatabaseLogin(t *testing.T) {
	s := shortcutScreen(t)
	path := credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath))
	store, err := credentials.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Create([]byte("12345678")); err != nil {
		t.Fatal(err)
	}
	id, err := settings.NewConnectionID()
	if err != nil {
		t.Fatal(err)
	}
	profile := settings.Connection{ID: id, SavedPassword: true, Alias: "saved", Host: "db.example", Port: 1521, User: "scott", Mode: "service", Name: "ORCL"}
	if err = store.Put(id, profile.Binding(), "saved-db-password"); err != nil {
		t.Fatal(err)
	}
	store.Lock()
	if err = settings.SaveConnections(settings.ConnectionsPath(s.prefsPath), []settings.Connection{profile}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s.connectionForm()
	s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	if s.masterForm == nil {
		t.Fatal("saved profile did not request unlock")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("manually-entered-db-password")
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl))
	select {
	case j := <-s.jobs:
		if j.action != "connect" || j.config.Password != "manually-entered-db-password" || j.config.User != "scott" {
			t.Fatal("manual credentials were not submitted")
		}
	default:
		t.Fatal("forgotten master blocked manual login")
	}
	if s.masterForm != nil || s.vault.Unlocked() {
		t.Fatal("manual login unlocked the store")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("manual login changed the store")
	}
}
