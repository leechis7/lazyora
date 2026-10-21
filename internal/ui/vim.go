package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

// This intentionally implements a documented subset, not an embedded Vim.
// Keep only the active editor, so closing tabs cannot retain their documents.
type vimState struct {
	editor         *tview.TextArea
	mode           string
	dot, anchor    int
	linewise       bool
	count, opCount int
	op             rune
	g              bool
	lastText       string
	from, to       int
	searchBack     bool
}

func nextCluster(text string, p int) int {
	if p >= len(text) {
		return len(text)
	}
	cluster, _, _, _ := uniseg.FirstGraphemeClusterInString(text[p:], -1)
	return p + len(cluster)
}
func prevCluster(text string, p int) int {
	if p <= 0 {
		return 0
	}
	// ASCII cannot be the trailing part of a combining/ZWJ sequence.
	if p >= 2 && text[p-2:p] == "\r\n" {
		return p - 2
	}
	if text[p-1] < utf8.RuneSelf {
		return p - 1
	}
	start := lineStart(text, p)
	if start == p {
		return p - 1
	}
	g := uniseg.NewGraphemes(text[start:p])
	previous := start
	for g.Next() {
		a, _ := g.Positions()
		previous = start + a
	}
	return previous
}
func normalPosition(text string, p int) int {
	p = min(max(p, 0), len(text))
	start, end := lineStart(text, p), lineEnd(text, p)
	if p == end && p > start {
		return prevCluster(text, p)
	}
	return p
}
func wordClass(text string, p int) int {
	if p >= len(text) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(text[p:])
	if unicode.IsSpace(r) {
		return 0
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return 1
	}
	return 2
}
func wordForward(text string, p int) int {
	class := wordClass(text, p)
	for p < len(text) && wordClass(text, p) == class {
		p = nextCluster(text, p)
	}
	for p < len(text) && wordClass(text, p) == 0 {
		p = nextCluster(text, p)
	}
	return p
}
func wordBack(text string, p int) int {
	// One forward pass avoids repeatedly rescanning a long Unicode word.
	g := uniseg.NewGraphemes(text[:p])
	previousClass, groupStart, lastWord := 0, 0, 0
	for g.Next() {
		at, _ := g.Positions()
		class := wordClass(text, at)
		if class != previousClass {
			groupStart = at
		}
		if class != 0 {
			lastWord = groupStart
		}
		previousClass = class
	}
	return lastWord
}
func wordEnd(text string, p int) int {
	p = nextCluster(text, p)
	for p < len(text) && wordClass(text, p) == 0 {
		p = nextCluster(text, p)
	}
	class := wordClass(text, p)
	for p < len(text) {
		q := nextCluster(text, p)
		if q == len(text) || wordClass(text, q) != class {
			break
		}
		p = q
	}
	return p
}
func verticalPosition(text string, p, step int) int {
	start := lineStart(text, p)
	column := uniseg.GraphemeClusterCount(text[start:p])
	if step < 0 {
		if start == 0 {
			return p
		}
		start = lineStart(text, start-1)
	} else {
		end := lineEnd(text, p)
		if end == len(text) {
			return p
		}
		start = end + 1
	}
	end := lineEnd(text, start)
	for column > 0 && start < end {
		start = nextCluster(text, start)
		column--
	}
	return normalPosition(text, start)
}

func (v *vimState) remember() {
	v.lastText = v.editor.GetText()
	_, v.from, v.to = v.editor.GetSelection()
}
func (v *vimState) position() {
	text := v.editor.GetText()
	v.dot = normalPosition(text, v.dot)
	if v.mode == "VISUAL" {
		a, b := min(v.anchor, v.dot), max(v.anchor, v.dot)
		if v.linewise {
			a, b = selectedLines(text, a, b)
		} else {
			b = nextCluster(text, b)
		}
		v.editor.Select(a, b)
	} else {
		v.editor.Select(v.dot, v.dot)
	}
}
func (s *screen) vimSync(e *tview.TextArea) {
	text := e.GetText()
	_, a, b := e.GetSelection()
	v := &s.vim
	if v.editor != e {
		*v = vimState{editor: e, mode: "NORMAL", dot: a}
	}
	if v.mode != "INSERT" && (v.lastText != text || v.from != a || v.to != b) {
		v.dot, v.op, v.count, v.g = a, 0, 0, false
		if a != b {
			v.mode, v.anchor, v.dot, v.linewise = "VISUAL", a, prevCluster(text, b), false
		} else {
			v.mode = "NORMAL"
		}
	}
	if v.mode == "INSERT" {
		v.dot = b
	} else {
		v.position()
	}
	v.remember()
}

