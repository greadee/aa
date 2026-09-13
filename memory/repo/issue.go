package repo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// IssueRepository stores and transitions canonical issues.
type IssueRepository struct {
	store *store.Store
	proj  projection.Projection
	newID idgen
}

// NewIssueRepository returns an issue repository over the store and projection.
func NewIssueRepository(s *store.Store, p projection.Projection) *IssueRepository {
	return &IssueRepository{store: s, proj: p, newID: func() string { return "iss_" + randomSuffix() }}
}

// Create validates and stores a new issue, assigning id and revision when absent.
func (r *IssueRepository) Create(issue v1.Issue) (v1.Issue, error) {
	if issue.ContractVersion == "" {
		issue.ContractVersion = v1.Version
	}
	issue.Kind = "issue"
	if issue.ID == "" {
		issue.ID = r.newID()
	}
	if issue.Status == "" {
		issue.Status = v1.IssueOpen
	}
	rev := 1
	issue.Revision = &rev
	if err := issue.Validate(); err != nil {
		return v1.Issue{}, err
	}
	if _, err := putTyped(r.store, r.proj, "issue", issue.ID, rev, issue); err != nil {
		return v1.Issue{}, err
	}
	return issue, nil
}

// Get returns an issue by id.
func (r *IssueRepository) Get(id string) (v1.Issue, error) {
	return getTyped[v1.Issue](r.store, "issue", id)
}

// List returns all issues, sorted by id.
func (r *IssueRepository) List() ([]v1.Issue, error) {
	records, err := r.store.ListRecords("issue")
	if err != nil {
		return nil, err
	}
	out := make([]v1.Issue, 0, len(records))
	for _, rec := range records {
		var issue v1.Issue
		if err := decode(rec, &issue); err != nil {
			return nil, err
		}
		out = append(out, issue)
	}
	return out, nil
}

// SetStatus transitions an issue to a new status.
func (r *IssueRepository) SetStatus(id string, status v1.IssueState) (v1.Issue, error) {
	issue, err := r.Get(id)
	if err != nil {
		return v1.Issue{}, err
	}
	if issue.Status == status {
		return issue, nil
	}
	if err := status.Validate(); err != nil {
		return v1.Issue{}, fmt.Errorf("issue %s: %w", id, err)
	}
	issue.Status = status
	rev := revisionOf(r.store, "issue", id) + 1
	issue.Revision = &rev
	if err := issue.Validate(); err != nil {
		return v1.Issue{}, err
	}
	if _, err := putTyped(r.store, r.proj, "issue", issue.ID, rev, issue); err != nil {
		return v1.Issue{}, err
	}
	return issue, nil
}

func randomSuffix() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
