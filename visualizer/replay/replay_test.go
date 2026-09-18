package replay

import (
	"errors"
	"reflect"
	"testing"

	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/layout"
	"github.com/greadee/aa/visualizer/session"
)

func ev(id string, seq int, typ visualizer.EventType, aggKind, aggID string) visualizer.Event {
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
	e1 := ev("e1", 1, "WORK_PACKAGE_CREATED", "work_package", "wp1")
	e2 := ev("e2", 2, "EXECUTION_STARTED", "assignment", "as1")
	e2.CausationID = "e1"
	e2.Payload = map[string]any{"workPackageId": "wp1"}
	e3 := ev("e3", 3, "ARTIFACT_RECORDED", "artifact", "ar1")
	e3.CausationID = "e1"
	e4 := ev("e4", 4, "EXECUTION_COMPLETED", "assignment", "as1")
	e4.CausationID = "e2"
	e4.CorrelationID = "c1"
	e4.Payload = map[string]any{"workPackageId": "wp1", "artifactId": "ar1"}
	return []visualizer.Event{e1, e2, e3, e4}
}

func TestBuildOneFramePerEventInSequenceOrder(t *testing.T) {
	list, err := Build("s1", scenario())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if list.SessionID != "s1" {
		t.Fatalf("session = %q", list.SessionID)
	}
	if len(list.Frames) != 4 {
		t.Fatalf("frames = %d, want 4", len(list.Frames))
	}
	for i, f := range list.Frames {
		if f.Index != i {
			t.Fatalf("frame[%d].Index = %d", i, f.Index)
		}
		if f.Sequence != i+1 {
			t.Fatalf("frame[%d].Sequence = %d, want %d", i, f.Sequence, i+1)
		}
		if _, ok := f.Graph.NodeByID(session.SessionNodeID("s1")); !ok {
			t.Fatalf("frame[%d] missing session node", i)
		}
		if len(f.Placement) != len(f.Graph.Nodes) {
			t.Fatalf("frame[%d] placed %d of %d nodes", i, len(f.Placement), len(f.Graph.Nodes))
		}
	}
}

func TestBuildIsOrderIndependent(t *testing.T) {
	events := scenario()
	shuffled := []visualizer.Event{events[3], events[1], events[0], events[2]}
	a, err := Build("s1", events)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	b, err := Build("s1", shuffled)
	if err != nil {
		t.Fatalf("Build shuffled: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("frame lists differ:\n%+v\n%+v", a, b)
	}
}

func TestBuildUsesProjectionAndLayoutPath(t *testing.T) {
	events := scenario()
	list, err := Build("s1", events)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	first := list.Frames[0]
	wantGraph, err := session.Project("s1", events[:1])
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if !reflect.DeepEqual(first.Graph, wantGraph) {
		t.Fatal("first frame graph is not the prefix projection")
	}
	wantPlacement, err := layout.Layout(wantGraph)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if !reflect.DeepEqual(first.Placement, wantPlacement) {
		t.Fatal("first frame placement is not the graph layout")
	}
}

func TestBuildRejectsEmpty(t *testing.T) {
	if _, err := Build("s1", nil); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
	if _, err := Build("", scenario()); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func mustCursor(t *testing.T, list FrameList) *Cursor {
	t.Helper()
	c, err := list.Cursor()
	if err != nil {
		t.Fatalf("Cursor: %v", err)
	}
	return c
}

func TestCursorSnapshotAndStep(t *testing.T) {
	c := mustCursor(t, mustBuild(t))
	f, err := c.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if f.Sequence != 1 || c.Index() != 0 || !c.AtStart() || c.AtEnd() {
		t.Fatalf("initial cursor = %+v index=%d", f, c.Index())
	}
	f, err = c.Step(1)
	if err != nil {
		t.Fatalf("Step: %v", err)
	}
	if f.Sequence != 2 || c.Index() != 1 {
		t.Fatalf("after Step(1): seq=%d index=%d", f.Sequence, c.Index())
	}
	f, err = c.Step(-1)
	if err != nil {
		t.Fatalf("Step back: %v", err)
	}
	if f.Sequence != 1 || c.Index() != 0 {
		t.Fatalf("after Step(-1): seq=%d index=%d", f.Sequence, c.Index())
	}
	if _, err := c.Step(-1); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if c.Index() != 0 {
		t.Fatalf("out-of-range step moved cursor to %d", c.Index())
	}
	if _, err := c.Step(c.Len()); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestCursorSeek(t *testing.T) {
	c := mustCursor(t, mustBuild(t))
	f, err := c.Seek(2)
	if err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if f.Sequence != 3 || c.Index() != 2 {
		t.Fatalf("Seek(2): seq=%d index=%d", f.Sequence, c.Index())
	}
	if _, err := c.Seek(-1); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if _, err := c.Seek(c.Len()); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if c.Index() != 2 {
		t.Fatalf("out-of-range seek moved cursor to %d", c.Index())
	}
	f, err = c.Reset()
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if f.Sequence != 1 || !c.AtStart() {
		t.Fatalf("after Reset: seq=%d index=%d", f.Sequence, c.Index())
	}
}

func TestCursorSeekSequence(t *testing.T) {
	c := mustCursor(t, mustBuild(t))
	f, err := c.SeekSequence(3)
	if err != nil {
		t.Fatalf("SeekSequence: %v", err)
	}
	if f.Sequence != 3 || c.Index() != 2 {
		t.Fatalf("SeekSequence(3): seq=%d index=%d", f.Sequence, c.Index())
	}
	if _, err := c.SeekSequence(99); !errors.Is(err, visualizer.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if c.Index() != 2 {
		t.Fatalf("failed seek moved cursor to %d", c.Index())
	}
	f, err = c.SeekSequence(4)
	if err != nil {
		t.Fatalf("SeekSequence: %v", err)
	}
	if !c.AtEnd() {
		t.Fatalf("cursor not at end after seeking last sequence: %d", c.Index())
	}
	if f.Sequence != 4 {
		t.Fatalf("seq = %d, want 4", f.Sequence)
	}
}

func TestCursorEmpty(t *testing.T) {
	if _, err := NewCursor(nil); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
	var zero Cursor
	if _, err := zero.Snapshot(); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("zero Snapshot err = %v, want ErrEmpty", err)
	}
	if _, err := zero.Seek(0); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("zero Seek err = %v, want ErrEmpty", err)
	}
	if len(zero.frames) != 0 || zero.Index() != 0 || zero.AtEnd() {
		t.Fatal("zero cursor has unexpected state")
	}
}

func mustBuild(t *testing.T) FrameList {
	t.Helper()
	list, err := Build("s1", scenario())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return list
}
