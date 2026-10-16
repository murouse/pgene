package pgene_test

import (
	"context"
	"testing"

	"github.com/murouse/pgene"
	"github.com/murouse/pgene/internal/testdb"
	"github.com/stretchr/testify/assert"
)

func TestAdvisoryLock(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantErrCode string
	}{
		{name: "same key waits", key: "k", wantErrCode: "55P03"}, // lock_not_available
		{name: "other key", key: "other"},
	}

	a := testdb.Connect(t, "")
	b := testdb.Connect(t, "")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pgene.InTx(t.Context(), a, func(ctx context.Context) error {
				if err := a.AdvisoryLock(ctx, "k"); err != nil {
					return err
				}
				return pgene.InTx(ctx, b, func(ctx context.Context) error {
					if _, err := b.Exec(ctx, pgene.Sql(`SET LOCAL lock_timeout = '100ms'`)); err != nil {
						return err
					}
					return b.AdvisoryLock(ctx, tt.key)
				})
			})

			assert.Equal(t, tt.wantErrCode, pgCode(err))
		})
	}
}
