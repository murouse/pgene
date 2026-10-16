package link

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgene"
	"github.com/samber/lo"
)

// Replace атомарно заменяет набор правых сущностей, связанных с leftID, на rightIDs
// и возвращает true, если набор связей изменился.
//
// Метод выполняет следующие шаги в рамках единой транзакции:
//  1. Блокирует левую запись (FOR NO KEY UPDATE), чтобы параллельные Replace не перемешали наборы.
//  2. Добавляет недостающие связи — только если все rightIDs существуют и не удалены.
//     Иначе возвращает RightNotFoundErr, ничего не изменив.
//  3. Удаляет связи, которых нет в rightIDs.
//
// Уже существующие связи не пересоздаются.
func (r *Resource[L, R]) Replace(ctx context.Context, leftID L, rightIDs []R) (bool, error) {
	rightIDs = lo.Uniq(rightIDs) // защита от дублей; никогда не nil, иначе ANY($2) получил бы NULL

	qLock := fmt.Sprintf(`SELECT id FROM %s WHERE id = $1 AND deleted_at IS NULL FOR NO KEY UPDATE`,
		r.cfg.LeftTable)

	// вставляем, только если найдены все правые записи: при ошибке ничего не меняется
	qInsert := fmt.Sprintf(`
		WITH found AS (
			SELECT id FROM %[1]s WHERE id = ANY($2) AND deleted_at IS NULL FOR SHARE
		), added AS (
			INSERT INTO %[2]s (%[3]s, %[4]s)
			SELECT $1, id FROM found
			WHERE (SELECT count(*) FROM found) = $3
			ON CONFLICT (%[3]s, %[4]s) DO NOTHING
			RETURNING 1
		)
		SELECT (SELECT count(*) FROM found) AS found,
		       (SELECT count(*) FROM added) AS added`,
		r.cfg.RightTable, r.cfg.Table, r.cfg.LeftColumn, r.cfg.RightColumn,
	)

	qDelete := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1 AND NOT (%s = ANY($2))`,
		r.cfg.Table, r.cfg.LeftColumn, r.cfg.RightColumn)

	return pgene.GetInTx(ctx, r.db, func(ctx context.Context) (bool, error) {
		var locked L
		if err := r.db.Get(ctx, pgene.Sql(qLock, leftID), &locked); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, r.cfg.LeftNotFoundErr
			}
			return false, fmt.Errorf("lock: %w", err)
		}

		var inserted struct {
			Found int
			Added int
		}
		if err := r.db.Get(ctx, pgene.Sql(qInsert, leftID, rightIDs, len(rightIDs)), &inserted); err != nil {
			return false, fmt.Errorf("insert: %w", err)
		}
		if inserted.Found != len(rightIDs) {
			return false, r.cfg.RightNotFoundErr
		}

		deleted, err := r.db.Exec(ctx, pgene.Sql(qDelete, leftID, rightIDs))
		if err != nil {
			return false, fmt.Errorf("delete: %w", err)
		}

		return inserted.Added > 0 || deleted.RowsAffected() > 0, nil
	})
}
