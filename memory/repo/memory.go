package repo

import (
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// MemoryRecordRepository stores and promotes canonical memory records.
//
// Knowledge enters through Propose, which always writes a CANDIDATE record;
// promotion out of CANDIDATE is explicit and validated by the shared lifecycle
// state machine. The repository never promotes on its own.
type MemoryRecordRepository struct {
	store *store.Store
	proj  projection.Projection
	newID idgen
}

// NewMemoryRecordRepository returns a memory-record repository over the store
// and projection.
func NewMemoryRecordRepository(s *store.Store, p projection.Projection) *MemoryRecordRepository {
	return &MemoryRecordRepository{store: s, proj: p, newID: func() string { return "mem_" + randomSuffix() }}
}

// Propose stores a new memory record in the CANDIDATE lifecycle. It rejects a
// record that is not (or does not default to) CANDIDATE, so knowledge cannot
// enter the hierarchy already promoted.
func (r *MemoryRecordRepository) Propose(record v2.MemoryRecord) (v2.MemoryRecord, error) {
	if record.ContractVersion == "" {
		record.ContractVersion = v2.Version
	}
	record.Kind = "memory_record"
	if record.ID == "" {
		record.ID = r.newID()
	}
	if record.Lifecycle == "" {
		record.Lifecycle = v2.MemoryCandidate
	}
	if record.Lifecycle != v2.MemoryCandidate {
		return v2.MemoryRecord{}, fmt.Errorf("memory record: propose requires CANDIDATE lifecycle, got %s", record.Lifecycle)
	}
	rev := 1
	record.Revision = &rev
	if err := record.Validate(); err != nil {
		return v2.MemoryRecord{}, err
	}
	if _, err := putTyped(r.store, r.proj, "memory_record", record.ID, rev, record); err != nil {
		return v2.MemoryRecord{}, err
	}
	return record, nil
}

// Get returns a memory record by id.
func (r *MemoryRecordRepository) Get(id string) (v2.MemoryRecord, error) {
	return getTyped[v2.MemoryRecord](r.store, "memory_record", id)
}

// List returns all memory records, sorted by id.
func (r *MemoryRecordRepository) List() ([]v2.MemoryRecord, error) {
	records, err := r.store.ListRecords("memory_record")
	if err != nil {
		return nil, err
	}
	out := make([]v2.MemoryRecord, 0, len(records))
	for _, rec := range records {
		var record v2.MemoryRecord
		if err := decode(rec, &record); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

// ListByLifecycle returns records in the given lifecycle, sorted by id.
func (r *MemoryRecordRepository) ListByLifecycle(lifecycle v2.MemoryLifecycle) ([]v2.MemoryRecord, error) {
	if err := lifecycle.Validate(); err != nil {
		return nil, err
	}
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	var out []v2.MemoryRecord
	for _, record := range all {
		if record.Lifecycle == lifecycle {
			out = append(out, record)
		}
	}
	return out, nil
}

// Transition applies a validated lifecycle transition.
func (r *MemoryRecordRepository) Transition(id string, to v2.MemoryLifecycle) (v2.MemoryRecord, error) {
	record, err := r.Get(id)
	if err != nil {
		return v2.MemoryRecord{}, err
	}
	if _, err := Transition(record.Lifecycle, to); err != nil {
		return v2.MemoryRecord{}, fmt.Errorf("memory record %s: %w", id, err)
	}
	if record.Lifecycle == to {
		return record, nil
	}
	record.Lifecycle = to
	rev := revisionOf(r.store, "memory_record", id) + 1
	record.Revision = &rev
	if err := record.Validate(); err != nil {
		return v2.MemoryRecord{}, err
	}
	if _, err := putTyped(r.store, r.proj, "memory_record", record.ID, rev, record); err != nil {
		return v2.MemoryRecord{}, err
	}
	return record, nil
}

// Supersede marks oldID superseded and points newID at it.
func (r *MemoryRecordRepository) Supersede(oldID, newID string) error {
	if oldID == newID {
		return fmt.Errorf("memory record: cannot supersede itself")
	}
	if _, err := r.Transition(oldID, v2.MemorySuperseded); err != nil {
		return err
	}
	next, err := r.Get(newID)
	if err != nil {
		return err
	}
	next.Supersedes = oldID
	rev := revisionOf(r.store, "memory_record", newID) + 1
	next.Revision = &rev
	if err := next.Validate(); err != nil {
		return err
	}
	_, err = putTyped(r.store, r.proj, "memory_record", next.ID, rev, next)
	return err
}
