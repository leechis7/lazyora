package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	go_ora "github.com/sijms/go-ora/v3"
)

const (
	MaxOutputBytes  = 256 * 1024
	MaxOutputLines  = 2000
	outputBatch     = 128
	outputBufferSQL = "BEGIN DBMS_OUTPUT.DISABLE; DBMS_OUTPUT.ENABLE(1000000); END;"
)

// Output is separate from SQL success and transaction state. A collection error
// must never turn successful DML into a reported execution failure.
type Output struct {
	Lines     []string
	Truncated bool
	Error     string
}

func (s *Session) OutputEnabled() bool { return s.outputEnabled }

func (s *Session) outputExec(ctx context.Context, text string, args ...any) error {
	if s.finishErr != nil {
		return s.finishErr
	}
	var err error
	if s.tx != nil {
		_, err = s.tx.ExecContext(ctx, text, args...)
	} else {
		_, err = s.conn.ExecContext(ctx, text, args...)
	}
	return err
}

// SetOutput changes only package state on this pinned connection. It neither
// begins nor resolves a user transaction, and never adds pending SQL records.
func (s *Session) SetOutput(ctx context.Context, enabled bool) error {
	text := "BEGIN DBMS_OUTPUT.DISABLE; END;"
	if enabled {
		text = outputBufferSQL
	}
	if err := s.outputExec(ctx, text); err != nil {
		return err
	}
	s.outputEnabled, s.outputReset = enabled, false
	return nil
}

func (s *Session) resetOutput(ctx context.Context) error {
	if !s.outputReset || !s.outputEnabled {
		return nil
	}
	if err := s.outputExec(ctx, outputBufferSQL); err != nil {
		return fmt.Errorf("clear incomplete DBMS_OUTPUT before running more SQL: %w", err)
	}
	s.outputReset = false
	return nil
}

func (s *Session) collectOutput() (out Output) {
	// The SQL context may already be cancelled. Collection has its own short
	// deadline and cannot cancel/rollback the pinned user transaction.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	used := 0
	for {
		var lines []string
		count := outputBatch
		err := s.outputExec(ctx, "BEGIN DBMS_OUTPUT.GET_LINES(:1, :2); END;",
			go_ora.Out{Dest: &lines, Size: outputBatch}, sql.Out{Dest: &count, In: true})
		if err != nil {
			out.Error, s.outputReset = err.Error(), true
			return out
		}
		if count < 0 || count > outputBatch || count > len(lines) {
			out.Error, s.outputReset = "invalid DBMS_OUTPUT batch length", true
			return out
		}
		for _, line := range lines[:count] {
			if len(out.Lines) == MaxOutputLines || used+len(line)+1 > MaxOutputBytes {
				out.Truncated = true
				// Purge the unread remainder so it is never attributed to the next
				// SQL statement. No user DML or commit is involved.
				s.outputReset = true
				if err = s.resetOutput(ctx); err != nil {
					out.Error = err.Error()
				}
				return out
			}
			out.Lines = append(out.Lines, line)
			used += len(line) + 1
		}
		if count < outputBatch {
			return out
		}
	}
}
