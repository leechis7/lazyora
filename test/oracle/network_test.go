package oracle_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

// Local Free only: its test account already has DBA privileges. Disconnect
// only the dedicated session opened by this test, never unrelated sessions.
func TestOracleNetworkIntegration(t *testing.T) {
	for _, commit := range []bool{true, false} {
		t.Run(fmt.Sprintf("commit=%v", commit), func(t *testing.T) {
			observer, s := workflowSession(t), workflowSession(t)
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			identity, err := s.Execute(ctx, "SELECT sid, serial# FROM v$session WHERE sid=TO_NUMBER(SYS_CONTEXT('USERENV','SID')) AND username=USER")
			if err != nil || len(identity.Rows) != 1 {
				t.Fatal("identify own test session:", err)
			}
			ids := make([]int, 2)
			for i := range ids {
				ids[i], err = strconv.Atoi(identity.Rows[0][i])
				if err != nil || ids[i] <= 0 {
					t.Fatal("invalid test session identity")
				}
			}
			table := fmt.Sprintf("LORA_NET_%x", time.Now().UnixNano())
			if _, err = observer.Execute(ctx, "CREATE TABLE "+table+" (n NUMBER)"); err != nil {
				t.Fatal(err)
			}
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
				defer stop()
				if _, err := observer.Execute(cleanup, "DROP TABLE "+table+" PURGE"); err != nil {
					t.Error(err)
				}
			}()
			if _, err = s.Execute(ctx, "INSERT INTO "+table+" VALUES (1)"); err != nil {
				t.Fatal(err)
			}
			if _, err = observer.Execute(ctx, fmt.Sprintf("ALTER SYSTEM DISCONNECT SESSION '%d,%d' IMMEDIATE", ids[0], ids[1])); err != nil {
				t.Fatal(err)
			}
			started := time.Now()
			if err = s.Finish(commit); err == nil || !s.TransactionUncertain() || !s.PendingChanges() {
				t.Fatalf("disconnected session accepted resolution: %v", err)
			}
			if time.Since(started) > 5*time.Second {
				t.Fatal("disconnected session resolution exceeded 5 seconds")
			}
			if err = s.Finish(commit); err == nil {
				t.Fatal("uncertain resolution accepted retry")
			}
			if _, err = s.Execute(ctx, "SELECT 1 FROM dual"); err == nil {
				t.Fatal("uncertain connection accepted SQL")
			}
			observed, err := observer.Execute(ctx, "SELECT COUNT(*) FROM "+table)
			if err != nil || len(observed.Rows) != 1 || observed.Rows[0][0] != "0" {
				t.Fatal("disconnect committed pending DML", err)
			}
			recovered := workflowSession(t)
			if _, err = recovered.Execute(ctx, "SELECT 1 FROM dual"); err != nil {
				t.Fatal(err)
			}
			t.Logf("server disconnect before %s: uncertain state retained; independent reconnect passed", map[bool]string{true: "COMMIT", false: "ROLLBACK"}[commit])
		})
	}
}
