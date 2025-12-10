// ABOUTME: Standardized error response types and helpers for HTTP handlers
// ABOUTME: Provides consistent error formatting across all plugins

package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standardized error response structure used across all plugins.
// This ensures consistent error handling and makes it easier for clients to parse errors.
//
// Usage:
//   WriteError(w, http.StatusBadRequest, "invalid_request", "The request body is malformed")
type ErrorResponse struct {
	Code    string `json:"code"`             // Machine-readable error code (e.g., "invalid_request", "not_found")
	Message string `json:"message"`          // Human-readable error message
	Status  int    `json:"status"`           // HTTP status code
	Field   string `json:"field,omitempty"`  // Optional: field that caused the error (for validation errors)
	Details string `json:"details,omitempty"` // Optional: additional error details
}

// WriteError writes a standardized error response to the HTTP response writer.
// This function ensures all error responses have a consistent format.
//
// Parameters:
//   - w: http.ResponseWriter to write the response to
//   - status: HTTP status code (e.g., http.StatusBadRequest)
//   - code: Machine-readable error code (e.g., "invalid_request")
//   - message: Human-readable error message
//
// Example:
//   WriteError(w, http.StatusBadRequest, "invalid_email", "Email address is invalid")
func WriteError(w http.ResponseWriter, status int, code, message string) {
	writeErrorResponse(w, ErrorResponse{
		Code:    code,
		Message: message,
		Status:  status,
	})
}

// WriteErrorWithField writes a standardized error response with a field reference.
// Use this for validation errors where you want to indicate which field caused the error.
//
// Parameters:
//   - w: http.ResponseWriter to write the response to
//   - status: HTTP status code (e.g., http.StatusBadRequest)
//   - code: Machine-readable error code
//   - message: Human-readable error message
//   - field: Field name that caused the error
//
// Example:
//   WriteErrorWithField(w, http.StatusBadRequest, "missing_field", "From email is required", "from.email")
func WriteErrorWithField(w http.ResponseWriter, status int, code, message, field string) {
	writeErrorResponse(w, ErrorResponse{
		Code:    code,
		Message: message,
		Status:  status,
		Field:   field,
	})
}

// WriteErrorWithDetails writes a standardized error response with additional details.
// Use this when you need to provide extra context about the error.
//
// Parameters:
//   - w: http.ResponseWriter to write the response to
//   - status: HTTP status code
//   - code: Machine-readable error code
//   - message: Human-readable error message
//   - details: Additional error details or context
//
// Example:
//   WriteErrorWithDetails(w, http.StatusInternalServerError, "database_error", "Failed to save record", "connection timeout after 30s")
func WriteErrorWithDetails(w http.ResponseWriter, status int, code, message, details string) {
	writeErrorResponse(w, ErrorResponse{
		Code:    code,
		Message: message,
		Status:  status,
		Details: details,
	})
}

// writeErrorResponse is a helper that serializes and writes the ErrorResponse to the ResponseWriter.
func writeErrorResponse(w http.ResponseWriter, resp ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	json.NewEncoder(w).Encode(resp)
}

// CommonErrorCodes defines standard error codes used across plugins
const (
	// Client errors (4xx)
	ErrInvalidRequest     = "invalid_request"
	ErrInvalidBody        = "invalid_request_body"
	ErrMissingField       = "missing_field"
	ErrValidationFailed   = "validation_failed"
	ErrNotFound           = "not_found"
	ErrUnauthorized       = "unauthorized"
	ErrForbidden          = "forbidden"
	ErrConflict           = "conflict"

	// Server errors (5xx)
	ErrInternal           = "internal_error"
	ErrDatabaseError      = "database_error"
	ErrServiceUnavailable = "service_unavailable"
	ErrNotImplemented     = "not_implemented"
)
