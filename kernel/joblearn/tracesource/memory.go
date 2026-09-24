package tracesource

import (
	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/repo"
)

// MemorySource adapts memory's canonical trace repository to the Source seam.
// It is the production wiring; the engine itself stays unaware of the store,
// mirroring promote/memory.go.
type MemorySource struct {
	traces *repo.TraceRepository
}

// NewMemorySource returns a Source backed by a trace repository.
func NewMemorySource(traces *repo.TraceRepository) (*MemorySource, error) {
	if traces == nil {
		return nil, invalid("trace repository is required")
	}
	return &MemorySource{traces: traces}, nil
}

// Traces returns every canonical trace in the repository's stable order.
func (s *MemorySource) Traces() ([]v1.Trace, error) {
	return s.traces.List()
}
