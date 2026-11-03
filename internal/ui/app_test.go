package ui

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leechis7/tuivory/go"
)

func TestDemoInterfaceAndConnectionForm(t *testing.T) {
	t.Setenv("LAZYORA_CONFIG", filepath.Join(t.TempDir(), "settings.json"))
	t.Setenv("LC_ALL", "C")
	terminal := tuivory.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	terminal.SetSize(100, 30)
	drawn := make(chan struct{})
	var once sync.Once
	var ui *screen
	finished := make(chan error, 1)
	go func() {
		finished <- run(true, terminal, func(s *screen) {
			ui = s
			s.app.SetAfterDrawFunc(func(tuivory.Screen) { once.Do(func() { close(drawn) }) })
		})
	}()
	select {
	case <-drawn:
	case err := <-finished:
		t.Fatalf("app stopped before first draw: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("app did not draw")
	}
	if ui.resultRows[0][1] != "Demo — no database connected" {
		t.Fatal("missing demo result")
	}
	// Exercise real key dispatch, not just the widget construction.
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyF9, 0, tuivory.ModNone))
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyRune, 's', tuivory.ModAlt))
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModNone))
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyEscape, 0, tuivory.ModNone))
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyCtrlQ, 0, tuivory.ModNone))
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyLeft, 0, tuivory.ModNone))
	ui.app.QueueEvent(tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModNone))
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("app did not stop")
	}
	if !strings.Contains(ui.status.GetText(false), "Connect first") {
		t.Fatal("demo executed a statement")
	}
}
