package ui

import (
	"fmt"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestDocumentationMouseDoesNotReachSchema(t *testing.T) {
	for _, kind := range []string{"help", "guide", "release_history", "about"} {
		for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 16}} {
			t.Run(fmt.Sprintf("%s/%dx%d", kind, size[0], size[1]), func(t *testing.T) {
				spaces, terminal := workspaceTestSet(t)
				s := spaces.active
				terminal.SetSize(size[0], size[1])
				s.rootLayout.SetRect(0, 0, size[0], size[1])
				s.rootLayout.Draw(terminal)
				switch kind {
				case "help":
					s.showHelp()
				case "guide":
					s.showGuide()
				case "release_history":
					s.showReleaseHistory()
				case "about":
					s.showAbout()
				}
				s.rootLayout.Draw(terminal)
				if kind == "about" {
					focus := s.app.GetFocus()
					for _, action := range []tuivory.MouseAction{tuivory.MouseLeftDown, tuivory.MouseLeftUp, tuivory.MouseLeftClick} {
						s.rootLayout.MouseHandler()(action, tuivory.NewEventMouse(12, 3, tuivory.Button1, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
					}
					if s.browserUsers.IsOpen() || s.browserTypes.IsOpen() || s.app.GetFocus() != focus || len(s.jobs) != 0 {
						t.Fatal("About backdrop clicks reached the schema browser")
					}
					return
				}
				topics := s.helpTopics
				if kind != "help" {
					topics = s.reference.topics
				}
				topics.SetCurrentItem(0)
				s.rootLayout.Draw(terminal)
				var captured tuivory.Primitive
				focus := func(p tuivory.Primitive) { s.app.SetFocus(p) }
				// Application captures the primitive returned by MouseLeftDown.
				// A click-only test misses the hidden dropdown stealing the release.
				deliver := func(action tuivory.MouseAction, x, y int) {
					target := tuivory.Primitive(s.rootLayout)
					if captured != nil {
						target = captured
					}
					_, captured = target.MouseHandler()(action, tuivory.NewEventMouse(x, y, tuivory.Button1, 0), focus)
				}
				x, y, _, h := topics.GetInnerRect()
				for _, index := range []int{0, 1, 2, 3, 1, 0} {
					if index >= h {
						continue
					}
					deliver(tuivory.MouseLeftDown, x+12, y+index)
					if s.browserUsers.IsOpen() || s.browserTypes.IsOpen() {
						t.Fatalf("contents row %d opened background schema dropdown (capture %T)", index, captured)
					}
					deliver(tuivory.MouseLeftUp, x+12, y+index)
					deliver(tuivory.MouseLeftClick, x+12, y+index)
					if topics.GetCurrentItem() != index {
						t.Fatalf("contents row %d was not selected (got %d)", index, topics.GetCurrentItem())
					}
					s.rootLayout.Draw(terminal)
				}
				// Borders and blank areas must not redirect focus into the editor
				// or result grid beneath a full-page document.
				focusBefore := s.app.GetFocus()
				deliver(tuivory.MouseLeftDown, x, y-1)
				deliver(tuivory.MouseLeftUp, x, y-1)
				deliver(tuivory.MouseLeftClick, x, y-1)
				if s.browserUsers.IsOpen() || s.browserTypes.IsOpen() || s.editor.HasFocus() || s.table.HasFocus() {
					t.Fatalf("document border passed through to background (was %T)", focusBefore)
				}
				if len(s.jobs) != 0 {
					t.Fatal("document clicks submitted a DB job")
				}
			})
		}
	}
}
