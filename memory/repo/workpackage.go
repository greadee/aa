package repo

import (
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// DefaultAcceptance is used when a work package is submitted without explicit
// acceptance criteria. Callers may supply their own criteria.
var DefaultAcceptance = []string{"Work package is implemented and verified."}

// WorkPackageRecord is the durable representation of a work package. It embeds
// the canonical v2 object and carries the scheduler capability extension, which
// the v2 work-package schema permits as an additional property.
type WorkPackageRecord struct {
	v2.WorkPackage
	Capabilities []string `json:"capabilities,omitempty"`
}

// WorkPackageRepository stores canonical work packages.
type WorkPackageRepository struct {
	store *store.Store
	proj  projection.Projection
	newID idgen
}

// NewWorkPackageRepository returns a work-package repository over the store and
// projection.
func NewWorkPackageRepository(s *store.Store, p projection.Projection) *WorkPackageRepository {
	return &WorkPackageRepository{store: s, proj: p, newID: func() string { return "wp_" + randomSuffix() }}
}

// NewID returns a fresh work-package ID.
func (r *WorkPackageRepository) NewID() string { return r.newID() }

// Create validates and stores a work package. Contract version, kind, state,
// acceptance, and revision default when omitted. An explicit ID must be unique.
func (r *WorkPackageRepository) Create(record WorkPackageRecord) (WorkPackageRecord, error) {
	if record.ContractVersion == "" {
		record.ContractVersion = v2.Version
	}
	record.Kind = "work_package"
	if record.State == "" {
		record.State = v2.WorkPackageProposed
	}
	if len(record.Acceptance) == 0 {
		record.Acceptance = append([]string(nil), DefaultAcceptance...)
	}
	if record.ID == "" {
		record.ID = r.newID()
	}
	rev := 1
	record.Revision = &rev
	if err := record.Validate(); err != nil {
		return WorkPackageRecord{}, err
	}
	if _, err := putTyped(r.store, r.proj, "work_package", record.ID, rev, record); err != nil {
		return WorkPackageRecord{}, err
	}
	return record, nil
}

// Get returns a work package by id.
func (r *WorkPackageRepository) Get(id string) (WorkPackageRecord, error) {
	return getTyped[WorkPackageRecord](r.store, "work_package", id)
}

// List returns all work packages sorted by id.
func (r *WorkPackageRepository) List() ([]WorkPackageRecord, error) {
	records, err := r.store.ListRecords("work_package")
	if err != nil {
		return nil, err
	}
	out := make([]WorkPackageRecord, 0, len(records))
	for _, rec := range records {
		var work WorkPackageRecord
		if err := decode(rec, &work); err != nil {
			return nil, err
		}
		out = append(out, work)
	}
	return out, nil
}

// ListByProject returns work packages for one project, sorted by id.
func (r *WorkPackageRepository) ListByProject(projectID string) ([]WorkPackageRecord, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	out := make([]WorkPackageRecord, 0, len(all))
	for _, work := range all {
		if work.ProjectID == projectID {
			out = append(out, work)
		}
	}
	return out, nil
}

// UpdateState changes a work package's canonical state and increments its revision.
func (r *WorkPackageRepository) UpdateState(id string, state v2.WorkPackageState) (WorkPackageRecord, error) {
	record, err := r.Get(id)
	if err != nil {
		return WorkPackageRecord{}, err
	}
	if err := state.Validate(); err != nil {
		return WorkPackageRecord{}, fmt.Errorf("work package %s: %w", id, err)
	}
	if record.State == state {
		return record, nil
	}
	record.State = state
	rev := revisionOf(r.store, "work_package", id) + 1
	record.Revision = &rev
	if err := record.Validate(); err != nil {
		return WorkPackageRecord{}, err
	}
	if _, err := putTyped(r.store, r.proj, "work_package", id, rev, record); err != nil {
		return WorkPackageRecord{}, err
	}
	return record, nil
}
