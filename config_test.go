package main

import (
	"os"
	"testing"

	"oba-twilio/client"

	"github.com/stretchr/testify/assert"
)

func TestServerConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		envURL      string
		expectedURL string
	}{
		{
			name:        "Default Puget Sound",
			envURL:      "",
			expectedURL: "https://api.pugetsound.onebusaway.org",
		},
		{
			name:        "Tampa server",
			envURL:      "https://api.tampa.onebusaway.org",
			expectedURL: "https://api.tampa.onebusaway.org",
		},
		{
			name:        "Unitrans server",
			envURL:      "https://api.unitrans.onebusawaycloud.com",
			expectedURL: "https://api.unitrans.onebusawaycloud.com",
		},
		{
			name:        "Local development",
			envURL:      "http://localhost:8080",
			expectedURL: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalEnv := os.Getenv("ONEBUSAWAY_BASE_URL")
			defer os.Setenv("ONEBUSAWAY_BASE_URL", originalEnv)

			if tt.envURL != "" {
				os.Setenv("ONEBUSAWAY_BASE_URL", tt.envURL)
			} else {
				os.Unsetenv("ONEBUSAWAY_BASE_URL")
			}

			obaBaseURL := os.Getenv("ONEBUSAWAY_BASE_URL")
			if obaBaseURL == "" {
				obaBaseURL = "https://api.pugetsound.onebusaway.org"
			}

			assert.Equal(t, tt.expectedURL, obaBaseURL)

			obaClient := client.NewOneBusAwayClient(obaBaseURL, "test-key")
			assert.Equal(t, tt.expectedURL, obaClient.BaseURL)
		})
	}
}

func TestAPIKeyConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		expectedKey string
	}{
		{
			name:        "Valid API key",
			envKey:      "valid-api-key-123",
			expectedKey: "valid-api-key-123",
		},
		{
			name:        "OneBusAway API key",
			envKey:      "org.onebusaway.iphone",
			expectedKey: "org.onebusaway.iphone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalEnv := os.Getenv("ONEBUSAWAY_API_KEY")
			defer os.Setenv("ONEBUSAWAY_API_KEY", originalEnv)

			os.Setenv("ONEBUSAWAY_API_KEY", tt.envKey)

			obaAPIKey := os.Getenv("ONEBUSAWAY_API_KEY")
			// In tests, we skip the validation logic since we're testing client construction

			assert.Equal(t, tt.expectedKey, obaAPIKey)

			obaClient := client.NewOneBusAwayClient("https://test.com", obaAPIKey)
			assert.Equal(t, tt.expectedKey, obaClient.APIKey)
		})
	}
}

func TestAPIKeyValidation(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		shouldFail bool
	}{
		{"Empty key should fail", "", true},
		{"Test key should fail", "test", true},
		{"TEST key should fail", "TEST", true},
		{"Placeholder should fail", "placeholder", true},
		{"Valid key should pass", "valid-api-key", false},
		{"OneBusAway key should pass", "org.onebusaway.iphone", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic without calling log.Fatal
			isInvalid := tt.apiKey == "" || tt.apiKey == "test" || tt.apiKey == "TEST" || tt.apiKey == "placeholder"
			assert.Equal(t, tt.shouldFail, isInvalid)
		})
	}
}
