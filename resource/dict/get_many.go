package dict

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

// GetMany возвращает активные записи с указанными ID одним запросом, по возрастанию id.
// Отсутствующие и удалённые ID пропускаются.
func (r *Resource[TM, TID]) GetMany(ctx context.Context, ids []TID, columns []string, opts ...Option) ([]TM, error) {
	return r.List(ctx, columns, append([]Option{Where(sq.Expr(r.col("id")+" = ANY(?)", ids))}, opts...)...)
}
