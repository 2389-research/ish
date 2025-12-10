# Error Response Format Analysis Report

Date: 2025-12-09
Scope: All HTTP handlers across all plugins

## Executive Summary

**Current State:** Highly inconsistent error response formats across 13 files with 8+ different error response patterns.

**Impact:**
- Clients must handle multiple error formats
- Difficult to debug API issues
- No standardized error codes
- Field validation errors inconsistently reported

**Solution:** Standardized error response struct with 3 helper functions (introduced in `internal/errors` package)

---

## Current Error Formats Found

### Format 1: Simple Message + Numeric Code (Discord)
**File:** `plugins/discord/handlers.go`
**Pattern:** Uses HTTP status code as error code field
```json
{
  "message": "string",
  "code": 400
}
```

**Issues:**
- No semantic error code (uses HTTP status)
- Cannot distinguish between different types of 400 errors
- Field validation errors not supported

**Occurrences in Discord:** ~20 calls across handlers

---

### Format 2: Message + Optional Field (SendGrid)
**File:** `plugins/sendgrid/handlers.go` and `plugins/sendgrid/plugin.go`
**Pattern:** Field parameter sometimes ignored in response
```json
{
  "message": "string"
}
```

**Issues:**
- No error code at all
- Field parameter passed but not included in response
- Inconsistent with some handlers that include field reference

**Occurrences in SendGrid:** ~15 calls

---

### Format 3: Plain Text HTTP Error (OAuth, Home Assistant, Google)
**Files:**
- `plugins/oauth/handlers.go`
- `plugins/homeassistant/plugin.go`
- `plugins/google/discovery.go`

**Pattern:** Using `http.Error()` which returns plain text
```
Invalid redirect_uri
```

**Issues:**
- Not JSON at all
- No error codes
- Clients cannot parse programmatically
- Breaks API contract

**Occurrences:** 30+ calls across multiple plugins

---

### Format 4: Message Only (GitHub)
**File:** `plugins/github/handlers.go`
**Pattern:** Simple map with just message
```json
{
  "message": "string"
}
```

**Issues:**
- No error code whatsoever
- No status field
- Cannot distinguish error types
- About 40 calls to writeError in GitHub alone

**Occurrences in GitHub:** ~40 calls

---

### Format 5: Numeric Twilio Error Code (Twilio)
**File:** `plugins/twilio/handlers.go`
**Pattern:** Three-part signature with numeric Twilio error codes
```json
{
  "code": 21602,
  "message": "string",
  "status": 400
}
```

**Issues:**
- Uses numeric codes instead of string codes
- Twilio-specific (not portable across plugins)
- Good structure but custom code scheme

**Occurrences in Twilio:** ~10 calls

---

### Format 6: Complex Google Error Format (Google Gmail)
**File:** `plugins/google/gmail.go`
**Pattern:** Four parameters with status and code fields
```go
writeError(w http.ResponseWriter, code int, message, status string)
```

**Issues:**
- Mixed HTTP status codes and string status
- Confusing API (two error representations)
- Inconsistent with other plugins

**Occurrences in Google Gmail:** ~10 calls

---

### Format 7: JSON Success Responses Without Error Handling (OAuth, various)
**Pattern:** Assumes encoding never fails
```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response) // Error ignored!
```

**Issues:**
- No error handling for encoding failures
- JSON encoding errors silently fail
- Response may be incomplete or malformed

**Occurrences:** Multiple places across OAuth and other handlers

---

## Files Needing Migration

### High Priority (Custom Helper Functions)

| File | Plugin | Helper Function | Error Count | Priority |
|------|--------|-----------------|-------------|----------|
| `plugins/discord/handlers.go` | Discord | `writeError(code, msg)` | ~20 | High |
| `plugins/github/handlers.go` | GitHub | `writeError(status, msg)` | ~40 | High |
| `plugins/sendgrid/handlers.go` | SendGrid | `writeError(status, msg, field)` | ~15 | High |
| `plugins/twilio/handlers.go` | Twilio | `writeError(status, code, msg)` | ~10 | High |
| `plugins/github/plugin.go` | GitHub | `writeError(status, msg)` | ~5 | High |
| `plugins/sendgrid/plugin.go` | SendGrid | `writeError(status, msg, field)` | ~8 | High |

### Medium Priority (Mixed http.Error and Custom Handlers)

| File | Plugin | Pattern | Error Count | Priority |
|------|--------|---------|-------------|----------|
| `plugins/oauth/handlers.go` | OAuth | `http.Error()` + custom JSON | ~8 | Medium |
| `plugins/oauth/plugin.go` | OAuth | `http.Error()` | ~6 | Medium |
| `plugins/twilio/plugin.go` | Twilio | `http.Error()` | ~4 | Medium |

### Lower Priority (Plain http.Error Calls)

| File | Plugin | Pattern | Error Count | Priority |
|------|--------|---------|-------------|----------|
| `plugins/homeassistant/plugin.go` | Home Assistant | `http.Error()` | ~20+ | Lower |
| `plugins/google/discovery.go` | Google | `http.Error()` | ~2 | Lower |
| `plugins/google/gmail.go` | Google | `writeError(custom)` | ~10 | Lower |
| `internal/admin/handlers.go` | Admin | `http.Error()` | ~3 | Lower |

---

## Migration Statistics

### Summary

| Metric | Count |
|--------|-------|
| **Total Files to Migrate** | 13 |
| **Total Error Handlers to Update** | 98+ |
| **Unique Error Response Formats** | 7 |
| **Files with Custom writeError** | 6 |
| **Files with plain http.Error** | 7 |
| **Files with JSON encoding errors** | 5+ |

### Breakdown by Error Type

