package session

import (
	"errors"
	"reflect"
	"testing"

	visualizer "github.com/greadee/aa/visualizer"
)

func base(id string, seq int, typ visualizer.EventType, aggKind, aggID string) visualizer.Event {
	e := visualizer.Event{}
	e.ContractVersion = "1.0"
	e.Kind = "event"
	e.ID = id
	e.ProjectID = "p1"
	e.Sequence = seq
	e.OccurredAt = "2026-01-01T00:00:0" + string(rune('0'+seq)) + "Z"
	e.Type = typ
	e.Aggregate = visualizer.EventAggregate{Kind: aggKind, ID: aggID}
	e.Actor = visualizer.Actor{Kind: "agent", ID: "worker_1"}
	return e
}

func scenario() []visualizer.Event {
	e1 := base("e1", 1, "WORK_PACKAGE_CREATED", "work_package", "wp1")
	e2 := base("e2", 2, "EXECUTION_STARTED", "assignment", "as1")
	e2.CausationID = "e1"
	e2.Payload = map[string]any{"workPackageId": "wp1"}
	e3 := base("e3", 3, "ARTIFACT_RECORDED", "artifact", "ar1")
	e3.CausationID = "e1"
	e4 := base("e4", 4, "EXECUTION_COMPLETED", "assignment", "as1")
	e4.CausationID = "e2"
	e4.CorrelationID = "c1"
	e4.Payload = map[string]any{"workPackageId": "wp1", "artifactId": "ar1"}
	return []visualizer.Event{e1, e2, e3, e4}
}

func hasEdge(g visualizer.Graph, from, to visualizer.NodeID, kind visualizer.EdgeKind) bool {
	for _, e := range g.Edges {
		if e.From == from && e.To == to && e.Kind == kind {
			return true
		}
	}
	return false
}

func TestProjectIsDeterministicAcrossOrder(t *testing.T) {
	events := scenario()
	shuffled := []visualizer.Event{events[3], events[1], events[0], events[2]}
	a, err := Project("s1", events)
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	b, err := Project("s1", shuffled)
	if err != nil {
		t.Fatalf("Project shuffled: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("graphs differ:\n%+v\n%+v", a, b)
	}
}

func TestProjectIdentityStable(t *testing.T) {
	g, err := Project("s1", scenario())
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	for _, want := range []visualizer.NodeID{
		SessionNodeID("s1"),
		NodeIDFor(visualizer.NodeWorkPackage, "wp1"),
		NodeIDFor(visualizer.NodeAssignment, "as1"),
		NodeIDFor(visualizer.NodeArtifact, "ar1"),
		ActorNodeID("worker_1"),
	} {
		if _, ok := g.NodeByID(want); !ok {
			t.Fatalf("missing node %q", want)
		}
	}
	// Identity is derived, so it is identical when re-derived.
	if NodeIDFor(visualizer.NodeWorkPackage, "wp1") != "work_package:wp1" {
		t.Fatal("NodeIDFor is not stable")
	}
}

func TestProjectRelations(t *testing.T) {
	g, err := Project("s1", scenario())
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	session := SessionNodeID("s1")
	wp := NodeIDFor(visualizer.NodeWorkPackage, "wp1")
	as := NodeIDFor(visualizer.NodeAssignment, "as1")
	ar := NodeIDFor(visualizer.NodeArtifact, "ar1")
	actor := ActorNodeID("worker_1")

	cases := []struct {
		from, to visualizer.NodeID
		kind     visualizer.EdgeKind
	}{
		{session, wp, visualizer.EdgeContains},
		{session, as, visualizer.EdgeContains},
		{session, ar, visualizer.EdgeContains},
		{actor, wp, visualizer.EdgeAccess},
		{actor, as, visualizer.EdgeAccess},
		{wp, as, visualizer.EdgeCausation},
		{wp, ar, visualizer.EdgeCausation},
		{as, wp, visualizer.EdgeAssigns},
		{as, ar, visualizer.EdgeProduces},
	}
	for _, c := range cases {
		if !hasEdge(g, c.from, c.to, c.kind) {
			t.Fatalf("missing edge %s -%s-> %s", c.from, c.kind, c.to)
		}
	}
}

func TestProjectAttachesCompatibilityMetadata(t *testing.T) {
	e := base("e1", 1, "WORK_PACKAGE_CREATED", "work_package", "wp1")
	e.Payload = map[string]any{"secondary_paths": []any{"a.go", "b.go"}}
	g, err := Project("s1", []visualizer.Event{e})
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	n, _ := g.NodeByID(NodeIDFor(visualizer.NodeWorkPackage, "wp1"))
	if n.Metadata["secondary_paths"] != "a.go,b.go" {
		t.Fatalf("metadata = %v", n.Metadata)
	}
}

func TestProjectRejectsInvalidCompatibilityMetadata(t *testing.T) {
	e := base("e1", 1, "WORK_PACKAGE_CREATED", "work_package", "wp1")
	e.Payload = map[string]any{"secondary_paths": 42}
	_, err := Project("s1", []visualizer.Event{e})
	if !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestProjectEmpty(t *testing.T) {
	if _, err := Project("s1", nil); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
	if _, err := Project("", scenario()); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestSummarize(t *testing.T) {
	s, err := Summarize("s1", scenario())
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if s.StartSequence != 1 || s.EndSequence != 4 || s.EventCount != 4 {
		t.Fatalf("summary = %+v", s)
	}
	if !reflect.DeepEqual(s.Actors, []string{"worker_1"}) {
		t.Fatalf("actors = %v", s.Actors)
	}
	if s.ProjectID != "p1" {
		t.Fatalf("project = %q", s.ProjectID)
	}
	if s.StartedAt.IsZero() || s.EndedAt.IsZero() {
		t.Fatalf("times = %v .. %v", s.StartedAt, s.EndedAt)
	}
}
