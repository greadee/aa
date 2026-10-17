package policies

import (
	"fmt"
	"sort"
)

// Registry holds durable policy definitions. It catalogues what policies
// exist; it does not decide applicability or enforce policy.
type Registry struct {
	specs map[string]PolicySpec
}

// NewRegistry returns an empty policy registry.
func NewRegistry() *Registry {
	return &Registry{specs: map[string]PolicySpec{}}
}

// Add validates and stores a spec. A duplicate id is a conflict.
func (r *Registry) Add(spec PolicySpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("policies: invalid policy spec: %w", err)
	}
	if _, ok := r.specs[spec.ID]; ok {
		return fmt.Errorf("policies: duplicate policy %q", spec.ID)
	}
	r.specs[spec.ID] = spec
	return nil
}

// Get returns a spec by id.
func (r *Registry) Get(id string) (PolicySpec, bool) {
	spec, ok := r.specs[id]
	return spec, ok
}

// List returns all specs sorted by id.
func (r *Registry) List() []PolicySpec {
	out := make([]PolicySpec, 0, len(r.specs))
	for _, spec := range r.specs {
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Count returns the number of specs.
func (r *Registry) Count() int { return len(r.specs) }
