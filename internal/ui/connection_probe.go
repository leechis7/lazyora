package ui

import (
	"context"
	"errors"
	"fmt"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"net/url"
	"strings"
)

func safeProbeError(err error, password string) string {
	if errors.Is(err, context.Canceled) {
		return "cancel"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	text := err.Error()
	if password != "" {
		for _, secret := range []string{password, url.QueryEscape(password), url.PathEscape(password)} {
			text = strings.ReplaceAll(text, secret, "***")
		}
	}
	runes := []rune(text)
	if len(runes) > 400 {
		text = string(runes[:400]) + "…"
	}
	return tview.Escape(text)
}
func (s *screen) acceptConnectionProbe(err error, config oracle.Config) {
	s.testingConnection = false
	if s.connectForm == nil {
		return
	}
	text := s.tr("connection_probe_ok")
	if err != nil {
		text = safeProbeError(err, config.Password)
		switch text {
		case "cancel":
			text = s.tr("connection_probe_cancel")
		case "timeout":
			text = s.tr("connection_probe_timeout")
		default:
			text = fmt.Sprintf(s.tr("connection_probe_error"), text)
		}
	}
	s.connectForm.SetTitle(text)
	s.status.SetText(text)
}
