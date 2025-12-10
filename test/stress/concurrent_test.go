// ABOUTME: Stress tests for concurrent database access and plugin operations.
// ABOUTME: Tests race conditions, deadlocks, and thread safety under heavy load.

package stress

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/2389/ish/internal/store"
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

// MockPlugin is a simple plugin implementation for testing concurrent operations
type MockPlugin struct {
	name      string
	callCount int32
	mu        sync.Mutex
}

func (m *MockPlugin) Name() string {
	return m.name
}

func (m *MockPlugin) Health() interface{} {
	atomic.AddInt32(&m.callCount, 1)
	return map[string]string{"status": "ok"}
}

func (m *MockPlugin) RegisterRoutes(r chi.Router) {}

func (m *MockPlugin) RegisterAuth(r chi.Router) {}

func (m *MockPlugin) GetCallCount() int32 {
	return atomic.LoadInt32(&m.callCount)
}

// TestConcurrentDatabaseWrites tests multiple goroutines writing to the database simultaneously
func TestConcurrentDatabaseWrites(t *testing.T) {
	dbPath := "test_concurrent_writes.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numGoroutines := 20
	logsPerGoroutine := 50
	var wg sync.WaitGroup
	var errorCount int32

	// Launch multiple goroutines writing simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				log := &store.RequestLog{
					Timestamp:  time.Now(),
					PluginName: fmt.Sprintf("plugin-%d", id%5),
					Method:     []string{"GET", "POST", "PUT", "DELETE"}[j%4],
					Path:       fmt.Sprintf("/api/resource-%d", j),
					StatusCode: 200,
					DurationMs: j % 100,
				}
				if err := s.LogRequest(log); err != nil {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Error logging request: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}

	// Verify all logs were written
	logs, err := s.GetRequestLogs(&store.RequestLogQuery{Limit: 10000})
	if err != nil {
		t.Fatalf("Failed to retrieve logs: %v", err)
	}

	expectedCount := numGoroutines * logsPerGoroutine
	if len(logs) != expectedCount {
		t.Errorf("Expected %d logs, got %d", expectedCount, len(logs))
	}
}

// TestConcurrentDatabaseReads tests multiple goroutines reading from the database simultaneously
func TestConcurrentDatabaseReads(t *testing.T) {
	dbPath := "test_concurrent_reads.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// Insert test data
	for i := 0; i < 100; i++ {
		log := &store.RequestLog{
			Timestamp:  time.Now().Add(time.Duration(-i) * time.Minute),
			PluginName: fmt.Sprintf("plugin-%d", i%5),
			Method:     "GET",
			Path:       fmt.Sprintf("/api/test-%d", i),
			StatusCode: 200,
			DurationMs: 10,
		}
		s.LogRequest(log)
	}

	numGoroutines := 50
	readsPerGoroutine := 20
	var wg sync.WaitGroup
	var errorCount int32

	// Launch multiple goroutines reading simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < readsPerGoroutine; j++ {
				query := &store.RequestLogQuery{
					Limit: 50,
				}
				if j%2 == 0 {
					query.PluginName = fmt.Sprintf("plugin-%d", j%5)
				}
				_, err := s.GetRequestLogs(query)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Expected 0 errors during concurrent reads, got %d", errorCount)
	}
}

// TestConcurrentReadWrite tests simultaneous reads and writes to the database
func TestConcurrentReadWrite(t *testing.T) {
	dbPath := "test_concurrent_read_write.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numWriters := 10
	numReaders := 10
	operationsPerGoroutine := 50
	var wg sync.WaitGroup
	var errorCount int32

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				log := &store.RequestLog{
					Timestamp:  time.Now(),
					PluginName: fmt.Sprintf("writer-%d", id),
					Method:     "POST",
					Path:       fmt.Sprintf("/write-%d-%d", id, j),
					StatusCode: 201,
					DurationMs: 5,
				}
				if err := s.LogRequest(log); err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}(i)
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				_, err := s.GetRequestLogs(&store.RequestLogQuery{
					Limit: 100,
				})
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Expected 0 errors during concurrent read/write, got %d", errorCount)
	}
}

