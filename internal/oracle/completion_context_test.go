package oracle

import (
	"strings"
	"testing"
)

func completionContext(t *testing.T, text string) CompletionContext {
	t.Helper()
	cursor := strings.Index(text, "|")
	if cursor < 0 {
		t.Fatal("missing cursor marker")
	}
	text = strings.Replace(text, "|", "", 1)
	context, err := SQLCompletionContext(text, cursor)
	if err != nil {
		t.Fatal(err)
	}
	return context
}
func TestCompletionContextQuotesAliasesAndStatementIsolation(t *testing.T) {
	for _, test := range []struct {
		sql, prefix, qualifier, name, alias string
		quoted                              bool
	}{
		{`SELECT s.ti| FROM app.sample_rows s`, "ti", "S", "SAMPLE_ROWS", "S", false},
		{`SELECT "별칭"."이|름" FROM "다른"."테이블" "별칭"`, "이", "별칭", "테이블", "별칭", true},
		{`SELECT s.| FROM sample_rows s`, "", "S", "SAMPLE_ROWS", "S", false},
		{`SELECT old.x FROM previous old; SELECT n.na| FROM newer n`, "na", "N", "NEWER", "N", false},
	} {
		c := completionContext(t, test.sql)
		if c.Prefix != test.prefix || c.Quoted != test.quoted || len(c.Qualifier) != 1 || c.Qualifier[0].Name != test.qualifier || len(c.Sources) != 1 || c.Sources[0].Name.Name != test.name || c.Sources[0].Alias.Name != test.alias {
			t.Fatalf("%s: %+v", test.sql, c)
		}
		text := strings.Replace(test.sql, "|", "", 1)
		replaced := text[:c.Start] + `"chosen"` + text[c.End:]
		if !strings.Contains(replaced, `."chosen" FROM`) {
			t.Fatalf("replacement consumed SQL: %s", replaced)
		}
	}
	c := completionContext(t, `SELECT * FROM app.sam|ple_rows`)
	if !c.ObjectsOnly || c.Prefix != "sam" || c.Qualifier[0].Name != "APP" {
		t.Fatalf("objects: %+v", c)
	}
}
func TestCompletionContextBlocksLiteralsCommentsAndBinds(t *testing.T) {
	for _, text := range []string{`SELECT 'lit|eral' FROM dual`, `SELECT q'[x; FROM bad b |]' FROM dual`, `SELECT nq'!text |!' FROM dual`, `-- SELECT |`, "SELECT /* x | */ 1 FROM dual", `SELECT :pa|ram FROM dual`, `SELECT 'unfinished|`, `SELECT /* unfinished|`} {
		if c := completionContext(t, text); !c.Blocked {
			t.Fatalf("not blocked: %s %+v", text, c)
		}
	}
	c := completionContext(t, `SELECT 'literal', s.| FROM sample_rows s -- pretend FROM wrong w`)
	if c.Blocked || len(c.Sources) != 1 || c.Sources[0].Name.Name != "SAMPLE_ROWS" {
		t.Fatalf("opaque tokens changed aliases: %+v", c)
	}
}
func TestCompletionContextIsolatesSubqueriesAndUnionBranches(t *testing.T) {
	c := completionContext(t, `SELECT a.id, (SELECT b.ti| FROM inner_table b) FROM outer_table a`)
	if len(c.Sources) != 1 || c.Sources[0].Name.Name != "INNER_TABLE" {
		t.Fatalf("subquery: %+v", c)
	}
	c = completionContext(t, `SELECT a.| FROM outer_table a WHERE EXISTS (SELECT 1 FROM unrelated a)`)
	if len(c.Sources) != 1 || c.Sources[0].Name.Name != "OUTER_TABLE" {
		t.Fatalf("outer scope: %+v", c)
	}
	c = completionContext(t, `SELECT a.| FROM first_table a UNION ALL SELECT a.id FROM second_table a`)
	if len(c.Sources) != 1 || c.Sources[0].Name.Name != "FIRST_TABLE" {
		t.Fatalf("union: %+v", c)
	}
	c = completionContext(t, `WITH c AS (SELECT 1 id FROM dual) SELECT c.| FROM c`)
	if len(c.Sources) != 1 || !c.Sources[0].Derived {
		t.Fatalf("CTE guessed as real table: %+v", c)
	}
	c = completionContext(t, `SELECT d.| FROM (SELECT 1 id FROM dual) d`)
	if len(c.Sources) != 1 || !c.Sources[0].Derived || c.Sources[0].Alias.Name != "D" {
		t.Fatalf("derived: %+v", c)
	}
}
func TestCompletionCTEColumnListsAndSelectInto(t *testing.T) {
	for _, text := range []string{
		`WITH c(id) AS (SELECT 1 FROM dual) SELECT c.| FROM c`,
		`WITH a AS (SELECT 1 id FROM dual), c(id) AS (SELECT 2 FROM dual) SELECT c.| FROM c`,
		`WITH c(id) AS (SELECT 1 FROM dual) SELECT 1 FROM dual UNION ALL SELECT c.| FROM c`,
		`WITH c(id) AS (SELECT 1 FROM dual) SELECT * FROM (SELECT c.| FROM c)`,
	} {
		c := completionContext(t, text)
		if len(c.Sources) != 1 || !c.Sources[0].Derived {
			t.Fatalf("CTE resolved as physical table: %s %+v", text, c)
		}
	}
	c := completionContext(t, `SELECT s.ti| INTO target_variable FROM sample_rows s`)
	if len(c.Sources) != 1 || c.Sources[0].Name.Name != "SAMPLE_ROWS" {
		t.Fatalf("INTO variable resolved as table: %+v", c)
	}
}

func TestCompletionContextUnclosedIdentifierAndUTF8Offsets(t *testing.T) {
	c := completionContext(t, `SELECT "이름|`)
	if c.Blocked || !c.Quoted || c.Prefix != "이름" || c.End-c.Start != len(`"이름`) {
		t.Fatalf("%+v", c)
	}
	if _, err := SQLCompletionContext("한글", 1); err == nil {
		t.Fatal("accepted partial UTF-8 cursor")
	}
}
func BenchmarkCompletionLongFile(b *testing.B) {
	text := strings.Repeat("SELECT id FROM prior_table;\n", 80000) + "SELECT s.ti FROM sample_rows s"
	cursor := len(text) - len(" FROM sample_rows s")
	b.ReportAllocs()
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		if _, err := SQLCompletionContext(text, cursor); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCompletionAtWordStartReplacesWholeToken(t *testing.T) {
	for _, sql := range []string{`SELECT s.|title FROM sample_rows s`, `SELECT |title FROM sample_rows`, `SELECT "s".|"이름" FROM "테이블" "s"`} {
		c := completionContext(t, sql)
		if c.Blocked || c.Start != strings.Index(sql, "|") || c.End <= c.Start || c.Prefix != "" {
			t.Fatalf("start of token: %+v", c)
		}
		original := strings.Replace(sql, "|", "", 1)
		replaced := original[:c.Start] + `"picked"` + original[c.End:]
		if strings.Contains(replaced, "title") || strings.Contains(replaced, "이름") {
			t.Fatalf("left existing token suffix: %s", replaced)
		}
	}
}
