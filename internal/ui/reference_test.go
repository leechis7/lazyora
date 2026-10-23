package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	resources "lazyora"
	"lazyora/internal/appinfo"
)

func TestReferenceMenuAndOfflineVersionHistory(t *testing.T) {
	s, _ := resultScreen(t)
	var ids []string
	for _, cmd := range commands {
		if cmd.group == menuHelp {
			ids = append(ids, cmd.id)
		}
	}
	if strings.Join(ids, ",") != "help,guide,release_history,diagnostic_info,issue_report,keymap_compare,about" {
		t.Fatalf("help order: %v", ids)
	}
	s.invoke("release_history")
	r := s.reference
	if r == nil || !strings.HasPrefix(r.articles[r.topics.GetCurrentItem()].title, "["+appinfo.Version+"]") {
		t.Fatal("release history did not start at the executable version")
	}
	data, err := resources.Documents.ReadFile("CHANGELOG.md")
	if err != nil || len(r.articles) != len(referenceArticles(string(data))) || len(r.articles) < 5 {
		t.Fatal("embedded history missing versions", err)
	}
	if r.articles[0].title != s.tr("release_unreleased") {
		t.Fatal("development notes not distinguished")
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if !strings.HasPrefix(s.clipboardText, "["+appinfo.Version+"]") || len(s.jobs) != 0 {
		t.Fatal("history copy changed state")
	}
}

func TestReferenceGuidesFocusResizeThemeAndRunningSQL(t *testing.T) {
	for _, lang := range []string{"ko", "en"} {
		for _, limited := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", lang, limited), func(t *testing.T) {
				s, terminal := resultScreen(t)
				s.prefs.Language, s.limitedColors = lang, limited
				s.app.SetFocus(s.editor)
				s.busy, s.pendingChanges = true, true
				cancelled := false
				s.cancel = func() { cancelled = true }
				s.invoke("guide")
				r := s.reference
				if r == nil || len(r.articles) != 8 || !strings.Contains(r.articles[0].body, "SELECT sysdate") {
					t.Fatal("localized offline guide missing")
				}
				_, root := s.pages.GetFrontPage()
				for _, size := range [][2]int{{120, 30}, {80, 24}, {40, 12}} {
					terminal.SetSize(size[0], size[1])
					root.SetRect(0, 0, size[0], size[1]-2)
					root.Draw(terminal)
					_, _, w, h := r.text.GetRect()
					if w < 35 || h < 4 {
						t.Fatalf("unreadable guide at %v: %dx%d", size, w, h)
					}
				}
				s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
				s.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
				if !r.topics.HasFocus() {
					t.Fatal("contents unreachable")
				}
				r.topics.SetCurrentItem(2)
				s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
				s.closeHelp()
				if !r.topics.HasFocus() {
					t.Fatal("nested help did not restore focus")
				}
				s.prefs.Theme = "light"
				s.applyTheme("")
				root.Draw(terminal)
				x, y, _, _ := r.text.GetInnerRect()
				_, _, style, _ := terminal.GetContent(x, y)
				_, bg, _ := style.Decompose()
				if bg != s.colors().background {
					t.Fatal("article text background did not follow light theme")
				}
				if !strings.Contains(r.text.GetText(true), r.articles[2].title) {
					t.Fatal("theme switch lost article")
				}
				s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
				if s.reference != nil || !s.editor.HasFocus() || cancelled || !s.busy || !s.pendingChanges || len(s.jobs) != 0 {
					t.Fatal("guide disturbed active SQL")
				}
			})
		}
	}
}

func TestDiagnosticsAllowlistAndIssueCopy(t *testing.T) {
	s, terminal := resultScreen(t)
	s.prefs.Language = "en"
	s.prefsPath = "/private/secret-config.json"
	s.editor.SetText("SELECT private_sql FROM secret_table", true)
	s.connectionIdentity.Alias = "secret-alias"
	s.connectionIdentity.Host = "secret-host"
	s.connectionIdentity.User = "secret-user"
	t.Setenv("LAZYORA_PASSWORD", "secret-password")
	t.Setenv("NO_COLOR", "secret-value")
	t.Setenv("TERM", "xterm[red]\n\x1b[0m")
	s.showDiagnostics()
	text := s.reference.text.GetText(true)
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	copy := s.clipboardText
	for _, value := range []string{"secret-config", "private_sql", "secret_table", "secret-alias", "secret-host", "secret-user", "secret-password", "secret-value"} {
		if strings.Contains(copy, value) {
			t.Fatalf("diagnostics leaked %s", value)
		}
	}
	if !strings.Contains(copy, "TERM: xterm[red][0m") || strings.Contains(copy, "\x1b") || !strings.Contains(text, "xterm[red][0m") || !strings.Contains(copy, "Keymap: default") || string(terminal.GetClipboardData()) != copy {
		t.Fatal("public diagnostics not preserved/sanitized")
	}
	s.closeReference()
	s.showIssueReport()
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if !strings.Contains(s.clipboardText, appinfo.ProjectURL+"/issues") || !strings.Contains(s.clipboardText, "Steps to reproduce:") || !strings.Contains(s.reference.text.GetText(true), "OS/CPU:") {
		t.Fatal("issue form/preview missing")
	}
	// Tab through every action, and activate the URL button without opening a browser.
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	button, ok := s.app.GetFocus().(*tview.Button)
	if !ok {
		t.Fatal("copy URL button unreachable")
	}
	button.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.clipboardText != appinfo.ProjectURL+"/issues" {
		t.Fatal("URL action did not copy just the URL")
	}
}

func TestReferenceMarkdownFencesAndEscapedTags(t *testing.T) {
	articles := referenceArticles("# Title\n## One\n```sql\n## not a section\nSELECT '[red]';\n```\n### Details\n## Two\ntext")
	if len(articles) != 2 || !strings.Contains(articles[0].body, "## not a section") {
		t.Fatal("fenced code split the contents")
	}
	s, _ := resultScreen(t)
	if !strings.Contains(s.referenceContent(articles[0]), "[red[]") {
		t.Fatal("documentation interpreted color tags")
	}
}
