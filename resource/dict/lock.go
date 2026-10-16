package dict

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgene"
)

// Lock блокирует активную запись до конца транзакции или возвращает Config.NotFoundErr.
// Вызывайте внутри pgene.InTx: вне транзакции блокировка снимается сразу после запроса.
//
// Используется FOR NO KEY UPDATE, а не FOR UPDATE: id не меняется,
// поэтому вставки в таблицы со ссылкой на запись не ждут конца транзакции.
func (r *Resource[TM, TID]) Lock(ctx context.Context, id TID, opts ...Option) error {
	qb := pgene.Sq().
		Select(r.col("id")).
		From(r.cfg.Table).
		Where(r.filter(id, buildOptions(opts))).
		Suffix("FOR NO KEY UPDATE")

	var locked TID
	if err := r.db.Get(ctx, qb, &locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.cfg.NotFoundErr
		}
		return fmt.Errorf("lock: %w", err)
	}

	return nil
}
