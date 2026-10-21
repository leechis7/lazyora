package oracle

import (
	"strings"
	"testing"
)

func TestFormatSQLPreservesOracleLexicalUnits(t *testing.T) {
	tests := []string{
		`select a,b,nvl(c,0) as value from tab t where a=:name and b>=1.25e-3 order by a,b;`,
		`SELECT N'한글',q'[x; FROM fake f :name -- quote]',nq'!한글 '' ;!',"Mixed.Name",1.e+2,.5f FROM "Quoted.Owner"."T" WHERE a<>b AND c!=d AND e||f='text';`,
		"SELECT /*+ INDEX(t idx) */ a, -- keep; :secret\n b FROM t /* multiline\n  unchanged  */ WHERE x BETWEEN 1 AND 2;",
		`DECLARE n NUMBER := 1; BEGIN IF n>=1 THEN n:=n+1; ELSIF n=0 THEN n:=2; ELSE n:=3; END IF; FOR i IN 1..10 LOOP DBMS_OUTPUT.PUT_LINE(i); END LOOP; EXCEPTION WHEN OTHERS THEN RAISE; END;`,
		`CREATE OR REPLACE FUNCTION "O"."F"(p NUMBER DEFAULT 1) RETURN VARCHAR2 IS v VARCHAR2(100); BEGIN v:=q'[literal with newline]'; RETURN v; END;`,
		"BEGIN NULL; END;\n/\nSELECT 1 FROM dual;\nBEGIN DBMS_OUTPUT.PUT_LINE('x'); END;\n/\n",
		`SELECT CASE WHEN a=1 THEN 'a' ELSE 'b' END AS value,COUNT(CASE WHEN b=2 THEN 1 END) FROM t WHERE EXISTS (SELECT 1 FROM u WHERE u.id=t.id);`,
		`WITH c AS (SELECT id,name FROM t),d(x) AS (SELECT id FROM u) SELECT c.id,d.x FROM c LEFT OUTER JOIN d ON c.id=d.x;`,
		`CREATE TABLE t(id NUMBER(10),name VARCHAR2(30),PRIMARY KEY(id));`,
		`SELECT a.id FROM a,b WHERE a.id=b.id(+) AND a.x=- -1;`,
		`BEGIN :output:=SQL%ROWCOUNT; OPEN :cursor FOR SELECT :1 FROM dual; END;`,
		"SELECT 'first\r\nsecond',\"한글\" FROM dual; -- trailing comment  ",
	}
	for _, text := range tests {
		formatted, err := FormatSQL(text)
		if err != nil {
			t.Errorf("input %s\nerror %v", text, err)
			continue
		}
		again, err := FormatSQL(formatted)
		if err != nil || again != formatted {
			t.Errorf("not idempotent:\n%s\nagain:\n%s\nerr %v", formatted, again, err)
		}
		if !strings.Contains(formatted, "\n") {
			t.Errorf("no formatting: %s", formatted)
		}
	}
}
func TestFormatSQLReadableSelectAndPLSQL(t *testing.T) {
	formatted, err := FormatSQL("SELECT a,b FROM t WHERE id=:id AND name='한글';")
	want := "SELECT\n  a,\n  b\nFROM t\nWHERE id = :id\n  AND name = '한글';"
	if err != nil || formatted != want {
		t.Fatalf("got:\n%s\nwant:\n%s\nerr %v", formatted, want, err)
	}
	formatted, err = FormatSQL("BEGIN IF x=1 THEN y:=2; ELSE y:=3; END IF; END;")
	if err != nil || !strings.Contains(formatted, "BEGIN\n  IF x = 1 THEN\n    y := 2;\n  ELSE\n    y := 3;\n  END IF;\nEND;") {
		t.Fatalf("PLSQL:\n%s\n%v", formatted, err)
	}
}
func TestFormatSQLKeepsScriptBoundariesAndOpaqueBytes(t *testing.T) {
	text := "CREATE OR REPLACE PROCEDURE p AS BEGIN DBMS_OUTPUT.PUT_LINE(q'[a; / -- literal]'); END;\n/\nSELECT N'한글',1.2 FROM dual;\n"
	formatted, err := FormatSQL(text)
	if err != nil {
		t.Fatal(err)
	}
	before, err := Statements(text)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Statements(formatted)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 2 || len(after) != 2 {
		t.Fatalf("unit boundaries: %d -> %d\n%s", len(before), len(after), formatted)
	}
	for i := range before {
		b, _ := formatAtoms(before[i].Text)
		a, _ := formatAtoms(after[i].Text)
		if len(a) != len(b) {
			t.Fatalf("statement %d changed", i)
		}
	}
	for _, value := range []string{"q'[a; / -- literal]'", "N'한글'", "1.2"} {
		if !strings.Contains(formatted, value) {
			t.Fatal(value)
		}
	}
}
func TestFormatSQLRejectsUnclosedOrOversizedInput(t *testing.T) {
	for _, text := range []string{"SELECT 'unfinished", "SELECT q'[unfinished", "SELECT /* unfinished", strings.Repeat(" ", MaxFormatBytes+1)} {
		if out, err := FormatSQL(text); err == nil || out != "" {
			t.Fatalf("unsafe format accepted: %v", err)
		}
	}
}

