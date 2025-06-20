package plausible

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"oba-twilio/analytics"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "https://plausible.io", config.APIURL)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 10*time.Second, config.FlushInterval)
	assert.Equal(t, 30*time.Second, config.HTTPTimeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.RetryDelay)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Domain: "example.com",
				APIURL: "https://plausible.io",
			},
			wantErr: false,
		},
		{
			name: "missing domain",
			config: Config{
				APIURL: "https://plausible.io",
			},
			wantErr: true,
		},
		{
			name: "empty API URL gets default",
			config: Config{
				Domain: "example.com",
				APIURL: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.config.APIURL == "" {
					assert.Equal(t, "https://plausible.io", tt.config.APIURL)
				}
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	config := Config{
		Domain: "example.com",
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, config.Domain, provider.config.Domain)
	assert.NotNil(t, provider.client)
	assert.NotNil(t, provider.eventBatch)

	// Clean up
	err = provider.Close()
	assert.NoError(t, err)
}

func TestNewProviderInvalidConfig(t *testing.T) {
	config := Config{
		// Missing domain
	}

	provider, err := NewProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestProvider_TrackEvent(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/event", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Domain:    "example.com",
		APIURL:    server.URL,
		BatchSize: 1, // Force immediate flush
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()
	event := analytics.NewEvent("test_event", "user-123")
	event.Properties["test_prop"] = "test_value"

	err = provider.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for the flush to complete
	time.Sleep(50 * time.Millisecond)
}

func TestProvider_TrackEventBatching(t *testing.T) {
	requestCount := 0
	var receivedEvents []PlausibleEvent

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var events []PlausibleEvent
		err := json.NewDecoder(r.Body).Decode(&events)
		assert.NoError(t, err)

		receivedEvents = append(receivedEvents, events...)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Domain:        "example.com",
		APIURL:        server.URL,
		BatchSize:     3,             // Batch 3 events
		FlushInterval: 1 * time.Hour, // Long interval to test batching
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()

	// Send 2 events (should not trigger flush)
	for i := 0; i < 2; i++ {
		event := analytics.NewEvent("test_event", "user-123")
		event.Properties["index"] = i
		err = provider.TrackEvent(ctx, event)
		assert.NoError(t, err)
	}

	// Give some time and verify no request was made
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, requestCount)

	// Send third event (should trigger flush)
	event := analytics.NewEvent("test_event", "user-123")
	event.Properties["index"] = 2
	err = provider.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for the flush to complete
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, requestCount)
	assert.Len(t, receivedEvents, 3)
}

func TestProvider_FlushInterval(t *testing.T) {
	requestCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Domain:        "example.com",
		APIURL:        server.URL,
		BatchSize:     100,                    // Large batch size
		FlushInterval: 100 * time.Millisecond, // Short interval
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()
	event := analytics.NewEvent("test_event", "user-123")

	err = provider.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Wait for flush interval
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 1, requestCount)
}

func TestProvider_Flush(t *testing.T) {
	requestCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Domain:        "example.com",
		APIURL:        server.URL,
		BatchSize:     100,           // Large batch size
		FlushInterval: 1 * time.Hour, // Long interval
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()
	event := analytics.NewEvent("test_event", "user-123")

	err = provider.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// No request should be made yet
	assert.Equal(t, 0, requestCount)

	// Manual flush should trigger request
	err = provider.Flush(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// Second flush with no events should not make a request
	err = provider.Flush(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount)
}

func TestProvider_RetryLogic(t *testing.T) {
	requestCount := 0

	// Create test server that fails first request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := Config{
		Domain:     "example.com",
		APIURL:     server.URL,
		BatchSize:  1,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()
	event := analytics.NewEvent("test_event", "user-123")

	err = provider.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give time for retries
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, requestCount) // First failed, second succeeded
}

func TestProvider_NoRetryOn4xx(t *testing.T) {
	requestCount := 0

	// Create test server that returns 400 (client error)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	config := Config{
		Domain:        "example.com",
		APIURL:        server.URL,
		BatchSize:     100,           // Don't auto-flush
		FlushInterval: 1 * time.Hour, // Don't auto-flush
		MaxRetries:    2,
		RetryDelay:    10 * time.Millisecond,
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()
	event := analytics.NewEvent("test_event", "user-123")

	// Track event (should succeed and not flush yet)
	err = provider.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Manual flush should trigger the request and fail with 400
	err = provider.Flush(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")

	// Should only have made 1 request (no retries on 4xx)
	assert.Equal(t, 1, requestCount)
}

func TestProvider_ConvertEvent(t *testing.T) {
	config := Config{
		Domain: "example.com",
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	event := analytics.NewEvent("sms_request", "user-hash-123")
	event.SessionID = "session-456"
	event.Version = 2
	event.Properties["language"] = "es-US"
	event.Properties["query"] = "75403"
	event.Properties["phone_number"] = "+12065551234" // Should be filtered out

	plausibleEvent := provider.convertEvent(event)

	assert.Equal(t, "sms_request", plausibleEvent.Name)
	assert.Equal(t, "example.com", plausibleEvent.Domain)
	assert.Contains(t, plausibleEvent.URL, "example.com")
	assert.NotEmpty(t, plausibleEvent.Timestamp)

	// Check properties
	assert.Equal(t, "es-US", plausibleEvent.Props["language"])
	assert.Equal(t, "75403", plausibleEvent.Props["query"])
	assert.Equal(t, "user-hash-123", plausibleEvent.Props["user_id"])
	assert.Equal(t, "session-456", plausibleEvent.Props["session_id"])
	assert.Equal(t, 2, plausibleEvent.Props["event_version"])

	// Sensitive data should be filtered out
	assert.NotContains(t, plausibleEvent.Props, "phone_number")
}

func TestProvider_Close(t *testing.T) {
	config := Config{
		Domain:        "example.com",
		FlushInterval: 100 * time.Millisecond,
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)

	// Close should work
	err = provider.Close()
	assert.NoError(t, err)

	// Second close should return error
	err = provider.Close()
	assert.Equal(t, analytics.ErrProviderClosed, err)

	// Operations after close should fail
	ctx := context.Background()
	event := analytics.NewEvent("test_event", "user-123")

	err = provider.TrackEvent(ctx, event)
	assert.Equal(t, analytics.ErrProviderClosed, err)

	err = provider.Flush(ctx)
	assert.Equal(t, analytics.ErrProviderClosed, err)
}

func TestProvider_InvalidEvent(t *testing.T) {
	config := Config{
		Domain: "example.com",
	}

	provider, err := NewProvider(config)
	assert.NoError(t, err)
	defer func() {
		_ = provider.Close()
	}()

	ctx := context.Background()
	event := analytics.Event{} // Invalid event (missing required fields)

	err = provider.TrackEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event")
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		StatusCode: 500,
		Status:     "Internal Server Error",
	}

	assert.Equal(t, "HTTP 500: Internal Server Error", err.Error())
}
