package localization

// LanguageConfig represents the language configuration for the application
type LanguageConfig struct {
	Primary   string   `json:"primary"`
	Supported []string `json:"supported"`
}

// VoiceStartContext extended template context for voice start with language
type VoiceStartContext struct {
	WelcomePrompt string
	Language      string `json:"language,omitempty"`
}

// VoiceFindStopContext extended template context for voice find stop with language
type VoiceFindStopContext struct {
	ArrivalsMessage string
	MinutesAfter    int
	Language        string `json:"language,omitempty"`
}
