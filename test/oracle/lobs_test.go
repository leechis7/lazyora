package oracle_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"lazyora/internal/oracle"
	"strings"
	"testing"
	"time"
)

func TestOracleLOBsIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	_, err := s.Execute(ctx, `CREATE TABLE lob_fixture(c CLOB,b BLOB)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Finish(false)
		if _, err := s.Execute(context.Background(), "DROP TABLE lob_fixture PURGE"); err != nil {
			t.Error(err)
		}
	})
	// Keep a surrogate pair across a 32768-unit read boundary.
	text := strings.Repeat("x", 32767) + "👩" + strings.Repeat("한글", 150000)
	binary := bytes.Repeat([]byte{0, 1, 255, 128}, 80000)
	_, err = s.Execute(ctx, `DECLARE c CLOB; b BLOB; chunk VARCHAR2(4000); raw_chunk RAW(4000); BEGIN
 INSERT INTO lob_fixture VALUES(EMPTY_CLOB(),EMPTY_BLOB()) RETURNING c,b INTO c,b;
 FOR i IN 1..10 LOOP DBMS_LOB.WRITEAPPEND(c,3000,RPAD('x',3000,'x')); END LOOP;
 DBMS_LOB.WRITEAPPEND(c,2767,RPAD('x',2767,'x'));
 DBMS_LOB.APPEND(c,TO_CLOB('👩'));
 FOR i IN 1..300 LOOP DBMS_LOB.APPEND(c,TO_CLOB(RPAD('한글',2000,'한글'))); END LOOP;
 raw_chunk:=HEXTORAW(RPAD('0001FF80',8000,'0001FF80'));
 FOR i IN 1..80 LOOP DBMS_LOB.WRITEAPPEND(b,4000,raw_chunk); END LOOP;
 END;`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.ExecutePaged(ctx, "SELECT c,b FROM lob_fixture")
	if err != nil || len(result.Values) != 1 {
		t.Fatalf("LOB fetch: %v", err)
	}
	for i, want := range [][]byte{[]byte(text), binary} {
		cell := result.Values[0][i]
		if !cell.Partial || cell.LOBID == 0 {
			t.Fatalf("not bounded: %+v", cell)
		}
		var out bytes.Buffer
		n, err := s.WriteCell(ctx, &out, cell)
		if err != nil || n != int64(len(want)) || !bytes.Equal(out.Bytes(), want) {
			t.Fatalf("LOB %d exact stream: bytes %d/%d err %v", i, n, len(want), err)
		}
		detail, err := s.PreviewCell(ctx, cell)
		if err != nil || !detail.Partial || len(detail.Text) <= len(cell.Text) {
			t.Fatalf("preview: %v", err)
		}
	}
	if !s.PendingChanges() {
		t.Fatal("LOB reads resolved transaction")
	}
	saved := result.Values[0][0]
	if _, err = s.Execute(ctx, "SELECT 1 FROM dual"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.WriteCell(ctx, io.Discard, saved); err == nil {
		t.Fatal("new SQL retained stale locator")
	}
	small, err := s.Execute(ctx, "SELECT TO_CLOB('한글👩'), EMPTY_CLOB(), TO_CLOB(NULL), TO_BLOB(HEXTORAW('0001FF')), EMPTY_BLOB(), TO_BLOB(NULL) FROM dual")
	if err != nil {
		t.Fatal(err)
	}
	cells := small.Values[0]
	if cells[0].Text != "한글👩" || cells[0].Type != "CLOB" || cells[1].Null || !cells[2].Null || cells[3].Text != hex.EncodeToString([]byte{0, 1, 255}) || cells[3].Type != "BLOB" || cells[4].Null || !cells[5].Null {
		t.Fatalf("typed NULL/empty: %+v", cells)
	}
	rows, err := s.OpenQuery(ctx, &oracle.QuerySpec{Text: "SELECT c,b FROM lob_fixture"})
	if err != nil {
		t.Fatal(err)
	}
	values, err := rows.Next()
	if err != nil {
		rows.Close()
		t.Fatal(err)
	}
	if values[0].Text != text || values[1].Text != fmt.Sprintf("%x", binary) {
		t.Fatal("full export values changed")
	}
	if err = rows.Close(); err != nil {
		t.Fatal(err)
	}
}
