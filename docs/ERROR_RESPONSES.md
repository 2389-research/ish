# Standardized Error Response Format

## Overview

All HTTP handlers across all plugins must return consistent, standardized error responses. This document defines the standard error response structure and provides migration examples.

## Standard Error Response Format

All error responses follow this JSON structure:

```json
{
  "code": "error_code",
  "message": "Human-readable error message",
  "status": 400,
  "field": "optional.field.name",
  "details": "Optional additional context"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | string | Yes | Machine-readable error code (e.g., `invalid_request`, `not_found`) |
| `message` | string | Yes | Human-readable error message for developers/users |
| `status` | integer | Yes | HTTP status code (e.g., 400, 404, 500) |
| `field` | string | No | Optional field name that caused the error (for validation errors) |
| `details` | string | No | Optional additional context or debugging information |

## Error Response Code Reference

### Client Errors (4xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_request` | 400 | Generic invalid request |
| `invalid_request_body` | 400 | Request body is malformed or unparseable |
| `missing_field` | 400 | Required field is missing |
| `validation_failed` | 400 | Field validation failed |
| `not_found` | 404 | Resource not found |
| `unauthorized` | 401 | Authentication required or failed |
| `forbidden` | 403 | Authenticated but insufficient permissions |
| `conflict` | 409 | Request conflicts with existing resource |

### Server Errors (5xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `internal_error` | 500 | Generic internal server error |
| `database_error` | 500 | Database operation failed |
| `service_unavailable` | 503 | Service temporarily unavailable |
| `not_implemented` | 501 | Feature not implemented |

## Helper Functions

All handlers should use the helper functions from `internal/errors` package:

### WriteError

Simple error with code and message:

```go
import "github.com/2389/ish/internal/errors"

// Basic error response
errors.WriteError(w, http.StatusBadRequest, errors.ErrInvalidBody, "Request body is malformed")
```

**Output:**
```json
{
  "code": "invalid_request_body",
  "message": "Request body is malformed",
  "status": 400
}
```

### WriteErrorWithField

Error with field reference (validation errors):

```go
errors.WriteErrorWithField(
    w,
    http.StatusBadRequest,
    errors.ErrMissingField,
    "From email is required",
    "from.email",
)
```

**Output:**
```json
{
  "code": "missing_field",
  "message": "From email is required",
  "status": 400,
  "field": "from.email"
}
```

### WriteErrorWithDetails

Error with additional context:

```go
errors.WriteErrorWithDetails(
    w,
    http.StatusInternalServerError,
    errors.ErrDatabaseError,
    "Failed to save record",
    "Connection timeout after 30s",
)
```

**Output:**
```json
{
  "code": "database_error",
  "message": "Failed to save record",
  "status": 500,
  "details": "Connection timeout after 30s"
}
```

## Migration Guide

### Before: Discord Plugin

```go
// Old format - inconsistent structure
func writeError(w http.ResponseWriter, code int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": message,
        "code":    code,  // Using HTTP status code as error code
    })
}

// Usage
writeError(w, 500, "Failed to create message")
```

**Response:**
```json
{
  "message": "Failed to create message",
  "code": 500
}
```

### After: Using Standardized Format

```go
import "github.com/2389/ish/internal/errors"

// Usage
errors.WriteError(w, http.StatusInternalServerError, errors.ErrInternal, "Failed to create message")
```

**Response:**
```json
{
  "code": "internal_error",
  "message": "Failed to create message",
  "status": 500
}
```

### Before: SendGrid Plugin

```go
// Old format - custom field for validation errors
func writeError(w http.ResponseWriter, status int, message, field string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": message,
        // Only sometimes included the field, no error code
    })
}

// Usage
writeError(w, http.StatusBadRequest, "from email is required", "from.email")
```

**Response:**
```json
{
  "message": "from email is required"
}
```

### After: Using Standardized Format

```go
import "github.com/2389/ish/internal/errors"

// Usage - now with proper error code and field
errors.WriteErrorWithField(
    w,
    http.StatusBadRequest,
    errors.ErrMissingField,
    "From email is required",
    "from.email",
)
```

**Response:**
```json
{
  "code": "missing_field",
  "message": "From email is required",
  "status": 400,
  "field": "from.email"
}
```

### Before: GitHub Plugin

```go
// Old format - no error code at all
func writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": message,
    })
}

// Usage
writeError(w, http.StatusNotFound, "repository not found")
```

**Response:**
```json
{
  "message": "repository not found"
}
```

### After: Using Standardized Format

