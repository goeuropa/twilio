package formatters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVoiceTemplateManager(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.startTemplate)
	assert.NotNil(t, manager.findStopTemplate)
	assert.NotNil(t, manager.errorTemplate)
	assert.NotNil(t, manager.disambiguationTemplate)
}

func TestRenderVoiceStart(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceStartContext{
		WelcomePrompt: "Welcome to OneBusAway. Please enter your stop ID.",
	}

	result, err := manager.RenderVoiceStart(ctx)
	require.NoError(t, err)

	// Check XML structure
	assert.Contains(t, result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "</Response>")
	assert.Contains(t, result, "<Gather")
	assert.Contains(t, result, "numDigits=\"6\"")
	assert.Contains(t, result, "action=\"/voice/find_stop\"")
	assert.Contains(t, result, "method=\"POST\"")
	assert.Contains(t, result, "<Say>Welcome to OneBusAway. Please enter your stop ID.</Say>")
}

func TestRenderVoiceFindStop(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceFindStopContext{
		ArrivalsMessage: "Route 8 to Seattle Center in 3 minutes.",
		MinutesAfter:    0,
	}

	result, err := manager.RenderVoiceFindStop(ctx)
	require.NoError(t, err)

	// Check XML structure
	assert.Contains(t, result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "</Response>")
	assert.Contains(t, result, "<Say>Route 8 to Seattle Center in 3 minutes.</Say>")

	// Should now contain gather for menu options
	assert.Contains(t, result, "<Gather")
	assert.Contains(t, result, "To hear more departures")
	assert.Contains(t, result, "press 1")
	assert.Contains(t, result, "To go back to the main menu")
	assert.Contains(t, result, "press 2")
	assert.Contains(t, result, "action=\"/voice/menu_action?minutesAfter=60\"")
}

func TestRenderVoiceFindStopWithMinutesAfter(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	// Test with MinutesAfter = 30 (should render minutesAfter=60 in URL)
	ctx := VoiceFindStopContext{
		ArrivalsMessage: "Route 8 to Seattle Center in 3 minutes.",
		MinutesAfter:    30,
	}

	result, err := manager.RenderVoiceFindStop(ctx)
	require.NoError(t, err)

	// Should contain the incremented value (30 + 30 = 60)
	assert.Contains(t, result, "action=\"/voice/menu_action?minutesAfter=60\"")

	// Test with MinutesAfter = 90 (should render minutesAfter=120 in URL)
	ctx.MinutesAfter = 90
	result, err = manager.RenderVoiceFindStop(ctx)
	require.NoError(t, err)

	// Should contain the incremented value (90 + 30 = 120)
	assert.Contains(t, result, "action=\"/voice/menu_action?minutesAfter=120\"")
}

func TestRenderVoiceError(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceErrorContext{
		ErrorMessage: "Sorry, I couldn't find that stop ID.",
	}

	result, err := manager.RenderVoiceError(ctx)
	require.NoError(t, err)

	// Check XML structure
	assert.Contains(t, result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "</Response>")
	assert.Contains(t, result, "<Say>Sorry, I couldn't find that stop ID.</Say>")
	assert.NotContains(t, result, "<Gather")
}

func TestRenderVoiceDisambiguation(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceDisambiguationContext{
		DisambiguationPrompt: "I found 2 stops. Press 1 for Metro stop, press 2 for Sound Transit stop.",
	}

	result, err := manager.RenderVoiceDisambiguation(ctx)
	require.NoError(t, err)

	// Check XML structure
	assert.Contains(t, result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "</Response>")
	assert.Contains(t, result, "<Gather")
	assert.Contains(t, result, "numDigits=\"1\"")
	assert.Contains(t, result, "action=\"/voice/find_stop\"")
	assert.Contains(t, result, "method=\"POST\"")
	assert.Contains(t, result, "<Say>I found 2 stops. Press 1 for Metro stop, press 2 for Sound Transit stop.</Say>")
}

