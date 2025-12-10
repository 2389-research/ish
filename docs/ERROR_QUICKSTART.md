# Error Response Standardization - Quick Start

## One-Minute Overview

**Problem:** 13 plugin files use 7 different error response formats.

**Solution:** Standardized error package with 3 helper functions.

**Status:** Framework complete, ready for plugin migration.

---

## What You Need to Know

### The Standard Format

All errors now return JSON with this structure:

```json
{
  "code": "error_code_string",
  "message": "Human-readable message",
  "status": 400,
  "field": "optional.field.name",
  "details": "optional additional context"
}
```

### The Three Helper Functions

```go
import "github.com/2389/ish/internal/errors"

// Simple error
errors.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")

// For validation errors with field reference
errors.WriteErrorWithField(w, http.StatusBadRequest, "missing_field", "Email required", "email")

// For errors with additional context
errors.WriteErrorWithDetails(w, http.StatusInternalServerError, "database_error", "Save failed", "timeout")
```

### Standard Error Codes

Use these constants instead of string literals:

```go
errors.ErrInvalidRequest      // 400
errors.ErrInvalidBody         // 400
errors.ErrMissingField        // 400
errors.ErrValidationFailed    // 400
errors.ErrNotFound            // 404
errors.ErrUnauthorized        // 401
errors.ErrForbidden           // 403
errors.ErrConflict            // 409
errors.ErrInternal            // 500
errors.ErrDatabaseError       // 500
errors.ErrServiceUnavailable  // 503
errors.ErrNotImplemented      // 501
```

---

## How to Migrate a Plugin

### Step 1: Add Import
```go
import "github.com/2389/ish/internal/errors"
```

### Step 2: Remove Old Error Function
Delete the custom `writeError()` function from your file.

### Step 3: Replace Error Calls
Find all `writeError()` or `http.Error()` calls and replace:

**Discord before:**
```go
writeError(w, 500, "Failed to create message")
```

**Discord after:**
```go
errors.WriteError(w, http.StatusInternalServerError, errors.ErrInternal, "Failed to create message")
```

**GitHub before:**
```go
writeError(w, http.StatusNotFound, "repository not found")
```

**GitHub after:**
```go
errors.WriteError(w, http.StatusNotFound, errors.ErrNotFound, "Repository not found")
```

**SendGrid before:**
```go
writeError(w, http.StatusBadRequest, "from email is required", "from.email")
```

**SendGrid after:**
```go
errors.WriteErrorWithField(w, http.StatusBadRequest, errors.ErrMissingField, "From email is required", "from.email")
```

### Step 4: Test
```bash
go test ./plugins/your_plugin -v
```

---

## Files Created

| File | Purpose |
|------|---------|
| `/internal/errors/errors.go` | Main error package with helper functions |
| `/internal/errors/errors_test.go` | Comprehensive test suite (all passing) |
| `/docs/ERROR_RESPONSES.md` | Detailed format guide with migration examples |
| `/docs/ERROR_RESPONSE_ANALYSIS.md` | Full analysis report |
| `/docs/ERROR_MIGRATION_SUMMARY.md` | File-by-file migration checklist |
| `/docs/ERROR_IMPLEMENTATION_EXAMPLES.md` | Real-world usage examples |
| `/docs/ERROR_QUICKSTART.md` | This file |

---

## Migration Status

**Completed:**
- ✓ Error package created
- ✓ Helper functions implemented
- ✓ 12 unit tests written and passing
- ✓ Documentation complete
- ✓ Migration examples provided

**Pending:**
- [ ] Discord plugin (`plugins/discord/handlers.go`) - 20 handlers
- [ ] GitHub plugin (`plugins/github/handlers.go`, `plugin.go`) - 45 handlers
- [ ] SendGrid plugin (`plugins/sendgrid/handlers.go`, `plugin.go`) - 23 handlers
- [ ] Twilio plugin (`plugins/twilio/handlers.go`, `plugin.go`) - 14 handlers
- [ ] OAuth plugin (`plugins/oauth/handlers.go`, `plugin.go`) - 14 handlers
- [ ] Home Assistant plugin (`plugins/homeassistant/plugin.go`) - 20+ handlers
- [ ] Google plugin (`plugins/google/gmail.go`, `discovery.go`) - 12 handlers
- [ ] Admin handlers (`internal/admin/handlers.go`) - 3 handlers

