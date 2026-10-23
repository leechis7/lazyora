package oracle

import (
	"context"
	"errors"
)

// ProbeConnection uses a separate connection and closes its read-only transaction.
// No working session or transaction is accepted by this API.
func ProbeConnection(ctx context.Context, config Config) error {
	session, err := Open(ctx, config)
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Execute(ctx, "SELECT 1 FROM dual")
	if err != nil {
		return err
	}
	if len(result.Rows) != 1 || len(result.Rows[0]) != 1 || result.Rows[0][0] != "1" {
		return errors.New("unexpected connection test response")
	}
	return nil
}
