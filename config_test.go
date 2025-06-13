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
			name:        "Default API key",
			envKey:      "",
			expectedKey: "test",
		},
		{
			name:        "Custom API key",
			envKey:      "custom-test-key",
			expectedKey: "custom-test-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalEnv := os.Getenv("ONEBUSAWAY_API_KEY")
			defer os.Setenv("ONEBUSAWAY_API_KEY", originalEnv)

			if tt.envKey != "" {
				os.Setenv("ONEBUSAWAY_API_KEY", tt.envKey)
			} else {
				os.Unsetenv("ONEBUSAWAY_API_KEY")
			}

			obaAPIKey := os.Getenv("ONEBUSAWAY_API_KEY")
			if obaAPIKey == "" {
				obaAPIKey = "test"
			}

			assert.Equal(t, tt.expectedKey, obaAPIKey)

			obaClient := client.NewOneBusAwayClient("https://test.com", obaAPIKey)
			assert.Equal(t, tt.expectedKey, obaClient.APIKey)
		})
	}
}
