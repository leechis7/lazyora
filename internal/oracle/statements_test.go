package oracle

import (
	"strings"
	"testing"
)

func TestStatementBoundaries(t *testing.T) {
	for _, tc := range []struct {
		text string
		want int
	}{
		{"SELECT ';' FROM dual; SELECT 2 FROM dual;", 2},
		{"-- ; /\nSELECT q'[a; / '' b]' FROM dual; /* ; */ SELECT nq'{한글;}' FROM dual;", 2},
		{"SELECT \"x;y\" FROM dual; SELECT 'it''s; fine' FROM dual", 2},
		{"BEGIN\n NULL; IF 1=1 THEN NULL; END IF;\nEND;\n/\nSELECT 1 FROM dual;", 2},
		{"DECLARE x NUMBER; BEGIN x:=1; END;\n/\nBEGIN NULL; END;\n/", 2},
		{"CREATE OR REPLACE PACKAGE p AS PROCEDURE f; END;\n/\nCREATE OR REPLACE PACKAGE BODY p AS PROCEDURE f IS BEGIN NULL; END; END;\n/", 2},
		{"CREATE OR REPLACE FUNCTION f RETURN NUMBER IS BEGIN RETURN 1; END;\n/\nSELECT f FROM dual;", 2},
		{"SELECT 10 / 2 FROM dual\n/ -- end\nSELECT 3 FROM dual;", 2},
		{" /* comment only */ -- done", 0},
		{";; SELECT 1 FROM dual;;", 1},
	} {
		got, err := Statements(tc.text)
		if err != nil || len(got) != tc.want {
			t.Errorf("%q: got %v, %v", tc.text, got, err)
		}
		for _, stmt := range got {
			if stmt.Text != strings.TrimSpace(tc.text[stmt.Start:stmt.End]) || stmt.Line < 1 {
				t.Fatal("source offsets lost")
			}
		}
	}
}

func TestStatementCursorAndInvalidInput(t *testing.T) {
	text := "-- 한글\nSELECT 1 FROM dual;\n\n-- next\nSELECT 2 FROM dual;\n"
	for _, tc := range []struct {
		offset   int
		contains string
	}{
		{0, "SELECT 1"}, {strings.Index(text, ";"), "SELECT 1"}, {strings.Index(text, ";") + 1, "SELECT 2"}, {strings.Index(text, "next"), "SELECT 2"}, {len(text), "SELECT 2"},
	} {
		got, err := AtStatement(text, tc.offset)
		if err != nil || !strings.Contains(got, tc.contains) {
			t.Fatalf("offset %d: %q %v", tc.offset, got, err)
		}
	}
	for _, bad := range []string{"SELECT 'oops", "SELECT q'[oops'", "SELECT 1 /* oops", "SELECT (1 FROM dual;", "SELECT ) FROM dual;"} {
		if _, err := AtStatement(bad, 0); err == nil {
			t.Errorf("accepted ambiguous SQL %q", bad)
		}
	}
}

func TestPrepareKeepsStoredPLSQLSemicolon(t *testing.T) {
	text := "CREATE OR REPLACE PROCEDURE p IS BEGIN NULL; END;\n/"
	got, query, err := Prepare(text)
	if err != nil || query || !strings.HasSuffix(got, "END;") {
		t.Fatalf("%q %v %v", got, query, err)
	}
}
