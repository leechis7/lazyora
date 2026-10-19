package sqlfile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRoundTripKoreanBOMAndLineEndings(t *testing.T) {
	for _, format := range []Format{{}, {BOM: true}, {CRLF: true}, {BOM: true, CRLF: true}} {
		t.Run(strings.Join([]string{map[bool]string{true: "BOM", false: "plain"}[format.BOM], map[bool]string{true: "CRLF", false: "LF"}[format.CRLF]}, "-"), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "한글 쿼리.sql")
			text := "-- 한글 주석\nSELECT '이채강' FROM dual;\n"
			doc, err := Save(path, text, format, Snapshot{})
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := Read(path)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Text != text || loaded.Format != format || loaded.Snapshot != doc.Snapshot || loaded.Dirty(text) || !loaded.Dirty(text+" ") {
				t.Fatalf("round trip: %+v", loaded)
			}
			raw, _ := os.ReadFile(path)
			if bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) != format.BOM || bytes.Contains(raw, []byte("\r\n")) != format.CRLF {
				t.Fatalf("wrong on-disk format: %q", raw)
			}
			updated := text + "SELECT 2 FROM dual;"
			if _, err = Save(path, updated, loaded.Format, loaded.Snapshot); err != nil {
				t.Fatal(err)
			}
			loaded, err = Read(path)
			if err != nil || loaded.Text != updated {
				t.Fatalf("resave: %+v %v", loaded, err)
			}
		})
	}
}

func TestExternalEditsAndDeletionAreNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.sql")
	doc, err := Save(path, "original", Format{}, Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Save(path, "mine", Format{}, doc.Snapshot); !errors.Is(err, ErrChanged) {
		t.Fatalf("expected external edit conflict: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "external" {
		t.Fatal("external edit lost")
	}
	current, _ := Inspect(path)
	if _, err = Save(path, "mine", Format{}, current); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err = Save(path, "mine", Format{}, current); !errors.Is(err, ErrChanged) {
		t.Fatalf("expected deletion conflict: %v", err)
	}
	if _, err = os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("deleted file recreated without confirmation")
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 0 {
		t.Fatal("failed save left temporary files")
	}
}

func TestInvalidFilesAndSaveFailuresPreserveExistingBytes(t *testing.T) {
	for _, invalid := range []string{"\xff", "SELECT\x00", strings.Repeat("x", MaxBytes+1)} {
		path := filepath.Join(t.TempDir(), "invalid.sql")
		if err := os.WriteFile(path, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(path); err == nil {
			t.Fatal("invalid file accepted")
		}
		if _, err := Save(path, invalid, Format{}, Snapshot{}); err == nil {
			t.Fatal("invalid text saved")
		}
		data, _ := os.ReadFile(path)
		if string(data) != invalid {
			t.Fatal("failed write changed bytes")
		}
	}
	path := filepath.Join(t.TempDir(), "query.sql")
	doc, err := Save(path, "original", Format{}, Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path+".lazyora.lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Save(path, "replacement", Format{}, doc.Snapshot); !errors.Is(err, ErrBusy) {
		t.Fatalf("lock ignored: %v", err)
	}
	if err = os.Remove(path + ".lazyora.lock"); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err = os.Chmod(path, 0440); err != nil {
			t.Fatal(err)
		}
		snapshot, _ := Inspect(path)
		if _, err = Save(path, "replacement", Format{}, snapshot); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("read-only file overwritten: %v", err)
		}
		if err = os.Chmod(path, 0640); err != nil {
			t.Fatal(err)
		}
		snapshot, _ = Inspect(path)
		if _, err = Save(path, "updated", Format{}, snapshot); err != nil {
			t.Fatal(err)
		}
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0640 {
			t.Fatal("existing permissions changed")
		}
	}
	if _, err = Save(filepath.Join(path, "missing.sql"), "text", Format{}, Snapshot{}); err == nil {
		t.Fatal("invalid parent accepted")
	}
}

func TestDirectoriesAndSymlinksAreRejected(t *testing.T) {
	dir := t.TempDir()
	if _, err := Read(dir); !errors.Is(err, ErrNotRegular) {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "file.sql")
	if _, err := Save(path, "", Format{}, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.sql")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Read(link); !errors.Is(err, ErrNotRegular) {
		t.Fatal(err)
	}
	if _, err := Save(link, "replace", Format{}, Snapshot{}); !errors.Is(err, ErrNotRegular) {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Fatal("symlink target changed")
	}
}

func TestResolveRelativeAndHomePaths(t *testing.T) {
	dir := t.TempDir()
	got, err := Resolve("한글 query.sql", dir)
	if err != nil || got != filepath.Join(dir, "한글 query.sql") {
		t.Fatalf("%q %v", got, err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err = Resolve("~/query.sql", dir)
	if err != nil || got != filepath.Join(home, "query.sql") {
		t.Fatalf("%q %v", got, err)
	}
	if _, err = Resolve("", dir); err == nil {
		t.Fatal("empty path accepted")
	}
}
