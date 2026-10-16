// Package testdb connects integration tests to PostgreSQL from PGENE_TEST_DSN.
package testdb

import (
	"os"
	"testing"

	"github.com/murouse/pgene"
	"github.com/stretchr/testify/require"
)

// Connect connects to PGENE_TEST_DSN and runs schema; skips the test if PGENE_TEST_DSN is not set.
func Connect(t *testing.T, schema string) *pgene.Client {
	t.Helper()

	dsn := os.Getenv("PGENE_TEST_DSN")
	if dsn == "" {
		t.Skip("PGENE_TEST_DSN is not set")
	}

	c, err := pgene.New(t.Context(), pgene.WithConnString(dsn))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	if schema != "" {
		_, err = c.Exec(t.Context(), pgene.Sql(schema))
		require.NoError(t, err)
	}

	return c
}
