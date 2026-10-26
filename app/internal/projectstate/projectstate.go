// Package projectstate owns durable project and work-package state for the
// control plane. It is the composition-layer model that binds canonical memory
// records to the kernel planner graph; it does not expose transport types.
package projectstate

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	"github.com/greadee/aa/kernel/api"
	"github.com/greadee/aa/kernel/scheduler"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

// State is the durable multi-project control-plane model.
type State struct {
	mu       sync.RWMutex
	projects map[string]*Project

	projectsRepo *repo.ProjectRepository
	workRepo     *repo.WorkPackageRepository
}

// Project is one project's durable record plus its in-process kernel graph.
type Project struct {
	Record    v2.ProjectRecord
	Graph     *planner.Graph
	Scheduler *scheduler.Scheduler
	API       *api.Service
}

// New loads projects and work packages from the store and rebuilds each graph.
// Corrupt or orphaned state fails closed.
func New(st *store.Store, proj projection.Projection) (*State, error) {
	s := &State{
		projects:     map[string]*Project{},
		projectsRepo: repo.NewProjectRepository(st, proj),
		workRepo:     repo.NewWorkPackageRepository(st, proj),
	}

	projects, err := s.projectsRepo.List()
	if err != nil {
		return nil, fmt.Errorf("projectstate: list projects: %w", err)
	}
	for _, record := range projects {
		if _, exists := s.projects[record.ID]; exists {
			return nil, fmt.Errorf("projectstate: duplicate project %q", record.ID)
		}
		s.projects[record.ID] = &Project{
			Record: record,
			Graph:  planner.NewGraph(record.ID, "graph_"+record.ID),
		}
	}

	work, err := s.workRepo.List()
	if err != nil {
		return nil, fmt.Errorf("projectstate: list work packages: %w", err)
	}
	for _, record := range work {
		project, ok := s.projects[record.ProjectID]
		if !ok {
			return nil, fmt.Errorf("projectstate: work package %q references unknown project %q", record.ID, record.ProjectID)
		}
		if err := project.Graph.Add(toPlanner(record)); err != nil {
			return nil, fmt.Errorf("projectstate: restore work package %q: %w", record.ID, err)
		}
	}

	ids := make([]string, 0, len(s.projects))
	for id := range s.projects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		project := s.projects[id]
		if err := project.Graph.Validate(); err != nil {
			return nil, fmt.Errorf("projectstate: validate project %q: %w", id, err)
		}
		if err := attachRuntime(project); err != nil {
			return nil, fmt.Errorf("projectstate: project %q: %w", id, err)
		}
	}
	return s, nil
}

func attachRuntime(project *Project) error {
	sched, err := scheduler.New(project.Graph, scheduler.Config{
		Registry: role_allocator.New(),
	})
	if err != nil {
		return err
	}
	service, err := api.New(api.Config{Scheduler: sched})
	if err != nil {
		return err
	}
	project.Scheduler = sched
	project.API = service
	return nil
}

// Projects returns project records sorted by id.
func (s *State) Projects() []v2.ProjectRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]v2.ProjectRecord, 0, len(s.projects))
	for _, project := range s.projects {
		out = append(out, project.Record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Project returns one project record.
func (s *State) Project(id string) (v2.ProjectRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[id]
	if !ok {
		return v2.ProjectRecord{}, false
	}
	return project.Record, true
}

// CreateProject persists a project and creates its graph/runtime.
func (s *State) CreateProject(id, name string) (v2.ProjectRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.projects[id]; exists {
		return v2.ProjectRecord{}, fmt.Errorf("%w: project %q", repo.ErrExists, id)
	}
	record, err := s.projectsRepo.Create(id, name)
	if err != nil {
		return v2.ProjectRecord{}, err
	}
	project := &Project{
		Record: record,
		Graph:  planner.NewGraph(record.ID, "graph_"+record.ID),
	}
	if err := attachRuntime(project); err != nil {
		return v2.ProjectRecord{}, err
	}
	s.projects[id] = project
	return record, nil
}

// WorkPackages returns one project's work packages sorted by id.
func (s *State) WorkPackages(projectID string) ([]repo.WorkPackageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.projects[projectID]; !ok {
		return nil, fmt.Errorf("projectstate: unknown project %q", projectID)
	}
	return s.workRepo.ListByProject(projectID)
}

// SubmitWork persists a work package and adds it to the project's graph.
func (s *State) SubmitWork(projectID, title string, dependsOn []string) (repo.WorkPackageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.projects[projectID]
	if !ok {
		return repo.WorkPackageRecord{}, fmt.Errorf("projectstate: unknown project %q", projectID)
	}
	if title == "" {
		return repo.WorkPackageRecord{}, fmt.Errorf("projectstate: title is required")
	}
	for _, dep := range dependsOn {
		if dep == "" {
			return repo.WorkPackageRecord{}, fmt.Errorf("projectstate: dependency id is required")
		}
		if _, ok := project.Graph.Get(dep); !ok {
			return repo.WorkPackageRecord{}, fmt.Errorf("projectstate: unknown dependency %q", dep)
		}
	}
	record, err := s.workRepo.Create(repo.WorkPackageRecord{
		WorkPackage: v2.WorkPackage{
			Envelope: v2.Envelope{
				ContractVersion: v2.Version,
				Kind:            "work_package",
				ID:              s.workRepo.NewID(),
				ProjectID:       projectID,
			},
			Title:        title,
			State:        v2.WorkPackageProposed,
			Dependencies: append([]string(nil), dependsOn...),
			Acceptance:   append([]string(nil), repo.DefaultAcceptance...),
		},
	})
	if err != nil {
		return repo.WorkPackageRecord{}, err
	}
	if err := project.Graph.Add(toPlanner(record)); err != nil {
		return repo.WorkPackageRecord{}, err
	}
	return record, nil
}

// ProjectRuntime returns the in-process scheduler/API for one project.
func (s *State) ProjectRuntime(projectID string) (*scheduler.Scheduler, *api.Service, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok {
		return nil, nil, false
	}
	return project.Scheduler, project.API, true
}

func toPlanner(record repo.WorkPackageRecord) planner.WorkPackage {
	return planner.WorkPackage{
		ID:           record.ID,
		Title:        record.Title,
		Role:         record.Role,
		Capabilities: append([]string(nil), record.Capabilities...),
		DependsOn:    append([]string(nil), record.Dependencies...),
		Acceptance:   append([]string(nil), record.Acceptance...),
		State:        fromContractState(record.State),
	}
}

func fromContractState(state v2.WorkPackageState) planner.State {
	switch state {
	case v2.WorkPackageReady:
		return planner.StateReady
	case v2.WorkPackageRunning, v2.WorkPackageQueued:
		return planner.StateRunning
	case v2.WorkPackageComplete, v2.WorkPackageVerified:
		return planner.StateCompleted
	case v2.WorkPackageDeficient:
		return planner.StateDeficient
	default:
		return planner.StateProposed
	}
}

// ErrNotFound is retained for callers that need to distinguish absent state.
var ErrNotFound = errors.New("projectstate: not found")
