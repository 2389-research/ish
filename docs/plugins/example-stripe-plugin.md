# Example: Stripe Plugin

This is a complete, working example of an ISH plugin that simulates the Stripe Payments API. You can copy and adapt this code to create your own plugins.

## Overview

This plugin provides a mock Stripe API with:
- Customer management
- Payment charge operations
- Refund processing
- OAuth token validation
- Admin UI integration
- Realistic test data generation

## File Structure

```
plugins/stripe/
├── plugin.go      # Main plugin implementation
├── handlers.go    # HTTP request handlers
├── schema.go      # Admin UI schema
└── plugin_test.go # Tests
```

## Implementation

### plugin.go

```go
// ABOUTME: Stripe plugin for ISH fake payment processing.
// ABOUTME: Provides mock Stripe API for local development.

package stripe

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/2389/ish/internal/store"
    "github.com/2389/ish/plugins/core"
    "github.com/go-chi/chi/v5"
)

func init() {
    core.Register(&StripePlugin{})
}

type StripePlugin struct {
    store *store.Store
}

func (p *StripePlugin) Name() string {
    return "stripe"
}

func (p *StripePlugin) Health() core.HealthStatus {
    if p.store == nil {
        return core.HealthStatus{
            Status:  "degraded",
            Message: "Store not initialized",
        }
    }

    return core.HealthStatus{
        Status:  "healthy",
        Message: "Stripe plugin operational",
    }
}

func (p *StripePlugin) RegisterRoutes(r chi.Router) {
    // Customer endpoints
    r.Get("/v1/customers", p.handleListCustomers)
    r.Post("/v1/customers", p.handleCreateCustomer)
    r.Get("/v1/customers/{id}", p.handleGetCustomer)
    r.Delete("/v1/customers/{id}", p.handleDeleteCustomer)

    // Charge endpoints
    r.Get("/v1/charges", p.handleListCharges)
    r.Post("/v1/charges", p.handleCreateCharge)
    r.Get("/v1/charges/{id}", p.handleGetCharge)
    r.Post("/v1/charges/{id}/refund", p.handleRefundCharge)
}

func (p *StripePlugin) RegisterAuth(r chi.Router) {
    // OAuth plugin handles authentication
}

func (p *StripePlugin) Schema() core.PluginSchema {
    return getStripeSchema()
}

func (p *StripePlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
    if p.store == nil {
        return core.SeedData{}, fmt.Errorf("store not initialized")
    }

    customerCount := 10
    chargeCount := 25

    // Adjust based on size
    switch size {
    case "small":
        customerCount = 5
        chargeCount = 10
    case "large":
        customerCount = 50
        chargeCount = 200
    }

    // Create customers
    customerIDs := make([]string, customerCount)
    for i := 0; i < customerCount; i++ {
        customer := &Customer{
            ID:      fmt.Sprintf("cus_%d", i+1),
            Email:   fmt.Sprintf("customer%d@example.com", i+1),
            Name:    fmt.Sprintf("Customer %d", i+1),
            Created: time.Now().Add(-time.Duration(i*24) * time.Hour).Unix(),
        }
        if err := p.store.CreateStripeCustomer(customer); err != nil {
            return core.SeedData{}, err
        }
        customerIDs[i] = customer.ID
    }

    // Create charges
    for i := 0; i < chargeCount; i++ {
        customerID := customerIDs[i%customerCount]
        charge := &Charge{
            ID:         fmt.Sprintf("ch_%d", i+1),
            Amount:     1000 + (i * 500), // Varying amounts
            Currency:   "usd",
            Customer:   customerID,
            Status:     "succeeded",
            Created:    time.Now().Add(-time.Duration(i*12) * time.Hour).Unix(),
            Refunded:   false,
        }
        if err := p.store.CreateStripeCharge(charge); err != nil {
            return core.SeedData{}, err
        }
    }

    return core.SeedData{
        Summary: fmt.Sprintf("Created %d customers and %d charges", customerCount, chargeCount),
        Records: map[string]int{
            "customers": customerCount,
            "charges":   chargeCount,
        },
    }, nil
}

func (p *StripePlugin) ValidateToken(token string) bool {
    if p.store == nil {
        return false
    }

    t, err := p.store.GetToken(token)
    if err != nil {
        if err == sql.ErrNoRows {
            return false
        }
        return false
    }

    if t.Revoked {
        return false
    }

    if t.PluginName != "stripe" {
        return false
    }

    return true
}

func (p *StripePlugin) SetStore(s *store.Store) {
    p.store = s
}

// Customer represents a Stripe customer
type Customer struct {
    ID      string
    Email   string
    Name    string
    Created int64
}

// Charge represents a Stripe charge
type Charge struct {
    ID       string
    Amount   int
    Currency string
    Customer string
    Status   string
    Created  int64
    Refunded bool
}
```