```go
import "github.com/2389/ish/internal/errors"

// Usage
errors.WriteError(w, http.StatusNotFound, errors.ErrNotFound, "Repository not found")
```

**Response:**
```json
{
  "code": "not_found",
  "message": "Repository not found",
  "status": 404
}
```

### Before: Twilio Plugin

```go
// Old format - custom error code mapping
func writeError(w http.ResponseWriter, statusCode, errorCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "code":    errorCode,  // Numeric Twilio error codes
        "message": message,
        "status":  statusCode,
    })
}

// Usage
writeError(w, http.StatusBadRequest, 21602, "Missing required parameter")
```

**Response:**
```json
{
  "code": 21602,
  "message": "Missing required parameter",
  "status": 400
}
```

### After: Using Standardized Format

```go
import "github.com/2389/ish/internal/errors"

// Usage - mapping Twilio codes to our standard codes
// Create a helper if Twilio-specific codes are needed
errors.WriteErrorWithField(
    w,
    http.StatusBadRequest,
    errors.ErrMissingField,
    "Missing required parameter: To, From, or Body",
    "to|from|body",
)
```

**Response:**
```json
{
  "code": "missing_field",
  "message": "Missing required parameter: To, From, or Body",
  "status": 400,
  "field": "to|from|body"
}
```

### Before: OAuth Plugin

```go
// Old format - using http.Error which doesn't support custom codes
http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)

// Mixed approach in other handlers
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
    "access_token":  accessToken,
    // No error handling on encoding failures
})
```

**Response (from http.Error):**
```
Invalid redirect_uri
```

### After: Using Standardized Format

```go
import "github.com/2389/ish/internal/errors"

// Usage
errors.WriteErrorWithField(
    w,
    http.StatusBadRequest,
    errors.ErrValidationFailed,
    "Invalid redirect_uri",
    "redirect_uri",
)
```

**Response:**
```json
{
  "code": "validation_failed",
  "message": "Invalid redirect_uri",
  "status": 400,
  "field": "redirect_uri"
}
```

### Before: Home Assistant Plugin

```go
// Old format - plain text error with http.Error
http.Error(w, "Unauthorized", http.StatusUnauthorized)
http.Error(w, "Invalid entity ID format. Must match pattern: domain.entity_name", http.StatusBadRequest)
```

**Response:**
```
Unauthorized
```

### After: Using Standardized Format

```go
import "github.com/2389/ish/internal/errors"

// For authentication errors
errors.WriteError(
    w,
    http.StatusUnauthorized,
    errors.ErrUnauthorized,
    "Authentication required or invalid token",
)

// For validation errors with details
errors.WriteErrorWithDetails(
    w,
    http.StatusBadRequest,
    errors.ErrValidationFailed,
    "Invalid entity ID format",
    "Must match pattern: domain.entity_name",
)
```

**Response:**
```json
{
  "code": "unauthorized",
  "message": "Authentication required or invalid token",
  "status": 401
}
```

## Implementation Checklist

For each plugin that needs updating:

- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Remove custom `writeError` functions
- [ ] Replace all `http.Error()` calls with `errors.WriteError()`
- [ ] Replace all custom error response encoding with appropriate helper
- [ ] For validation errors, use `WriteErrorWithField`
- [ ] For complex errors, use `WriteErrorWithDetails`
- [ ] Test that error responses are valid JSON
- [ ] Update error handling in tests to match new format

## Testing Error Responses

Example test for validating error response format:

```go
func TestErrorResponse(t *testing.T) {
    req, _ := http.NewRequest("GET", "/invalid", nil)
    w := httptest.NewRecorder()

    errors.WriteError(w, http.StatusNotFound, errors.ErrNotFound, "Resource not found")

    // Verify structure
    var resp map[string]interface{}
    json.NewDecoder(w.Body).Decode(&resp)

    assert.Equal(t, "not_found", resp["code"])
    assert.Equal(t, "Resource not found", resp["message"])
    assert.Equal(t, float64(404), resp["status"])
    assert.Equal(t, http.StatusNotFound, w.Code)
}
```

## Backwards Compatibility Considerations

Some plugins (like Twilio) may need to maintain backwards compatibility with existing error codes while using the standardized structure. In these cases:

1. Include the plugin-specific error code in the `details` field
2. Use a standard error code in the `code` field
3. Document the mapping in plugin-specific error handling

Example:
```go
errors.WriteErrorWithDetails(
    w,
    http.StatusBadRequest,
    errors.ErrMissingField,
    "Missing required parameter",
    "twilio_error_code: 21602",
)
```