func TestFormatSQLRangePreservesSurroundingCodeAndIndent(t *testing.T) {
	text := "BEGIN\n    SELECT a,b FROM t;\n  NULL;\nEND;"
	start := strings.Index(text, "SELECT")
	end := strings.Index(text, ";\n") + 1
	replacement, err := FormatSQLRange(text, start, end)
	if err != nil || replacement != "SELECT\n      a,\n      b\n    FROM t;" {
		t.Fatalf("%q %v", replacement, err)
	}
	updated := text[:start] + replacement + text[end:]
	if !strings.HasPrefix(updated, "BEGIN\n    SELECT") || !strings.HasSuffix(updated, "\n  NULL;\nEND;") {
		t.Fatal(updated)
	}
	text = "SELECT a,b FROM t"
	replacement, err = FormatSQLRange(text, len("SELECT"), len(text))
	if err != nil || !strings.HasPrefix(replacement, " ") {
		t.Fatalf("boundary whitespace lost: %q %v", replacement, err)
	}
	for _, source := range []string{"SELECT 'a,b' FROM t", "SELECT q'[a,b]' FROM t", "SELECT 1 -- a,b\nFROM t", "SELECT /* a,b */ 1 FROM t"} {
		from := strings.Index(source, "a,b")
		if _, err := FormatSQLRange(source, from, from+3); err == nil {
			t.Fatal("formatted inside opaque token", source)
		}
	}
}
func TestFormatSQLFunctionsRangesAndNestedCaseRemainStable(t *testing.T) {
	for _, text := range []string{
		`SELECT EXTRACT(YEAR FROM d),TRIM(BOTH 'x' FROM name),SUM(n) OVER(PARTITION BY a ORDER BY b) FROM t;`,
		`BEGIN IF CASE WHEN x=1 THEN TRUE ELSE FALSE END THEN NULL; END IF; END;`,
		`BEGIN FOR i IN 1..10 LOOP NULL; END LOOP; END;`,
		`BEGIN CASE x WHEN 1 THEN IF y=2 THEN NULL; END IF; ELSE NULL; END CASE; END;`,
	} {
		formatted, err := FormatSQL(text)
		if err != nil {
			t.Fatal(err)
		}
		again, err := FormatSQL(formatted)
		if err != nil || again != formatted {
			t.Fatalf("unstable:\n%s\n%s\n%v", formatted, again, err)
		}
	}
	text := "SELECT a,b FROM t"
	formatted, err := FormatSQL(text)
	if err != nil {
		t.Fatal(err)
	}
	mapped := FormattedCursor(text, formatted, len("SELECT a"))
	if mapped < 1 || formatted[mapped-1] != 'a' {
		t.Fatalf("cursor mapped to %d in %q", mapped, formatted)
	}
}

func TestFormatDepthLimitAlsoAppliesAtEndOfInput(t *testing.T) {
	if _, err := FormatSQL(strings.Repeat("(", maxFormatDepth+1)); err == nil {
		t.Fatal("accepted oversized incomplete nesting")
	}
}
func BenchmarkFormatSQLLarge(b *testing.B) {
	text := strings.Repeat("SELECT a,b,NVL(c,0) FROM t WHERE id=:id;\n", 1000)
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		if _, err := FormatSQL(text); err != nil {
			b.Fatal(err)
		}
	}
}

func TestFormatCannotRepairInvalidBindAdjacencySilently(t *testing.T) {
	if _, err := FormatSQL("SELECT : name FROM dual"); err == nil {
		t.Fatal("silently changed bind syntax")
	}
}
