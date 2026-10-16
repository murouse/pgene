package dict

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgene"
)

// Get возвращает одну активную запись по её ID (где deleted_at IS NULL).
// Если запись не найдена, возвращает Config.NotFoundErr.
func (r *Resource[TM, TID]) Get(ctx context.Context, id TID, columns []string, opts ...Option) (TM, error) {
	o := buildOptions(opts)

	qb := o.extendSelect(pgene.Sq().
		Select(columns...).
		From(r.cfg.Table).
		Where(r.filter(id, o)))

	var item TM
	if err := r.db.Get(ctx, qb, &item); err != nil {
		var zero TM
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, r.cfg.NotFoundErr
		}
		return zero, fmt.Errorf("get: %w", err)
	}

	return item, nil
}
