package controlplane

import (
	"context"
	"fmt"
	"sync"
)

// Fake is a deterministic in-memory Client for offline tests and pilots. It
// makes no network calls.
type Fake struct {
	mu          sync.Mutex
	projects    []Project
	work        map[string][]WorkPackage
	assignments []Assignment
	nodes       []Node
	tools       []Tool
	seq         int
}

// NewFake returns a small, deterministic control plane.
func NewFake() *Fake {
	return &Fake{
		projects: []Project{{ContractVersion: "2.0", Kind: "project_record", ID: "default", Name: "Default", State: "CREATED", LayoutVersion: "1"}},
		work:     map[string][]WorkPackage{},
		nodes:    []Node{{ID: "local", Authority: true, State: "ready"}},
		tools:    []Tool{{ID: "tool_screen", Kind: "builtin", Sandbox: true}},
	}
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
	for _, project := range f.projects {
		if project.ID == id {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("project %q not found", id)
}

func (f *Fake) CreateProject(_ context.Context, id, name string) error {
	if id == "" {
		return fmt.Errorf("project id is required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.projects = append(f.projects, Project{ContractVersion: "2.0", Kind: "project_record", ID: id, Name: name, State: "CREATED", LayoutVersion: "1"})
	return nil
}

func (f *Fake) WorkPackages(_ context.Context, projectID string) ([]WorkPackage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WorkPackage(nil), f.work[projectID]...), nil
}

func (f *Fake) Graph(_ context.Context, projectID string) (TaskGraph, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	nodes := make([]TaskGraphNode, 0, len(f.work[projectID]))
	for _, work := range f.work[projectID] {
		nodes = append(nodes, TaskGraphNode{WorkPackageID: work.ID, DependsOn: append([]string(nil), work.DependsOn...), Readiness: "ready"})
	}
	return TaskGraph{ContractVersion: "2.0", Kind: "task_graph", ID: "graph_" + projectID, ProjectID: projectID, Revision: 1, Nodes: nodes}, nil
}

func (f *Fake) SubmitWork(_ context.Context, projectID, title string, dependsOn []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("wp-%d", f.seq)
	f.work[projectID] = append(f.work[projectID], WorkPackage{
		ContractVersion: "2.0",
		Kind:            "work_package",
		ID:              id,
		Title:           title,
		State:           "PROPOSED",
		Acceptance:      []string{"Work package is implemented and verified."},
		DependsOn:       dependsOn,
	})
	return id, nil
}

func (f *Fake) Assignments(context.Context) ([]Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Assignment(nil), f.assignments...), nil
}

func (f *Fake) Approve(_ context.Context, gateID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.assignments = append(f.assignments, Assignment{ID: gateID, State: "APPROVED"})
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
