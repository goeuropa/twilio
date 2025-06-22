package common

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/twilio/twilio-go/twiml"

	"oba-twilio/formatters"
	"oba-twilio/localization"
	"oba-twilio/models"
)

// ErrorHandler provides centralized error handling with localization support
type ErrorHandler struct {
	LocalizationManager *localization.LocalizationManager
}

// NewErrorHandler creates a new error handler instance
func NewErrorHandler(locManager *localization.LocalizationManager) *ErrorHandler {
	return &ErrorHandler{
		LocalizationManager: locManager,
	}
}

// HandleSMSError handles errors for SMS responses with proper localization
func (e *ErrorHandler) HandleSMSError(c *gin.Context, err error, language string) {
	if err == nil {
		err = errors.New("unknown error occurred")
	}

	e.logError("SMS error", err)

	message := e.getLocalizedErrorMessage(err, language, "sms")

	c.Header("Content-Type", "text/xml")
	twiml, twimlErr := formatters.GenerateTwiMLSMS(message)
	if twimlErr != nil {
		log.Printf("Failed to generate SMS TwiML for error: %v", twimlErr)
		// Fallback to plain text error
		fallbackMsg := e.LocalizationManager.GetString("error.general", language)
		twiml, _ = formatters.GenerateTwiMLSMS(fallbackMsg)
	}

	c.String(http.StatusOK, twiml)
}

// HandleVoiceError handles errors for voice responses with proper localization
func (e *ErrorHandler) HandleVoiceError(c *gin.Context, err error, language string) {
	if err == nil {
		err = errors.New("unknown error occurred")
	}

	e.logError("Voice error", err)

	message := e.getLocalizedErrorMessage(err, language, "voice")

	c.Header("Content-Type", "text/xml")

	// Use TwiML library for voice errors
	say := &twiml.VoiceSay{
		Message:  message,
		Language: language,
	}

	if twimlResult, err := twiml.Voice([]twiml.Element{say}); err != nil {
		log.Printf("Failed to generate voice error TwiML: %v", err)
		// Fallback to simple XML
		fallback := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><Response><Say>%s</Say></Response>`, message)
		c.String(http.StatusOK, fallback)
	} else {
		c.String(http.StatusOK, twimlResult)
	}
}

// HandleValidationError handles validation errors with appropriate HTTP status
func (e *ErrorHandler) HandleValidationError(c *gin.Context, err error, channel, language string) {
	if err == nil {
		err = errors.New("validation failed")
	}

	e.logError("Validation error", err)

	message := e.LocalizationManager.GetString("error.validation_failed", language)

	c.Header("Content-Type", "text/xml")

	var twimlResult string
	var twimlErr error

	switch strings.ToLower(channel) {
	case "voice":
		say := &twiml.VoiceSay{
			Message:  message,
			Language: language,
		}
		twimlResult, twimlErr = twiml.Voice([]twiml.Element{say})
		if twimlErr != nil {
			log.Printf("Failed to generate voice TwiML for validation error: %v", twimlErr)
			// Fallback to simple XML
			twimlResult = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><Response><Say>%s</Say></Response>`, message)
		}
	default: // SMS or fallback
		twimlResult, twimlErr = formatters.GenerateTwiMLSMS(message)
		if twimlErr != nil {
			log.Printf("Failed to generate SMS TwiML for validation error: %v", twimlErr)
			twimlResult, _ = formatters.GenerateTwiMLSMS(e.LocalizationManager.GetString("error.general", language))
		}
	}

	c.String(http.StatusBadRequest, twimlResult)
}

