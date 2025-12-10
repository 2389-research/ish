# Error Response Standardization - Complete Index

## Overview

This directory contains the complete standardized error response framework for the ISH plugin system. This index helps you navigate all the resources.

**Last Updated:** December 9, 2025
**Status:** Framework complete and tested. Ready for plugin migration.

---

## Quick Navigation

### For Quick Start
Start here if you just want to understand and use the error framework:
- **[ERROR_QUICKSTART.md](ERROR_QUICKSTART.md)** - 1-minute overview, common patterns, FAQ

### For Implementation
These files contain the actual code and how to use it:
- **[/internal/errors/errors.go](/internal/errors/errors.go)** - Main error package (109 lines)
- **[/internal/errors/errors_test.go](/internal/errors/errors_test.go)** - Test suite (354 lines, all passing)
- **[ERROR_IMPLEMENTATION_EXAMPLES.md](ERROR_IMPLEMENTATION_EXAMPLES.md)** - Real-world code examples

### For Migration
Use these to migrate plugins:
- **[ERROR_MIGRATION_SUMMARY.md](ERROR_MIGRATION_SUMMARY.md)** - File-by-file migration checklist
- **[ERROR_RESPONSES.md](ERROR_RESPONSES.md)** - Format specification and migration examples

### For Analysis
Understand the current state and impact:
- **[ERROR_RESPONSE_ANALYSIS.md](ERROR_RESPONSE_ANALYSIS.md)** - Detailed analysis report

---

## File Descriptions

### Code Files

#### `/internal/errors/errors.go` (109 lines)
The main error package. Contains:
- `ErrorResponse` struct with fields: code, message, status, field (optional), details (optional)
- `WriteError()` - simple error response
- `WriteErrorWithField()` - validation error with field reference
- `WriteErrorWithDetails()` - error with additional context
- 12 standard error code constants

**Usage:**
```go
import "github.com/2389/ish/internal/errors"

errors.WriteError(w, http.StatusBadRequest, errors.ErrInvalidBody, "Invalid JSON")
errors.WriteErrorWithField(w, http.StatusBadRequest, errors.ErrMissingField, "Email required", "email")
errors.WriteErrorWithDetails(w, http.StatusInternalServerError, errors.ErrDatabaseError, "Save failed", "timeout")
```

#### `/internal/errors/errors_test.go` (354 lines)
Comprehensive test suite with 12 tests covering:
- Basic error responses (4 scenarios)
- Field validation (3 scenarios)
- Error with details (2 scenarios)
- JSON marshaling (3 scenarios)
- Response consistency
- HTTP headers
- Status code matching for all HTTP statuses
- Error code constants

**Status:** All 12 tests PASSING ✓

### Documentation Files

#### `ERROR_QUICKSTART.md` (280 lines)
**Best for:** Getting started quickly

Contents:
- One-minute overview
- Standard format and helper functions reference
- Error codes quick reference
- Migration steps
- Common patterns
- Testing
- FAQ

Read this first if you're in a hurry.

#### `ERROR_RESPONSES.md` (359 lines)
**Best for:** Understanding the specification and migrating

Contents:
- Standard error response format (detailed)
- Error response code reference
- Helper function documentation
- Usage examples
- Migration guide with before/after examples for:
  - Discord plugin
  - SendGrid plugin
  - GitHub plugin
  - Twilio plugin
  - OAuth plugin
  - Home Assistant plugin
- Implementation checklist
- Testing guidance

This is your main reference for the standard format.

#### `ERROR_RESPONSE_ANALYSIS.md` (416 lines)
**Best for:** Understanding the problem and impact

Contents:
- Executive summary
- Current error formats found (7 different types)
- Impact analysis
- Files needing migration (13 files, 151+ handlers)
- Detailed format examples
- Statistics and metrics
- Error code mapping guide
- Plugin-specific considerations
- Testing strategy
- Migration priority
- Validation checklist

Read this to understand why standardization is needed.

#### `ERROR_MIGRATION_SUMMARY.md` (273 lines)
**Best for:** Migrating each plugin

