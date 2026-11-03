package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/leechis7/tuivory/go"
)

func TestConnectionProbeUsesCurrentFormWithoutSavingOrReplacingSession(t *testing.T) {
	s, _ := resultScreen(t)
	s.connectionForm()
	for i, value := range map[int]string{connectionHost: "example", connectionPort: "1522", connectionUser: "SCOTT", connectionPassword: "secret&pwd", connectionName: "ORCL"} {
		s.connectForm.GetFormItem(i).(*tuivory.InputField).SetText(value)
	}
	s.pendingChanges = true
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'v', tuivory.ModAlt))
	j := <-s.jobs
	if j.action != "test_connect" || j.config.Host != "example" || j.config.Password != "secret&pwd" || !s.testingConnection || !s.pendingChanges {
		t.Fatal("test did not use isolated job/current fields")
	}
	deadline, ok := j.ctx.Deadline()
	if !ok || time.Until(deadline) > 16*time.Second {
		t.Fatal("test timeout missing")
	}
	s.testConnection()
	if len(s.jobs) != 0 {
		t.Fatal("duplicate test queued")
	}
	s.closeConnection()
	if !s.pages.HasPage("connect") || j.ctx.Err() != context.Canceled {
		t.Fatal("cancel closed form or failed to cancel")
	}
	s.busy = false
	s.cancel = nil
	s.acceptConnectionProbe(errors.New("failed secret&pwd and secret%26pwd"), j.config)
	if strings.Contains(s.connectForm.GetTitle(), "secret") || !s.pendingChanges || s.resultRows[0][0] != "1" {
		t.Fatal("error leaked secret or changed working data")
	}
	s.closeConnection()
}
