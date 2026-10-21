package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"lazyora/internal/oracle"
)

func TestRelationNavigationAndBackPreserveOriginalObjectAndSQL(t *testing.T) {
	s := definitionScreen(t)
	original, sql := s.object, s.editor.GetText()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'g', tcell.ModAlt))
	if s.definitionSection != "grants" {
		t.Fatal("object grant shortcut not connected")
	}
	s.showDefinition("dependents")
	target := oracle.Object{Owner: "APP", Name: "잘못된[red]", Kind: "VIEW", Status: "INVALID"}
	data := oracle.ObjectDefinition{Relations: []oracle.ObjectRelation{{Object: target, Dependency: "HARD", Visible: true}}}
	s.acceptDefinition(s.definitionObject, "dependents", data, nil)
	s.copyDefinition()
	if !strings.Contains(s.clipboardText, "잘못된[red]") || !strings.Contains(s.clipboardText, "INVALID") {
		t.Fatal("raw target/status copy missing")
	}
	s.openDefinitionRelation()
	if s.definitionObject != target || len(s.definitionTrail) != 1 || s.object != original || s.editor.GetText() != sql {
		t.Fatal("navigation changed original object/editor")
	}
	s.key(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModAlt))
	if s.definitionObject != original || s.definitionSection != "dependents" || len(s.definitionRows) != 1 || len(s.definitionTrail) != 0 {
		t.Fatal("back lost metadata/history")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.object != original || !s.objectTable.HasFocus() || len(s.jobs) != 0 {
		t.Fatal("close lost original object focus or executed SQL")
	}
}

func TestRelationsBlockRemoteUnknownAndBusyTargets(t *testing.T) {
	s := definitionScreen(t)
	s.showDefinition("dependencies")
	for _, relation := range []oracle.ObjectRelation{
		{Object: oracle.Object{Owner: "APP", Name: "T", Kind: "TABLE"}, Link: "REMOTE", Visible: true},
		{Object: oracle.Object{Owner: "APP", Name: "T", Kind: "TABLE"}},
	} {
		s.acceptDefinition(s.definitionObject, "dependencies", oracle.ObjectDefinition{Relations: []oracle.ObjectRelation{relation}}, nil)
		before := s.definitionObject
		s.openDefinitionRelation()
		if s.definitionObject != before || len(s.definitionTrail) != 0 || len(s.jobs) != 0 {
			t.Fatal("unavailable target was opened")
		}
	}
	s.acceptDefinition(s.definitionObject, "dependencies", oracle.ObjectDefinition{Relations: []oracle.ObjectRelation{{Object: oracle.Object{Owner: "APP", Name: "T", Kind: "TABLE"}, Visible: true}}}, nil)
	s.busy = true
	s.openDefinitionRelation()
	if len(s.definitionTrail) != 0 {
		t.Fatal("navigated during a DB operation")
	}
}
