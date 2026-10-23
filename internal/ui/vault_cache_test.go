package ui

import (
	"errors"
	"lazyora/internal/credentials"
	"lazyora/internal/settings"
	"path/filepath"
	"testing"
)

func TestVaultUnlockReusedAcrossWorkspacesAndReleasedAtShutdown(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	if err := first.passwordStore(); err != nil {
		t.Fatal(err)
	}
	store := first.vault
	if err := store.Create([]byte("strong master password")); err != nil {
		t.Fatal(err)
	}
	second := secondWorkspace(m)
	called := false
	second.ensurePasswords(func() { called = true })
	if !called || second.masterForm != nil || second.vault != store {
		t.Fatal("workspace asked again for master password")
	}
	second.connectionForm()
	second.closeConnection()
	second.ensurePasswords(func() {})
	if second.masterForm != nil {
		t.Fatal("connection dialog lost unlock")
	}
	second.stopWorker()
	if !store.Unlocked() {
		t.Fatal("closing another workspace locked the process cache")
	}
	m.shutdown()
	if store.Unlocked() {
		t.Fatal("process shutdown retained key")
	}
}
func TestVaultExplicitLockStoreIsolationAndExternalChange(t *testing.T) {
	m, _ := workspaceTestSet(t)
	first := m.active
	if err := first.passwordStore(); err != nil {
		t.Fatal(err)
	}
	store := first.vault
	if err := store.Create([]byte("strong master password")); err != nil {
		t.Fatal(err)
	}
	second := secondWorkspace(m)
	if err := second.passwordStore(); err != nil {
		t.Fatal(err)
	}
	second.lockPasswords()
	if store.Unlocked() || first.vault != nil || second.vault != nil {
		t.Fatal("explicit lock did not clear all references")
	}
	if err := first.passwordStore(); err != nil {
		t.Fatal(err)
	}
	if first.vault.Unlocked() {
		t.Fatal("lock silently reused a key")
	}
	if err := first.vault.Unlock([]byte("strong master password")); err != nil {
		t.Fatal(err)
	}
	second.prefsPath = filepath.Join(t.TempDir(), "other.json")
	if err := second.passwordStore(); err != nil {
		t.Fatal(err)
	}
	if second.vault == first.vault || second.vault.Unlocked() {
		t.Fatal("different store inherited key")
	}
	path := credentials.Path(first.prefsPath, settings.ConnectionsPath(first.prefsPath))
	external, err := credentials.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = external.Unlock([]byte("strong master password")); err != nil {
		t.Fatal(err)
	}
	if err = external.ChangeMaster([]byte("different master password")); err != nil {
		t.Fatal(err)
	}
	external.Lock()
	old := first.vault
	if err = first.passwordStore(); !errors.Is(err, credentials.ErrChanged) || old.Unlocked() || first.vault != nil {
		t.Fatal("external replacement reused stale key", err)
	}
	if err = first.passwordStore(); err != nil || first.vault.Unlocked() {
		t.Fatal("replacement not reopened locked", err)
	}
}
