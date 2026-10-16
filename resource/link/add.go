package link

import (
	"context"
	"fmt"

	"github.com/murouse/pgene"
)

// Add устанавливает связь и возвращает true, если связь была добавлена, и false, если связь уже была.
// Если левой или правой записи нет или она удалена, возвращает LeftNotFoundErr или RightNotFoundErr.
//
// Обе записи блокируются FOR SHARE, поэтому параллельное удаление любой из них
// дождётся конца транзакции и не оставит связь с удалённой записью.
func (r *Resource[L, R]) Add(ctx context.Context, left L, right R) (bool, error) {
	q := fmt.Sprintf(`
		WITH l AS (
			SELECT id FROM %[1]s WHERE id = $1 AND deleted_at IS NULL FOR SHARE
		), r AS (
			SELECT id FROM %[2]s WHERE id = $2 AND deleted_at IS NULL FOR SHARE
		), added AS (
			INSERT INTO %[3]s (%[4]s, %[5]s)
			SELECT l.id, r.id FROM l, r
			ON CONFLICT (%[4]s, %[5]s) DO NOTHING
			RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM l) AS left_found,
		       EXISTS (SELECT 1 FROM r) AS right_found,
		       EXISTS (SELECT 1 FROM added) AS added`,
		r.cfg.LeftTable, r.cfg.RightTable, r.cfg.Table, r.cfg.LeftColumn, r.cfg.RightColumn,
	)

	var res struct {
		LeftFound  bool
		RightFound bool
		Added      bool
	}
	if err := r.db.Get(ctx, pgene.Sql(q, left, right), &res); err != nil {
		return false, fmt.Errorf("insert: %w", err)
	}

	switch {
	case !res.LeftFound:
		return false, r.cfg.LeftNotFoundErr
	case !res.RightFound:
		return false, r.cfg.RightNotFoundErr
	}

	return res.Added, nil
}
