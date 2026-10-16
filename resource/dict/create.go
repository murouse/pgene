package dict

import (
	"context"

	"github.com/murouse/pgene"
)

// Create вставляет новую запись в таблицу и возвращает её ID.
func (r *Resource[TM, TID]) Create(ctx context.Context, data map[string]any) (TID, error) {
	qb := pgene.Sq().
		Insert(r.cfg.Table).
		SetMap(quoteKeys(data)).
		Suffix("RETURNING id")

	var id TID
	if err := r.db.Get(ctx, qb, &id); err != nil {
		var zero TID
		return zero, r.wrap("insert", err)
	}

	return id, nil
}
