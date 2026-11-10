package settings

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestFavoritesRejectStaleUpdates(t *testing.T) {
	path := FavoritesPath(filepath.Join(t.TempDir(), "sql-favorites.json"))
	original := []Favorite{{Name: "조회", Tags: "월간", SQL: "SELECT :account FROM dual;"}}
	if err := SaveFavorites(path, nil, original); err != nil {
		t.Fatal(err)
	}
	if err := SaveFavorites(path, nil, nil); !errors.Is(err, ErrFavoritesChanged) {
		t.Fatalf("stale writer was not rejected: %v", err)
	}
	if current, err := LoadFavorites(path); err != nil || len(current) != 1 || current[0] != original[0] {
		t.Fatalf("favorite was lost: %v, %v", current, err)
	}
}
