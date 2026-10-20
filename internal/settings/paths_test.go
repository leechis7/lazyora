package settings_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lazyora/internal/credentials"
	"lazyora/internal/settings"
)

func TestConnectionsPathCaseIsolation(t *testing.T) {
	for _, name := range []string{"connections.json", "Connections.json", "CONNECTIONS.JSON", "cOnNeCtIoNs.JsOn"} {
		path := filepath.Join(t.TempDir(), name)
		want := filepath.Join(filepath.Dir(path), "connection-profiles.json")
		// Check both before and after creation: the selected path must be stable.
		if got := settings.ConnectionsPath(path); got != want {
			t.Fatalf("%s: got %s want %s", name, got, want)
		}
		if err := settings.Save(path, settings.Config{Language: "ko", Mouse: true}); err != nil {
			t.Fatal(err)
		}
		if got := settings.ConnectionsPath(path); got != want {
			t.Fatal("creating preferences changed the profile path")
		}
	}
	if settings.ConnectionsPath("") != "" || settings.ConnectionsPath(filepath.Join("dir", "settings.json")) != filepath.Join("dir", "connections.json") {
		t.Fatal("default path behavior changed")
	}
}

func TestConnectionFilesStaySeparate(t *testing.T) {
	for _, name := range []string{"settings.json", "connections.json", "Connections.json", "CONNECTIONS.JSON", "connection-profiles.json", "credentials.enc", "CREDENTIALS.ENC", "secret-credentials.enc"} {
		t.Run(name, func(t *testing.T) {
			prefsPath := filepath.Join(t.TempDir(), name)
			config := settings.Config{Language: "ko", Mouse: true}
			if err := settings.Save(prefsPath, config); err != nil {
				t.Fatal(err)
			}
			prefs, err := os.ReadFile(prefsPath)
			if err != nil {
				t.Fatal(err)
			}
			profilePath := settings.ConnectionsPath(prefsPath)
			secretPath := credentials.Path(prefsPath, profilePath)
			if strings.EqualFold(profilePath, prefsPath) || strings.EqualFold(secretPath, prefsPath) || strings.EqualFold(secretPath, profilePath) {
				t.Fatal("case-insensitive path collision")
			}
			profile := settings.Connection{Alias: "test", Host: "localhost", Port: 1521, User: "app", Mode: "service", Name: "FREEPDB1"}
			base := []settings.Connection{profile}
			if err := settings.SaveConnections(profilePath, nil, base); err != nil {
				t.Fatal(err)
			}
			public, err := os.ReadFile(profilePath)
			if err != nil {
				t.Fatal(err)
			}
			store, err := credentials.Open(secretPath)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Lock()
			if err := store.Create([]byte("path isolation master password")); err != nil {
				t.Fatal(err)
			}
			secret, err := os.ReadFile(secretPath)
			if err != nil {
				t.Fatal(err)
			}
			for path, want := range map[string][]byte{prefsPath: prefs, profilePath: public} {
				if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
					t.Fatal("profile/credential creation overwrote an existing file", err)
				}
			}
			// Settings updates must preserve profiles and encrypted passwords too.
			config.Mouse = false
			if err := settings.Save(prefsPath, config); err != nil {
				t.Fatal(err)
			}
			for path, want := range map[string][]byte{profilePath: public, secretPath: secret} {
				if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
					t.Fatal("preference save overwrote profile/credential data", err)
				}
			}
			if got, err := settings.Load(prefsPath); err != nil || got != config {
				t.Fatal("preferences no longer readable", err)
			}
			profile.Alias = "updated"
			if err := settings.SaveConnections(profilePath, base, []settings.Connection{profile}); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(secretPath); err != nil || !bytes.Equal(got, secret) {
				t.Fatal("profile update overwrote credentials", err)
			}
		})
	}
}
