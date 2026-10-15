package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Connection intentionally has no password or DSN field. Secret storage must
// use a separate design with explicit key management before being introduced.
type Connection struct {
	Alias string `json:"alias"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
	User  string `json:"user"`
	Mode  string `json:"mode"`
	Name  string `json:"name"`
}

func ConnectionsPath(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	path := filepath.Join(filepath.Dir(settingsPath), "connections.json")
	if path == filepath.Clean(settingsPath) {
		return filepath.Join(filepath.Dir(settingsPath), "connection-profiles.json")
	}
	return path
}

func (c Connection) Validate() error {
	if strings.TrimSpace(c.Alias) == "" || strings.TrimSpace(c.Host) == "" || strings.TrimSpace(c.User) == "" || strings.TrimSpace(c.Name) == "" {
		return errors.New("alias, host, username and service name/SID are required")
	}
	if c.Port < 1 || c.Port > 65535 || (c.Mode != "service" && c.Mode != "sid") {
		return errors.New("invalid connection port or mode")
	}
	return nil
}

func validateConnections(profiles []Connection) error {
	seen := make(map[string]bool)
	for _, c := range profiles {
		if err := c.Validate(); err != nil {
			return err
		}
		if c.Alias != strings.TrimSpace(c.Alias) || seen[c.Alias] {
			return errors.New("connection aliases must be unique and have no surrounding spaces")
		}
		seen[c.Alias] = true
	}
	return nil
}

func LoadConnections(path string) ([]Connection, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var profiles []Connection
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("read connections: %w", err)
	}
	if err := validateConnections(profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func SaveConnections(path string, profiles []Connection) error {
	if err := validateConnections(profiles); err != nil {
		return err
	}
	return saveJSON(path, profiles)
}
