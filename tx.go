package pgo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// txContextKey is used to store transaction in context.
type txContextKey struct{}

// nestedTxContextKey marks nested transaction (no-op).
type nestedTxContextKey struct{}

// txWrapper wraps pgx.Tx to allow interface abstraction.
type txWrapper struct {
	pgx.Tx
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

// BeginTx starts a new transaction or marks nested transaction.
// Stores transaction in context.
func (c *Client) BeginTx(ctx context.Context, opts ...TxOption) (context.Context, error) {
	// если транзакция уже есть
	if _, ok := extractTx(ctx); ok {
		// добавляем флаг вложенности
		return context.WithValue(ctx, nestedTxContextKey{}, struct{}{}), nil
	}

	txOpts := txOptions{}
	for _, opt := range opts {
		opt(&txOpts)
	}

	// иначе создаем транзакцию
	tx, err := c.pool.BeginTx(ctx, txOpts.TxOptions)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}

	return context.WithValue(ctx, txContextKey{}, &txWrapper{tx}), nil
}

// CommitTx commits transaction if not nested.
func (c *Client) CommitTx(ctx context.Context) error {
	// если нет транзакции, ошибка
	tx, ok := extractTx(ctx)
	if !ok {
		return fmt.Errorf("not a transaction")
	}

	// если вложенная транзакция, ничего не делаем
	if _, nested := ctx.Value(nestedTxContextKey{}).(struct{}); nested {
		return nil
	}

	// иначе коммитим
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// RollbackTx rolls back transaction if not nested.
func (c *Client) RollbackTx(ctx context.Context) error {
	// если нет транзакции, ошибка
	tx, ok := extractTx(ctx)
	if !ok {
		return fmt.Errorf("not a transaction")
	}

	// если вложенная транзакция, ничего не делаем
	if _, nested := ctx.Value(nestedTxContextKey{}).(struct{}); nested {
		return nil
	}

	// иначе роллбэчим
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("rollback: %w", err)
	}

	return nil
}

// InTx executes handler within transaction.
// Handles begin/commit/rollback automatically.
// Nested transactions are treated as no-op.
func InTx(ctx context.Context, tc TxController, handler func(context.Context) error, opts ...TxOption) (err error) {
	ctx, err = tc.BeginTx(ctx, opts...)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
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
		return fmt.Errorf("handler tx: %w", err)
	}

	if err = tc.CommitTx(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
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

// extractTx extracts transaction from context.
func extractTx(ctx context.Context) (*txWrapper, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*txWrapper)
	return tx, ok
}
