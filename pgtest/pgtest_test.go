package pgtest_test

import (
	"context"
	"testing"

	"github.com/murouse/pgene"
	"github.com/murouse/pgene/internal/testdb"
	"github.com/murouse/pgene/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const schema = `
	DROP TABLE IF EXISTS pgtest_items;
	CREATE TABLE pgtest_items (name text PRIMARY KEY);`

func insert(ctx context.Context, c *pgene.Client, name string) error {
	_, err := c.Exec(ctx, pgene.Sql(`INSERT INTO pgtest_items (name) VALUES ($1)`, name))
	return err
}

func names(ctx context.Context, t *testing.T, c *pgene.Client) []string {
	t.Helper()

	var names []string
	require.NoError(t, c.Select(ctx, pgene.Sql(`SELECT name FROM pgtest_items ORDER BY name`), &names))
	return names
}

func TestTx(t *testing.T) {
	c := testdb.Connect(t, schema)

	t.Run("sandbox", func(t *testing.T) {
		ctx := pgtest.Tx(t, c)
		calls := 0

		// вне транзакций кода хук срабатывает сразу, как без песочницы
		c.AfterCommit(ctx, func() { calls++ })
		assert.Equal(t, 1, calls)

		// транзакция кода под тестом коммитится и запускает хуки
		err := pgene.InTx(ctx, c, func(ctx context.Context) error {
			c.AfterCommit(ctx, func() { calls++ })
			return insert(ctx, c, "a")
		})
		require.NoError(t, err)
		assert.Equal(t, 2, calls)

		// её ошибка не ломает песочницу
		err = pgene.InTx(ctx, c, func(ctx context.Context) error { return insert(ctx, c, "a") })
		_, isDuplicate := pgene.IsDuplicateKeyError(err)
		assert.True(t, isDuplicate)
		assert.Equal(t, []string{"a"}, names(ctx, t, c))
	})

	assert.Empty(t, names(t.Context(), t, c), "sandbox must be rolled back")
}
