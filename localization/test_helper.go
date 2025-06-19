package localization

import (
	"sync"
)

// NewTestManager creates a basic localization manager for testing
func NewTestManager() *LocalizationManager {
	return &LocalizationManager{
		strings: map[string]map[string]string{
			"en-US": {
				"voice.welcome":                   "Welcome to OneBusAway transit information. Please enter your stop ID followed by the pound key.",
				"voice.error.invalid_request":     "Invalid request format.",
				"voice.error.no_stops_found":      "Sorry, I couldn't find any stops with that ID. Please check the stop ID and try again.",
				"voice.error.template_failed":     "Error generating response.",
				"voice.error.service_unavailable": "OneBusAway service is temporarily unavailable. Please try again in a moment.",
				"voice.menu.more_departures":      "To hear more departures, press 1.",
				"voice.menu.main_menu":            "To go back to the main menu, press 2.",
				"voice.language.spanish_option":   "Para continuar en Español, presione asterisco.",
				"voice.language.choose_language":  "To choose another language, press asterisk.",
				"sms.no_stops_found":              "Sorry, no stops found with ID %s. Please check and try again.",
				"sms.no_arrivals":                 "No upcoming arrivals found for this stop.",
				"sms.service_unavailable":         "OneBusAway service is temporarily unavailable. Please try again.",
			},
		},
		mu:                 sync.RWMutex{},
		defaultLanguage:    "en-US",
		supportedLanguages: []string{"en-US"},
	}
}
