# Twilio Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a self-contained Twilio SMS and Voice API emulator plugin for ISH with async webhook delivery simulation.

**Architecture:** Follows ISH's DatabasePlugin interface pattern (like Discord, Google, OAuth). Auto-accept auth creates accounts on first request. Background goroutines deliver status webhooks with realistic delays (100ms-30s) matching production Twilio behavior.

**Tech Stack:** Go 1.21+, SQLite3, chi router, crypto/rand for SID generation, net/http for webhook delivery

---

## Task 1: Create Plugin Structure & Store Foundation

**Files:**
- Create: `plugins/twilio/plugin.go`
- Create: `plugins/twilio/store.go`
- Create: `plugins/twilio/store_test.go`

### Step 1: Write test for store initialization

Create `plugins/twilio/store_test.go`:

```go
// ABOUTME: Unit tests for Twilio plugin store layer
// ABOUTME: Tests account, message, call, and webhook operations

package twilio

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestNewTwilioStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store, err := NewTwilioStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	if store == nil {
		t.Fatal("Store is nil")
	}

	// Verify all tables were created
	tables := []string{
		"twilio_accounts",
		"twilio_phone_numbers",
		"twilio_messages",
		"twilio_calls",
		"twilio_webhook_configs",
		"twilio_webhook_queue",
	}
	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("table %s was not created", table)
		}
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/twilio
go test -v -run TestNewTwilioStore
```

Expected: FAIL with "no such file or directory" or "undefined: NewTwilioStore"

### Step 3: Write minimal store implementation

Create `plugins/twilio/store.go`:

```go
// ABOUTME: Database layer for Twilio plugin
// ABOUTME: Manages accounts, phone numbers, messages, calls, and webhook queue

package twilio

import (
	"database/sql"
)

type TwilioStore struct {
	db *sql.DB
}

func NewTwilioStore(db *sql.DB) (*TwilioStore, error) {
	store := &TwilioStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TwilioStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS twilio_accounts (
			account_sid TEXT PRIMARY KEY,
			auth_token TEXT NOT NULL,
			friendly_name TEXT,
			status TEXT DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS twilio_phone_numbers (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			phone_number TEXT NOT NULL,
			friendly_name TEXT,
			voice_url TEXT,
			voice_method TEXT DEFAULT 'POST',
			sms_url TEXT,
			sms_method TEXT DEFAULT 'POST',
			status_callback TEXT,
			status_callback_method TEXT DEFAULT 'POST',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_phone_numbers_account ON twilio_phone_numbers(account_sid)`,

		`CREATE TABLE IF NOT EXISTS twilio_messages (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			from_number TEXT NOT NULL,
			to_number TEXT NOT NULL,
			body TEXT,
			status TEXT DEFAULT 'queued',
			direction TEXT,
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date_sent TIMESTAMP,
			date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			num_segments INTEGER DEFAULT 1,
			price REAL,
			price_unit TEXT DEFAULT 'USD',
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_account ON twilio_messages(account_sid)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_status ON twilio_messages(status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_date ON twilio_messages(date_created)`,

		`CREATE TABLE IF NOT EXISTS twilio_calls (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			from_number TEXT NOT NULL,
			to_number TEXT NOT NULL,
			status TEXT DEFAULT 'initiated',
			direction TEXT,
			duration INTEGER,
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			answered_by TEXT,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_account ON twilio_calls(account_sid)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_status ON twilio_calls(status)`,

		`CREATE TABLE IF NOT EXISTS twilio_webhook_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_sid TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			event_type TEXT NOT NULL,
			url TEXT NOT NULL,
			method TEXT DEFAULT 'POST',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_configs_account ON twilio_webhook_configs(account_sid)`,

		`CREATE TABLE IF NOT EXISTS twilio_webhook_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_sid TEXT NOT NULL,
			webhook_url TEXT NOT NULL,
			payload TEXT NOT NULL,
			scheduled_at TIMESTAMP NOT NULL,
			delivered_at TIMESTAMP,
			status TEXT DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_queue_schedule ON twilio_webhook_queue(scheduled_at, status)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}
```

### Step 4: Run test to verify it passes

```bash
cd plugins/twilio
go test -v -run TestNewTwilioStore
```

Expected: PASS

### Step 5: Write plugin skeleton

Create `plugins/twilio/plugin.go`:

```go
// ABOUTME: Twilio SMS and Voice API plugin for ISH
// ABOUTME: Simulates Twilio Messaging and Voice APIs with webhook callbacks

package twilio

import (
	"context"
	"database/sql"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&TwilioPlugin{})
}

type TwilioPlugin struct {
	store *TwilioStore
}

func (p *TwilioPlugin) Name() string {
	return "twilio"
}

func (p *TwilioPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Twilio plugin operational",
	}
}

func (p *TwilioPlugin) RegisterRoutes(r chi.Router) {
	// Routes will be added in Task 3
}

func (p *TwilioPlugin) RegisterAuth(r chi.Router) {
	// Twilio uses HTTP Basic Auth, handled per-request
}

func (p *TwilioPlugin) Schema() core.PluginSchema {
	// Schema will be added in Task 5
	return core.PluginSchema{}
}

func (p *TwilioPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// Seeding will be added in Task 6
	return core.SeedData{}, nil
}

func (p *TwilioPlugin) ValidateToken(token string) bool {
	// Token validation will be added in Task 2
	return false
}

func (p *TwilioPlugin) SetDB(db *sql.DB) error {
	store, err := NewTwilioStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}
```

### Step 6: Run build to verify compilation

```bash
cd /Users/harper/Public/src/2389/ish
go build ./...
```

Expected: Build succeeds

### Step 7: Commit Task 1

```bash
git add plugins/twilio/plugin.go plugins/twilio/store.go plugins/twilio/store_test.go
git commit -m "feat(twilio): add plugin structure and store foundation

- Implement TwilioPlugin skeleton with DatabasePlugin interface
- Create store with 6 tables: accounts, phone_numbers, messages, calls, webhook_configs, webhook_queue
- Add store initialization test
- Follow Discord plugin pattern for self-contained architecture"
```

---

## Task 2: Implement Account Auto-Creation & Auth

**Files:**
- Modify: `plugins/twilio/store.go` (add account methods)
- Modify: `plugins/twilio/plugin.go` (add auth middleware)
- Modify: `plugins/twilio/store_test.go` (add auth tests)

### Step 1: Write test for account auto-creation

Add to `plugins/twilio/store_test.go`:

```go
func TestGetOrCreateAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewTwilioStore(db)

	// First call should create
	account, err := store.GetOrCreateAccount("AC123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	if account.AccountSid != "AC123" {
		t.Fatalf("Expected SID 'AC123', got '%s'", account.AccountSid)
	}
	if account.AuthToken == "" {
		t.Fatal("Auth token should be generated")
	}

	originalToken := account.AuthToken

	// Second call should return same account
	account2, err := store.GetOrCreateAccount("AC123")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}
	if account2.AuthToken != originalToken {
		t.Fatal("Auth token should not change on second call")
	}
}

func TestValidateAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewTwilioStore(db)

	account, _ := store.GetOrCreateAccount("AC456")

	// Valid credentials
	if !store.ValidateAccount("AC456", account.AuthToken) {
		t.Fatal("Valid credentials should pass")
	}

	// Invalid token
	if store.ValidateAccount("AC456", "wrongtoken") {
		t.Fatal("Invalid token should fail")
	}

	// Invalid account
	if store.ValidateAccount("ACnonexistent", "anytoken") {
		t.Fatal("Nonexistent account should fail")
	}
}
```

### Step 2: Run tests to verify they fail

```bash
cd plugins/twilio
go test -v -run "TestGetOrCreateAccount|TestValidateAccount"
```

Expected: FAIL with "undefined: store.GetOrCreateAccount"

