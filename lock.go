package pgene

import (
	"context"
)

// AdvisoryLock takes a transaction-level advisory lock on key, waiting until it is free.
// Use it to serialize work by a business key when there is no row to lock, e.g. "user:42".
// The lock is released when the transaction ends; ctx must hold a transaction of the client.
func (c *Client) AdvisoryLock(ctx context.Context, key string) error {
	tx, ok := c.tx(ctx)
	if !ok {
		return ErrNoTx
	}

	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", key)
	return err
}
