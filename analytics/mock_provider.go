package analytics

import (
	"context"
	"sync"
)

// MockProvider is a mock implementation of the Analytics interface for testing.
type MockProvider struct {
	mu     sync.Mutex
	events []Event
	closed bool

	// Control behavior for testing
	TrackEventFunc func(ctx context.Context, event Event) error
	FlushFunc      func(ctx context.Context) error
	CloseFunc      func() error
}

// NewMockProvider creates a new mock analytics provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		events: make([]Event, 0),
	}
}

// TrackEvent implements the Analytics interface.
func (m *MockProvider) TrackEvent(ctx context.Context, event Event) error {
	if m.TrackEventFunc != nil {
		return m.TrackEventFunc(ctx, event)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrProviderClosed
	}

	m.events = append(m.events, event.Clone())
	return nil
}

// Flush implements the Analytics interface.
func (m *MockProvider) Flush(ctx context.Context) error {
	if m.FlushFunc != nil {
		return m.FlushFunc(ctx)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrProviderClosed
	}

	return nil
}

// Close implements the Analytics interface.
func (m *MockProvider) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrProviderClosed
	}

	m.closed = true
	return nil
}

// GetEvents returns all tracked events (for testing).
func (m *MockProvider) GetEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	events := make([]Event, len(m.events))
	copy(events, m.events)
	return events
}

// GetEventCount returns the number of tracked events (for testing).
func (m *MockProvider) GetEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.events)
}

// Clear removes all tracked events (for testing).
func (m *MockProvider) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = make([]Event, 0)
}

// IsClosed returns whether the provider has been closed (for testing).
func (m *MockProvider) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closed
}