### Step 3: Implement account types and methods in store

Add to `plugins/twilio/store.go`:

```go
import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Account struct {
	AccountSid   string
	AuthToken    string
	FriendlyName string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func generateAuthToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *TwilioStore) GetOrCreateAccount(accountSid string) (*Account, error) {
	// Try to get existing account
	var account Account
	err := s.db.QueryRow(`
		SELECT account_sid, auth_token, friendly_name, status, created_at, updated_at
		FROM twilio_accounts
		WHERE account_sid = ?
	`, accountSid).Scan(
		&account.AccountSid,
		&account.AuthToken,
		&account.FriendlyName,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err == nil {
		return &account, nil
	}

	// Account doesn't exist, create it
	authToken, err := generateAuthToken()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_accounts (account_sid, auth_token)
		VALUES (?, ?)
	`, accountSid, authToken)
	if err != nil {
		return nil, err
	}

	// Return the created account
	return s.GetOrCreateAccount(accountSid)
}

func (s *TwilioStore) ValidateAccount(accountSid, authToken string) bool {
	var storedToken string
	err := s.db.QueryRow(`
		SELECT auth_token FROM twilio_accounts
		WHERE account_sid = ? AND status = 'active'
	`, accountSid).Scan(&storedToken)

	if err != nil {
		return false
	}

	return storedToken == authToken
}
```

### Step 4: Run tests to verify they pass

```bash
cd plugins/twilio
go test -v -run "TestGetOrCreateAccount|TestValidateAccount"
```

Expected: PASS

### Step 5: Implement auth middleware in plugin

Add to `plugins/twilio/plugin.go`:

```go
import (
	"encoding/base64"
	"strings"
)

// extractBasicAuth extracts username and password from HTTP Basic Auth header
func extractBasicAuth(authHeader string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
	if err != nil {
		return "", "", false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return "", "", false
	}

	return credentials[0], credentials[1], true
}

