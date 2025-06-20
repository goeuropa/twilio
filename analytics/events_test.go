package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {
	before := time.Now()
	event := NewEvent("test_event", "user-123")
	after := time.Now()

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "test_event", event.Name)
	assert.Equal(t, "user-123", event.UserID)
	assert.Equal(t, 1, event.Version)
	assert.NotNil(t, event.Properties)
	assert.Empty(t, event.SessionID)

	// Verify timestamp is reasonable
	assert.True(t, event.Timestamp.After(before.Add(-time.Second)))
	assert.True(t, event.Timestamp.Before(after.Add(time.Second)))
}

func TestNewEventWithSession(t *testing.T) {
	event := NewEventWithSession("test_event", "user-123", "session-456")

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "test_event", event.Name)
	assert.Equal(t, "user-123", event.UserID)
	assert.Equal(t, "session-456", event.SessionID)
	assert.Equal(t, 1, event.Version)
	assert.NotNil(t, event.Properties)
}

func TestHashPhoneNumber(t *testing.T) {
	tests := []struct {
		name        string
		phoneNumber string
		salt        string
		want        string
	}{
		{
			name:        "valid phone number",
			phoneNumber: "+12065551234",
			salt:        "test-salt",
			want:        "6e7c4e6f5a6b8d9f6e5c4b3a2e1f0d9c8b7a6e5d4c3b2a1f0e9d8c7b6a5e4d3c",
		},
		{
			name:        "empty phone number",
			phoneNumber: "",
			salt:        "test-salt",
			want:        "",
		},
		{
			name:        "different salt produces different hash",
			phoneNumber: "+12065551234",
			salt:        "different-salt",
			want:        "different-hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashPhoneNumber(tt.phoneNumber, tt.salt)

			if tt.phoneNumber == "" {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 64) // SHA256 produces 64 character hex string

				// Verify same input produces same output
				result2 := HashPhoneNumber(tt.phoneNumber, tt.salt)
				assert.Equal(t, result, result2)

				// Verify different salt produces different hash
				if tt.name == "different salt produces different hash" {
					result3 := HashPhoneNumber(tt.phoneNumber, "test-salt")
					assert.NotEqual(t, result, result3)
				}
			}
		})
	}
}

func TestSMSRequestEvent(t *testing.T) {
	event := SMSRequestEvent("user-hash", "es-US", "75403")

	assert.Equal(t, EventSMSRequest, event.Name)
	assert.Equal(t, "user-hash", event.UserID)
	assert.Equal(t, "es-US", event.Properties[PropLanguage])
	assert.Equal(t, "75403", event.Properties[PropQuery])
}

func TestVoiceRequestEvent(t *testing.T) {
	event := VoiceRequestEvent("user-hash", "fr-US")

	assert.Equal(t, EventVoiceRequest, event.Name)
	assert.Equal(t, "user-hash", event.UserID)
	assert.Equal(t, "fr-US", event.Properties[PropLanguage])
}

func TestStopLookupEvent(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		wantEvent string
	}{
		{
			name:      "successful lookup",
			success:   true,
			wantEvent: EventStopLookupSuccess,
		},
		{
			name:      "failed lookup",
			success:   false,
			wantEvent: EventStopLookupFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := StopLookupEvent("user-hash", "1_75403", "1", tt.success, 150)

			assert.Equal(t, tt.wantEvent, event.Name)
			assert.Equal(t, "user-hash", event.UserID)
			assert.Equal(t, "1_75403", event.Properties[PropStopID])
			assert.Equal(t, "1", event.Properties[PropAgencyID])
			assert.Equal(t, int64(150), event.Properties[PropLatencyMS])
		})
	}
}

func TestErrorEvent(t *testing.T) {
	event := ErrorEvent("user-hash", "api_error", "connection timeout")

	assert.Equal(t, EventErrorOccurred, event.Name)
	assert.Equal(t, "user-hash", event.UserID)
	assert.Equal(t, "api_error", event.Properties[PropErrorType])
	assert.Equal(t, "connection timeout", event.Properties[PropErrorMessage])
}

func TestDisambiguationPresentedEvent(t *testing.T) {
	event := DisambiguationPresentedEvent("user-hash", "session-123", 3)

	assert.Equal(t, EventSMSDisambiguationPresent, event.Name)
	assert.Equal(t, "user-hash", event.UserID)
	assert.Equal(t, "session-123", event.SessionID)
	assert.Equal(t, 3, event.Properties[PropChoiceCount])
}

func TestDisambiguationSelectedEvent(t *testing.T) {
	event := DisambiguationSelectedEvent("user-hash", "session-123", 2, "1_75403")

	assert.Equal(t, EventSMSDisambiguationSelect, event.Name)
	assert.Equal(t, "user-hash", event.UserID)
	assert.Equal(t, "session-123", event.SessionID)
	assert.Equal(t, 2, event.Properties[PropChoiceIndex])
	assert.Equal(t, "1_75403", event.Properties[PropStopID])
}
