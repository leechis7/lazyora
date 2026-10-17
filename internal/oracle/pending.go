package oracle

import (
	"time"
	"unicode/utf8"
)

const MaxPendingStatements = 100
const MaxPendingSQLBytes = 64 * 1024

type StatementRecord struct {
	At        time.Time
	SQL       string
	Affected  int64
	Error     string
	Truncated bool
}

// PendingWork is an execution journal, not a database row-level change set.
// It is bounded in memory and discarded only after successful resolution.
type PendingWork struct {
	Statements []StatementRecord
	Omitted    int
}

func (s *Session) recordStatement(at time.Time, statement string, affected int64, err error) {
	record := StatementRecord{At: at, SQL: statement, Affected: affected}
	if len(record.SQL) > MaxPendingSQLBytes {
		record.SQL = record.SQL[:MaxPendingSQLBytes]
		for !utf8.ValidString(record.SQL) {
			record.SQL = record.SQL[:len(record.SQL)-1]
		}
		record.Truncated = true
	}
	if err != nil {
		record.Error = err.Error()
	}
	if len(s.work.Statements) == MaxPendingStatements {
		copy(s.work.Statements, s.work.Statements[1:])
		s.work.Statements = s.work.Statements[:MaxPendingStatements-1]
		s.work.Omitted++
	}
	s.work.Statements = append(s.work.Statements, record)
}

func (s *Session) PendingWork() PendingWork {
	return PendingWork{Statements: append([]StatementRecord(nil), s.work.Statements...), Omitted: s.work.Omitted}
}
