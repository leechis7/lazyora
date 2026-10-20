package oracle

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestQueryReplaySnapshotAndValidation(t *testing.T) {
	values := map[string]BindValue{"N": {Kind: "text", Text: "original"}, "UNUSED": {Text: "do not retain"}}
	spec, err := NewQuerySpec("SELECT :n, q'[FOR UPDATE]' FROM dual;", values)
	if err != nil {
		t.Fatal(err)
	}
	values["N"] = BindValue{Kind: "text", Text: "changed"}
	if len(spec.Bindings) != 1 || spec.Bindings["N"].Text != "original" {
		t.Fatal("replay aliases bind memory or retains unrelated values")
	}
	for _, text := range []string{"DELETE FROM t", "SELECT 1 FROM dual; DELETE FROM t", "SELECT * FROM t FOR /* lock */ UPDATE", "BEGIN NULL; END;"} {
		if _, err := NewQuerySpec(text, nil); err == nil {
			t.Errorf("accepted replay: %s", text)
		}
	}
	if _, err := NewQuerySpec("SELECT :missing FROM dual", nil); err == nil {
		t.Fatal("accepted missing binds")
	}
}

func TestQueryStreamKeepsAllRowsAndTransaction(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items SET n = 9").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(context.Background(), "UPDATE items SET n = 9"); err != nil {
		t.Fatal(err)
	}
	journal := len(s.PendingWork().Statements)
	rows := sqlmock.NewRowsWithColumnDefinition(sqlmock.NewColumn("N").OfType("NUMBER", 0), sqlmock.NewColumn("TEXT").OfType("VARCHAR2", ""))
	for i := 0; i < 2501; i++ {
		rows.AddRow(i, nil)
	}
	mock.ExpectQuery("SELECT n, text FROM items WHERE name=:LAZYORA_B1").WithArgs("한글").WillReturnRows(rows).RowsWillBeClosed()
	spec, _ := NewQuerySpec("SELECT n, text FROM items WHERE name=:name", map[string]BindValue{"NAME": {Kind: "text", Text: "한글"}})
	cursor, err := s.OpenQuery(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for {
		row, err := cursor.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if row[0].Type != "NUMBER" || !row[1].Null {
			t.Fatal("metadata lost")
		}
		count++
	}
	if err = cursor.Close(); err != nil {
		t.Fatal(err)
	}
	if count != 2501 || !s.PendingChanges() || len(s.PendingWork().Statements) != journal {
		t.Fatal("stream truncated or changed pending work")
	}
	mock.ExpectRollback()
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}

func TestQueryStreamCancellationAndScanErrorsCloseCursor(t *testing.T) {
	for _, cancelled := range []bool{true, false} {
		t.Run(map[bool]string{true: "cancel", false: "row error"}[cancelled], func(t *testing.T) {
			s, mock := mockSession(t)
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE items SET n = 1").WillReturnResult(sqlmock.NewResult(0, 1))
			if _, err := s.Execute(context.Background(), "UPDATE items SET n = 1"); err != nil {
				t.Fatal(err)
			}
			rows := sqlmock.NewRows([]string{"N"}).AddRow(1).AddRow(2)
			if !cancelled {
				rows.RowError(1, errors.New("network read failure"))
			}
			mock.ExpectQuery("SELECT n FROM items").WillReturnRows(rows).RowsWillBeClosed()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			spec, _ := NewQuerySpec("SELECT n FROM items", nil)
			cursor, err := s.OpenQuery(ctx, spec)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = cursor.Next(); err != nil {
				t.Fatal(err)
			}
			if cancelled {
				cancel()
			}
			if _, err = cursor.Next(); err == nil || err == io.EOF {
				t.Fatal("partial result reported complete")
			}
			cursor.Close()
			// No implicit transaction resolution on cursor failure/cancellation.
			mock.ExpectExec("UPDATE items SET n = 2").WillReturnResult(sqlmock.NewResult(0, 1))
			if _, err = s.Execute(context.Background(), "UPDATE items SET n = 2"); err != nil {
				t.Fatal(err)
			}
			mock.ExpectRollback()
			if err = s.Finish(false); err != nil {
				t.Fatal(err)
			}
		})
	}
}
