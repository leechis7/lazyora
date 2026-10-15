package ui

import (
	"bytes"
	"io"

	"github.com/gdamore/tcell/v2"
)

// tcell emits DECSCUSR even for its default cursor. Older SecureCRT sessions
// render the trailing q instead of consuming it. Keep the terminal's existing
// cursor shape: LazyOra only needs to show, hide and position the cursor.
var defaultCursorSequence = []byte("\x1b[0 q")

type compatibleTTY struct {
	tcell.Tty
	pending []byte
}

func (t *compatibleTTY) Write(data []byte) (int, error) {
	buffer := append(t.pending, data...)
	buffer = bytes.ReplaceAll(buffer, defaultCursorSequence, nil)
	// A control sequence may straddle writes. Hold only a possible prefix.
	keep := 0
	for size := 1; size < len(defaultCursorSequence) && size <= len(buffer); size++ {
		if bytes.Equal(buffer[len(buffer)-size:], defaultCursorSequence[:size]) {
			keep = size
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
	tty, err := tcell.NewDevTty()
	if err != nil {
		return nil, err
	}
	screen, err := tcell.NewTerminfoScreenFromTty(&compatibleTTY{Tty: tty})
	if err != nil {
		tty.Close()
		return nil, err
	}
	return screen, nil
}
