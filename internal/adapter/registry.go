package adapter

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/liemle3893/autoflow/internal/core"
)

// Registry manages adapter instances with lazy (on-first-access) initialisation.
// All exported methods are safe for concurrent use.
type Registry struct {
	mu        sync.Mutex
	adapters  map[string]Adapter
	connected map[string]bool
}

// NewRegistry returns an empty, ready-to-use Registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters:  make(map[string]Adapter),
		connected: make(map[string]bool),
	}
}

// Register stores adapter a under name. It must be called before the first
// call to Get for that name. Calling Register twice with the same name
// overwrites the previous entry.
func (r *Registry) Register(name string, a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = a
	// Reset connected state in case a replacement adapter is registered.
	delete(r.connected, name)
}

// Get returns the adapter registered under name. On the first call it invokes
// Connect; subsequent calls return the same instance without reconnecting.
//
// Returns a ConfigError when name is not registered.
// Returns a ConnectionError when Connect fails.
func (r *Registry) Get(ctx context.Context, name string) (Adapter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.adapters[name]
	if !ok {
		return nil, core.ConfigError(
			fmt.Sprintf("adapter %q is not registered", name),
			fmt.Sprintf("add a %q adapter block to e2e.config.yaml and call Register before Get", name),
			nil,
		)
	}

	if r.connected[name] {
		return a, nil
	}

	if err := a.Connect(ctx); err != nil {
		return nil, core.ConnectionError(
			name,
			fmt.Sprintf("adapter %q failed to connect: %v", name, err),
			err,
		)
	}

	r.connected[name] = true
	return a, nil
}

// CloseAll closes every adapter that has been successfully connected.
// Errors from Close are silently swallowed; callers that require error
// visibility should iterate Names and call Get + Close manually.
func (r *Registry) CloseAll(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, a := range r.adapters {
		if r.connected[name] {
			// Ignore close errors — best-effort teardown.
			_ = a.Close(ctx)
			delete(r.connected, name)
		}
	}
}

// Has reports whether an adapter with the given name has been registered.
func (r *Registry) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.adapters[name]
	return ok
}

// Names returns a sorted slice of all registered adapter names.
func (r *Registry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make([]string, 0, len(r.adapters))
	for n := range r.adapters {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
