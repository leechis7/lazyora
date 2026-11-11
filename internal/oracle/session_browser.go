package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type SessionID struct {
	Instance, SID, Serial int
}

func (id SessionID) String() string {
	return fmt.Sprintf("%d:%d,%d", id.Instance, id.SID, id.Serial)
}

type SessionEntry struct {
	ID, BlockedBy                    SessionID
	User, Status, Kind, Event, SQLID string
	Machine, Program, Logon          string
	Waiters                          int
	Self, CanTerminate               bool
}

type SessionList struct {
	Entries   []SessionEntry
	Truncated bool
}

var (
	ErrSessionChanged   = errors.New("the confirmed session is no longer an eligible blocker")
	ErrSessionProtected = errors.New("the current or a protected session cannot be terminated")
)

const sessionBrowserLimit = 500

const sessionWaiterCounts = `
	LEFT JOIN (
		SELECT blocking_instance, blocking_session, COUNT(*) AS waiters
		FROM gv$session WHERE blocking_session_status = 'VALID'
		GROUP BY blocking_instance, blocking_session
	) w ON w.blocking_instance = s.inst_id AND w.blocking_session = s.sid`

const sessionLogon = `TO_CHAR(s.logon_time, 'YYYYMMDDHH24MISS', 'NLS_DATE_LANGUAGE=American')`

const sessionBrowserFields = `s.inst_id, s.sid, s.serial#, NVL(s.username, ' '), s.status, s.type,
	NVL(s.event, ' '), NVL(s.sql_id, ' '), NVL(s.machine, ' '), NVL(s.program, ' '),
	NVL(` + sessionLogon + `, ' '),
	CASE WHEN s.blocking_session_status = 'VALID' THEN NVL(s.blocking_instance, 0) ELSE 0 END,
	CASE WHEN s.blocking_session_status = 'VALID' THEN NVL(s.blocking_session, 0) ELSE 0 END,
	NVL(b.serial#, 0), NVL(w.waiters, 0),
	CASE WHEN s.sid = SYS_CONTEXT('USERENV', 'SID') AND s.inst_id = SYS_CONTEXT('USERENV', 'INSTANCE') THEN 1 ELSE 0 END`

const sessionBrowserFrom = ` FROM gv$session s ` + sessionWaiterCounts + `
	LEFT JOIN gv$session b ON s.blocking_session_status = 'VALID'
		AND b.inst_id = s.blocking_instance AND b.sid = s.blocking_session`

// Fail closed on databases without the account metadata needed to protect
// Oracle-maintained accounts. Those databases can still list sessions.
const sessionEligibility = `s.type = 'USER' AND s.username IS NOT NULL
	AND s.status NOT IN ('KILLED', 'SNIPED')
	AND s.con_id = SYS_CONTEXT('USERENV', 'CON_ID')
	AND EXISTS (SELECT 1 FROM all_users u
		WHERE u.user_id = s.user# AND u.username = s.username AND u.oracle_maintained = 'N')
	AND EXISTS (SELECT 1 FROM session_privs WHERE privilege = 'ALTER SYSTEM')`

func (s *Session) browserSessionID(ctx context.Context) (SessionID, error) {
	var id SessionID
	const query = `SELECT TO_NUMBER(SYS_CONTEXT('USERENV', 'INSTANCE')),
		TO_NUMBER(SYS_CONTEXT('USERENV', 'SID')) FROM dual`
	row := s.conn.QueryRowContext
	if s.tx != nil {
		row = s.tx.QueryRowContext
	}
	err := row(ctx, query).Scan(&id.Instance, &id.SID)
	return id, err
}

func sameSessionSlot(a, b SessionID) bool {
	return a.Instance == b.Instance && a.SID == b.SID
}

