package pgene

// Meter is a placeholder for metrics/telemetry integration.
type Meter interface{}

// noopMeter is a no-op metrics implementation.
type noopMeter struct{}
