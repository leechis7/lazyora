package exportdata

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

type fullDiskWriter struct {
	io.Writer
	remaining int
}

func (w *fullDiskWriter) Write(p []byte) (int, error) {
	if len(p) <= w.remaining {
		n, err := w.Writer.Write(p)
		w.remaining -= n
		return n, err
	}
	n, err := w.Writer.Write(p[:w.remaining])
	w.remaining -= n
	if err != nil {
		return n, err
	}
	return n, &os.PathError{Op: "write", Path: "temporary export", Err: syscall.ENOSPC}
}

func TestDiskFullPreservesDestinationForEveryFormat(t *testing.T) {
	for _, format := range []string{"csv", "tsv", "sql", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "saved."+format)
			original := []byte("existing result must survive")
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			revision, err := Inspect(path)
			if err != nil {
				t.Fatal(err)
			}
			// Inject ENOSPC after a real partial write, including compressed
			// XLSX output and delayed buffered flushes. No real disk is filled.
			_, err = SaveContent(context.Background(), path, revision, func(writer io.Writer) (int64, error) {
				return WriteStream(context.Background(), &fullDiskWriter{Writer: writer, remaining: 64}, generatedSource(2500, strings.Repeat("한글", 20)), Options{Format: format, Table: "TARGET", Unlimited: true})
			})
			if !errors.Is(err, syscall.ENOSPC) {
				t.Fatalf("disk full hidden: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != string(original) {
				t.Fatal("partial export replaced original", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
				t.Fatal("failed export leaked temporary file or lock", err)
			}
			latest, err := Inspect(path)
			if err != nil || latest != revision {
				t.Fatal("failed export changed original revision", err)
			}
		})
	}
}
