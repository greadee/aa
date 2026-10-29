// Package state holds the console's view-model layer: the active view and the
// data loaded from the control plane. Surfaces render this; nothing else.
package state

import (
	"context"

	"github.com/greadee/aa/ui/controlplane"
)

// View identifies a console view.
type View string

// Views.
const (
	ViewProjects  View = "projects"
	ViewGraph     View = "graph"
	ViewMemory    View = "memory"
	ViewNodes     View = "nodes"
	ViewTools     View = "tools"
	ViewReports   View = "reports"
	ViewApprovals View = "approvals"
)

// Nav is the navigation order.
var Nav = []View{ViewProjects, ViewGraph, ViewMemory, ViewNodes, ViewTools, ViewReports, ViewApprovals}

// Model is the console view model.
type Model struct {
	Active      View
	ProjectID   string
	Status      controlplane.Status
	Project     controlplane.Project
	Projects    []controlplane.Project
	Work        []controlplane.WorkPackage
	Graph       controlplane.TaskGraph
	Assignments []controlplane.Assignment
	Nodes       []controlplane.Node
	Tools       []controlplane.Tool
	Message     string
}

// Load builds a model by reading the control plane. A status read failure is
// returned; the other reads are best-effort so the console still renders.
func Load(ctx context.Context, c controlplane.Client, projectID string) (Model, error) {
	m := Model{Active: ViewProjects, ProjectID: projectID}
	st, err := c.Status(ctx)
	if err != nil {
		return m, err
	}
	m.Status = st
	m.Project, _ = c.Project(ctx, projectID)
	m.Projects, _ = c.Projects(ctx)
	m.Work, _ = c.WorkPackages(ctx, projectID)
	m.Graph, _ = c.Graph(ctx, projectID)
	m.Assignments, _ = c.Assignments(ctx)
	m.Nodes, _ = c.Nodes(ctx)
	m.Tools, _ = c.Tools(ctx)
	return m, nil
}

// Cycle returns the next/previous view in the nav.
func Cycle(active View, delta int) View {
	idx := 0
	for i, v := range Nav {
		if v == active {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(Nav)) % len(Nav)
	return Nav[idx]
}
