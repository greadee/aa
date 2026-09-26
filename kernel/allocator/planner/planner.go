// Package plan models work packages and their dependency graph and computes
// deterministic dispatch readiness. It never calls a model.
package planner

import (
	"errors"
	"fmt"
	"sort"
)

// State is the lifecycle state of a work package in the graph.
type State string

// Work-package states used by the kernel graph.
const (
	StateProposed  State = "PROPOSED"
	StateReady     State = "READY"
	StateRunning   State = "RUNNING"
	StateCompleted State = "COMPLETED"
	StateDeficient State = "DEFICIENT"
)

// WorkPackage is a bounded, verifiable unit of work.
type WorkPackage struct {
	ID           string
	Title        string
	Role         string
	Trade        string
	Capabilities []string
	DependsOn    []string
	Acceptance   []string
	State        State
}

// Graph is a validated dependency graph over work packages.
type Graph struct {
	ProjectID string
	ID        string
	Revision  int
	nodes     map[string]*WorkPackage
	order     []string
}

// NewGraph returns an empty graph.
func NewGraph(projectID, id string) *Graph {
	return &Graph{ProjectID: projectID, ID: id, nodes: make(map[string]*WorkPackage), Revision: 1}
}

// ErrUnknownNode is returned for operations on a missing work package.
var ErrUnknownNode = errors.New("plan: unknown work package")

// Add inserts a work package. The state defaults to PROPOSED.
func (g *Graph) Add(wp WorkPackage) error {
	if wp.ID == "" {
		return fmt.Errorf("plan: work package id is required")
	}
	if _, ok := g.nodes[wp.ID]; ok {
		return fmt.Errorf("plan: duplicate work package %s", wp.ID)
	}
	if wp.State == "" {
		wp.State = StateProposed
	}
	copy := wp
	g.nodes[wp.ID] = &copy
	g.order = append(g.order, wp.ID)
	return nil
}

// Get returns a work package by id.
func (g *Graph) Get(id string) (WorkPackage, bool) {
	wp, ok := g.nodes[id]
	if !ok {
		return WorkPackage{}, false
	}
	return *wp, true
}

// List returns work packages in insertion order.
func (g *Graph) List() []WorkPackage {
	out := make([]WorkPackage, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, *g.nodes[id])
	}
	return out
}

// Validate checks for missing dependencies, self-dependencies, and cycles.
func (g *Graph) Validate() error {
	for _, id := range g.order {
		wp := g.nodes[id]
		for _, dep := range wp.DependsOn {
			if dep == id {
				return fmt.Errorf("plan: %s depends on itself", id)
			}
			if _, ok := g.nodes[dep]; !ok {
				return fmt.Errorf("plan: %s depends on missing %s", id, dep)
			}
		}
	}
	if _, err := g.Topological(); err != nil {
		return err
	}
	return nil
}

// Topological returns ids in deterministic dependency order.
func (g *Graph) Topological() ([]string, error) {
	indegree := make(map[string]int, len(g.nodes))
	children := make(map[string][]string, len(g.nodes))
	for _, id := range g.order {
		indegree[id] = len(g.nodes[id].DependsOn)
		for _, dep := range g.nodes[id].DependsOn {
			children[dep] = append(children[dep], id)
		}
	}
	for k := range children {
		sort.Strings(children[k])
	}
	var queue []string
	for _, id := range g.order {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	var order []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
		sort.Strings(queue)
	}
	if len(order) != len(g.nodes) {
		return nil, fmt.Errorf("plan: dependency cycle detected")
	}
	return order, nil
}

// SetState updates a work package's state.
func (g *Graph) SetState(id string, state State) error {
	wp, ok := g.nodes[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownNode, id)
	}
	wp.State = state
	return nil
}

// Ready returns the ids of work packages that are not yet dispatched and whose
// dependencies are all completed, sorted by id.
func (g *Graph) Ready() []string {
	var ready []string
	for _, id := range g.order {
		wp := g.nodes[id]
		if wp.State != StateProposed && wp.State != StateReady {
			continue
		}
		blocked := false
		for _, dep := range wp.DependsOn {
			depWP, ok := g.nodes[dep]
			if !ok || depWP.State != StateCompleted {
				blocked = true
				break
			}
		}
		if !blocked {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	return ready
}
