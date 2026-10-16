package link

import (
	"context"
	"fmt"

	"github.com/murouse/pgene"
)

// Rights возвращает ID правых записей, связанных с left, по возрастанию; удалённые записи пропускаются.
func (r *Resource[L, R]) Rights(ctx context.Context, left L) ([]R, error) {
	var ids []R
	if err := r.db.Select(ctx, r.linked(r.cfg.RightColumn, r.cfg.RightTable, r.cfg.LeftColumn, left), &ids); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	return ids, nil
}

// Lefts возвращает ID левых записей, связанных с right, по возрастанию; удалённые записи пропускаются.
func (r *Resource[L, R]) Lefts(ctx context.Context, right R) ([]L, error) {
	var ids []L
	if err := r.db.Select(ctx, r.linked(r.cfg.LeftColumn, r.cfg.LeftTable, r.cfg.RightColumn, right), &ids); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	return ids, nil
}

// linked выбирает колонку col связок со значением id в колонке byCol, пропуская удалённые записи table.
func (r *Resource[L, R]) linked(col, table, byCol string, id any) pgene.Sqlizer {
	return pgene.Sql(fmt.Sprintf(`
		SELECT l.%[1]s FROM %[3]s l
		JOIN %[2]s t ON t.id = l.%[1]s AND t.deleted_at IS NULL
		WHERE l.%[4]s = $1
		ORDER BY l.%[1]s`,
		col, table, r.cfg.Table, byCol,
	), id)
}
