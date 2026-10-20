package settings

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Connection stores only public connection metadata. Passwords live exclusively
// in the separate encrypted credentials store, associated by stable IDs.
type Connection struct {
	ID            string `json:"id,omitempty"`
	SavedPassword bool   `json:"saved_password,omitempty"`
	Alias         string `json:"alias"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Mode          string `json:"mode"`
	Name          string `json:"name"`
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

func NewConnectionID() (string, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	return hex.EncodeToString(id), nil
}

// Binding prevents using an encrypted password for an edited host or account.
func (c Connection) Binding() string {
	data, _ := json.Marshal([]any{c.Host, c.Port, c.User, c.Mode, c.Name})
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (c Connection) Validate() error {
	if c.ID != "" {
		id, err := hex.DecodeString(c.ID)
		if err != nil || len(id) != 16 {
			return errors.New("invalid connection ID")
		}
	}
	if c.SavedPassword && c.ID == "" {
		return errors.New("saved credentials require a connection ID")
	}
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
	ids := make(map[string]bool)
	for _, c := range profiles {
		if err := c.Validate(); err != nil {
			return err
		}
		if c.Alias != strings.TrimSpace(c.Alias) || seen[c.Alias] {
			return errors.New("connection aliases must be unique and have no surrounding spaces")
		}
		seen[c.Alias] = true
		if c.ID != "" {
			if ids[c.ID] {
				return errors.New("connection IDs must be unique")
			}
			ids[c.ID] = true
		}
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
