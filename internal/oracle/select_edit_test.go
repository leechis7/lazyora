package oracle

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sijms/go-ora/v3/types"
)

func TestSelectEditAllowsOnlyDirectSingleTableLineage(t *testing.T) {
	for _, sql := range []string{
		"SELECT t.ROWID, t.* FROM scott.emp t WHERE deptno=:n ORDER BY empno DESC",
		"SELECT ROWID AS rid, empno AS id, ename name FROM emp",
		`SELECT e.ROWID, e."한글" AS "표시명" FROM "Mixed"."Table" e WHERE e."한글"=q'[FROM SELECT JOIN]'`,
		"/* SELECT */ SELECT e.rowid, e.id FROM emp e WHERE id IN (1,2)",
	} {
		if parseSelectEdit(sql) == nil {
			t.Errorf("supported query rejected: %s", sql)
		}
	}
	for _, sql := range []string{
		"SELECT * FROM emp", "SELECT 'AAAAAA' AS rowid, id FROM emp",
		"SELECT fake AS rowid, id FROM emp", "SELECT ROWIDTOCHAR(ROWID), id FROM emp",
		"SELECT t.ROWID, d.name FROM emp t", "SELECT ROWID, salary*2 AS salary FROM emp",
		"SELECT ROWID, COUNT(*) FROM emp", "SELECT DISTINCT ROWID, id FROM emp",
		"SELECT ROWID, id FROM emp UNION SELECT ROWID, id FROM other",
		"SELECT e.ROWID, e.id FROM emp e JOIN dept d ON e.id=d.id",
		"SELECT e.ROWID, e.id FROM emp e, dept d", "SELECT ROWID, id FROM (SELECT * FROM emp)",
		"WITH x AS (SELECT * FROM emp) SELECT ROWID,id FROM x",
		"SELECT ROWID, id FROM emp WHERE id IN (SELECT id FROM other)",
		"SELECT ROWID, id FROM emp@remote", "SELECT ROWID, id FROM emp FOR UPDATE",
		"SELECT ROWID, id FROM emp AS OF TIMESTAMP :n", "SELECT ROWID, id FROM emp GROUP BY id,ROWID",
		"SELECT ROWID, ROWID, id FROM emp", "SELECT ROWID, FROM emp", "SELECT ROWID,id FROM emp; DELETE FROM emp",
		"SELECT ROWID, t.* AS x FROM emp t", "SELECT ROWID, id+1 id FROM emp",
	} {
		if parseSelectEdit(sql) != nil {
			t.Errorf("ambiguous query allowed: %s", sql)
		}
	}
}

func expectSelectEditMetadata(m sqlmock.Sqlmock) {
	m.ExpectQuery("CURRENT_SCHEMA").WillReturnRows(sqlmock.NewRows([]string{"SCHEMA"}).AddRow("APP"))
	m.ExpectQuery("SELECT COUNT.*FROM all_tables").WithArgs("APP", "ITEMS").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	m.ExpectQuery("SELECT c.column_name").WithArgs("APP", "ITEMS").WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE", "LEN", "PREC", "SCALE", "CHARLEN", "SEM", "NULL", "COMMENT"}).
		AddRow("ID", "NUMBER", nil, nil, nil, nil, nil, "N", nil).
		AddRow("TITLE", "VARCHAR2", 100, nil, nil, nil, nil, "Y", nil))
}

type editTestConverter struct{}

func (editTestConverter) ConvertValue(value any) (driver.Value, error) {
	if n, ok := value.(*types.Number); ok {
		return n.String()
	}
	if n, ok := value.(types.Number); ok {
		return n.String()
	}
	return driver.DefaultParameterConverter.ConvertValue(value)
}

