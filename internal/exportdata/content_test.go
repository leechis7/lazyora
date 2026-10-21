package exportdata

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCellContentFailureAndCancelPreserveDestination(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "cell.bin")
		if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
		rev, _ := Inspect(path)
		ctx, cancel := context.WithCancel(context.Background())
		_, err := SaveContent(ctx, path, rev, func(w io.Writer) (int64, error) {
			w.Write([]byte("partial"))
			if cancelled {
				cancel()
				return 7, nil
			}
			return 7, io.ErrUnexpectedEOF
		})
		cancel()
		if err == nil {
			t.Fatal("failed write replaced destination")
		}
		if cancelled && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if string(data) != "original" {
			t.Fatal("destination damaged")
		}
		entries, _ := os.ReadDir(filepath.Dir(path))
		if len(entries) != 1 {
			t.Fatal("temporary file/lock leaked")
		}
	}
}
