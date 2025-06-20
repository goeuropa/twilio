package handlers

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"oba-twilio/client"
	"oba-twilio/formatters"
	"oba-twilio/localization"
	"oba-twilio/models"
)

// Pre-compiled regex patterns for performance
var (
	timeRegex     = regexp.MustCompile(`^\+(\d+)$`)
	hourRegex     = regexp.MustCompile(`^\+(\d+)h$`)
	nextHourRegex = regexp.MustCompile(`^next\s+(\d+)?\s*hours?$`)
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

	// Get or create SMS session for language persistence
	smsSession := h.getOrCreateSMSSession(req.From)

	// Check for keywords first
	if h.handleKeywords(c, req, smsSession) {
		return
	}

	// Check if user is responding to a disambiguation request
	if choice := formatters.IsDisambiguationChoice(req.Body); choice > 0 {
		h.handleDisambiguationChoice(c, req, choice)
		return
	}

	// Clear any existing disambiguation session for new queries
	h.SessionStore.ClearDisambiguationSession(req.From)

	// Extract language and stop ID from message
	language, stopID := h.extractLanguageFromMessage(req.Body, smsSession)
	if stopID == "" {
		errorMsg := h.LocalizationManager.GetString("sms.error.invalid_stop", language)
		twiml, _ := formatters.GenerateTwiMLSMS(errorMsg)
		c.String(http.StatusOK, twiml)
		return
	}

	// Update SMS session with current language and stop
	smsSession.Language = language
	smsSession.LastStopID = stopID
	smsSession.LastQueryTime = time.Now().Unix()
	if err := h.SessionStore.SetSMSSession(req.From, smsSession); err != nil {
		log.Printf("Failed to set SMS session for %s: %v", req.From, err)
	}

	// Find all matching stops for the given ID
	matchingStops, err := h.OBAClient.FindAllMatchingStops(stopID)
	if err != nil {
		log.Printf("Error finding matching stops for %s: %v", stopID, err)
		var message string
		if strings.Contains(err.Error(), "cannot be empty") {
			message = h.LocalizationManager.GetString("sms.error.invalid_stop", language)
		} else if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection") {
			message = h.LocalizationManager.GetString("sms.service_unavailable", language)
		} else {
			message = h.LocalizationManager.GetString("sms.error.search_failed", language, stopID)
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
	h.getAndFormatArrivalsWithStopNameAndSession(c, req.From, matchingStops[0].FullStopID, matchingStops[0].DisplayText, smsSession)
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

	// Get SMS session for this user to maintain consistency
	smsSession := h.getOrCreateSMSSession(req.From)
	smsSession.LastStopID = selectedStop.FullStopID
	h.getAndFormatArrivalsWithStopNameAndSession(c, req.From, selectedStop.FullStopID, selectedStop.DisplayText, smsSession)
}

func (h *SMSHandler) getAndFormatArrivalsWithStopNameAndSession(c *gin.Context, phoneNumber string, fullStopID string, stopDisplayName string, session *models.SMSSession) {
	var obaResp *models.OneBusAwayResponse
	var err error

	// Use session window or default
	window := session.WindowMinutes
	if window == 0 {
		window = 30
	}

	// Get arrivals with specified window (use default method if window is 30)
	if window == 30 {
		obaResp, err = h.OBAClient.GetArrivalsAndDepartures(fullStopID)
	} else {
		obaResp, err = h.OBAClient.GetArrivalsAndDeparturesWithWindow(fullStopID, window)
	}
	if err != nil {
		log.Printf("OneBusAway API error for stop %s: %v", fullStopID, err)
		errorMsg := h.LocalizationManager.GetString("sms.service_unavailable", session.Language)
		twiml, _ := formatters.GenerateTwiMLSMS(errorMsg)
		c.String(http.StatusOK, twiml)
		return
	}

	arrivals := h.OBAClient.ProcessArrivals(obaResp)

	// Use stop name from response if display name is empty
	if stopDisplayName == "" && obaResp.Data.Entry.StopId != "" {
		stopDisplayName = obaResp.Data.Entry.StopId // This should be improved to get actual stop name
	}

	message := formatters.FormatSMSResponse(arrivals, stopDisplayName)

	// Add menu hints if there are arrivals
	if len(arrivals) > 0 {
		moreHint := h.LocalizationManager.GetString("sms.menu.more_hint", session.Language)
		helpHint := h.LocalizationManager.GetString("sms.menu.help_hint", session.Language)
		message += fmt.Sprintf("\n\n%s | %s", moreHint, helpHint)
	}

	// Update session with current stop atomically
	newSession := *session
	newSession.LastStopID = fullStopID
	newSession.LastQueryTime = time.Now().Unix()
	if err := h.SessionStore.SetSMSSession(phoneNumber, &newSession); err != nil {
		log.Printf("Failed to set SMS session for %s: %v", phoneNumber, err)
		return
	}
	*session = newSession

	twiml, err := formatters.GenerateTwiMLSMS(message)
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		errorMsg := h.LocalizationManager.GetString("sms.service_unavailable", session.Language)
		twiml, _ = formatters.GenerateTwiMLSMS(errorMsg)
	}

	c.String(http.StatusOK, twiml)
}

