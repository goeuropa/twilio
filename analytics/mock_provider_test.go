package analytics

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockProvider_TrackEvent(t *testing.T) {
	ctx := context.Background()
	mock := NewMockProvider()

	event1 := NewEvent("test_event_1", "user-1")
	event2 := NewEvent("test_event_2", "user-2")

	// Track events
	err := mock.TrackEvent(ctx, event1)
	assert.NoError(t, err)

	err = mock.TrackEvent(ctx, event2)
	assert.NoError(t, err)

	// Verify events were tracked
	events := mock.GetEvents()
	assert.Len(t, events, 2)
	assert.Equal(t, event1.ID, events[0].ID)
	assert.Equal(t, event2.ID, events[1].ID)
	assert.Equal(t, 2, mock.GetEventCount())
}

func TestMockProvider_Clear(t *testing.T) {
	ctx := context.Background()
	mock := NewMockProvider()

	// Track some events
	err := mock.TrackEvent(ctx, NewEvent("test_event", "user-1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.GetEventCount())

	// Clear events
	mock.Clear()
	assert.Equal(t, 0, mock.GetEventCount())
	assert.Empty(t, mock.GetEvents())
}

func TestMockProvider_Close(t *testing.T) {
	ctx := context.Background()
	mock := NewMockProvider()

	// Close the provider
	err := mock.Close()
	assert.NoError(t, err)
	assert.True(t, mock.IsClosed())

	// Subsequent operations should fail
	err = mock.TrackEvent(ctx, NewEvent("test_event", "user-1"))
	assert.Equal(t, ErrProviderClosed, err)

	err = mock.Flush(ctx)
	assert.Equal(t, ErrProviderClosed, err)

	err = mock.Close()
	assert.Equal(t, ErrProviderClosed, err)
}

func TestMockProvider_CustomFunctions(t *testing.T) {
	ctx := context.Background()
	mock := NewMockProvider()

	// Test custom TrackEventFunc
	customErr := errors.New("custom track error")
	mock.TrackEventFunc = func(ctx context.Context, event Event) error {
		return customErr
	}

	err := mock.TrackEvent(ctx, NewEvent("test_event", "user-1"))
	assert.Equal(t, customErr, err)
	assert.Equal(t, 0, mock.GetEventCount()) // Event should not be tracked

	// Test custom FlushFunc
	flushErr := errors.New("custom flush error")
	mock.FlushFunc = func(ctx context.Context) error {
		return flushErr
	}

	err = mock.Flush(ctx)
	assert.Equal(t, flushErr, err)

	// Test custom CloseFunc
	closeErr := errors.New("custom close error")
	mock.CloseFunc = func() error {
		return closeErr
	}

	err = mock.Close()
	assert.Equal(t, closeErr, err)
	assert.False(t, mock.IsClosed()) // Close should not complete
}

func TestMockProvider_EventCloning(t *testing.T) {
	ctx := context.Background()
	mock := NewMockProvider()

	// Create event with properties
	event := NewEvent("test_event", "user-1")
	event.Properties["key"] = "original"

	// Track the event
	err := mock.TrackEvent(ctx, event)
	assert.NoError(t, err)

	// Modify the original event
	event.Properties["key"] = "modified"

	// Verify the tracked event was cloned (not affected by modification)
	trackedEvents := mock.GetEvents()
	assert.Len(t, trackedEvents, 1)
	assert.Equal(t, "original", trackedEvents[0].Properties["key"])
}
