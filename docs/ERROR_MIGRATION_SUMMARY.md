# Error Response Migration Summary

## Quick Reference

**Total Files to Migrate:** 13
**Total Error Handlers:** 98+
**Different Formats Found:** 7
**Standard Format:** See `/internal/errors/errors.go`

---

## Files Requiring Updates

### By Priority

#### Priority 1: High (Custom Helper Functions - 98 handlers)
1. **plugins/discord/handlers.go** - 20 error calls
   - Current: `writeError(code, msg)`
   - After: Use `errors.WriteError()`
   - Status: Need migration

2. **plugins/github/handlers.go** - 40 error calls
   - Current: `writeError(status, msg)`
   - After: Use `errors.WriteError()` or `errors.WriteErrorWithField()`
   - Status: Need migration

3. **plugins/sendgrid/handlers.go** - 15 error calls
   - Current: `writeError(status, msg, field)`
   - After: Use `errors.WriteError()` or `errors.WriteErrorWithField()`
   - Status: Need migration

4. **plugins/twilio/handlers.go** - 10 error calls
   - Current: `writeError(status, code, msg)` with numeric codes
   - After: Use `errors.WriteError()` or `errors.WriteErrorWithDetails()`
   - Status: Need migration

5. **plugins/github/plugin.go** - 5 error calls
   - Current: `writeError(status, msg)`
   - After: Use `errors.WriteError()`
   - Status: Need migration

6. **plugins/sendgrid/plugin.go** - 8 error calls
   - Current: `writeError(status, msg, field)`
   - After: Use `errors.WriteError()` or `errors.WriteErrorWithField()`
   - Status: Need migration

#### Priority 2: Medium (Mixed Patterns - 18 handlers)
7. **plugins/oauth/handlers.go** - 8 error calls
   - Current: `http.Error()` and custom JSON response
   - After: Use `errors.WriteError()` functions
   - Status: Need migration

8. **plugins/oauth/plugin.go** - 6 error calls
   - Current: `http.Error()`
   - After: Use `errors.WriteError()`
   - Status: Need migration

9. **plugins/twilio/plugin.go** - 4 error calls
   - Current: `http.Error()`
   - After: Use `errors.WriteError()`
   - Status: Need migration

#### Priority 3: Lower (Plain http.Error - 35+ handlers)
10. **plugins/homeassistant/plugin.go** - 20+ error calls
    - Current: `http.Error()` with plain text
    - After: Use `errors.WriteError()`
    - Status: Need migration

11. **plugins/google/gmail.go** - 10 error calls
    - Current: `writeError(code, msg, status)` (custom format)
    - After: Use `errors.WriteError()`
    - Status: Need migration

12. **plugins/google/discovery.go** - 2 error calls
    - Current: `http.Error()`
    - After: Use `errors.WriteError()`
    - Status: Need migration

13. **internal/admin/handlers.go** - 3 error calls
    - Current: `http.Error()`
    - After: Use `errors.WriteError()`
    - Status: Need migration

---

## Standard Error Response Format

All errors must return JSON with this structure:

```json
{
  "code": "error_code",
  "message": "Human readable message",
  "status": 400,
  "field": "optional.field.name",
  "details": "optional additional context"
}
```

### Helper Functions Available

```go
import "github.com/2389/ish/internal/errors"

// Simple error
errors.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")

// Validation error with field
errors.WriteErrorWithField(w, http.StatusBadRequest, "missing_field", "Email is required", "email")

// Error with additional context
errors.WriteErrorWithDetails(w, http.StatusInternalServerError, "database_error", "Save failed", "Connection timeout")
```

---

## Standard Error Codes

```
Client Errors (4xx):
- invalid_request       → 400 Bad Request
- invalid_request_body  → 400 Bad Request
- missing_field         → 400 Bad Request
- validation_failed     → 400 Bad Request
- not_found             → 404 Not Found
- unauthorized          → 401 Unauthorized
- forbidden             → 403 Forbidden
- conflict              → 409 Conflict

Server Errors (5xx):
- internal_error        → 500 Internal Server Error
- database_error        → 500 Internal Server Error
- service_unavailable   → 503 Service Unavailable
- not_implemented       → 501 Not Implemented
```

---

## Migration Checklist per Plugin

### Discord (`plugins/discord/handlers.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Remove `writeError()` function
- [ ] Replace 20 `writeError()` calls with `errors.WriteError()`
- [ ] Run tests - verify JSON responses valid
- [ ] No `http.Error()` calls should remain

### GitHub (`plugins/github/handlers.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Remove `writeError()` function
- [ ] Replace 40 `writeError()` calls
  - Use `errors.WriteError()` for general errors
  - Use `errors.WriteErrorWithField()` for validation errors
- [ ] Run tests - verify 50+ error cases
- [ ] No `http.Error()` calls should remain

### SendGrid (`plugins/sendgrid/handlers.go` and `plugin.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Remove `writeError()` function from both files
- [ ] Replace 23 `writeError()` calls
  - Use `errors.WriteErrorWithField()` for validation errors with field
  - Use `errors.WriteError()` for general errors
- [ ] Run tests
- [ ] No `http.Error()` calls should remain

