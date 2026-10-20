package oracle_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleReconnectIntegration(t *testing.T) {
	for _, choice := range []oracle.ReconnectResolution{oracle.ReconnectCommit, oracle.ReconnectRollback} {
		t.Run(string(choice), func(t *testing.T) {
			s := workflowSession(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			portText := os.Getenv("ORACLE_PORT")
			if portText == "" {
				portText = "1521"
			}
			port, err := oracle.ParsePort(portText)
			if err != nil {
				t.Fatal(err)
			}
			config := oracle.Config{Host: "127.0.0.1", Port: port, User: "lazyora", Password: os.Getenv("APP_USER_PASSWORD"), Mode: "service", Name: "FREEPDB1"}
			table := fmt.Sprintf("LORA_RECON_%d", time.Now().UnixNano()%1000000000000)
			if _, err = s.Execute(ctx, "CREATE TABLE "+table+" (n NUMBER)"); err != nil {
				t.Fatal(err)
			}
			// A separate cleanup session still works if the original was closed.
			defer func() {
				s.Close()
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cleanupCancel()
				cleanup, openErr := oracle.Open(cleanupCtx, config)
				if openErr != nil {
					t.Error("could not open fixture cleanup session")
					return
				}
				defer cleanup.Close()
				if _, dropErr := cleanup.Execute(cleanupCtx, "DROP TABLE "+table+" PURGE"); dropErr != nil {
					t.Error(dropErr)
				}
			}()
			if _, err = s.Execute(ctx, "INSERT INTO "+table+" VALUES (1)"); err != nil {
				t.Fatal(err)
			}
			if err = s.Finish(true); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Execute(ctx, "UPDATE "+table+" SET n=2"); err != nil {
				t.Fatal(err)
			}
			if got, reconnectErr := oracle.Reconnect(ctx, s, config, ""); reconnectErr == nil || got != s || !s.PendingChanges() {
				t.Fatal("unconfirmed reconnect discarded pending work")
			}
			bad := config
			bad.Password = "invalid-test-password"
			if got, reconnectErr := oracle.Reconnect(ctx, s, bad, choice); reconnectErr == nil || got != s || !s.PendingChanges() {
				t.Fatal("failed target changed existing session")
			}
			before, err := s.Execute(ctx, "SELECT n FROM "+table)
			if err != nil || len(before.Rows) != 1 || before.Rows[0][0] != "2" {
				t.Fatalf("failed target lost uncommitted update: %v", err)
			}
			next, err := oracle.Reconnect(ctx, s, config, choice)
			if err != nil {
				t.Fatal(err)
			}
			s = next
			after, err := s.Execute(ctx, "SELECT n FROM "+table)
			want := "1"
			if choice == oracle.ReconnectCommit {
				want = "2"
			}
			if err != nil || len(after.Rows) != 1 || after.Rows[0][0] != want || s.PendingChanges() {
				t.Fatalf("reconnect did not honor %s: %v", choice, err)
			}
		})
	}
}
