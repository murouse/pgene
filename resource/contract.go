package resource

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/murouse/pgene"
)

type DB interface {
	pgene.TxController

	Get(ctx context.Context, sql pgene.Sqlizer, dest any) error
	Select(ctx context.Context, sql pgene.Sqlizer, dest any) error
	Exec(ctx context.Context, sql pgene.Sqlizer) (pgconn.CommandTag, error)
}

type (
	QueryFunc         func(ctx context.Context) error
	ExtendBuilderFunc func(b sq.SelectBuilder) sq.SelectBuilder
)