// getOrCreateSMSSession gets existing SMS session or creates a new one
func (h *SMSHandler) getOrCreateSMSSession(phoneNumber string) *models.SMSSession {
	session := h.SessionStore.GetSMSSession(phoneNumber)
	if session == nil {
		session = &models.SMSSession{
			Language:      h.LocalizationManager.GetPrimaryLanguage(),
			WindowMinutes: 30, // Default window
			LastQueryTime: time.Now().Unix(),
		}
	}
	return session
}

// handleKeywords processes special keywords like 'more', 'help', 'new'
func (h *SMSHandler) handleKeywords(c *gin.Context, req models.TwilioSMSRequest, session *models.SMSSession) bool {
	message := strings.TrimSpace(strings.ToLower(req.Body))

	switch message {
	case "help":
		h.handleHelpRequest(c, req, session)
		return true
	case "more", "later", "extend":
		h.handleMoreRequest(c, req, session)
		return true
	case "new", "stop", "clear":
		h.handleNewRequest(c, req, session)
		return true
	}

	// Check for language switching keywords
	if h.handleLanguageSwitching(c, req, session, message) {
		return true
	}

	// Check for time-based queries like "+30", "next hour"
	if h.handleTimeQuery(c, req, session, message) {
		return true
	}

	return false
}

// handleHelpRequest provides usage help
func (h *SMSHandler) handleHelpRequest(c *gin.Context, req models.TwilioSMSRequest, session *models.SMSSession) {
	helpMsg := h.LocalizationManager.GetString("sms.help", session.Language)
	twiml, _ := formatters.GenerateTwiMLSMS(helpMsg)
	c.String(http.StatusOK, twiml)
}

// handleMoreRequest extends the departure window
func (h *SMSHandler) handleMoreRequest(c *gin.Context, req models.TwilioSMSRequest, session *models.SMSSession) {
	if session.LastStopID == "" {
		errorMsg := h.LocalizationManager.GetString("sms.session.expired", session.Language)
		twiml, _ := formatters.GenerateTwiMLSMS(errorMsg)
		c.String(http.StatusOK, twiml)
		return
	}

	// Extend the window (30, 60, 90 minutes)
	newWindow := session.WindowMinutes + 30
	if newWindow > 120 {
		newWindow = 120 // Cap at 2 hours
	}

	// Update session atomically
	updatedSession := *session
	updatedSession.WindowMinutes = newWindow
	updatedSession.LastQueryTime = time.Now().Unix()
	if err := h.SessionStore.SetSMSSession(req.From, &updatedSession); err != nil {
		log.Printf("Failed to set SMS session for %s: %v", req.From, err)
		return
	}
	*session = updatedSession

	h.getAndFormatArrivalsWithStopNameAndSession(c, req.From, session.LastStopID, "", session)
}

