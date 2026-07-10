package pgene

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgconn"
)

// Sqlizer represents any query that can be converted to SQL + args.
type Sqlizer interface {
	ToSql() (string, []any, error)
}

// executor abstracts Exec method for pool and transaction.
type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Exec executes a query (INSERT/UPDATE/DELETE).
// Uses transaction if present in context.
func (c *Client) Exec(ctx context.Context, sql Sqlizer) (pgconn.CommandTag, error) {
	query, args, err := sql.ToSql()
	if err != nil {
		return pgconn.CommandTag{}, err
	}

	ex := executor(c.pool)
	if tx, ok := extractTx(ctx); ok {
		ex = tx
	}

	tag, err := ex.Exec(ctx, query, args...)
	if err != nil {
		return pgconn.CommandTag{}, err
	}

	return tag, nil
}

// Select executes a query and scans multiple rows into dest.
// Uses transaction if present in context.
func (c *Client) Select(ctx context.Context, sql Sqlizer, dest any) error {
	query, args, err := sql.ToSql()
	if err != nil {
		return err
	}

	qr := pgxscan.Querier(c.pool)
	if tx, ok := extractTx(ctx); ok {
		qr = tx
	}

	return pgxscan.Select(ctx, qr, dest, query, args...)
}

// Get executes a query and scans a single row into dest.
// Uses transaction if present in context.
func (c *Client) Get(ctx context.Context, sql Sqlizer, dest any) error {
	query, args, err := sql.ToSql()
	if err != nil {
		return err
	}

	qr := pgxscan.Querier(c.pool)
	if tx, ok := extractTx(ctx); ok {
		qr = tx
	}

	return pgxscan.Get(ctx, qr, dest, query, args...)
}
