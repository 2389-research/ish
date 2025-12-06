// ABOUTME: Tests for plugin registry thread-safe operations and functionality.
// ABOUTME: Validates registration, retrieval, duplicate detection, and concurrent access.

package core

import (
	"context"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
)

// mockPlugin implements the Plugin interface for testing
type mockPlugin struct {
	name string
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Health() HealthStatus {
	return HealthStatus{Status: "healthy", Message: "OK"}
}

func (m *mockPlugin) RegisterRoutes(r chi.Router) {}

func (m *mockPlugin) RegisterAuth(r chi.Router) {}

func (m *mockPlugin) Schema() PluginSchema {
	return PluginSchema{}
}

func (m *mockPlugin) Seed(ctx context.Context, size string) (SeedData, error) {
	return SeedData{}, nil
}

func (m *mockPlugin) ValidateToken(token string) bool {
	return false
}

// resetRegistry clears the registry for testing
func resetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Plugin)
}

func TestRegister(t *testing.T) {
	resetRegistry()

	plugin := &mockPlugin{name: "test-plugin"}
	Register(plugin)

	if len(registry) != 1 {
		t.Errorf("expected 1 plugin in registry, got %d", len(registry))
	}

	if _, exists := registry["test-plugin"]; !exists {
		t.Error("plugin 'test-plugin' not found in registry")
	}
}

func TestRegisterDuplicatePanic(t *testing.T) {
	resetRegistry()

	plugin1 := &mockPlugin{name: "duplicate"}
	plugin2 := &mockPlugin{name: "duplicate"}

	Register(plugin1)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration, but didn't panic")
		}
	}()

	Register(plugin2)
}

func TestGet(t *testing.T) {
	resetRegistry()

	plugin := &mockPlugin{name: "test-plugin"}
	Register(plugin)

	retrieved, ok := Get("test-plugin")
	if !ok {
		t.Error("expected to find 'test-plugin', but it wasn't found")
	}

	if retrieved.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got %q", retrieved.Name())
	}
}

func TestGetNonExistent(t *testing.T) {
	resetRegistry()

	_, ok := Get("non-existent")
	if ok {
		t.Error("expected Get to return false for non-existent plugin")
	}
}

func TestAll(t *testing.T) {
	resetRegistry()

	plugin1 := &mockPlugin{name: "plugin1"}
	plugin2 := &mockPlugin{name: "plugin2"}
	plugin3 := &mockPlugin{name: "plugin3"}

	Register(plugin1)
	Register(plugin2)
	Register(plugin3)

	all := All()
	if len(all) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(all))
	}

	// Verify all plugins are present
	names := make(map[string]bool)
	for _, p := range all {
		names[p.Name()] = true
	}

	expected := []string{"plugin1", "plugin2", "plugin3"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected plugin %q in All(), but it wasn't found", name)
		}
	}
}

func TestNames(t *testing.T) {
	resetRegistry()

	plugin1 := &mockPlugin{name: "alpha"}
	plugin2 := &mockPlugin{name: "beta"}
	plugin3 := &mockPlugin{name: "gamma"}

	Register(plugin1)
	Register(plugin2)
	Register(plugin3)

	names := Names()
	if len(names) != 3 {
		t.Errorf("expected 3 plugin names, got %d", len(names))
	}

	// Verify all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	expected := []string{"alpha", "beta", "gamma"}
	for _, name := range expected {
		if !nameSet[name] {
			t.Errorf("expected plugin name %q in Names(), but it wasn't found", name)
		}
	}
}

func TestThreadSafeConcurrentRegistration(t *testing.T) {
	resetRegistry()

	var wg sync.WaitGroup
	pluginCount := 100

	for i := 0; i < pluginCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			plugin := &mockPlugin{name: string(rune('a' + index))}
			Register(plugin)
		}(i)
	}

	wg.Wait()

	if len(registry) != pluginCount {
		t.Errorf("expected %d plugins after concurrent registration, got %d", pluginCount, len(registry))
	}
}

func TestThreadSafeConcurrentReads(t *testing.T) {
	resetRegistry()

	// Register some plugins
	for i := 0; i < 10; i++ {
		plugin := &mockPlugin{name: string(rune('a' + i))}
		Register(plugin)
	}

	var wg sync.WaitGroup
	concurrentReads := 1000

	for i := 0; i < concurrentReads; i++ {
		wg.Add(3)

		// Concurrent Get operations
		go func() {
			defer wg.Done()
			Get("a")
		}()

		// Concurrent All operations
		go func() {
			defer wg.Done()
			All()
		}()

		// Concurrent Names operations
		go func() {
			defer wg.Done()
			Names()
		}()
	}

	wg.Wait()
}

func TestThreadSafeMixedOperations(t *testing.T) {
	resetRegistry()

	var wg sync.WaitGroup

	// Register initial plugins
	for i := 0; i < 5; i++ {
		plugin := &mockPlugin{name: string(rune('a' + i))}
		Register(plugin)
	}

	// Mix of concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			Get("a")
		}()

		go func() {
			defer wg.Done()
			All()
		}()

		go func() {
			defer wg.Done()
			Names()
		}()
	}

	wg.Wait()

	// Verify registry integrity
	all := All()
	names := Names()

	if len(all) != len(names) {
		t.Errorf("inconsistent state: All() returned %d plugins, Names() returned %d names", len(all), len(names))
	}
}

func TestEmptyRegistry(t *testing.T) {
	resetRegistry()

	all := All()
	if len(all) != 0 {
		t.Errorf("expected empty All() result, got %d plugins", len(all))
	}

	names := Names()
	if len(names) != 0 {
		t.Errorf("expected empty Names() result, got %d names", len(names))
	}

	_, ok := Get("anything")
	if ok {
		t.Error("expected Get to return false for empty registry")
	}
}
