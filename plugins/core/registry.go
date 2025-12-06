// ABOUTME: Plugin registry for registering and retrieving plugins.
// ABOUTME: Plugins register themselves in init() functions.

package core

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Plugin)
	mu       sync.RWMutex
)

// Register adds a plugin to the registry
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()

	name := p.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("plugin %q already registered", name))
	}
	registry[name] = p
}

// Get retrieves a plugin by name
func Get(name string) (Plugin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// All returns all registered plugins
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()

	plugins := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		plugins = append(plugins, p)
	}
	return plugins
}

// Names returns all registered plugin names
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
