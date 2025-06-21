package voice

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"oba-twilio/middleware"
	"oba-twilio/models"
	"oba-twilio/validation"
)

func (h *Handler) HandleVoiceStart(c *gin.Context) {
	var req models.TwilioVoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		language := h.getLanguageFromRequest(c)
		h.ErrorHandler.HandleValidationError(c, err, "voice", language)
		return
	}

	// Validate phone number
	if err := validation.ValidatePhoneNumber(req.From); err != nil {
		language := h.getLanguageFromRequest(c)
		h.ErrorHandler.HandleValidationError(c, err, "voice", language)
		return
	}

	log.Printf("Received voice call from %s", req.From)

	language := h.getLanguageFromRequest(c)

	// Track voice request
	if h.analyticsManager != nil {
		middleware.TrackVoiceRequest(c.Request.Context(), h.analyticsManager, req.From, language, h.analyticsHashSalt)
	}

	// Handle language selection based on supported languages count
	languageCount := h.LocalizationManager.GetLanguageCount()

	c.Header("Content-Type", "text/xml")

	switch languageCount {
	case 1:
		// Single language - proceed directly
		h.renderMainMenuWithLanguage(c, language)
	case 2:
		// Two languages - show main menu with Spanish option
		h.renderMainMenuWithSpanishOption(c, language)
	default:
		// Multiple languages - show main menu with language selection option
		h.renderMainMenuWithLanguageSelection(c, language)
	}
}

// renderMainMenuWithLanguage renders the main menu for single language setup
func (h *Handler) renderMainMenuWithLanguage(c *gin.Context, language string) {
	prompt := h.LocalizationManager.GetString("voice.welcome", language)
	log.Printf("Rendering main menu with language %s, prompt: %s", language, prompt)

	twiml, err := h.TemplateManager.RenderVoiceStart(VoiceStartContext{
		WelcomePrompt: prompt,
		Language:      language,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		errorMsg := h.LocalizationManager.GetString("voice.error.template_failed", language)
		twiml, _ = h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: errorMsg,
		})
	}

	log.Printf("Generated initial voice TwiML, length: %d", len(twiml))
	if len(twiml) < 1000 {
		log.Printf("Initial voice TwiML content: %s", twiml)
	}

	c.String(http.StatusOK, twiml)
}

// renderMainMenuWithSpanishOption renders the main menu with Spanish language option
func (h *Handler) renderMainMenuWithSpanishOption(c *gin.Context, language string) {
	prompt := h.LocalizationManager.GetString("voice.welcome", language)
	spanishOption := h.LocalizationManager.GetString("voice.language.spanish_option", language)

	fullPrompt := prompt + " " + spanishOption

	twiml, err := h.TemplateManager.RenderVoiceStart(VoiceStartContext{
		WelcomePrompt: fullPrompt,
		Language:      language,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		errorMsg := h.LocalizationManager.GetString("voice.error.template_failed", language)
		twiml, _ = h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: errorMsg,
		})
	}

	c.String(http.StatusOK, twiml)
}

// renderMainMenuWithLanguageSelection renders the main menu with language selection option
func (h *Handler) renderMainMenuWithLanguageSelection(c *gin.Context, language string) {
	prompt := h.LocalizationManager.GetString("voice.welcome", language)
	languageOption := h.LocalizationManager.GetString("voice.language.choose_language", language)

	fullPrompt := prompt + " " + languageOption

	twiml, err := h.TemplateManager.RenderVoiceStart(VoiceStartContext{
		WelcomePrompt: fullPrompt,
		Language:      language,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		errorMsg := h.LocalizationManager.GetString("voice.error.template_failed", language)
		twiml, _ = h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: errorMsg,
		})
	}

	c.String(http.StatusOK, twiml)
}
