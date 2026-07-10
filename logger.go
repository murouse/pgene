package pgene

// Logger defines minimal logging interface used by the client.
type Logger interface {
	Debugf(msg string, args ...any)
	Infof(msg string, args ...any)
	Warnf(msg string, args ...any)
	Errorf(msg string, args ...any)
}

// noopLogger is a no-op logger implementation.
type noopLogger struct{}

func (n *noopLogger) Debugf(_ string, _ ...any) {}

func (n *noopLogger) Infof(_ string, _ ...any) {}

func (n *noopLogger) Warnf(_ string, _ ...any) {}

func (n *noopLogger) Errorf(_ string, _ ...any) {}
