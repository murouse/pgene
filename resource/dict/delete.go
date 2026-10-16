package dict

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgene"
)

// Delete помечает запись как удаленную (soft delete). Если записи нет, возвращает Config.NotFoundErr.
// Проверки перед удалением делайте после Lock в той же транзакции (см. пример в описании пакета).
func (r *Resource[TM, TID]) Delete(ctx context.Context, id TID, opts ...Option) error {
	qb := pgene.Sq().
		Update(r.cfg.Table).
		Set("updated_at", sq.Expr("now()")).
		Set("deleted_at", sq.Expr("now()")).
		Where(r.filter(id, buildOptions(opts)))

	tag, err := r.db.Exec(ctx, qb)
	if err != nil {
		return r.wrap("delete", err)
	}
	if tag.RowsAffected() == 0 {
		return r.cfg.NotFoundErr
	}

	return nil
}
