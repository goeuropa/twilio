package localization

import (
	"sync"
)

// NewTestManagerWithStrings creates a localization manager with custom strings and languages
func NewTestManagerWithStrings(strings map[string]map[string]string, supportedLanguages []string) *LocalizationManager {
	return &LocalizationManager{
		strings:            strings,
		mu:                 sync.RWMutex{},
		defaultLanguage:    supportedLanguages[0],
		supportedLanguages: supportedLanguages,
	}
}

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
				"sms.menu.more_hint":              "Reply 'more' for later buses",
				"sms.menu.help_hint":              "Reply 'help' for usage info",
				"sms.error.invalid_stop":          "Please send a valid stop ID (e.g., 75403).",
				"sms.help":                        "Send a stop ID (e.g., 75403) to get bus arrivals. Reply 'more' for later buses, 'help' for this message.",
				"sms.keyword.more":                "Showing buses in the next %d minutes:",
				"sms.keyword.invalid":             "I don't understand '%s'. Send a stop ID or reply 'help' for usage info.",
				"sms.session.expired":             "Session expired. Please send a stop ID to get started.",
				"sms.language.switched":           "Language switched to English",
				"sms.menu.new_hint":               "Reply 'new' for a different stop",
				"sms.error.search_failed":         "Sorry, I couldn't search for stop %s. Please check the stop ID and try again.",
			},
		},
		mu:                 sync.RWMutex{},
		defaultLanguage:    "en-US",
		supportedLanguages: []string{"en-US"},
	}
}
