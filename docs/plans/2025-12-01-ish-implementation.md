# ISH Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a fake Google API emulator (Gmail, Calendar, People) for local client development.

**Architecture:** Single Go binary with chi router, SQLite storage, and dumb auth middleware. Three API modules share a common store layer.

**Tech Stack:** Go 1.21+, chi router, go-sqlite3, cobra CLI

---

## Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `go.sum`

**Step 1: Initialize module**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go mod init github.com/2389/ish
```

**Step 2: Add dependencies**

Run:
```bash
go get github.com/go-chi/chi/v5
go get github.com/mattn/go-sqlite3
go get github.com/spf13/cobra
```

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: initialize go module with dependencies"
```

---

## Task 2: Create Store Layer - Schema and Migrations

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

**Step 1: Write failing test for store initialization**

Create `internal/store/store_test.go`:
```go
// ABOUTME: Tests for SQLite store initialization and schema migrations.
// ABOUTME: Verifies database setup and table creation.

package store

import (
	"os"
	"testing"
)

func TestNewStore_CreatesDatabase(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify tables exist
	tables := []string{"users", "gmail_labels", "gmail_threads", "gmail_messages", "calendars", "calendar_events", "people"}
	for _, table := range tables {
		var name string
		err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/store/store.go`:
```go
// ABOUTME: SQLite store for fake Google API data.
// ABOUTME: Handles database initialization, migrations, and connection management.

package store

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS gmail_labels (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		name TEXT NOT NULL,
		type TEXT DEFAULT 'user'
	);

	CREATE TABLE IF NOT EXISTS gmail_threads (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		snippet TEXT
	);

	CREATE TABLE IF NOT EXISTS gmail_messages (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		thread_id TEXT REFERENCES gmail_threads(id),
		label_ids TEXT,
		snippet TEXT,
		internal_date INTEGER,
		payload TEXT
	);

	CREATE TABLE IF NOT EXISTS calendars (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		summary TEXT
	);

	CREATE TABLE IF NOT EXISTS calendar_events (
		id TEXT PRIMARY KEY,
		calendar_id TEXT REFERENCES calendars(id),
		summary TEXT,
		description TEXT,
		start_time TEXT,
		end_time TEXT,
		attendees TEXT
	);

	CREATE TABLE IF NOT EXISTS people (
		resource_name TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		data TEXT
	);
	`
	_, err := s.db.Exec(schema)
	return err
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add SQLite store with schema migrations"
```

---

## Task 3: Store Layer - User Operations

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

**Step 1: Write failing test for user operations**

Add to `internal/store/store_test.go`:
```go
func TestStore_CreateAndGetUser(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Create user
	err = s.CreateUser("harper")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// Get user
	exists, err := s.UserExists("harper")
	if err != nil {
		t.Fatalf("UserExists() error = %v", err)
	}
	if !exists {
		t.Error("UserExists() = false, want true")
	}

	// Non-existent user
	exists, err = s.UserExists("nobody")
	if err != nil {
		t.Fatalf("UserExists() error = %v", err)
	}
	if exists {
		t.Error("UserExists() = true for non-existent user")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_CreateAndGetUser
```

Expected: FAIL (methods don't exist)

**Step 3: Write minimal implementation**

Add to `internal/store/store.go`:
```go
func (s *Store) CreateUser(id string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO users (id) VALUES (?)", id)
	return err
}

func (s *Store) UserExists(id string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", id).Scan(&count)
	return count > 0, err
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_CreateAndGetUser
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add user create and exists operations"
```

---

## Task 4: Store Layer - Gmail Operations

**Files:**
- Create: `internal/store/gmail.go`
- Create: `internal/store/gmail_test.go`

**Step 1: Write failing test for Gmail message operations**

Create `internal/store/gmail_test.go`:
```go
// ABOUTME: Tests for Gmail-related store operations.
// ABOUTME: Covers message listing, retrieval, and filtering.

package store

import (
	"os"
	"testing"
)

func TestStore_GmailMessages(t *testing.T) {
	dbPath := "test_gmail.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Setup user
	s.CreateUser("harper")

	// Create thread and message
	thread := &GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Hello world"}
	err = s.CreateGmailThread(thread)
	if err != nil {
		t.Fatalf("CreateGmailThread() error = %v", err)
	}

	msg := &GmailMessage{
		ID:           "msg_1",
		UserID:       "harper",
		ThreadID:     "thr_1",
		LabelIDs:     []string{"INBOX", "STARRED"},
		Snippet:      "Hello world",
		InternalDate: 1733000000000,
		Payload:      `{"headers":[{"name":"Subject","value":"Test"}]}`,
	}
	err = s.CreateGmailMessage(msg)
	if err != nil {
		t.Fatalf("CreateGmailMessage() error = %v", err)
	}

	// List messages
	msgs, nextToken, err := s.ListGmailMessages("harper", 10, "", "")
	if err != nil {
		t.Fatalf("ListGmailMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("ListGmailMessages() got %d messages, want 1", len(msgs))
	}
	if nextToken != "" {
		t.Errorf("ListGmailMessages() nextToken = %q, want empty", nextToken)
	}

	// Get single message
	got, err := s.GetGmailMessage("harper", "msg_1")
	if err != nil {
		t.Fatalf("GetGmailMessage() error = %v", err)
	}
	if got.ID != "msg_1" {
		t.Errorf("GetGmailMessage() ID = %q, want %q", got.ID, "msg_1")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_GmailMessages
```

Expected: FAIL (types and methods don't exist)

**Step 3: Write minimal implementation**

Create `internal/store/gmail.go`:
```go
// ABOUTME: Gmail-related store operations for messages and threads.
// ABOUTME: Handles CRUD and listing with pagination support.

package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type GmailThread struct {
	ID      string
	UserID  string
	Snippet string
}

type GmailMessage struct {
	ID           string
	UserID       string
	ThreadID     string
	LabelIDs     []string
	Snippet      string
	InternalDate int64
	Payload      string
}

func (s *Store) CreateGmailThread(t *GmailThread) error {
	_, err := s.db.Exec(
		"INSERT INTO gmail_threads (id, user_id, snippet) VALUES (?, ?, ?)",
		t.ID, t.UserID, t.Snippet,
	)
	return err
}

func (s *Store) CreateGmailMessage(m *GmailMessage) error {
	labelJSON, _ := json.Marshal(m.LabelIDs)
	_, err := s.db.Exec(
		"INSERT INTO gmail_messages (id, user_id, thread_id, label_ids, snippet, internal_date, payload) VALUES (?, ?, ?, ?, ?, ?, ?)",
		m.ID, m.UserID, m.ThreadID, string(labelJSON), m.Snippet, m.InternalDate, m.Payload,
	)
	return err
}

func (s *Store) ListGmailMessages(userID string, maxResults int, pageToken string, query string) ([]GmailMessage, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT id, user_id, thread_id, label_ids, snippet, internal_date, payload FROM gmail_messages WHERE user_id = ?"
	args := []any{userID}

	if query != "" {
		sqlQuery += " AND snippet LIKE ?"
		args = append(args, "%"+query+"%")
	}

	sqlQuery += " ORDER BY internal_date DESC LIMIT ? OFFSET ?"
	args = append(args, maxResults+1, offset) // +1 to check if there's more

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var messages []GmailMessage
	for rows.Next() {
		var m GmailMessage
		var labelJSON string
		err := rows.Scan(&m.ID, &m.UserID, &m.ThreadID, &labelJSON, &m.Snippet, &m.InternalDate, &m.Payload)
		if err != nil {
			return nil, "", err
		}
		json.Unmarshal([]byte(labelJSON), &m.LabelIDs)
		messages = append(messages, m)
	}

	var nextToken string
	if len(messages) > maxResults {
		messages = messages[:maxResults]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + maxResults)))
	}

	return messages, nextToken, nil
}

func (s *Store) GetGmailMessage(userID, messageID string) (*GmailMessage, error) {
	var m GmailMessage
	var labelJSON string
	err := s.db.QueryRow(
		"SELECT id, user_id, thread_id, label_ids, snippet, internal_date, payload FROM gmail_messages WHERE user_id = ? AND id = ?",
		userID, messageID,
	).Scan(&m.ID, &m.UserID, &m.ThreadID, &labelJSON, &m.Snippet, &m.InternalDate, &m.Payload)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(labelJSON), &m.LabelIDs)
	return &m, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_GmailMessages
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add Gmail message and thread operations"
```

---

## Task 5: Store Layer - Calendar Operations

**Files:**
- Create: `internal/store/calendar.go`
- Create: `internal/store/calendar_test.go`

**Step 1: Write failing test for Calendar operations**

Create `internal/store/calendar_test.go`:
```go
// ABOUTME: Tests for Calendar-related store operations.
// ABOUTME: Covers event listing, retrieval, and time filtering.

package store

import (
	"os"
	"testing"
)

func TestStore_CalendarEvents(t *testing.T) {
	dbPath := "test_calendar.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Setup user and calendar
	s.CreateUser("harper")
	cal := &Calendar{ID: "cal_primary", UserID: "harper", Summary: "Primary"}
	err = s.CreateCalendar(cal)
	if err != nil {
		t.Fatalf("CreateCalendar() error = %v", err)
	}

	// Create event
	evt := &CalendarEvent{
		ID:          "evt_1",
		CalendarID:  "cal_primary",
		Summary:     "Coffee",
		Description: "Discuss things",
		StartTime:   "2025-12-01T10:00:00Z",
		EndTime:     "2025-12-01T11:00:00Z",
		Attendees:   `[{"email":"harper@example.com"}]`,
	}
	err = s.CreateCalendarEvent(evt)
	if err != nil {
		t.Fatalf("CreateCalendarEvent() error = %v", err)
	}

	// List events
	events, nextToken, err := s.ListCalendarEvents("cal_primary", 10, "", "", "")
	if err != nil {
		t.Fatalf("ListCalendarEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Errorf("ListCalendarEvents() got %d events, want 1", len(events))
	}
	if nextToken != "" {
		t.Errorf("ListCalendarEvents() nextToken = %q, want empty", nextToken)
	}

	// Get single event
	got, err := s.GetCalendarEvent("cal_primary", "evt_1")
	if err != nil {
		t.Fatalf("GetCalendarEvent() error = %v", err)
	}
	if got.Summary != "Coffee" {
		t.Errorf("GetCalendarEvent() Summary = %q, want %q", got.Summary, "Coffee")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_CalendarEvents
```

Expected: FAIL (types and methods don't exist)

**Step 3: Write minimal implementation**

Create `internal/store/calendar.go`:
```go
// ABOUTME: Calendar-related store operations for calendars and events.
// ABOUTME: Handles CRUD and listing with time range filtering.

package store

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
)

type Calendar struct {
	ID      string
	UserID  string
	Summary string
}

type CalendarEvent struct {
	ID          string
	CalendarID  string
	Summary     string
	Description string
	StartTime   string
	EndTime     string
	Attendees   string
}

func (s *Store) CreateCalendar(c *Calendar) error {
	_, err := s.db.Exec(
		"INSERT INTO calendars (id, user_id, summary) VALUES (?, ?, ?)",
		c.ID, c.UserID, c.Summary,
	)
	return err
}

func (s *Store) CreateCalendarEvent(e *CalendarEvent) error {
	_, err := s.db.Exec(
		"INSERT INTO calendar_events (id, calendar_id, summary, description, start_time, end_time, attendees) VALUES (?, ?, ?, ?, ?, ?, ?)",
		e.ID, e.CalendarID, e.Summary, e.Description, e.StartTime, e.EndTime, e.Attendees,
	)
	return err
}

func (s *Store) ListCalendarEvents(calendarID string, maxResults int, pageToken string, timeMin string, timeMax string) ([]CalendarEvent, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT id, calendar_id, summary, description, start_time, end_time, attendees FROM calendar_events WHERE calendar_id = ?"
	args := []any{calendarID}

	if timeMin != "" {
		sqlQuery += " AND start_time >= ?"
		args = append(args, timeMin)
	}
	if timeMax != "" {
		sqlQuery += " AND start_time <= ?"
		args = append(args, timeMax)
	}

	sqlQuery += " ORDER BY start_time ASC LIMIT ? OFFSET ?"
	args = append(args, maxResults+1, offset)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		err := rows.Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees)
		if err != nil {
			return nil, "", err
		}
		events = append(events, e)
	}

	var nextToken string
	if len(events) > maxResults {
		events = events[:maxResults]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + maxResults)))
	}

	return events, nextToken, nil
}

