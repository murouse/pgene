package dict

import (
	sq "github.com/Masterminds/squirrel"
)

// Option уточняет вызов метода.
type Option func(o *options)

type options struct {
	where  []sq.Sqlizer
	extend []func(b sq.SelectBuilder) sq.SelectBuilder
}

func buildOptions(opts []Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Where добавляет условие к любому методу, например ограничение по владельцу:
//
//	tags.Delete(ctx, id, dict.Where(sq.Eq{"owner_id": userID}))
func Where(pred sq.Sqlizer) Option {
	return func(o *options) {
		o.where = append(o.where, pred)
	}
}

// Extend дорабатывает SELECT в Get и List: JOIN, сортировка, LIMIT, FOR SHARE.
// Колонки таблицы справочника уточняйте её именем ("tags.name"), если добавляете JOIN.
func Extend(fn func(b sq.SelectBuilder) sq.SelectBuilder) Option {
	return func(o *options) {
		o.extend = append(o.extend, fn)
	}
}

func (o *options) extendSelect(b sq.SelectBuilder) sq.SelectBuilder {
	for _, fn := range o.extend {
		b = fn(b)
	}
	return b
}
