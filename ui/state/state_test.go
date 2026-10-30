package state

import (
	"context"
	"testing"

	"github.com/greadee/aa/ui/controlplane"
)

func TestLoadAndCycle(t *testing.T) {
	m, err := Load(context.Background(), controlplane.NewFake(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if m.Status.Projects == 0 || len(m.Nodes) == 0 || len(m.Tools) == 0 {
		t.Fatalf("model not loaded: %+v", m)
	}
	if m.Project.ID != "default" || m.Graph.Kind != "task_graph" {
		t.Fatalf("project state not loaded: %+v", m)
	}
	if len(m.Issues) == 0 || len(m.Strategies) == 0 {
		t.Fatalf("memory read model not loaded: %+v", m)
	}
	if len(m.Sessions) == 0 || m.Report.SessionID == "" {
		t.Fatalf("reports read model not loaded: %+v", m)
	}
	if got := Cycle(ViewProjects, 1); got != ViewGraph {
		t.Fatalf("cycle forward = %s", got)
	}
	if got := Cycle(ViewProjects, -1); got != ViewApprovals {
		t.Fatalf("cycle backward = %s", got)
	}
}