func (s *Store) GetCalendarEvent(calendarID, eventID string) (*CalendarEvent, error) {
	var e CalendarEvent
	err := s.db.QueryRow(
		"SELECT id, calendar_id, summary, description, start_time, end_time, attendees FROM calendar_events WHERE calendar_id = ? AND id = ?",
		calendarID, eventID,
	).Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_CalendarEvents
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add Calendar event operations"
```

---

## Task 6: Store Layer - People Operations

**Files:**
- Create: `internal/store/people.go`
- Create: `internal/store/people_test.go`

**Step 1: Write failing test for People operations**

Create `internal/store/people_test.go`:
```go
// ABOUTME: Tests for People-related store operations.
// ABOUTME: Covers contact listing, retrieval, and search.

package store

import (
	"os"
	"testing"
)

func TestStore_People(t *testing.T) {
	dbPath := "test_people.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Setup user
	s.CreateUser("harper")

	// Create person
	p := &Person{
		ResourceName: "people/c123",
		UserID:       "harper",
		Data:         `{"names":[{"displayName":"Alice"}],"emailAddresses":[{"value":"alice@example.com"}]}`,
	}
	err = s.CreatePerson(p)
	if err != nil {
		t.Fatalf("CreatePerson() error = %v", err)
	}

	// Search people
	people, nextToken, err := s.SearchPeople("harper", "alice", 10, "")
	if err != nil {
		t.Fatalf("SearchPeople() error = %v", err)
	}
	if len(people) != 1 {
		t.Errorf("SearchPeople() got %d people, want 1", len(people))
	}
	if nextToken != "" {
		t.Errorf("SearchPeople() nextToken = %q, want empty", nextToken)
	}

	// Get single person
	got, err := s.GetPerson("harper", "people/c123")
	if err != nil {
		t.Fatalf("GetPerson() error = %v", err)
	}
	if got.ResourceName != "people/c123" {
		t.Errorf("GetPerson() ResourceName = %q, want %q", got.ResourceName, "people/c123")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_People
```

Expected: FAIL (types and methods don't exist)

**Step 3: Write minimal implementation**

Create `internal/store/people.go`:
```go
// ABOUTME: People-related store operations for contacts.
// ABOUTME: Handles CRUD, listing, and search functionality.

package store

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
)

type Person struct {
	ResourceName string
	UserID       string
	Data         string
}

func (s *Store) CreatePerson(p *Person) error {
	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		p.ResourceName, p.UserID, p.Data,
	)
	return err
}

func (s *Store) SearchPeople(userID string, query string, pageSize int, pageToken string) ([]Person, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT resource_name, user_id, data FROM people WHERE user_id = ?"
	args := []any{userID}

	if query != "" {
		sqlQuery += " AND data LIKE ?"
		args = append(args, "%"+query+"%")
	}

	sqlQuery += " ORDER BY resource_name ASC LIMIT ? OFFSET ?"
	args = append(args, pageSize+1, offset)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data)
		if err != nil {
			return nil, "", err
		}
		people = append(people, p)
	}

	var nextToken string
	if len(people) > pageSize {
		people = people[:pageSize]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + pageSize)))
	}

	return people, nextToken, nil
}

