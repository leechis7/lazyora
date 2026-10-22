package ui

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"unicode"

	resources "lazyora"
	"lazyora/internal/appinfo"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type referenceArticle struct{ title, body string }
type referenceView struct {
	kind     string
	articles []referenceArticle
	text     *tview.TextView
	topics   *tview.List
	focus    tview.Primitive
	copy     func() string
}

// Only level-two headings outside code fences divide articles. A version's
// level-three Added/Fixed headings stay inside that version's release note.
func referenceArticles(markdown string) []referenceArticle {
	var articles []referenceArticle
	var current *referenceArticle
	fenced := false
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
		}
		if !fenced && strings.HasPrefix(line, "## ") {
			articles = append(articles, referenceArticle{title: strings.TrimSpace(strings.TrimPrefix(line, "## "))})
			current = &articles[len(articles)-1]
		} else if current != nil {
			current.body += line + "\n"
		}
	}
	return articles
}

var referenceLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

func referencePlain(text string) string {
	text = referenceLink.ReplaceAllString(text, "$1 ($2)")
	text = strings.ReplaceAll(text, "**", "")
	return strings.ReplaceAll(text, "`", "")
}

func (s *screen) referenceContent(article referenceArticle) string {
	if s.reference != nil && (s.reference.kind == "diagnostic_info" || s.reference.kind == "issue_report") {
		return s.helpSubheading(article.title) + tview.Escape(article.body)
	}
	var out strings.Builder
	out.WriteString(s.helpSubheading(referencePlain(article.title)))
	fenced := false
	for _, line := range strings.Split(article.body, "\n") {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			out.WriteByte('\n')
			continue
		}
		if !fenced && strings.HasPrefix(line, "### ") {
			out.WriteByte('\n')
			out.WriteString(s.helpSubheading(strings.TrimPrefix(line, "### ")))
			continue
		}
		if fenced {
			out.WriteString("  ")
		} else {
			line = referencePlain(line)
		}
		// Markdown and environment values are data, never tview color tags.
		out.WriteString(tview.Escape(line))
		out.WriteByte('\n')
	}
	return out.String()
}

func (s *screen) showGuide() {
	data, _ := resources.Documents.ReadFile("docs/guide." + s.prefs.Language + ".md")
	if len(data) == 0 {
		data, _ = resources.Documents.ReadFile("docs/guide.en.md")
	}
	s.showReference("guide", referenceArticles(string(data)), nil)
}

func (s *screen) showReleaseHistory() {
	data, _ := resources.Documents.ReadFile("CHANGELOG.md")
	articles := referenceArticles(string(data))
	for i := range articles {
		if strings.HasPrefix(articles[i].title, "[Unreleased]") {
			articles[i].title = s.tr("release_unreleased")
		}
		if s.prefs.Language != "ko" {
			articles[i].body = s.tr("release_language_note") + "\n\n" + articles[i].body
		}
	}
	s.showReference("release_history", articles, nil)
	// Start at the executable's release, keeping development notes separate.
	for i, article := range articles {
		if strings.HasPrefix(article.title, "["+appinfo.Version+"]") {
			s.reference.topics.SetCurrentItem(i)
			break
		}
	}
}

func publicEnvironment(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	runes := []rune(value)
	if len(runes) > 120 {
		value = string(runes[:120]) + "…"
	}
	return value
}

