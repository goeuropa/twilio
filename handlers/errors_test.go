package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"oba-twilio/localization"
	"oba-twilio/models"
)

func TestNewErrorHandler(t *testing.T) {
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	assert.NotNil(t, errorHandler)
	assert.Equal(t, locManager, errorHandler.LocalizationManager)
}

func TestMapAppErrorToUserMessage(t *testing.T) {
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name         string
		appError     *models.AppError
		language     string
		expectedKey  string
		expectedArgs []interface{}
	}{
		{
			name: "API timeout error",
			appError: &models.AppError{
				Code:    models.ErrorCodeAPITimeout,
				Message: "timeout occurred",
				Details: "connection timeout after 30s",
			},
			language:     "en-US",
			expectedKey:  "error.api_timeout",
			expectedArgs: nil,
		},
		{
			name: "Invalid stop ID error",
			appError: &models.AppError{
				Code:    models.ErrorCodeInvalidStopID,
				Message: "invalid format",
				Details: "Stop ID 'abc' is not in the expected format",
			},
			language:     "en-US",
			expectedKey:  "error.invalid_stop_id",
			expectedArgs: nil,
		},
		{
			name: "Stop not found error",
			appError: &models.AppError{
				Code:    models.ErrorCodeStopNotFound,
				Message: "stop not found",
				Details: "Stop '12345' does not exist",
			},
			language:     "en-US",
			expectedKey:  "error.stop_not_found",
			expectedArgs: []interface{}{"12345"},
		},
		{
			name: "Service unavailable error",
			appError: &models.AppError{
				Code:    models.ErrorCodeServiceUnavailable,
				Message: "service down",
				Details: "maintenance mode",
			},
			language:     "en-US",
			expectedKey:  "error.service_unavailable",
			expectedArgs: nil,
		},
		{
			name: "Network error",
			appError: &models.AppError{
				Code:    models.ErrorCodeNetworkError,
				Message: "network issue",
				Details: "DNS resolution failed",
			},
			language:     "en-US",
			expectedKey:  "error.network_error",
			expectedArgs: nil,
		},
		{
			name: "Internal error",
			appError: &models.AppError{
				Code:    models.ErrorCodeInternalError,
				Message: "internal issue",
				Details: "unexpected error",
			},
			language:     "en-US",
			expectedKey:  "error.internal_error",
			expectedArgs: nil,
		},
		{
			name: "Unknown error code",
			appError: &models.AppError{
				Code:    "UNKNOWN_ERROR",
				Message: "unknown issue",
				Details: "unexpected error type",
			},
			language:     "en-US",
			expectedKey:  "error.general",
			expectedArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, args := errorHandler.mapAppErrorToUserMessage(tt.appError, tt.language)
			assert.Equal(t, tt.expectedKey, key)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestExtractStopIDFromError(t *testing.T) {
	tests := []struct {
		name     string
		details  string
		expected string
	}{
		{
			name:     "Extract from stop not found details",
			details:  "Stop '12345' does not exist",
			expected: "12345",
		},
		{
			name:     "Extract from invalid stop ID details",
			details:  "Stop ID 'abc123' is not in the expected format",
			expected: "abc123",
		},
		{
			name:     "No stop ID in details",
			details:  "Generic error message",
			expected: "",
		},
		{
			name:     "Empty details",
			details:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStopIDFromError(tt.details)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleSMSError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name           string
		err            error
		language       string
		expectedStatus int
		containsText   string
	}{
		{
			name: "AppError - API timeout",
			err: &models.AppError{
				Code:    models.ErrorCodeAPITimeout,
				Message: "timeout",
				Details: "connection timeout",
			},
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "OneBusAway service is temporarily unavailable",
		},
		{
			name: "AppError - Stop not found",
			err: &models.AppError{
				Code:    models.ErrorCodeStopNotFound,
				Message: "not found",
				Details: "Stop '12345' does not exist",
			},
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "Sorry, no stops found with ID 12345",
		},
		{
			name:           "Generic error",
			err:            errors.New("unexpected error"),
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "An unexpected error occurred",
		},
		{
			name:           "Nil error",
			err:            nil,
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "An unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			errorHandler.HandleSMSError(c, tt.err, tt.language)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "text/xml")
			assert.Contains(t, w.Body.String(), tt.containsText)
		})
	}
}

func TestHandleVoiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name           string
		err            error
		language       string
		expectedStatus int
		containsText   string
	}{
		{
			name: "AppError - Service unavailable",
			err: &models.AppError{
				Code:    models.ErrorCodeServiceUnavailable,
				Message: "service down",
				Details: "maintenance",
			},
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "OneBusAway service is currently unavailable",
		},
		{
			name: "AppError - Invalid stop ID",
			err: &models.AppError{
				Code:    models.ErrorCodeInvalidStopID,
				Message: "invalid",
				Details: "Stop ID 'abc' is not in the expected format",
			},
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "Invalid stop ID",
		},
		{
			name:           "Generic error",
			err:            errors.New("unexpected error"),
			language:       "en-US",
			expectedStatus: http.StatusOK,
			containsText:   "An unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			errorHandler.HandleVoiceError(c, tt.err, tt.language)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "text/xml")
			assert.Contains(t, w.Body.String(), tt.containsText)
		})
	}
}

func TestHandleValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name           string
		err            error
		channel        string
		language       string
		expectedStatus int
	}{
		{
			name:           "SMS validation error",
			err:            errors.New("validation failed"),
			channel:        "sms",
			language:       "en-US",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Voice validation error",
			err:            errors.New("validation failed"),
			channel:        "voice",
			language:       "en-US",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid channel defaults to SMS",
			err:            errors.New("validation failed"),
			channel:        "invalid",
			language:       "en-US",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			errorHandler.HandleValidationError(c, tt.err, tt.channel, tt.language)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "text/xml")
		})
	}
}

func TestHandleInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name           string
		err            error
		channel        string
		language       string
		expectedStatus int
	}{
		{
			name:           "SMS internal error",
			err:            errors.New("internal error"),
			channel:        "sms",
			language:       "en-US",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Voice internal error",
			err:            errors.New("internal error"),
			channel:        "voice",
			language:       "en-US",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			errorHandler.HandleInternalError(c, tt.err, tt.channel, tt.language)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "text/xml")
		})
	}
}

func TestGetLocalizedErrorMessage(t *testing.T) {
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name           string
		err            error
		language       string
		channel        string
		expectedString string
	}{
		{
			name: "AppError - API timeout",
			err: &models.AppError{
				Code:    models.ErrorCodeAPITimeout,
				Message: "timeout",
				Details: "connection timeout",
			},
			language:       "en-US",
			channel:        "sms",
			expectedString: "OneBusAway service is temporarily unavailable",
		},
		{
			name: "AppError - Stop not found with extracted ID",
			err: &models.AppError{
				Code:    models.ErrorCodeStopNotFound,
				Message: "not found",
				Details: "Stop '12345' does not exist",
			},
			language:       "en-US",
			channel:        "sms",
			expectedString: "Sorry, no stops found with ID 12345",
		},
		{
			name:           "Generic timeout error",
			err:            errors.New("connection timeout occurred"),
			language:       "en-US",
			channel:        "sms",
			expectedString: "OneBusAway service is temporarily unavailable",
		},
		{
			name:           "Generic invalid stop error",
			err:            errors.New("invalid stop ID provided"),
			language:       "en-US",
			channel:        "sms",
			expectedString: "Invalid stop ID",
		},
		{
			name:           "Generic not found error",
			err:            errors.New("no stops found for query"),
			language:       "en-US",
			channel:        "sms",
			expectedString: "Sorry, no stops found with ID %s",
		},
		{
			name:           "Generic service error",
			err:            errors.New("service unavailable at this time"),
			language:       "en-US",
			channel:        "sms",
			expectedString: "OneBusAway service is currently unavailable",
		},
		{
			name:           "Unknown error",
			err:            errors.New("unexpected system error"),
			language:       "en-US",
			channel:        "sms",
			expectedString: "An unexpected error occurred",
		},
		{
			name:           "Spanish language - API timeout",
			err:            errors.New("timeout occurred"),
			language:       "es-US",
			channel:        "sms",
			expectedString: "El servicio OneBusAway no está disponible temporalmente",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errorHandler.getLocalizedErrorMessage(tt.err, tt.language, tt.channel)
			assert.Contains(t, result, tt.expectedString)
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name: "AppError - API timeout (retryable)",
			err: &models.AppError{
				Code: models.ErrorCodeAPITimeout,
			},
			retryable: true,
		},
		{
			name: "AppError - Network error (retryable)",
			err: &models.AppError{
				Code: models.ErrorCodeNetworkError,
			},
			retryable: true,
		},
		{
			name: "AppError - Invalid stop ID (not retryable)",
			err: &models.AppError{
				Code: models.ErrorCodeInvalidStopID,
			},
			retryable: false,
		},
		{
			name:      "Generic timeout error (retryable)",
			err:       errors.New("connection timeout"),
			retryable: true,
		},
		{
			name:      "Generic network error (retryable)",
			err:       errors.New("network connection failed"),
			retryable: true,
		},
		{
			name:      "Generic unavailable error (retryable)",
			err:       errors.New("service unavailable"),
			retryable: true,
		},
		{
			name:      "Non-retryable error",
			err:       errors.New("invalid input format"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errorHandler.IsRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestGetErrorMetrics(t *testing.T) {
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	tests := []struct {
		name               string
		err                error
		expectedErrorCode  string
		expectedRetryable  bool
		expectedHasDetails bool
		expectedHasCause   bool
	}{
		{
			name: "AppError with details and cause",
			err: &models.AppError{
				Code:    models.ErrorCodeAPITimeout,
				Message: "timeout",
				Details: "connection timeout after 30s",
				Cause:   errors.New("underlying error"),
			},
			expectedErrorCode:  "API_TIMEOUT",
			expectedRetryable:  true,
			expectedHasDetails: true,
			expectedHasCause:   true,
		},
		{
			name: "AppError without details or cause",
			err: &models.AppError{
				Code:    models.ErrorCodeInvalidStopID,
				Message: "invalid format",
			},
			expectedErrorCode:  "INVALID_STOP_ID",
			expectedRetryable:  false,
			expectedHasDetails: false,
			expectedHasCause:   false,
		},
		{
			name:               "Generic error",
			err:                errors.New("generic error"),
			expectedErrorCode:  "GENERIC_ERROR",
			expectedRetryable:  false,
			expectedHasDetails: false,
			expectedHasCause:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := errorHandler.GetErrorMetrics(tt.err)

			assert.Equal(t, tt.expectedErrorCode, metrics["error_code"])
			assert.Equal(t, tt.expectedRetryable, metrics["retryable"])
			assert.Equal(t, tt.expectedHasDetails, metrics["has_details"])
			assert.Equal(t, tt.expectedHasCause, metrics["has_cause"])
		})
	}
}

func TestLogError(t *testing.T) {
	locManager := createTestLocalizationManager()
	errorHandler := NewErrorHandler(locManager)

	// Test that logError doesn't panic with various inputs
	errorHandler.logError("test context", errors.New("test error"))
	errorHandler.logError("test context", nil)
	errorHandler.logError("", errors.New("test error"))
}

// Helper function to create a test localization manager
func createTestLocalizationManager() *localization.LocalizationManager {
	strings := map[string]map[string]string{
		"en-US": {
			"error.api_timeout":         "OneBusAway service is temporarily unavailable. Please try again in a moment.",
			"error.invalid_stop_id":     "Invalid stop ID. Please provide a valid stop ID.",
			"error.stop_not_found":      "Sorry, no stops found with ID %s. Please check and try again.",
			"error.service_unavailable": "OneBusAway service is currently unavailable. Please try again later.",
			"error.network_error":       "Network connection error. Please check your connection and try again.",
			"error.internal_error":      "An internal error occurred. Please try again.",
			"error.validation_failed":   "Invalid input. Please check your request and try again.",
			"error.general":             "An unexpected error occurred. Please try again.",
			"error.invalid_request":     "Invalid request format. Please try again.",
		},
		"es-US": {
			"error.api_timeout":         "El servicio OneBusAway no está disponible temporalmente. Inténtelo de nuevo en un momento.",
			"error.invalid_stop_id":     "ID de parada inválido. Proporcione un ID de parada válido.",
			"error.stop_not_found":      "Lo siento, no se encontraron paradas con ID %s. Verifique e inténtelo de nuevo.",
			"error.service_unavailable": "El servicio OneBusAway no está disponible actualmente. Inténtelo de nuevo más tarde.",
			"error.network_error":       "Error de conexión de red. Verifique su conexión e inténtelo de nuevo.",
			"error.internal_error":      "Ocurrió un error interno. Inténtelo de nuevo.",
			"error.validation_failed":   "Entrada inválida. Verifique su solicitud e inténtelo de nuevo.",
			"error.general":             "Ocurrió un error inesperado. Inténtelo de nuevo.",
			"error.invalid_request":     "Formato de solicitud inválido. Inténtelo de nuevo.",
		},
	}

	return localization.NewTestManagerWithStrings(strings, []string{"en-US", "es-US"})
}