Contents:
- Quick reference table
- File-by-file migration list organized by priority:
  - Priority 1: High (6 files, 98 handlers)
  - Priority 2: Medium (3 files, 18 handlers)
  - Priority 3: Lower (4 files, 35+ handlers)
- Standard format and helper functions
- Standard error codes
- Migration checklist per plugin
- Before/after examples
- Testing requirements
- Progress tracking table

Use this to track migration work.

#### `ERROR_IMPLEMENTATION_EXAMPLES.md` (447 lines)
**Best for:** Actual code implementation

Contents:
- Common error scenarios (11 scenarios):
  1. Invalid request body
  2. Missing required field
  3. Field validation error
  4. Multiple field validation
  5. Resource not found
  6. Unauthorized
  7. Forbidden
  8. Conflict
  9. Database error
  10. Service unavailable
  11. Not implemented
- Plugin-specific migration examples:
  - Discord (before/after)
  - GitHub (before/after)
  - SendGrid (before/after with validation)
  - Twilio (before/after with error code mapping)
  - OAuth (before/after)
- Unit test examples
- Implementation checklist

This has real code you can copy and adapt.

---

## Error Codes Reference

### Client Errors (4xx)

| Code | HTTP | Use When |
|------|------|----------|
| `invalid_request` | 400 | Generic invalid request |
| `invalid_request_body` | 400 | Malformed JSON/body |
| `missing_field` | 400 | Required field is missing |
| `validation_failed` | 400 | Field validation failed |
| `not_found` | 404 | Resource doesn't exist |
| `unauthorized` | 401 | Not authenticated |
| `forbidden` | 403 | Not authorized |
| `conflict` | 409 | Resource conflict |

### Server Errors (5xx)

| Code | HTTP | Use When |
|------|------|----------|
| `internal_error` | 500 | Generic server error |
| `database_error` | 500 | Database operation failed |
| `service_unavailable` | 503 | Service is down |
| `not_implemented` | 501 | Feature not yet implemented |

---

## Standard Format

All errors return this JSON structure:

```json
{
  "code": "error_code",
  "message": "Human-readable message",
  "status": 400,
  "field": "optional.field.name",
  "details": "optional additional context"
}
```

### Field Descriptions

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| code | Yes | string | Machine-readable error code (e.g., "invalid_request") |
| message | Yes | string | Human-readable error message for developers/users |
| status | Yes | integer | HTTP status code (same as HTTP response code) |
| field | No | string | Field name that caused the error (validation errors) |
| details | No | string | Additional debugging info (errors with context) |

---

## Helper Functions

### WriteError
For simple errors without field or details:

```go
errors.WriteError(w, http.StatusBadRequest, errors.ErrInvalidBody, "Request body is invalid")
```

**Response:**
```json
{
  "code": "invalid_request_body",
  "message": "Request body is invalid",
  "status": 400
}
```

### WriteErrorWithField
For validation errors with field reference:

```go
errors.WriteErrorWithField(w, http.StatusBadRequest, errors.ErrMissingField, "Email is required", "email")
```

**Response:**
```json
{
  "code": "missing_field",
  "message": "Email is required",
  "status": 400,
  "field": "email"
}
```

### WriteErrorWithDetails
For errors with additional context:

```go
errors.WriteErrorWithDetails(w, http.StatusInternalServerError, errors.ErrDatabaseError, "Failed to save", "connection timeout")
```

**Response:**
```json
{
  "code": "database_error",
  "message": "Failed to save",
  "status": 500,
  "details": "connection timeout"
}
```

---

## Migration Status

### Completed (Phase 1)
- ✓ Error package created (`/internal/errors/errors.go`)
- ✓ Helper functions implemented (3 functions)
- ✓ Error code constants defined (12 codes)
- ✓ Test suite written (12 tests, all passing)
- ✓ Documentation complete (5 documents)
- ✓ Examples provided (11 scenarios)

### Pending (Phases 2-4)
- [ ] Discord plugin (20 handlers)
- [ ] GitHub plugin (45 handlers)
- [ ] SendGrid plugin (23 handlers)
- [ ] Twilio plugin (14 handlers)
- [ ] OAuth plugin (14 handlers)
- [ ] Home Assistant plugin (20+ handlers)
- [ ] Google plugin (12 handlers)
- [ ] Admin handlers (3 handlers)

