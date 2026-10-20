package oracle

import (
	"strings"
	"testing"
	"unicode"
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

func TestStatementLineNumbersAndByteOffsets(t *testing.T) {
	text := "-- 한글\r\n\r\nSELECT q'[여러\n줄; /]' FROM dual;\r\n" +
		"/* 주석\n여러 줄 */\r\nSELECT nq'{다음;}' FROM dual;\r\n" +
		"BEGIN\r\n NULL;\r\nEND;\r\n/ -- 끝\r\n\r\nSELECT 4 FROM dual;"
	got, err := Statements(text)
	if err != nil || len(got) != 4 {
		t.Fatalf("statements=%v err=%v", got, err)
	}
	for i, wantLine := range []int{3, 7, 8, 13} {
		if got[i].Line != wantLine || got[i].Text != strings.TrimSpace(text[got[i].Start:got[i].End]) {
			t.Fatalf("statement %d: %+v, want line %d", i, got[i], wantLine)
		}
	}
	if _, err := Statements("-- 한글\nSELECT (1\n/ -- delimiter\n"); err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("wrong error line: %v", err)
	}
}

func TestLargeStatementFilesKeepBoundaries(t *testing.T) {
	const count = 80000
	const statement = "SELECT 1 FROM dual;\n"
	text := strings.Repeat(statement, count)
	got, err := Statements(text)
	if err != nil || len(got) != count {
		t.Fatalf("count=%d err=%v", len(got), err)
	}
	for i, stmt := range got {
		start := 0
		if i > 0 {
			start = i*len(statement) - 1 // preserve whitespace after the prior semicolon
		}
		if stmt.Line != i+1 || stmt.Start != start || stmt.End != (i+1)*len(statement)-1 {
			t.Fatalf("statement %d lost position: %+v", i, stmt)
		}
	}
	division := "SELECT 1" + strings.Repeat(" / 2", 40000) + " FROM dual;\n/ -- end\nSELECT 2 FROM dual;"
	got, err = Statements(division)
	if err != nil || len(got) != 2 || got[1].Line != 3 {
		t.Fatalf("division split incorrectly: count=%d err=%v", len(got), err)
	}
}

func FuzzStatementOffsets(f *testing.F) {
	for _, text := range []string{
		"SELECT 1 FROM dual;\nSELECT 2 FROM dual;",
		"-- 주석\r\nSELECT nq'[줄\n; /]' FROM dual;\n/ -- 끝\nBEGIN NULL; END;\n/",
		"SELECT 10/2/1 FROM dual\n/\n/* 주석 */ SELECT q'{a;/}' FROM dual;",
	} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		got, err := Statements(text)
		if err != nil {
			return
		}
		previous := 0
		for _, stmt := range got {
			if stmt.Start < previous || stmt.End > len(text) || stmt.End <= stmt.Start {
				t.Fatal("invalid byte offsets")
			}
			raw := text[stmt.Start:stmt.End]
			if stmt.Text != strings.TrimFunc(raw, unicode.IsSpace) {
				t.Fatal("statement text does not match source range")
			}
			tokens, err := sqlTokens(raw)
			if err != nil || len(tokens) == 0 {
				t.Fatal("invalid statement emitted")
			}
			// Independent, deliberately simple reference for the original line rule.
			if want := 1 + strings.Count(text[:stmt.Start+tokens[0].start], "\n"); stmt.Line != want {
				t.Fatalf("line=%d want=%d", stmt.Line, want)
			}
			previous = stmt.End
		}
	})
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
