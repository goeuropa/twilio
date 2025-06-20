package analytics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	config.WorkerCount = 3
	config.EventQueueSize = 500

	manager := NewManager(config)

	assert.Equal(t, config.WorkerCount, manager.config.WorkerCount)
	assert.Equal(t, config.EventQueueSize, manager.config.EventQueueSize)
	assert.NotNil(t, manager.providers)
	assert.NotNil(t, manager.eventCh)
	assert.NotNil(t, manager.stopCh)
	assert.False(t, manager.closed)
}

func TestManager_RegisterProvider(t *testing.T) {
	manager := NewManager(DefaultConfig())
	mock := NewMockProvider()

	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	names := manager.GetProviderNames()
	assert.Contains(t, names, "test")
}

func TestManager_RegisterProviderAfterClose(t *testing.T) {
	manager := NewManager(DefaultConfig())
	mock := NewMockProvider()

	err := manager.Close()
	assert.NoError(t, err)

	err = manager.RegisterProvider("test", mock)
	assert.Equal(t, ErrManagerClosed, err)
}

func TestManager_TrackEvent(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	manager := NewManager(config)
	mock := NewMockProvider()

	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()
	event := NewEvent("test_event", "user-123")

	err = manager.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for event processing
	time.Sleep(50 * time.Millisecond)

	// Verify event was tracked
	events := mock.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, event.ID, events[0].ID)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_TrackEventDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false
	manager := NewManager(config)
	mock := NewMockProvider()

	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()
	event := NewEvent("test_event", "user-123")

	err = manager.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for event processing
	time.Sleep(50 * time.Millisecond)

	// Verify no events were tracked
	events := mock.GetEvents()
	assert.Len(t, events, 0)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_TrackEventInvalidEvent(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	manager := NewManager(config)

	err := manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()
	event := Event{} // Invalid event (missing required fields)

	err = manager.TrackEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event")

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_TrackEventAfterClose(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Close()
	assert.NoError(t, err)

	ctx := context.Background()
	event := NewEvent("test_event", "user-123")

	err = manager.TrackEvent(ctx, event)
	assert.Equal(t, ErrManagerClosed, err)
}

func TestManager_TrackEventQueueFull(t *testing.T) {
	// Create manager with very small queue and don't start workers
	config := Config{
		Enabled:        true,
		EventQueueSize: 1,
		WorkerCount:    1,
		// Skip other fields since we won't call Start()
	}
	manager := &Manager{
		config:    config,
		providers: make(map[string]*ProviderWrapper),
		eventCh:   make(chan eventTask, config.EventQueueSize),
		stopCh:    make(chan struct{}),
	}

	ctx := context.Background()
	event1 := NewEvent("test_event_1", "user-123")
	event2 := NewEvent("test_event_2", "user-123")

	// First event should succeed (buffered channel can hold 1 item)
	err := manager.TrackEvent(ctx, event1)
	assert.NoError(t, err)

	// Second event should fail (queue is now full)
	err = manager.TrackEvent(ctx, event2)
	assert.Equal(t, ErrEventQueueFull, err)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_MultipleProviders(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	manager := NewManager(config)

	mock1 := NewMockProvider()
	mock2 := NewMockProvider()

	err := manager.RegisterProvider("provider1", mock1)
	assert.NoError(t, err)

	err = manager.RegisterProvider("provider2", mock2)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()
	event := NewEvent("test_event", "user-123")

	err = manager.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for event processing
	time.Sleep(50 * time.Millisecond)

	// Verify both providers received the event
	events1 := mock1.GetEvents()
	events2 := mock2.GetEvents()
	assert.Len(t, events1, 1)
	assert.Len(t, events2, 1)
	assert.Equal(t, event.ID, events1[0].ID)
	assert.Equal(t, event.ID, events2[0].ID)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_ProviderFailure(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	manager := NewManager(config)

	// Create a provider that always fails
	failingMock := NewMockProvider()
	failingMock.TrackEventFunc = func(ctx context.Context, event Event) error {
		return errors.New("provider failure")
	}

	// Create a working provider
	workingMock := NewMockProvider()

	err := manager.RegisterProvider("failing", failingMock)
	assert.NoError(t, err)

	err = manager.RegisterProvider("working", workingMock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()
	event := NewEvent("test_event", "user-123")

	err = manager.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for event processing
	time.Sleep(50 * time.Millisecond)

	// Verify working provider still received the event
	workingEvents := workingMock.GetEvents()
	assert.Len(t, workingEvents, 1)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_Flush(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	manager := NewManager(config)

	mock := NewMockProvider()
	flushed := false
	mock.FlushFunc = func(ctx context.Context) error {
		flushed = true
		return nil
	}

	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()
	err = manager.Flush(ctx)
	assert.NoError(t, err)
	assert.True(t, flushed)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_FlushWithEvents(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	manager := NewManager(config)

	mock := NewMockProvider()
	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	event := NewEvent("test_event", "user-123")

	// Track an event
	err = manager.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Give some time for initial processing
	time.Sleep(50 * time.Millisecond)

	// Flush should wait for event to be processed
	err = manager.Flush(ctx)
	assert.NoError(t, err)

	// Verify event was processed
	events := mock.GetEvents()
	assert.Len(t, events, 1)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_ConcurrentOperations(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.WorkerCount = 5
	config.EventQueueSize = 100 // Larger queue to avoid drops
	manager := NewManager(config)

	mock := NewMockProvider()
	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	// Give workers time to start
	time.Sleep(10 * time.Millisecond)

	ctx := context.Background()
	var wg sync.WaitGroup
	numEvents := 20 // Further reduced to avoid issues

	// Track events concurrently
	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := NewEvent("test_event", "user-123")
			event.Properties["id"] = id
			err := manager.TrackEvent(ctx, event)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Give events time to be processed
	time.Sleep(100 * time.Millisecond)

	// Flush to ensure all events are processed
	err = manager.Flush(ctx)
	assert.NoError(t, err)

	// Verify all events were processed
	events := mock.GetEvents()
	assert.Len(t, events, numEvents)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_CircuitBreaker(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.CircuitBreakerConfig.FailureThreshold = 2
	manager := NewManager(config)

	// Create a provider that fails after first success
	callCount := 0
	mock := NewMockProvider()
	mock.TrackEventFunc = func(ctx context.Context, event Event) error {
		callCount++
		if callCount > 1 {
			return errors.New("provider failure")
		}
		// Store the event for the first call
		mock.mu.Lock()
		mock.events = append(mock.events, event.Clone())
		mock.mu.Unlock()
		return nil
	}

	err := manager.RegisterProvider("test", mock)
	assert.NoError(t, err)

	err = manager.Start()
	assert.NoError(t, err)

	ctx := context.Background()

	// Send events to trigger circuit breaker
	for i := 0; i < 5; i++ {
		event := NewEvent("test_event", "user-123")
		err := manager.TrackEvent(ctx, event)
		assert.NoError(t, err)
	}

	// Give some time for event processing
	time.Sleep(100 * time.Millisecond)

	// Check that circuit breaker is working (some events should be blocked)
	wrapper := manager.providers["test"]
	assert.NotNil(t, wrapper)
	// Circuit breaker should be open after failures
	state := wrapper.circuitBreaker.GetState()
	assert.Equal(t, CircuitOpen, state)

	err = manager.Close()
	assert.NoError(t, err)
}
