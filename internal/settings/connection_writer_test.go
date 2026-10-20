package settings

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

func testProfile(alias string) Connection {
	return Connection{Alias: alias, Host: "localhost", Port: 1521, User: "app", Mode: "service", Name: "FREEPDB1"}
}

func TestConnectionsRejectStaleSnapshotAndReleaseLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	base := []Connection{testProfile("first")}
	if err := SaveConnections(path, nil, base); err != nil {
		t.Fatal(err)
	}
	added := append(slices.Clone(base), testProfile("second"))
	if err := SaveConnections(path, base, added); err != nil {
		t.Fatal(err)
	}
	stale := slices.Clone(base)
	stale[0].Host = "edited-by-stale-instance"
	if err := SaveConnections(path, base, stale); !errors.Is(err, ErrConnectionsChanged) {
		t.Fatalf("stale update accepted: %v", err)
	}
	got, err := LoadConnections(path)
	if err != nil || !slices.Equal(got, added) {
		t.Fatal("stale writer lost another instance's update", err)
	}
	got[0].Host = "reviewed-edit"
	if err := SaveConnections(path, added, got); err != nil {
		t.Fatal("conflict leaked the writer lock", err)
	}
}

func TestConnectionsLockCoversIntermediateAndRecoveryWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	base := []Connection{testProfile("first")}
	writer, err := BeginConnectionsUpdate(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := writer.Save(base); err != nil {
		t.Fatal(err)
	}
	// Simulate a credential failure and rollback while a competitor observes
	// the intermediate public metadata. Its snapshot must never be writable.
	for _, snapshot := range [][]Connection{nil, base} {
		if err := SaveConnections(path, snapshot, []Connection{testProfile("competitor")}); !errors.Is(err, ErrConnectionsBusy) {
			t.Fatalf("multi-file update allowed a competing writer: %v", err)
		}
	}
	if err := writer.Save(nil); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Save(base); err == nil {
		t.Fatal("closed writer could bypass the process lock")
	}
	if err := SaveConnections(path, base, base); !errors.Is(err, ErrConnectionsChanged) {
		t.Fatal("competitor's intermediate snapshot was accepted after rollback", err)
	}
	if err := SaveConnections(path, nil, base); err != nil {
		t.Fatal(err)
	}
}

func TestConnectionsConcurrentAddsPreserveAllProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	var workers sync.WaitGroup
	for i := range 12 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				base, err := LoadConnections(path)
				if err != nil {
					t.Error(err)
					return
				}
				next := append(slices.Clone(base), testProfile(fmt.Sprint(i)))
				err = SaveConnections(path, base, next)
				if err == nil {
					return
				}
				if !errors.Is(err, ErrConnectionsBusy) && !errors.Is(err, ErrConnectionsChanged) {
					t.Error(err)
					return
				}
				time.Sleep(time.Millisecond)
			}
			t.Error("timed out retrying concurrent update")
		}()
	}
	workers.Wait()
	got, err := LoadConnections(path)
	if err != nil || len(got) != 12 {
		t.Fatalf("concurrent updates lost profiles: count=%d err=%v", len(got), err)
	}
}

func TestConnectionWriterProcessLock(t *testing.T) {
	if path := os.Getenv("LAZYORA_TEST_CONNECTION_LOCK"); path != "" {
		if _, err := BeginConnectionsUpdate(path, nil); !errors.Is(err, ErrConnectionsBusy) {
			t.Fatal("other process bypassed lock", err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "connections.json")
	writer, err := BeginConnectionsUpdate(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestConnectionWriterProcessLock$")
	cmd.Env = append(os.Environ(), "LAZYORA_TEST_CONNECTION_LOCK="+path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("process lock: %v\n%s", err, output)
	}
}

func TestConnectionsFailuresKeepDataAndCleanLocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	if err := os.WriteFile(path, []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BeginConnectionsUpdate(path, nil); err == nil {
		t.Fatal("malformed file accepted")
	}
	if _, err := os.Stat(path + ".lock"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read failure leaked lock")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	bad := testProfile("duplicate")
	if err := SaveConnections(path, nil, []Connection{bad, bad}); err == nil {
		t.Fatal("invalid profiles written")
	}
	if err := SaveConnections(path, nil, []Connection{testProfile("valid")}); err != nil {
		t.Fatal("validation failure leaked lock", err)
	}
	if _, err := BeginConnectionsUpdate("", nil); err == nil {
		t.Fatal("empty path accepted")
	}
}
