package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"oba-twilio/client"
	"oba-twilio/formatters"
	"oba-twilio/models"
)

type VoiceHandler struct {
	OBAClient       client.OneBusAwayClientInterface
	SessionStore    *SessionStore
	TemplateManager *formatters.VoiceTemplateManager
}

func NewVoiceHandler(obaClient client.OneBusAwayClientInterface) *VoiceHandler {
	templateManager, err := formatters.NewVoiceTemplateManager()
	if err != nil {
		log.Fatalf("Failed to initialize voice template manager: %v", err)
	}

	return &VoiceHandler{
		OBAClient:       obaClient,
		SessionStore:    NewSessionStore(),
		TemplateManager: templateManager,
	}
}

func (h *VoiceHandler) Close() {
	if h.SessionStore != nil {
		h.SessionStore.Close()
	}
}

func (h *VoiceHandler) HandleVoiceStart(c *gin.Context) {
	var req models.TwilioVoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Failed to bind voice request: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "Invalid request format.",
		})
		c.String(http.StatusBadRequest, twiml)
		return
	}

	log.Printf("Received voice call from %s", req.From)

	prompt := "Welcome to OneBusAway transit information. Please enter your stop ID followed by the pound key."

	c.Header("Content-Type", "text/xml")
	twiml, err := h.TemplateManager.RenderVoiceStart(formatters.VoiceStartContext{
		WelcomePrompt: prompt,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "Error generating response.",
		})
	}

	c.String(http.StatusOK, twiml)
}

func (h *VoiceHandler) HandleFindStop(c *gin.Context) {
	var req models.TwilioVoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Failed to bind voice input request: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "Invalid request format.",
		})
		c.String(http.StatusBadRequest, twiml)
		return
	}

	log.Printf("Received voice input from %s: %s", req.From, req.Digits)

	c.Header("Content-Type", "text/xml")

	// Check if user is responding to a disambiguation request
	if choice := h.parseDisambiguationChoice(req.Digits); choice > 0 {
		h.handleVoiceDisambiguationChoice(c, req, choice)
		return
	}

	// Clear any existing disambiguation session for new queries
	h.SessionStore.ClearDisambiguationSession(req.From)

	stopID := req.Digits
	if stopID == "" {
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "I didn't receive any digits. Please try calling again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	if len(stopID) < 3 || len(stopID) > 10 {
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "Invalid stop ID. Please try calling again with a valid stop ID.",
		})
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
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: message,
		})
		c.String(http.StatusOK, twiml)
		return
	}

	if len(matchingStops) == 0 {
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
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
			log.Printf("Failed to store disambiguation session for %s: %v", req.From, err)
			twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
				ErrorMessage: "Sorry, there was an error processing your request. Please try again.",
			})
			c.String(http.StatusOK, twiml)
			return
		}

		// Use TwiML Gather to collect the user's choice
		twiml, err := h.TemplateManager.RenderVoiceDisambiguation(formatters.VoiceDisambiguationContext{
			DisambiguationPrompt: disambiguationMsg,
		})
		if err != nil {
			log.Printf("Failed to generate TwiML gather: %v", err)
			twiml, _ = h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
				ErrorMessage: "Error generating response.",
			})
		}
		c.String(http.StatusOK, twiml)
		return
	}

	// Single stop found, get arrivals directly
	h.getAndFormatVoiceArrivals(c, matchingStops[0].FullStopID)
}

// parseDisambiguationChoice checks if the input digits represent a single-digit choice (1-9)
func (h *VoiceHandler) parseDisambiguationChoice(digits string) int {
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
func (h *VoiceHandler) handleVoiceDisambiguationChoice(c *gin.Context, req models.TwilioVoiceRequest, choice int) {
	session := h.SessionStore.GetDisambiguationSession(req.From)
	if session == nil {
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
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
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: fmt.Sprintf("Please press a number between 1 and %d.", maxChoice),
		})
		c.String(http.StatusOK, twiml)
		return
	}

	selectedStop := session.StopOptions[choice-1]
	h.SessionStore.ClearDisambiguationSession(req.From)

	log.Printf("User %s selected stop %s: %s", req.From, selectedStop.FullStopID, selectedStop.DisplayText)

	h.getAndFormatVoiceArrivals(c, selectedStop.FullStopID)
}

// formatVoiceDisambiguationMessage creates a voice-friendly disambiguation message
func (h *VoiceHandler) formatVoiceDisambiguationMessage(stops []models.StopOption, stopID string) string {
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

// getAndFormatVoiceArrivals fetches arrival information and formats it for voice response
func (h *VoiceHandler) getAndFormatVoiceArrivals(c *gin.Context, fullStopID string) {
	obaResp, err := h.OBAClient.GetArrivalsAndDepartures(fullStopID)
	if err != nil {
		log.Printf("OneBusAway API error for stop %s: %v", fullStopID, err)
		twiml, _ := h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "Sorry, I couldn't get arrival information for that stop. Please try again.",
		})
		c.String(http.StatusOK, twiml)
		return
	}

	arrivals := h.OBAClient.ProcessArrivals(obaResp)
	stopName := obaResp.Data.Entry.StopId // ABXOXO: FIXME // obaResp.Data.Entry.Stop.Name

	message := formatters.FormatVoiceResponse(arrivals, stopName)

	twiml, err := h.TemplateManager.RenderVoiceFindStop(formatters.VoiceFindStopContext{
		ArrivalsMessage: message,
	})
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = h.TemplateManager.RenderVoiceError(formatters.VoiceErrorContext{
			ErrorMessage: "Error generating response.",
		})
	}

	c.String(http.StatusOK, twiml)
}
