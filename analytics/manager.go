package analytics

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager orchestrates multiple analytics providers with worker pool pattern.
type Manager struct {
	config    Config
	providers map[string]*ProviderWrapper
	eventCh   chan eventTask
	workers   sync.WaitGroup
	stopCh    chan struct{}
	closed    bool
	mu        sync.RWMutex
	once      sync.Once
}

// ProviderWrapper wraps an analytics provider with circuit breaker and error handling.
type ProviderWrapper struct {
	provider       Analytics
	circuitBreaker *CircuitBreaker
	name           string
}

// eventTask represents a task to process an event.
type eventTask struct {
	event Event
	ctx   context.Context
}

// NewManager creates a new analytics manager with the given configuration.
func NewManager(config Config) *Manager {
	if err := config.Validate(); err != nil {
		log.Printf("Analytics config validation failed: %v", err)
		config = DefaultConfig()
	}

	return &Manager{
		config:    config,
		providers: make(map[string]*ProviderWrapper),
		eventCh:   make(chan eventTask, config.EventQueueSize),
		stopCh:    make(chan struct{}),
	}
}

// Start initializes the manager and starts worker goroutines.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrManagerClosed
	}

	m.once.Do(func() {
		// Start worker goroutines
		for i := 0; i < m.config.WorkerCount; i++ {
			m.workers.Add(1)
			go m.worker()
		}
	})

	return nil
}

// RegisterProvider adds a new analytics provider to the manager.
func (m *Manager) RegisterProvider(name string, provider Analytics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrManagerClosed
	}

	wrapper := &ProviderWrapper{
		provider:       provider,
		circuitBreaker: NewCircuitBreaker(m.config.CircuitBreakerConfig),
		name:           name,
	}

	m.providers[name] = wrapper
	return nil
}

// TrackEvent queues an event for processing by all registered providers.
func (m *Manager) TrackEvent(ctx context.Context, event Event) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrManagerClosed
	}

	if !m.config.Enabled {
		return nil
	}

	// Validate event
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Queue event for processing
	select {
	case m.eventCh <- eventTask{event: event.Clone(), ctx: ctx}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full, drop the event (non-blocking)
		return ErrEventQueueFull
	}
}

// Flush waits for all queued events to be processed and flushes all providers.
func (m *Manager) Flush(ctx context.Context) error {
	m.mu.RLock()
	providers := make([]*ProviderWrapper, 0, len(m.providers))
	for _, wrapper := range m.providers {
		providers = append(providers, wrapper)
	}
	m.mu.RUnlock()

	// Wait for queue to empty
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if len(m.eventCh) == 0 {
				goto flushProviders
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

flushProviders:
	// Flush all providers
	var wg sync.WaitGroup
	errCh := make(chan error, len(providers))

	for _, wrapper := range providers {
		wg.Add(1)
		go func(w *ProviderWrapper) {
			defer wg.Done()
			if err := w.provider.Flush(ctx); err != nil {
				errCh <- fmt.Errorf("flush failed for provider %s: %w", w.name, err)
			}
		}(wrapper)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("flush errors: %v", errors)
	}

	return nil
}

// Close gracefully shuts down the manager and all providers.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrManagerClosed
	}

	m.closed = true

	// Signal workers to stop
	close(m.stopCh)

	// Wait for workers to finish
	m.workers.Wait()

	// Close event channel
	close(m.eventCh)

	// Close all providers
	var errors []error
	for name, wrapper := range m.providers {
		if err := wrapper.provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("close failed for provider %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("close errors: %v", errors)
	}

	return nil
}

// GetProviderNames returns the names of all registered providers.
func (m *Manager) GetProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// worker processes events from the event queue.
func (m *Manager) worker() {
	defer m.workers.Done()

	for {
		select {
		case task, ok := <-m.eventCh:
			if !ok {
				return
			}
			m.processEvent(task)
		case <-m.stopCh:
			return
		}
	}
}

// processEvent sends an event to all registered providers.
func (m *Manager) processEvent(task eventTask) {
	m.mu.RLock()
	providers := make([]*ProviderWrapper, 0, len(m.providers))
	for _, wrapper := range m.providers {
		providers = append(providers, wrapper)
	}
	m.mu.RUnlock()

	// Send event to all providers in parallel
	var wg sync.WaitGroup
	for _, wrapper := range providers {
		wg.Add(1)
		go func(w *ProviderWrapper) {
			defer wg.Done()
			m.sendToProvider(w, task)
		}(wrapper)
	}
	wg.Wait()
}

// sendToProvider sends an event to a specific provider with circuit breaker protection.
func (m *Manager) sendToProvider(wrapper *ProviderWrapper, task eventTask) {
	err := wrapper.circuitBreaker.Call(func() error {
		return wrapper.provider.TrackEvent(task.ctx, task.event)
	})

	if err != nil {
		// Log error but don't fail the overall operation
		log.Printf("Analytics provider %s failed to track event: %v", wrapper.name, err)
	}
}
