package routines

import (
	"fmt"
	"sort"
)

// Registry holds durable routine specifications. It stores and serves specs;
// it performs no execution and owns no workflow runtime.
type Registry struct {
	specs map[string]RoutineSpec
}

// NewRegistry returns an empty routine registry.
func NewRegistry() *Registry {
	return &Registry{specs: map[string]RoutineSpec{}}
}

// Add validates and stores a spec. A duplicate id is a conflict.
func (r *Registry) Add(spec RoutineSpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("routines: invalid routine spec: %w", err)
	}
	if _, ok := r.specs[spec.ID]; ok {
		return fmt.Errorf("routines: duplicate routine %q", spec.ID)
	}
	r.specs[spec.ID] = spec
	return nil
}

// Get returns a spec by id.
func (r *Registry) Get(id string) (RoutineSpec, bool) {
	spec, ok := r.specs[id]
	return spec, ok
}

// List returns all specs sorted by id.
func (r *Registry) List() []RoutineSpec {
	out := make([]RoutineSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Count returns the number of specs.
func (r *Registry) Count() int { return len(r.specs) }
