// Package resource содержит общее для типовых ресурсов: справочников (dict) и связок (link).
package resource

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/murouse/pgene"
)

// DB — то, что ресурсам нужно от клиента; *pgene.Client его реализует.
type DB interface {
	pgene.TxController

	Get(ctx context.Context, sql pgene.Sqlizer, dest any) error
	Select(ctx context.Context, sql pgene.Sqlizer, dest any) error
	Exec(ctx context.Context, sql pgene.Sqlizer) (pgconn.CommandTag, error)
}

// ErrNotFound возвращается, если в конфиге ресурса не задана своя ошибка «не найдено».
var ErrNotFound = errors.New("resource: not found")

// Constraints сопоставляет имя ограничения БД (unique, foreign key, check) с доменной ошибкой:
//
//	resource.Constraints{"tags_name_uq": ErrTagExists}
type Constraints map[string]error

// Match возвращает доменную ошибку для ограничения, нарушенного в err, или nil.
func (c Constraints) Match(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	return c[pgErr.ConstraintName]
}
