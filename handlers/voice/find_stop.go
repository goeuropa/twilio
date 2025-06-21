package voice

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"oba-twilio/formatters"
	"oba-twilio/models"
	"oba-twilio/validation"
)

func (h *Handler) HandleFindStop(c *gin.Context) {
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

	// Validate call SID if provided
	if req.CallSid != "" {
		if err := validation.ValidateTwilioCallSid(req.CallSid); err != nil {
			log.Printf("Invalid call SID from %s: %v", req.From, err)
		}
	}

	// Sanitize digits input
	req.Digits = validation.SanitizeUserInput(req.Digits)

	log.Printf("Received voice input from %s: %s", req.From, req.Digits)

	c.Header("Content-Type", "text/xml")

	// Check if user is responding to a disambiguation request
	if choice := h.parseDisambiguationChoice(req.Digits); choice > 0 {
		// Additional validation for the choice
		session := h.SessionStore.GetDisambiguationSession(req.From)
		if session != nil {
			maxChoices := len(session.StopOptions)
			if maxChoices > 9 {
				maxChoices = 9 // Voice interface limits to single digits
			}
			if err := validation.ValidateDisambiguationChoice(req.Digits, maxChoices); err != nil {
				log.Printf("Invalid disambiguation choice from %s: %v", req.From, err)
				language := h.getLanguageFromRequest(c)
				errorMsg := h.LocalizationManager.GetString("voice.error.invalid_choice", language, maxChoices)
				twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
					ErrorMessage: errorMsg,
				})
				c.String(http.StatusOK, twiml)
				return
			}
		}
		h.handleVoiceDisambiguationChoice(c, req, choice)
		return
	}

	// Clear any existing disambiguation session for new queries
	h.SessionStore.ClearDisambiguationSession(req.From)

	stopID := req.Digits
	if stopID == "" {
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "I didn't receive any digits. Please try calling again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	// Validate stop ID format and security
	if err := validation.ValidateStopID(stopID); err != nil {
		log.Printf("Invalid stop ID from %s: %s, error: %v", req.From, stopID, err)
		language := h.getLanguageFromRequest(c)
		errorMsg := h.LocalizationManager.GetString("voice.error.invalid_stop_id", language)
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: errorMsg,
		})
		c.String(http.StatusOK, twiml)
		return
	}

	// Find all matching stops for the given ID
	matchingStops, err := h.OBAClient.FindAllMatchingStops(stopID)
	if err != nil {
		language := h.getLanguageFromRequest(c)
		h.ErrorHandler.HandleVoiceError(c, err, language)
		return
	}

	if len(matchingStops) == 0 {
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "Sorry, I couldn't find any stops with that ID. Please check the stop ID and try again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	// If multiple stops found, ask user to disambiguate
	if len(matchingStops) > 1 {
		disambiguationMsg := h.formatVoiceDisambiguationMessage(matchingStops, stopID)

		// Store disambiguation session
		session := &models.DisambiguationSession{
			StopOptions: matchingStops,
		}
		if err := h.SessionStore.SetDisambiguationSession(req.From, session); err != nil {
			language := h.getLanguageFromRequest(c)
			h.ErrorHandler.HandleInternalError(c, err, "voice", language)
			return
		}

		// Use TwiML Gather to collect the user's choice
		twiml, err := h.TemplateManager.RenderVoiceDisambiguation(VoiceDisambiguationContext{
			DisambiguationPrompt: disambiguationMsg,
		})
		if err != nil {
			log.Printf("Failed to generate TwiML gather: %v", err)
			twiml, _ = h.TemplateManager.RenderVoiceError(VoiceErrorContext{
				ErrorMessage: "Error generating response.",
			})
		}
		c.String(http.StatusOK, twiml)
		return
	}

	// Single stop found, get arrivals directly
	h.getAndFormatVoiceArrivalsWithSession(c, req.From, matchingStops[0].FullStopID, 0)
}

// parseDisambiguationChoice checks if the input digits represent a single-digit choice (1-9)
func (h *Handler) parseDisambiguationChoice(digits string) int {
	if len(digits) != 1 {
		return 0
	}

	choice, err := strconv.Atoi(digits)
	if err != nil || choice < 1 || choice > 9 {
		return 0
	}

	return choice
}

// handleVoiceDisambiguationChoice processes the user's disambiguation choice
func (h *Handler) handleVoiceDisambiguationChoice(c *gin.Context, req models.TwilioVoiceRequest, choice int) {
	session := h.SessionStore.GetDisambiguationSession(req.From)
	if session == nil {
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: "No active selection. Please call again and enter a stop ID to get started.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	if choice < 1 || choice > len(session.StopOptions) {
		maxChoice := len(session.StopOptions)
		if maxChoice > 9 {
			maxChoice = 9 // Limit voice choices to single digits
		}
		twiml, _ := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: fmt.Sprintf("Please press a number between 1 and %d.", maxChoice),
		})
		c.String(http.StatusOK, twiml)
		return
	}

	selectedStop := session.StopOptions[choice-1]
	h.SessionStore.ClearDisambiguationSession(req.From)

	log.Printf("User %s selected stop %s: %s", req.From, selectedStop.FullStopID, selectedStop.DisplayText)

	h.getAndFormatVoiceArrivalsWithSession(c, req.From, selectedStop.FullStopID, 0)
}

