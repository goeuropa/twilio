package voice

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"oba-twilio/models"
)

func (h *Handler) HandleVoiceMenuAction(c *gin.Context) {
	var req models.TwilioVoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		language := h.getLanguageFromRequest(c)
		h.ErrorHandler.HandleValidationError(c, err, "voice", language)
		return
	}

	log.Printf("Received voice menu action from %s: %s", req.From, req.Digits)

	c.Header("Content-Type", "text/xml")

	switch req.Digits {
	case "1":
		// Option 1: Hear more departures
		h.handleExtendDepartures(c, req)
	case "2":
		// Option 2: Return to main menu
		h.handleReturnToMainMenu(c, req)
	default:
		// Invalid choice
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "Please press 1 or 2.",
		})
		c.String(http.StatusOK, twiml)
	}
}

// handleExtendDepartures extends the departure window and retrieves more arrivals
func (h *Handler) handleExtendDepartures(c *gin.Context, req models.TwilioVoiceRequest) {
	session := h.SessionStore.GetVoiceSession(req.From)
	if session == nil {
		// No session exists, return to main menu
		h.returnToMainMenu(c)
		return
	}

	// Get minutesAfter from query parameter
	minutesAfterStr := c.Query("minutesAfter")
	if minutesAfterStr == "" {
		log.Printf("Missing minutesAfter parameter in request from %s", req.From)
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "Sorry, there was an error processing your request. Please try again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	newMinutesAfter, err := strconv.Atoi(minutesAfterStr)
	if err != nil {
		log.Printf("Invalid minutesAfter parameter: %s from %s", minutesAfterStr, req.From)
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "Sorry, there was an error processing your request. Please try again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	// Update the session
	session.MinutesAfter = newMinutesAfter
	if err := h.SessionStore.SetVoiceSession(req.From, session); err != nil {
		log.Printf("Failed to update voice session for %s: %v", req.From, err)
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "Sorry, there was an error processing your request. Please try again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	log.Printf("Extended departures window for %s to %d minutes", req.From, newMinutesAfter)

	// Get arrivals with extended window
	h.getAndFormatVoiceArrivalsWithSession(c, req.From, session.StopID, newMinutesAfter)
}

// handleReturnToMainMenu clears the voice session and returns to the start menu
func (h *Handler) handleReturnToMainMenu(c *gin.Context, req models.TwilioVoiceRequest) {
	h.SessionStore.ClearVoiceSession(req.From)
	log.Printf("Cleared voice session for %s, returning to main menu", req.From)
	h.returnToMainMenu(c)
}

// returnToMainMenu renders the main menu
func (h *Handler) returnToMainMenu(c *gin.Context) {
	prompt := "Welcome to OneBusAway transit information. Please enter your stop ID followed by the pound key."

	twiml, err := h.TemplateManager.RenderVoiceStart(VoiceStartContext{
		WelcomePrompt: prompt,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "Error generating response.",
		})
	}

	c.String(http.StatusOK, twiml)
}
