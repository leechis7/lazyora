package ui

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// UI backend access belongs to the published SDK, including tests. Do not
// reintroduce a local shim when new UI code needs another backend capability.
func TestUIUsesPublishedTuivoryEntryPoint(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "third_party", "vendor", "bin":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			name, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if name == "github.com/rivo/tview" || strings.HasPrefix(name, "github.com/rivo/tview/") || name == "github.com/gdamore/tcell/v2" || strings.HasPrefix(name, "github.com/gdamore/tcell/v2/") || strings.HasPrefix(name, "github.com/leechis7/tuivory/go/") {
				t.Errorf("%s imports %s; use the published Tuivory root facade", path, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
