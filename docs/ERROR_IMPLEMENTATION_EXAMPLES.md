# Error Response Implementation Examples

This document provides real-world code examples for each error scenario using the standardized error package.

## Import Statement

All handlers should import the errors package:

```go
import "github.com/2389/ish/internal/errors"
```

---

## Common Error Scenarios

### 1. Invalid Request Body (Malformed JSON)

**Scenario:** User sends invalid JSON

```go
func (p *MyPlugin) createItem(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name string `json:"name"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.WriteError(
            w,
            http.StatusBadRequest,
            errors.ErrInvalidBody,
            "Request body contains invalid JSON",
        )
        return
    }
}
```

**Response:**
```json
{
  "code": "invalid_request_body",
  "message": "Request body contains invalid JSON",
  "status": 400
}
```

---

### 2. Missing Required Field

**Scenario:** User doesn't provide a required field

```go
func (p *MyPlugin) createUser(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email string `json:"email"`
        Name  string `json:"name"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.WriteError(w, http.StatusBadRequest, errors.ErrInvalidBody, "Invalid JSON")
        return
    }

    if req.Email == "" {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrMissingField,
            "Email is required",
            "email",
        )
        return
    }

    if req.Name == "" {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrMissingField,
            "Name is required",
            "name",
        )
        return
    }
}
```

**Response for missing email:**
```json
{
  "code": "missing_field",
  "message": "Email is required",
  "status": 400,
  "field": "email"
}
```

---

### 3. Field Validation Error

**Scenario:** Field format is invalid

```go
func (p *SendGridPlugin) sendMail(w http.ResponseWriter, r *http.Request) {
    var req struct {
        FromEmail string `json:"from_email"`
        ToEmail   string `json:"to_email"`
    }

    json.NewDecoder(r.Body).Decode(&req)

    // Validate email format
    if _, err := mail.ParseAddress(req.FromEmail); err != nil {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrValidationFailed,
            "Invalid email format",
            "from_email",
        )
        return
    }

    if _, err := mail.ParseAddress(req.ToEmail); err != nil {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrValidationFailed,
            "Invalid email format",
            "to_email",
        )
        return
    }
}
```

**Response:**
```json
{
  "code": "validation_failed",
  "message": "Invalid email format",
  "status": 400,
  "field": "to_email"
}
```

---

### 4. Multiple Field Validation Errors

**Scenario:** Validate multiple fields, but report first error (or combine in field)

**Option A: Report first error individually**
```go
func validatePhoneNumbers(w http.ResponseWriter, to, from string) bool {
    if !isValidE164(to) {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrValidationFailed,
            "Invalid phone number format (E.164 required)",
            "to",
        )
        return false
    }

    if !isValidE164(from) {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrValidationFailed,
            "Invalid phone number format (E.164 required)",
            "from",
        )
        return false
    }
    return true
}
```

**Option B: Report all fields together**
```go
func validatePhoneNumbers(w http.ResponseWriter, to, from string) bool {
    var invalidFields []string
    if !isValidE164(to) {
        invalidFields = append(invalidFields, "to")
    }
    if !isValidE164(from) {
        invalidFields = append(invalidFields, "from")
    }

    if len(invalidFields) > 0 {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrValidationFailed,
            "Invalid phone number format (E.164 required)",
            strings.Join(invalidFields, ","),
        )
        return false
    }
    return true
}
```

**Response:**
```json
{
  "code": "validation_failed",
  "message": "Invalid phone number format (E.164 required)",
  "status": 400,
  "field": "to,from"
}
```

---

### 5. Resource Not Found

**Scenario:** User requests a non-existent resource

```go
func (p *GitHubPlugin) getRepository(w http.ResponseWriter, r *http.Request) {
    repoID := chi.URLParam(r, "repo_id")

    repo, err := p.store.GetRepository(repoID)
    if err != nil {
        errors.WriteError(
            w,
            http.StatusNotFound,
            errors.ErrNotFound,
            "Repository not found",
        )
        return
    }

    // Return repository...
}
```

**Response:**
```json
{
  "code": "not_found",
  "message": "Repository not found",
  "status": 404
}
```

---

### 6. Unauthorized (Missing Authentication)

**Scenario:** Request lacks authentication

```go
func (p *MyPlugin) protectedHandler(w http.ResponseWriter, r *http.Request) {
    user, ok := getUserFromContext(r)
    if !ok {
        errors.WriteError(
            w,
            http.StatusUnauthorized,
            errors.ErrUnauthorized,
            "Authentication required",
        )
        return
    }

    // Process authenticated request...
}
```

**Response:**
```json
{
  "code": "unauthorized",
  "message": "Authentication required",
  "status": 401
}
```

---

### 7. Forbidden (Insufficient Permissions)

**Scenario:** User is authenticated but lacks permissions

```go
func (p *MyPlugin) deleteUser(w http.ResponseWriter, r *http.Request) {
    user, _ := getUserFromContext(r)
    userID := chi.URLParam(r, "user_id")

    // Check if user can delete other users
    if user.ID != userID && !user.IsAdmin {
        errors.WriteError(
            w,
            http.StatusForbidden,
            errors.ErrForbidden,
            "You do not have permission to delete this user",
        )
        return
    }

    // Proceed with deletion...
}
```

**Response:**
```json
{
  "code": "forbidden",
  "message": "You do not have permission to delete this user",
  "status": 403
}
```

---

### 8. Conflict (Resource Conflict)

**Scenario:** Request conflicts with existing resource

```go
func (p *MyPlugin) createUser(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email string `json:"email"`
    }

    json.NewDecoder(r.Body).Decode(&req)

    // Check if user already exists
    existing, _ := p.store.GetUserByEmail(req.Email)
    if existing != nil {
        errors.WriteErrorWithField(
            w,
            http.StatusConflict,
            errors.ErrConflict,
            "User with this email already exists",
            "email",
        )
        return
    }

    // Create user...
}
```

**Response:**
```json
{
  "code": "conflict",
  "message": "User with this email already exists",
  "status": 409,
  "field": "email"
}
```

---

### 9. Database Error (Server Error)

**Scenario:** Database operation fails

```go
func (p *SendGridPlugin) createMessage(w http.ResponseWriter, r *http.Request) {
    // ... parse request ...

    message, err := p.store.CreateMessage(req)
    if err != nil {
        errors.WriteErrorWithDetails(
            w,
            http.StatusInternalServerError,
            errors.ErrDatabaseError,
            "Failed to save message",
            err.Error(), // Include actual error details
        )
        return
    }

    // Return message...
}
```

**Response:**
```json
{
  "code": "database_error",
  "message": "Failed to save message",
  "status": 500,
  "details": "sql: connection refused"
}
```

---

### 10. Service Unavailable

**Scenario:** External service or database is down

```go
func (p *GooglePlugin) sendEmail(w http.ResponseWriter, r *http.Request) {
    // Check if Gmail API is available
    if !isGmailServiceAvailable() {
        errors.WriteError(
            w,
            http.StatusServiceUnavailable,
            errors.ErrServiceUnavailable,
            "Gmail service is temporarily unavailable",
        )
        return
    }

    // Send email...
}
```

**Response:**
```json
{
  "code": "service_unavailable",
  "message": "Gmail service is temporarily unavailable",
  "status": 503
}
```

---

### 11. Not Implemented

**Scenario:** Feature exists but is not yet implemented

```go
func (p *AdminPlugin) exportData(w http.ResponseWriter, r *http.Request) {
    format := r.URL.Query().Get("format")

    if format == "csv" {
        errors.WriteError(
            w,
            http.StatusNotImplemented,
            errors.ErrNotImplemented,
            "CSV export is not yet implemented",
        )
        return
    }

    // Handle JSON export...
}
```

**Response:**
```json
{
  "code": "not_implemented",
  "message": "CSV export is not yet implemented",
  "status": 501
}
```

---

## Plugin-Specific Examples

### Discord Plugin Example

**Before:**
```go
func writeError(w http.ResponseWriter, code int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": message,
        "code":    code,
    })
}

func (p *DiscordPlugin) executeWebhook(w http.ResponseWriter, r *http.Request) {
    if p.store == nil {
        writeError(w, 500, "Plugin not initialized")
        return
    }

    var req struct {
        Content string `json:"content"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "Invalid request body")
        return
    }
}
```

**After:**
```go
import "github.com/2389/ish/internal/errors"

