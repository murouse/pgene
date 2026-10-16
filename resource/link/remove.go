package link

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgene"
)

// Remove удаляет связь и возвращает true, если связь была удалена, и false, если связи не существовало.
func (r *Resource[L, R]) Remove(ctx context.Context, left L, right R) (bool, error) {
	qb := pgene.Sq().
		Delete(r.cfg.Table).
		Where(sq.Eq{
			r.cfg.LeftColumn:  left,
			r.cfg.RightColumn: right,
		})

	tag, err := r.db.Exec(ctx, qb)
	if err != nil {
		return false, fmt.Errorf("delete: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}
