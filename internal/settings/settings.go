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
	Language string `json:"language"`
	Mouse    bool   `json:"mouse"`
}

func Default() Config {
	language := "en"
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if locale := os.Getenv(key); locale != "" {
			if strings.HasPrefix(strings.ToLower(locale), "ko") {
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
	if c.Language != "en" && c.Language != "ko" {
		return errors.New("language must be en or ko")
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
