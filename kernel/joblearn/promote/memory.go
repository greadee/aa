package promote

import (
	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/repo"
)

// MemorySink adapts memory's typed memory-record repository to the Sink seam.
// It is the production wiring; the promotion engine itself stays unaware of the
// canonical store.
type MemorySink struct {
	records *repo.MemoryRecordRepository
}

// NewMemorySink returns a Sink backed by a memory-record repository.
func NewMemorySink(records *repo.MemoryRecordRepository) (*MemorySink, error) {
	if records == nil {
		return nil, invalid("memory record repository is required")
	}
	return &MemorySink{records: records}, nil
}

// Propose stores a candidate record.
func (s *MemorySink) Propose(record v1.MemoryRecord) (v1.MemoryRecord, error) {
	return s.records.Propose(record)
}

// Transition applies a validated lifecycle transition.
func (s *MemorySink) Transition(id string, to v1.MemoryLifecycle) (v1.MemoryRecord, error) {
	return s.records.Transition(id, to)
}