func (s *screen) vimOperation(op rune, a, b int, linewise bool) {
	v := &s.vim
	e := v.editor
	text := e.GetText()
	a, b = min(a, b), max(a, b)
	if linewise {
		a, b = selectedLines(text, a, b)
	}
	s.yankText, s.yankLine = text[a:b], linewise
	if linewise && !strings.HasSuffix(s.yankText, "\n") {
		s.yankText += "\n"
	}
	s.sendClipboard(s.yankText)
	v.mode, v.op, v.count, v.opCount, v.g = "NORMAL", 0, 0, 0, false
	if op != 'y' {
		replacement := ""
		if linewise && op == 'c' && b < len(text) {
			replacement = "\n"
		}
		if linewise && op == 'd' && b == len(text) && a > 0 && !strings.HasSuffix(text[a:b], "\n") {
			a--
		}
		e.Replace(a, b, replacement)
	}
	v.dot = a
	if op == 'c' {
		v.mode = "INSERT"
		e.Select(a, a)
	} else {
		v.position()
	}
}

func (s *screen) vimPaste(before bool) {
	v := &s.vim
	e := v.editor
	text := e.GetText()
	value := s.clipboardText
	if value == "" {
		return
	}
	p := v.dot
	if s.yankLine && value == s.yankText {
		p = lineStart(text, p)
		if !before {
			p = lineEnd(text, v.dot)
			if p < len(text) {
				p++
			} else {
				value = "\n" + strings.TrimSuffix(value, "\n")
			}
		}
	} else if !before && p < lineEnd(text, p) {
		p = nextCluster(text, p)
	}
	e.Replace(p, p, value)
	v.dot = p
	v.position()
}

