package pgene

import (
	"time"

	"github.com/jackc/pgx/v5"
)

// defaultConnConfig returns default connection settings.
func defaultConnConfig() *ConnConfig {
	return &ConnConfig{
		StatementCacheCapacity:   512,
		DescriptionCacheCapacity: 512,
		DefaultQueryExecMode:     pgx.QueryExecModeCacheStatement,
	}
}

// defaultPoolConfig returns default pool settings.
func defaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConns:          4,
		MinConns:          1,
		MinIdleConns:      0,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
	}
}

// defaultConstructorConfig returns default constructor settings.
func defaultConstructorConfig() *ConstructorConfig {
	return &ConstructorConfig{
		Ping:        true,
		PingTimeout: 5 * time.Second,
	}
}