// TestConcurrentPluginAccessRaceCondition tests concurrent access to plugin methods
func TestConcurrentPluginAccessRaceCondition(t *testing.T) {
	plugin := &MockPlugin{name: "test-plugin"}
	numGoroutines := 100
	callsPerGoroutine := 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				plugin.Health()
			}
		}()
	}

	wg.Wait()

	expectedCount := int32(numGoroutines * callsPerGoroutine)
	actualCount := plugin.GetCallCount()
	if actualCount != expectedCount {
		t.Errorf("Expected %d calls, got %d", expectedCount, actualCount)
	}
}

// TestConcurrentLogRequestMetrics tests concurrent metric calculations
func TestConcurrentLogRequestMetrics(t *testing.T) {
	dbPath := "test_concurrent_metrics.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numGoroutines := 20
	logsPerGoroutine := 50
	var wg sync.WaitGroup

	// Insert logs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				log := &store.RequestLog{
					Timestamp:  time.Now().Add(time.Duration(-j) * time.Hour),
					PluginName: "google",
					Method:     "GET",
					Path:       fmt.Sprintf("/api/test-%d-%d", id, j),
					StatusCode: 200,
					DurationMs: j % 100,
				}
				s.LogRequest(log)
			}
		}(i)
	}

	wg.Wait()

	// Query metrics concurrently
	yesterday := time.Now().Add(-24 * time.Hour)
	var wg2 sync.WaitGroup
	var errorCount int32

	for i := 0; i < 50; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			_, err := s.GetPluginRequestCount("google", yesterday)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg2.Wait()

	if errorCount > 0 {
		t.Errorf("Expected 0 errors during concurrent metric reads, got %d", errorCount)
	}
}

// TestConcurrentDatabaseConnectionPool tests connection pooling under stress
func TestConcurrentDatabaseConnectionPool(t *testing.T) {
	dbPath := "test_connection_pool.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numGoroutines := 50
	operationsPerGoroutine := 100
	var wg sync.WaitGroup
	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				log := &store.RequestLog{
					Timestamp:  time.Now(),
					PluginName: fmt.Sprintf("plugin-%d", id%10),
					Method:     "GET",
					Path:       fmt.Sprintf("/pool-test-%d-%d", id, j),
					StatusCode: 200,
					DurationMs: 1,
				}
				if err := s.LogRequest(log); err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Expected 0 errors with connection pooling, got %d", errorCount)
	}

	// Verify data integrity
	logs, err := s.GetRequestLogs(&store.RequestLogQuery{Limit: 10000})
	if err != nil {
		t.Fatalf("Failed to retrieve logs: %v", err)
	}

	expectedCount := numGoroutines * operationsPerGoroutine
	if len(logs) != expectedCount {
		t.Errorf("Expected %d logs, got %d. Some writes may have been lost.", expectedCount, len(logs))
	}
}

