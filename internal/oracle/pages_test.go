package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func pageFixture(t *testing.T, count int) (*Session, sqlmock.Sqlmock, Result) {
	t.Helper()
	s, m := mockSession(t)
	m.ExpectBegin()
	m.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(context.Background(), "UPDATE items SET n=1"); err != nil {
		t.Fatal(err)
	}
	rows := sqlmock.NewRows([]string{"N"})
	for i := 0; i < count; i++ {
		rows.AddRow(i)
	}
	m.ExpectQuery("SELECT n FROM items").WillReturnRows(rows).RowsWillBeClosed()
	ctx, cancel := context.WithCancel(context.Background())
	first, err := s.ExecutePaged(ctx, "SELECT n FROM items")
	cancel() // Worker cancels each finished job; the retained cursor must survive.
	if err != nil {
		t.Fatal(err)
	}
	return s, m, first
}

func TestPagedQueryUsesSameCursorAndSurvivesJobContext(t *testing.T) {
	s, m, first := pageFixture(t, 2507)
	if len(first.Rows) != 1000 || first.PageID == 0 || !first.Truncated {
		t.Fatal("initial cursor missing")
	}
	if _, err := s.FetchPage(context.Background(), first.PageID+1); err == nil {
		t.Fatal("stale cursor token accepted")
	}
	second, err := s.FetchPage(context.Background(), first.PageID)
	if err != nil || len(second.Rows) != 1000 || second.Rows[0][0] != "1000" || second.PageID != first.PageID {
		t.Fatalf("second page: %v", err)
	}
	last, err := s.FetchPage(context.Background(), second.PageID)
	if err != nil || len(last.Rows) != 507 || last.Rows[0][0] != "2000" || last.Rows[506][0] != "2506" || last.PageID != 0 || last.Truncated {
		t.Fatalf("last page: %+v %v", last, err)
	}
	if !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatal("fetch changed pending journal/transaction")
	}
	m.ExpectRollback()
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}

func TestPageCancelClosesOnlyCursorAndLimitsRetention(t *testing.T) {
	for _, mode := range []string{"cancel", "bytes", "rows", "commit"} {
		t.Run(mode, func(t *testing.T) {
			s, m, first := pageFixture(t, 1002)
			switch mode {
			case "cancel":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := s.FetchPage(ctx, first.PageID); !errors.Is(err, context.Canceled) {
					t.Fatalf("cancel: %v", err)
				}
			case "bytes", "rows":
				if mode == "bytes" {
					s.page.bytes = MaxGridBytes - 1
				} else {
					s.page.count = MaxGridRows
				}
				page, err := s.FetchPage(context.Background(), first.PageID)
				if err != nil || !page.PageLimited || page.PageID != 0 || len(page.Rows) != 0 {
					t.Fatalf("limit: %+v %v", page, err)
				}
			case "commit":
				m.ExpectCommit()
				if err := s.Finish(true); err != nil {
					t.Fatal(err)
				}
				if s.PageID() != 0 {
					t.Fatal("commit retained cursor")
				}
				return
			}
			if s.PageID() != 0 || !s.PendingChanges() {
				t.Fatal("cursor close resolved DML or leaked")
			}
			m.ExpectRollback()
			if err := s.Finish(false); err != nil {
				t.Fatal(err)
			}
		})
	}
}
