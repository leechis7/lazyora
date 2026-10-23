package ui

import (
	"bytes"
	"io"
	"os"
	"runtime"

	"github.com/gdamore/tcell/v2"
)

// tcell emits DECSCUSR even for its default cursor. Older SecureCRT sessions
// render the trailing q instead of consuming it. Keep the terminal's existing
// cursor shape: LazyOra only needs to show, hide and position the cursor.
var defaultCursorSequence = []byte("\x1b[0 q")

// Win32-input-mode reports key releases as CSI ... _. On SSH terminals these
// may arrive after cooked input is restored, leaking into the shell. Keep
// CSI-u/modifyOtherKeys for modified keys without opting into Win32 reports.
var win32InputEnable = []byte("\x1b[?9001h")
var win32InputDisable = []byte("\x1b[?9001l")

type compatibleTTY struct {
	tcell.Tty
	pending []byte
}

func (t *compatibleTTY) Write(data []byte) (int, error) {
	buffer := append(t.pending, data...)
	buffer = bytes.ReplaceAll(buffer, defaultCursorSequence, nil)
	buffer = bytes.ReplaceAll(buffer, win32InputEnable, win32InputDisable)
	// A control sequence may straddle writes. Hold only a possible prefix.
	keep := 0
	for _, sequence := range [][]byte{defaultCursorSequence, win32InputEnable} {
		for size := 1; size < len(sequence) && size <= len(buffer); size++ {
			if bytes.Equal(buffer[len(buffer)-size:], sequence[:size]) {
				keep = max(keep, size)
			}
		}
	}
	t.pending = append([]byte(nil), buffer[len(buffer)-keep:]...)
	if err := writeTerminal(t.Tty, buffer[:len(buffer)-keep]); err != nil {
		return 0, err
	}
	return len(data), nil
}

func writeTerminal(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}

func (t *compatibleTTY) Stop() error {
	err := writeTerminal(t.Tty, t.pending)
	t.pending = nil
	stopErr := t.Tty.Stop()
	if err != nil {
		return err
	}
	return stopErr
}

func newCompatibleScreen() (tcell.Screen, error) {
	// Windows consoles need tcell's console fallback when TERM is unset.
	if runtime.GOOS == "windows" {
		return tcell.NewScreen()
	}

	tty, err := tcell.NewDevTty()
	if err != nil {
		return nil, err
	}
	screen, err := newCompatibleScreenFromTTY(&compatibleTTY{Tty: tty})
	if err != nil {
		tty.Close()
		return nil, err
	}
	return screen, nil
}

func newCompatibleScreenFromTTY(tty tcell.Tty) (tcell.Screen, error) {
	if os.Getenv("TERM") == "" {
		// Select a conservative fallback locally, without modifying the process
		// environment or replacing an explicitly configured terminal protocol.
		info, err := tcell.LookupTerminfo("ansi")
		if err != nil {
			return nil, err
		}
		return tcell.NewTerminfoScreenFromTtyTerminfo(tty, info)
	}
	return tcell.NewTerminfoScreenFromTty(tty)
}
