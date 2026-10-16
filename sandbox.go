package pgene

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// sandboxTx is a transaction meant to be rolled back. Transactions started in it behave
// as top-level ones: they run in savepoints and run their hooks on their own commit.
type sandboxTx struct {
	pgx.Tx
}

func (sandboxTx) afterCommit(fn func()) { fn() }

func (t sandboxTx) nest(ctx context.Context, _ NestedMode) (txScope, error) {
	return beginSavepoint(ctx, t, runNow{})
}

// BeginSandbox starts a transaction to be rolled back with RollbackTx, for tests and dry runs.
// Code running in it works as without it: its transactions commit (as savepoints) and run
// their after-commit hooks, yet rolling the sandbox back undoes everything.
func (c *Client) BeginSandbox(ctx context.Context) (context.Context, error) {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return ctx, fmt.Errorf("begin: %w", err)
	}

	return context.WithValue(ctx, txKey{c.pool}, sandboxTx{tx}), nil
}
