package pgene

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Sqlizer represents any query that can be converted to SQL + args.
type Sqlizer interface {
	ToSql() (string, []any, error)
}

// querier abstracts query methods shared by pool and transaction.
type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Exec executes a query (INSERT/UPDATE/DELETE).
// Uses transaction if present in context.
func (c *Client) Exec(ctx context.Context, sql Sqlizer) (pgconn.CommandTag, error) {
	query, args, err := sql.ToSql()
	if err != nil {
		return pgconn.CommandTag{}, err
	}

	return c.querier(ctx).Exec(ctx, query, args...)
}

// Select executes a query and scans multiple rows into dest.
// Uses transaction if present in context.
func (c *Client) Select(ctx context.Context, sql Sqlizer, dest any) error {
	query, args, err := sql.ToSql()
	if err != nil {
		return err
	}

	return pgxscan.Select(ctx, c.querier(ctx), dest, query, args...)
}

// Get executes a query and scans a single row into dest.
// Uses transaction if present in context.
func (c *Client) Get(ctx context.Context, sql Sqlizer, dest any) error {
	query, args, err := sql.ToSql()
	if err != nil {
		return err
	}

	return pgxscan.Get(ctx, c.querier(ctx), dest, query, args...)
}

// querier returns the client's transaction from context, or the pool if there is none.
func (c *Client) querier(ctx context.Context) querier {
	if tx, ok := c.tx(ctx); ok {
		return tx
	}
	return c.pool
}
