package analytics

import (
	"time"
)

// Config holds the configuration for the analytics system.
type Config struct {
	// Enabled determines if analytics is active
	Enabled bool

	// Salt for hashing phone numbers (should be kept secret)
	HashSalt string

	// Worker pool configuration
	WorkerCount     int
	EventQueueSize  int
	ShutdownTimeout time.Duration

	// Error handling
	MaxRetries           int
	RetryBackoff         time.Duration
	CircuitBreakerConfig CircuitBreakerConfig

	// Provider configurations
	Providers []ProviderConfig
}

// ProviderConfig holds configuration for a specific analytics provider.
type ProviderConfig struct {
	// Name identifies the provider (e.g., "plausible", "google")
	Name string

	// Enabled determines if this provider is active
	Enabled bool

	// Provider-specific configuration as key-value pairs
	Config map[string]interface{}
}

// CircuitBreakerConfig holds configuration for the circuit breaker pattern.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int

	// SuccessThreshold is the number of successes before closing the circuit
	SuccessThreshold int

	// Timeout is how long to wait before trying again after circuit opens
	Timeout time.Duration
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:         false,
		WorkerCount:     5,
		EventQueueSize:  1000,
		ShutdownTimeout: 5 * time.Second,
		MaxRetries:      3,
		RetryBackoff:    time.Second,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
		},
		Providers: []ProviderConfig{},
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.WorkerCount < 1 {
		c.WorkerCount = 1
	}
	if c.EventQueueSize < 10 {
		c.EventQueueSize = 10
	}
	if c.ShutdownTimeout < time.Second {
		c.ShutdownTimeout = time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0
	}
	if c.RetryBackoff < 100*time.Millisecond {
		c.RetryBackoff = 100 * time.Millisecond
	}
	return nil
}
