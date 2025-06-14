package models

import "fmt"

// ErrorCode represents a specific error category for programmatic handling
type ErrorCode string

const (
	ErrorCodeAPITimeout         ErrorCode = "API_TIMEOUT"
	ErrorCodeInvalidStopID      ErrorCode = "INVALID_STOP_ID"
	ErrorCodeStopNotFound       ErrorCode = "STOP_NOT_FOUND"
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeInvalidResponse    ErrorCode = "INVALID_RESPONSE"
	ErrorCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrorCodeNetworkError       ErrorCode = "NETWORK_ERROR"
	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
)

// AppError represents a structured error with error code and contextual information
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// Unwrap allows errors.Is and errors.As to work with wrapped errors
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAPITimeoutError creates an error for API timeout scenarios
func NewAPITimeoutError(details string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeAPITimeout,
		Message: "OneBusAway service is temporarily unavailable",
		Details: details,
		Cause:   cause,
	}
}

// NewInvalidStopIDError creates an error for invalid stop ID format
func NewInvalidStopIDError(stopID string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeInvalidStopID,
		Message: "Invalid stop ID format",
		Details: fmt.Sprintf("Stop ID '%s' is not in the expected format", stopID),
		Cause:   cause,
	}
}

// NewStopNotFoundError creates an error when a stop cannot be found
func NewStopNotFoundError(stopID string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeStopNotFound,
		Message: "Stop not found",
		Details: fmt.Sprintf("Stop '%s' does not exist", stopID),
		Cause:   cause,
	}
}

// NewServiceUnavailableError creates an error for service unavailability
func NewServiceUnavailableError(details string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeServiceUnavailable,
		Message: "OneBusAway service is currently unavailable",
		Details: details,
		Cause:   cause,
	}
}

// NewInvalidResponseError creates an error for malformed API responses
func NewInvalidResponseError(details string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeInvalidResponse,
		Message: "Invalid response from OneBusAway API",
		Details: details,
		Cause:   cause,
	}
}

// NewValidationFailedError creates an error for validation failures
func NewValidationFailedError(details string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeValidationFailed,
		Message: "Input validation failed",
		Details: details,
		Cause:   cause,
	}
}

// NewNetworkError creates an error for network-related issues
func NewNetworkError(details string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeNetworkError,
		Message: "Network communication error",
		Details: details,
		Cause:   cause,
	}
}

// NewInternalError creates an error for internal application errors
func NewInternalError(details string, cause error) *AppError {
	return &AppError{
		Code:    ErrorCodeInternalError,
		Message: "Internal application error",
		Details: details,
		Cause:   cause,
	}
}