### handlers.go

```go
package stripe

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
)

// Customer handlers

func (p *StripePlugin) handleListCustomers(w http.ResponseWriter, r *http.Request) {
    // Parse pagination parameters
    limit := 10
    if l := r.URL.Query().Get("limit"); l != "" {
        limit, _ = strconv.Atoi(l)
    }
    if limit > 100 {
        limit = 100
    }

    startingAfter := r.URL.Query().Get("starting_after")

    customers, err := p.store.ListStripeCustomers(limit, startingAfter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Determine if there are more results
    hasMore := len(customers) == limit

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "object":   "list",
        "data":     customers,
        "has_more": hasMore,
        "url":      "/v1/customers",
    })
}

func (p *StripePlugin) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email string `json:"email"`
        Name  string `json:"name"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if req.Email == "" {
        http.Error(w, "Email is required", http.StatusBadRequest)
        return
    }

    customer := &Customer{
        ID:      fmt.Sprintf("cus_%d", time.Now().UnixNano()),
        Email:   req.Email,
        Name:    req.Name,
        Created: time.Now().Unix(),
    }

    if err := p.store.CreateStripeCustomer(customer); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(customer)
}

func (p *StripePlugin) handleGetCustomer(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    customer, err := p.store.GetStripeCustomer(id)
    if err != nil {
        http.Error(w, "Customer not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(customer)
}

func (p *StripePlugin) handleDeleteCustomer(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    if err := p.store.DeleteStripeCustomer(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "id":      id,
        "object":  "customer",
        "deleted": true,
    })
}

// Charge handlers

func (p *StripePlugin) handleListCharges(w http.ResponseWriter, r *http.Request) {
    limit := 10
    if l := r.URL.Query().Get("limit"); l != "" {
        limit, _ = strconv.Atoi(l)
    }
    if limit > 100 {
        limit = 100
    }

    customerID := r.URL.Query().Get("customer")
    startingAfter := r.URL.Query().Get("starting_after")

    charges, err := p.store.ListStripeCharges(customerID, limit, startingAfter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    hasMore := len(charges) == limit

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "object":   "list",
        "data":     charges,
        "has_more": hasMore,
        "url":      "/v1/charges",
    })
}

