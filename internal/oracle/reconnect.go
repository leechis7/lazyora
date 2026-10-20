package oracle

import (
	"context"
	"errors"
	"fmt"
)

type ReconnectResolution string

const (
	ReconnectCommit   ReconnectResolution = "commit"
	ReconnectRollback ReconnectResolution = "rollback"
	// ReconnectDiscardUncertain requires a separate, explicit warning in the UI.
	// It abandons a session whose previous commit/rollback outcome is unknown.
	ReconnectDiscardUncertain ReconnectResolution = "discard_uncertain"
)

// Reconnect replaces a session only after the target has opened and any pending
// transaction has been explicitly resolved. On error it returns current, which
// remains owned by the caller. Like Session, it must run on the single worker.
func Reconnect(ctx context.Context, current *Session, config Config, resolution ReconnectResolution) (*Session, error) {
	return reconnect(ctx, current, resolution, func() (*Session, error) { return Open(ctx, config) })
}

func reconnect(ctx context.Context, current *Session, resolution ReconnectResolution, open func() (*Session, error)) (*Session, error) {
	if resolution != "" && resolution != ReconnectCommit && resolution != ReconnectRollback && resolution != ReconnectDiscardUncertain {
		return current, errors.New("invalid reconnect transaction choice")
	}
	if resolution == ReconnectDiscardUncertain && (current == nil || !current.TransactionUncertain()) {
		return current, errors.New("only an uncertain transaction may be explicitly abandoned")
	}
	if current != nil && current.PendingChanges() && resolution == "" {
		return current, errors.New("uncommitted changes: choose commit or rollback before reconnecting")
	}
	if err := ctx.Err(); err != nil {
		return current, err
	}
	next, err := open()
	if err != nil {
		return current, err
	}
	// Opening can race cancellation. Do not resolve the old transaction if the
	// user cancelled while the target connection was being established.
	if err = ctx.Err(); err != nil {
		next.Close()
		return current, err
	}
	if current != nil {
		if current.PendingChanges() && resolution != ReconnectDiscardUncertain {
			if err = current.Finish(resolution == ReconnectCommit); err != nil {
				next.Close()
				return current, fmt.Errorf("could not finish the current transaction; connection was not switched: %w", err)
			}
		}
		current.Close()
	}
	return next, nil
}
