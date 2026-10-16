package pgene

import (
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Option configures Client during initialization.
type Option func(*config)

// WithConnString sets raw connection string.
func WithConnString(connString string) Option {
	return func(c *config) {
		c.connString = connString
	}
}

// WithCreds builds connection string from credentials, escaping special characters.
func WithCreds(creds *Creds) Option {
	return func(c *config) {
		dsn := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(creds.User, creds.Pass),
			Host:     net.JoinHostPort(creds.Host, strconv.Itoa(creds.Port)),
			Path:     creds.DB,
			RawQuery: url.Values{"sslmode": {creds.SSLMode}}.Encode(),
		}
		c.connString = dsn.String()
	}
}

// WithPgxConfig adjusts pgx pool config parsed from the connection string,
// e.g. to set MaxConns or a query tracer. Fields not touched keep values from the DSN or pgx defaults.
func WithPgxConfig(configure func(*pgxpool.Config)) Option {
	return func(c *config) {
		c.pgxConfig = append(c.pgxConfig, configure)
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
