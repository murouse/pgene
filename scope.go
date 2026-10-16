package pgene

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// txScope is a transaction stored in context. Besides queries, it collects after-commit hooks
// and decides how transactions started inside it are nested.
type txScope interface {
	pgx.Tx
	hookSink
	nest(ctx context.Context, mode NestedMode) (txScope, error)
}

// hookSink accepts after-commit hooks.
type hookSink interface {
	afterCommit(fn func())
}

// runNow runs hooks right away: outside a transaction and after a top-level commit.
type runNow struct{}

func (runNow) afterCommit(fn func()) { fn() }

// hookedTx is a top-level transaction or a savepoint that hands its hooks to parent on commit
// and drops them on rollback.
type hookedTx struct {
	pgx.Tx
	hooks  []func()
	parent hookSink // runNow for a top-level transaction, the outer transaction for a savepoint
}

func (t *hookedTx) afterCommit(fn func()) {
	t.hooks = append(t.hooks, fn)
}

func (t *hookedTx) Commit(ctx context.Context) error {
	if err := t.Tx.Commit(ctx); err != nil {
		return err
	}

	for _, fn := range t.hooks {
		t.parent.afterCommit(fn)
	}
	return nil
}

func (t *hookedTx) nest(ctx context.Context, mode NestedMode) (txScope, error) {
	return nestings[mode].nest(ctx, t)
}

// beginSavepoint starts a savepoint of outer that hands its hooks to parent on commit.
func beginSavepoint(ctx context.Context, outer pgx.Tx, parent hookSink) (txScope, error) {
	sp, err := outer.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &hookedTx{Tx: sp, parent: parent}, nil
}

// AfterCommit runs fn once the transaction in ctx is committed at the top level,
// e.g. to drop a cache or send an event. fn is dropped if the transaction, or the savepoint
// it was registered in, is rolled back. Without a transaction fn runs right away.
//
// Delivery is not guaranteed: if the process stops between commit and fn, fn is lost.
func (c *Client) AfterCommit(ctx context.Context, fn func()) {
	if tx, ok := c.tx(ctx); ok {
		tx.afterCommit(fn)
		return
	}
	fn()
}
