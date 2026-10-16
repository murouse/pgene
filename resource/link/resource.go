// Package link управляет связями «многие ко многим» в таблице-связке между двумя справочниками.
package link

import (
	"github.com/murouse/pgene/resource"
)

// Config описывает таблицу-связку.
//
// Требования к схеме:
//   - на (LeftColumn, RightColumn) есть уникальный индекс или первичный ключ;
//   - LeftTable и RightTable — справочники с колонками id и deleted_at (как в пакете dict).
type Config struct {
	Table       string // таблица-связка, например "rack_tags"
	LeftColumn  string // колонка левой сущности в связке, например "rack_id"
	RightColumn string // колонка правой сущности в связке, например "tag_id"

	LeftTable  string // таблица левой сущности, например "racks"
	RightTable string // таблица правой сущности, например "tags"

	LeftNotFoundErr  error // левой записи нет или она удалена; по умолчанию resource.ErrNotFound
	RightNotFoundErr error // правой записи нет или она удалена; по умолчанию resource.ErrNotFound
}

// Resource отвечает за проставление/снятие связей (hard delete) в одной таблице-связке.
type Resource[L any, R comparable] struct {
	db  resource.DB
	cfg Config
}

func New[L any, R comparable](db resource.DB, cfg Config) *Resource[L, R] {
	if cfg.LeftNotFoundErr == nil {
		cfg.LeftNotFoundErr = resource.ErrNotFound
	}
	if cfg.RightNotFoundErr == nil {
		cfg.RightNotFoundErr = resource.ErrNotFound
	}

	return &Resource[L, R]{
		db:  db,
		cfg: cfg,
	}
}
