package dict

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgene"
)

// List возвращает список активных записей (где deleted_at IS NULL).
// Сортировка по id добавляется последней: сортировки из Extend идут раньше,
// а порядок записей с равными значениями всегда один и тот же.
func (r *Resource[TM, TID]) List(ctx context.Context, columns []string, opts ...Option) ([]TM, error) {
	o := buildOptions(opts)

	qb := o.extendSelect(pgene.Sq().
		Select(columns...).
		From(r.cfg.Table).
		Where(append(sq.And{sq.Eq{r.col("deleted_at"): nil}}, o.where...))).
		OrderBy(r.col("id"))

	var items []TM
	if err := r.db.Select(ctx, qb, &items); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	return items, nil
}
