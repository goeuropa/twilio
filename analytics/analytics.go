// Package analytics provides interfaces and types for tracking user interactions
// and system events in the OneBusAway Twilio integration.
package analytics

import (
	"context"
	"time"
)

// Analytics defines the interface that all analytics providers must implement.
// It provides methods for tracking events, flushing pending data, and cleanup.
type Analytics interface {
	// TrackEvent sends an event to the analytics provider.
	// The implementation must be non-blocking and handle errors gracefully.
	TrackEvent(ctx context.Context, event Event) error

	// Flush sends any buffered events to the analytics provider.
	// This should be called during graceful shutdown.
	Flush(ctx context.Context) error

	// Close cleans up any resources used by the provider.
	Close() error
}

// Event represents a generic analytics event with common fields
// that all events share regardless of type.
type Event struct {
	// ID is a unique identifier for deduplication (typically a UUID)
	ID string `json:"id"`

	// Name is the event type (e.g., "sms_request", "voice_request")
	Name string `json:"name"`

	// Properties contains event-specific data
	Properties map[string]interface{} `json:"props"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Version is the schema version for this event
	Version int `json:"version"`

	// UserID is a hashed identifier for the user (e.g., hashed phone number)
	UserID string `json:"user_id,omitempty"`

	// SessionID identifies a multi-step interaction session
	SessionID string `json:"session_id,omitempty"`
}

// Validate checks that the event has all required fields.
func (e *Event) Validate() error {
	if e.ID == "" {
		return ErrMissingEventID
	}
	if e.Name == "" {
		return ErrMissingEventName
	}
	if e.Timestamp.IsZero() {
		return ErrMissingTimestamp
	}
	return nil
}

// Clone creates a deep copy of the event to prevent data races.
func (e *Event) Clone() Event {
	clone := *e
	if e.Properties != nil {
		clone.Properties = make(map[string]interface{}, len(e.Properties))
		for k, v := range e.Properties {
			clone.Properties[k] = v
		}
	}
	return clone
}
