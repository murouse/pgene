// Package pgtest helps to test code that uses pgene.
package pgtest

import (
	"context"
	"testing"

	"github.com/murouse/pgene"
)

// Tx starts a sandbox transaction (see pgene.Client.BeginSandbox) and rolls it back when the test ends,
// so the test leaves no data behind. Pass the returned context to the code under test.
//
// Each such test holds one connection of the pool until it ends.
func Tx(t testing.TB, c *pgene.Client) context.Context {
	t.Helper()

	ctx, err := c.BeginSandbox(t.Context())
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}

	t.Cleanup(func() {
		if err := c.RollbackTx(context.WithoutCancel(ctx)); err != nil {
			t.Errorf("pgtest: %v", err)
		}
	})

	return ctx
}
