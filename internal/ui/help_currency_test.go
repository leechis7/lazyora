package ui

import (
	"strings"
	"testing"

	"github.com/rivo/tview"
)

// New menu commands must be documented, including actions without a shortcut.
func TestHelpCoversEveryMenuCommandAndCurrentBinding(t *testing.T) {
	covered := map[string]bool{}
	for _, section := range helpSections {
		for _, id := range section.commands {
			covered[id] = true
		}
	}
	for _, language := range []string{"en", "ko"} {
		for _, keymap := range []string{"", "vim", "notepad++", "vscode", "intellij"} {
			s := &screen{}
			s.prefs.Language, s.prefs.EditorKeymap = language, keymap
			for _, command := range commands {
				if !covered[command.id] {
					t.Errorf("%s is missing from help topics", command.id)
				}
				rows := tview.TranslateANSI(s.helpCommands([]string{command.id}))
				if key := s.editorBindingLabels(command.id, 2); key != "" {
					if !strings.Contains(rows, key) {
						t.Errorf("%s does not show its current binding %s", command.id, key)
					}
				} else {
					path := s.tr(menuGroups[command.group].label) + " > " + s.tr(command.label)
					if !strings.Contains(rows, tview.Escape(path)) {
						t.Errorf("%s has no keyboard binding and no menu path", command.id)
					}
				}
			}
			help := s.mainHelp()
			for _, text := range []string{"F10 / Alt+M", "Shift+F10", "Esc", "Alt+T", "TERM=ansi", "TERM=xterm"} {
				if !strings.Contains(help, text) {
					t.Errorf("%s help is missing current instructions %s", language, text)
				}
			}
		}
	}
}

func TestFullscreenHelpContextMentionsEscapeRestore(t *testing.T) {
	s, _ := resultScreen(t)
	s.togglePanelFullscreen("sql")
	if !strings.Contains(s.helpContext("main"), s.tr("restore_split")) {
		t.Fatal("fullscreen context does not explain Esc restore")
	}
}
