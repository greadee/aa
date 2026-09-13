package v1

import "fmt"

// TaskGraphNode is one work package and its dependencies in the graph.
type TaskGraphNode struct {
	WorkPackageID string         `json:"workPackageId"`
	DependsOn     []string       `json:"dependsOn,omitempty"`
	Readiness     ReadinessState `json:"readiness"`
}

// TaskGraph is a versioned dependency graph over work packages.
type TaskGraph struct {
	Envelope
	Nodes  []TaskGraphNode `json:"nodes"`
	Digest string          `json:"digest,omitempty"`
}

// Validate checks the task-graph contract.
func (g TaskGraph) Validate() error {
	if g.Kind != "task_graph" {
		return fmt.Errorf("kind: expected %q, got %q", "task_graph", g.Kind)
	}
	if err := g.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireIdentifier("projectId", g.ProjectID); err != nil {
		return err
	}
	if g.Revision == nil || *g.Revision < 0 {
		return fmt.Errorf("revision: is required and must be >= 0")
	}
	if len(g.Nodes) == 0 {
		return fmt.Errorf("nodes: at least one node is required")
	}
	for i, n := range g.Nodes {
		if err := RequireIdentifier(fmt.Sprintf("nodes[%d].workPackageId", i), n.WorkPackageID); err != nil {
			return err
		}
		if err := n.Readiness.Validate(); err != nil {
			return fmt.Errorf("nodes[%d]: %w", i, err)
		}
		for j, d := range n.DependsOn {
			if err := RequireIdentifier(fmt.Sprintf("nodes[%d].dependsOn[%d]", i, j), d); err != nil {
				return err
			}
		}
	}
	if g.Digest != "" {
		return RequireHash("digest", g.Digest)
	}
	return nil
}
