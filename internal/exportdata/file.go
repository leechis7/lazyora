package exportdata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrChanged = errors.New("destination changed; retry and review overwrite")

type Revision struct {
	Exists   bool
	Size     int64
	Modified time.Time
	Mode     os.FileMode
}

func Inspect(path string) (Revision, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Revision{}, nil
	}
	if err != nil {
		return Revision{}, err
	}
	if !info.Mode().IsRegular() {
		return Revision{}, fmt.Errorf("select a regular file, not a directory or symbolic link")
	}
	return Revision{true, info.Size(), info.ModTime(), info.Mode().Perm()}, nil
}

// Save writes a temporary file beside the destination and replaces it only on
// complete success. Revision checks detect observed external changes; the lock
// cooperates with both SQL saves and other exports from LazyOra.
func Save(ctx context.Context, path string, expected Revision, data Dataset, options Options) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	lock, err := os.OpenFile(path+".lazyora.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return 0, err
	}
	lock.Close()
	defer os.Remove(path + ".lazyora.lock")
	current, err := Inspect(path)
	if err != nil {
		return 0, err
	}
	if current != expected {
		return 0, ErrChanged
	}
	mode := os.FileMode(0600)
	if current.Exists {
		mode = current.Mode
		if mode&0222 == 0 {
			return 0, os.ErrPermission
		}
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return 0, err
		}
		if err = file.Close(); err != nil {
			return 0, err
		}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".lazyora-export-*.tmp")
	if err != nil {
		return 0, err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(mode); err == nil {
		err = Write(ctx, file, data, options)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	latest, err := Inspect(path)
	if err != nil {
		return 0, err
	}
	if latest != expected {
		return 0, ErrChanged
	}
	if err = ctx.Err(); err != nil {
		return 0, err
	}
	if err = os.Rename(file.Name(), path); err != nil {
		return 0, err
	}
	return len(data.Rows), nil
}