**Total:** 151+ error handlers need updating

---

## Common Patterns

### Pattern 1: Invalid Input
```go
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    errors.WriteError(w, http.StatusBadRequest, errors.ErrInvalidBody, "Request body is invalid")
    return
}
```

### Pattern 2: Missing Field
```go
if req.Email == "" {
    errors.WriteErrorWithField(w, http.StatusBadRequest, errors.ErrMissingField, "Email is required", "email")
    return
}
```

### Pattern 3: Not Found
```go
item, err := store.Get(id)
if err != nil {
    errors.WriteError(w, http.StatusNotFound, errors.ErrNotFound, "Item not found")
    return
}
```

### Pattern 4: Database Error
```go
if err := store.Save(item); err != nil {
    errors.WriteErrorWithDetails(w, http.StatusInternalServerError, errors.ErrDatabaseError, "Failed to save item", err.Error())
    return
}
```

---

## Testing Your Changes

### Quick Test
```bash
# Test the error package
go test ./internal/errors -v

# Test your modified plugin
go test ./plugins/your_plugin -v
```

### Verify Response Format
```bash
# Run the server and test an error endpoint
curl -X GET http://localhost:8080/api/nonexistent
# Should return: {"code":"not_found","message":"...","status":404}
```

---

## FAQ

**Q: Do I need to handle all 151 handlers at once?**
A: No. Prioritize by plugin size: Discord (20), GitHub (45), SendGrid (23), Twilio (14).

**Q: What if my plugin uses custom error codes (like Twilio)?**
A: Map them to standard codes and store plugin-specific codes in the `details` field.

**Q: Do I need to update existing tests?**
A: Yes. Your error assertions need to match the new response format.

**Q: Can I use this in other files?**
A: Yes. The error package is in `internal/errors` and can be imported anywhere.

**Q: What about backwards compatibility?**
A: New format is additive only - no fields are removed, just structured better.

---

## Support Documents

- **Full Details:** `/docs/ERROR_RESPONSES.md`
- **Analysis Report:** `/docs/ERROR_RESPONSE_ANALYSIS.md`
- **File-by-File Guide:** `/docs/ERROR_MIGRATION_SUMMARY.md`
- **Code Examples:** `/docs/ERROR_IMPLEMENTATION_EXAMPLES.md`
- **Source Code:** `/internal/errors/errors.go`
- **Tests:** `/internal/errors/errors_test.go`

---

## Next Steps

1. **Review** the error package: `/internal/errors/errors.go`
2. **Read** migration examples: `/docs/ERROR_IMPLEMENTATION_EXAMPLES.md`
3. **Pick** a plugin to migrate (start with Discord)
4. **Update** all error handlers in that plugin
5. **Test** with `go test ./plugins/discord -v`
6. **Commit** with message: `refactor: standardize error responses in [plugin] plugin`
7. **Repeat** for remaining plugins

---

## Error Code Reference

### 4xx Client Errors

| Code | HTTP | Meaning |
|------|------|---------|
| `invalid_request` | 400 | Generic invalid request |
| `invalid_request_body` | 400 | Malformed JSON/body |
| `missing_field` | 400 | Required field missing |
| `validation_failed` | 400 | Field validation failed |
| `not_found` | 404 | Resource doesn't exist |
| `unauthorized` | 401 | Not authenticated |
| `forbidden` | 403 | Not authorized |
| `conflict` | 409 | Resource conflict |

### 5xx Server Errors

| Code | HTTP | Meaning |
|------|------|---------|
| `internal_error` | 500 | Generic server error |
| `database_error` | 500 | Database operation failed |
| `service_unavailable` | 503 | Service down |
| `not_implemented` | 501 | Feature not yet implemented |

---

## Validation

All work is validated by:
- ✓ 12 passing unit tests in `internal/errors`
- ✓ Comprehensive documentation
- ✓ Real-world examples for each plugin
- ✓ Clear migration path for each file

---

**Ready to start?** Pick the Discord plugin and follow the pattern in `/docs/ERROR_IMPLEMENTATION_EXAMPLES.md`
