package ui

import (
	"bytes"
	"io"
	"testing"

	"github.com/gdamore/tcell/v2"
)

type recordedTTY struct {
	tcell.Tty
	bytes.Buffer
	stopped bool
}

func (t *recordedTTY) Read(data []byte) (int, error)  { return 0, io.EOF }
func (t *recordedTTY) Write(data []byte) (int, error) { return t.Buffer.Write(data) }
func (t *recordedTTY) Stop() error                    { t.stopped = true; return nil }

func TestTerminalKeepsTextAndControlsButSkipsDefaultCursorStyle(t *testing.T) {
	// Includes legitimate q text, colors, cursor motion, mouse and non-default
	// cursor commands. Test every split boundary, including inside DECSCUSR.
	input := "SELECT 'q', '한글' FROM dual;\x1b[0 q\x1b[31mred\x1b[0m\x1b[3;4H\x1b[0 q\x1b[?1000h\x1b[2 q"
	want := "SELECT 'q', '한글' FROM dual;\x1b[31mred\x1b[0m\x1b[3;4H\x1b[?1000h\x1b[2 q"
	for split := 0; split <= len(input); split++ {
		tty := &recordedTTY{}
		filtered := &compatibleTTY{Tty: tty}
		for _, part := range []string{input[:split], input[split:]} {
			written, err := filtered.Write([]byte(part))
			if err != nil || written != len(part) {
				t.Fatalf("split %d: write=%d err=%v", split, written, err)
			}
		}
		if err := filtered.Stop(); err != nil || !tty.stopped || tty.String() != want {
			t.Fatalf("split %d: output=%q err=%v", split, tty.String(), err)
		}
	}
}

func TestTerminalFlushesIncompleteControlOnStop(t *testing.T) {
	tty := &recordedTTY{}
	filtered := &compatibleTTY{Tty: tty}
	if _, err := filtered.Write([]byte("text\x1b[0 ")); err != nil {
		t.Fatal(err)
	}
	if err := filtered.Stop(); err != nil {
		t.Fatal(err)
	}
	if tty.String() != "text\x1b[0 " || !tty.stopped {
		t.Fatalf("incomplete control was lost: %q", tty.String())
	}
}