### Twilio (`plugins/twilio/handlers.go` and `plugin.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Remove `writeError()` function from handlers.go
- [ ] Replace 14 `writeError()` calls
  - Map Twilio error codes to standard codes
  - Consider using `details` field for Twilio-specific codes if needed
- [ ] Replace 4 `http.Error()` calls in plugin.go with `errors.WriteError()`
- [ ] Run tests
- [ ] No `http.Error()` calls should remain

### OAuth (`plugins/oauth/handlers.go` and `plugin.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Replace 8 `http.Error()` calls in handlers.go with `errors.WriteError()`
- [ ] Replace 6 `http.Error()` calls in plugin.go with `errors.WriteError()`
- [ ] Run tests
- [ ] No `http.Error()` calls should remain

### Home Assistant (`plugins/homeassistant/plugin.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Replace 20+ `http.Error()` calls
  - Map error messages to appropriate error codes
  - Use `errors.WriteErrorWithDetails()` for informative messages
- [ ] Run tests
- [ ] Verify JSON responses (currently returning plain text)

### Google Services (`plugins/google/gmail.go`, `discovery.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Gmail: Remove `writeError()` function, replace 10 calls
- [ ] Discovery: Replace 2 `http.Error()` calls
- [ ] Run tests
- [ ] No custom error functions should remain

### Admin (`internal/admin/handlers.go`)
- [ ] Import `github.com/2389/ish/internal/errors`
- [ ] Replace 3 `http.Error()` calls with `errors.WriteError()`
- [ ] Run tests

---

## Before/After Examples

### Discord
```go
// BEFORE
writeError(w, 500, "Failed to create message")
// Response: {"message":"Failed to create message","code":500}

// AFTER
errors.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create message")
// Response: {"code":"internal_error","message":"Failed to create message","status":500}
```

### GitHub
```go
// BEFORE
writeError(w, http.StatusNotFound, "repository not found")
// Response: {"message":"repository not found"}

// AFTER
errors.WriteError(w, http.StatusNotFound, "not_found", "Repository not found")
// Response: {"code":"not_found","message":"Repository not found","status":404}
```

### SendGrid (Validation Error)
```go
// BEFORE
writeError(w, http.StatusBadRequest, "from email is required", "from.email")
// Response: {"message":"from email is required"}

// AFTER
errors.WriteErrorWithField(w, http.StatusBadRequest, "missing_field", "From email is required", "from.email")
// Response: {"code":"missing_field","message":"From email is required","status":400,"field":"from.email"}
```

### OAuth
```go
// BEFORE
http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
// Response: Invalid redirect_uri (plain text!)

// AFTER
errors.WriteErrorWithField(w, http.StatusBadRequest, "validation_failed", "Invalid redirect_uri", "redirect_uri")
// Response: {"code":"validation_failed","message":"Invalid redirect_uri","status":400,"field":"redirect_uri"}
```

### Home Assistant
```go
// BEFORE
http.Error(w, "Unauthorized", http.StatusUnauthorized)
// Response: Unauthorized (plain text!)

// AFTER
errors.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
// Response: {"code":"unauthorized","message":"Authentication required","status":401}
```

---

## Testing Requirements

Each migrated handler needs tests to verify:

1. ✓ Correct error code returned
2. ✓ Correct HTTP status code (in header)
3. ✓ Status code included in response body
4. ✓ Message is present and clear
5. ✓ Field included (if applicable)
6. ✓ Response is valid JSON
7. ✓ Content-Type header is application/json

Example test:
```go
func TestHandlerError(t *testing.T) {
    req, _ := http.NewRequest("GET", "/test", nil)
    w := httptest.NewRecorder()

    // Trigger an error condition
    handleSomething(w, req)

    // Verify response
    assert.Equal(t, http.StatusNotFound, w.Code)
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

    var resp map[string]interface{}
    json.NewDecoder(w.Body).Decode(&resp)
    assert.Equal(t, "not_found", resp["code"])
    assert.Equal(t, float64(404), resp["status"])
}
```

---

## Documentation References

- **Detailed Format Guide:** `/docs/ERROR_RESPONSES.md`
- **Full Analysis Report:** `/docs/ERROR_RESPONSE_ANALYSIS.md`
- **Helper Functions:** `/internal/errors/errors.go`

---

## Progress Tracking

| File | Status | Priority | Handlers |
|------|--------|----------|----------|
| discord/handlers.go | Pending | High | 20 |
| github/handlers.go | Pending | High | 40 |
| github/plugin.go | Pending | High | 5 |
| sendgrid/handlers.go | Pending | High | 15 |
| sendgrid/plugin.go | Pending | High | 8 |
| twilio/handlers.go | Pending | High | 10 |
| oauth/handlers.go | Pending | Medium | 8 |
| oauth/plugin.go | Pending | Medium | 6 |
| twilio/plugin.go | Pending | Medium | 4 |
| homeassistant/plugin.go | Pending | Lower | 20+ |
| google/gmail.go | Pending | Lower | 10 |
| google/discovery.go | Pending | Lower | 2 |
| admin/handlers.go | Pending | Lower | 3 |
| **TOTAL** | **Pending** | - | **151+** |
