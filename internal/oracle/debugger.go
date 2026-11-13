package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v3"
)

type DebugAction struct {
	Kind  string
	Line  int
	Watch string
}

type DebugState struct {
	Line, Depth  int
	Owner, Unit  string
	Stack, Value string
	Null         bool
	Source       string
	Watch        string
}

type debugProbe struct {
	conn *sql.Conn
}

func (p debugProbe) move(ctx context.Context, action string) (DebugState, bool, error) {
	call := "dbms_debug.synchronize(r, info)"
	flag, ok := map[string]string{"start": "", "continue": "0", "over": "dbms_debug.break_next_line", "into": "dbms_debug.break_any_call", "out": "dbms_debug.break_any_return", "stop": "dbms_debug.abort_execution"}[action]
	if !ok {
		return DebugState{}, false, errors.New("unknown debugger command")
	}
	if flag != "" {
		call = "dbms_debug.continue(r, " + flag + ", info)"
	}
	block := `DECLARE r dbms_debug.runtime_info;
	info PLS_INTEGER := dbms_debug.info_getStackDepth + dbms_debug.info_getLineInfo;
	rc PLS_INTEGER;
	BEGIN rc := ` + call + `;
	:rc := rc; :finished := NVL(r.terminated, 0);
	:ln := NVL(r.line#, 0); :depth := NVL(r.stackdepth, 0);
	:owner := r.program.owner; :unit := r.program.name;
	END;`
	var code, finished int
	var state DebugState
	_, err := p.conn.ExecContext(ctx, block,
		sql.Named("rc", sql.Out{Dest: &code}), sql.Named("finished", sql.Out{Dest: &finished}),
		sql.Named("ln", sql.Out{Dest: &state.Line}), sql.Named("depth", sql.Out{Dest: &state.Depth}),
		sql.Named("owner", go_ora.Out{Dest: &state.Owner, Size: 128}), sql.Named("unit", go_ora.Out{Dest: &state.Unit, Size: 128}))
	if err == nil && code != 0 {
		err = fmt.Errorf("Oracle debugger %s returned %d", action, code)
	}
	return state, finished != 0, err
}

func (p debugProbe) breakpoint(ctx context.Context, object Object, line int) error {
	if line < 1 || line > 1000000 {
		return errors.New("enter a positive Oracle source line")
	}
	var code int
	_, err := p.conn.ExecContext(ctx, `DECLARE u dbms_debug.program_info; n PLS_INTEGER;
	BEGIN u.namespace := dbms_debug.namespace_pkgspec_or_toplevel;
	u.owner := :owner; u.name := :unit;
	:rc := dbms_debug.set_breakpoint(u, :ln, n);
	END;`, sql.Named("owner", object.Owner), sql.Named("unit", object.Name),
		sql.Named("rc", sql.Out{Dest: &code}), sql.Named("ln", line))
	if err == nil && code != 0 {
		err = fmt.Errorf("Oracle rejected breakpoint at line %d (code %d)", line, code)
	}
	return err
}

func (p debugProbe) inspect(ctx context.Context, state DebugState, watch string) (DebugState, error) {
	_, err := p.conn.ExecContext(ctx, `DECLARE frames dbms_debug.backtrace_table; b VARCHAR2(32767); i PLS_INTEGER;
	BEGIN
	dbms_debug.print_backtrace(frames);
	i := frames.LAST;
	WHILE i IS NOT NULL AND LENGTH(NVL(b,' ')) < 28000 LOOP
	  b := b || i || '  ' || CASE WHEN frames(i).name IS NULL THEN '<anonymous>' ELSE frames(i).owner || '.' || frames(i).name END ||
	       '  : ' || frames(i).line# || CHR(10);
	  i := frames.PRIOR(i);
	END LOOP;
	:stack := b; END;`,
		sql.Named("stack", go_ora.Out{Dest: &state.Stack, Size: 32767}))
	if err != nil || watch == "" {
		return state, err
	}
	var code, null int
	_, err = p.conn.ExecContext(ctx, `DECLARE v VARCHAR2(32767); c PLS_INTEGER; BEGIN
	c := dbms_debug.get_value(:name, 0, v);
	:rc := c; :val := v;
	IF c = dbms_debug.error_nullvalue THEN :isnull := 1; ELSE :isnull := 0; END IF;
	END;`, sql.Named("name", watch), sql.Named("rc", sql.Out{Dest: &code}),
		sql.Named("val", go_ora.Out{Dest: &state.Value, Size: 32767}), sql.Named("isnull", sql.Out{Dest: &null}))
	state.Watch = watch
	state.Null = null != 0
	if err == nil && code != 0 && !state.Null {
		err = fmt.Errorf("Oracle could not read %s (code %d)", watch, code)
	}
	return state, err
}

