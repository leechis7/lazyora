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

func TestLocaleLanguageSelectionAndPriority(t *testing.T) {
	for _, tc := range []struct{ name, all, messages, lang, want string }{
		{"unset", "", "", "", "en"},
		{"korean lang", "", "", "ko_KR.UTF-8", "ko"},
		{"plain ko", "", "", "ko", "ko"},
		{"hyphen", "", "", "ko-KR", "ko"},
		{"case and spaces", "", "", " KO_kr.UTF-8 ", "ko"},
		{"messages priority", "", "ko_KR.UTF-8", "en_US.UTF-8", "ko"},
		{"all priority", "C", "ko_KR.UTF-8", "ko_KR.UTF-8", "en"},
		{"whitespace unset", " ", "", "ko_KR.UTF-8", "ko"},
		{"other language", "", "", "ja_JP.UTF-8", "en"},
		{"not ko prefix", "", "", "kok_IN.UTF-8", "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LC_ALL", tc.all)
			t.Setenv("LC_MESSAGES", tc.messages)
			t.Setenv("LANG", tc.lang)
			if got := Default().Language; got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestUnsetSavedLanguageUsesLocaleWithoutError(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	path := filepath.Join(t.TempDir(), "settings.json")
	for _, lang := range []string{"ko_KR.UTF-8", "en_US.UTF-8"} {
		t.Setenv("LANG", lang)
		want := Default().Language
		for _, data := range []string{`{}`, `{"mouse":false}`, `{"language":"","mouse":false}`, `{"language":null,"mouse":false}`} {
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := Load(path)
			if err != nil || got.Language != want {
				t.Fatalf("unset language fallback failed: %+v %v", got, err)
			}
			if data != `{}` && got.Mouse {
				t.Fatal("locale fallback discarded saved mouse preference")
			}
		}
	}
	// Explicit preferences take precedence over the locale in either direction.
	for _, saved := range []string{"en", "ko"} {
		locale := "ko_KR.UTF-8"
		if saved == "ko" {
			locale = "en_US.UTF-8"
		}
		t.Setenv("LANG", locale)
		if err := Save(path, Config{Language: saved, Mouse: false}); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil || got.Language != saved {
			t.Fatal("locale replaced explicit preference")
		}
	}
	if err := os.WriteFile(path, []byte(`{"language":"invalid"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("invalid explicit language silently accepted")
	}
}
