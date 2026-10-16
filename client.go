package pgene

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps pgx connection pool and provides high-level DB API.
type Client struct {
	pool *pgxpool.Pool
}

// New creates a new Client with provided options.
// Validates config, initializes pgx pool and optionally pings DB.
func New(ctx context.Context, opts ...Option) (*Client, error) {
	cfg := &config{
		constructor: defaultConstructorConfig(),
		logger:      &noopLogger{},
		meter:       &noopMeter{},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.connString == "" {
		return nil, fmt.Errorf("connection string is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.connString)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	for _, configure := range cfg.pgxConfig {
		configure(poolConfig)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// ping with timeout
	if cfg.constructor.Ping {
		pingCtx, cancel := context.WithTimeout(ctx, cfg.constructor.PingTimeout)
		defer cancel()

		if err := pool.Ping(pingCtx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("ping db: %w", err)
		}
	}

	return &Client{pool: pool}, nil
}

// Close gracefully closes the underlying connection pool.
func (c *Client) Close() error {
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// Pool exposes underlying pgxpool.Pool for advanced usage.
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}
