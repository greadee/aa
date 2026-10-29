package repo

import (
	"encoding/json"
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// AssignmentRecord is the durable projection of a scheduler assignment.
//
// Mapping from kernel/scheduler.Assignment:
//
//	Assignment.ID            -> AssignmentRecord.ID
//	Assignment.ProjectID     -> AssignmentRecord.ProjectID
//	Assignment.WorkPackageID -> AssignmentRecord.WorkPackageID
//	Assignment.WorkerID      -> AssignmentRecord.WorkerID
//	Assignment.State         -> AssignmentRecord.State (string)
//	Assignment.Attempt       -> AssignmentRecord.Attempt
//	Assignment.Lease.Owner   -> AssignmentRecord.LeaseOwner
//	Assignment.Lease.ExpiresAt -> AssignmentRecord.LeaseExpiresAt (RFC3339Nano)
//
// The record is internal to memory; it is not a v2 contract object. The
// scheduler emits this shape through its Sink.Record seam.
type AssignmentRecord struct {
	ID             string `json:"id"`
	ProjectID      string `json:"projectId"`
	WorkPackageID  string `json:"workPackageId"`
	WorkerID       string `json:"workerId"`
	State          string `json:"state"`
	Attempt        int    `json:"attempt"`
	LeaseOwner     string `json:"leaseOwner,omitempty"`
	LeaseExpiresAt string `json:"leaseExpiresAt,omitempty"`
}

// AssignmentRepository stores internal assignment records.
type AssignmentRepository struct {
	store *store.Store
	proj  projection.Projection
}

// NewAssignmentRepository returns an assignment repository over the store and projection.
func NewAssignmentRepository(s *store.Store, p projection.Projection) *AssignmentRepository {
	return &AssignmentRepository{store: s, proj: p}
}

// SaveJSON decodes the scheduler's assignment record and stores it, incrementing
// the record revision. Sink callers can forward the raw JSON.
func (r *AssignmentRepository) SaveJSON(data []byte) error {
	var record AssignmentRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("assignment record: %w", err)
	}
	_, err := r.Save(record)
	return err
}

// Validate checks the record's required fields and state enum.
func (rec AssignmentRecord) Validate() error {
	if rec.ID == "" || rec.ProjectID == "" || rec.WorkPackageID == "" || rec.WorkerID == "" {
		return fmt.Errorf("assignment record: id, projectId, workPackageId, and workerId are required")
	}
	if rec.Attempt < 0 {
		return fmt.Errorf("assignment record: attempt must be non-negative")
	}
	if err := v2.AssignmentState(rec.State).Validate(); err != nil {
		return fmt.Errorf("assignment record: %w", err)
	}
	return nil
}

// Save validates and stores an assignment record.
func (r *AssignmentRepository) Save(record AssignmentRecord) (AssignmentRecord, error) {
	if err := record.Validate(); err != nil {
		return AssignmentRecord{}, err
	}
	rev := revisionOf(r.store, "assignment", record.ID) + 1
	if _, err := putTyped(r.store, r.proj, "assignment", record.ID, rev, record); err != nil {
		return AssignmentRecord{}, err
	}
	return record, nil
}

// Get returns an assignment by id.
func (r *AssignmentRepository) Get(id string) (AssignmentRecord, error) {
	record, err := getTyped[AssignmentRecord](r.store, "assignment", id)
	if err != nil {
		return AssignmentRecord{}, err
	}
	if err := record.Validate(); err != nil {
		return AssignmentRecord{}, fmt.Errorf("assignment record %s: %w", id, err)
	}
	return record, nil
}

// List returns all assignment records sorted by id.
func (r *AssignmentRepository) List() ([]AssignmentRecord, error) {
	records, err := r.store.ListRecords("assignment")
	if err != nil {
		return nil, err
	}
	out := make([]AssignmentRecord, 0, len(records))
	for _, rec := range records {
		var record AssignmentRecord
		if err := decode(rec, &record); err != nil {
			return nil, fmt.Errorf("assignment record %s: %w", rec.ID, err)
		}
		if err := record.Validate(); err != nil {
			return nil, fmt.Errorf("assignment record %s: %w", rec.ID, err)
		}
		out = append(out, record)
	}
	return out, nil
}

// ListByProject returns assignments for one project sorted by id.
func (r *AssignmentRepository) ListByProject(projectID string) ([]AssignmentRecord, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	out := make([]AssignmentRecord, 0, len(all))
	for _, record := range all {
		if record.ProjectID == projectID {
			out = append(out, record)
		}
	}
	return out, nil
}