**Total Pending:** 151+ handlers across 13 files

---

## How to Use This Index

### I want to understand the problem
→ Read [ERROR_RESPONSE_ANALYSIS.md](ERROR_RESPONSE_ANALYSIS.md)

### I want to get started quickly
→ Read [ERROR_QUICKSTART.md](ERROR_QUICKSTART.md)

### I want to migrate a plugin
1. Read [ERROR_MIGRATION_SUMMARY.md](ERROR_MIGRATION_SUMMARY.md) to see your plugin's priority
2. Read [ERROR_IMPLEMENTATION_EXAMPLES.md](ERROR_IMPLEMENTATION_EXAMPLES.md) to see examples
3. Reference [ERROR_RESPONSES.md](ERROR_RESPONSES.md) for the specification
4. Follow the patterns in [/internal/errors/errors.go](/internal/errors/errors.go)

### I want to implement error handling
→ Read [ERROR_IMPLEMENTATION_EXAMPLES.md](ERROR_IMPLEMENTATION_EXAMPLES.md)

### I want the detailed specification
→ Read [ERROR_RESPONSES.md](ERROR_RESPONSES.md)

### I want to verify tests pass
```bash
go test ./internal/errors -v
```

---

## Statistics

### Code & Tests
- Package code: 109 lines
- Test code: 354 lines
- Tests passing: 12/12 ✓

### Documentation
- ERROR_RESPONSES.md: 359 lines
- ERROR_RESPONSE_ANALYSIS.md: 416 lines
- ERROR_MIGRATION_SUMMARY.md: 273 lines
- ERROR_IMPLEMENTATION_EXAMPLES.md: 447 lines
- ERROR_QUICKSTART.md: 280 lines
- ERROR_RESPONSES_INDEX.md: This file

### Files Needing Migration
- Total files: 13
- Total handlers: 151+
- Different error formats found: 7

---

## Key Takeaways

1. **One Standard Format:** All plugins use the same JSON error structure
2. **Three Helper Functions:** WriteError, WriteErrorWithField, WriteErrorWithDetails
3. **12 Standard Codes:** Consistent error codes across all plugins
4. **Fully Tested:** All helper functions tested and working
5. **Well Documented:** 5 different documents covering all aspects
6. **Ready to Migrate:** All examples and patterns provided

---

## Next Steps

1. **Review** the error package: `/internal/errors/errors.go`
2. **Pick** a plugin to migrate (start with Discord - 20 handlers)
3. **Follow** the pattern in [ERROR_IMPLEMENTATION_EXAMPLES.md](ERROR_IMPLEMENTATION_EXAMPLES.md)
4. **Use** the migration checklist in [ERROR_MIGRATION_SUMMARY.md](ERROR_MIGRATION_SUMMARY.md)
5. **Test** with `go test ./plugins/your_plugin -v`
6. **Commit** with message: `refactor: standardize error responses in [plugin] plugin`

---

## Questions?

Refer to:
- [ERROR_QUICKSTART.md](ERROR_QUICKSTART.md) - FAQ section
- [ERROR_IMPLEMENTATION_EXAMPLES.md](ERROR_IMPLEMENTATION_EXAMPLES.md) - Real code examples
- `/internal/errors/errors_test.go` - Test examples

---

## File Tree

```
.
├── internal/
│   └── errors/
│       ├── errors.go (main package)
│       └── errors_test.go (tests, all passing)
└── docs/
    ├── ERROR_RESPONSES_INDEX.md (this file)
    ├── ERROR_QUICKSTART.md (start here!)
    ├── ERROR_RESPONSES.md (format specification)
    ├── ERROR_RESPONSE_ANALYSIS.md (detailed analysis)
    ├── ERROR_MIGRATION_SUMMARY.md (migration checklist)
    └── ERROR_IMPLEMENTATION_EXAMPLES.md (code examples)
```

---

**Status:** Complete and ready for use
**Last Updated:** December 9, 2025
**Version:** 1.0
