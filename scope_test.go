package pgene_test

import (
	"context"
	"errors"
	"testing"

	"github.com/murouse/pgene"
	"github.com/murouse/pgene/internal/testdb"
	"github.com/stretchr/testify/assert"
)

func TestAfterCommit(t *testing.T) {
	boom := errors.New("boom")

	// хук регистрируется во вложенной транзакции
	tests := []struct {
		name      string
		mode      pgene.NestedMode
		nestedErr error
		outerErr  error
		wantCalls int
	}{
		{name: "join, outer commits", mode: pgene.NestedJoin, wantCalls: 1},
		{name: "join fails, outer commits", mode: pgene.NestedJoin, nestedErr: boom, wantCalls: 1}, // данные блока тоже остаются
		{name: "join, outer rolls back", mode: pgene.NestedJoin, outerErr: boom},
		{name: "savepoint, both commit", mode: pgene.NestedSavepoint, wantCalls: 1},
		{name: "savepoint rolls back", mode: pgene.NestedSavepoint, nestedErr: boom},
		{name: "savepoint commits, outer rolls back", mode: pgene.NestedSavepoint, outerErr: boom},
	}

	c := testdb.Connect(t, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			_ = pgene.InTx(t.Context(), c, func(ctx context.Context) error {
				_ = pgene.InTx(ctx, c, func(ctx context.Context) error {
					c.AfterCommit(ctx, func() { calls++ })
					return tt.nestedErr
				}, pgene.WithNestedMode(tt.mode))
				return tt.outerErr
			})

			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestAfterCommitTopLevel(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantSeen []int // сколько строк видит другой клиент при каждом вызове хука
	}{
		{name: "commit", wantSeen: []int{1}},
		{name: "rollback", err: errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testdb.Connect(t, itemsSchema)
			b := testdb.Connect(t, "")

			var seen []int
			_ = pgene.InTx(t.Context(), a, func(ctx context.Context) error {
				if err := insert(ctx, a, "x"); err != nil {
					return err
				}
				a.AfterCommit(ctx, func() {
					var n int
					assert.NoError(t, b.Get(t.Context(), pgene.Sql(`SELECT count(*) FROM tx_items`), &n))
					seen = append(seen, n)
				})
				return tt.err
			})

			assert.Equal(t, tt.wantSeen, seen)
		})
	}
}

func TestAfterCommitWithoutTx(t *testing.T) {
	c := testdb.Connect(t, "")

	calls := 0
	c.AfterCommit(t.Context(), func() { calls++ })

	assert.Equal(t, 1, calls)
}
