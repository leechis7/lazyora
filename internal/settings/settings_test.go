package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSavedPreferencesAndInvalidUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lazyora", "settings.json")
	want := Config{Language: "ko", Mouse: false}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != want {
		t.Fatalf("settings were not persisted: %+v, %v", got, err)
	}
	before, _ := os.ReadFile(path)
	if err := Save(path, Config{Language: "unknown"}); err == nil {
		t.Fatal("invalid language was saved")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid save changed existing preferences")
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatal("temporary settings files were left behind")
	}
}

func TestDefaultLocaleAndBrokenSettings(t *testing.T) {
	t.Setenv("LC_ALL", "ko_KR.UTF-8")
	path := filepath.Join(t.TempDir(), "settings.json")
	defaults, err := Load(path)
	if err != nil || defaults.Language != "ko" || !defaults.Mouse {
		t.Fatalf("wrong defaults: %+v, %v", defaults, err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err == nil || got != defaults {
		t.Fatal("broken settings did not return defaults and an error")
	}
	t.Setenv("LC_ALL", "C")
	t.Setenv("LANG", "ko_KR.UTF-8")
	if Default().Language != "en" {
		t.Fatal("LC_ALL did not override LANG")
	}
	t.Setenv("LAZYORA_CONFIG", path)
	if actual, err := Path(); err != nil || actual != path {
		t.Fatal("config path override was ignored")
	}
}
