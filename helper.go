package pgene

import (
	sq "github.com/Masterminds/squirrel"
)

// Sq returns squirrel statement builder configured for PostgreSQL ($ placeholders).
func Sq() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

// RawQuery is a simple Sqlizer implementation for raw queries.
type RawQuery struct {
	query string
	args  []any
}

// ToSql returns raw query and arguments.
func (pq *RawQuery) ToSql() (string, []any, error) {
	return pq.query, pq.args, nil
}

// Sql creates a new raw SQL query wrapper.
func Sql(query string, args ...interface{}) *RawQuery {
	return &RawQuery{
		query: query,
		args:  args,
	}
}