// An explicit allowlist: never inspect connection profiles, SQL, history,
// credential files, process arguments, usernames, hostnames or config paths.
func (s *screen) diagnosticContent() string {
	var out strings.Builder
	fmt.Fprintf(&out, "%s v%s\nOS/CPU: %s/%s\nGo: %s\n", appinfo.Name, appinfo.Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
	if s.terminal != nil {
		w, h := s.terminal.Size()
		fmt.Fprintf(&out, "Terminal: %dx%d\nColors: %d\n", w, h, s.terminal.Colors())
		fmt.Fprintf(&out, "Terminal mouse support: %v\n", s.terminal.HasMouse())
	}
	theme, keymap := s.prefs.Theme, s.prefs.EditorKeymap
	if theme == "" {
		theme = "dark"
	}
	if keymap == "" {
		keymap = "default"
	}
	fmt.Fprintf(&out, "Language: %s\nTheme: %s\nKeymap: %s\nMouse: %v\n", s.prefs.Language, theme, keymap, s.prefs.Mouse)
	for _, key := range []string{"TERM", "COLORTERM", "LANG", "LC_ALL", "LC_MESSAGES", "LC_CTYPE"} {
		fmt.Fprintf(&out, "%s: %s\n", key, publicEnvironment(os.Getenv(key)))
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	fmt.Fprintf(&out, "NO_COLOR present: %v\n", noColor)
	return out.String()
}

func (s *screen) showDiagnostics() {
	snapshot := s.diagnosticContent()
	s.showReference("diagnostic_info", []referenceArticle{{s.tr("diagnostic_info"), s.tr("diagnostics_note") + "\n\n" + snapshot}}, func() string { return snapshot })
}

func (s *screen) issueTemplate() string {
	return appinfo.ProjectURL + "/issues\n\n" + s.tr("issue_template") + "\n\n" + s.diagnosticContent()
}

func (s *screen) showIssueReport() {
	template := s.issueTemplate()
	s.showReference("issue_report", []referenceArticle{{s.tr("issue_report"), s.tr("issue_note") + "\n\n" + template}}, func() string { return template })
}

func (s *screen) showReference(kind string, articles []referenceArticle, copyContent func() string) {
	if s.reference != nil || len(articles) == 0 {
		return
	}
	r := &referenceView{kind: kind, articles: articles, focus: s.app.GetFocus(), copy: copyContent}
	s.reference = r
	text := s.newTextView().SetDynamicColors(true).SetWrap(true).SetWordWrap(true).SetScrollable(true).SetTextColor(s.colors().text)
	text.SetBorder(true).SetTitle(s.tr(kind)).SetBorderPadding(1, 1, 2, 2)
	s.styleBox(text.Box)
	text.SetBackgroundColor(s.colors().background)
	r.text = text
	topics := tview.NewList().ShowSecondaryText(false).SetMainTextColor(s.colors().text).SetMainTextStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface)).SetHighlightFullLine(true).SetSelectedFocusOnly(false).SetSelectedStyle(s.focusStyle().Underline(false))
	topics.SetBorder(true).SetTitle(s.tr("reference_topics"))
	s.styleList(topics)
	topics.SetBackgroundColor(s.colors().surface)
	r.topics = topics
	for _, article := range articles {
		topics.AddItem(tview.Escape(article.title), "", 0, nil)
	}
	topics.SetChangedFunc(func(index int, _, _ string, _ rune) {
		text.SetText(s.referenceContent(articles[index])).ScrollToBeginning()
	})
	topics.SetSelectedFunc(func(int, string, string, rune) { s.app.SetFocus(text) })
	body := s.newFlex().AddItem(topics, 28, 0, false).AddItem(text, 0, 1, true)
	if len(articles) == 1 {
		body.RemoveItem(topics)
	}
	body.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
		if h < 16 {
			text.SetBorderPadding(0, 0, 1, 1)
		} else {
			text.SetBorderPadding(1, 1, 2, 2)
		}
		if len(articles) > 1 {
			if w < 76 {
				body.SetDirection(tview.FlexRow).ResizeItem(topics, min(5, max(3, h/4)), 0)
			} else {
				body.SetDirection(tview.FlexColumn).ResizeItem(topics, 28, 0)
			}
		}
		return x, y, w, h
	})
	buttons := s.newFlex()
	add := func(label string, selected func()) {
		buttons.AddItem(s.clickable(tview.NewButton(label).SetSelectedFunc(selected).SetStyle(s.buttonStyle()).SetActivatedStyle(s.buttonFocusStyle())), 0, 1, false)
	}
	add(s.tr("copy"), s.copyReference)
	if kind == "issue_report" {
		add(s.tr("copy_url"), func() { s.sendClipboard(appinfo.ProjectURL + "/issues") })
	}
	add(s.tr("close"), s.closeReference)
	root := s.newFlex().SetDirection(tview.FlexRow).AddItem(body, 0, 1, true).AddItem(buttons, 1, 0, false)
	text.SetText(s.referenceContent(articles[0]))
	s.pages.AddPage("reference", &formPopupLayer{Flex: root, content: root}, true, true)
	if len(articles) > 1 {
		s.app.SetFocus(topics)
	} else {
		s.app.SetFocus(text)
	}
}

func (s *screen) copyReference() {
	if s.reference == nil {
		return
	}
	r := s.reference
	if r.copy != nil {
		s.sendClipboard(r.copy())
		return
	}
	article := r.articles[r.topics.GetCurrentItem()]
	s.sendClipboard(article.title + "\n\n" + article.body)
}

func (s *screen) closeReference() {
	if s.reference == nil {
		return
	}
	focus := s.reference.focus
	s.pages.RemovePage("reference")
	s.reference = nil
	if focus != nil {
		s.app.SetFocus(focus)
	}
}

func (s *screen) referenceKey(e *tcell.EventKey) *tcell.EventKey {
	if len(s.reference.articles) > 1 && s.listDetailArrow(e, s.reference.topics, s.reference.text) {
		return nil
	}
	switch e.Key() {
	case tcell.KeyEscape:
		s.closeReference()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyCtrlC:
		s.copyReference()
		return nil
	case tcell.KeyCtrlQ:
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		if len(s.reference.articles) > 1 {
			s.dialogTab(e, s.reference.topics, s.reference.text)
		} else {
			s.dialogTab(e, s.reference.text)
		}
		return nil
	}
	if e.Key() == tcell.KeyRune && e.Modifiers()&tcell.ModAlt != 0 && unicode.ToLower(e.Rune()) == 'c' {
		s.copyReference()
		return nil
	}
	if s.buttonArrow(e) {
		return nil
	}
	return e
}