func (p *DiscordPlugin) executeWebhook(w http.ResponseWriter, r *http.Request) {
    if p.store == nil {
        errors.WriteError(
            w,
            http.StatusInternalServerError,
            errors.ErrInternal,
            "Plugin not initialized",
        )
        return
    }

    var req struct {
        Content string `json:"content"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.WriteError(
            w,
            http.StatusBadRequest,
            errors.ErrInvalidBody,
            "Request body is invalid",
        )
        return
    }
}
```

---

### GitHub Plugin Example

**Before:**
```go
func writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": message,
    })
}

func (p *GitHubPlugin) getRepository(w http.ResponseWriter, r *http.Request) {
    owner := chi.URLParam(r, "owner")
    repoName := chi.URLParam(r, "repo")

    repo, err := p.store.GetRepositoryByFullName(owner + "/" + repoName)
    if err != nil {
        writeError(w, http.StatusNotFound, "repository not found")
        return
    }
}
```

**After:**
```go
import "github.com/2389/ish/internal/errors"

func (p *GitHubPlugin) getRepository(w http.ResponseWriter, r *http.Request) {
    owner := chi.URLParam(r, "owner")
    repoName := chi.URLParam(r, "repo")

    repo, err := p.store.GetRepositoryByFullName(owner + "/" + repoName)
    if err != nil {
        errors.WriteError(
            w,
            http.StatusNotFound,
            errors.ErrNotFound,
            "Repository not found",
        )
        return
    }
}
```

---

### SendGrid Plugin Example

**Before:**
```go
func writeError(w http.ResponseWriter, status int, message, field string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": message,
        // field parameter ignored!
    })
}

func (p *SendGridPlugin) sendMail(w http.ResponseWriter, r *http.Request) {
    var req SendMailRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body", "")
        return
    }

    if req.From.Email == "" {
        writeError(w, http.StatusBadRequest, "from email is required", "from.email")
        return
    }
}
```

**After:**
```go
import "github.com/2389/ish/internal/errors"

func (p *SendGridPlugin) sendMail(w http.ResponseWriter, r *http.Request) {
    var req SendMailRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.WriteError(
            w,
            http.StatusBadRequest,
            errors.ErrInvalidBody,
            "Request body is invalid",
        )
        return
    }

    if req.From.Email == "" {
        errors.WriteErrorWithField(
            w,
            http.StatusBadRequest,
            errors.ErrMissingField,
            "From email is required",
            "from.email",
        )
        return
    }
}
```

---

### Twilio Plugin Example

**Before:**
```go
func writeError(w http.ResponseWriter, statusCode, errorCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "code":    errorCode,    // Numeric Twilio codes
        "message": message,
        "status":  statusCode,
    })
}

func (p *TwilioPlugin) sendMessage(w http.ResponseWriter, r *http.Request) {
    if to == "" {
        writeError(w, http.StatusBadRequest, 21602, "Missing required parameter: To")
        return
    }
}
```

**After (with Twilio code mapping):**
```go
import "github.com/2389/ish/internal/errors"

func (p *TwilioPlugin) sendMessage(w http.ResponseWriter, r *http.Request) {
    if to == "" {
        errors.WriteErrorWithDetails(
            w,
            http.StatusBadRequest,
            errors.ErrMissingField,
            "Missing required parameter: To",
            "twilio_error_code:21602",
        )
        return
    }
}
```

---

### OAuth Plugin Example

**Before:**
```go
// Mixed approaches - http.Error and custom JSON
http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)

// In token handler:
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)  // No error handling!
```

**After:**
```go
import "github.com/2389/ish/internal/errors"

errors.WriteErrorWithField(
    w,
    http.StatusBadRequest,
    errors.ErrValidationFailed,
    "Invalid redirect_uri format",
    "redirect_uri",
)

// For successful responses, still use json.NewEncoder but handle errors
w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(response); err != nil {
    errors.WriteError(
        w,
        http.StatusInternalServerError,
        errors.ErrInternal,
        "Failed to encode response",
    )
}
```

---

## Testing Error Responses

### Unit Test Example

```go
func TestSendMailInvalidFrom(t *testing.T) {
    req, _ := http.NewRequest("POST", "/v3/mail/send",
        strings.NewReader(`{"from_email":""}`))
    w := httptest.NewRecorder()

    handler := NewSendGridPlugin()
    handler.sendMail(w, req)

    // Verify error response
    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }

    var resp map[string]interface{}
    json.NewDecoder(w.Body).Decode(&resp)

    if resp["code"] != "missing_field" {
        t.Errorf("expected error code missing_field, got %v", resp["code"])
    }
    if resp["field"] != "from.email" {
        t.Errorf("expected field from.email, got %v", resp["field"])
    }
}
```

---

## Checklist for Implementation

- [ ] Import `github.com/2389/ish/internal/errors` at top of file
- [ ] Remove any custom `writeError()` or `writeErrorResponse()` functions
- [ ] Replace all `http.Error()` calls with appropriate error helper
- [ ] For validation errors, use `WriteErrorWithField()`
- [ ] For complex errors, use `WriteErrorWithDetails()`
- [ ] For simple errors, use `WriteError()`
- [ ] Verify all error responses are valid JSON
- [ ] Add or update unit tests for error cases
- [ ] Test that status codes in header match body
- [ ] Verify Content-Type header is set to application/json