// handleNewRequest clears session for new stop query
func (h *SMSHandler) handleNewRequest(c *gin.Context, req models.TwilioSMSRequest, session *models.SMSSession) {
	h.SessionStore.ClearSMSSession(req.From)
	h.SessionStore.ClearDisambiguationSession(req.From)
	helpMsg := h.LocalizationManager.GetString("sms.help", session.Language)
	twiml, _ := formatters.GenerateTwiMLSMS(helpMsg)
	c.String(http.StatusOK, twiml)
}

// handleLanguageSwitching processes language change requests
func (h *SMSHandler) handleLanguageSwitching(c *gin.Context, req models.TwilioSMSRequest, session *models.SMSSession, message string) bool {
	languageMap := map[string]string{
		"english":  "en-US",
		"español":  "es-US",
		"spanish":  "es-US",
		"français": "fr-US",
		"french":   "fr-US",
		"deutsch":  "de-US",
		"german":   "de-US",
	}

	if newLang, found := languageMap[message]; found && h.LocalizationManager.IsSupported(newLang) {
		session.Language = newLang
		if err := h.SessionStore.SetSMSSession(req.From, session); err != nil {
			log.Printf("Failed to set SMS session for %s: %v", req.From, err)
		}
		switchedMsg := h.LocalizationManager.GetString("sms.language.switched", newLang)
		twiml, _ := formatters.GenerateTwiMLSMS(switchedMsg)
		c.String(http.StatusOK, twiml)
		return true
	}

	return false
}

// handleTimeQuery processes time-based queries like "+30", "next hour"
func (h *SMSHandler) handleTimeQuery(c *gin.Context, req models.TwilioSMSRequest, session *models.SMSSession, message string) bool {
	if session.LastStopID == "" {
		return false
	}

	// Match patterns like "+30", "+1h", "next hour", "next 2 hours"

	var newWindow int

	if matches := timeRegex.FindStringSubmatch(message); len(matches) > 1 {
		if minutes, err := strconv.Atoi(matches[1]); err == nil && minutes > 0 && minutes <= 120 {
			newWindow = minutes
		}
	} else if matches := hourRegex.FindStringSubmatch(message); len(matches) > 1 {
		if hours, err := strconv.Atoi(matches[1]); err == nil && hours > 0 && hours <= 2 {
			newWindow = hours * 60
		}
	} else if matches := nextHourRegex.FindStringSubmatch(message); len(matches) > 0 {
		hours := 1
		if matches[1] != "" {
			if h, err := strconv.Atoi(matches[1]); err == nil && h > 0 && h <= 2 {
				hours = h
			}
		}
		newWindow = hours * 60
	} else if message == "next hour" {
		newWindow = 60
	}

	if newWindow > 0 {
		// Update session atomically
		updatedSession := *session
		updatedSession.WindowMinutes = newWindow
		updatedSession.LastQueryTime = time.Now().Unix()
		if err := h.SessionStore.SetSMSSession(req.From, &updatedSession); err != nil {
			log.Printf("Failed to set SMS session for %s: %v", req.From, err)
			return false
		}
		*session = updatedSession
		h.getAndFormatArrivalsWithStopNameAndSession(c, req.From, session.LastStopID, "", session)
		return true
	}

	return false
}

// extractLanguageFromMessage extracts language prefix and stop ID from SMS message
func (h *SMSHandler) extractLanguageFromMessage(message string, session *models.SMSSession) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(message), " ", 2)
	if len(parts) == 2 {
		if h.LocalizationManager.IsSupported(parts[0]) {
			return parts[0], formatters.ExtractStopID(parts[1])
		}
	}
	// Use session language if available, otherwise primary language
	language := session.Language
	if language == "" {
		language = h.LocalizationManager.GetPrimaryLanguage()
	}
	return language, formatters.ExtractStopID(message)
}