func (s *Session) ListSessions(ctx context.Context) (SessionList, error) {
	self, err := s.browserSessionID(ctx)
	if err != nil {
		return SessionList{}, err
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return SessionList{}, err
	}
	defer conn.Close()

	query := func(eligibility string) (*sql.Rows, error) {
		return conn.QueryContext(ctx, `SELECT * FROM (SELECT `+sessionBrowserFields+`, `+eligibility+
			sessionBrowserFrom+` ORDER BY CASE WHEN w.waiters > 0 THEN 0
				WHEN s.blocking_session_status = 'VALID' THEN 1
				WHEN s.type = 'USER' THEN 2 ELSE 3 END, s.inst_id, s.sid) WHERE ROWNUM <= :limit`, sessionBrowserLimit+2)
	}
	rows, err := query("CASE WHEN " + sessionEligibility + " THEN 1 ELSE 0 END")
	if err != nil && (strings.Contains(err.Error(), "ORA-00904:") || strings.Contains(err.Error(), "ORA-00942:") || strings.Contains(err.Error(), "ORA-01031:")) {
		rows, err = query("0")
	}
	if err != nil {
		return SessionList{}, err
	}
	defer rows.Close()
	var list SessionList
	for rows.Next() {
		var item SessionEntry
		var monitor, eligible int
		err := rows.Scan(&item.ID.Instance, &item.ID.SID, &item.ID.Serial, &item.User, &item.Status, &item.Kind,
			&item.Event, &item.SQLID, &item.Machine, &item.Program, &item.Logon, &item.BlockedBy.Instance,
			&item.BlockedBy.SID, &item.BlockedBy.Serial, &item.Waiters, &monitor, &eligible)
		if err != nil {
			return SessionList{}, err
		}
		if monitor != 0 {
			continue
		}
		item.Self = sameSessionSlot(self, item.ID)
		item.CanTerminate = eligible == 1 && !item.Self && item.Waiters > 0 && item.Logon != " "
		if len(list.Entries) == sessionBrowserLimit {
			list.Truncated = true
			break
		}
		list.Entries = append(list.Entries, item)
	}
	return list, rows.Err()
}

func (s *Session) TerminateBlockingSession(ctx context.Context, target SessionEntry) (marked bool, err error) {
	if target.ID.Instance <= 0 || target.ID.SID <= 0 || target.ID.Serial <= 0 ||
		strings.TrimSpace(target.User) == "" || strings.TrimSpace(target.Logon) == "" {
		return false, ErrSessionProtected
	}
	self, err := s.browserSessionID(ctx)
	if err != nil {
		return false, err
	}
	if sameSessionSlot(self, target.ID) {
		return false, ErrSessionProtected
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	const recheck = `SELECT TO_NUMBER(SYS_CONTEXT('USERENV', 'INSTANCE'))
		FROM gv$session s WHERE s.inst_id = :inst AND s.sid = :sid AND s.serial# = :serial
		AND s.username = :username AND ` + sessionLogon + ` = :logon
		AND ` + sessionEligibility + `
		AND NOT (s.sid = SYS_CONTEXT('USERENV', 'SID') AND s.inst_id = SYS_CONTEXT('USERENV', 'INSTANCE'))
		AND EXISTS (SELECT 1 FROM gv$session waiter
			WHERE waiter.blocking_session_status = 'VALID'
			AND waiter.blocking_instance = s.inst_id AND waiter.blocking_session = s.sid)`
	var commandInstance int
	err = conn.QueryRowContext(ctx, recheck, target.ID.Instance, target.ID.SID, target.ID.Serial, target.User, target.Logon).Scan(&commandInstance)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrSessionChanged
	}
	if err != nil {
		return false, err
	}

	// Oracle requires a quoted SID/serial literal. Only scanned integers
	// enter the command, and serial# guards against SID reuse after recheck.
	identity := fmt.Sprintf("%d,%d", target.ID.SID, target.ID.Serial)
	if target.ID.Instance != commandInstance {
		identity += fmt.Sprintf(",@%d", target.ID.Instance)
	}
	_, err = conn.ExecContext(ctx, "ALTER SYSTEM KILL SESSION '"+identity+"' IMMEDIATE")
	if err != nil && strings.Contains(err.Error(), "ORA-00031:") {
		return true, nil
	}
	return false, err
}
