package controlplane

import (
	"context"
	"fmt"
	"sync"
)

// Fake is a deterministic in-memory Client for offline tests and pilots. It
// makes no network calls and enforces the same duplicate, not-found, and
// validation semantics as the HTTP control plane.
type Fake struct {
	mu          sync.Mutex
	projects    []Project
	work        map[string][]WorkPackage
	assignments []Assignment
	decisions   map[string]string
	nodes       []Node
	tools       []Tool
	seq         int
}

// NewFake returns a small, deterministic control plane.
func NewFake() *Fake {
	return &Fake{
		projects:  []Project{{ContractVersion: "2.0", Kind: "project_record", ID: "default", Name: "Default", State: "CREATED", LayoutVersion: "1"}},
		work:      map[string][]WorkPackage{},
		decisions: map[string]string{},
		nodes:     []Node{{ID: "local", Authority: true, State: "ready"}},
		tools:     []Tool{{ID: "tool_screen", Kind: "builtin", Sandbox: true}},
	}
}

func invalid(message string) error {
	return &Error{Status: 400, Code: "aa.invalid_params", Message: message}
}

func notFound(message string) error {
	return &Error{Status: 404, Code: "aa.not_found", Message: message}
}

func conflict(message string) error {
	return &Error{Status: 409, Code: "aa.conflict", Message: message}
}

func (f *Fake) findProjectLocked(id string) (Project, bool) {
	for _, project := range f.projects {
		if project.ID == id {
			return project, true
		}
	}
	return Project{}, false
}

func (f *Fake) workPackageLocked(id string) (string, WorkPackage, bool) {
	for projectID, packages := range f.work {
		for _, work := range packages {
			if work.ID == id {
				return projectID, work, true
			}
		}
	}
	return "", WorkPackage{}, false
}

func (f *Fake) Status(context.Context) (Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Status{Projects: len(f.projects), ActiveAssignments: len(f.assignments), Nodes: len(f.nodes)}, nil
}

func (f *Fake) Projects(context.Context) ([]Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Project(nil), f.projects...), nil
}

func (f *Fake) Project(_ context.Context, id string) (Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	project, ok := f.findProjectLocked(id)
	if !ok {
		return Project{}, notFound(fmt.Sprintf("project %q not found", id))
	}
	return project, nil
}

func (f *Fake) CreateProject(_ context.Context, id, name string) error {
	if id == "" {
		return invalid("project id is required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.findProjectLocked(id); ok {
		return conflict(fmt.Sprintf("project %q already exists", id))
	}
	f.projects = append(f.projects, Project{ContractVersion: "2.0", Kind: "project_record", ID: id, Name: name, State: "CREATED", LayoutVersion: "1"})
	return nil
}

func (f *Fake) WorkPackages(_ context.Context, projectID string) ([]WorkPackage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.findProjectLocked(projectID); !ok {
		return nil, notFound(fmt.Sprintf("project %q not found", projectID))
	}
	return append([]WorkPackage{}, f.work[projectID]...), nil
}

func (f *Fake) Graph(_ context.Context, projectID string) (TaskGraph, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.findProjectLocked(projectID); !ok {
		return TaskGraph{}, notFound(fmt.Sprintf("project %q not found", projectID))
	}
	nodes := make([]TaskGraphNode, 0, len(f.work[projectID]))
	for _, work := range f.work[projectID] {
		nodes = append(nodes, TaskGraphNode{WorkPackageID: work.ID, DependsOn: append([]string(nil), work.DependsOn...), Readiness: "ready"})
	}
	return TaskGraph{ContractVersion: "2.0", Kind: "task_graph", ID: "graph_" + projectID, ProjectID: projectID, Revision: 1, Nodes: nodes}, nil
}

func (f *Fake) SubmitWork(_ context.Context, projectID, title string, dependsOn []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.findProjectLocked(projectID); !ok {
		return "", notFound(fmt.Sprintf("project %q not found", projectID))
	}
	if title == "" {
		return "", invalid("title is required")
	}
	for _, dep := range dependsOn {
		if _, _, ok := f.workPackageLocked(dep); !ok {
			return "", invalid(fmt.Sprintf("unknown dependency %q", dep))
		}
	}
	f.seq++
	id := fmt.Sprintf("wp-%d", f.seq)
	f.work[projectID] = append(f.work[projectID], WorkPackage{
		ContractVersion: "2.0",
		Kind:            "work_package",
		ID:              id,
		Title:           title,
		State:           "PROPOSED",
		Acceptance:      []string{"Work package is implemented and verified."},
		DependsOn:       append([]string(nil), dependsOn...),
	})
	return id, nil
}

func (f *Fake) Assignments(context.Context) ([]Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Assignment(nil), f.assignments...), nil
}

func (f *Fake) Approve(ctx context.Context, gateID string) error {
	return f.Decide(ctx, gateID, "approve", "console", "")
}

func (f *Fake) Reject(ctx context.Context, gateID, actor, reason string) error {
	return f.Decide(ctx, gateID, "reject", actor, reason)
}

func (f *Fake) Decide(_ context.Context, gateID, decision, actor, reason string) error {
	if decision != "approve" && decision != "reject" {
		return invalid("decision must be approve or reject")
	}
	if actor == "" {
		return invalid("actor is required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, _, ok := f.workPackageLocked(gateID); !ok {
		return notFound(fmt.Sprintf("work package %q not found", gateID))
	}
	if _, ok := f.decisions[gateID]; ok {
		return conflict(fmt.Sprintf("work package %q already has a decision", gateID))
	}
	f.decisions[gateID] = decision
	_ = reason
	return nil
}

func (f *Fake) Nodes(context.Context) ([]Node, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Node(nil), f.nodes...), nil
}

func (f *Fake) Tools(context.Context) ([]Tool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Tool(nil), f.tools...), nil
}
