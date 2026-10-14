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
		projects: []Project{{ID: "default", Name: "Default"}},
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

func (f *Fake) CreateProject(_ context.Context, id, name string) error {
	if id == "" {
		return fmt.Errorf("project id is required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.projects = append(f.projects, Project{ID: id, Name: name})
	return nil
}

func (f *Fake) WorkPackages(_ context.Context, projectID string) ([]WorkPackage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WorkPackage(nil), f.work[projectID]...), nil
}

func (f *Fake) SubmitWork(_ context.Context, projectID, title string, dependsOn []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("wp-%d", f.seq)
	f.work[projectID] = append(f.work[projectID], WorkPackage{ID: id, Title: title, State: "PROPOSED", DependsOn: dependsOn})
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
