package pgene

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoTx is returned by CommitTx and RollbackTx when ctx holds no transaction of the client.
var ErrNoTx = errors.New("not a transaction")

// txKey stores the current transaction of one pool in context,
// so transactions of different clients never mix.
type txKey struct {
	pool *pgxpool.Pool
}

// TxController defines transaction lifecycle operations.
type TxController interface {
	BeginTx(ctx context.Context, opts ...TxOption) (context.Context, error)
	CommitTx(ctx context.Context) error
	RollbackTx(ctx context.Context) error
}

// TxOption configures transaction options.
type TxOption func(o *txOptions)

// txOptions wraps pgx.TxOptions.
type txOptions struct {
	pgx.TxOptions
	nestedMode NestedMode
}

// WithIsolationLevel sets transaction isolation level.
func WithIsolationLevel(isoLevel pgx.TxIsoLevel) TxOption {
	return func(o *txOptions) {
		o.IsoLevel = isoLevel
	}
}

// WithAccessMode sets transaction access mode.
func WithAccessMode(accessMode pgx.TxAccessMode) TxOption {
	return func(o *txOptions) {
		o.AccessMode = accessMode
	}
}

// WithDeferrableMode sets transaction deferrable mode.
func WithDeferrableMode(deferrableMode pgx.TxDeferrableMode) TxOption {
	return func(o *txOptions) {
		o.DeferrableMode = deferrableMode
	}
}

// WithNestedMode sets how the transaction behaves when started inside another one (NestedJoin by default).
// Has no effect on a top-level transaction.
func WithNestedMode(mode NestedMode) TxOption {
	return func(o *txOptions) {
		o.nestedMode = mode
	}
}

// BeginTx starts a new transaction, or a nested one if ctx already holds a transaction of this client.
// Stores transaction in context. Isolation, access and deferrable modes apply to top-level transactions only.
func (c *Client) BeginTx(ctx context.Context, opts ...TxOption) (context.Context, error) {
	txOpts := txOptions{}
	for _, opt := range opts {
		opt(&txOpts)
	}

	var (
		tx  txScope
		err error
	)
	if outer, ok := c.tx(ctx); ok {
		tx, err = outer.nest(ctx, txOpts.nestedMode)
	} else {
		tx, err = c.begin(ctx, txOpts.TxOptions)
	}
	if err != nil {
		return ctx, fmt.Errorf("begin: %w", err)
	}

	return context.WithValue(ctx, txKey{c.pool}, tx), nil
}

// CommitTx commits the transaction from ctx (does nothing for a joined nested one).
func (c *Client) CommitTx(ctx context.Context) error {
	tx, ok := c.tx(ctx)
	if !ok {
		return ErrNoTx
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// RollbackTx rolls back the transaction from ctx (does nothing for a joined nested one).
func (c *Client) RollbackTx(ctx context.Context) error {
	tx, ok := c.tx(ctx)
	if !ok {
		return ErrNoTx
	}

	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("rollback: %w", err)
	}

	return nil
}

// InTx executes handler within transaction.
// Handles begin/commit/rollback automatically; handler error is returned as is.
// Nested transactions follow WithNestedMode.
func InTx(ctx context.Context, tc TxController, handler func(context.Context) error, opts ...TxOption) (err error) {
	ctx, err = tc.BeginTx(ctx, opts...)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tc.RollbackTx(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				slog.ErrorContext(ctx, "rollback tx panic", "error", rbErr)
			}
			panic(p)
		}

		if err != nil {
			if rbErr := tc.RollbackTx(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				slog.ErrorContext(ctx, "rollback tx", "error", rbErr)
			}
		}
	}()

	if err = handler(ctx); err != nil {
		return err
	}

	return tc.CommitTx(ctx)
}

// GetInTx is a generic version of InTx that returns a value from handler.
// All transaction semantics (commit/rollback/panic handling) are identical to InTx.
func GetInTx[T any](ctx context.Context, tc TxController, handler func(context.Context) (T, error), opts ...TxOption) (T, error) {
	var res T
	err := InTx(ctx, tc, func(ctx context.Context) error {
		var innerErr error
		res, innerErr = handler(ctx)
		return innerErr
	}, opts...)
	return res, err
}

func GetInTx2[A, B any](ctx context.Context, tc TxController, handler func(context.Context) (A, B, error), opts ...TxOption) (A, B, error) {
	var (
		resA A
		resB B
	)
	err := InTx(ctx, tc, func(ctx context.Context) error {
		var innerErr error
		resA, resB, innerErr = handler(ctx)
		return innerErr
	}, opts...)
	return resA, resB, err
}

// begin starts a top-level transaction that runs its hooks right after commit.
func (c *Client) begin(ctx context.Context, opts pgx.TxOptions) (txScope, error) {
	tx, err := c.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &hookedTx{Tx: tx, parent: runNow{}}, nil
}

// tx extracts the client's transaction from context.
func (c *Client) tx(ctx context.Context) (txScope, bool) {
	tx, ok := ctx.Value(txKey{c.pool}).(txScope)
	return tx, ok
}
