package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/credentials"
	"lazyora/internal/settings"
)

func TestConnectionSaveConflictPreservesOtherInstanceAndInputs(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		t.Run(language, func(t *testing.T) {
			a, b := shortcutScreen(t), shortcutScreen(t)
			b.prefsPath, b.prefs.Language = a.prefsPath, language
			a.connectionForm()
			fillConnection(a, "first", "service")
			a.saveConnection()
			a.closeConnection()
			a.connectionForm()
			b.connectionForm()
			fillConnection(a, "second", "sid")
			a.saveConnection()
			path := settings.ConnectionsPath(a.prefsPath)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fillConnection(b, "first", "service")
			b.connectForm.GetFormItem(connectionHost).(*tview.InputField).SetText("stale-edit.example")
			b.saveConnection()
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) || !strings.Contains(b.connectForm.GetTitle(), b.tr("connections_changed")) {
				t.Fatal("stale instance lost another profile or did not report conflict", err)
			}
			if b.connectForm.GetFormItem(connectionHost).(*tview.InputField).GetText() != "stale-edit.example" || b.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "secret-only-in-memory" {
				t.Fatal("conflict discarded typed inputs")
			}
			// Explicitly reopen/review the latest list, then save the edited profile.
			b.closeConnection()
			b.connectionForm()
			fillConnection(b, "first", "service")
			b.connectForm.GetFormItem(connectionHost).(*tview.InputField).SetText("reviewed-edit.example")
			b.saveConnection()
			profiles, err := settings.LoadConnections(path)
			if err != nil || len(profiles) != 2 || profiles[0].Host != "reviewed-edit.example" || profiles[1].Alias != "second" {
				t.Fatal("reviewed retry lost profiles", err)
			}
		})
	}
}

func savedConnectionScreen(t *testing.T) (*screen, *credentials.Store) {
	t.Helper()
	s := shortcutScreen(t)
	store, err := credentials.Open(credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create([]byte("master password for conflict tests")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Lock)
	s.vault = store
	s.connectionForm()
	fillConnection(s, "saved", "service")
	s.connectForm.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(true)
	s.saveConnection()
	return s, store
}

func TestProfileLockPreventsPasswordSaveAndReset(t *testing.T) {
	s, _ := savedConnectionScreen(t)
	path := settings.ConnectionsPath(s.prefsPath)
	profiles, err := settings.LoadConnections(path)
	if err != nil || len(profiles) != 1 || !profiles[0].SavedPassword {
		t.Fatal("saved profile fixture missing", err)
	}
	publicBefore, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	secretPath := credentials.Path(s.prefsPath, path)
	secretBefore, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := settings.BeginConnectionsUpdate(path, profiles)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("new password must stay in memory")
	s.saveConnection()
	if !strings.Contains(s.connectForm.GetTitle(), s.tr("connections_busy")) {
		t.Fatal("busy profile writer was ignored")
	}
	s.passwordManager()
	s.vaultForm.GetButton(3).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(tview.Primitive) {})
	modalChoice(s, s.programModal, 0)
	if !strings.Contains(s.vaultForm.GetTitle(), s.tr("connections_busy")) {
		t.Fatal("reset ignored another profile writer")
	}
	publicAfter, publicErr := os.ReadFile(path)
	secretAfter, secretErr := os.ReadFile(secretPath)
	if publicErr != nil || secretErr != nil || !bytes.Equal(publicBefore, publicAfter) || !bytes.Equal(secretBefore, secretAfter) {
		t.Fatal("busy save/reset changed public or encrypted data")
	}
}

func TestCredentialConflictRestoresProfilesWithoutLosingOtherSecrets(t *testing.T) {
	s, _ := savedConnectionScreen(t)
	path := settings.ConnectionsPath(s.prefsPath)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	secretPath := credentials.Path(s.prefsPath, path)
	other, err := credentials.Open(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(other.Lock)
	if err := other.Unlock([]byte("master password for conflict tests")); err != nil {
		t.Fatal(err)
	}
	id, err := settings.NewConnectionID()
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Put(id, "other-binding", "other-instance-secret"); err != nil {
		t.Fatal(err)
	}
	secretsBefore, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("new-secret-never-written")
	s.saveConnection()
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) || !strings.Contains(s.connectForm.GetTitle(), s.tr("vault_changed")) {
		t.Fatal("credential failure did not restore public metadata", err)
	}
	secretsAfter, err := os.ReadFile(secretPath)
	if err != nil || !bytes.Equal(secretsBefore, secretsAfter) || bytes.Contains(after, []byte("new-secret")) {
		t.Fatal("failed recovery lost another instance's encrypted secret", err)
	}
	if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "new-secret-never-written" {
		t.Fatal("failed save discarded typed password")
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatal("failed credential update leaked profile lock", err)
	}
}
