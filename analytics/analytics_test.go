package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr error
	}{
		{
			name: "valid event",
			event: Event{
				ID:        "test-123",
				Name:      "test_event",
				Timestamp: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "missing ID",
			event: Event{
				Name:      "test_event",
				Timestamp: time.Now(),
			},
			wantErr: ErrMissingEventID,
		},
		{
			name: "missing name",
			event: Event{
				ID:        "test-123",
				Timestamp: time.Now(),
			},
			wantErr: ErrMissingEventName,
		},
		{
			name: "missing timestamp",
			event: Event{
				ID:   "test-123",
				Name: "test_event",
			},
			wantErr: ErrMissingTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvent_Clone(t *testing.T) {
	original := Event{
		ID:        "test-123",
		Name:      "test_event",
		Timestamp: time.Now(),
		Version:   1,
		UserID:    "user-123",
		SessionID: "session-123",
		Properties: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
	}

	clone := original.Clone()

	// Verify all fields are copied
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Timestamp, clone.Timestamp)
	assert.Equal(t, original.Version, clone.Version)
	assert.Equal(t, original.UserID, clone.UserID)
	assert.Equal(t, original.SessionID, clone.SessionID)
	assert.Equal(t, original.Properties, clone.Properties)

	// Verify deep copy of properties
	clone.Properties["key1"] = "modified"
	assert.Equal(t, "value1", original.Properties["key1"])
	assert.Equal(t, "modified", clone.Properties["key1"])
}

func TestEvent_CloneNilProperties(t *testing.T) {
	original := Event{
		ID:         "test-123",
		Name:       "test_event",
		Timestamp:  time.Now(),
		Properties: nil,
	}

	clone := original.Clone()
	assert.Nil(t, clone.Properties)
}
