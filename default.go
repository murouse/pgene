package pgene

import (
	"time"
)

// defaultConstructorConfig returns default constructor settings.
func defaultConstructorConfig() *ConstructorConfig {
	return &ConstructorConfig{
		Ping:        true,
		PingTimeout: 5 * time.Second,
	}
}
