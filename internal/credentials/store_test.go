package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const master = "a long master passphrase!"

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "credentials.enc"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Create([]byte(master)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Lock)
	return s
}
func TestStoreLifecycleAndNoPlaintext(t *testing.T) {
	s := newStore(t)
	secret := "한글-DB-secret-123!"
	if err := s.Put("profile", "host/account", secret); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(s.path)
	for _, plain := range []string{master, secret, "host/account", "profile"} {
		if bytes.Contains(data, []byte(plain)) {
			t.Fatal("plaintext found on disk")
		}
	}
	info, _ := os.Stat(s.path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("not private")
	}
	// Nonces are fresh on each write, even with identical plaintext.
	if err := s.Put("profile", "host/account", secret); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(s.path)
	if bytes.Equal(data, again) {
		t.Fatal("reused nonce")
	}
	s.Lock()
	if _, _, err := s.Get("profile", "host/account"); !errors.Is(err, ErrLocked) {
		t.Fatal(err)
	}
	reopened, err := Open(s.path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Lock()
	if err = reopened.Unlock([]byte("wrong password")); !errors.Is(err, ErrUnlock) || reopened.Unlocked() {
		t.Fatal("wrong password accepted")
	}
	if err = reopened.Unlock([]byte(master)); err != nil {
		t.Fatal(err)
	}
	got, found, err := reopened.Get("profile", "host/account")
	if err != nil || !found || got != secret {
		t.Fatal("round trip failed")
	}
	if _, found, err = reopened.Get("profile", "other-host/account"); err != nil || found {
		t.Fatal("password escaped its binding")
	}
	if err = reopened.ChangeMaster([]byte("different master passphrase")); err != nil {
		t.Fatal(err)
	}
	reopened.Lock()
	if err = reopened.Unlock([]byte(master)); !errors.Is(err, ErrUnlock) {
		t.Fatal("old master still works")
	}
	if err = reopened.Unlock([]byte("different master passphrase")); err != nil {
		t.Fatal(err)
	}
	if err = reopened.Delete("profile"); err != nil {
		t.Fatal(err)
	}
	if _, found, err = reopened.Get("profile", "host/account"); err != nil || found {
		t.Fatal("delete failed")
	}
	if err = reopened.Reset(); err != nil {
		t.Fatal(err)
	}
	if reopened.Exists() || reopened.Unlocked() {
		t.Fatal("reset did not lock")
	}
	if _, err = os.Stat(s.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("reset kept file")
	}
}

func TestCorruptionAndKDFBoundsPreserveOriginal(t *testing.T) {
	s := newStore(t)
	original, _ := os.ReadFile(s.path)
	var e envelope
	if err := json.Unmarshal(original, &e); err != nil {
		t.Fatal(err)
	}
	e.Memory = ^uint32(0)
	bad, _ := json.Marshal(e)
	if err := os.WriteFile(s.path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	corrupt, err := Open(s.path)
	if !errors.Is(err, ErrInvalid) {
		t.Fatal("unbounded KDF parameters accepted")
	}
	if err = corrupt.Unlock([]byte(master)); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.path)
	if !bytes.Equal(bad, after) {
		t.Fatal("corruption overwritten")
	}
	if err = corrupt.Reset(); err != nil {
		t.Fatal("cannot explicitly reset corrupt vault", err)
	}
	if err = os.WriteFile(s.path, original, 0600); err != nil {
		t.Fatal(err)
	}
	e.Memory = memoryKiB
	e.Ciphertext[len(e.Ciphertext)-1] ^= 1
	tampered, _ := json.Marshal(e)
	if err = os.WriteFile(s.path, tampered, 0600); err != nil {
		t.Fatal(err)
	}
	corrupt, err = Open(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if err = corrupt.Unlock([]byte(master)); !errors.Is(err, ErrUnlock) {
		t.Fatal("tampering accepted", err)
	}
	after, _ = os.ReadFile(s.path)
	if !bytes.Equal(tampered, after) {
		t.Fatal("tampered file overwritten")
	}
}

func TestConcurrentInstancesAndFailedWrites(t *testing.T) {
	s := newStore(t)
	second, err := Open(s.path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Lock()
	if err = second.Unlock([]byte(master)); err != nil {
		t.Fatal(err)
	}
	if err = s.Put("a", "binding", "first"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.path)
	if err = second.Put("b", "binding", "second"); !errors.Is(err, ErrChanged) {
		t.Fatal("stale writer accepted", err)
	}
	if _, _, err = second.Get("a", "binding"); !errors.Is(err, ErrChanged) {
		t.Fatal("stale reader accepted", err)
	}
	after, _ := os.ReadFile(s.path)
	if !bytes.Equal(before, after) {
		t.Fatal("stale writer lost data")
	}
	if err = os.WriteFile(s.path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err = s.Put("b", "binding", "second"); !errors.Is(err, ErrBusy) {
		t.Fatal("writer lock ignored", err)
	}
	after, _ = os.ReadFile(s.path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed write changed data")
	}
	if err = os.Remove(s.path + ".lock"); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeMaster([]byte("short")); !errors.Is(err, ErrWeak) {
		t.Fatal("weak master accepted")
	}
	if v, ok, err := s.Get("a", "binding"); err != nil || !ok || v != "first" {
		t.Fatal("failed change lost original key")
	}
}

func TestPathsAndUnsafeFiles(t *testing.T) {
	dir := t.TempDir()
	prefs := filepath.Join(dir, "credentials.enc")
	path := Path(prefs, filepath.Join(dir, "connections.json"))
	if path == prefs || Path("", "") != "" {
		t.Fatal("preference collision")
	}
	if runtime.GOOS != "windows" {
		target := filepath.Join(dir, "target")
		if err := os.WriteFile(target, []byte("secret"), 0600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(link); !errors.Is(err, ErrPermissions) {
			t.Fatal("symlink accepted")
		}
		if err := os.Chmod(target, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(target); !errors.Is(err, ErrPermissions) {
			t.Fatal("public permissions accepted")
		}
	}
}

func TestMasterMinimumIsEightCharacters(t *testing.T) {
	for _, value := range []string{"12345678", "가나다라마바사아"} {
		if err := CheckMaster([]byte(value)); err != nil {
			t.Errorf("eight characters rejected: %v", err)
		}
	}
	for _, value := range []string{"1234567", "가나다라마바사", "        "} {
		if err := CheckMaster([]byte(value)); !errors.Is(err, ErrWeak) {
			t.Error("short or blank master accepted")
		}
	}
}
