package oracle

import (
	"context"
	"database/sql"
	"errors"

	go_ora "github.com/sijms/go-ora/v3"
)

const MaxGridRows = 10000
const MaxGridBytes = 64 * 1024 * 1024 // Estimate includes retained cell/row overhead.

type fetchWatchKey struct{}

type queryPage struct {
	rows         *sql.Rows
	types        []*sql.ColumnType
	columns      []string
	cancel       context.CancelFunc
	id           uint64
	pending      []Cell
	count, bytes int
}

func (s *Session) ClosePage() {
	if s.page != nil {
		page := s.page
		s.page = nil
		page.cancel()
		_ = page.rows.Close()
	}
}

func (s *Session) PageID() uint64 {
	if s.page != nil {
		return s.page.id
	}
	return 0
}

// FetchPage continues one live cursor; stale tabs can never fetch another
// tab's query. A job's timeout cancels this fetch, not the transaction.
func (s *Session) FetchPage(ctx context.Context, id uint64) (Result, error) {
	if s.finishErr != nil {
		return Result{}, s.finishErr
	}
	if s.page == nil || id == 0 || s.page.id != id {
		return Result{}, errors.New("query cursor is no longer available; execute the SQL again")
	}
	stop := context.AfterFunc(ctx, s.page.cancel)
	defer stop()
	return s.readPage(ctx)
}

func (s *Session) readPage(ctx context.Context) (result Result, err error) {
	page := s.page
	result.Columns = append([]string(nil), page.columns...)
	defer func() {
		if err != nil {
			s.ClosePage()
		}
	}()
	stop, err := s.pageFetchContext(ctx)
	if err != nil {
		return result, err
	}
	defer stop()
	ctx = context.WithValue(ctx, fetchWatchKey{}, true)
	for {
		if err = ctx.Err(); err != nil {
			return result, err
		}
		cells := page.pending
		page.pending = nil
		if cells == nil {
			if !page.rows.Next() {
				err = ctx.Err()
				if err == nil {
					err = page.rows.Err()
				}
				s.ClosePage()
				return result, err
			}
			cells, err = s.scanCells(ctx, page.rows, page.types, true)
			if err != nil {
				return result, err
			}
		}
		cost := 64
		for _, cell := range cells {
			cost += len(cell.Text) + len(cell.Type) + 512 + min(len(cell.Text), 2048)*2
		}
		if page.count >= MaxGridRows || page.bytes+cost > MaxGridBytes {
			result.Truncated, result.PageLimited = true, true
			s.ClosePage()
			return result, nil
		}
		if len(result.Rows) == MaxRows {
			page.pending = cells
			result.Truncated, result.PageID = true, page.id
			return result, nil
		}
		row := make([]string, len(cells))
		for i, cell := range cells {
			row[i] = cell.Text
		}
		result.Rows = append(result.Rows, row)
		result.Values = append(result.Values, cells)
		page.count++
		page.bytes += cost
	}
}

// go-ora's QueryContext watcher ends when QueryContext returns; later driver
// Rows.Next network fetches need their own watcher. database/sql cancellation
// alone cannot interrupt a blocked Next while Rows.Close waits for its lock.
// Access the public driver API only inside Conn.Raw, once per page, never per
// row (important for fetch throughput). Other drivers/sqlmock use normal ctx.
func (s *Session) pageFetchContext(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return func() {}, err
	}
	type watcher interface {
		StartContext(context.Context) chan struct{}
		EndContext(chan struct{})
	}
	var done chan struct{}
	watchContext, cancelWatch := context.WithCancel(ctx)
	err := s.conn.Raw(func(raw any) error {
		if conn, ok := raw.(*go_ora.Connection); ok {
			if session, ok := conn.GetSession().(watcher); ok {
				done = session.StartContext(watchContext)
			}
		}
		return nil
	})
	if err != nil {
		cancelWatch()
		return func() {}, err
	}
	return func() {
		defer cancelWatch() // Also releases a watcher if Raw fails on a broken connection.
		if done == nil {
			return
		}
		_ = s.conn.Raw(func(raw any) error {
			if conn, ok := raw.(*go_ora.Connection); ok {
				if session, ok := conn.GetSession().(watcher); ok {
					session.EndContext(done)
				}
			}
			return nil
		})
	}, nil
}
