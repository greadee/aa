package models

import (
	"fmt"
	"sort"
)

// Registry holds durable model specifications. It stores and serves specs; it
// performs no allocation or execution.
type Registry struct {
	specs map[string]ModelSpec
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{specs: map[string]ModelSpec{}}
}

// NewDefaultRegistry returns a registry populated from DefaultCatalog.
func NewDefaultRegistry() (*Registry, error) {
	r := NewRegistry()
	for _, spec := range DefaultCatalog() {
		if err := r.Add(spec); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Add validates and stores a spec. A duplicate id is a conflict.
func (r *Registry) Add(spec ModelSpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("models: invalid model spec: %w", err)
	}
	if _, ok := r.specs[spec.ID]; ok {
		return fmt.Errorf("models: duplicate model %q", spec.ID)
	}
	r.specs[spec.ID] = spec
	return nil
}

// Get returns a spec by id.
func (r *Registry) Get(id string) (ModelSpec, bool) {
	spec, ok := r.specs[id]
	return spec, ok
}

// List returns all specs sorted by id.
func (r *Registry) List() []ModelSpec {
	out := make([]ModelSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Count returns the number of specs.
func (r *Registry) Count() int { return len(r.specs) }