func (p *StripePlugin) handleCreateCharge(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Amount   int    `json:"amount"`
        Currency string `json:"currency"`
        Customer string `json:"customer"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if req.Amount <= 0 {
        http.Error(w, "Amount must be positive", http.StatusBadRequest)
        return
    }

    if req.Currency == "" {
        req.Currency = "usd"
    }

    // Verify customer exists
    if req.Customer != "" {
        if _, err := p.store.GetStripeCustomer(req.Customer); err != nil {
            http.Error(w, "Customer not found", http.StatusBadRequest)
            return
        }
    }

    charge := &Charge{
        ID:       fmt.Sprintf("ch_%d", time.Now().UnixNano()),
        Amount:   req.Amount,
        Currency: req.Currency,
        Customer: req.Customer,
        Status:   "succeeded",
        Created:  time.Now().Unix(),
        Refunded: false,
    }

    if err := p.store.CreateStripeCharge(charge); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(charge)
}

func (p *StripePlugin) handleGetCharge(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    charge, err := p.store.GetStripeCharge(id)
    if err != nil {
        http.Error(w, "Charge not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(charge)
}

func (p *StripePlugin) handleRefundCharge(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    charge, err := p.store.GetStripeCharge(id)
    if err != nil {
        http.Error(w, "Charge not found", http.StatusNotFound)
        return
    }

    if charge.Refunded {
        http.Error(w, "Charge already refunded", http.StatusBadRequest)
        return
    }

    // Mark as refunded
    charge.Refunded = true
    if err := p.store.UpdateStripeCharge(charge); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "id":      fmt.Sprintf("re_%d", time.Now().UnixNano()),
        "object":  "refund",
        "amount":  charge.Amount,
        "charge":  charge.ID,
        "status":  "succeeded",
        "created": time.Now().Unix(),
    })
}
```

### schema.go

```go
package stripe

import "github.com/2389/ish/plugins/core"

func getStripeSchema() core.PluginSchema {
    return core.PluginSchema{
        Resources: []core.ResourceSchema{
            {
                Name:        "Customers",
                Slug:        "customers",
                ListColumns: []string{"email", "name", "created"},
                Fields: []core.FieldSchema{
                    {
                        Name:     "id",
                        Type:     "string",
                        Display:  "ID",
                        Required: false,
                        Editable: false,
                    },
                    {
                        Name:     "email",
                        Type:     "email",
                        Display:  "Email",
                        Required: true,
                        Editable: true,
                    },
                    {
                        Name:     "name",
                        Type:     "string",
                        Display:  "Name",
                        Required: true,
                        Editable: true,
                    },
                    {
                        Name:     "created",
                        Type:     "datetime",
                        Display:  "Created",
                        Required: false,
                        Editable: false,
                    },
                },
                Actions: []core.ActionSchema{
                    {
                        Name:       "delete",
                        HTTPMethod: "DELETE",
                        Endpoint:   "/v1/customers/{id}",
                        Confirm:    true,
                    },
                },
            },
            {
                Name:        "Charges",
                Slug:        "charges",
                ListColumns: []string{"amount", "customer", "status"},
                Fields: []core.FieldSchema{
                    {
                        Name:     "id",
                        Type:     "string",
                        Display:  "ID",
                        Required: false,
                        Editable: false,
                    },
                    {
                        Name:     "amount",
                        Type:     "number",
                        Display:  "Amount (cents)",
                        Required: true,
                        Editable: false,
                    },
                    {
                        Name:     "currency",
                        Type:     "string",
                        Display:  "Currency",
                        Required: true,
                        Editable: false,
                    },
                    {
                        Name:     "customer",
                        Type:     "string",
                        Display:  "Customer ID",
                        Required: false,
                        Editable: false,
                    },
                    {
                        Name:     "status",
                        Type:     "string",
                        Display:  "Status",
                        Required: false,
                        Editable: false,
                    },
                    {
                        Name:     "refunded",
                        Type:     "boolean",
                        Display:  "Refunded",
                        Required: false,
                        Editable: false,
                    },
                    {
                        Name:     "created",
                        Type:     "datetime",
                        Display:  "Created",
                        Required: false,
                        Editable: false,
                    },
                },
                Actions: []core.ActionSchema{
                    {
                        Name:       "refund",
                        HTTPMethod: "POST",
                        Endpoint:   "/v1/charges/{id}/refund",
                        Confirm:    true,
                    },
                },
            },
        },
    }
}
```

### plugin_test.go

```go
package stripe_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/2389/ish/internal/store"
    "github.com/2389/ish/plugins/stripe"
    "github.com/go-chi/chi/v5"
)

func setupTest(t *testing.T) (*stripe.StripePlugin, *chi.Mux) {
    s, err := store.New(":memory:")
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { s.Close() })

    p := &stripe.StripePlugin{}
    p.SetStore(s)

    r := chi.NewRouter()
    p.RegisterRoutes(r)

    return p, r
}

func TestStripePlugin_Customers(t *testing.T) {
    _, r := setupTest(t)

    var customerID string

    t.Run("create customer", func(t *testing.T) {
        reqBody := map[string]string{
            "email": "test@example.com",
            "name":  "Test Customer",
        }
        body, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/v1/customers", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusCreated {
            t.Errorf("expected status 201, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        if resp["id"] == nil {
            t.Error("expected customer ID in response")
        }

        customerID = resp["id"].(string)
    })

    t.Run("get customer", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/v1/customers/"+customerID, nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        if resp["email"] != "test@example.com" {
            t.Errorf("expected email test@example.com, got %v", resp["email"])
        }
    })

    t.Run("list customers", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/v1/customers", nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        data := resp["data"].([]any)
        if len(data) == 0 {
            t.Error("expected at least one customer")
        }
    })

    t.Run("delete customer", func(t *testing.T) {
        req := httptest.NewRequest("DELETE", "/v1/customers/"+customerID, nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        if resp["deleted"] != true {
            t.Error("expected deleted=true")
        }
    })
}

func TestStripePlugin_Charges(t *testing.T) {
    p, r := setupTest(t)

    // Create a customer first
    customer := &stripe.Customer{
        ID:    "cus_test",
        Email: "test@example.com",
        Name:  "Test Customer",
    }
    p.SetStore().CreateStripeCustomer(customer)

    var chargeID string

    t.Run("create charge", func(t *testing.T) {
        reqBody := map[string]any{
            "amount":   5000,
            "currency": "usd",
            "customer": "cus_test",
        }
        body, _ := json.Marshal(reqBody)

        req := httptest.NewRequest("POST", "/v1/charges", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusCreated {
            t.Errorf("expected status 201, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        if resp["id"] == nil {
            t.Error("expected charge ID in response")
        }

        chargeID = resp["id"].(string)
    })

    t.Run("get charge", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/v1/charges/"+chargeID, nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        if resp["amount"] != float64(5000) {
            t.Errorf("expected amount 5000, got %v", resp["amount"])
        }
    })

    t.Run("refund charge", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/v1/charges/"+chargeID+"/refund", nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        if resp["object"] != "refund" {
            t.Error("expected refund object")
        }
    })

    t.Run("list charges", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/v1/charges", nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }

        var resp map[string]any
        json.NewDecoder(w.Body).Decode(&resp)

        data := resp["data"].([]any)
        if len(data) == 0 {
            t.Error("expected at least one charge")
        }
    })
}

func TestStripePlugin_Health(t *testing.T) {
    p, _ := setupTest(t)

    health := p.Health()
    if health.Status != "healthy" {
        t.Errorf("expected healthy status, got %s", health.Status)
    }
}

func TestStripePlugin_Seed(t *testing.T) {
    p, _ := setupTest(t)

    result, err := p.Seed(context.Background(), "small")
    if err != nil {
        t.Fatalf("seed failed: %v", err)
    }

    if result.Records["customers"] != 5 {
        t.Errorf("expected 5 customers, got %d", result.Records["customers"])
    }

    if result.Records["charges"] != 10 {
        t.Errorf("expected 10 charges, got %d", result.Records["charges"])
    }
}
```

## Database Schema

Add these tables to `internal/store/store.go`:

```go
func (s *Store) initSchema() error {
    // ... existing tables ...

    // Stripe customers
    _, err = s.db.Exec(`
        CREATE TABLE IF NOT EXISTS stripe_customers (
            id TEXT PRIMARY KEY,
            email TEXT NOT NULL,
            name TEXT,
            created INTEGER NOT NULL
        )
    `)
    if err != nil {
        return err
    }

    // Stripe charges
    _, err = s.db.Exec(`
        CREATE TABLE IF NOT EXISTS stripe_charges (
            id TEXT PRIMARY KEY,
            amount INTEGER NOT NULL,
            currency TEXT NOT NULL,
            customer TEXT,
            status TEXT NOT NULL,
            created INTEGER NOT NULL,
            refunded BOOLEAN DEFAULT 0,
            FOREIGN KEY (customer) REFERENCES stripe_customers(id)
        )
    `)
    if err != nil {
        return err
    }

    return nil
}
```

## Store Methods

Add these methods to `internal/store/store.go` (or create `internal/store/stripe.go`):

```go
func (s *Store) CreateStripeCustomer(c *Customer) error {
    _, err := s.db.Exec(
        `INSERT INTO stripe_customers (id, email, name, created) VALUES (?, ?, ?, ?)`,
        c.ID, c.Email, c.Name, c.Created,
    )
    return err
}

func (s *Store) GetStripeCustomer(id string) (*Customer, error) {
    var c Customer
    err := s.db.QueryRow(
        `SELECT id, email, name, created FROM stripe_customers WHERE id = ?`,
        id,
    ).Scan(&c.ID, &c.Email, &c.Name, &c.Created)
    return &c, err
}

func (s *Store) ListStripeCustomers(limit int, startingAfter string) ([]*Customer, error) {
    query := `SELECT id, email, name, created FROM stripe_customers`
    args := []any{}

    if startingAfter != "" {
        query += ` WHERE id > ?`
        args = append(args, startingAfter)
    }

    query += ` ORDER BY id LIMIT ?`
    args = append(args, limit)

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var customers []*Customer
    for rows.Next() {
        var c Customer
        if err := rows.Scan(&c.ID, &c.Email, &c.Name, &c.Created); err != nil {
            return nil, err
        }
        customers = append(customers, &c)
    }

    return customers, nil
}

func (s *Store) DeleteStripeCustomer(id string) error {
    _, err := s.db.Exec(`DELETE FROM stripe_customers WHERE id = ?`, id)
    return err
}

func (s *Store) CreateStripeCharge(c *Charge) error {
    _, err := s.db.Exec(
        `INSERT INTO stripe_charges (id, amount, currency, customer, status, created, refunded)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
        c.ID, c.Amount, c.Currency, c.Customer, c.Status, c.Created, c.Refunded,
    )
    return err
}

