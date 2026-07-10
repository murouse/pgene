package pgene

import (
	"time"

	"github.com/jackc/pgx/v5"
)

// config is an internal aggregate of all configuration groups.
type config struct {
	conn        *ConnConfig
	pool        *PoolConfig
	constructor *ConstructorConfig
	logger      Logger
	meter       Meter
}

// ConnConfig defines pgx connection-level settings.
type ConnConfig struct {
	ConnString string

	StatementCacheCapacity   int
	DescriptionCacheCapacity int
	DefaultQueryExecMode     pgx.QueryExecMode
}

// PoolConfig defines connection pool behavior.
type PoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MinIdleConns      int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// ConstructorConfig controls client initialization behavior.
type ConstructorConfig struct {
	Ping        bool
	PingTimeout time.Duration
}

// Creds represents database credentials used to build DSN.
type Creds struct {
	User    string
	Pass    string
	Host    string
	Port    int
	DB      string
	SSLMode string
}
