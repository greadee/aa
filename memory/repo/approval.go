package repo

import (
	"fmt"
	"time"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// Approval decisions.
const (
	DecisionApprove = "approve"
	DecisionReject  = "reject"
)

// ApprovalRecord is the durable projection of a human approval decision.
//
// Mapping from the control plane decision:
//
//	workPackageId -> ApprovalRecord.WorkPackageID
//	projectId     -> ApprovalRecord.ProjectID
//	decision      -> ApprovalRecord.Decision (approve|reject)
//	actor         -> ApprovalRecord.Actor
//	reason        -> ApprovalRecord.Reason
//	decision time -> ApprovalRecord.DecidedAt (RFC3339Nano)
//
// The record is internal to memory; it is not a v2 contract object. It is keyed
// by work package so the current decision can be re-applied on restart.
type ApprovalRecord struct {
	WorkPackageID string `json:"workPackageId"`
	ProjectID     string `json:"projectId"`
	Decision      string `json:"decision"`
	Actor         string `json:"actor"`
	Reason        string `json:"reason,omitempty"`
	DecidedAt     string `json:"decidedAt"`
}

// ApprovalRepository stores internal approval records.
type ApprovalRepository struct {
	store *store.Store
	proj  projection.Projection
}

// NewApprovalRepository returns an approval repository over the store and projection.
func NewApprovalRepository(s *store.Store, p projection.Projection) *ApprovalRepository {
	return &ApprovalRepository{store: s, proj: p}
}

// Save validates and stores an approval decision, incrementing its revision.
func (r *ApprovalRepository) Save(record ApprovalRecord) (ApprovalRecord, error) {
	if record.WorkPackageID == "" || record.ProjectID == "" {
		return ApprovalRecord{}, fmt.Errorf("approval record: workPackageId and projectId are required")
	}
	if record.Decision != DecisionApprove && record.Decision != DecisionReject {
		return ApprovalRecord{}, fmt.Errorf("approval record: decision must be approve or reject")
	}
	if record.Actor == "" {
		return ApprovalRecord{}, fmt.Errorf("approval record: actor is required")
	}
	if record.DecidedAt == "" {
		record.DecidedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	id := "approval_" + record.WorkPackageID
	rev := revisionOf(r.store, "approval", id) + 1
	if _, err := putTyped(r.store, r.proj, "approval", id, rev, record); err != nil {
		return ApprovalRecord{}, err
	}
	return record, nil
}

// Get returns the current approval for a work package.
func (r *ApprovalRepository) Get(workPackageID string) (ApprovalRecord, error) {
	return getTyped[ApprovalRecord](r.store, "approval", "approval_"+workPackageID)
}

// List returns all approval records sorted by work package id.
func (r *ApprovalRepository) List() ([]ApprovalRecord, error) {
	records, err := r.store.ListRecords("approval")
	if err != nil {
		return nil, err
	}
	out := make([]ApprovalRecord, 0, len(records))
	for _, rec := range records {
		var record ApprovalRecord
		if err := decode(rec, &record); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

// ListByProject returns approvals for one project.
func (r *ApprovalRepository) ListByProject(projectID string) ([]ApprovalRecord, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	out := make([]ApprovalRecord, 0, len(all))
	for _, record := range all {
		if record.ProjectID == projectID {
			out = append(out, record)
		}
	}
	return out, nil
}