func (s *screen) vimKey(e *tview.TextArea, event *tcell.EventKey) bool {
	if event.Key() == tcell.KeyEnter && event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0 {
		return false
	}
	s.vimSync(e)
	v := &s.vim
	defer v.remember()
	key, mod := event.Key(), event.Modifiers()&^tcell.ModCtrl
	if key == tcell.KeyEscape {
		if v.mode == "INSERT" {
			v.dot = prevCluster(e.GetText(), v.dot)
		}
		v.mode, v.op, v.count, v.opCount, v.g = "NORMAL", 0, 0, 0, false
		v.position()
		return true
	}
	if key == tcell.KeyCtrlR && mod == 0 {
		s.nativeEditorKey(e, tcell.KeyCtrlY)
		_, v.dot, _ = e.GetSelection()
		v.mode = "NORMAL"
		v.position()
		return true
	}
	if v.mode == "INSERT" {
		return false
	}
	var char rune
	switch key {
	case tcell.KeyRune:
		if event.Modifiers()&tcell.ModCtrl != 0 {
			return false
		}
		char = event.Rune()
	case tcell.KeyLeft, tcell.KeyBackspace, tcell.KeyBackspace2:
		char = 'h'
	case tcell.KeyRight:
		char = 'l'
	case tcell.KeyUp:
		char = 'k'
	case tcell.KeyDown:
		char = 'j'
	case tcell.KeyHome:
		char = '0'
	case tcell.KeyEnd:
		char = '$'
	case tcell.KeyEnter:
		return true
	case tcell.KeyDelete:
		char = 'x'
	default:
		return false // App commands, menus and function keys retain their routing.
	}
	text := e.GetText()
	if char >= '0' && char <= '9' && (char != '0' || v.count > 0) {
		v.count = min(999, v.count*10+int(char-'0'))
		return true
	}
	count := max(1, v.count)
	if v.op != 0 {
		count = min(999, count*max(1, v.opCount))
	}
	hadCount := v.count > 0
	v.count = 0
	if v.mode == "VISUAL" && strings.ContainsRune("dycx", char) {
		_, a, b := e.GetSelection()
		// Selection already includes its final grapheme/newline.
		op := char
		if op == 'x' {
			op = 'd'
		}
		if v.linewise && b > a {
			b--
		}
		s.vimOperation(op, a, b, v.linewise)
		return true
	}
	if char == 'g' && !v.g {
		v.g = true
		v.count = count
		return true
	}
	old := v.dot
	target, motion, inclusive, linewise := old, true, false, false
	switch char {
	case 'h':
		for i := 0; i < count && target > lineStart(text, target); i++ {
			target = prevCluster(text, target)
		}
	case 'l':
		for i := 0; i < count && target < lineEnd(text, target); i++ {
			target = nextCluster(text, target)
		}
	case 'j', 'k':
		step := 1
		if char == 'k' {
			step = -1
		}
		for i := 0; i < count; i++ {
			target = verticalPosition(text, target, step)
		}
		linewise = true
	case 'w':
		if v.op == 'c' && wordClass(text, target) != 0 {
			for i := 0; i < count; i++ {
				class := wordClass(text, target)
				for target < len(text) && wordClass(text, target) == class {
					target = nextCluster(text, target)
				}
				if i < count-1 {
					for target < len(text) && wordClass(text, target) == 0 {
						target = nextCluster(text, target)
					}
				}
			}
		} else {
			for i := 0; i < count; i++ {
				target = wordForward(text, target)
			}
		}
	case 'b':
		for i := 0; i < count; i++ {
			target = wordBack(text, target)
		}
	case 'e':
		for i := 0; i < count; i++ {
			target = wordEnd(text, target)
		}
		inclusive = true
	case '0':
		target = lineStart(text, target)
	case '^':
		target = lineStart(text, target)
		for target < lineEnd(text, target) && (text[target] == ' ' || text[target] == '\t') {
			target++
		}
	case '$':
		target = lineEnd(text, target)
		inclusive = true
	case 'G':
		target = len(text)
		if hadCount {
			target = 0
			for i := 1; i < count && target < len(text); i++ {
				end := lineEnd(text, target)
				target = min(len(text), end+1)
			}
		}
		linewise = true
	case 'g':
		if !v.g {
			return true
		}
		target = 0
		for i := 1; i < count && target < len(text); i++ {
			target = min(len(text), lineEnd(text, target)+1)
		}
		linewise = true
	default:
		motion = false
	}
	if motion {
		v.g = false
		if v.op != 0 {
			a, b := old, target
			if inclusive && b >= a && b < lineEnd(text, b) {
				b = nextCluster(text, b)
			}
			if linewise {
				a, b = min(a, b), max(a, b)
				a, b = selectedLines(text, a, b)
				if b > a {
					b--
				}
			}
			s.vimOperation(v.op, a, b, linewise)
		} else {
			v.dot = target
			v.position()
		}
		return true
	}
	v.g = false
	if v.op != 0 {
		op := v.op
		if char == op {
			a, b := lineStart(text, old), old
			for i := 1; i < count; i++ {
				end := lineEnd(text, b)
				if end == len(text) {
					break
				}
				b = end + 1
			}
			s.vimOperation(op, a, b, true)
		} else {
			v.op, v.opCount = 0, 0
		}
		return true
	}
	switch char {
	case 'v', 'V':
		if v.mode == "VISUAL" && v.linewise == (char == 'V') {
			v.mode = "NORMAL"
		} else {
			v.mode, v.anchor, v.linewise = "VISUAL", old, char == 'V'
		}
		v.position()
	case 'i', 'a', 'I', 'A':
		if char == 'a' && old < lineEnd(text, old) {
			old = nextCluster(text, old)
		}
		if char == 'I' {
			old = lineStart(text, old)
		}
		if char == 'A' {
			old = lineEnd(text, old)
		}
		v.mode = "INSERT"
		e.Select(old, old)
	case 'o', 'O':
		p := lineStart(text, old)
		replacement := "\n"
		if char == 'o' {
			p = lineEnd(text, old)
			if p < len(text) {
				p++
			} else {
				replacement = "\n"
			}
		}
		e.Replace(p, p, replacement)
		if char == 'o' && p == len(text) {
			p++
		}
		e.Select(p, p)
		v.mode = "INSERT"
	case 'd', 'y', 'c':
		v.op, v.opCount = char, count
	case 'x', 'X':
		a, b := old, old
		for i := 0; i < count; i++ {
			if char == 'x' {
				if b < lineEnd(text, b) {
					b = nextCluster(text, b)
				}
			} else if a > lineStart(text, a) {
				a = prevCluster(text, a)
			}
		}
		s.vimOperation('d', a, b, false)
	case 'D', 'C':
		end := lineEnd(text, old)
		op := 'd'
		if char == 'C' {
			op = 'c'
		}
		s.vimOperation(op, old, end, false)
	case 'p', 'P':
		s.vimPaste(char == 'P')
	case 'u':
		s.nativeEditorKey(e, tcell.KeyCtrlZ)
		_, v.dot, _ = e.GetSelection()
		v.mode = "NORMAL"
		v.position()
	case '/', '?':
		v.searchBack = char == '?'
		s.showSearch(false)
	case 'n', 'N':
		back := v.searchBack
		if char == 'N' {
			back = !back
		}
		s.findEditor(e, back)
		_, v.dot, _ = e.GetSelection()
		v.mode = "NORMAL"
		v.position()
	}
	// Unknown normal-mode text keys never insert SQL accidentally.
	return true
}