func (s *Store) GetPerson(userID, resourceName string) (*Person, error) {
	var p Person
	err := s.db.QueryRow(
		"SELECT resource_name, user_id, data FROM people WHERE user_id = ? AND resource_name = ?",
		userID, resourceName,
	).Scan(&p.ResourceName, &p.UserID, &p.Data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("person not found")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/store/... -v -run TestStore_People
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add People contact operations"
```

---

## Task 7: Auth Middleware

**Files:**
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/middleware_test.go`

**Step 1: Write failing test for auth middleware**

Create `internal/auth/middleware_test.go`:
```go
// ABOUTME: Tests for authentication middleware.
// ABOUTME: Verifies token parsing and user extraction from headers.

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_ExtractsUser(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		wantUser   string
	}{
		{"user prefix", "Bearer user:harper", "harper"},
		{"simple token", "Bearer harper", "harper"},
		{"no header", "", "default"},
		{"empty bearer", "Bearer ", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser string
			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = UserFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if gotUser != tt.wantUser {
				t.Errorf("UserFromContext() = %q, want %q", gotUser, tt.wantUser)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/auth/... -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/auth/middleware.go`:
```go
// ABOUTME: Authentication middleware for fake Google API requests.
// ABOUTME: Parses Bearer tokens and extracts user identity for request context.

package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := extractUser(r.Header.Get("Authorization"))
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) string {
	user, ok := ctx.Value(userContextKey).(string)
	if !ok || user == "" {
		return "default"
	}
	return user
}

func extractUser(authHeader string) string {
	if authHeader == "" {
		return "default"
	}

	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return "default"
	}

	// Check for "user:" prefix
	if strings.HasPrefix(token, "user:") {
		return strings.TrimPrefix(token, "user:")
	}

	return token
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/auth/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/auth/
git commit -m "feat(auth): add middleware for Bearer token parsing"
```

---

## Task 8: Gmail HTTP Handlers

**Files:**
- Create: `internal/gmail/handlers.go`
- Create: `internal/gmail/handlers_test.go`

**Step 1: Write failing test for Gmail handlers**

Create `internal/gmail/handlers_test.go`:
```go
// ABOUTME: Tests for Gmail API HTTP handlers.
// ABOUTME: Verifies message listing and retrieval endpoints.

package gmail

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

func TestHandlers_ListMessages(t *testing.T) {
	dbPath := "test_gmail_handlers.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Test"})
	s.CreateGmailMessage(&store.GmailMessage{
		ID: "msg_1", UserID: "harper", ThreadID: "thr_1",
		LabelIDs: []string{"INBOX"}, Snippet: "Hello", InternalDate: 1733000000000,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/gmail/v1/users/me/messages", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	msgs, ok := resp["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Errorf("messages count = %d, want 1", len(msgs))
	}
}

func TestHandlers_GetMessage(t *testing.T) {
	dbPath := "test_gmail_handlers2.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Test"})
	s.CreateGmailMessage(&store.GmailMessage{
		ID: "msg_1", UserID: "harper", ThreadID: "thr_1",
		LabelIDs: []string{"INBOX"}, Snippet: "Hello", InternalDate: 1733000000000,
		Payload: `{"headers":[{"name":"Subject","value":"Test"}]}`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/gmail/v1/users/me/messages/msg_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["id"] != "msg_1" {
		t.Errorf("id = %v, want msg_1", resp["id"])
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/gmail/... -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/gmail/handlers.go`:
```go
// ABOUTME: HTTP handlers for Gmail API endpoints.
// ABOUTME: Implements message listing and retrieval with Google-compatible responses.

package gmail

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/gmail/v1/users/{userId}", func(r chi.Router) {
		r.Get("/messages", h.listMessages)
		r.Get("/messages/{messageId}", h.getMessage)
	})
}

func (h *Handlers) listMessages(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	maxResults := 100
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")
	query := r.URL.Query().Get("q")

	messages, nextToken, err := h.store.ListGmailMessages(userID, maxResults, pageToken, query)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	msgList := make([]map[string]any, len(messages))
	for i, m := range messages {
		msgList[i] = map[string]any{
			"id":       m.ID,
			"threadId": m.ThreadID,
		}
	}

	resp := map[string]any{
		"messages":           msgList,
		"resultSizeEstimate": len(messages),
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) getMessage(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}
	messageID := chi.URLParam(r, "messageId")

	msg, err := h.store.GetGmailMessage(userID, messageID)
	if err != nil {
		writeError(w, 404, "Message not found", "NOT_FOUND")
		return
	}

	// Parse payload JSON
	var payload map[string]any
	json.Unmarshal([]byte(msg.Payload), &payload)

	resp := map[string]any{
		"id":           msg.ID,
		"threadId":     msg.ThreadID,
		"labelIds":     msg.LabelIDs,
		"snippet":      msg.Snippet,
		"internalDate": strconv.FormatInt(msg.InternalDate, 10),
		"payload":      payload,
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, message, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	})
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/gmail/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/gmail/
git commit -m "feat(gmail): add HTTP handlers for messages"
```

---

## Task 9: Calendar HTTP Handlers

**Files:**
- Create: `internal/calendar/handlers.go`
- Create: `internal/calendar/handlers_test.go`

**Step 1: Write failing test for Calendar handlers**

Create `internal/calendar/handlers_test.go`:
```go
// ABOUTME: Tests for Calendar API HTTP handlers.
// ABOUTME: Verifies event listing and retrieval endpoints.

package calendar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

func TestHandlers_ListEvents(t *testing.T) {
	dbPath := "test_calendar_handlers.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary"})
	s.CreateCalendarEvent(&store.CalendarEvent{
		ID: "evt_1", CalendarID: "primary", Summary: "Coffee",
		StartTime: "2025-12-01T10:00:00Z", EndTime: "2025-12-01T11:00:00Z",
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/calendar/v3/calendars/primary/events", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Errorf("items count = %d, want 1", len(items))
	}
}

func TestHandlers_GetEvent(t *testing.T) {
	dbPath := "test_calendar_handlers2.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary"})
	s.CreateCalendarEvent(&store.CalendarEvent{
		ID: "evt_1", CalendarID: "primary", Summary: "Coffee",
		Description: "Discuss things",
		StartTime: "2025-12-01T10:00:00Z", EndTime: "2025-12-01T11:00:00Z",
		Attendees: `[{"email":"harper@example.com"}]`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/calendar/v3/calendars/primary/events/evt_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["summary"] != "Coffee" {
		t.Errorf("summary = %v, want Coffee", resp["summary"])
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/calendar/... -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/calendar/handlers.go`:
```go
// ABOUTME: HTTP handlers for Calendar API endpoints.
// ABOUTME: Implements event listing and retrieval with Google-compatible responses.

package calendar

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/store"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/calendar/v3/calendars/{calendarId}", func(r chi.Router) {
		r.Get("/events", h.listEvents)
		r.Get("/events/{eventId}", h.getEvent)
	})
}

func (h *Handlers) listEvents(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")

	maxResults := 250
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")
	timeMin := r.URL.Query().Get("timeMin")
	timeMax := r.URL.Query().Get("timeMax")

	events, nextToken, err := h.store.ListCalendarEvents(calendarID, maxResults, pageToken, timeMin, timeMax)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	items := make([]map[string]any, len(events))
	for i, e := range events {
		var attendees []any
		json.Unmarshal([]byte(e.Attendees), &attendees)

		items[i] = map[string]any{
			"id":          e.ID,
			"summary":     e.Summary,
			"description": e.Description,
			"start":       map[string]string{"dateTime": e.StartTime},
			"end":         map[string]string{"dateTime": e.EndTime},
			"attendees":   attendees,
		}
	}

	resp := map[string]any{
		"kind":  "calendar#events",
		"items": items,
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) getEvent(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")
	eventID := chi.URLParam(r, "eventId")

	evt, err := h.store.GetCalendarEvent(calendarID, eventID)
	if err != nil {
		writeError(w, 404, "Event not found", "NOT_FOUND")
		return
	}

	var attendees []any
	json.Unmarshal([]byte(evt.Attendees), &attendees)

	resp := map[string]any{
		"kind":        "calendar#event",
		"id":          evt.ID,
		"summary":     evt.Summary,
		"description": evt.Description,
		"start":       map[string]string{"dateTime": evt.StartTime},
		"end":         map[string]string{"dateTime": evt.EndTime},
		"attendees":   attendees,
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, message, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	})
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/calendar/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/calendar/
git commit -m "feat(calendar): add HTTP handlers for events"
```

---

## Task 10: People HTTP Handlers

**Files:**
- Create: `internal/people/handlers.go`
- Create: `internal/people/handlers_test.go`

**Step 1: Write failing test for People handlers**

Create `internal/people/handlers_test.go`:
```go
// ABOUTME: Tests for People API HTTP handlers.
// ABOUTME: Verifies contact search and retrieval endpoints.

package people

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

func TestHandlers_SearchContacts(t *testing.T) {
	dbPath := "test_people_handlers.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreatePerson(&store.Person{
		ResourceName: "people/c123",
		UserID:       "harper",
		Data:         `{"names":[{"displayName":"Alice Smith"}],"emailAddresses":[{"value":"alice@example.com"}]}`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/people/v1/people:searchContacts?query=alice", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
}

func TestHandlers_GetPerson(t *testing.T) {
	dbPath := "test_people_handlers2.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreatePerson(&store.Person{
		ResourceName: "people/c123",
		UserID:       "harper",
		Data:         `{"names":[{"displayName":"Alice Smith"}],"emailAddresses":[{"value":"alice@example.com"}]}`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/people/v1/people/c123", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["resourceName"] != "people/c123" {
		t.Errorf("resourceName = %v, want people/c123", resp["resourceName"])
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/people/... -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `internal/people/handlers.go`:
```go
// ABOUTME: HTTP handlers for People API endpoints.
// ABOUTME: Implements contact search and retrieval with Google-compatible responses.

package people

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/people/v1/people:searchContacts", h.searchContacts)
	r.Get("/people/v1/people/{resourceId}", h.getPerson)
}

func (h *Handlers) searchContacts(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())
	query := r.URL.Query().Get("query")

	pageSize := 10
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")

	people, nextToken, err := h.store.SearchPeople(userID, query, pageSize, pageToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	results := make([]map[string]any, len(people))
	for i, p := range people {
		var data map[string]any
		json.Unmarshal([]byte(p.Data), &data)

		person := map[string]any{
			"resourceName": p.ResourceName,
		}
		// Merge data fields
		for k, v := range data {
			person[k] = v
		}

		results[i] = map[string]any{
			"person": person,
		}
	}

	resp := map[string]any{
		"results": results,
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) getPerson(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())
	resourceID := chi.URLParam(r, "resourceId")
	resourceName := "people/" + resourceID

	// Handle full resource name format
	if strings.HasPrefix(resourceID, "people/") {
		resourceName = resourceID
	}

	person, err := h.store.GetPerson(userID, resourceName)
	if err != nil {
		writeError(w, 404, "Person not found", "NOT_FOUND")
		return
	}

	var data map[string]any
	json.Unmarshal([]byte(person.Data), &data)

	resp := map[string]any{
		"resourceName": person.ResourceName,
	}
	// Merge data fields
	for k, v := range data {
		resp[k] = v
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, message, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	})
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./internal/people/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/people/
git commit -m "feat(people): add HTTP handlers for contacts"
```

---

## Task 11: CLI and Server Wiring

**Files:**
- Create: `cmd/ish/main.go`
- Create: `cmd/ish/main_test.go`

**Step 1: Write failing test for server startup**

Create `cmd/ish/main_test.go`:
```go
// ABOUTME: Tests for CLI commands and server wiring.
// ABOUTME: Verifies health check and basic server functionality.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestServer_Healthz(t *testing.T) {
	dbPath := "test_main.db"
	defer os.Remove(dbPath)

	srv, err := newServer(dbPath)
	if err != nil {
		t.Fatalf("newServer() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./cmd/ish/... -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write minimal implementation**

Create `cmd/ish/main.go`:
```go
// ABOUTME: Entry point for the ISH fake Google API server.
// ABOUTME: Wires together store, auth, and API handlers with CLI commands.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/calendar"
	"github.com/2389/ish/internal/gmail"
	"github.com/2389/ish/internal/people"
	"github.com/2389/ish/internal/store"
)

var (
	port   string
	dbPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ish",
		Short: "Fake Google API server",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE:  runServe,
	}
	serveCmd.Flags().StringVarP(&port, "port", "p", getEnv("ISH_PORT", "9000"), "Port to listen on")
	serveCmd.Flags().StringVarP(&dbPath, "db", "d", getEnv("ISH_DB_PATH", "./ish.db"), "Database path")

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with test data",
		RunE:  runSeed,
	}
	seedCmd.Flags().StringVarP(&dbPath, "db", "d", getEnv("ISH_DB_PATH", "./ish.db"), "Database path")

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database (wipe and reseed)",
		RunE:  runReset,
	}
	resetCmd.Flags().StringVarP(&dbPath, "db", "d", getEnv("ISH_DB_PATH", "./ish.db"), "Database path")

	rootCmd.AddCommand(serveCmd, seedCmd, resetCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	srv, err := newServer(dbPath)
	if err != nil {
		return err
	}

	addr := ":" + port
	log.Printf("ISH server listening on %s", addr)
	log.Printf("Database: %s", dbPath)
	return http.ListenAndServe(addr, srv)
}

func newServer(dbPath string) (http.Handler, error) {
	s, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %w", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(auth.Middleware)

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// API handlers
	gmail.NewHandlers(s).RegisterRoutes(r)
	calendar.NewHandlers(s).RegisterRoutes(r)
	people.NewHandlers(s).RegisterRoutes(r)

	return r, nil
}

func runSeed(cmd *cobra.Command, args []string) error {
	s, err := store.New(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	return seedData(s)
}

func runReset(cmd *cobra.Command, args []string) error {
	// Remove existing database
	os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	return seedData(s)
}

func seedData(s *store.Store) error {
	// Create default user
	if err := s.CreateUser("harper"); err != nil {
		return err
	}
	log.Println("Created user: harper")

	// Gmail data
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Welcome to ISH"})
	s.CreateGmailThread(&store.GmailThread{ID: "thr_2", UserID: "harper", Snippet: "Meeting tomorrow"})

	messages := []store.GmailMessage{
		{ID: "msg_1", UserID: "harper", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "Welcome to ISH, your fake Google API server!", InternalDate: 1733000000000, Payload: `{"headers":[{"name":"From","value":"ish@example.com"},{"name":"Subject","value":"Welcome to ISH"}]}`},
		{ID: "msg_2", UserID: "harper", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "Getting started guide attached.", InternalDate: 1733000100000, Payload: `{"headers":[{"name":"From","value":"ish@example.com"},{"name":"Subject","value":"Re: Welcome to ISH"}]}`},
		{ID: "msg_3", UserID: "harper", ThreadID: "thr_2", LabelIDs: []string{"INBOX", "STARRED"}, Snippet: "Don't forget our meeting tomorrow at 10am.", InternalDate: 1733000200000, Payload: `{"headers":[{"name":"From","value":"alice@example.com"},{"name":"Subject","value":"Meeting tomorrow"}]}`},
		{ID: "msg_4", UserID: "harper", ThreadID: "thr_2", LabelIDs: []string{"INBOX"}, Snippet: "I'll bring the coffee!", InternalDate: 1733000300000, Payload: `{"headers":[{"name":"From","value":"bob@example.com"},{"name":"Subject","value":"Re: Meeting tomorrow"}]}`},
		{ID: "msg_5", UserID: "harper", ThreadID: "thr_2", LabelIDs: []string{"INBOX", "IMPORTANT"}, Snippet: "Agenda attached for review.", InternalDate: 1733000400000, Payload: `{"headers":[{"name":"From","value":"alice@example.com"},{"name":"Subject","value":"Re: Meeting tomorrow"}]}`},
	}
	for _, m := range messages {
		s.CreateGmailMessage(&m)
	}
	log.Printf("Created %d Gmail messages", len(messages))

	// Calendar data
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary Calendar"})

	events := []store.CalendarEvent{
		{ID: "evt_1", CalendarID: "primary", Summary: "Team Standup", Description: "Daily sync", StartTime: "2025-12-01T09:00:00Z", EndTime: "2025-12-01T09:30:00Z", Attendees: `[{"email":"harper@example.com"},{"email":"alice@example.com"}]`},
		{ID: "evt_2", CalendarID: "primary", Summary: "Project Review", Description: "Q4 review", StartTime: "2025-12-01T14:00:00Z", EndTime: "2025-12-01T15:00:00Z", Attendees: `[{"email":"harper@example.com"},{"email":"bob@example.com"}]`},
		{ID: "evt_3", CalendarID: "primary", Summary: "Coffee Chat", Description: "Casual sync", StartTime: "2025-12-02T10:00:00Z", EndTime: "2025-12-02T10:30:00Z", Attendees: `[{"email":"harper@example.com"}]`},
	}
	for _, e := range events {
		s.CreateCalendarEvent(&e)
	}
	log.Printf("Created %d Calendar events", len(events))

	// People data
	contacts := []store.Person{
		{ResourceName: "people/c1", UserID: "harper", Data: `{"names":[{"displayName":"Alice Smith"}],"emailAddresses":[{"value":"alice@example.com"}],"photos":[{"url":"https://example.com/alice.png"}]}`},
		{ResourceName: "people/c2", UserID: "harper", Data: `{"names":[{"displayName":"Bob Jones"}],"emailAddresses":[{"value":"bob@example.com"}],"photos":[{"url":"https://example.com/bob.png"}]}`},
		{ResourceName: "people/c3", UserID: "harper", Data: `{"names":[{"displayName":"Charlie Brown"}],"emailAddresses":[{"value":"charlie@example.com"}],"photos":[{"url":"https://example.com/charlie.png"}]}`},
	}
	for _, p := range contacts {
		s.CreatePerson(&p)
	}
	log.Printf("Created %d People contacts", len(contacts))

	log.Println("Seed complete!")
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./cmd/ish/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add cmd/ish/
git commit -m "feat(cli): add serve, seed, and reset commands"
```

---

## Task 12: End-to-End Integration Test

**Files:**
- Create: `e2e_test.go`

**Step 1: Write integration test**

Create `e2e_test.go` in project root:
```go
// ABOUTME: End-to-end integration tests for the ISH server.
// ABOUTME: Verifies full request/response flow across all APIs.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/calendar"
	"github.com/2389/ish/internal/gmail"
	"github.com/2389/ish/internal/people"
	"github.com/2389/ish/internal/store"
)

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	dbPath := "test_e2e.db"

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}

	// Seed test data
	s.CreateUser("harper")
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Test thread"})
	s.CreateGmailMessage(&store.GmailMessage{ID: "msg_1", UserID: "harper", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "Test message", InternalDate: 1733000000000, Payload: `{"headers":[]}`})
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary"})
	s.CreateCalendarEvent(&store.CalendarEvent{ID: "evt_1", CalendarID: "primary", Summary: "Test Event", StartTime: "2025-12-01T10:00:00Z", EndTime: "2025-12-01T11:00:00Z"})
	s.CreatePerson(&store.Person{ResourceName: "people/c1", UserID: "harper", Data: `{"names":[{"displayName":"Test Person"}]}`})

	r := chi.NewRouter()
	r.Use(auth.Middleware)
	gmail.NewHandlers(s).RegisterRoutes(r)
	calendar.NewHandlers(s).RegisterRoutes(r)
	people.NewHandlers(s).RegisterRoutes(r)

	srv := httptest.NewServer(r)

	cleanup := func() {
		srv.Close()
		s.Close()
		os.Remove(dbPath)
	}

	return srv, cleanup
}

func TestE2E_GmailFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client := srv.Client()

	// List messages
	req, _ := http.NewRequest("GET", srv.URL+"/gmail/v1/users/me/messages", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list messages error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("list status = %d, want 200", resp.StatusCode)
	}

	var listResp map[string]any
	json.NewDecoder(resp.Body).Decode(&listResp)
	msgs := listResp["messages"].([]any)
	if len(msgs) != 1 {
		t.Errorf("messages count = %d, want 1", len(msgs))
	}

	// Get single message
	req, _ = http.NewRequest("GET", srv.URL+"/gmail/v1/users/me/messages/msg_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("get message error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("get status = %d, want 200", resp.StatusCode)
	}
}

func TestE2E_CalendarFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client := srv.Client()

	// List events
	req, _ := http.NewRequest("GET", srv.URL+"/calendar/v3/calendars/primary/events", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list events error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("list status = %d, want 200", resp.StatusCode)
	}

	var listResp map[string]any
	json.NewDecoder(resp.Body).Decode(&listResp)
	items := listResp["items"].([]any)
	if len(items) != 1 {
		t.Errorf("items count = %d, want 1", len(items))
	}

	// Get single event
	req, _ = http.NewRequest("GET", srv.URL+"/calendar/v3/calendars/primary/events/evt_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("get event error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("get status = %d, want 200", resp.StatusCode)
	}
}

func TestE2E_PeopleFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client := srv.Client()

	// Search contacts
	req, _ := http.NewRequest("GET", srv.URL+"/people/v1/people:searchContacts?query=Test", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("search contacts error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("search status = %d, want 200", resp.StatusCode)
	}

	var searchResp map[string]any
	json.NewDecoder(resp.Body).Decode(&searchResp)
	results := searchResp["results"].([]any)
	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}

	// Get single person
	req, _ = http.NewRequest("GET", srv.URL+"/people/v1/people/c1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("get person error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("get status = %d, want 200", resp.StatusCode)
	}
}
```

**Step 2: Run tests**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go test ./... -v
```

Expected: ALL PASS

**Step 3: Commit**

```bash
git add e2e_test.go
git commit -m "test: add end-to-end integration tests"
```

---

## Task 13: Documentation - V2 Features

**Files:**
- Create: `docs/v2-roadmap.md`

**Step 1: Write V2 roadmap documentation**

Create `docs/v2-roadmap.md`:
```markdown
# ISH v2 Roadmap

Future enhancements planned for ISH (fake Google API server).

## Chaos Mode

Environment variables to inject failures for testing client resilience:

| Variable | Example | Effect |
|----------|---------|--------|
| `ISH_RATE_LIMIT_PERCENT` | `10` | 10% of requests return HTTP 429 |
| `ISH_RANDOM_500_PERCENT` | `2` | 2% of requests return HTTP 500 |
| `ISH_LATENCY_MS` | `100` | Add 100ms artificial latency |
| `ISH_LATENCY_JITTER_MS` | `50` | Add 0-50ms random jitter |

### Implementation Notes

- Middleware-based, applied before handlers
- Configurable per-endpoint or global
- Log when chaos is triggered for debugging

## Eventual Consistency Simulation

Simulate Google's distributed system behavior:

- Insert operations succeed immediately
- Inserted items don't appear in list queries for N milliseconds
- Configurable via `ISH_CONSISTENCY_DELAY_MS`

### Use Cases

- Test polling/retry logic in clients
- Verify UI handles "just created but not visible" states
- Stress test eventual consistency assumptions

## Write Operations

### Gmail
- `POST /gmail/v1/users/{userId}/messages/send` - Store message locally
- `POST /gmail/v1/users/{userId}/messages/{id}/modify` - Update labels
- `DELETE /gmail/v1/users/{userId}/messages/{id}` - Soft delete

### Calendar
- `POST /calendar/v3/calendars/{calendarId}/events` - Create event
- `PUT /calendar/v3/calendars/{calendarId}/events/{eventId}` - Update event
- `DELETE /calendar/v3/calendars/{calendarId}/events/{eventId}` - Delete event

### People
- `POST /people/v1/people:createContact` - Create contact
- `PATCH /people/v1/{resourceName}:updateContact` - Update contact
- `DELETE /people/v1/{resourceName}:deleteContact` - Delete contact

## Record/Replay Mode

Proxy mode for capturing real Google API traffic:

### Record Mode
```bash
ISH_MODE=record ISH_UPSTREAM=https://www.googleapis.com ish serve
```

- Forward requests to real Google
- Store request/response pairs with timestamps
- Save to `recordings/` directory as JSON

### Replay Mode
```bash
ISH_MODE=replay ISH_RECORDING_DIR=./recordings ish serve
```

- Match incoming requests to recorded responses
- Deterministic testing with real data shapes
- Useful for CI/CD without Google credentials

## Batch Operations

Support Google's batch endpoint pattern:

- `POST /batch` - Multiple operations in single request
- Multipart request body
- Individual status codes per operation

## Watch/Push Notifications

Webhook support for change notifications:

- `POST /gmail/v1/users/{userId}/watch` - Start watching
- `POST /gmail/v1/users/{userId}/stop` - Stop watching
- Configurable webhook delivery to localhost endpoints

## Labels Management

Full Gmail labels API:

- `GET /gmail/v1/users/{userId}/labels` - List labels
- `POST /gmail/v1/users/{userId}/labels` - Create label
- `PATCH /gmail/v1/users/{userId}/labels/{id}` - Update label
- `DELETE /gmail/v1/users/{userId}/labels/{id}` - Delete label

## ACL Support

Calendar sharing and permissions:

- `GET /calendar/v3/calendars/{calendarId}/acl` - List ACLs
- `POST /calendar/v3/calendars/{calendarId}/acl` - Add ACL
- `DELETE /calendar/v3/calendars/{calendarId}/acl/{ruleId}` - Remove ACL

## Metrics & Observability

- Prometheus metrics endpoint at `/metrics`
- Request counts, latencies, error rates
- Per-endpoint and per-user breakdowns
```

**Step 2: Commit**

```bash
git add docs/v2-roadmap.md
git commit -m "docs: add v2 roadmap with chaos mode and future features"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Initialize Go module | go.mod, go.sum |
| 2 | Store layer - schema | internal/store/store.go |
| 3 | Store layer - users | internal/store/store.go |
| 4 | Store layer - Gmail | internal/store/gmail.go |
| 5 | Store layer - Calendar | internal/store/calendar.go |
| 6 | Store layer - People | internal/store/people.go |
| 7 | Auth middleware | internal/auth/middleware.go |
| 8 | Gmail handlers | internal/gmail/handlers.go |
| 9 | Calendar handlers | internal/calendar/handlers.go |
| 10 | People handlers | internal/people/handlers.go |
| 11 | CLI and server | cmd/ish/main.go |
| 12 | E2E tests | e2e_test.go |
| 13 | V2 documentation | docs/v2-roadmap.md |

Total: 13 tasks, ~13 commits
