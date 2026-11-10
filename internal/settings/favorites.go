package settings

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const favoritesFileLimit = 16 << 20

type Favorite struct {
	Name string `json:"name"`
	Tags string `json:"tags,omitempty"`
	SQL  string `json:"sql"`
}

var (
	ErrInvalidFavorite  = errors.New("invalid SQL favorite")
	ErrFavoritesChanged = errors.New("SQL favorites changed; reopen the list before saving")
)

func FavoritesPath(prefsPath string) string {
	if prefsPath == "" {
		return ""
	}
	name := "sql-favorites.json"
	if strings.EqualFold(filepath.Base(prefsPath), name) {
		name = "saved-sql.json"
	}
	return filepath.Join(filepath.Dir(prefsPath), name)
}

func validateFavorites(items []Favorite) error {
	if len(items) > 200 {
		return ErrInvalidFavorite
	}
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" || utf8.RuneCountInString(item.Name) > 40 ||
			!utf8.ValidString(item.Tags) || utf8.RuneCountInString(item.Tags) > 80 ||
			strings.TrimSpace(item.SQL) == "" || !utf8.ValidString(item.SQL) || len(item.SQL) > 64<<10 {
			return ErrInvalidFavorite
		}
	}
	return nil
}

func LoadFavorites(path string) ([]Favorite, error) {
	if path == "" {
		return nil, errors.New("SQL favorites path is unavailable")
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, favoritesFileLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > favoritesFileLimit {
		return nil, ErrInvalidFavorite
	}
	var items []Favorite
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, validateFavorites(items)
}

func SaveFavorites(path string, expected, items []Favorite) error {
	if path == "" {
		return errors.New("SQL favorites path is unavailable")
	}
	if err := validateFavorites(items); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	lockPath := path + ".lock"
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(lockPath)
	if err := lock.Close(); err != nil {
		return err
	}
	current, err := LoadFavorites(path)
	if err != nil {
		return err
	}
	if !slices.Equal(current, expected) {
		return ErrFavoritesChanged
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	if len(data)+1 > favoritesFileLimit {
		return ErrInvalidFavorite
	}
	return saveJSON(path, items)
}
