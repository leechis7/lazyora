package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
	"lazyora/internal/settings"
)

type displayCapability struct {
	mode, reason string
	korean       bool
}

// TERM and color counts describe protocols, not font coverage. CanDisplay is
// only an encoding/terminfo estimate; never send terminal probes or change locale.
func detectDisplay(terminal tcell.Screen, requested string) displayCapability {
	c := displayCapability{mode: "unicode", reason: "display_reason_auto", korean: true}
	if terminal != nil {
		c.korean = terminal.CanDisplay('한', false)
	}
	switch requested {
	case "ascii":
		c.mode, c.reason = "ascii", "display_reason_ascii"
		return c
	case "unicode":
		c.reason = "display_reason_unicode"
		return c
	}
	if terminal == nil {
		c.reason = "display_reason_unknown"
		return c
	}
	charset := strings.ToUpper(strings.TrimSpace(terminal.CharacterSet()))
	if charset == "US-ASCII" || charset == "ASCII" || charset == "ANSI_X3.4-1968" {
		c.mode, c.reason = "ascii", "display_reason_charset"
		return c
	}
	for _, r := range "─│┌▸▦" {
		if !terminal.CanDisplay(r, false) {
			c.mode, c.reason = "ascii", "display_reason_glyph"
			return c
		}
	}
	return c
}
func (s *screen) displayCapability() displayCapability {
	mode := s.prefs.DisplayMode
	if s.displayOverride != "" {
		mode = s.displayOverride
	}
	return detectDisplay(s.terminal, mode)
}
func (s *screen) asciiDisplay() bool { return s.displayCapability().mode == "ascii" }
func (s *screen) uiLanguage() string {
	c := s.displayCapability()
	if c.mode == "ascii" || s.prefs.Language == "ko" && !c.korean {
		return "en"
	}
	return s.prefs.Language
}
func (s *screen) displaySummary() string {
	c := s.displayCapability()
	override := ""
	if s.displayOverride != "" {
		override = "; --display-mode=" + s.displayOverride
	}
	return fmt.Sprintf("%s (%s%s)\n%s", c.mode, s.tr(c.reason), override, s.tr("display_font_note"))
}

// displayLayout applies fallback to drawing only. Widgets retain their original
// strings for editing, identities, hit testing, clipboard and export operations.
type displayLayout struct {
	*tview.Flex
	owner *screen
}

func (d *displayLayout) Draw(terminal tcell.Screen) {
	if d.owner.asciiDisplay() {
		terminal = asciiScreen{terminal}
	}
	d.Flex.Draw(terminal)
}

type asciiScreen struct{ tcell.Screen }

func (a asciiScreen) CanDisplay(r rune, _ bool) bool { return r >= 0x20 && r <= 0x7e }

// Tuivory's drawing adapter uses tcell's newer Put API. Keep all positioned
// writes in the same display-only conversion path, regardless of wrapper order.
func (a asciiScreen) Put(x, y int, text string, st tcell.Style) (string, int) {
	w, h := a.Size()
	if x < 0 || y < 0 || x >= w || y >= h {
		return text, 0
	}
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		if g.Width() == 0 {
			continue
		}
		_, end := g.Positions()
		runes := g.Runes()
		a.SetContent(x, y, runes[0], runes[1:], st)
		return text[end:], g.Width()
	}
	return "", 0
}
func (a asciiScreen) SetCell(x, y int, st tcell.Style, chars ...rune) {
	if len(chars) > 0 {
		a.SetContent(x, y, chars[0], chars[1:], st)
	}
}
func (a asciiScreen) PutStrStyled(x, y int, text string, st tcell.Style) {
	for text != "" {
		rest, width := a.Put(x, y, text, st)
		if width == 0 {
			return
		}
		text, x = rest, x+width
	}
}
func (a asciiScreen) PutStr(x, y int, text string) {
	a.PutStrStyled(x, y, text, tcell.StyleDefault)
}

func (a asciiScreen) SetContent(x, y int, r rune, comb []rune, st tcell.Style) {
	if r < 128 {
		// Combining marks are display-only too; retain the base cell width.
		a.Screen.SetContent(x, y, r, nil, st)
		return
	}
	replacement := asciiDecoration(r)
	width := max(1, uniseg.StringWidth(string(append([]rune{r}, comb...))))
	a.Screen.SetContent(x, y, replacement, nil, st)
	if width == 2 {
		a.Screen.SetContent(x+1, y, '?', nil, st)
	}
}
func asciiDecoration(r rune) rune {
	switch r {
	case '─', '━', '┄', '┅', '┈', '┉', '–', '—', '−':
		return '-'
	case '═':
		return '='
	case '│', '┃', '┆', '┇', '┊', '┋', '║':
		return '|'
	case '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼', '╔', '╗', '╚', '╝', '╠', '╣', '╦', '╩', '╬', '╭', '╮', '╯', '╰':
		return '+'
	case '▸', '▶', '►', '→', '›', '»':
		return '>'
	case '◂', '◀', '◄', '←', '‹', '«':
		return '<'
	case '▾', '▼', '↓':
		return 'v'
	case '▴', '▲', '↑':
		return '^'
	case '↔':
		return '-'
	case '↕':
		return '|'
	case '·', '•', '…', '⋯':
		return '.'
	case '✓', '✔', '☑':
		return 'x'
	case '☐', '□', '▦', '◇':
		return '#'
	case '‘', '’':
		return '\''
	case '“', '”':
		return '"'
	case '\u00a0', '\u2007', '\u202f':
		return ' '
	}
	return '?'
}

// RunWithDisplayMode provides a one-run override without writing preferences.
func RunWithDisplayMode(demo bool, mode string) error {
	if err := settings.ValidateDisplayMode(mode); err != nil {
		return err
	}
	return runWithDisplayMode(demo, nil, nil, mode)
}
