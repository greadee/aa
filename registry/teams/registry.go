package teams

import (
	"fmt"
	"sort"
)

// Registry holds durable team specifications. It stores and serves specs; it
// performs no allocation, scheduling, or execution.
type Registry struct {
	specs map[string]TeamSpec
}

// NewRegistry returns an empty team registry.
func NewRegistry() *Registry {
	return &Registry{specs: map[string]TeamSpec{}}
}

// Add validates and stores a spec. A duplicate id is a conflict.
func (r *Registry) Add(spec TeamSpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("teams: invalid team spec: %w", err)
	}
	if _, ok := r.specs[spec.ID]; ok {
		return fmt.Errorf("teams: duplicate team %q", spec.ID)
	}
	r.specs[spec.ID] = spec
	return nil
}

// Get returns a spec by id.
func (r *Registry) Get(id string) (TeamSpec, bool) {
	spec, ok := r.specs[id]
	return spec, ok
}

// List returns all specs sorted by id.
func (r *Registry) List() []TeamSpec {
	out := make([]TeamSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Count returns the number of specs.
func (r *Registry) Count() int { return len(r.specs) }
