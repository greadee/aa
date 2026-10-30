// Package projectstate owns durable project and work-package state for the
// control plane. It is the composition-layer model that binds canonical memory
// records to the kernel planner graph; it does not expose transport types.
package projectstate

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	"github.com/greadee/aa/kernel/api"
	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/kernel/scheduler"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

// Options configures the kernel runtime attached to each project.
type Options struct {
	Lifecycle scheduler.WorkerLifecycle
}

// State is the durable multi-project control-plane model.
type State struct {
	mu       sync.RWMutex
	projects map[string]*Project
	opts     Options

	projectsRepo *repo.ProjectRepository
	workRepo     *repo.WorkPackageRepository
	assignments  *repo.AssignmentRepository
	approvals    *repo.ApprovalRepository
}

// Project is one project's durable record plus its in-process kernel graph.
type Project struct {
	Record    v2.ProjectRecord
	Graph     *planner.Graph
	Gate      *gate.HumanGate
	Scheduler *scheduler.Scheduler
	API       *api.Service
}

// New loads projects and work packages from the store and rebuilds each graph.
// Corrupt or orphaned state fails closed.
func New(st *store.Store, proj projection.Projection, options ...Options) (*State, error) {
	opts := Options{}
	if len(options) > 0 {
		opts = options[0]
	}
	s := &State{
		projects:     map[string]*Project{},
		opts:         opts,
		projectsRepo: repo.NewProjectRepository(st, proj),
		workRepo:     repo.NewWorkPackageRepository(st, proj),
		assignments:  repo.NewAssignmentRepository(st, proj),
		approvals:    repo.NewApprovalRepository(st, proj),
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
		workPackage, err := toPlanner(record)
		if err != nil {
			return nil, fmt.Errorf("projectstate: restore work package %q: %w", record.ID, err)
		}
		if err := project.Graph.Add(workPackage); err != nil {
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
		if err := attachRuntime(project, s.opts, assignmentSink{assignments: s.assignments, work: s.workRepo}); err != nil {
			return nil, fmt.Errorf("projectstate: project %q: %w", id, err)
		}
	}
	if err := s.restoreAssignments(); err != nil {
		return nil, err
	}
	if err := s.restoreApprovals(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) restoreAssignments() error {
	assignments, err := s.assignments.List()
	if err != nil {
		return fmt.Errorf("projectstate: list assignments: %w", err)
	}
	byProject := map[string][]scheduler.RestoreAssignment{}
	for _, assignment := range assignments {
		project, ok := s.projects[assignment.ProjectID]
		if !ok {
			return fmt.Errorf("projectstate: assignment %q references unknown project %q", assignment.ID, assignment.ProjectID)
		}
		if _, ok := project.Graph.Get(assignment.WorkPackageID); !ok {
			return fmt.Errorf("projectstate: assignment %q references unknown work package %q", assignment.ID, assignment.WorkPackageID)
		}
		byProject[assignment.ProjectID] = append(byProject[assignment.ProjectID], scheduler.RestoreAssignment{
			ID:             assignment.ID,
			ProjectID:      assignment.ProjectID,
			WorkPackageID:  assignment.WorkPackageID,
			WorkerID:       assignment.WorkerID,
			State:          assignment.State,
			Attempt:        assignment.Attempt,
			LeaseOwner:     assignment.LeaseOwner,
			LeaseExpiresAt: assignment.LeaseExpiresAt,
		})
	}
	now := time.Now().UTC()
	for projectID, records := range byProject {
		project := s.projects[projectID]
		updated, err := project.Scheduler.Restore(records, now)
		if err != nil {
			return fmt.Errorf("projectstate: restore assignments for %q: %w", projectID, err)
		}
		for _, record := range updated {
			if _, err := s.assignments.Save(repo.AssignmentRecord{
				ID:             record.ID,
				ProjectID:      record.ProjectID,
				WorkPackageID:  record.WorkPackageID,
				WorkerID:       record.WorkerID,
				State:          record.State,
				Attempt:        record.Attempt,
				LeaseOwner:     record.LeaseOwner,
				LeaseExpiresAt: record.LeaseExpiresAt,
			}); err != nil {
				return fmt.Errorf("projectstate: reconcile assignment %q: %w", record.ID, err)
			}
		}
	}
	return nil
}

func (s *State) restoreApprovals() error {
	approvals, err := s.approvals.List()
	if err != nil {
		return fmt.Errorf("projectstate: list approvals: %w", err)
	}
	for _, approval := range approvals {
		project, ok := s.projects[approval.ProjectID]
		if !ok {
			return fmt.Errorf("projectstate: approval for %q references unknown project %q", approval.WorkPackageID, approval.ProjectID)
		}
		if _, ok := project.Graph.Get(approval.WorkPackageID); !ok {
			return fmt.Errorf("projectstate: approval references unknown work package %q", approval.WorkPackageID)
		}
		if approval.Decision == repo.DecisionApprove {
			project.API.Approve(approval.WorkPackageID)
		} else {
			project.API.Reject(approval.WorkPackageID, approval.Reason)
		}
	}
	return nil
}

func attachRuntime(project *Project, opts Options, sink scheduler.Sink) error {
	humanGate := gate.NewHumanGate()
	sched, err := scheduler.New(project.Graph, scheduler.Config{
		Registry:  role_allocator.New(),
		Lifecycle: opts.Lifecycle,
		Sink:      sink,
		Gates:     []gate.Gate{humanGate},
	})
	if err != nil {
		return err
	}
	service, err := api.New(api.Config{Scheduler: sched})
	if err != nil {
		return err
	}
	project.Gate = humanGate
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
	return s.createLocked(id, name)
}

// EnsureProject returns an existing project or creates it. It is idempotent and
// is used to seed the default project.
func (s *State) EnsureProject(id, name string) (v2.ProjectRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if project, ok := s.projects[id]; ok {
		return project.Record, false, nil
	}
	record, err := s.createLocked(id, name)
	if err != nil {
		return v2.ProjectRecord{}, false, err
	}
	return record, true, nil
}

func (s *State) createLocked(id, name string) (v2.ProjectRecord, error) {
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
	if err := attachRuntime(project, s.opts, assignmentSink{assignments: s.assignments, work: s.workRepo}); err != nil {
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
	workPackage, err := toPlanner(record)
	if err != nil {
		return repo.WorkPackageRecord{}, err
	}
	if err := project.Graph.Add(workPackage); err != nil {
		return repo.WorkPackageRecord{}, err
	}
	return record, nil
}

// ProjectForWorkPackage finds the project that owns a work package.
func (s *State) ProjectForWorkPackage(workPackageID string) (*Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, project := range s.projects {
		if _, ok := project.Graph.Get(workPackageID); ok {
			return project, true
		}
	}
	return nil, false
}

// AssignmentStatuses returns durable assignment views across all projects.
func (s *State) AssignmentStatuses() []scheduler.AssignmentStatus {
	records, err := s.assignments.List()
	if err != nil {
		return nil
	}
	out := make([]scheduler.AssignmentStatus, 0, len(records))
	for _, record := range records {
		out = append(out, scheduler.AssignmentStatus{
			ID:            record.ID,
			WorkPackageID: record.WorkPackageID,
			WorkerID:      record.WorkerID,
			State:         record.State,
			Attempt:       record.Attempt,
		})
	}
	return out
}

// ActiveAssignments counts assignments that have not reached a terminal state.
func (s *State) ActiveAssignments() int {
	active := 0
	for _, assignment := range s.AssignmentStatuses() {
		switch assignment.State {
		case "accepted", "failed", "canceled", "expired":
		default:
			active++
		}
	}
	return active
}

// RecordApproval persists a decision and applies it to the project scheduler.
// Reject never approves. If the work package already has a decision or has
// reached a terminal assignment state, the decision is rejected as resolved.
func (s *State) RecordApproval(projectID, workPackageID, decision, actor, reason string) (repo.ApprovalRecord, error) {
	if decision != repo.DecisionApprove && decision != repo.DecisionReject {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: decision must be approve or reject", ErrInvalidApproval)
	}
	if actor == "" {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: actor is required", ErrInvalidApproval)
	}
	// Serialize the entire decision: lookup, pending-gate validation, durable
	// save, and gate application. Concurrent approve/reject calls for one work
	// package therefore produce exactly one decision.
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.projects[projectID]
	if !ok {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: project %q", ErrNotFound, projectID)
	}
	if _, ok := project.Graph.Get(workPackageID); !ok {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: work package %q", ErrNotFound, workPackageID)
	}

	if _, err := s.approvals.Get(workPackageID); err == nil {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: work package %q", ErrApprovalResolved, workPackageID)
	} else if !errors.Is(err, store.ErrNotFound) {
		return repo.ApprovalRecord{}, err
	}
	assignments, err := s.assignments.ListByProject(projectID)
	if err != nil {
		return repo.ApprovalRecord{}, err
	}
	var latest *repo.AssignmentRecord
	for i := range assignments {
		assignment := &assignments[i]
		if assignment.WorkPackageID != workPackageID {
			continue
		}
		if latest == nil || assignment.Attempt >= latest.Attempt {
			latest = assignment
		}
	}
	if latest == nil {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: work package %q has no pending gate", ErrApprovalResolved, workPackageID)
	}
	if latest.State != "awaiting_gates" {
		return repo.ApprovalRecord{}, fmt.Errorf("%w: work package %q assignment is %s", ErrApprovalResolved, workPackageID, latest.State)
	}

	record, err := s.approvals.Save(repo.ApprovalRecord{
		WorkPackageID: workPackageID,
		ProjectID:     projectID,
		Decision:      decision,
		Actor:         actor,
		Reason:        reason,
	})
	if err != nil {
		return repo.ApprovalRecord{}, err
	}
	if decision == repo.DecisionApprove {
		project.API.Approve(workPackageID)
	} else {
		project.API.Reject(workPackageID, reason)
	}
	// The durable record and the gate now agree. Resolve advances an in-memory
	// awaiting-gates assignment; its error is ignored because a restarted
	// process may not have restored scheduler state yet.
	_, _ = project.API.Resolve(workPackageID)
	return record, nil
}

// ApprovalRecords returns all durable approval decisions.
func (s *State) ApprovalRecords() ([]repo.ApprovalRecord, error) {
	return s.approvals.List()
}

// WorkPackage returns one project-scoped work package.
func (s *State) WorkPackage(projectID, id string) (repo.WorkPackageRecord, bool) {
	record, err := s.workRepo.Get(id)
	if err != nil || record.ProjectID != projectID {
		return repo.WorkPackageRecord{}, false
	}
	return record, true
}

// Graph returns the project's task graph in contract v2 shape.
func (s *State) Graph(projectID string) (v2.TaskGraph, bool) {
	s.mu.RLock()
	project, ok := s.projects[projectID]
	if !ok {
		s.mu.RUnlock()
		return v2.TaskGraph{}, false
	}
	graph := project.Graph
	ready := map[string]bool{}
	for _, id := range graph.Ready() {
		ready[id] = true
	}
	revision := graph.Revision
	nodes := make([]v2.TaskGraphNode, 0)
	for _, wp := range graph.List() {
		nodes = append(nodes, v2.TaskGraphNode{
			WorkPackageID: wp.ID,
			DependsOn:     append([]string(nil), wp.DependsOn...),
			Readiness:     readinessFor(wp, ready),
		})
	}
	s.mu.RUnlock()
	return v2.TaskGraph{
		Envelope: v2.Envelope{
			ContractVersion: v2.Version,
			Kind:            "task_graph",
			ID:              "graph_" + projectID,
		},
		ProjectID: projectID,
		Revision:  &revision,
		Nodes:     nodes,
	}, true
}

func readinessFor(wp planner.WorkPackage, ready map[string]bool) v2.ReadinessState {
	switch wp.State {
	case planner.StateCompleted:
		return v2.ReadinessSatisfied
	case planner.StateRunning:
		return v2.ReadinessDispatched
	case planner.StateReady:
		return v2.ReadinessReady
	case planner.StateProposed:
		if ready[wp.ID] {
			return v2.ReadinessReady
		}
		return v2.ReadinessBlocked
	default:
		return v2.ReadinessBlocked
	}
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

type assignmentSink struct {
	assignments *repo.AssignmentRepository
	work        *repo.WorkPackageRepository
}

func (assignmentSink) Event(int, string, map[string]any) error { return nil }

func (s assignmentSink) Record(kind, id string, data []byte) error {
	if kind != "assignment" {
		return nil
	}
	return s.assignments.SaveJSON(data)
}

// RecordWorkPackageState durably records scheduler lifecycle transitions so a
// completed or running work package keeps its state across restart.
func (s assignmentSink) RecordWorkPackageState(projectID, workPackageID, state string) error {
	contractState, err := toContractState(state)
	if err != nil {
		return err
	}
	_, err = s.work.UpdateState(workPackageID, contractState)
	return err
}

func toContractState(state string) (v2.WorkPackageState, error) {
	switch state {
	case string(planner.StateProposed):
		return v2.WorkPackageProposed, nil
	case string(planner.StateReady):
		return v2.WorkPackageReady, nil
	case string(planner.StateRunning):
		return v2.WorkPackageRunning, nil
	case string(planner.StateCompleted):
		return v2.WorkPackageComplete, nil
	case string(planner.StateDeficient):
		return v2.WorkPackageDeficient, nil
	default:
		return "", fmt.Errorf("unknown planner state %q", state)
	}
}

func toPlanner(record repo.WorkPackageRecord) (planner.WorkPackage, error) {
	state, err := fromContractState(record.State)
	if err != nil {
		return planner.WorkPackage{}, err
	}
	return planner.WorkPackage{
		ID:           record.ID,
		Title:        record.Title,
		Role:         record.Role,
		Capabilities: append([]string(nil), record.Capabilities...),
		DependsOn:    append([]string(nil), record.Dependencies...),
		Acceptance:   append([]string(nil), record.Acceptance...),
		State:        state,
	}, nil
}

// fromContractState maps every canonical work-package state explicitly. Unknown
// states fail closed instead of being silently treated as PROPOSED.
func fromContractState(state v2.WorkPackageState) (planner.State, error) {
	switch state {
	case v2.WorkPackageProposed:
		return planner.StateProposed, nil
	case v2.WorkPackageReady:
		return planner.StateReady, nil
	case v2.WorkPackageQueued, v2.WorkPackageRunning, v2.WorkPackageVerifying:
		return planner.StateRunning, nil
	case v2.WorkPackageComplete, v2.WorkPackageVerified:
		return planner.StateCompleted, nil
	case v2.WorkPackageDeficient:
		return planner.StateDeficient, nil
	default:
		return "", fmt.Errorf("unknown work package state %q", state)
	}
}

// ErrNotFound is returned when the project or work package does not exist.
var ErrNotFound = errors.New("projectstate: not found")

// ErrInvalidApproval is returned for a malformed approval decision.
var ErrInvalidApproval = errors.New("projectstate: invalid approval")

// ErrApprovalResolved is returned when a work package already has a decision
// or has reached a terminal assignment state.
var ErrApprovalResolved = errors.New("projectstate: approval already resolved")
