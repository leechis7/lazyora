package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Theme        string `json:"theme,omitempty"`
	Language     string `json:"language"`
	Mouse        bool   `json:"mouse"`
	EditorKeymap string `json:"editor_keymap,omitempty"`
}

func Default() Config {
	language := "en"
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale := strings.TrimSpace(os.Getenv(key)); locale != "" {
			base := strings.FieldsFunc(strings.ToLower(locale), func(r rune) bool { return r == '_' || r == '-' || r == '.' || r == '@' })
			if len(base) > 0 && base[0] == "ko" {
				language = "ko"
			}
			break
		}
	}
	return Config{Language: language, Mouse: true}
}

func Path() (string, error) {
	if path := os.Getenv("LAZYORA_CONFIG"); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazyora", "settings.json"), nil
}

func (c Config) Validate() error {
	if c.Theme != "" && c.Theme != "dark" && c.Theme != "light" {
		return errors.New("theme must be dark or light")
	}
	if c.Language != "en" && c.Language != "ko" {
		return errors.New("language must be en or ko")
	}
	if c.EditorKeymap != "" && c.EditorKeymap != "vim" && c.EditorKeymap != "notepad++" {
		return errors.New("editor_keymap must be empty (default), vim or notepad++")
	}
	return nil
}

func Load(path string) (Config, error) {
	defaults := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	c := defaults
	if err = json.Unmarshal(data, &c); err != nil {
		return defaults, fmt.Errorf("read settings: %w", err)
	}
	if c.Language == "" {
		c.Language = defaults.Language
	}
	if err = c.Validate(); err != nil {
		return defaults, err
	}
	return c, nil
}

func Save(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	return saveJSON(path, c)
}

func saveJSON(path string, value any) error {
	if path == "" {
		return errors.New("settings path is unavailable")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