// formatVoiceDisambiguationMessage creates a voice-friendly disambiguation message
func (h *Handler) formatVoiceDisambiguationMessage(stops []models.StopOption, stopID string) string {
	msg := fmt.Sprintf("I found %d stops with ID %s. ", len(stops), stopID)

	// Limit to first 9 options for voice interface (single digit choices)
	maxStops := len(stops)
	if maxStops > 9 {
		maxStops = 9
	}

	for i := 0; i < maxStops; i++ {
		stop := stops[i]
		msg += fmt.Sprintf("Press %d for %s. ", i+1, stop.StopName)
	}

	if len(stops) > 9 {
		msg += "Only showing first 9 options. "
	}

	msg += "Which stop would you like?"
	return msg
}

// getAndFormatVoiceArrivalsWithSession fetches arrival information with a custom window and formats it for voice response
func (h *Handler) getAndFormatVoiceArrivalsWithSession(c *gin.Context, phoneNumber, fullStopID string, minutesAfter int) {
	var obaResp *models.OneBusAwayResponse
	var err error

	// Use 30 minutes as default window if minutesAfter is 0, otherwise use provided value
	window := minutesAfter
	if window == 0 {
		window = 30
	}

	obaResp, err = h.OBAClient.GetArrivalsAndDeparturesWithWindow(fullStopID, window)

	if err != nil {
		language := h.getLanguageFromRequest(c)
		h.ErrorHandler.HandleVoiceError(c, err, language)
		return
	}

	arrivals := h.OBAClient.ProcessArrivals(obaResp)

	// Get the human-readable stop name instead of using the technical stop ID
	stopName := ""
	if stopInfo, err := h.OBAClient.GetStopInfo(fullStopID); err == nil && stopInfo != nil {
		stopName = stopInfo.StopName
	} else {
		// Fall back to stop ID if we can't get the stop name
		stopName = obaResp.Data.Entry.StopId
	}

	language := h.getLanguageFromRequest(c)
	log.Printf("Formatting voice response for %s: stop=%s, arrivals=%d", phoneNumber, stopName, len(arrivals))

	message := formatters.FormatVoiceResponse(arrivals, stopName, h.LocalizationManager, language)
	log.Printf("Voice message for %s: %s", phoneNumber, message)
	if message == "" {
		log.Printf("Empty voice response generated for %s", phoneNumber)
		language := h.getLanguageFromRequest(c)
		h.ErrorHandler.HandleVoiceError(c, fmt.Errorf("failed to format voice response"), language)
		return
	}

	// Set up voice session for menu options
	session := &models.VoiceSession{
		StopID:       fullStopID,
		MinutesAfter: minutesAfter,
	}
	if err := h.SessionStore.SetVoiceSession(phoneNumber, session); err != nil {
		log.Printf("Failed to set voice session for %s: %v", phoneNumber, err)
	}
	menuPrompt := h.LocalizationManager.GetString("voice.menu.more_departures", language) + " " + h.LocalizationManager.GetString("voice.menu.main_menu", language)

	log.Printf("Rendering TwiML for %s with message length: %d", phoneNumber, len(message))
	twiml, err := h.TemplateManager.RenderVoiceFindStop(VoiceFindStopContext{
		ArrivalsMessage: message,
		MinutesAfter:    minutesAfter,
		MenuPrompt:      menuPrompt,
		Language:        language,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML for %s: %v", phoneNumber, err)
		errorMsg := h.LocalizationManager.GetString("voice.error.template_failed", language)
		errorTwiml, err2 := h.TemplateManager.RenderVoiceError(VoiceErrorContext{
			ErrorMessage: errorMsg,
		})
		if err2 != nil {
			log.Printf("Failed to generate error TwiML for %s: %v", phoneNumber, err2)
			// Use absolute fallback
			twiml = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><Response><Say>%s</Say></Response>`, errorMsg)
		} else {
			twiml = errorTwiml
		}
	}

	log.Printf("Generated TwiML for %s, length: %d", phoneNumber, len(twiml))
	// Log first 500 chars of TwiML for debugging
	if len(twiml) > 500 {
		log.Printf("TwiML content preview: %s...", twiml[:500])
	} else {
		log.Printf("TwiML content: %s", twiml)
	}

	c.String(http.StatusOK, twiml)
}