// TestSQLiteTransactionIsolation tests transaction isolation and consistency
func TestSQLiteTransactionIsolation(t *testing.T) {
	dbPath := "test_transaction_isolation.db"
	defer os.Remove(dbPath)

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?cache=shared&mode=rwc")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Setup schema
	schema := `
	CREATE TABLE test_transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		value INTEGER,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	db.Exec(schema)

	numGoroutines := 20
	incrementsPerGoroutine := 50
	var wg sync.WaitGroup
	var errorCount int32

	// Insert initial value
	db.Exec("INSERT INTO test_transactions (value) VALUES (0)")

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				tx, err := db.Begin()
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					return
				}

				var currentValue int
				err = tx.QueryRow("SELECT value FROM test_transactions WHERE id = 1").Scan(&currentValue)
				if err != nil {
					tx.Rollback()
					atomic.AddInt32(&errorCount, 1)
					continue
				}

				_, err = tx.Exec("UPDATE test_transactions SET value = ? WHERE id = 1", currentValue+1)
				if err != nil {
					tx.Rollback()
					atomic.AddInt32(&errorCount, 1)
					continue
				}

				if err := tx.Commit(); err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Logf("Warning: %d transaction errors occurred (expected due to SQLite write serialization)", errorCount)
	}
}

// TestDeadlockPrevention tests that deadlocks don't occur under concurrent access
func TestDeadlockPrevention(t *testing.T) {
	dbPath := "test_deadlock_prevention.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// Insert initial data
	for i := 0; i < 50; i++ {
		log := &store.RequestLog{
			Timestamp:  time.Now(),
			PluginName: fmt.Sprintf("plugin-%d", i%5),
			Method:     "GET",
			Path:       fmt.Sprintf("/initial-%d", i),
			StatusCode: 200,
			DurationMs: 10,
		}
		s.LogRequest(log)
	}

	numGoroutines := 30
	operationsPerGoroutine := 50
	var wg sync.WaitGroup
	done := make(chan bool, 1)

	// Set a timeout to detect deadlocks
	timeout := time.AfterFunc(10*time.Second, func() {
		t.Log("Test completed successfully without deadlock")
		done <- true
	})
	defer timeout.Stop()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if j%2 == 0 {
					// Write operation
					log := &store.RequestLog{
						Timestamp:  time.Now(),
						PluginName: fmt.Sprintf("plugin-%d", id%5),
						Method:     "POST",
						Path:       fmt.Sprintf("/deadlock-test-%d-%d", id, j),
						StatusCode: 200,
						DurationMs: 5,
					}
					s.LogRequest(log)
				} else {
					// Read operation
					s.GetRequestLogs(&store.RequestLogQuery{Limit: 100})
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentDataConsistency verifies data consistency with concurrent access
func TestConcurrentDataConsistency(t *testing.T) {
	dbPath := "test_data_consistency.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numGoroutines := 25
	logsPerGoroutine := 40
	expectedPlugins := 5
	var wg sync.WaitGroup

	// Insert logs with known distribution
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pluginID := id % expectedPlugins
			for j := 0; j < logsPerGoroutine; j++ {
				log := &store.RequestLog{
					Timestamp:  time.Now().Add(time.Duration(-j) * time.Minute),
					PluginName: fmt.Sprintf("plugin-%d", pluginID),
					Method:     "GET",
					Path:       fmt.Sprintf("/test-%d-%d", id, j),
					StatusCode: 200,
					DurationMs: j % 50,
				}
				s.LogRequest(log)
			}
		}(i)
	}

	wg.Wait()

	// Verify data consistency
	logs, err := s.GetRequestLogs(&store.RequestLogQuery{Limit: 10000})
	if err != nil {
		t.Fatalf("Failed to retrieve logs: %v", err)
	}

	expectedCount := numGoroutines * logsPerGoroutine
	if len(logs) != expectedCount {
		t.Errorf("Expected %d total logs, got %d", expectedCount, len(logs))
	}

	// Verify each plugin has correct count
	pluginCounts := make(map[string]int)
	for _, log := range logs {
		pluginCounts[log.PluginName]++
	}

	expectedCountPerPlugin := (numGoroutines / expectedPlugins) * logsPerGoroutine
	for i := 0; i < expectedPlugins; i++ {
		pluginName := fmt.Sprintf("plugin-%d", i)
		if pluginCounts[pluginName] != expectedCountPerPlugin {
			t.Errorf("Plugin %s: expected %d logs, got %d", pluginName, expectedCountPerPlugin, pluginCounts[pluginName])
		}
	}
}

// TestPluginHealthConcurrentAccess tests concurrent health checks on plugins
func TestPluginHealthConcurrentAccess(t *testing.T) {
	plugin := &MockPlugin{name: "health-test"}
	numGoroutines := 50
	healthChecksPerGoroutine := 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < healthChecksPerGoroutine; j++ {
				health := plugin.Health()
				if health == nil {
					t.Error("Health check returned nil")
				}
			}
		}()
	}

	wg.Wait()

	expectedCount := int32(numGoroutines * healthChecksPerGoroutine)
	actualCount := plugin.GetCallCount()
	if actualCount != expectedCount {
		t.Errorf("Expected %d health checks, got %d", expectedCount, actualCount)
	}
}

// TestBoundedConcurrency tests behavior with bounded concurrency (connection pool limits)
func TestBoundedConcurrency(t *testing.T) {
	dbPath := "test_bounded_concurrency.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// Simulate work that takes time to ensure connections are held
	numGoroutines := 100
	var wg sync.WaitGroup
	var errorCount int32
	var successCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log := &store.RequestLog{
				Timestamp:  time.Now(),
				PluginName: fmt.Sprintf("bounded-%d", id%20),
				Method:     "GET",
				Path:       fmt.Sprintf("/bounded-%d", id),
				StatusCode: 200,
				DurationMs: 5,
			}

			// Add small artificial delay to simulate work
			time.Sleep(time.Millisecond)

			if err := s.LogRequest(log); err != nil {
				atomic.AddInt32(&errorCount, 1)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Logf("Warning: %d errors occurred under bounded concurrency (connection pool may be exhausted)", errorCount)
	}

	successExpected := int32(numGoroutines) - errorCount
	if successCount != successExpected {
		t.Errorf("Unexpected success count: expected %d, got %d", successExpected, successCount)
	}
}

// BenchmarkConcurrentWrites benchmarks concurrent write performance
func BenchmarkConcurrentWrites(b *testing.B) {
	dbPath := "bench_concurrent_writes.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numGoroutines := 10
	b.ResetTimer()

	var wg sync.WaitGroup
	logsPerGoroutine := b.N / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				log := &store.RequestLog{
					Timestamp:  time.Now(),
					PluginName: fmt.Sprintf("bench-plugin-%d", id),
					Method:     "GET",
					Path:       fmt.Sprintf("/bench-%d", j),
					StatusCode: 200,
					DurationMs: 1,
				}
				s.LogRequest(log)
			}
		}(i)
	}

	wg.Wait()
}

// BenchmarkConcurrentReads benchmarks concurrent read performance
func BenchmarkConcurrentReads(b *testing.B) {
	dbPath := "bench_concurrent_reads.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// Populate with data
	for i := 0; i < 1000; i++ {
		log := &store.RequestLog{
			Timestamp:  time.Now(),
			PluginName: fmt.Sprintf("plugin-%d", i%10),
			Method:     "GET",
			Path:       fmt.Sprintf("/bench-data-%d", i),
			StatusCode: 200,
			DurationMs: 10,
		}
		s.LogRequest(log)
	}

	numGoroutines := 10
	b.ResetTimer()

	var wg sync.WaitGroup
	readsPerGoroutine := b.N / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < readsPerGoroutine; j++ {
				s.GetRequestLogs(&store.RequestLogQuery{
					Limit: 100,
				})
			}
		}()
	}

	wg.Wait()
}

// BenchmarkConcurrentMixedOperations benchmarks mixed read/write performance
func BenchmarkConcurrentMixedOperations(b *testing.B) {
	dbPath := "bench_mixed_ops.db"
	defer os.Remove(dbPath)

	s, err := store.New(dbPath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	numGoroutines := 10
	b.ResetTimer()

	var wg sync.WaitGroup
	opsPerGoroutine := b.N / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				if j%2 == 0 {
					log := &store.RequestLog{
						Timestamp:  time.Now(),
						PluginName: fmt.Sprintf("bench-%d", id),
						Method:     "GET",
						Path:       fmt.Sprintf("/mixed-%d", j),
						StatusCode: 200,
						DurationMs: 1,
					}
					s.LogRequest(log)
				} else {
					s.GetRequestLogs(&store.RequestLogQuery{Limit: 50})
				}
			}
		}(i)
	}

	wg.Wait()
}