func TestRenderVoiceStartWithSpecialCharacters(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceStartContext{
		WelcomePrompt: "Welcome to OneBusAway & transit info. Press # when done.",
	}

	result, err := manager.RenderVoiceStart(ctx)
	require.NoError(t, err)

	// For voice templates, special characters are left unescaped for TTS compatibility
	assert.Contains(t, result, "Welcome to OneBusAway & transit info. Press # when done.")
}

func TestRenderVoiceErrorWithSpecialCharacters(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceErrorContext{
		ErrorMessage: "Error: stop ID \"12345\" not found & unavailable.",
	}

	result, err := manager.RenderVoiceError(ctx)
	require.NoError(t, err)

	// For voice templates, special characters are left unescaped for TTS compatibility
	assert.Contains(t, result, "Error: stop ID \"12345\" not found & unavailable.")
}

func TestRenderVoiceFindStopEmpty(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceFindStopContext{
		ArrivalsMessage: "",
	}

	result, err := manager.RenderVoiceFindStop(ctx)
	require.NoError(t, err)

	// Should still render valid XML even with empty content
	assert.Contains(t, result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "<Say></Say>")
	assert.Contains(t, result, "</Response>")
}

func TestRenderVoiceErrorEmpty(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	ctx := VoiceErrorContext{
		ErrorMessage: "",
	}

	result, err := manager.RenderVoiceError(ctx)
	require.NoError(t, err)

	// Should still render valid XML even with empty content
	assert.Contains(t, result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, result, "<Response>")
	assert.Contains(t, result, "<Say></Say>")
	assert.Contains(t, result, "</Response>")
}

func TestAllTemplatesProduceValidXML(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	tests := []struct {
		name     string
		template func() (string, error)
	}{
		{
			"VoiceStart",
			func() (string, error) {
				return manager.RenderVoiceStart(VoiceStartContext{
					WelcomePrompt: "Test prompt",
				})
			},
		},
		{
			"VoiceFindStop",
			func() (string, error) {
				return manager.RenderVoiceFindStop(VoiceFindStopContext{
					ArrivalsMessage: "Test arrivals",
				})
			},
		},
		{
			"VoiceError",
			func() (string, error) {
				return manager.RenderVoiceError(VoiceErrorContext{
					ErrorMessage: "Test error",
				})
			},
		},
		{
			"VoiceDisambiguation",
			func() (string, error) {
				return manager.RenderVoiceDisambiguation(VoiceDisambiguationContext{
					DisambiguationPrompt: "Test disambiguation",
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.template()
			require.NoError(t, err)

			// Check basic XML structure
			assert.True(t, strings.HasPrefix(result, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"))
			assert.Contains(t, result, "<Response>")
			assert.Contains(t, result, "</Response>")

			// Count opening and closing tags for basic validation
			openResponseCount := strings.Count(result, "<Response>")
			closeResponseCount := strings.Count(result, "</Response>")
			assert.Equal(t, 1, openResponseCount, "Should have exactly one opening Response tag")
			assert.Equal(t, 1, closeResponseCount, "Should have exactly one closing Response tag")

			openSayCount := strings.Count(result, "<Say>")
			closeSayCount := strings.Count(result, "</Say>")
			assert.Equal(t, openSayCount, closeSayCount, "Say tags should be balanced")
		})
	}
}

func TestTemplateManagerConcurrency(t *testing.T) {
	manager, err := NewVoiceTemplateManager()
	require.NoError(t, err)

	// Test concurrent access to template manager
	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx := VoiceStartContext{
				WelcomePrompt: "Concurrent test " + string(rune('0'+id)),
			}
			result, err := manager.RenderVoiceStart(ctx)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			assert.Contains(t, result, "Concurrent test")
		case err := <-errors:
			t.Errorf("Unexpected error in concurrent test: %v", err)
		}
	}
}
