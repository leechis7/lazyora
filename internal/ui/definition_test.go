package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func definitionScreen(t *testing.T) *screen {
	s := demoBrowserScreen(t)
	s.activateBrowser(s.browser.GetCurrentNode())
	return s
}
func TestDefinitionReadonlyDDLAndVisibleTabNavigation(t *testing.T) {
	s := definitionScreen(t)
	sql := s.editor.GetText()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModAlt))
	if page, _ := s.pages.GetFrontPage(); page != "definition" {
		t.Fatal("DDL window missing")
	}
	if s.statusKey != "definition_loaded" {
		t.Fatal("finished metadata still showed loading")
	}
	ddl := s.definitionDDL
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if s.clipboardText != ddl || s.editor.GetText() != sql || len(s.jobs) != 0 {
		t.Fatal("preview mutated editor or submitted SQL")
	}
	for i := 0; i < 8; i++ {
		s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
		if s.definitionTable.HasFocus() {
			t.Fatal("DDL Tab focused invisible table")
		}
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "object" || !s.objectTable.HasFocus() {
		t.Fatal("preview close lost object focus")
	}
}
func TestDefinitionRawRowsConditionsAndErrors(t *testing.T) {
	s := definitionScreen(t)
	s.showDefinition("constraints")
	data := oracle.ObjectDefinition{Constraints: []oracle.Constraint{{Name: "검사[red]", Kind: "C", Status: "ENABLED", Columns: []string{"이름"}, Condition: "\"이름\" <> 'a\tb'"}}, Limited: true}
	s.acceptDefinition(s.definitionObject, "constraints", data, nil)
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if !strings.Contains(s.clipboardText, "검사[red]") || !strings.Contains(s.definitionInfo.GetText(false), s.tr("definition_limited")) {
		t.Fatal("raw copy/limit notice missing")
	}
	s.app.SetFocus(s.definitionText)
	s.copyDefinition()
	if !strings.Contains(s.clipboardText, data.Constraints[0].Condition) {
		t.Fatal("condition detail missing")
	}
	buttons := s.definitionRoot.GetItem(s.definitionRoot.GetItemCount() - 1).(*tview.Flex)
	copyButton := buttons.GetItem(0).(*tview.Button)
	s.app.SetFocus(copyButton)
	copyButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if !strings.Contains(s.clipboardText, data.Constraints[0].Condition) {
		t.Fatal("copy button lost last detail focus")
	}
	s.showDefinition("ddl")
	s.acceptDefinition(s.definitionObject, "ddl", oracle.ObjectDefinition{}, errors.New("ORA-31603 permission"))
	old := s.clipboardText
	s.copyDefinition()
	if !strings.Contains(s.definitionText.GetText(false), "ORA-31603") || s.clipboardText != old {
		t.Fatal("error became a copied DDL")
	}
}
func TestDefinitionLazyRefreshAndHelpKeepFocus(t *testing.T) {
	s := definitionScreen(t)
	s.showDefinition("ddl")
	s.demo = false
	s.connected = true
	s.showDefinition("constraints")
	j := <-s.jobs
	if s.definitionText.GetText(false) != s.tr("browser_loading") {
		t.Fatal("pending metadata was presented as a final empty result")
	}
	if j.action != "definition" || j.text != "constraints" || j.object != s.object {
		t.Fatal("wrong metadata request")
	}
	s.showHelp()
	focus := s.app.GetFocus()
	s.busy = false
	s.acceptDefinition(j.object, j.text, oracle.ObjectDefinition{}, nil)
	if s.app.GetFocus() != focus {
		t.Fatal("metadata completion stole help focus")
	}
	s.closeHelp()
	if s.app.GetFocus() != s.definitionTable {
		t.Fatal("return from help focused a replaced hidden widget")
	}
	s.showDefinition("ddl")
	if len(s.jobs) != 0 {
		t.Fatal("cached DDL repeated DB request")
	}
	s.refreshDefinition()
	j = <-s.jobs
	if j.text != "ddl" {
		t.Fatal("refresh targeted wrong section")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "definition" {
		t.Fatal("busy escape closed preview instead of cancelling")
	}
}
