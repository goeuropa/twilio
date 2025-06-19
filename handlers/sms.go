package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"oba-twilio/client"
	"oba-twilio/formatters"
	"oba-twilio/localization"
	"oba-twilio/models"
)

type SMSHandler struct {
	OBAClient           client.OneBusAwayClientInterface
	SessionStore        *SessionStore
	LocalizationManager *localization.LocalizationManager
}

func NewSMSHandler(obaClient client.OneBusAwayClientInterface, locManager *localization.LocalizationManager) *SMSHandler {
	return &SMSHandler{
		OBAClient:           obaClient,
		SessionStore:        NewSessionStore(),
		LocalizationManager: locManager,
	}
}

func (h *SMSHandler) Close() {
	if h.SessionStore != nil {
		h.SessionStore.Close()
	}
}

func (h *SMSHandler) HandleSMS(c *gin.Context) {
	var req models.TwilioSMSRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Failed to bind SMS request: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLSMS("Invalid request format.")
		c.String(http.StatusBadRequest, twiml)
		return
	}

	log.Printf("Received SMS from %s: %s", req.From, req.Body)

	c.Header("Content-Type", "text/xml")

	// Check if user is responding to a disambiguation request
	if choice := formatters.IsDisambiguationChoice(req.Body); choice > 0 {
		h.handleDisambiguationChoice(c, req, choice)
		return
	}

	// Clear any existing disambiguation session for new queries
	h.SessionStore.ClearDisambiguationSession(req.From)

	// Extract language and stop ID from message
	language, stopID := h.extractLanguageFromMessage(req.Body)
	if stopID == "" {
		twiml, _ := formatters.GenerateTwiMLSMS("Please send a valid stop ID (e.g., 75403).")
		c.String(http.StatusOK, twiml)
		return
	}

	// Find all matching stops for the given ID
	matchingStops, err := h.OBAClient.FindAllMatchingStops(stopID)
	if err != nil {
		log.Printf("Error finding matching stops for %s: %v", stopID, err)
		var message string
		if strings.Contains(err.Error(), "cannot be empty") {
			message = "Please provide a valid stop ID."
		} else if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection") {
			message = "OneBusAway service is temporarily unavailable. Please try again in a moment."
		} else {
			message = fmt.Sprintf("Sorry, I couldn't search for stop %s. Please check the stop ID and try again.", stopID)
		}
		twiml, _ := formatters.GenerateTwiMLSMS(message)
		c.String(http.StatusOK, twiml)
		return
	}

	if len(matchingStops) == 0 {
		errorMsg := h.LocalizationManager.GetString("sms.no_stops_found", language, stopID)
		twiml, _ := formatters.GenerateTwiMLSMS(errorMsg)
		c.String(http.StatusOK, twiml)
		return
	}

	// If multiple stops found, ask user to disambiguate
	if len(matchingStops) > 1 {
		disambiguationMsg := formatters.FormatDisambiguationMessage(matchingStops, stopID)

		// Store disambiguation session
		session := &models.DisambiguationSession{
			StopOptions: matchingStops,
		}
		if err := h.SessionStore.SetDisambiguationSession(req.From, session); err != nil {
			log.Printf("Failed to store disambiguation session for %s: %v", req.From, err)
			twiml, _ := formatters.GenerateTwiMLSMS("Sorry, there was an error processing your request. Please try again.")
			c.String(http.StatusOK, twiml)
			return
		}

		twiml, _ := formatters.GenerateTwiMLSMS(disambiguationMsg)
		c.String(http.StatusOK, twiml)
		return
	}

	// Single stop found, get arrivals directly
	h.getAndFormatArrivalsWithStopName(c, matchingStops[0].FullStopID, matchingStops[0].DisplayText)
}

func (h *SMSHandler) handleDisambiguationChoice(c *gin.Context, req models.TwilioSMSRequest, choice int) {
	session := h.SessionStore.GetDisambiguationSession(req.From)
	if session == nil {
		twiml, _ := formatters.GenerateTwiMLSMS("No active selection. Please send a stop ID to get started.")
		c.String(http.StatusOK, twiml)
		return
	}

	if choice < 1 || choice > len(session.StopOptions) {
		twiml, _ := formatters.GenerateTwiMLSMS(fmt.Sprintf("Please choose a number between 1 and %d.", len(session.StopOptions)))
		c.String(http.StatusOK, twiml)
		return
	}

	selectedStop := session.StopOptions[choice-1]
	h.SessionStore.ClearDisambiguationSession(req.From)

	log.Printf("User %s selected stop %s: %s", req.From, selectedStop.FullStopID, selectedStop.DisplayText)

	h.getAndFormatArrivalsWithStopName(c, selectedStop.FullStopID, selectedStop.DisplayText)
}

func (h *SMSHandler) getAndFormatArrivalsWithStopName(c *gin.Context, fullStopID string, stopDisplayName string) {
	obaResp, err := h.OBAClient.GetArrivalsAndDepartures(fullStopID)
	if err != nil {
		log.Printf("OneBusAway API error for stop %s: %v", fullStopID, err)
		twiml, _ := formatters.GenerateTwiMLSMS("Sorry, I couldn't get arrival information for that stop. Please try again.")
		c.String(http.StatusOK, twiml)
		return
	}

	arrivals := h.OBAClient.ProcessArrivals(obaResp)

	message := formatters.FormatSMSResponse(arrivals, stopDisplayName)

	twiml, err := formatters.GenerateTwiMLSMS(message)
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = formatters.GenerateTwiMLSMS("Error generating response.")
	}

	c.String(http.StatusOK, twiml)
}

// extractLanguageFromMessage extracts language prefix and stop ID from SMS message
func (h *SMSHandler) extractLanguageFromMessage(message string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(message), " ", 2)
	if len(parts) == 2 {
		if h.LocalizationManager.IsSupported(parts[0]) {
			return parts[0], formatters.ExtractStopID(parts[1])
		}
	}
	return h.LocalizationManager.GetPrimaryLanguage(), formatters.ExtractStopID(message)
}