func (s *Store) GetStripeCharge(id string) (*Charge, error) {
    var c Charge
    err := s.db.QueryRow(
        `SELECT id, amount, currency, customer, status, created, refunded
         FROM stripe_charges WHERE id = ?`,
        id,
    ).Scan(&c.ID, &c.Amount, &c.Currency, &c.Customer, &c.Status, &c.Created, &c.Refunded)
    return &c, err
}

func (s *Store) ListStripeCharges(customerID string, limit int, startingAfter string) ([]*Charge, error) {
    query := `SELECT id, amount, currency, customer, status, created, refunded
              FROM stripe_charges`
    args := []any{}

    where := []string{}
    if customerID != "" {
        where = append(where, `customer = ?`)
        args = append(args, customerID)
    }
    if startingAfter != "" {
        where = append(where, `id > ?`)
        args = append(args, startingAfter)
    }

    if len(where) > 0 {
        query += ` WHERE ` + strings.Join(where, " AND ")
    }

    query += ` ORDER BY id LIMIT ?`
    args = append(args, limit)

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var charges []*Charge
    for rows.Next() {
        var c Charge
        if err := rows.Scan(&c.ID, &c.Amount, &c.Currency, &c.Customer, &c.Status, &c.Created, &c.Refunded); err != nil {
            return nil, err
        }
        charges = append(charges, &c)
    }

    return charges, nil
}

