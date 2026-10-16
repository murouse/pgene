package pgene_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/murouse/pgene"
	"github.com/murouse/pgene/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const itemsSchema = `
	DROP TABLE IF EXISTS tx_items;
	CREATE TABLE tx_items (name text PRIMARY KEY);`

var nestedModes = map[string]pgene.NestedMode{
	"join":      pgene.NestedJoin,
	"savepoint": pgene.NestedSavepoint,
}

func insert(ctx context.Context, c *pgene.Client, name string) error {
	_, err := c.Exec(ctx, pgene.Sql(`INSERT INTO tx_items (name) VALUES ($1)`, name))
	return err
}

func items(t *testing.T, c *pgene.Client) []string {
	t.Helper()

	var names []string
	require.NoError(t, c.Select(t.Context(), pgene.Sql(`SELECT name FROM tx_items ORDER BY name`), &names))
	return names
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestTxIsolatedPerClient(t *testing.T) {
	a := testdb.Connect(t, itemsSchema)
	b := testdb.Connect(t, "")

	var seenByB int
	err := pgene.InTx(t.Context(), a, func(ctx context.Context) error {
		if err := insert(ctx, a, "x"); err != nil {
			return err
		}
		return b.Get(ctx, pgene.Sql(`SELECT count(*) FROM tx_items`), &seenByB)
	})

	require.NoError(t, err)
	assert.Zero(t, seenByB, "client b must not run in the transaction of client a")
}

func TestNestedFailure(t *testing.T) {
	tests := []struct {
		name        string
		mode        pgene.NestedMode
		wantErrCode string
		wantItems   []string
	}{
		{name: "join aborts outer tx", mode: pgene.NestedJoin, wantErrCode: "25P02"}, // in_failed_sql_transaction
		{name: "savepoint keeps outer tx", mode: pgene.NestedSavepoint, wantItems: []string{"a", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testdb.Connect(t, itemsSchema)

			var nestedErr error
			err := pgene.InTx(t.Context(), c, func(ctx context.Context) error {
				if err := insert(ctx, c, "a"); err != nil {
					return err
				}
				nestedErr = pgene.InTx(ctx, c, func(ctx context.Context) error {
					if err := insert(ctx, c, "b"); err != nil {
						return err
					}
					return insert(ctx, c, "a")
				}, pgene.WithNestedMode(tt.mode))
				return insert(ctx, c, "c")
			})

			_, isDuplicate := pgene.IsDuplicateKeyError(nestedErr)
			assert.True(t, isDuplicate)
			assert.Equal(t, tt.wantErrCode, pgCode(err))
			assert.ElementsMatch(t, tt.wantItems, items(t, c))
		})
	}
}

func TestNestedRollsBackWithOuter(t *testing.T) {
	boom := errors.New("boom")

	for name, mode := range nestedModes {
		t.Run(name, func(t *testing.T) {
			c := testdb.Connect(t, itemsSchema)

			err := pgene.InTx(t.Context(), c, func(ctx context.Context) error {
				err := pgene.InTx(ctx, c, func(ctx context.Context) error {
					return insert(ctx, c, "a")
				}, pgene.WithNestedMode(mode))
				return errors.Join(err, boom)
			})

			assert.ErrorIs(t, err, boom)
			assert.Empty(t, items(t, c))
		})
	}
}

func TestHandlerErrorNotWrapped(t *testing.T) {
	c := testdb.Connect(t, "")
	boom := errors.New("boom")

	err := pgene.InTx(t.Context(), c, func(context.Context) error { return boom })

	assert.Equal(t, boom, err)
}

func TestWithoutTx(t *testing.T) {
	c := testdb.Connect(t, "")

	tests := map[string]func(context.Context) error{
		"commit":        c.CommitTx,
		"rollback":      c.RollbackTx,
		"advisory lock": func(ctx context.Context) error { return c.AdvisoryLock(ctx, "k") },
	}

	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, call(t.Context()), pgene.ErrNoTx)
		})
	}
}
