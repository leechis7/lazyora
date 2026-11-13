package oracle_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleDebuggerIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	exec := func(sql string) {
		t.Helper()
		if _, err := s.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE TABLE lazyora_dbg_rows (n NUMBER)")
	exec("CREATE OR REPLACE PROCEDURE lazyora_dbg_child AS\nBEGIN\n NULL;\n NULL;\nEND;")
	exec("ALTER PROCEDURE lazyora_dbg_child COMPILE DEBUG")
	exec("CREATE OR REPLACE PROCEDURE lazyora_dbg_probe AS\n n NUMBER := 41;\nBEGIN\n n := n + 1;\n n := n + 2;\n lazyora_dbg_child;\n INSERT INTO lazyora_dbg_rows VALUES (n);\nEND;")
	exec("ALTER PROCEDURE lazyora_dbg_probe COMPILE DEBUG")
	t.Cleanup(func() {
		_ = s.Finish(false)
		for _, sql := range []string{"DROP PROCEDURE lazyora_dbg_probe", "DROP PROCEDURE lazyora_dbg_child", "DROP TABLE lazyora_dbg_rows PURGE"} {
			if _, err := s.Execute(context.Background(), sql); err != nil {
				t.Error(err)
			}
		}
	})
	exec("INSERT INTO lazyora_dbg_rows VALUES (99)")
	actions := make(chan oracle.DebugAction, 1)
	type update struct {
		state oracle.DebugState
		err   error
	}
	updates := make(chan update, 1)
	done := make(chan error, 1)
	start := func(runCtx context.Context) {
		go func() {
			done <- s.DebugProcedure(runCtx, "lazyora_dbg_probe", 4, actions, func(state oracle.DebugState, err error) {
				select {
				case updates <- update{state, err}:
				case <-runCtx.Done():
				}
			})
		}()
	}
	receive := func() oracle.DebugState {
		t.Helper()
		select {
		case u := <-updates:
			if u.err != nil {
				t.Fatal(u.err)
			}
			return u.state
		case err := <-done:
			t.Fatalf("debugger exited: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		return oracle.DebugState{}
	}
	start(ctx)
	if state := receive(); state.Line != 4 || state.Unit != "LAZYORA_DBG_PROBE" || !strings.Contains(state.Source, "n := n + 1") || !strings.Contains(state.Stack, "LAZYORA_DBG_PROBE") {
		t.Fatalf("breakpoint or stack incorrect: %+v", state)
	}
	actions <- oracle.DebugAction{Kind: "watch", Watch: "N"}
	if value := receive().Value; value != "41" {
		t.Fatalf("watch = %q", value)
	}
	actions <- oracle.DebugAction{Kind: "over"}
	if line := receive().Line; line != 5 {
		t.Fatalf("step stopped at %d", line)
	}
	actions <- oracle.DebugAction{Kind: "watch", Watch: "N"}
	if value := receive().Value; value != "42" {
		t.Fatalf("watch after step = %q", value)
	}
	actions <- oracle.DebugAction{Kind: "over"}
	if line := receive().Line; line != 6 {
		t.Fatalf("expected the child call on line 6, got %d", line)
	}
	actions <- oracle.DebugAction{Kind: "into"}
	if state := receive(); state.Unit != "LAZYORA_DBG_CHILD" || !strings.Contains(state.Source, "lazyora_dbg_child") || !strings.Contains(state.Stack, "LAZYORA_DBG_PROBE") {
		t.Fatalf("did not step into child: %+v", state)
	}
	actions <- oracle.DebugAction{Kind: "breakpoint", Line: 4}
	receive()
	actions <- oracle.DebugAction{Kind: "continue"}
	if state := receive(); state.Unit != "LAZYORA_DBG_CHILD" || state.Line != 4 {
		t.Fatalf("new breakpoint was not applied to the displayed child: %+v", state)
	}
	actions <- oracle.DebugAction{Kind: "out"}
	if state := receive(); state.Unit != "LAZYORA_DBG_PROBE" {
		t.Fatalf("did not step out of child: %+v", state)
	}
	actions <- oracle.DebugAction{Kind: "continue"}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	start(ctx)
	receive()
	actions <- oracle.DebugAction{Kind: "stop"}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("stop: %v", err)
	}
	stopCtx, stop := context.WithCancel(ctx)
	defer stop()
	start(stopCtx)
	receive()
	stop()
	if err := <-done; err != context.Canceled {
		t.Fatalf("cancelled pause did not abort and roll back cleanly: %v", err)
	}

	if _, err := s.Execute(ctx, "SELECT 1 FROM dual"); err != nil {
		t.Fatalf("connection unusable after two debug runs: %v", err)
	}
	if !s.PendingChanges() {
		t.Fatal("debugger cleared editor transaction")
	}
	result, err := s.Execute(ctx, "SELECT n FROM lazyora_dbg_rows ORDER BY n")
	if err != nil || len(result.Rows) != 1 || result.Rows[0][0] != "99" {
		t.Fatalf("debugger changed transaction contents: %v, %v", result.Rows, err)
	}
}
