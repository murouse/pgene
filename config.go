package pgene

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// config is an internal aggregate of all configuration groups.
type config struct {
	connString  string
	pgxConfig   []func(*pgxpool.Config)
	constructor *ConstructorConfig
	logger      Logger
	meter       Meter
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
