package analytics

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed allows requests to pass through
	CircuitClosed CircuitState = iota
	// CircuitOpen blocks all requests
	CircuitOpen
	// CircuitHalfOpen allows a limited number of requests to test if the service has recovered
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for analytics providers.
type CircuitBreaker struct {
	mu              sync.RWMutex
	config          CircuitBreakerConfig
	state           CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	lastAttemptTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// Call executes the given function if the circuit allows it.
func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.CanCall() {
		return ErrProviderUnavailable
	}

	err := fn()
	cb.RecordResult(err)
	return err
}

// CanCall checks if the circuit breaker allows a call to proceed.
func (cb *CircuitBreaker) CanCall() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		// In half-open state, allow calls to test recovery
		return true
	default:
		return false
	}
}

// RecordResult records the result of a call and updates the circuit state.
func (cb *CircuitBreaker) RecordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastAttemptTime = time.Now()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()
		cb.successes = 0

		// Check if we should open the circuit
		if cb.state == CircuitClosed && cb.failures >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
		} else if cb.state == CircuitHalfOpen {
			// Single failure in half-open state reopens the circuit
			cb.state = CircuitOpen
			cb.failures = cb.config.FailureThreshold
		}
	} else {
		cb.successes++

		if cb.state == CircuitHalfOpen && cb.successes >= cb.config.SuccessThreshold {
			// Enough successes to close the circuit
			cb.state = CircuitClosed
			cb.failures = 0
		} else if cb.state == CircuitClosed && cb.failures > 0 {
			// Reset failure count on success in closed state
			cb.failures = 0
		}
	}
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to its initial state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailureTime = time.Time{}
	cb.lastAttemptTime = time.Time{}
}