| Error Type | Count | Status |
|------------|-------|--------|
| Custom `writeError()` calls | 98 | Needs migration |
| `http.Error()` calls | 35+ | Needs migration |
| JSON encoding without error handling | 15+ | Needs fix |

---

## Current Error Response Examples

### Successful Response
All handlers correctly return JSON for success:
```json
{
  "id": "123",
  "name": "example",
  "created_at": "2025-12-09T10:30:00Z"
}
```

### Error Response (Discord)
```json
{
  "message": "Failed to create message",
  "code": 500
}
```
**Problem:** HTTP code, no semantic error code

### Error Response (GitHub)
```json
{
  "message": "repository not found"
}
```
**Problem:** No code, no status

### Error Response (Home Assistant)
```
Internal server error
```
**Problem:** Plain text, not JSON

### Error Response (Twilio)
```json
{
  "code": 21602,
  "message": "Missing required parameter",
  "status": 400
}
```
**Good:** Has code, message, and status but uses numeric codes

---

## Standardized Format Benefits

### Before (Current State)
```go
// Discord
writeError(w, 500, "Failed to create message")
// Response: {"message":"Failed to create message","code":500}

// GitHub
writeError(w, http.StatusNotFound, "repository not found")
// Response: {"message":"repository not found"}

// Home Assistant
http.Error(w, "Unauthorized", http.StatusUnauthorized)
// Response: Unauthorized (plain text)

// Twilio
writeError(w, http.StatusBadRequest, 21602, "Missing parameter")
// Response: {"code":21602,"message":"Missing parameter","status":400}
```

### After (Standardized)
```go
// All handlers use same pattern
errors.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create message")
// Response: {"code":"internal_error","message":"Failed to create message","status":500}

errors.WriteError(w, http.StatusNotFound, "not_found", "Repository not found")
// Response: {"code":"not_found","message":"Repository not found","status":404}

errors.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
// Response: {"code":"unauthorized","message":"Authentication required","status":401}

errors.WriteErrorWithField(w, http.StatusBadRequest, "missing_field", "Missing parameter", "parameter_name")
// Response: {"code":"missing_field","message":"Missing parameter","status":400,"field":"parameter_name"}
```

---

## Error Code Mapping

### Standard Error Codes to Implement

| Code | HTTP Status | Use Case |
|------|------------|----------|
| `invalid_request` | 400 | Generic invalid request |
| `invalid_request_body` | 400 | Malformed JSON/body |
| `missing_field` | 400 | Required field missing |
| `validation_failed` | 400 | Field validation failed |
| `not_found` | 404 | Resource not found |
| `unauthorized` | 401 | Authentication required |
| `forbidden` | 403 | Insufficient permissions |
| `conflict` | 409 | Resource conflict |
| `internal_error` | 500 | Generic server error |
| `database_error` | 500 | Database operation failed |
| `service_unavailable` | 503 | Service down |
| `not_implemented` | 501 | Feature not implemented |

---

## Implementation Notes

### Plugin-Specific Considerations

#### Twilio
- Uses numeric error codes from Twilio API
- Consider keeping compatibility: put Twilio code in `details` field
- Example: `{"code":"missing_field","message":"...","status":400,"details":"twilio_error:21602"}`

#### OAuth
- Handles OAuth standard error responses
- Should use standard codes for consistency
- May need to map OAuth error codes to standard codes

#### Home Assistant
- Many errors use `http.Error()` with plain text
- No error codes currently supported
- Should map error types to our error codes

#### Google
- Complex error structures for different services
- Gmail, Calendar, People, Tasks handlers vary
- Need consistent mapping across all Google services

#### Discord
- Already has custom error helper
- Just needs format adjustment (remove numeric code)

---

## Testing Strategy

### Unit Tests Required

1. **Error Response Structure**
   - Verify all fields present
   - Verify JSON marshaling
   - Verify HTTP headers set correctly

2. **Error Code Coverage**
   - Each error code used at least once
   - Field validation errors include field name
   - Details field only included when provided

3. **HTTP Status Codes**
   - Correct status code returned
   - Status code matches semantic meaning
   - Status code in both header and body matches

4. **Backwards Compatibility** (if needed)
   - Fields in response (none should break existing clients)
   - JSON structure is additive only
   - No removal of required fields

### Integration Tests Required

1. **End-to-end error flows** for each plugin
2. **Error response consistency** across all handlers
3. **Field validation** with multiple field errors

---

## Migration Priority

### Phase 1 (Immediate)
Create standardized error package and helper functions.
- **Time:** ~2 hours
- **Risk:** Low (new code, no existing code affected)
- **Deliverables:** `internal/errors/errors.go`, `docs/ERROR_RESPONSES.md`

### Phase 2 (High Priority)
Migrate plugins with custom `writeError()` functions:
- Discord, GitHub, SendGrid, Twilio
- **Time:** ~4 hours total
- **Risk:** Medium (replaces existing functions, need tests)
- **Impact:** Covers ~98 error handlers

### Phase 3 (Medium Priority)
Migrate plugins using `http.Error()` with custom wrappers:
- OAuth, Home Assistant
- **Time:** ~3 hours total
- **Risk:** Medium (changes response format)

### Phase 4 (Cleanup)
Update admin handlers and remaining plain `http.Error()` calls:
- **Time:** ~1 hour
- **Risk:** Low (few handlers affected)

---

## Validation Checklist

- [ ] Error package created with 3 helper functions
- [ ] Constants defined for all standard error codes
- [ ] Documentation written with examples
- [ ] Migration guide provided for each plugin
- [ ] Tests written for error helper functions
- [ ] All error handlers updated
- [ ] No `http.Error()` calls remain in handler code
- [ ] No custom `writeError()` functions remain
- [ ] JSON encoding errors are handled
- [ ] HTTP headers set consistently
- [ ] Status code in response matches HTTP header
