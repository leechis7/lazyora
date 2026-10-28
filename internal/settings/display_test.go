package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayModeBackwardsCompatibilityAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"language":"ko","mouse":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil || c.DisplayMode != "" || c.Language != "ko" {
		t.Fatal("old settings did not default to auto", err)
	}
	for _, mode := range []string{"auto", "unicode", "ascii"} {
		c.DisplayMode = mode
		if err := Save(path, c); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path)
		if err != nil || loaded != c {
			t.Fatal("mode did not round trip", mode, err)
		}
	}
	c.DisplayMode = "invalid"
	if err := Save(path, c); err == nil {
		t.Fatal("invalid display mode saved")
	}
	previous, err := Load(path)
	if err != nil || previous.DisplayMode != "ascii" {
		t.Fatal("rejected save changed original settings")
	}
}
