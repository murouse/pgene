// Package dict — CRUD над справочниками с soft delete.
//
// Каждый метод — один запрос без собственной транзакции. Проверки и побочные действия
// оборачивайте в pgene.InTx вместе с вызовом метода; Lock блокирует запись, чтобы
// проверка не устарела к моменту изменения:
//
//	err := pgene.InTx(ctx, db, func(ctx context.Context) error {
//		if err := tags.Lock(ctx, id); err != nil {
//			return err
//		}
//		if err := ensureTagUnused(ctx, id); err != nil {
//			return err
//		}
//		return tags.Delete(ctx, id)
//	})
package dict

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgene/resource"
)

// Config описывает таблицу справочника.
//
// В таблице должны быть колонки id, updated_at и deleted_at: активными считаются записи
// с deleted_at IS NULL. Уникальные индексы стоит делать частичными (WHERE deleted_at IS NULL),
// иначе удалённая запись не даст создать такую же.
type Config struct {
	Table       string               // название таблицы
	NotFoundErr error                // ошибка «запись не найдена»; по умолчанию resource.ErrNotFound
	Constraints resource.Constraints // ошибки для нарушенных ограничений, например already_exist
}

// Resource отвечает за crud над справочниками с soft-delete
type Resource[TM, TID any] struct {
	db  resource.DB
	cfg Config
}

func New[TM, TID any](db resource.DB, cfg Config) *Resource[TM, TID] {
	if cfg.NotFoundErr == nil {
		cfg.NotFoundErr = resource.ErrNotFound
	}

	return &Resource[TM, TID]{
		db:  db,
		cfg: cfg,
	}
}

// col возвращает колонку, уточнённую именем таблицы, чтобы она не конфликтовала с JOIN из Extend.
func (r *Resource[TM, TID]) col(name string) string {
	return r.cfg.Table + "." + name
}

// filter — условие «активная запись с этим id» вместе с условиями из Where.
func (r *Resource[TM, TID]) filter(id TID, o *options) sq.And {
	return append(sq.And{sq.Eq{r.col("id"): id, r.col("deleted_at"): nil}}, o.where...)
}

// wrap заменяет нарушение ограничения из Config.Constraints доменной ошибкой, остальные ошибки оборачивает.
func (r *Resource[TM, TID]) wrap(op string, err error) error {
	if domainErr := r.cfg.Constraints.Match(err); domainErr != nil {
		return domainErr
	}
	return fmt.Errorf("%s: %w", op, err)
}

// quoteKeys экранирует имена колонок из data, чтобы ключи не могли внедрить SQL.
func quoteKeys(data map[string]any) map[string]any {
	quoted := make(map[string]any, len(data))
	for col, val := range data {
		quoted[pgx.Identifier{col}.Sanitize()] = val
	}
	return quoted
}