// DebugProcedure runs on two temporary sessions from this workspace's pool.
// The editor's pinned connection and transaction are never used for debugging.
func (s *Session) DebugProcedure(ctx context.Context, reference string, line int, actions <-chan DebugAction, report func(DebugState, error)) (err error) {
	if s.finishErr != nil {
		return s.finishErr
	}
	if line < 1 || line > 1000000 {
		return errors.New("enter a positive Oracle source line")
	}
	owner, name, err := ParseObjectReference(reference)
	if err != nil {
		return err
	}
	open, stopOpen := context.WithTimeout(ctx, 15*time.Second)
	defer stopOpen()
	target, err := s.db.Conn(open)
	if err != nil {
		return err
	}
	defer target.Close()
	control, err := s.db.Conn(open)
	if err != nil {
		return err
	}
	defer control.Close()
	var obj Object
	var debug string
	var args int
	err = target.QueryRowContext(open, `SELECT o.owner, o.object_name, o.object_type, p.plsql_debug,
	(SELECT COUNT(*) FROM all_arguments a WHERE a.owner=o.owner AND a.object_name=o.object_name AND a.package_name IS NULL)
	FROM all_objects o JOIN all_plsql_object_settings p ON p.owner=o.owner AND p.name=o.object_name AND p.type=o.object_type
	WHERE o.owner=SYS_CONTEXT('USERENV','SESSION_USER') AND o.owner=NVL(:1,o.owner)
	AND o.object_name=:2 AND o.object_type='PROCEDURE' AND o.status='VALID'`, owner, name).
		Scan(&obj.Owner, &obj.Name, &obj.Kind, &debug, &args)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (!strings.EqualFold(debug, "TRUE") || args != 0) {
		return errors.New("choose a valid standalone procedure without parameters in your login schema, compiled for debugging")
	}
	if err != nil {
		return err
	}
	source, err := readDebugSource(open, target, obj.Owner, obj.Name)
	if err != nil {
		return err
	}
	if source == "" {
		return errors.New("debug source is unavailable")
	}
	var instance, controlInstance int

	const instanceSQL = `SELECT TO_NUMBER(SYS_CONTEXT('USERENV','INSTANCE')) FROM dual`
	if err = target.QueryRowContext(open, instanceSQL).Scan(&instance); err != nil {
		return err
	}
	if err = control.QueryRowContext(open, instanceSQL).Scan(&controlInstance); err != nil {
		return err
	}
	if instance != controlInstance {
		return errors.New("debug connections must use the same Oracle instance")
	}
	tx, err := target.BeginTx(context.WithoutCancel(ctx), nil)
	if err != nil {
		return err
	}
	defer func() {
		if rollback := tx.Rollback(); rollback != nil {
			err = errors.Join(err, fmt.Errorf("could not verify debug transaction rollback: %w", rollback))
		}
	}()
	var id string
	_, err = tx.ExecContext(open, `DECLARE t PLS_INTEGER; BEGIN
	:id := dbms_debug.initialize;
	t := dbms_debug.set_timeout(90);
	dbms_debug.set_timeout_behaviour(dbms_debug.abort_on_timeout);
	dbms_debug.debug_on;
	END;`, sql.Named("id", go_ora.Out{Dest: &id, Size: 32767}))
	if err != nil {
		return fmt.Errorf("initialize Oracle debugger: %w", err)
	}
	_, err = control.ExecContext(open, `DECLARE t PLS_INTEGER; BEGIN
	t := dbms_debug.set_timeout(15); dbms_debug.attach_session(:id);
	END;`, id)
	if err != nil {
		return fmt.Errorf("attach Oracle debugger: %w", err)
	}

	run, stopRun := context.WithCancel(context.WithoutCancel(ctx))
	done := make(chan error, 1)
	go func() { _, e := tx.ExecContext(run, "BEGIN "+QualifiedObject(obj)+"; END;"); done <- e }()
	joined := false
	defer func() {
		stopRun()
		if !joined {
			<-done
		}
	}()
	probe := debugProbe{control}
	state, finished, err := probe.move(ctx, "start")
	if err != nil {
		return err
	}
	if finished {
		return errors.New("target terminated before the breakpoint was installed")
	}
	if err = probe.breakpoint(ctx, obj, line); err != nil {
		_, _, stopErr := probe.move(ctx, "stop")
		return errors.Join(err, stopErr)
	}
	for action := (DebugAction{Kind: "continue"}); ; {
		state, finished, err = probe.move(ctx, action.Kind)
		if err != nil {
			return err
		}
		if finished || action.Kind == "stop" {
			select {
			case e := <-done:
				joined = true
				if action.Kind == "stop" {
					return context.Canceled
				}
				return e
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if state.Owner == obj.Owner && state.Unit == obj.Name {
			state.Source = source
		} else if state.Owner == obj.Owner && state.Unit != "" {
			state.Source, _ = readDebugSource(ctx, control, state.Owner, state.Unit)
		}
		state, err = probe.inspect(ctx, state, "")
		report(state, err)
		timer := time.NewTimer(75 * time.Second)
	wait:
		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, _, stopErr := probe.move(cleanup, "stop")
				cancel()
				if stopErr != nil {
					return errors.Join(ctx.Err(), stopErr)
				}
				return ctx.Err()
			case <-timer.C:
				_, _, stopErr := probe.move(ctx, "stop")
				return errors.Join(errors.New("debugger pause timed out"), stopErr)
			case next, ok := <-actions:
				if !ok {
					timer.Stop()
					_, _, stopErr := probe.move(ctx, "stop")
					if stopErr != nil {
						return errors.Join(context.Canceled, stopErr)
					}
					return context.Canceled
				}
				action = next
				if action.Kind == "watch" {
					if owner, name, e := ParseObjectReference(action.Watch); e != nil || owner != "" || name == "" {
						report(state, errors.New("enter a scalar variable name"))
					} else {
						watch := name
						if strings.HasPrefix(strings.TrimSpace(action.Watch), `"`) {
							watch = strings.TrimSpace(action.Watch)
						}
						value, e := probe.inspect(ctx, state, watch)
						report(value, e)
					}
					continue
				}
				if action.Kind == "breakpoint" {
					if state.Owner != obj.Owner || state.Unit == "" || state.Source == "" {
						report(state, errors.New("breakpoints can only be added to a displayed standalone procedure in your schema"))
					} else {
						current := Object{Owner: state.Owner, Name: state.Unit, Kind: "PROCEDURE"}
						report(state, probe.breakpoint(ctx, current, action.Line))
					}
					continue
				}
				break wait
			}
		}
		timer.Stop()
	}
}

// Limit source to standalone procedures in the debugged user's schema. The
// Oracle line numbers are preserved, without adding CREATE OR REPLACE.
func readDebugSource(ctx context.Context, conn *sql.Conn, owner, unit string) (string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT text FROM all_source
	WHERE owner = :1 AND name = :2 AND type = 'PROCEDURE' ORDER BY line`, owner, unit)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var source strings.Builder
	for rows.Next() {
		var text sql.NullString
		if err = rows.Scan(&text); err != nil {
			return "", err
		}
		if source.Len()+len(text.String)+1 > 1<<20 {
			return "", errors.New("debug source exceeds 1 MiB")
		}
		source.WriteString(strings.TrimRight(text.String, "\r\n"))
		source.WriteByte('\n')
	}
	return source.String(), rows.Err()
}
