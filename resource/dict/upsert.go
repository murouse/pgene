package dict

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgene"
)

// Upsert вставляет запись или обновляет активную запись с теми же значениями колонок key
// и возвращает её ID и true, если запись создана или изменена.
//
// Колонки key должны быть в data и покрываться уникальным индексом
// (частичным WHERE deleted_at IS NULL или обычным). Удалённые записи не обновляются.
func (r *Resource[TM, TID]) Upsert(ctx context.Context, key []string, data map[string]any) (TID, bool, error) {
	set := quoteKeys(data)

	keyCols := make([]string, len(key))
	byKey := sq.Eq{r.col("deleted_at"): nil}
	for i, col := range key {
		keyCols[i] = pgx.Identifier{col}.Sanitize()
		byKey[r.col(keyCols[i])] = data[col]
	}

	// обновляем только изменившиеся значения, как Update
	var updates, changes []string
	for _, col := range slices.Sorted(maps.Keys(set)) {
		if slices.Contains(keyCols, col) {
			continue
		}
		updates = append(updates, col+" = EXCLUDED."+col)
		changes = append(changes, r.col(col)+" IS DISTINCT FROM EXCLUDED."+col)
	}

	action := "DO NOTHING"
	if len(updates) > 0 {
		action = fmt.Sprintf("DO UPDATE SET %s, updated_at = now() WHERE %s",
			strings.Join(updates, ", "), strings.Join(changes, " OR "))
	}

	qb := pgene.Sq().
		Insert(r.cfg.Table).
		SetMap(set).
		Suffix(fmt.Sprintf("ON CONFLICT (%s) WHERE deleted_at IS NULL %s RETURNING id",
			strings.Join(keyCols, ", "), action))

	var id TID
	err := r.db.Get(ctx, qb, &id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return id, false, r.wrap("upsert", err)
	}

	// запись уже есть и не изменилась: RETURNING пуст, берём её id отдельно
	if err := r.db.Get(ctx, pgene.Sq().Select("id").From(r.cfg.Table).Where(byKey), &id); err != nil {
		return id, false, fmt.Errorf("get existing: %w", err)
	}

	return id, false, nil
}
