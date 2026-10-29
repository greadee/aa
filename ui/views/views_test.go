package views

import (
	"strings"
	"testing"

	"github.com/greadee/aa/ui/controlplane"
	"github.com/greadee/aa/ui/state"
)

func TestRenderDashboard(t *testing.T) {
	m := state.Model{
		Active:    state.ViewProjects,
		ProjectID: "default",
		Status:    controlplane.Status{Projects: 1, Nodes: 1},
		Projects:  []controlplane.Project{{ID: "default", Name: "Default"}},
		Work:      []controlplane.WorkPackage{{ID: "wp-1", Title: "first", State: "PROPOSED"}},
	}
	out := Render(m)
	for _, want := range []string{"aa console", "\u25b8 projects", "Default", "wp-1", "PROPOSED", "projects=1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderToolsView(t *testing.T) {
	m := state.Model{Active: state.ViewTools, Tools: []controlplane.Tool{{ID: "tool_screen", Kind: "builtin", Sandbox: true}}}
	out := Render(m)
	if !strings.Contains(out, "tool_screen") || !strings.Contains(out, "sandbox") {
		t.Fatalf("tools view:\n%s", out)
	}
}

func TestRenderGraphReadiness(t *testing.T) {
	m := state.Model{
		Active:    state.ViewGraph,
		ProjectID: "default",
		Work:      []controlplane.WorkPackage{{ID: "wp-1", Title: "first", State: "PROPOSED"}},
		Graph:     controlplane.TaskGraph{Nodes: []controlplane.TaskGraphNode{{WorkPackageID: "wp-1", Readiness: "ready"}}},
	}
	out := Render(m)
	if !strings.Contains(out, "ready") {
		t.Fatalf("readiness missing:\n%s", out)
	}
}