func selectEditSession(t *testing.T) (*Session, sqlmock.Sqlmock) {
	t.Helper()
	db, m, err := sqlmock.New(sqlmock.ValueConverterOption(editTestConverter{}))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s := &Session{db: db, conn: conn}
	t.Cleanup(func() {
		s.Close()
		if err := m.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return s, m
}

func TestSelectRowIDPagesMapAliasesAndConflictCheckOriginalValues(t *testing.T) {
	s, m := selectEditSession(t)
	m.ExpectBegin()
	expectSelectEditMetadata(m)
	rows := sqlmock.NewRows([]string{"RID", "KEY", "CAPTION"})
	for i := 0; i < 1002; i++ {
		rows.AddRow(fmt.Sprintf("AAAAAAAABAAAAA%04d", i), i, "old")
	}
	m.ExpectQuery("SELECT t.ROWID AS rid").WithArgs("0").WillReturnRows(rows).RowsWillBeClosed()
	r, err := s.ExecutePaged(context.Background(), "SELECT t.ROWID AS rid, t.id AS key, t.title AS caption FROM items t WHERE id>=:n ORDER BY id", map[string]BindValue{"N": {Kind: "number", Text: "0"}})
	if err != nil || r.EditID == 0 || r.PageID == 0 || r.EditRowIDColumn != 1 || r.CanEditColumn(0) || !r.CanEditColumn(2) {
		t.Fatalf("first page: %+v %v", r, err)
	}
	if len(r.EditRowIDs) != 1000 || r.EditRowIDs[999] != "AAAAAAAABAAAAA0999" || r.EditColumnNames[2] != "TITLE" {
		t.Fatal("first page physical identities missing")
	}
	page, err := s.FetchPage(context.Background(), r.PageID)
	if err != nil || page.EditID != r.EditID || page.EditRowIDColumn != 1 || len(s.edit.rows) != 1002 || len(page.Rows) != 2 || page.PageID != 0 {
		t.Fatalf("last page: %+v %v", page, err)
	}
	if len(page.EditRowIDs) != 2 || page.EditRowIDs[0] != "AAAAAAAABAAAAA1000" || page.EditRowIDs[1] != "AAAAAAAABAAAAA1001" {
		t.Fatal("next page identities mismatched")
	}
	if err = s.ApplyEdit(context.Background(), EditChange{ID: r.EditID, Row: 1001, Column: 0, Text: "other"}); err == nil {
		t.Fatal("ROWID allowed editing")
	}
	m.ExpectExec("SAVEPOINT LAZYORA_").WillReturnResult(sqlmock.NewResult(0, 0))
	m.ExpectQuery("SELECT ROWIDTOCHAR").WithArgs("AAAAAAAABAAAAA1001").WillReturnRows(sqlmock.NewRows([]string{"ROWID"}).AddRow("AAAAAAAABAAAAA1001"))
	m.ExpectExec(`UPDATE "APP"\."ITEMS" SET "TITLE" = :new_value WHERE ROWID = CHARTOROWID\(:row_id\).*"ID".*"TITLE"`).WithArgs("new", "AAAAAAAABAAAAA1001", "1001", "old").WillReturnResult(sqlmock.NewResult(0, 0))
	m.ExpectExec("ROLLBACK TO SAVEPOINT").WillReturnResult(sqlmock.NewResult(0, 0))
	if err = s.ApplyEdit(context.Background(), EditChange{ID: r.EditID, Row: 1001, Column: 2, Text: "new"}); !errors.Is(err, ErrEditConflict) {
		t.Fatalf("lost-update conflict: %v", err)
	}
	if s.PendingChanges() || s.EditID() != 0 {
		t.Fatal("conflict changed transaction or retained snapshot")
	}
	m.ExpectRollback()
}

func TestSelectRowIDUnverifiedMetadataAndInvalidRowIDsStayReadOnly(t *testing.T) {
	for _, mode := range []string{"view", "metadata error", "invalid rowid", "duplicate projection"} {
		t.Run(mode, func(t *testing.T) {
			s, m := mockSession(t)
			m.ExpectBegin()
			sql := "SELECT ROWID,id,title FROM items"
			if mode == "duplicate projection" {
				sql = "SELECT ROWID,id,id FROM items"
			}
			if mode == "metadata error" {
				m.ExpectQuery("CURRENT_SCHEMA").WillReturnError(errors.New("dictionary denied"))
			} else if mode == "view" {
				m.ExpectQuery("CURRENT_SCHEMA").WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow("APP"))
				m.ExpectQuery("SELECT COUNT.*FROM all_tables").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
			} else {
				expectSelectEditMetadata(m)
			}
			rid := "AAAAAAAABAAAAA0001"
			if mode == "invalid rowid" {
				rid = "not a physical rowid"
			}
			m.ExpectQuery("SELECT ROWID").WillReturnRows(sqlmock.NewRows([]string{"ROWID", "ID", "TITLE"}).AddRow(rid, 1, "old"))
			r, err := s.ExecutePaged(context.Background(), sql)
			if err != nil || r.EditID != 0 || len(r.Rows) != 1 {
				t.Fatalf("query lost or granted write access: %+v %v", r, err)
			}
			m.ExpectRollback()
		})
	}
}

func TestSelectRowIDDateSnapshotAndUnsupportedColumns(t *testing.T) {
	snap := &editSnapshot{rowIDColumn: 0, columns: []Column{{}, {Name: "D", DataType: "DATE"}, {Name: "TS", DataType: "TIMESTAMP"}, {}}}
	cells := []Cell{{Text: "AAAAAAAABAAAAA0001"}, {Text: "2026-10-22 12:34:56Z", Type: "DATE"}, {Text: "2026-10-22 12:34:56.123456Z", Type: "TIMESTAMP"}, {Text: "LOB preview", LOBID: 1, Partial: true}}
	if !snap.appendSelectRow(cells) || snap.rows[0][1].Text != "2026-10-22 12:34:56" || snap.rows[0][2].Text != "2026-10-22 12:34:56.123456000" {
		t.Fatal("date values not canonical")
	}
	if cells[1].Text != "2026-10-22 12:34:56Z" {
		t.Fatal("display text changed")
	}
}
