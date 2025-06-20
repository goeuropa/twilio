package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.False(t, config.Enabled)
	assert.Equal(t, 5, config.WorkerCount)
	assert.Equal(t, 1000, config.EventQueueSize)
	assert.Equal(t, 5*time.Second, config.ShutdownTimeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.RetryBackoff)
	assert.Equal(t, 5, config.CircuitBreakerConfig.FailureThreshold)
	assert.Equal(t, 3, config.CircuitBreakerConfig.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.CircuitBreakerConfig.Timeout)
	assert.Empty(t, config.Providers)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   Config
	}{
		{
			name: "valid config unchanged",
			config: Config{
				WorkerCount:     10,
				EventQueueSize:  500,
				ShutdownTimeout: 10 * time.Second,
				MaxRetries:      5,
				RetryBackoff:    2 * time.Second,
			},
			want: Config{
				WorkerCount:     10,
				EventQueueSize:  500,
				ShutdownTimeout: 10 * time.Second,
				MaxRetries:      5,
				RetryBackoff:    2 * time.Second,
			},
		},
		{
			name: "invalid worker count corrected",
			config: Config{
				WorkerCount:     0,
				EventQueueSize:  100,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      1,
				RetryBackoff:    time.Second,
			},
			want: Config{
				WorkerCount:     1,
				EventQueueSize:  100,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      1,
				RetryBackoff:    time.Second,
			},
		},
		{
			name: "invalid queue size corrected",
			config: Config{
				WorkerCount:     5,
				EventQueueSize:  5,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      1,
				RetryBackoff:    time.Second,
			},
			want: Config{
				WorkerCount:     5,
				EventQueueSize:  10,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      1,
				RetryBackoff:    time.Second,
			},
		},
		{
			name: "invalid shutdown timeout corrected",
			config: Config{
				WorkerCount:     5,
				EventQueueSize:  100,
				ShutdownTimeout: 100 * time.Millisecond,
				MaxRetries:      1,
				RetryBackoff:    time.Second,
			},
			want: Config{
				WorkerCount:     5,
				EventQueueSize:  100,
				ShutdownTimeout: time.Second,
				MaxRetries:      1,
				RetryBackoff:    time.Second,
			},
		},
		{
			name: "negative max retries corrected",
			config: Config{
				WorkerCount:     5,
				EventQueueSize:  100,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      -5,
				RetryBackoff:    time.Second,
			},
			want: Config{
				WorkerCount:     5,
				EventQueueSize:  100,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      0,
				RetryBackoff:    time.Second,
			},
		},
		{
			name: "invalid retry backoff corrected",
			config: Config{
				WorkerCount:     5,
				EventQueueSize:  100,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      1,
				RetryBackoff:    10 * time.Millisecond,
			},
			want: Config{
				WorkerCount:     5,
				EventQueueSize:  100,
				ShutdownTimeout: 2 * time.Second,
				MaxRetries:      1,
				RetryBackoff:    100 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assert.NoError(t, err)
			assert.Equal(t, tt.want.WorkerCount, tt.config.WorkerCount)
			assert.Equal(t, tt.want.EventQueueSize, tt.config.EventQueueSize)
			assert.Equal(t, tt.want.ShutdownTimeout, tt.config.ShutdownTimeout)
			assert.Equal(t, tt.want.MaxRetries, tt.config.MaxRetries)
			assert.Equal(t, tt.want.RetryBackoff, tt.config.RetryBackoff)
		})
	}
}
