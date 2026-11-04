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
	for _, want := range []string{"aa console", "\u25b8 projects", "Default", "wp-1", "PROPOSED", "projects=1", "j/k or arrows/Tab"} {
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

func TestRenderMemoryAndReports(t *testing.T) {
	m := state.Model{
		Active:    state.ViewMemory,
		ProjectID: "default",
		Issues:    []controlplane.MemoryItem{{ID: "issue-demo", Status: "open", Severity: "P3", Title: "Demo issue"}},
		Strategies: []controlplane.MemoryItem{
			{ID: "strategy-demo", Lifecycle: "ACTIVE", Title: "Demo strategy"},
		},
	}
	out := Render(m)
	if !strings.Contains(out, "issue-demo") || !strings.Contains(out, "strategy-demo") {
		t.Fatalf("memory view:\n%s", out)
	}

	m.Active = state.ViewReports
	m.Sessions = []controlplane.VisualizerSession{{ID: "session-demo", EventCount: 3}}
	m.Report = controlplane.ObservationReport{SessionID: "session-demo", Events: 3, FirstSequence: 1, LastSequence: 3}
	out = Render(m)
	if !strings.Contains(out, "session-demo") || !strings.Contains(out, "REPORT") {
		t.Fatalf("reports view:\n%s", out)
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