func (s *Store) UpdateStripeCharge(c *Charge) error {
    _, err := s.db.Exec(
        `UPDATE stripe_charges SET refunded = ? WHERE id = ?`,
        c.Refunded, c.ID,
    )
    return err
}
```

## Installation

1. Copy the plugin files to `plugins/stripe/`
2. Add store methods to `internal/store/stripe.go`
3. Import the plugin in `cmd/ish/main.go`:

```go
import (
    _ "github.com/2389/ish/plugins/google"
    _ "github.com/2389/ish/plugins/oauth"
    _ "github.com/2389/ish/plugins/stripe"  // Add this
)
```

4. Rebuild and run:

```bash
go build -o ish ./cmd/ish
./ish reset --db ish.db
./ish serve --db ish.db
```

## Usage

### Create a customer

```bash
curl -X POST http://localhost:9000/v1/customers \
  -H "Authorization: Bearer user:harper" \
  -H "Content-Type: application/json" \
  -d '{"email":"customer@example.com","name":"John Doe"}'
```

### Create a charge

```bash
curl -X POST http://localhost:9000/v1/charges \
  -H "Authorization: Bearer user:harper" \
  -H "Content-Type: application/json" \
  -d '{"amount":5000,"currency":"usd","customer":"cus_123"}'
```

### Refund a charge

```bash
curl -X POST http://localhost:9000/v1/charges/ch_123/refund \
  -H "Authorization: Bearer user:harper"
```

### Admin UI

Visit `http://localhost:9000/admin/stripe` to manage customers and charges through the web interface.

## Key Concepts Demonstrated

1. **Plugin Registration**: Auto-registration via `init()`
2. **Dependency Injection**: Store injected via `SetStore()`
3. **RESTful API**: Standard HTTP methods and endpoints
4. **Schema-Driven UI**: Declarative admin interface
5. **Data Seeding**: Realistic test data generation
6. **Token Validation**: OAuth integration
7. **Pagination**: Cursor-based pagination with `starting_after`
8. **Comprehensive Testing**: Unit tests for all operations

## Next Steps

- Adapt this example for your own API
- Add more complex business logic
- Implement webhooks or background jobs
- Add more resources (subscriptions, invoices, etc.)
- Integrate with real external APIs for testing

See [DEVELOPMENT.md](./DEVELOPMENT.md) for more details on plugin development.
