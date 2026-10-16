package dict

import (
	"context"
	"fmt"
	"maps"
	"slices"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgene"
)

// Update обновляет поля записи по ID и возвращает true, если значения действительно изменились.
// Если ни одно значение не отличается от текущего, запись (и её updated_at) не трогается.
// Если записи нет, возвращает Config.NotFoundErr.
//
// Колонки типа json (не jsonb) не поддерживаются: в PostgreSQL для json нет оператора сравнения.
func (r *Resource[TM, TID]) Update(ctx context.Context, id TID, data map[string]any, opts ...Option) (bool, error) {
	filter := r.filter(id, buildOptions(opts))
	set := quoteKeys(data)

	// колонки в фиксированном порядке: один текст запроса — один prepared statement
	changes := make(sq.Or, 0, len(set))
	for _, col := range slices.Sorted(maps.Keys(set)) {
		changes = append(changes, sq.Expr(col+" IS DISTINCT FROM ?", set[col]))
	}

	qb := pgene.Sq().
		Update(r.cfg.Table).
		SetMap(set).
		Set("updated_at", sq.Expr("now()")).
		Where(filter).
		Where(changes)

	tag, err := r.db.Exec(ctx, qb)
	if err != nil {
		return false, r.wrap("update", err)
	}
	if tag.RowsAffected() > 0 {
		return true, nil
	}

	// ничего не обновили: либо записи нет, либо значения те же
	return false, r.ensureExists(ctx, filter)
}

// ensureExists возвращает Config.NotFoundErr, если под filter нет записи.
func (r *Resource[TM, TID]) ensureExists(ctx context.Context, filter sq.Sqlizer) error {
	qb := pgene.Sq().
		Select("1").
		From(r.cfg.Table).
		Where(filter).
		Prefix("SELECT EXISTS (").
		Suffix(")")

	var exists bool
	if err := r.db.Get(ctx, qb, &exists); err != nil {
		return fmt.Errorf("check exists: %w", err)
	}
	if !exists {
		return r.cfg.NotFoundErr
	}

	return nil
}
