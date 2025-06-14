package formatters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"oba-twilio/models"
)

func TestFormatSMSResponse(t *testing.T) {
	arrivals := []models.Arrival{
		{
			RouteShortName:      "8",
			TripHeadsign:        "Seattle Center",
			MinutesUntilArrival: 3,
		},
		{
			RouteShortName:      "43",
			TripHeadsign:        "Capitol Hill",
			MinutesUntilArrival: 8,
		},
		{
			RouteShortName:      "49",
			TripHeadsign:        "U District",
			MinutesUntilArrival: 12,
		},
		{
			RouteShortName:      "50",
			TripHeadsign:        "Fremont",
			MinutesUntilArrival: 15,
		},
	}

	result := FormatSMSResponse(arrivals, "Pine St & 3rd Ave")

	assert.Contains(t, result, "Pine St & 3rd Ave")
	assert.Contains(t, result, "Route 8 to Seattle Center: 3 min")
	assert.Contains(t, result, "Route 43 to Capitol Hill: 8 min")
	assert.Contains(t, result, "Route 49 to U District: 12 min")
	assert.NotContains(t, result, "Route 50")

	lines := strings.Split(result, "\n")
	assert.Equal(t, 4, len(lines))
}

func TestFormatSMSResponse_Empty(t *testing.T) {
	result := FormatSMSResponse([]models.Arrival{}, "Test Stop")
	assert.Equal(t, "No upcoming arrivals found for this stop.", result)
}

func TestFormatVoiceResponse(t *testing.T) {
	arrivals := []models.Arrival{
		{
			RouteShortName:      "8",
			TripHeadsign:        "Seattle Center",
			MinutesUntilArrival: 3,
		},
		{
			RouteShortName:      "43",
			TripHeadsign:        "Capitol Hill",
			MinutesUntilArrival: 1,
		},
	}

	result := FormatVoiceResponse(arrivals, "Pine St & 3rd Ave")

	assert.Contains(t, result, "Arrivals for Pine St & 3rd Ave")
	assert.Contains(t, result, "Route 8 to Seattle Center in 3 minutes")
	assert.Contains(t, result, "Route 43 to Capitol Hill in 1 minute")
}

func TestGenerateTwiMLSMS(t *testing.T) {
	result, err := GenerateTwiMLSMS("Test message")

	assert.NoError(t, err)
	assert.Contains(t, result, "<?xml version=\"1.0\"")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "<Message>Test message</Message>")
	assert.Contains(t, result, "</Response>")
}



func TestExtractStopID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid stop ID", "75403", "75403"},
		{"Valid with spaces", " 75403 ", "75403"},
		{"Valid short ID", "123", "123"},
		{"Valid long ID", "1234567890", "1234567890"},
		{"Invalid too short", "12", ""},
		{"Invalid too long", "12345678901", ""},
		{"Invalid with letters", "75403a", ""},
		{"Invalid with text", "stop 75403", ""},
		{"Empty string", "", ""},
		{"Only spaces", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractStopID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatArrivalTime(t *testing.T) {
	tests := []struct {
		minutes  int
		expected string
	}{
		{0, "Now"},
		{-1, "Now"},
		{1, "1 min"},
		{5, "5 min"},
		{30, "30 min"},
	}

	for _, tt := range tests {
		result := formatArrivalTime(tt.minutes)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFormatArrivalTimeVoice(t *testing.T) {
	tests := []struct {
		minutes  int
		expected string
	}{
		{0, "arriving now"},
		{-1, "arriving now"},
		{1, "in 1 minute"},
		{5, "in 5 minutes"},
		{30, "in 30 minutes"},
	}

	for _, tt := range tests {
		result := formatArrivalTimeVoice(tt.minutes)
		assert.Equal(t, tt.expected, result)
	}
}

func TestIsDisambiguationChoice(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"2", 2},
		{"9", 9},
		{"10", 10},
		{"99", 99},
		{"0", 0},
		{"100", 0},
		{"a", 0},
		{"1a", 0},
		{" 1 ", 1},
		{" 10 ", 10},
		{"", 0},
		{"01", 1},                 // Leading zero should still work
		{"999999999999999999", 0}, // Overflow protection
	}

	for _, tt := range tests {
		result := IsDisambiguationChoice(tt.input)
		assert.Equal(t, tt.expected, result, "Input: %s", tt.input)
	}
}

func TestFormatDisambiguationMessage(t *testing.T) {
	tests := []struct {
		name             string
		stopOptions      []models.StopOption
		originalID       string
		expectedContains []string
	}{
		{
			name: "Multiple options",
			stopOptions: []models.StopOption{
				{
					FullStopID:  "1_12345",
					DisplayText: "King County Metro: Pine St & 3rd Ave",
				},
				{
					FullStopID:  "40_12345",
					DisplayText: "Sound Transit: University Street Station",
				},
			},
			originalID: "12345",
			expectedContains: []string{
				"Multiple stops found for 12345",
				"1) King County Metro: Pine St & 3rd Ave",
				"2) Sound Transit: University Street Station",
				"Reply with the number to choose",
			},
		},
		{
			name:        "No options",
			stopOptions: []models.StopOption{},
			originalID:  "99999",
			expectedContains: []string{
				"No stops found for ID 99999",
			},
		},
		{
			name: "Single option",
			stopOptions: []models.StopOption{
				{
					FullStopID:  "1_12345",
					DisplayText: "King County Metro: Pine St & 3rd Ave",
				},
			},
			originalID:       "12345",
			expectedContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDisambiguationMessage(tt.stopOptions, tt.originalID)

			if len(tt.stopOptions) == 1 {
				assert.Empty(t, result)
			} else {
				for _, expected := range tt.expectedContains {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}
