package analytics

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          time.Second,
	}
	cb := NewCircuitBreaker(config)

	assert.Equal(t, CircuitClosed, cb.GetState())
	assert.True(t, cb.CanCall())
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          time.Second,
	}
	cb := NewCircuitBreaker(config)

	testErr := errors.New("test error")

	// Record failures
	for i := 0; i < config.FailureThreshold-1; i++ {
		err := cb.Call(func() error { return testErr })
		assert.Equal(t, testErr, err)
		assert.Equal(t, CircuitClosed, cb.GetState())
	}

	// One more failure should open the circuit
	err := cb.Call(func() error { return testErr })
	assert.Equal(t, testErr, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Subsequent calls should fail immediately
	err = cb.Call(func() error { return nil })
	assert.Equal(t, ErrProviderUnavailable, err)
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	err := cb.Call(func() error { return errors.New("error") })
	assert.NotNil(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Should transition to half-open
	assert.True(t, cb.CanCall())
	assert.Equal(t, CircuitHalfOpen, cb.GetState())
}

func TestCircuitBreaker_ClosesAfterSuccesses(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	err := cb.Call(func() error { return errors.New("error") })
	assert.NotNil(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Record successes in half-open state
	for i := 0; i < config.SuccessThreshold; i++ {
		err = cb.Call(func() error { return nil })
		assert.NoError(t, err)
		if i < config.SuccessThreshold-1 {
			assert.Equal(t, CircuitHalfOpen, cb.GetState())
		}
	}

	// Circuit should be closed now
	assert.Equal(t, CircuitClosed, cb.GetState())
}

func TestCircuitBreaker_ReopensOnHalfOpenFailure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	err := cb.Call(func() error { return errors.New("error") })
	assert.NotNil(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Fail in half-open state
	err = cb.Call(func() error { return errors.New("error") })
	assert.NotNil(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          time.Second,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	err := cb.Call(func() error { return errors.New("error") })
	assert.NotNil(t, err)
	assert.Equal(t, CircuitOpen, cb.GetState())

	// Reset
	cb.Reset()
	assert.Equal(t, CircuitClosed, cb.GetState())
	assert.True(t, cb.CanCall())
}
