// Package goquadapter lets goqu expressions be passed to pgene.Client.
package goquadapter

import (
	goqu "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // register postgres dialect for goqu
	goquexp "github.com/doug-martin/goqu/v9/exp"
)

// Wrapper is an adapter that makes goqu expressions compatible with pgene.Sqlizer.
//
// Reason:
// - goqu uses ToSQL()
// - pgene expects ToSql()
// This bridges the API mismatch between the two libraries.
type Wrapper struct {
	goquexp.SQLExpression
}

// ToSql adapts goqu's ToSQL() method to match Sqlizer interface.
func (w *Wrapper) ToSql() (string, []any, error) {
	return w.ToSQL()
}

// Wrap wraps a goqu expression into a Sqlizer-compatible type.
//
// goqu inlines values into SQL unless the dataset is prepared, so call Prepared(true)
// to send them as parameters and let pgx reuse cached statements.
//
// Usage:
//
//	c.Exec(ctx, goquadapter.Wrap(goquadapter.Dialect().Select(...).Prepared(true)))
func Wrap(expr goquexp.SQLExpression) *Wrapper {
	return &Wrapper{expr}
}

// Dialect returns a PostgreSQL dialect for goqu.
// Extracted into a helper for consistency and convenience.
func Dialect() goqu.DialectWrapper {
	return goqu.Dialect("postgres")
}
