package ui

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestMissingTermFallsBackWithoutChangingEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix TERM selection; Windows uses tcell native console selection")
	}
	for _, name := range []string{"unset", "empty", "xterm", "invalid"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TERM", "")
			if name == "unset" {
				if err := os.Unsetenv("TERM"); err != nil {
					t.Fatal(err)
				}
			} else if name == "xterm" {
				t.Setenv("TERM", "xterm")
			} else if name == "invalid" {
				t.Setenv("TERM", "lazyora-unknown-terminal-for-test")
			}
			before, existed := os.LookupEnv("TERM")
			screen, err := newCompatibleScreenFromTTY(&recordedTTY{})
			if name == "invalid" {
				if err == nil {
					t.Fatal("an explicitly unsupported terminal was silently replaced")
				}
			} else if err != nil {
				t.Fatal(err)
			} else if screen.HasMouse() != (name == "xterm") {
				t.Fatal("fallback or explicit terminal capability was not preserved")
			}
			after, exists := os.LookupEnv("TERM")
			if before != after || existed != exists {
				t.Fatal("terminal initialization changed TERM")
			}
		})
	}
}

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

func TestTerminalAvoidsWin32KeyReleaseReportsWithoutDisablingModifiedKeys(t *testing.T) {
	input := "\x1b[>4;2m\x1b[>1u\x1b[?9001hSELECT 'q'\x1b[0 q\x1b[?9001l\x1b[<u\x1b[>4;0m"
	want := "\x1b[>4;2m\x1b[>1u\x1b[?9001lSELECT 'q'\x1b[?9001l\x1b[<u\x1b[>4;0m"
	for split := 0; split <= len(input); split++ {
		tty := &recordedTTY{}
		filtered := &compatibleTTY{Tty: tty}
		for _, part := range []string{input[:split], input[split:]} {
			if _, err := filtered.Write([]byte(part)); err != nil {
				t.Fatal(err)
			}
		}
		if err := filtered.Stop(); err != nil || tty.String() != want {
			t.Fatalf("split %d: output=%q err=%v", split, tty.String(), err)
		}
	}
}

func TestEightColorMouseCapabilityDependsOnTerminalProtocol(t *testing.T) {
	for _, name := range []string{"ansi", "xterm"} {
		t.Run(name, func(t *testing.T) {
			info, err := tcell.LookupTerminfo(name)
			if err != nil {
				t.Fatal(err)
			}
			if info.Colors != 8 {
				t.Fatalf("%s no longer describes an 8-color terminal", name)
			}
			screen, err := tcell.NewTerminfoScreenFromTtyTerminfo(&recordedTTY{}, info)
			if err != nil {
				t.Fatal(err)
			}
			if screen.HasMouse() != (name == "xterm") {
				t.Fatal("mouse capability does not match the terminal protocol")
			}
		})
	}
}
