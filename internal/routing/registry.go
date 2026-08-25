package routing

import (
	"fmt"
	"sync/atomic"
)

// Registry publishes immutable routing snapshots for concurrent readers.
type Registry struct{ current atomic.Value }

// NewRegistry creates a registry with an empty snapshot.
func NewRegistry() *Registry {
	registry := &Registry{}
	empty, _ := BuildSnapshot(nil, nil)
	registry.current.Store(empty)
	return registry
}

// Get returns the current immutable snapshot.
func (registry *Registry) Get() *RoutingSnapshot { return registry.current.Load().(*RoutingSnapshot) }

// Reload builds and atomically publishes a replacement snapshot.
func (registry *Registry) Reload(services []Service, routes []Route) error {
	snapshot, err := BuildSnapshot(services, routes)
	if err != nil {
		return fmt.Errorf("build routing snapshot: %w", err)
	}
	registry.current.Store(snapshot)
	return nil
}
