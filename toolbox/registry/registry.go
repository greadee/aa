// Package registry holds validated tool, plugin, and MCP manifests and the
// provider that executes each tool kind.
//
// Kinds sit behind the Provider seam so adding a tool or an MCP server is a
// registration, not a change to toolbox core. Manifests are validated against
// the tool_manifest contract before they enter the registry.
package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"

	toolbox "github.com/greadee/aa/toolbox"
)

// Provider executes one tool kind. A provider receives a manifest that has
// already been validated and registered.
type Provider interface {
	// Kind returns the tool kind this provider handles.
	Kind() string
	// Invoke runs the tool and returns its output.
	Invoke(ctx context.Context, manifest toolbox.Manifest, input map[string]any) (map[string]any, error)
}

// Registry holds validated manifests and the provider for each kind.
type Registry struct {
	mu        sync.RWMutex
	manifests map[toolbox.ToolID]toolbox.Manifest
	providers map[string]Provider
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{
		manifests: map[toolbox.ToolID]toolbox.Manifest{},
		providers: map[string]Provider{},
	}
}

// RegisterProvider associates a provider with a tool kind. Registering the
// same kind twice is a conflict; a new tool of that kind needs no core change.
func (r *Registry) RegisterProvider(p Provider) error {
	if p == nil {
		return fmt.Errorf("%w: nil provider", toolbox.ErrInvalid)
	}
	kind := p.Kind()
	if kind == "" {
		return fmt.Errorf("%w: provider kind is required", toolbox.ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[kind]; exists {
		return fmt.Errorf("%w: provider for kind %q already registered", toolbox.ErrConflict, kind)
	}
	r.providers[kind] = p
	return nil
}

// Register validates and stores a manifest. A duplicate ID is a conflict.
func (r *Registry) Register(m toolbox.Manifest) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("%w: %s", toolbox.ErrInvalid, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.manifests[toolbox.ToolID(m.ID)]; exists {
		return fmt.Errorf("%w: tool %q already registered", toolbox.ErrConflict, m.ID)
	}
	r.manifests[toolbox.ToolID(m.ID)] = m
	return nil
}

// Replace validates and stores a manifest, overwriting any existing ID.
func (r *Registry) Replace(m toolbox.Manifest) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("%w: %s", toolbox.ErrInvalid, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.manifests[toolbox.ToolID(m.ID)] = m
	return nil
}

// Get returns the manifest for an ID.
func (r *Registry) Get(id toolbox.ToolID) (toolbox.Manifest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.manifests[id]
	return m, ok
}

// Remove deletes a manifest and reports whether it existed.
func (r *Registry) Remove(id toolbox.ToolID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.manifests[id]; !ok {
		return false
	}
	delete(r.manifests, id)
	return true
}

// List returns manifests filtered by kind (empty means all), sorted by ID for
// deterministic output.
func (r *Registry) List(kind string) []toolbox.Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]toolbox.Manifest, 0, len(r.manifests))
	for _, m := range r.manifests {
		if kind != "" && m.ToolKind != kind {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ProviderFor returns the provider registered for a tool kind.
func (r *Registry) ProviderFor(kind string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[kind]
	return p, ok
}

// ProviderForManifest returns the provider for a manifest's kind.
func (r *Registry) ProviderForManifest(m toolbox.Manifest) (Provider, bool) {
	return r.ProviderFor(m.ToolKind)
}

// Count returns the number of registered manifests.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.manifests)
}
