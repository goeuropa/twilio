// Package plausible provides a Plausible Analytics provider implementation
// for the analytics system.
package plausible

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"oba-twilio/analytics"
)

// Provider implements the Analytics interface for Plausible Analytics.
type Provider struct {
	config      Config
	client      *http.Client
	eventBatch  []analytics.Event
	batchMutex  sync.Mutex
	flushTicker *time.Ticker
	stopCh      chan struct{}
	closed      bool
	closedMutex sync.RWMutex
	wg          sync.WaitGroup
}

// Config holds configuration for the Plausible provider.
type Config struct {
	// Domain is the domain configured in Plausible (required)
	Domain string

	// APIURL is the Plausible API endpoint (default: https://plausible.io)
	APIURL string

	// APIKey is the API key for authentication (optional for public events)
	APIKey string

	// BatchSize is the number of events to batch before sending (default: 100)
	BatchSize int

	// FlushInterval is how often to flush events (default: 10s)
	FlushInterval time.Duration

	// HTTPTimeout for requests (default: 30s)
	HTTPTimeout time.Duration

	// MaxRetries for failed requests (default: 3)
	MaxRetries int

	// RetryDelay for failed requests (default: 1s)
	RetryDelay time.Duration
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		APIURL:        "https://plausible.io",
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
		HTTPTimeout:   30 * time.Second,
		MaxRetries:    3,
		RetryDelay:    time.Second,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Domain == "" {
		return analytics.ErrMissingDomain
	}
	if c.APIURL == "" {
		c.APIURL = "https://plausible.io"
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = 10 * time.Second
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 30 * time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = time.Second
	}
	return nil
}

// NewProvider creates a new Plausible analytics provider.
func NewProvider(config Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plausible config: %w", err)
	}

	provider := &Provider{
		config: config,
		client: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		eventBatch: make([]analytics.Event, 0, config.BatchSize),
		stopCh:     make(chan struct{}),
	}

	// Start periodic flush if interval is set
	if config.FlushInterval > 0 {
		provider.flushTicker = time.NewTicker(config.FlushInterval)
		provider.wg.Add(1)
		go provider.flushWorker()
	}

	return provider, nil
}

// TrackEvent implements the Analytics interface.
func (p *Provider) TrackEvent(ctx context.Context, event analytics.Event) error {
	p.closedMutex.RLock()
	if p.closed {
		p.closedMutex.RUnlock()
		return analytics.ErrProviderClosed
	}
	p.closedMutex.RUnlock()

	// Validate event
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event for plausible: %w", err)
	}

	p.batchMutex.Lock()
	defer p.batchMutex.Unlock()

	// Add event to batch
	p.eventBatch = append(p.eventBatch, event.Clone())

	// Flush if batch is full
	if len(p.eventBatch) >= p.config.BatchSize {
		return p.flushBatchLocked(ctx)
	}

	return nil
}

// Flush implements the Analytics interface.
func (p *Provider) Flush(ctx context.Context) error {
	p.closedMutex.RLock()
	if p.closed {
		p.closedMutex.RUnlock()
		return analytics.ErrProviderClosed
	}
	p.closedMutex.RUnlock()

	p.batchMutex.Lock()
	defer p.batchMutex.Unlock()

	return p.flushBatchLocked(ctx)
}

// Close implements the Analytics interface.
func (p *Provider) Close() error {
	p.closedMutex.Lock()
	defer p.closedMutex.Unlock()

	if p.closed {
		return analytics.ErrProviderClosed
	}

	p.closed = true

	// Stop flush ticker
	if p.flushTicker != nil {
		p.flushTicker.Stop()
	}

	// Signal flush worker to stop
	close(p.stopCh)

	// Wait for flush worker to finish
	p.wg.Wait()

	// Flush any remaining events
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.batchMutex.Lock()
	err := p.flushBatchLocked(ctx)
	p.batchMutex.Unlock()

	return err
}

// flushWorker periodically flushes events.
func (p *Provider) flushWorker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.flushTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), p.config.HTTPTimeout)
			p.batchMutex.Lock()
			err := p.flushBatchLocked(ctx)
			p.batchMutex.Unlock()
			cancel()

			if err != nil {
				// Log error but continue
				// In a real implementation, you might want to use a proper logger
				fmt.Printf("Plausible flush error: %v\n", err)
			}

		case <-p.stopCh:
			return
		}
	}
}

// flushBatchLocked sends batched events to Plausible (caller must hold batchMutex).
func (p *Provider) flushBatchLocked(ctx context.Context) error {
	if len(p.eventBatch) == 0 {
		return nil
	}

	// Convert events to Plausible format
	plausibleEvents := make([]PlausibleEvent, 0, len(p.eventBatch))
	for _, event := range p.eventBatch {
		plausibleEvent := p.convertEvent(event)
		plausibleEvents = append(plausibleEvents, plausibleEvent)
	}

	// Send events
	err := p.sendEvents(ctx, plausibleEvents)
	if err != nil {
		return fmt.Errorf("failed to send events to plausible: %w", err)
	}

	// Clear batch on success
	p.eventBatch = p.eventBatch[:0]
	return nil
}

// convertEvent converts an analytics.Event to PlausibleEvent format.
func (p *Provider) convertEvent(event analytics.Event) PlausibleEvent {
	plausibleEvent := PlausibleEvent{
		Name:      event.Name,
		Domain:    p.config.Domain,
		URL:       fmt.Sprintf("https://%s/analytics-event", p.config.Domain),
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Props:     make(map[string]interface{}),
	}

	// Copy properties, filtering out sensitive data
	for key, value := range event.Properties {
		// Skip potentially sensitive properties
		if key == "phone_number" || key == "message_content" {
			continue
		}
		plausibleEvent.Props[key] = value
	}

	// Add analytics-specific metadata
	if event.UserID != "" {
		plausibleEvent.Props["user_id"] = event.UserID
	}
	if event.SessionID != "" {
		plausibleEvent.Props["session_id"] = event.SessionID
	}
	if event.Version > 0 {
		plausibleEvent.Props["event_version"] = event.Version
	}

	return plausibleEvent
}

// sendEvents sends events to Plausible with retry logic.
func (p *Provider) sendEvents(ctx context.Context, events []PlausibleEvent) error {
	if len(events) == 0 {
		return nil
	}

	payload, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	url := fmt.Sprintf("%s/api/event", p.config.APIURL)

	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff with jitter
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * p.config.RetryDelay
			jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
			delay += jitter

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := p.sendRequest(ctx, url, payload)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on client errors (4xx)
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			break
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// sendRequest sends a single HTTP request to Plausible.
func (p *Provider) sendRequest(ctx context.Context, url string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OneBusAway-Twilio/1.0")

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	return nil
}

// PlausibleEvent represents an event in Plausible's format.
type PlausibleEvent struct {
	Name      string                 `json:"name"`
	Domain    string                 `json:"domain"`
	URL       string                 `json:"url"`
	Timestamp string                 `json:"timestamp,omitempty"`
	Props     map[string]interface{} `json:"props,omitempty"`
}

// HTTPError represents an HTTP error response.
type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Status)
}