// requireAuth middleware validates HTTP Basic Auth (Account SID + Auth Token)
func (p *TwilioPlugin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"code": 20003, "message": "Authenticate"}`, http.StatusUnauthorized)
			return
		}

		accountSid, authToken, ok := extractBasicAuth(authHeader)
		if !ok {
			http.Error(w, `{"code": 20003, "message": "Authenticate"}`, http.StatusUnauthorized)
			return
		}

		// Auto-create account if it doesn't exist
		if _, err := p.store.GetOrCreateAccount(accountSid); err != nil {
			http.Error(w, `{"code": 20005, "message": "Internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Validate credentials
		if !p.store.ValidateAccount(accountSid, authToken) {
			http.Error(w, `{"code": 20003, "message": "Authenticate"}`, http.StatusUnauthorized)
			return
		}

		// Store account SID in context for handlers
		ctx := r.Context()
		ctx = context.WithValue(ctx, "account_sid", accountSid)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (p *TwilioPlugin) ValidateToken(token string) bool {
	// Token validation happens per-request via HTTP Basic Auth
	// This method is not used for Twilio auth pattern
	return false
}
```

Add import to top of `plugins/twilio/plugin.go`:

```go
import (
	"context"
	"net/http"
	// ... existing imports
)
```

### Step 6: Run build to verify compilation

```bash
cd /Users/harper/Public/src/2389/ish
go build ./...
```

Expected: Build succeeds

### Step 7: Commit Task 2

```bash
git add plugins/twilio/store.go plugins/twilio/plugin.go plugins/twilio/store_test.go
git commit -m "feat(twilio): implement HTTP Basic Auth with account auto-creation

- Add GetOrCreateAccount method with random token generation
- Add ValidateAccount method for credential checking
- Implement requireAuth middleware for HTTP Basic Auth
- Account SID is username, auth token is password
- Auto-create accounts on first request (like Discord webhooks)"
```

---

## Task 3: Implement SMS API Handlers

**Files:**
- Create: `plugins/twilio/handlers.go`
- Create: `plugins/twilio/handlers_test.go`
- Modify: `plugins/twilio/store.go` (add message methods)
- Modify: `plugins/twilio/plugin.go` (register routes)

### Step 1: Write test for sending SMS

Create `plugins/twilio/handlers_test.go`:

```go
// ABOUTME: Unit tests for Twilio API HTTP handlers
// ABOUTME: Tests SMS and Voice API endpoints

package twilio

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestPlugin(t *testing.T) (*TwilioPlugin, *sql.DB) {
	db := setupTestDB(t)
	plugin := &TwilioPlugin{}
	if err := plugin.SetDB(db); err != nil {
		t.Fatalf("Failed to set DB: %v", err)
	}
	return plugin, db
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestSendMessage(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Create account first
	account, _ := plugin.store.GetOrCreateAccount("AC123")

	// Prepare request
	form := url.Values{}
	form.Set("To", "+15551234567")
	form.Set("From", "+15559876543")
	form.Set("Body", "Test message")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC123/Messages.json", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("AC123", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.sendMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.HasPrefix(response["sid"].(string), "SM") {
		t.Fatalf("Expected SID to start with SM, got %s", response["sid"])
	}

	if response["to"] != "+15551234567" {
		t.Fatalf("Expected to +15551234567, got %s", response["to"])
	}

	if response["status"] != "queued" {
		t.Fatalf("Expected status queued, got %s", response["status"])
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/twilio
go test -v -run TestSendMessage
```

Expected: FAIL with "undefined: plugin.sendMessage"

### Step 3: Implement message types and store methods

Add to `plugins/twilio/store.go`:

```go
type Message struct {
	Sid         string
	AccountSid  string
	FromNumber  string
	ToNumber    string
	Body        string
	Status      string
	Direction   string
	DateCreated time.Time
	DateSent    *time.Time
	DateUpdated time.Time
	NumSegments int
	Price       float64
	PriceUnit   string
}

func generateSID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

func (s *TwilioStore) CreateMessage(accountSid, from, to, body string) (*Message, error) {
	sid, err := generateSID("SM")
	if err != nil {
		return nil, err
	}

	// Calculate segments (160 chars per segment)
	numSegments := (len(body) + 159) / 160
	if numSegments == 0 {
		numSegments = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_messages (sid, account_sid, from_number, to_number, body, status, direction, num_segments, price, price_unit)
		VALUES (?, ?, ?, ?, ?, 'queued', 'outbound-api', ?, ?, 'USD')
	`, sid, accountSid, from, to, body, numSegments, float64(numSegments)*0.0075)

	if err != nil {
		return nil, err
	}

	return s.GetMessage(sid)
}

func (s *TwilioStore) GetMessage(sid string) (*Message, error) {
	var msg Message
	var dateSent sql.NullTime

	err := s.db.QueryRow(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		WHERE sid = ?
	`, sid).Scan(
		&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
		&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
		&msg.NumSegments, &msg.Price, &msg.PriceUnit,
	)

	if err != nil {
		return nil, err
	}

	if dateSent.Valid {
		msg.DateSent = &dateSent.Time
	}

	return &msg, nil
}

func (s *TwilioStore) UpdateMessageStatus(sid, status string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE twilio_messages
		SET status = ?, date_updated = ?, date_sent = CASE WHEN ? IN ('sent', 'delivered') AND date_sent IS NULL THEN ? ELSE date_sent END
		WHERE sid = ?
	`, status, now, status, now, sid)
	return err
}
```

### Step 4: Implement sendMessage handler

Create `plugins/twilio/handlers.go`:

```go
// ABOUTME: HTTP handlers for Twilio API endpoints
// ABOUTME: Implements SMS and Voice API routes

package twilio

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (p *TwilioPlugin) sendMessage(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter")
		return
	}

	to := r.FormValue("To")
	from := r.FormValue("From")
	body := r.FormValue("Body")

	if to == "" || from == "" {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter To or From")
		return
	}

	message, err := p.store.CreateMessage(accountSid, from, to, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(messageToResponse(message))
}

func messageToResponse(msg *Message) map[string]interface{} {
	response := map[string]interface{}{
		"sid":           msg.Sid,
		"account_sid":   msg.AccountSid,
		"from":          msg.FromNumber,
		"to":            msg.ToNumber,
		"body":          msg.Body,
		"status":        msg.Status,
		"direction":     msg.Direction,
		"date_created":  msg.DateCreated.Format(time.RFC1123Z),
		"date_updated":  msg.DateUpdated.Format(time.RFC1123Z),
		"num_segments":  msg.NumSegments,
		"price":         msg.Price,
		"price_unit":    msg.PriceUnit,
		"error_code":    nil,
		"error_message": nil,
	}

	if msg.DateSent != nil {
		response["date_sent"] = msg.DateSent.Format(time.RFC1123Z)
	} else {
		response["date_sent"] = nil
	}

	return response
}

func writeError(w http.ResponseWriter, statusCode, errorCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    errorCode,
		"message": message,
		"status":  statusCode,
	})
}
```

### Step 5: Register SMS routes in plugin

Modify `plugins/twilio/plugin.go` RegisterRoutes method:

```go
func (p *TwilioPlugin) RegisterRoutes(r chi.Router) {
	// SMS API
	r.Route("/2010-04-01/Accounts/{AccountSid}/Messages.json", func(r chi.Router) {
		r.Post("/", p.requireAuth(p.sendMessage))
		r.Get("/", p.requireAuth(p.listMessages))
	})
	r.Get("/2010-04-01/Accounts/{AccountSid}/Messages/{MessageSid}.json", p.requireAuth(p.getMessage))
}
```

### Step 6: Run test to verify it passes

```bash
cd plugins/twilio
go test -v -run TestSendMessage
```

Expected: PASS

### Step 7: Implement list and get message handlers

Add to `plugins/twilio/handlers.go`:

```go
import (
	"strconv"
)

func (p *TwilioPlugin) getMessage(w http.ResponseWriter, r *http.Request) {
	messageSid := chi.URLParam(r, "MessageSid")

	message, err := p.store.GetMessage(messageSid)
	if err != nil {
		writeError(w, http.StatusNotFound, 20404, "Message not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messageToResponse(message))
}

func (p *TwilioPlugin) listMessages(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	pageSize := 50
	if ps := r.URL.Query().Get("PageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	messages, err := p.store.ListMessages(accountSid, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	responseMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		responseMessages[i] = messageToResponse(&msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": responseMessages,
		"page":     0,
		"page_size": pageSize,
	})
}
```

Add to `plugins/twilio/store.go`:

```go
func (s *TwilioStore) ListMessages(accountSid string, limit int) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		WHERE account_sid = ?
		ORDER BY date_created DESC
		LIMIT ?
	`, accountSid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var dateSent sql.NullTime

		err := rows.Scan(
			&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
			&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
			&msg.NumSegments, &msg.Price, &msg.PriceUnit,
		)
		if err != nil {
			return nil, err
		}

		if dateSent.Valid {
			msg.DateSent = &dateSent.Time
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
```

### Step 8: Run all tests

```bash
cd plugins/twilio
go test -v
```

Expected: All tests PASS

### Step 9: Commit Task 3

```bash
git add plugins/twilio/handlers.go plugins/twilio/handlers_test.go plugins/twilio/store.go plugins/twilio/plugin.go
git commit -m "feat(twilio): implement SMS API endpoints

- Add POST /Messages.json for sending SMS
- Add GET /Messages.json for listing messages
- Add GET /Messages/{sid}.json for getting message details
- Implement message creation with SID generation (SM prefix)
- Calculate segments based on 160 char limit
- Return Twilio-formatted JSON responses
- Add comprehensive handler tests"
```

---

## Task 4: Implement Voice API Handlers

**Files:**
- Modify: `plugins/twilio/handlers.go`
- Modify: `plugins/twilio/store.go` (add call methods)
- Modify: `plugins/twilio/handlers_test.go` (add call tests)
- Modify: `plugins/twilio/plugin.go` (register call routes)

### Step 1: Write test for initiating call

Add to `plugins/twilio/handlers_test.go`:

```go
func TestInitiateCall(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC456")

	form := url.Values{}
	form.Set("To", "+15551234567")
	form.Set("From", "+15559876543")
	form.Set("Url", "http://example.com/twiml")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC456/Calls.json", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("AC456", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.initiateCall)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	if !strings.HasPrefix(response["sid"].(string), "CA") {
		t.Fatalf("Expected SID to start with CA, got %s", response["sid"])
	}

	if response["status"] != "initiated" {
		t.Fatalf("Expected status initiated, got %s", response["status"])
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/twilio
go test -v -run TestInitiateCall
```

Expected: FAIL with "undefined: plugin.initiateCall"

### Step 3: Implement call types and store methods

Add to `plugins/twilio/store.go`:

```go
type Call struct {
	Sid         string
	AccountSid  string
	FromNumber  string
	ToNumber    string
	Status      string
	Direction   string
	Duration    *int
	DateCreated time.Time
	DateUpdated time.Time
	AnsweredBy  string
}

func (s *TwilioStore) CreateCall(accountSid, from, to string) (*Call, error) {
	sid, err := generateSID("CA")
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_calls (sid, account_sid, from_number, to_number, status, direction)
		VALUES (?, ?, ?, ?, 'initiated', 'outbound-api')
	`, sid, accountSid, from, to)

	if err != nil {
		return nil, err
	}

	return s.GetCall(sid)
}

func (s *TwilioStore) GetCall(sid string) (*Call, error) {
	var call Call
	var duration sql.NullInt64

	err := s.db.QueryRow(`
		SELECT sid, account_sid, from_number, to_number, status, direction,
		       duration, date_created, date_updated, answered_by
		FROM twilio_calls
		WHERE sid = ?
	`, sid).Scan(
		&call.Sid, &call.AccountSid, &call.FromNumber, &call.ToNumber,
		&call.Status, &call.Direction, &duration, &call.DateCreated,
		&call.DateUpdated, &call.AnsweredBy,
	)

	if err != nil {
		return nil, err
	}

	if duration.Valid {
		dur := int(duration.Int64)
		call.Duration = &dur
	}

	return &call, nil
}

func (s *TwilioStore) UpdateCallStatus(sid, status string, duration *int) error {
	if duration != nil {
		_, err := s.db.Exec(`
			UPDATE twilio_calls
			SET status = ?, date_updated = ?, duration = ?
			WHERE sid = ?
		`, status, time.Now(), *duration, sid)
		return err
	}

	_, err := s.db.Exec(`
		UPDATE twilio_calls
		SET status = ?, date_updated = ?
		WHERE sid = ?
	`, status, time.Now(), sid)
	return err
}

func (s *TwilioStore) ListCalls(accountSid string, limit int) ([]Call, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, status, direction,
		       duration, date_created, date_updated, answered_by
		FROM twilio_calls
		WHERE account_sid = ?
		ORDER BY date_created DESC
		LIMIT ?
	`, accountSid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []Call
	for rows.Next() {
		var call Call
		var duration sql.NullInt64

		err := rows.Scan(
			&call.Sid, &call.AccountSid, &call.FromNumber, &call.ToNumber,
			&call.Status, &call.Direction, &duration, &call.DateCreated,
			&call.DateUpdated, &call.AnsweredBy,
		)
		if err != nil {
			return nil, err
		}

		if duration.Valid {
			dur := int(duration.Int64)
			call.Duration = &dur
		}

		calls = append(calls, call)
	}

	return calls, nil
}
```

### Step 4: Implement call handlers

Add to `plugins/twilio/handlers.go`:

```go
func (p *TwilioPlugin) initiateCall(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter")
		return
	}

	to := r.FormValue("To")
	from := r.FormValue("From")
	url := r.FormValue("Url")

	if to == "" || from == "" || url == "" {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter To, From, or Url")
		return
	}

	call, err := p.store.CreateCall(accountSid, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(callToResponse(call))
}

func (p *TwilioPlugin) getCall(w http.ResponseWriter, r *http.Request) {
	callSid := chi.URLParam(r, "CallSid")

	call, err := p.store.GetCall(callSid)
	if err != nil {
		writeError(w, http.StatusNotFound, 20404, "Call not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(callToResponse(call))
}

func (p *TwilioPlugin) listCalls(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	pageSize := 50
	if ps := r.URL.Query().Get("PageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	calls, err := p.store.ListCalls(accountSid, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	responseCalls := make([]map[string]interface{}, len(calls))
	for i, call := range calls {
		responseCalls[i] = callToResponse(&call)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calls":     responseCalls,
		"page":      0,
		"page_size": pageSize,
	})
}

func callToResponse(call *Call) map[string]interface{} {
	response := map[string]interface{}{
		"sid":          call.Sid,
		"account_sid":  call.AccountSid,
		"from":         call.FromNumber,
		"to":           call.ToNumber,
		"status":       call.Status,
		"direction":    call.Direction,
		"date_created": call.DateCreated.Format(time.RFC1123Z),
		"date_updated": call.DateUpdated.Format(time.RFC1123Z),
		"answered_by":  call.AnsweredBy,
	}

	if call.Duration != nil {
		response["duration"] = strconv.Itoa(*call.Duration)
	} else {
		response["duration"] = nil
	}

	return response
}
```

### Step 5: Register call routes

Modify `plugins/twilio/plugin.go` RegisterRoutes:

```go
func (p *TwilioPlugin) RegisterRoutes(r chi.Router) {
	// SMS API
	r.Route("/2010-04-01/Accounts/{AccountSid}/Messages.json", func(r chi.Router) {
		r.Post("/", p.requireAuth(p.sendMessage))
		r.Get("/", p.requireAuth(p.listMessages))
	})
	r.Get("/2010-04-01/Accounts/{AccountSid}/Messages/{MessageSid}.json", p.requireAuth(p.getMessage))

	// Voice API
	r.Route("/2010-04-01/Accounts/{AccountSid}/Calls.json", func(r chi.Router) {
		r.Post("/", p.requireAuth(p.initiateCall))
		r.Get("/", p.requireAuth(p.listCalls))
	})
	r.Get("/2010-04-01/Accounts/{AccountSid}/Calls/{CallSid}.json", p.requireAuth(p.getCall))

	// Phone Numbers API
	r.Get("/2010-04-01/Accounts/{AccountSid}/IncomingPhoneNumbers.json", p.requireAuth(p.listPhoneNumbers))
}
```

### Step 6: Run tests

```bash
cd plugins/twilio
go test -v
```

Expected: All tests PASS

### Step 7: Commit Task 4

```bash
git add plugins/twilio/handlers.go plugins/twilio/store.go plugins/twilio/handlers_test.go plugins/twilio/plugin.go
git commit -m "feat(twilio): implement Voice API endpoints

- Add POST /Calls.json for initiating calls
- Add GET /Calls.json for listing calls
- Add GET /Calls/{sid}.json for call details
- Implement call creation with CA prefix SIDs
- Track call status and duration
- Return Twilio-formatted call responses"
```

---

## Task 5: Implement Phone Numbers & Admin Schema

**Files:**
- Modify: `plugins/twilio/store.go` (add phone number methods)
- Modify: `plugins/twilio/handlers.go` (add phone number handler)
- Create: `plugins/twilio/schema.go`
- Modify: `plugins/twilio/plugin.go` (implement Schema method)

### Step 1: Implement phone number store methods

Add to `plugins/twilio/store.go`:

```go
type PhoneNumber struct {
	Sid                    string
	AccountSid             string
	PhoneNumber            string
	FriendlyName           string
	VoiceURL               string
	VoiceMethod            string
	SmsURL                 string
	SmsMethod              string
	StatusCallback         string
	StatusCallbackMethod   string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (s *TwilioStore) CreatePhoneNumber(accountSid, phoneNumber, friendlyName string) (*PhoneNumber, error) {
	sid, err := generateSID("PN")
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_phone_numbers (sid, account_sid, phone_number, friendly_name)
		VALUES (?, ?, ?, ?)
	`, sid, accountSid, phoneNumber, friendlyName)

	if err != nil {
		return nil, err
	}

	return s.GetPhoneNumber(sid)
}

func (s *TwilioStore) GetPhoneNumber(sid string) (*PhoneNumber, error) {
	var pn PhoneNumber
	err := s.db.QueryRow(`
		SELECT sid, account_sid, phone_number, friendly_name, voice_url, voice_method,
		       sms_url, sms_method, status_callback, status_callback_method,
		       created_at, updated_at
		FROM twilio_phone_numbers
		WHERE sid = ?
	`, sid).Scan(
		&pn.Sid, &pn.AccountSid, &pn.PhoneNumber, &pn.FriendlyName,
		&pn.VoiceURL, &pn.VoiceMethod, &pn.SmsURL, &pn.SmsMethod,
		&pn.StatusCallback, &pn.StatusCallbackMethod,
		&pn.CreatedAt, &pn.UpdatedAt,
	)
	return &pn, err
}

func (s *TwilioStore) ListPhoneNumbers(accountSid string) ([]PhoneNumber, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, phone_number, friendly_name, voice_url, voice_method,
		       sms_url, sms_method, status_callback, status_callback_method,
		       created_at, updated_at
		FROM twilio_phone_numbers
		WHERE account_sid = ?
		ORDER BY created_at DESC
	`, accountSid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var numbers []PhoneNumber
	for rows.Next() {
		var pn PhoneNumber
		err := rows.Scan(
			&pn.Sid, &pn.AccountSid, &pn.PhoneNumber, &pn.FriendlyName,
			&pn.VoiceURL, &pn.VoiceMethod, &pn.SmsURL, &pn.SmsMethod,
			&pn.StatusCallback, &pn.StatusCallbackMethod,
			&pn.CreatedAt, &pn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, pn)
	}

	return numbers, nil
}
```

### Step 2: Implement phone number list handler

Add to `plugins/twilio/handlers.go`:

```go
func (p *TwilioPlugin) listPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	numbers, err := p.store.ListPhoneNumbers(accountSid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	responseNumbers := make([]map[string]interface{}, len(numbers))
	for i, num := range numbers {
		responseNumbers[i] = map[string]interface{}{
			"sid":                      num.Sid,
			"account_sid":              num.AccountSid,
			"phone_number":             num.PhoneNumber,
			"friendly_name":            num.FriendlyName,
			"voice_url":                num.VoiceURL,
			"voice_method":             num.VoiceMethod,
			"sms_url":                  num.SmsURL,
			"sms_method":               num.SmsMethod,
			"status_callback":          num.StatusCallback,
			"status_callback_method":   num.StatusCallbackMethod,
			"date_created":             num.CreatedAt.Format(time.RFC1123Z),
			"date_updated":             num.UpdatedAt.Format(time.RFC1123Z),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"incoming_phone_numbers": responseNumbers,
	})
}
```

### Step 3: Create admin schema

Create `plugins/twilio/schema.go`:

```go
// ABOUTME: Admin UI schema definitions for Twilio plugin
// ABOUTME: Defines resources, fields, and actions for schema-driven UI

package twilio

import "github.com/2389/ish/plugins/core"

func (p *TwilioPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Name:        "twilio",
		DisplayName: "Twilio",
		Description: "Twilio SMS and Voice API emulator",
		Resources: []core.ResourceSchema{
			{
				Name:        "accounts",
				DisplayName: "Accounts",
				Fields: []core.FieldSchema{
					{Name: "account_sid", DisplayName: "Account SID", Type: "string", PrimaryKey: true},
					{Name: "auth_token", DisplayName: "Auth Token", Type: "string", Sensitive: true},
					{Name: "friendly_name", DisplayName: "Friendly Name", Type: "string"},
					{Name: "status", DisplayName: "Status", Type: "string"},
					{Name: "created_at", DisplayName: "Created", Type: "timestamp"},
				},
			},
			{
				Name:        "messages",
				DisplayName: "Messages",
				Fields: []core.FieldSchema{
					{Name: "sid", DisplayName: "Message SID", Type: "string", PrimaryKey: true},
					{Name: "account_sid", DisplayName: "Account", Type: "string"},
					{Name: "from_number", DisplayName: "From", Type: "string"},
					{Name: "to_number", DisplayName: "To", Type: "string"},
					{Name: "body", DisplayName: "Body", Type: "text"},
					{Name: "status", DisplayName: "Status", Type: "string"},
					{Name: "direction", DisplayName: "Direction", Type: "string"},
					{Name: "date_created", DisplayName: "Created", Type: "timestamp"},
				},
			},
			{
				Name:        "calls",
				DisplayName: "Calls",
				Fields: []core.FieldSchema{
					{Name: "sid", DisplayName: "Call SID", Type: "string", PrimaryKey: true},
					{Name: "account_sid", DisplayName: "Account", Type: "string"},
					{Name: "from_number", DisplayName: "From", Type: "string"},
					{Name: "to_number", DisplayName: "To", Type: "string"},
					{Name: "status", DisplayName: "Status", Type: "string"},
					{Name: "duration", DisplayName: "Duration (s)", Type: "integer"},
					{Name: "date_created", DisplayName: "Created", Type: "timestamp"},
				},
			},
			{
				Name:        "phone_numbers",
				DisplayName: "Phone Numbers",
				Fields: []core.FieldSchema{
					{Name: "sid", DisplayName: "SID", Type: "string", PrimaryKey: true},
					{Name: "account_sid", DisplayName: "Account", Type: "string"},
					{Name: "phone_number", DisplayName: "Phone Number", Type: "string"},
					{Name: "friendly_name", DisplayName: "Friendly Name", Type: "string"},
					{Name: "voice_url", DisplayName: "Voice URL", Type: "string"},
					{Name: "sms_url", DisplayName: "SMS URL", Type: "string"},
					{Name: "created_at", DisplayName: "Created", Type: "timestamp"},
				},
			},
		},
	}
}
```

### Step 4: Run build

```bash
cd /Users/harper/Public/src/2389/ish
go build ./...
```

Expected: Build succeeds

### Step 5: Commit Task 5

```bash
git add plugins/twilio/store.go plugins/twilio/handlers.go plugins/twilio/schema.go plugins/twilio/plugin.go
git commit -m "feat(twilio): add phone numbers API and admin schema

- Implement phone number store methods with PN prefix SIDs
- Add GET /IncomingPhoneNumbers.json endpoint
- Create admin UI schema for accounts, messages, calls, phone numbers
- Schema-driven UI will auto-generate forms and lists"
```

---

## Task 6: Implement Seeding

**Files:**
- Modify: `plugins/twilio/plugin.go` (implement Seed method)
- Create: `plugins/twilio/seed.go`

### Step 1: Create seed implementation

Create `plugins/twilio/seed.go`:

```go
// ABOUTME: Test data seeding for Twilio plugin
// ABOUTME: Generates realistic accounts, phone numbers, messages, and calls

package twilio

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/2389/ish/plugins/core"
)

func (p *TwilioPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	var accounts, phoneNumbers, messages, calls int

	switch size {
	case "small":
		accounts, phoneNumbers, messages, calls = 1, 3, 10, 5
	case "medium":
		accounts, phoneNumbers, messages, calls = 3, 10, 50, 20
	case "large":
		accounts, phoneNumbers, messages, calls = 10, 30, 200, 100
	default:
		accounts, phoneNumbers, messages, calls = 1, 3, 10, 5
	}

	// Create accounts
	accountSids := make([]string, accounts)
	for i := 0; i < accounts; i++ {
		sid := fmt.Sprintf("AC%032d", i+1)
		accountSids[i] = sid
		if _, err := p.store.GetOrCreateAccount(sid); err != nil {
			return core.SeedData{}, err
		}
	}

	// Create phone numbers
	phoneNumberList := make([]string, phoneNumbers)
	numbersPerAccount := phoneNumbers / accounts
	for i := 0; i < phoneNumbers; i++ {
		accountIdx := i / numbersPerAccount
		if accountIdx >= accounts {
			accountIdx = accounts - 1
		}

		phoneNum := fmt.Sprintf("+1555%07d", 1000000+i)
		phoneNumberList[i] = phoneNum
		friendlyName := fmt.Sprintf("Test Number %d", i+1)

		if _, err := p.store.CreatePhoneNumber(accountSids[accountIdx], phoneNum, friendlyName); err != nil {
			return core.SeedData{}, err
		}
	}

	// Create messages
	messageBodies := []string{
		"Your verification code is 123456",
		"Your package has been shipped",
		"Reminder: You have an appointment tomorrow at 2 PM",
		"Thanks for your order! Order #12345 is confirmed",
		"Your account balance is $100.00",
		"Hello! How can we help you today?",
		"Your reservation is confirmed for 6 PM",
		"Security alert: New login from Chrome",
	}

	for i := 0; i < messages; i++ {
		accountIdx := rand.Intn(accounts)
		fromIdx := rand.Intn(phoneNumbers)
		toPhone := fmt.Sprintf("+1555%07d", 2000000+rand.Intn(1000000))
		body := messageBodies[rand.Intn(len(messageBodies))]

		msg, err := p.store.CreateMessage(accountSids[accountIdx], phoneNumberList[fromIdx], toPhone, body)
		if err != nil {
			return core.SeedData{}, err
		}

		// Set some messages to delivered status
		if i%2 == 0 {
			p.store.UpdateMessageStatus(msg.Sid, "delivered")
		} else if i%3 == 0 {
			p.store.UpdateMessageStatus(msg.Sid, "sent")
		}
	}

	// Create calls
	for i := 0; i < calls; i++ {
		accountIdx := rand.Intn(accounts)
		fromIdx := rand.Intn(phoneNumbers)
		toPhone := fmt.Sprintf("+1555%07d", 3000000+rand.Intn(1000000))

		call, err := p.store.CreateCall(accountSids[accountIdx], phoneNumberList[fromIdx], toPhone)
		if err != nil {
			return core.SeedData{}, err
		}

		// Set some calls to completed with duration
		if i%2 == 0 {
			duration := 30 + rand.Intn(570) // 30-600 seconds
			p.store.UpdateCallStatus(call.Sid, "completed", &duration)
		} else if i%3 == 0 {
			p.store.UpdateCallStatus(call.Sid, "in-progress", nil)
		}
	}

	summary := fmt.Sprintf("Created %d accounts, %d phone numbers, %d messages, %d calls",
		accounts, phoneNumbers, messages, calls)

	return core.SeedData{
		Summary: summary,
		Records: map[string]int{
			"accounts":      accounts,
			"phone_numbers": phoneNumbers,
			"messages":      messages,
			"calls":         calls,
		},
	}, nil
}
```

### Step 2: Run seed test

```bash
cd /Users/harper/Public/src/2389/ish
go build -o ish ./cmd/ish
./ish seed -d test_twilio.db
```

Expected: Output shows "Created X accounts, Y phone numbers, Z messages, W calls"

### Step 3: Verify seeded data

```bash
sqlite3 test_twilio.db "SELECT COUNT(*) FROM twilio_accounts;"
sqlite3 test_twilio.db "SELECT COUNT(*) FROM twilio_messages;"
sqlite3 test_twilio.db "SELECT COUNT(*) FROM twilio_calls;"
```

Expected: Non-zero counts matching seed size

### Step 4: Clean up test database

```bash
rm test_twilio.db
```

### Step 5: Commit Task 6

```bash
git add plugins/twilio/seed.go plugins/twilio/plugin.go
git commit -m "feat(twilio): implement test data seeding

- Add Seed method with small/medium/large sizes
- Generate realistic US phone numbers (+1-555-xxx-xxxx)
- Create messages with common SMS patterns (verification, notifications)
- Create calls with varied durations (30-600s)
- Mix of statuses: queued/sent/delivered for messages, initiated/in-progress/completed for calls"
```

---

## Task 7: Register Plugin in Main

**Files:**
- Modify: `cmd/ish/main.go`

### Step 1: Add Twilio plugin import

Modify `cmd/ish/main.go`:

```go
import (
	// ... existing imports
	_ "github.com/2389/ish/plugins/discord" // Register Discord plugin
	_ "github.com/2389/ish/plugins/google"  // Register Google plugin
	_ "github.com/2389/ish/plugins/oauth"   // Register OAuth plugin
	_ "github.com/2389/ish/plugins/twilio"  // Register Twilio plugin
)
```

### Step 2: Run build

```bash
cd /Users/harper/Public/src/2389/ish
go build -o ish ./cmd/ish
```

Expected: Build succeeds

### Step 3: Test plugin registration

```bash
./ish seed -d test_full.db
./ish serve -d test_full.db -p 9000 &
SERVER_PID=$!
sleep 2

# Test account auto-creation and message send
curl -X POST "http://localhost:9000/2010-04-01/Accounts/AC999/Messages.json" \
  -u "AC999:anytoken" \
  -d "To=+15551111111" \
  -d "From=+15559999999" \
  -d "Body=Test message" \
  | jq .

kill $SERVER_PID
rm test_full.db
```

Expected: Returns JSON with `sid` starting with "SM", status "queued"

### Step 4: Commit Task 7

```bash
git add cmd/ish/main.go
git commit -m "feat(twilio): register plugin in main

- Add Twilio plugin import to enable auto-registration
- Plugin now loads on server start alongside Discord, Google, OAuth"
```

---

## Task 8: Implement Webhook System

**Files:**
- Create: `plugins/twilio/webhooks.go`
- Modify: `plugins/twilio/plugin.go` (start webhook worker)
- Modify: `plugins/twilio/handlers.go` (queue webhooks on message/call creation)

### Step 1: Create webhook worker

Create `plugins/twilio/webhooks.go`:

```go
// ABOUTME: Async webhook delivery system for Twilio plugin
// ABOUTME: Simulates realistic status callback timing

package twilio

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// StartWebhookWorker polls the webhook queue and delivers pending webhooks
func (p *TwilioPlugin) StartWebhookWorker(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processWebhookQueue()
		}
	}
}

func (p *TwilioPlugin) processWebhookQueue() {
	// Get pending webhooks that are ready to deliver
	webhooks, err := p.store.GetPendingWebhooks(time.Now())
	if err != nil {
		log.Printf("Error fetching pending webhooks: %v", err)
		return
	}

	for _, webhook := range webhooks {
		p.deliverWebhook(webhook)
	}
}

func (p *TwilioPlugin) deliverWebhook(webhook WebhookQueueItem) {
	// Parse payload as form values
	values, err := url.ParseQuery(webhook.Payload)
	if err != nil {
		log.Printf("Error parsing webhook payload: %v", err)
		p.store.MarkWebhookFailed(webhook.ID)
		return
	}

	// Send POST request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm(webhook.WebhookURL, values)
	if err != nil {
		log.Printf("Error delivering webhook to %s: %v", webhook.WebhookURL, err)
		p.store.MarkWebhookFailed(webhook.ID)
		return
	}
	defer resp.Body.Close()

	// Mark as delivered
	if err := p.store.MarkWebhookDelivered(webhook.ID); err != nil {
		log.Printf("Error marking webhook delivered: %v", err)
	}
}

// QueueMessageWebhook schedules a webhook for a message status change
func (p *TwilioPlugin) QueueMessageWebhook(messageSid, status string, delay time.Duration) error {
	msg, err := p.store.GetMessage(messageSid)
	if err != nil {
		return err
	}

	// Get phone number config to find webhook URLs
	phoneNumbers, err := p.store.ListPhoneNumbers(msg.AccountSid)
	if err != nil {
		return err
	}

	var webhookURL string
	for _, pn := range phoneNumbers {
		if pn.PhoneNumber == msg.FromNumber && pn.StatusCallback != "" {
			webhookURL = pn.StatusCallback
			break
		}
	}

	// No webhook configured, skip
	if webhookURL == "" {
		return nil
	}

	// Build form-encoded payload
	payload := url.Values{}
	payload.Set("MessageSid", msg.Sid)
	payload.Set("MessageStatus", status)
	payload.Set("From", msg.FromNumber)
	payload.Set("To", msg.ToNumber)
	payload.Set("Body", msg.Body)
	payload.Set("AccountSid", msg.AccountSid)

	return p.store.QueueWebhook(messageSid, webhookURL, payload.Encode(), time.Now().Add(delay))
}

// QueueCallWebhook schedules a webhook for a call status change
func (p *TwilioPlugin) QueueCallWebhook(callSid, status string, delay time.Duration) error {
	call, err := p.store.GetCall(callSid)
	if err != nil {
		return err
	}

	phoneNumbers, err := p.store.ListPhoneNumbers(call.AccountSid)
	if err != nil {
		return err
	}

	var webhookURL string
	for _, pn := range phoneNumbers {
		if pn.PhoneNumber == call.FromNumber && pn.StatusCallback != "" {
			webhookURL = pn.StatusCallback
			break
		}
	}

	if webhookURL == "" {
		return nil
	}

	payload := url.Values{}
	payload.Set("CallSid", call.Sid)
	payload.Set("CallStatus", status)
	payload.Set("From", call.FromNumber)
	payload.Set("To", call.ToNumber)
	payload.Set("AccountSid", call.AccountSid)

	if call.Duration != nil {
		payload.Set("CallDuration", fmt.Sprintf("%d", *call.Duration))
	}

	return p.store.QueueWebhook(callSid, webhookURL, payload.Encode(), time.Now().Add(delay))
}

// SimulateMessageLifecycle progresses a message through realistic status transitions
func (p *TwilioPlugin) SimulateMessageLifecycle(messageSid string) {
	// queued → sent (100ms)
	time.AfterFunc(100*time.Millisecond, func() {
		p.store.UpdateMessageStatus(messageSid, "sent")
		p.QueueMessageWebhook(messageSid, "sent", 0)
	})

	// sent → delivered (500ms)
	time.AfterFunc(600*time.Millisecond, func() {
		p.store.UpdateMessageStatus(messageSid, "delivered")
		p.QueueMessageWebhook(messageSid, "delivered", 0)
	})
}

// SimulateCallLifecycle progresses a call through realistic status transitions
func (p *TwilioPlugin) SimulateCallLifecycle(callSid string) {
	// initiated → ringing (200ms)
	time.AfterFunc(200*time.Millisecond, func() {
		p.store.UpdateCallStatus(callSid, "ringing", nil)
		p.QueueCallWebhook(callSid, "ringing", 0)
	})

	// ringing → in-progress (800ms)
	time.AfterFunc(1000*time.Millisecond, func() {
		p.store.UpdateCallStatus(callSid, "in-progress", nil)
		p.QueueCallWebhook(callSid, "in-progress", 0)
	})

	// in-progress → completed (5-30s)
	completionDelay := time.Duration(5000+rand.Intn(25000)) * time.Millisecond
	time.AfterFunc(1000*time.Millisecond+completionDelay, func() {
		duration := 5 + rand.Intn(26) // 5-30 seconds
		p.store.UpdateCallStatus(callSid, "completed", &duration)
		p.QueueCallWebhook(callSid, "completed", 0)
	})
}
```

### Step 2: Add webhook queue store methods

Add to `plugins/twilio/store.go`:

```go
type WebhookQueueItem struct {
	ID          int
	ResourceSid string
	WebhookURL  string
	Payload     string
	ScheduledAt time.Time
	DeliveredAt *time.Time
	Status      string
	Attempts    int
	CreatedAt   time.Time
}

func (s *TwilioStore) QueueWebhook(resourceSid, webhookURL, payload string, scheduledAt time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO twilio_webhook_queue (resource_sid, webhook_url, payload, scheduled_at)
		VALUES (?, ?, ?, ?)
	`, resourceSid, webhookURL, payload, scheduledAt)
	return err
}

func (s *TwilioStore) GetPendingWebhooks(now time.Time) ([]WebhookQueueItem, error) {
	rows, err := s.db.Query(`
		SELECT id, resource_sid, webhook_url, payload, scheduled_at, delivered_at, status, attempts, created_at
		FROM twilio_webhook_queue
		WHERE status = 'pending' AND scheduled_at <= ? AND attempts < 3
		ORDER BY scheduled_at ASC
		LIMIT 100
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []WebhookQueueItem
	for rows.Next() {
		var w WebhookQueueItem
		var deliveredAt sql.NullTime

		err := rows.Scan(&w.ID, &w.ResourceSid, &w.WebhookURL, &w.Payload,
			&w.ScheduledAt, &deliveredAt, &w.Status, &w.Attempts, &w.CreatedAt)
		if err != nil {
			return nil, err
		}

		if deliveredAt.Valid {
			w.DeliveredAt = &deliveredAt.Time
		}

		webhooks = append(webhooks, w)
	}

	return webhooks, nil
}

func (s *TwilioStore) MarkWebhookDelivered(id int) error {
	_, err := s.db.Exec(`
		UPDATE twilio_webhook_queue
		SET status = 'delivered', delivered_at = ?
		WHERE id = ?
	`, time.Now(), id)
	return err
}

func (s *TwilioStore) MarkWebhookFailed(id int) error {
	_, err := s.db.Exec(`
		UPDATE twilio_webhook_queue
		SET status = 'failed', attempts = attempts + 1
		WHERE id = ?
	`, id)
	return err
}
```

### Step 3: Start webhook worker in plugin

Modify `plugins/twilio/plugin.go`:

```go
func (p *TwilioPlugin) SetDB(db *sql.DB) error {
	store, err := NewTwilioStore(db)
	if err != nil {
		return err
	}
	p.store = store

	// Start webhook worker
	go p.StartWebhookWorker(context.Background())

	return nil
}
```

### Step 4: Queue webhooks when creating messages/calls

Modify `plugins/twilio/handlers.go` sendMessage:

```go
func (p *TwilioPlugin) sendMessage(w http.ResponseWriter, r *http.Request) {
	// ... existing code ...

	message, err := p.store.CreateMessage(accountSid, from, to, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	// Queue immediate webhook for "queued" status
	p.QueueMessageWebhook(message.Sid, "queued", 0)

	// Start async lifecycle simulation
	go p.SimulateMessageLifecycle(message.Sid)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(messageToResponse(message))
}
```

Modify `plugins/twilio/handlers.go` initiateCall:

```go
func (p *TwilioPlugin) initiateCall(w http.ResponseWriter, r *http.Request) {
	// ... existing code ...

	call, err := p.store.CreateCall(accountSid, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	// Queue immediate webhook for "initiated" status
	p.QueueCallWebhook(call.Sid, "initiated", 0)

	// Start async lifecycle simulation
	go p.SimulateCallLifecycle(call.Sid)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(callToResponse(call))
}
```

### Step 5: Run build

```bash
cd /Users/harper/Public/src/2389/ish
go build -o ish ./cmd/ish
```

Expected: Build succeeds

### Step 6: Commit Task 8

```bash
git add plugins/twilio/webhooks.go plugins/twilio/store.go plugins/twilio/plugin.go plugins/twilio/handlers.go
git commit -m "feat(twilio): implement async webhook delivery system

- Add webhook worker that polls queue every 100ms
- Queue webhooks for message/call status changes
- Simulate realistic lifecycle timing:
  - SMS: queued → sent (100ms) → delivered (500ms)
  - Calls: initiated → ringing (200ms) → in-progress (800ms) → completed (5-30s)
- Form-encode webhook payloads matching Twilio format
- Retry failed deliveries up to 3 times
- Start worker on plugin initialization"
```

---

## Task 9: Integration Tests

**Files:**
- Create: `plugins/twilio/integration_test.go`

### Step 1: Write integration test for full SMS flow

Create `plugins/twilio/integration_test.go`:

```go
// ABOUTME: Integration tests for Twilio plugin
// ABOUTME: Tests end-to-end flows including webhook delivery

package twilio

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestFullSMSFlow(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Create account with phone number and webhook config
	account, _ := plugin.store.GetOrCreateAccount("ACTEST")
	phoneNumber, _ := plugin.store.CreatePhoneNumber("ACTEST", "+15551234567", "Test Phone")

	// Update phone number with status callback
	_, err := db.Exec(`
		UPDATE twilio_phone_numbers
		SET status_callback = 'http://example.com/webhook'
		WHERE sid = ?
	`, phoneNumber.Sid)
	if err != nil {
		t.Fatalf("Failed to set webhook: %v", err)
	}

	// Send message
	form := url.Values{}
	form.Set("To", "+15559876543")
	form.Set("From", "+15551234567")
	form.Set("Body", "Integration test")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/ACTEST/Messages.json",
		bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("ACTEST", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.sendMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)
	messageSid := response["sid"].(string)

	// Wait for lifecycle to complete
	time.Sleep(800 * time.Millisecond)

	// Verify message status progressed to delivered
	msg, _ := plugin.store.GetMessage(messageSid)
	if msg.Status != "delivered" {
		t.Fatalf("Expected status delivered, got %s", msg.Status)
	}

	// Verify webhooks were queued (3 total: queued, sent, delivered)
	var webhookCount int
	db.QueryRow("SELECT COUNT(*) FROM twilio_webhook_queue WHERE resource_sid = ?", messageSid).Scan(&webhookCount)
	if webhookCount != 3 {
		t.Fatalf("Expected 3 webhooks queued, got %d", webhookCount)
	}
}

func TestFullCallFlow(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("ACTEST2")
	phoneNumber, _ := plugin.store.CreatePhoneNumber("ACTEST2", "+15551111111", "Test Phone")

	db.Exec(`
		UPDATE twilio_phone_numbers
		SET status_callback = 'http://example.com/call-webhook'
		WHERE sid = ?
	`, phoneNumber.Sid)

	// Initiate call
	form := url.Values{}
	form.Set("To", "+15559999999")
	form.Set("From", "+15551111111")
	form.Set("Url", "http://example.com/twiml")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/ACTEST2/Calls.json",
		bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("ACTEST2", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.initiateCall)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)
	callSid := response["sid"].(string)

	// Wait for ringing status
	time.Sleep(300 * time.Millisecond)

	call, _ := plugin.store.GetCall(callSid)
	if call.Status != "ringing" {
		t.Fatalf("Expected status ringing, got %s", call.Status)
	}

	// Wait for in-progress
	time.Sleep(900 * time.Millisecond)

	call, _ = plugin.store.GetCall(callSid)
	if call.Status != "in-progress" {
		t.Fatalf("Expected status in-progress, got %s", call.Status)
	}
}

func TestAuthValidation(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Create account
	account, _ := plugin.store.GetOrCreateAccount("ACAUTH")

	// Valid auth
	req := httptest.NewRequest("GET", "/2010-04-01/Accounts/ACAUTH/Messages.json", nil)
	req.Header.Set("Authorization", basicAuth("ACAUTH", account.AuthToken))

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.listMessages)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 with valid auth, got %d", rr.Code)
	}

	// Invalid token
	req = httptest.NewRequest("GET", "/2010-04-01/Accounts/ACAUTH/Messages.json", nil)
	req.Header.Set("Authorization", basicAuth("ACAUTH", "wrongtoken"))

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with invalid token, got %d", rr.Code)
	}

	// Missing auth
	req = httptest.NewRequest("GET", "/2010-04-01/Accounts/ACAUTH/Messages.json", nil)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with missing auth, got %d", rr.Code)
	}
}
```

### Step 2: Add helper for basic auth to handlers_test

Add to `plugins/twilio/handlers_test.go`:

```go
import (
	"strings"
)
```

### Step 3: Run integration tests

```bash
cd plugins/twilio
go test -v -run "TestFull|TestAuth"
```

Expected: All integration tests PASS

### Step 4: Commit Task 9

```bash
git add plugins/twilio/integration_test.go plugins/twilio/handlers_test.go
git commit -m "test(twilio): add integration tests

- Test full SMS flow: send → webhook queue → status progression
- Test full call flow: initiate → ringing → in-progress
- Test HTTP Basic Auth validation (valid/invalid/missing)
- Verify webhook queueing for all status transitions
- End-to-end verification of async lifecycle simulation"
```

---

## Task 10: Documentation & Final Verification

**Files:**
- Create: `plugins/twilio/README.md`
- Modify: `README.md` (add Twilio to plugin list)

### Step 1: Create plugin README

Create `plugins/twilio/README.md`:

```markdown
# Twilio Plugin

Mock Twilio SMS and Voice APIs for local development testing.

## Features

- **SMS API**: Send messages, list messages, get message details
- **Voice API**: Initiate calls, list calls, get call details
- **Phone Numbers**: List configured phone numbers
- **Auto-accept Auth**: HTTP Basic Auth with account auto-creation
- **Async Webhooks**: Realistic status callback timing
- **Admin UI**: Schema-driven resource management

## Authentication

Use HTTP Basic Auth with Account SID as username and Auth Token as password:

```bash
curl -u AC123:token123 http://localhost:9000/2010-04-01/Accounts/AC123/Messages.json
```

Accounts are auto-created on first request. The auth token is randomly generated and returned.

## SMS Example

```bash
# Send SMS
curl -X POST "http://localhost:9000/2010-04-01/Accounts/AC123/Messages.json" \
  -u "AC123:token123" \
  -d "To=+15551234567" \
  -d "From=+15559876543" \
  -d "Body=Hello World"

# List messages
curl "http://localhost:9000/2010-04-01/Accounts/AC123/Messages.json" \
  -u "AC123:token123"

# Get message details
curl "http://localhost:9000/2010-04-01/Accounts/AC123/Messages/SM456.json" \
  -u "AC123:token123"
```

## Voice Example

```bash
# Initiate call
curl -X POST "http://localhost:9000/2010-04-01/Accounts/AC123/Calls.json" \
  -u "AC123:token123" \
  -d "To=+15551234567" \
  -d "From=+15559876543" \
  -d "Url=http://example.com/twiml"

# List calls
curl "http://localhost:9000/2010-04-01/Accounts/AC123/Calls.json" \
  -u "AC123:token123"
```

## Webhook Callbacks

Configure `status_callback` on phone numbers to receive async status updates:

```sql
UPDATE twilio_phone_numbers
SET status_callback = 'http://localhost:3000/webhook'
WHERE phone_number = '+15559876543';
```

### SMS Status Progression

- `queued` (immediate)
- `sent` (+100ms)
- `delivered` (+500ms)

### Call Status Progression

- `initiated` (immediate)
- `ringing` (+200ms)
- `in-progress` (+800ms)
- `completed` (+5-30s, random duration)

## Admin UI

Visit `http://localhost:9000/admin/twilio` to manage:

- Accounts (view SIDs, auth tokens)
- Messages (send, view status)
- Calls (initiate, track duration)
- Phone Numbers (configure webhooks)

## Testing

```bash
cd plugins/twilio
go test -v
```

Includes unit tests, handler tests, and integration tests for full SMS/Voice flows.
```

### Step 2: Update main README

Modify `/Users/harper/Public/src/2389/ish/README.md` to add Twilio to plugin list:

Find the "Built-in Plugins" section and add:

```markdown
### Built-in Plugins

- **Google Plugin**: Gmail, Calendar, People, and Tasks APIs
- **OAuth Plugin**: Mock OAuth 2.0 provider for testing authentication flows
- **Discord Plugin**: Discord webhook API v10 for testing webhook integrations
- **Twilio Plugin**: Twilio SMS and Voice APIs with async webhook callbacks
```

### Step 3: Run full test suite

```bash
cd /Users/harper/Public/src/2389/ish
go test ./plugins/twilio/...
```

Expected: All tests PASS

### Step 4: Run end-to-end verification

```bash
# Build
go build -o ish ./cmd/ish

# Seed database
./ish seed -d verify.db

# Start server
./ish serve -d verify.db -p 9000 &
SERVER_PID=$!
sleep 2

# Test SMS API
echo "Testing SMS..."
curl -s -X POST "http://localhost:9000/2010-04-01/Accounts/AC001/Messages.json" \
  -u "AC001:anytoken" \
  -d "To=+15551111111" \
  -d "From=+15559999999" \
  -d "Body=Test" \
  | jq '.sid, .status'

# Test Voice API
echo "Testing Voice..."
curl -s -X POST "http://localhost:9000/2010-04-01/Accounts/AC001/Calls.json" \
  -u "AC001:anytoken" \
  -d "To=+15551111111" \
  -d "From=+15559999999" \
  -d "Url=http://example.com/twiml" \
  | jq '.sid, .status'

# Test list endpoints
echo "Testing list endpoints..."
curl -s "http://localhost:9000/2010-04-01/Accounts/AC001/Messages.json" \
  -u "AC001:anytoken" \
  | jq '.messages | length'

curl -s "http://localhost:9000/2010-04-01/Accounts/AC001/IncomingPhoneNumbers.json" \
  -u "AC001:anytoken" \
  | jq '.incoming_phone_numbers | length'

# Cleanup
kill $SERVER_PID
rm verify.db
```

Expected: All endpoints return valid JSON responses

### Step 5: Commit Task 10

```bash
git add plugins/twilio/README.md README.md
git commit -m "docs(twilio): add comprehensive documentation

- Create plugin README with API examples
- Document authentication pattern
- Explain webhook callback timing
- Add SMS and Voice code examples
- Update main README to list Twilio plugin
- Include testing instructions"
```

---

## Final Verification Checklist

Run through this checklist to ensure everything works:

- [ ] All tests pass: `go test ./plugins/twilio/...`
- [ ] Server builds: `go build ./cmd/ish`
- [ ] Seeding works: `./ish seed -d test.db`
- [ ] Server starts: `./ish serve -d test.db`
- [ ] SMS API works (POST, GET list, GET detail)
- [ ] Voice API works (POST, GET list, GET detail)
- [ ] Phone Numbers API works (GET list)
- [ ] HTTP Basic Auth validates credentials
- [ ] Account auto-creation on first request
- [ ] Admin UI shows Twilio resources
- [ ] Webhooks queue and deliver
- [ ] Message status progresses (queued → sent → delivered)
- [ ] Call status progresses (initiated → ringing → in-progress → completed)

## Success Criteria

✅ **Complete** when:
1. All 10 tasks committed
2. All tests passing
3. Full verification checklist checked
4. Server runs and handles Twilio API requests
5. Webhooks deliver with realistic timing
6. Admin UI displays all Twilio resources