// HandleInternalError handles internal server errors
func (e *ErrorHandler) HandleInternalError(c *gin.Context, err error, channel, language string) {
	if err == nil {
		err = errors.New("internal server error")
	}

	e.logError("Internal error", err)

	message := e.LocalizationManager.GetString("error.internal_error", language)

	c.Header("Content-Type", "text/xml")

	var twimlResult string
	var twimlErr error

	switch strings.ToLower(channel) {
	case "voice":
		say := &twiml.VoiceSay{
			Message:  message,
			Language: language,
		}
		twimlResult, twimlErr = twiml.Voice([]twiml.Element{say})
		if twimlErr != nil {
			log.Printf("Failed to generate voice TwiML for internal error: %v", twimlErr)
			// Fallback to simple XML
			twimlResult = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><Response><Say>%s</Say></Response>`, message)
		}
	default: // SMS or fallback
		twimlResult, twimlErr = formatters.GenerateTwiMLSMS(message)
		if twimlErr != nil {
			log.Printf("Failed to generate SMS TwiML for internal error: %v", twimlErr)
			twimlResult, _ = formatters.GenerateTwiMLSMS(e.LocalizationManager.GetString("error.general", language))
		}
	}

	c.String(http.StatusInternalServerError, twimlResult)
}

// getLocalizedErrorMessage gets a localized error message based on the error type
func (e *ErrorHandler) getLocalizedErrorMessage(err error, language, channel string) string {
	// Check if it's an AppError
	if appErr, ok := err.(*models.AppError); ok {
		key, args := e.mapAppErrorToUserMessage(appErr, language)
		if len(args) > 0 {
			return e.LocalizationManager.GetString(key, language, args...)
		}
		return e.LocalizationManager.GetString(key, language)
	}

	// Handle specific error patterns for backwards compatibility
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "connection") {
		return e.LocalizationManager.GetString("error.api_timeout", language)
	}

	if strings.Contains(errMsg, "invalid") && strings.Contains(errMsg, "stop") {
		return e.LocalizationManager.GetString("error.invalid_stop_id", language)
	}

	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no stops") {
		return e.LocalizationManager.GetString("error.stop_not_found", language)
	}

	if strings.Contains(errMsg, "service") && strings.Contains(errMsg, "unavailable") {
		return e.LocalizationManager.GetString("error.service_unavailable", language)
	}

	// Default to general error message
	return e.LocalizationManager.GetString("error.general", language)
}

// mapAppErrorToUserMessage maps AppError codes to localization keys and arguments
func (e *ErrorHandler) mapAppErrorToUserMessage(appErr *models.AppError, language string) (string, []interface{}) {
	switch appErr.Code {
	case models.ErrorCodeAPITimeout:
		return "error.api_timeout", nil

	case models.ErrorCodeInvalidStopID:
		return "error.invalid_stop_id", nil

	case models.ErrorCodeStopNotFound:
		// Try to extract stop ID from error details for better user message
		if stopID := ExtractStopIDFromError(appErr.Details); stopID != "" {
			return "error.stop_not_found", []interface{}{stopID}
		}
		return "error.stop_not_found", []interface{}{"the requested stop"}

	case models.ErrorCodeServiceUnavailable:
		return "error.service_unavailable", nil

	case models.ErrorCodeInvalidResponse:
		return "error.service_unavailable", nil // User-friendly message for technical issues

	case models.ErrorCodeValidationFailed:
		return "error.validation_failed", nil

	case models.ErrorCodeNetworkError:
		return "error.network_error", nil

	case models.ErrorCodeInternalError:
		return "error.internal_error", nil

	default:
		return "error.general", nil
	}
}

// ExtractStopIDFromError extracts stop ID from error details for better user messages
func ExtractStopIDFromError(details string) string {
	// Pattern to match stop IDs in error messages
	// Matches: "Stop 'ID' does not exist" or "Stop ID 'ID' is not..."
	patterns := []string{
		`Stop '([^']+)' does not exist`,
		`Stop ID '([^']+)' is not`,
		`stop '([^']+)'`,
		`ID '([^']+)'`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(details); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// logError logs error details for debugging and monitoring
func (e *ErrorHandler) logError(context string, err error) {
	if err == nil {
		return
	}

	if appErr, ok := err.(*models.AppError); ok {
		log.Printf("%s: AppError[%s] %s - %s", context, appErr.Code, appErr.Message, appErr.Details)
		if appErr.Cause != nil {
			log.Printf("%s: Caused by: %v", context, appErr.Cause)
		}
	} else {
		log.Printf("%s: %v", context, err)
	}
}

// IsRetryableError determines if an error is retryable by the user
func (e *ErrorHandler) IsRetryableError(err error) bool {
	if appErr, ok := err.(*models.AppError); ok {
		switch appErr.Code {
		case models.ErrorCodeAPITimeout, models.ErrorCodeNetworkError, models.ErrorCodeServiceUnavailable:
			return true
		default:
			return false
		}
	}

	// Check for common retryable error patterns
	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{"timeout", "connection", "network", "unavailable", "temporary"}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// GetErrorMetrics returns basic error metrics for monitoring
func (e *ErrorHandler) GetErrorMetrics(err error) map[string]interface{} {
	metrics := map[string]interface{}{
		"retryable": e.IsRetryableError(err),
	}

	if appErr, ok := err.(*models.AppError); ok {
		metrics["error_code"] = string(appErr.Code)
		metrics["has_details"] = appErr.Details != ""
		metrics["has_cause"] = appErr.Cause != nil
	} else {
		metrics["error_code"] = "GENERIC_ERROR"
		metrics["has_details"] = false
		metrics["has_cause"] = false
	}

	return metrics
}
