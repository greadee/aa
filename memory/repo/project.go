package repo

import (
	"errors"
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// ProjectRepository stores canonical project records.
type ProjectRepository struct {
	store *store.Store
	proj  projection.Projection
}

// NewProjectRepository returns a project repository over the store and projection.
func NewProjectRepository(s *store.Store, p projection.Projection) *ProjectRepository {
	return &ProjectRepository{store: s, proj: p}
}

// Create validates and stores a new project record. ID is required and must be
// unique.
func (r *ProjectRepository) Create(id, name string) (v2.ProjectRecord, error) {
	if id == "" {
		return v2.ProjectRecord{}, fmt.Errorf("project id: is required")
	}
	if _, err := r.Get(id); err == nil {
		return v2.ProjectRecord{}, fmt.Errorf("%w: project %q", ErrExists, id)
	} else if !errors.Is(err, store.ErrNotFound) {
		return v2.ProjectRecord{}, err
	}
	rev := 1
	record := v2.ProjectRecord{
		Envelope: v2.Envelope{
			ContractVersion: v2.Version,
			Kind:            "project_record",
			ID:              id,
			Revision:        &rev,
		},
		Name:          name,
		State:         v2.ProjectCreated,
		LayoutVersion: "1",
	}
	if err := record.Validate(); err != nil {
		return v2.ProjectRecord{}, err
	}
	if _, err := putTyped(r.store, r.proj, "project_record", id, rev, record); err != nil {
		return v2.ProjectRecord{}, err
	}
	return record, nil
}

// Get returns a project by id.
func (r *ProjectRepository) Get(id string) (v2.ProjectRecord, error) {
	return getTyped[v2.ProjectRecord](r.store, "project_record", id)
}

// List returns all projects sorted by id.
func (r *ProjectRepository) List() ([]v2.ProjectRecord, error) {
	records, err := r.store.ListRecords("project_record")
	if err != nil {
		return nil, err
	}
	out := make([]v2.ProjectRecord, 0, len(records))
	for _, rec := range records {
		var project v2.ProjectRecord
		if err := decode(rec, &project); err != nil {
			return nil, err
		}
		out = append(out, project)
	}
	return out, nil
}
