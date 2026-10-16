package pgene

import (
	"context"
)

// NestedMode defines how a transaction started inside another one of the same client behaves.
type NestedMode int

const (
	// NestedJoin runs the nested block in the outer transaction: its commit and rollback do nothing,
	// and a failed query aborts the whole outer transaction. This is the default.
	NestedJoin NestedMode = iota
	// NestedSavepoint runs the nested block in a SAVEPOINT: its rollback undoes only the block,
	// so the outer transaction stays usable, e.g. after a unique violation.
	NestedSavepoint
)

var nestings = [...]nesting{
	NestedJoin:      join{},
	NestedSavepoint: savepoint{},
}

// nesting opens a transaction inside the outer one according to NestedMode.
type nesting interface {
	nest(ctx context.Context, outer txScope) (txScope, error)
}

// join runs the nested block in the outer transaction.
type join struct{}

func (join) nest(_ context.Context, outer txScope) (txScope, error) {
	return joinedTx{outer}, nil
}

// savepoint runs the nested block in a SAVEPOINT of the outer transaction.
type savepoint struct{}

func (savepoint) nest(ctx context.Context, outer txScope) (txScope, error) {
	return beginSavepoint(ctx, outer, outer)
}

// joinedTx sends queries and hooks to the outer transaction but leaves commit and rollback to its owner.
type joinedTx struct {
	txScope
}

func (joinedTx) Commit(context.Context) error { return nil }

func (joinedTx) Rollback(context.Context) error { return nil }
