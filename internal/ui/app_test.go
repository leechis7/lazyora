package ui

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestDemoInterfaceAndConnectionForm(t *testing.T) {
	terminal := tcell.NewSimulationScreen("UTF-8")
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
			s.app.SetAfterDrawFunc(func(tcell.Screen) { once.Do(func() { close(drawn) }) })
		})
	}()
	select {
	case <-drawn:
	case err := <-finished:
		t.Fatalf("app stopped before first draw: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("app did not draw")
	}
	if ui.table.GetCell(1, 1).Text != "Demo — no database connected" {
		t.Fatal("missing demo result")
	}
	// Exercise real key dispatch, not just the widget construction.
	ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyF4, 0, tcell.ModNone))
	ui.app.QueueUpdate(func() {})
	ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModNone))
	ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModNone))
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
