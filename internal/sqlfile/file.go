// Package sqlfile reads UTF-8 SQL documents and saves them without truncating
// existing files on failure. Text is LF internally; original BOM/EOL is retained.
package sqlfile

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const MaxBytes = 16 * 1024 * 1024

var (
	ErrEncoding   = errors.New("SQL files must be UTF-8 text without NUL bytes")
	ErrTooLarge   = errors.New("SQL file exceeds the 16 MiB limit")
	ErrNotRegular = errors.New("select a regular file, not a directory or symbolic link")
	ErrChanged    = errors.New("file changed outside LazyOra")
	ErrBusy       = errors.New("file is being saved by another LazyOra instance")
)

type Format struct{ BOM, CRLF bool }
type Snapshot struct {
	Exists bool
	digest [32]byte
	mode   os.FileMode
}
type Document struct {
	Path, Text string
	Format     Format
	Snapshot   Snapshot
}

func Resolve(path, base string) (string, error) {
	if path == "" {
		return "", errors.New("enter a file path")
	}
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	return filepath.Abs(path)
}
func contents(path string) ([]byte, Snapshot, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, Snapshot{}, nil
	}
	if err != nil {
		return nil, Snapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return nil, Snapshot{}, ErrNotRegular
	}
	if info.Size() > MaxBytes {
		return nil, Snapshot{}, ErrTooLarge
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, Snapshot{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return nil, Snapshot{}, err
	}
	if len(data) > MaxBytes {
		return nil, Snapshot{}, ErrTooLarge
	}
	return data, Snapshot{true, sha256.Sum256(data), info.Mode().Perm()}, nil
}
func Inspect(path string) (Snapshot, error) { _, snapshot, err := contents(path); return snapshot, err }
func Read(path string) (*Document, error) {
	data, snapshot, err := contents(path)
	if err != nil {
		return nil, err
	}
	if !snapshot.Exists {
		return nil, os.ErrNotExist
	}
	format := Format{}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		format.BOM = true
		data = data[3:]
	}
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, ErrEncoding
	}
	if newline := bytes.IndexByte(data, '\n'); newline > 0 {
		format.CRLF = data[newline-1] == '\r'
	}
	return &Document{Path: path, Text: strings.ReplaceAll(string(data), "\r\n", "\n"), Format: format, Snapshot: snapshot}, nil
}
func (d *Document) Dirty(text string) bool { return d != nil && d.Text != text }

func Save(path, text string, format Format, expected Snapshot) (*Document, error) {
	if !utf8.ValidString(text) || strings.IndexByte(text, 0) >= 0 {
		return nil, ErrEncoding
	}
	data := []byte(text)
	if format.CRLF {
		data = []byte(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n"))
	}
	if format.BOM {
		data = append([]byte{0xef, 0xbb, 0xbf}, data...)
	}
	if len(data) > MaxBytes {
		return nil, ErrTooLarge
	}
	// Cooperative lock plus snapshot checks prevent a second LazyOra instance
	// or an already-observed external edit from being overwritten silently.
	lock, err := os.OpenFile(path+".lazyora.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil, ErrBusy
	}
	if err != nil {
		return nil, err
	}
	lock.Close()
	defer os.Remove(path + ".lazyora.lock")
	current, err := Inspect(path)
	if err != nil {
		return nil, err
	}
	if current != expected {
		return nil, ErrChanged
	}
	mode := os.FileMode(0600)
	if current.Exists {
		mode = current.mode
		if mode&0222 == 0 {
			return nil, os.ErrPermission
		}
		// A writable directory alone is not permission to overwrite a read-only file.
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".lazyora-sql-*.tmp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(mode); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	latest, err := Inspect(path)
	if err != nil {
		return nil, err
	}
	if latest != expected {
		return nil, ErrChanged
	}
	if err = os.Rename(file.Name(), path); err != nil {
		return nil, err
	}
	snapshot := Snapshot{true, sha256.Sum256(data), mode}
	return &Document{Path: path, Text: text, Format: format, Snapshot: snapshot}, nil
}
