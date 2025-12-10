// ABOUTME: Unit tests for standardized error response helpers
// ABOUTME: Validates error response format, JSON marshaling, and HTTP headers

package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteError tests the basic WriteError helper function
func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		code           string
		message        string
		expectedCode   string
		expectedStatus int
		expectedField  string
		expectedDetail string
	}{
		{
			name:           "bad request error",
			status:         http.StatusBadRequest,
			code:           ErrInvalidBody,
			message:        "Request body is malformed",
			expectedCode:   ErrInvalidBody,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found error",
			status:         http.StatusNotFound,
			code:           ErrNotFound,
			message:        "Resource not found",
			expectedCode:   ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unauthorized error",
			status:         http.StatusUnauthorized,
			code:           ErrUnauthorized,
			message:        "Authentication required",
			expectedCode:   ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "internal server error",
			status:         http.StatusInternalServerError,
			code:           ErrInternal,
			message:        "Internal server error",
			expectedCode:   ErrInternal,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.status, tt.code, tt.message)

			// Verify HTTP status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify Content-Type header
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			// Parse response
			var resp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify response fields
			if resp.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, resp.Code)
			}
			if resp.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, resp.Message)
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.Status)
			}
			if resp.Field != "" {
				t.Errorf("expected empty field, got %s", resp.Field)
			}
			if resp.Details != "" {
				t.Errorf("expected empty details, got %s", resp.Details)
			}
		})
	}
}

// TestWriteErrorWithField tests the WriteErrorWithField helper function
func TestWriteErrorWithField(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		code           string
		message        string
		field          string
		expectedField  string
	}{
		{
			name:          "missing email field",
			status:        http.StatusBadRequest,
			code:          ErrMissingField,
			message:       "Email is required",
			field:         "email",
			expectedField: "email",
		},
		{
			name:          "nested field validation",
			status:        http.StatusBadRequest,
			code:          ErrValidationFailed,
			message:       "Invalid phone number",
			field:         "contact.phone",
			expectedField: "contact.phone",
		},
		{
			name:          "multiple fields",
			status:        http.StatusBadRequest,
			code:          ErrValidationFailed,
			message:       "Multiple validation errors",
			field:         "email,phone,address",
			expectedField: "email,phone,address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteErrorWithField(w, tt.status, tt.code, tt.message, tt.field)

			// Verify HTTP status code
			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			// Parse response
			var resp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify field is included
			if resp.Field != tt.expectedField {
				t.Errorf("expected field %q, got %q", tt.expectedField, resp.Field)
			}
			if resp.Details != "" {
				t.Errorf("expected empty details, got %s", resp.Details)
			}
		})
	}
}

// TestWriteErrorWithDetails tests the WriteErrorWithDetails helper function
func TestWriteErrorWithDetails(t *testing.T) {
	tests := []struct {
		name             string
		status           int
		code             string
		message          string
		details          string
		expectedDetails  string
	}{
		{
			name:            "database error with details",
			status:          http.StatusInternalServerError,
			code:            ErrDatabaseError,
			message:         "Failed to save record",
			details:         "Connection timeout after 30s",
			expectedDetails: "Connection timeout after 30s",
		},
		{
			name:            "service error with detailed message",
			status:          http.StatusServiceUnavailable,
			code:            ErrServiceUnavailable,
			message:         "Service temporarily unavailable",
			details:         "Database maintenance in progress",
			expectedDetails: "Database maintenance in progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteErrorWithDetails(w, tt.status, tt.code, tt.message, tt.details)

			// Verify HTTP status code
			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			// Parse response
			var resp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify details field
			if resp.Details != tt.expectedDetails {
				t.Errorf("expected details %q, got %q", tt.expectedDetails, resp.Details)
			}
			if resp.Field != "" {
				t.Errorf("expected empty field, got %s", resp.Field)
			}
		})
	}
}

// TestErrorResponseJSON tests that all response types produce valid JSON
func TestErrorResponseJSON(t *testing.T) {
	tests := []struct {
		name        string
		writeFunc   func(w http.ResponseWriter)
		expectedLen int
	}{
		{
			name: "simple error",
			writeFunc: func(w http.ResponseWriter) {
				WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request")
			},
			expectedLen: 3, // code, message, status
		},
		{
			name: "error with field",
			writeFunc: func(w http.ResponseWriter) {
				WriteErrorWithField(w, http.StatusBadRequest, ErrMissingField, "Field required", "email")
			},
			expectedLen: 4, // code, message, status, field
		},
		{
			name: "error with details",
			writeFunc: func(w http.ResponseWriter) {
				WriteErrorWithDetails(w, http.StatusInternalServerError, ErrInternal, "Error occurred", "Details here")
			},
			expectedLen: 4, // code, message, status, details
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.writeFunc(w)

			// Verify it's valid JSON
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			// Verify required fields are present
			requiredFields := []string{"code", "message", "status"}
			for _, field := range requiredFields {
				if _, ok := resp[field]; !ok {
					t.Errorf("required field %q missing from response", field)
				}
			}
		})
	}
}

// TestErrorResponseConsistency tests that the same error always produces the same response
func TestErrorResponseConsistency(t *testing.T) {
	// Make the same error twice
	w1 := httptest.NewRecorder()
	WriteError(w1, http.StatusNotFound, ErrNotFound, "Resource not found")

	w2 := httptest.NewRecorder()
	WriteError(w2, http.StatusNotFound, ErrNotFound, "Resource not found")

	var resp1, resp2 ErrorResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	json.NewDecoder(w2.Body).Decode(&resp2)

	if resp1.Code != resp2.Code {
		t.Errorf("error code not consistent: %s != %s", resp1.Code, resp2.Code)
	}
	if resp1.Message != resp2.Message {
		t.Errorf("error message not consistent: %s != %s", resp1.Message, resp2.Message)
	}
	if resp1.Status != resp2.Status {
		t.Errorf("error status not consistent: %d != %d", resp1.Status, resp2.Status)
	}
}

// TestErrorResponseHTTPHeaders tests that correct headers are set
func TestErrorResponseHTTPHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid")

	// Verify Content-Type
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}

	// Verify status code is in header
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status code %d in header, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify body is not empty (Content-Length is set by httptest when body is written)
	if w.Body.Len() == 0 {
		t.Errorf("expected response body to be written")
	}
}

// TestErrorResponseStatusCodeMatch tests that status code in body matches HTTP header
func TestErrorResponseStatusCodeMatch(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, statusCode, "error_code", "error message")

			// Verify HTTP header status
			if w.Code != statusCode {
				t.Errorf("expected HTTP status %d, got %d", statusCode, w.Code)
			}

			// Verify body status
			var resp ErrorResponse
			json.NewDecoder(w.Body).Decode(&resp)
			if resp.Status != statusCode {
				t.Errorf("expected body status %d, got %d", statusCode, resp.Status)
			}
		})
	}
}

// TestCommonErrorCodes tests that all common error codes are defined
func TestCommonErrorCodes(t *testing.T) {
	codes := []string{
		ErrInvalidRequest,
		ErrInvalidBody,
		ErrMissingField,
		ErrValidationFailed,
		ErrNotFound,
		ErrUnauthorized,
		ErrForbidden,
		ErrConflict,
		ErrInternal,
		ErrDatabaseError,
		ErrServiceUnavailable,
		ErrNotImplemented,
	}

	for _, code := range codes {
		if code == "" {
			t.Errorf("error code constant is empty")
		}
	}
}
