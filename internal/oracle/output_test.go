package oracle

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	go_ora "github.com/sijms/go-ora/v3"
)

type outputConverter struct {
	batches [][]string
	index   int
}

func (c *outputConverter) ConvertValue(value any) (driver.Value, error) {
	switch out := value.(type) {
	case go_ora.Out:
		if c.index < len(c.batches) {
			*out.Dest.(*[]string) = c.batches[c.index]
		}
		c.index++
		return int64(0), nil
	default:
		return driver.DefaultParameterConverter.ConvertValue(value)
	}
}

func outputSession(t *testing.T, batches ...[]string) (*Session, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(&outputConverter{batches: batches}))
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
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return s, mock
}

type outputCount int

func (n outputCount) Match(value driver.Value) bool {
	out, ok := value.(sql.Out)
	if !ok {
		return false
	}
	*out.Dest.(*int) = int(n)
	return true
}

func expectOutput(mock sqlmock.Sqlmock, count int) *sqlmock.ExpectedExec {
	return mock.ExpectExec(regexp.QuoteMeta("BEGIN DBMS_OUTPUT.GET_LINES(:1, :2); END;")).WithArgs(sqlmock.AnyArg(), outputCount(count))
}

func TestOutputControlDoesNotResolveOrCreateTransactions(t *testing.T) {
	s, mock := outputSession(t)
	mock.ExpectExec(regexp.QuoteMeta(outputBufferSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := s.SetOutput(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if !s.OutputEnabled() || s.tx != nil || s.PendingChanges() {
		t.Fatal("output control created transaction work")
	}
	mock.ExpectBegin()
	var err error
	s.tx, err = s.conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.pending = true
	mock.ExpectExec("DBMS_OUTPUT.DISABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	if err = s.SetOutput(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if s.OutputEnabled() || !s.PendingChanges() || s.tx == nil {
		t.Fatal("output control resolved pending user work")
	}
	mock.ExpectRollback()
}

func TestOutputPreservesSQLFailureAndSeparatesCollectionFailure(t *testing.T) {
	s, mock := outputSession(t)
	s.outputEnabled = true
	mock.ExpectBegin()
	sqlErr := errors.New("original SQL error")
	mock.ExpectExec("BEGIN broken").WillReturnError(sqlErr)
	expectOutput(mock, 0).WillReturnError(errors.New("output read error"))
	result, err := s.Execute(context.Background(), "BEGIN broken; END;")
	if !errors.Is(err, sqlErr) || result.Output.Error != "output read error" || !s.PendingChanges() || !s.outputReset {
		t.Fatalf("execution and collection outcomes mixed: %+v %v", result.Output, err)
	}
	mock.ExpectExec(regexp.QuoteMeta(outputBufferSQL)).WillReturnError(errors.New("cannot reset"))
	if _, err = s.Execute(context.Background(), "DELETE FROM items"); err == nil || !strings.Contains(err.Error(), "clear incomplete") {
		t.Fatal("new SQL ran with uncollected output from an earlier statement")
	}
	mock.ExpectRollback()
}

func TestOutputCollectionUsesFreshContextAfterCancellation(t *testing.T) {
	s, mock := outputSession(t, []string{"before cancel"})
	s.outputEnabled = true
	mock.ExpectBegin()
	expectOutput(mock, 1).WillReturnResult(sqlmock.NewResult(0, 0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := s.Execute(ctx, "BEGIN NULL; END;")
	if !errors.Is(err, context.Canceled) || result.Output.Error != "" || len(result.Output.Lines) != 1 || result.Output.Lines[0] != "before cancel" {
		t.Fatalf("cancelled SQL context prevented collection: %+v %v", result.Output, err)
	}
	mock.ExpectRollback()
}

func TestOutputReadErrorDoesNotTurnSuccessfulDMLIntoFailure(t *testing.T) {
	s, mock := outputSession(t)
	s.outputEnabled = true
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 3))
	expectOutput(mock, 0).WillReturnError(errors.New("read unavailable"))
	result, err := s.Execute(context.Background(), "UPDATE items SET n=1")
	if err != nil || result.Affected != 3 || result.Output.Error != "read unavailable" || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatalf("collection changed successful user work: %+v %v", result, err)
	}
	mock.ExpectRollback()
}

func TestOutputBoundedBatchesAndUnreadRemainderPurge(t *testing.T) {
	for _, byBytes := range []bool{false, true} {
		line := ""
		batches := make([][]string, 16)
		if byBytes {
			line = strings.Repeat("한", 1000)
			batches = batches[:1]
		}
		for i := range batches {
			for j := 0; j < outputBatch; j++ {
				batches[i] = append(batches[i], line)
			}
		}
		s, mock := outputSession(t, batches...)
		s.outputEnabled = true
		for range batches {
			expectOutput(mock, outputBatch).WillReturnResult(sqlmock.NewResult(0, 0))
		}
		mock.ExpectExec(regexp.QuoteMeta(outputBufferSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
		out := s.collectOutput()
		used := 0
		for _, text := range out.Lines {
			used += len(text) + 1
		}
		if !out.Truncated || out.Error != "" || len(out.Lines) > MaxOutputLines || used > MaxOutputBytes || s.outputReset {
			t.Fatalf("output limit/purge: lines=%d bytes=%d truncated=%v err=%s", len(out.Lines), used, out.Truncated, out.Error)
		}
		if !byBytes && len(out.Lines) != MaxOutputLines {
			t.Fatal("line limit did not retain expected prefix")
		}
	}
}
