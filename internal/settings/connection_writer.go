package settings

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
)

var (
	ErrConnectionsChanged = errors.New("connections changed; reopen the connection window before saving")
	ErrConnectionsBusy    = errors.New("connections are being saved by another instance; try again later")
)

// ConnectionWriter holds a cooperative process lock across public profile and
// encrypted credential updates, including recovery writes. Close releases only
// the lock: this is not an automatic rollback transaction. Use on one goroutine.
type ConnectionWriter struct {
	path   string
	closed bool
}

// BeginConnectionsUpdate checks that expected still matches the loaded list.
// Callers must close the writer even if a later credential operation fails.
func BeginConnectionsUpdate(path string, expected []Connection) (*ConnectionWriter, error) {
	if path == "" {
		return nil, errors.New("connections path is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil, ErrConnectionsBusy
	}
	if err != nil {
		return nil, err
	}
	w := &ConnectionWriter{path: path}
	if err = file.Close(); err != nil {
		w.Close()
		return nil, err
	}
	current, err := LoadConnections(path)
	if err == nil && !slices.Equal(current, expected) {
		err = ErrConnectionsChanged
	}
	if err != nil {
		w.Close()
		return nil, err
	}
	return w, nil
}

func (w *ConnectionWriter) Save(profiles []Connection) error {
	if w.closed {
		return errors.New("connection writer is closed")
	}
	if err := validateConnections(profiles); err != nil {
		return err
	}
	return saveJSON(w.path, profiles)
}

func (w *ConnectionWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	return os.Remove(w.path + ".lock")
}

// SaveConnections performs a single optimistic update. For multi-step updates,
// use one ConnectionWriter for the entire operation instead of reacquiring it.
func SaveConnections(path string, expected, profiles []Connection) error {
	w, err := BeginConnectionsUpdate(path, expected)
	if err != nil {
		return err
	}
	defer w.Close()
	return w.Save(profiles)
}
