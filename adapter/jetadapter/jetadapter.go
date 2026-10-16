// Package jetadapter lets jet statements be passed to pgene.Client.
package jetadapter

import (
	jet "github.com/go-jet/jet/v2/postgres"
)

// Wrapper is an adapter that makes jet statements compatible with pgene.Sqlizer.
type Wrapper struct {
	jet.Statement
}

// ToSql adapts jet's Sql() method to match the Sqlizer interface.
func (w *Wrapper) ToSql() (string, []any, error) {
	query, args := w.Sql()
	return query, args, nil
}

// Wrap wraps a jet statement into a Sqlizer-compatible type.
//
// Usage:
//
//	c.Exec(ctx, jetadapter.Wrap(jet.SELECT(...)))
func Wrap(statement jet.Statement) *Wrapper {
	return &Wrapper{statement}
}
