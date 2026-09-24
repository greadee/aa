package memory

import (
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/query"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

// Memory wires the canonical store, the rebuildable projection, the typed
// repositories, and the query API into one entry point.
type Memory struct {
	store      *store.Store
	projection projection.Projection
	query      *query.Query
	issues     *repo.IssueRepository
	strategies *repo.StrategyRepository
	records    *repo.MemoryRecordRepository
	traces     *repo.TraceRepository
}

// Open prepares a memory rooted at root and rebuilds the projection.
func Open(root string) (*Memory, error) {
	s, err := store.Open(root)
	if err != nil {
		return nil, err
	}
	p := projection.NewMem()
	if err := projection.Rebuild(p, s); err != nil {
		return nil, err
	}
	m := &Memory{store: s, projection: p}
	m.query = query.New(p, s)
	m.issues = repo.NewIssueRepository(s, p)
	m.strategies = repo.NewStrategyRepository(s, p)
	m.records = repo.NewMemoryRecordRepository(s, p)
	m.traces = repo.NewTraceRepository(s, p)
	return m, nil
}

// Store returns the canonical store.
func (m *Memory) Store() *store.Store { return m.store }

// Projection returns the current projection.
func (m *Memory) Projection() projection.Projection { return m.projection }

// Query returns the read API.
func (m *Memory) Query() *query.Query { return m.query }

// Issues returns the issue repository.
func (m *Memory) Issues() *repo.IssueRepository { return m.issues }

// Strategies returns the strategy repository.
func (m *Memory) Strategies() *repo.StrategyRepository { return m.strategies }

// Records returns the memory-record repository.
func (m *Memory) Records() *repo.MemoryRecordRepository { return m.records }

// Traces returns the canonical trace repository.
func (m *Memory) Traces() *repo.TraceRepository { return m.traces }

// Rebuild discards and reconstructs the projection from canonical records.
func (m *Memory) Rebuild() error {
	return projection.Rebuild(m.projection, m.store)
}

// InitProject writes the project manifest once.
func (m *Memory) InitProject(manifest []byte) (bool, error) {
	return m.store.InitProject(manifest)
}

// PutRecord writes a canonical record.
func (m *Memory) PutRecord(rec store.Record) (bool, error) {
	return m.store.PutRecord(rec)
}

// AppendEvent appends a canonical event.
func (m *Memory) AppendEvent(event []byte) (bool, error) {
	return m.store.AppendEvent(event)
}
