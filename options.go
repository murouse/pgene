package pgene

import "fmt"

// Option configures Client during initialization.
type Option func(*config)

// WithConnString sets raw connection string.
func WithConnString(connString string) Option {
	return func(c *config) {
		c.conn.ConnString = connString
	}
}

// WithCreds builds connection string from credentials.
func WithCreds(creds *Creds) Option {
	return func(c *config) {
		c.conn.ConnString = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			creds.User,
			creds.Pass,
			creds.Host,
			creds.Port,
			creds.DB,
			creds.SSLMode,
		)
	}
}

// WithConnConfig overrides connection config.
func WithConnConfig(connConfig *ConnConfig) Option {
	return func(c *config) {
		c.conn = connConfig
	}
}

// WithPoolConfig overrides pool config.
func WithPoolConfig(poolConfig *PoolConfig) Option {
	return func(c *config) {
		c.pool = poolConfig
	}
}

// WithConstructorConfig overrides constructor behavior.
func WithConstructorConfig(constructorConfig *ConstructorConfig) Option {
	return func(c *config) {
		c.constructor = constructorConfig
	}
}

// WithLogger sets custom logger implementation.
func WithLogger(logger Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithMeter sets metrics implementation (placeholder).
func WithMeter(meter Meter) Option {
	return func(c *config) {
		c.meter = meter
	}
}
