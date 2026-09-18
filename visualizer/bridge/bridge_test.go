package bridge

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/replay"
	"github.com/greadee/aa/visualizer/source"
)

func ev(id string, seq int, typ visualizer.EventType, aggKind, aggID string) visualizer.Event {
	e := visualizer.Event{}
	e.ContractVersion = "1.0"
	e.Kind = "event"
	e.ID = id
	e.ProjectID = "p1"
	e.Sequence = seq
	e.OccurredAt = fmt.Sprintf("2026-01-01T00:00:%02dZ", seq)
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

func consume(t *testing.T, events ...visualizer.Event) *Bridge {
	t.Helper()
	f := source.NewFake()
	sub, err := f.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	b, err := New("s1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := f.Append("s1", events...); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := b.Consume(context.Background(), sub); err != nil {
		t.Fatalf("Consume: %v", err)
	}
	return b
}

func TestConsumeLiveEqualsReplay(t *testing.T) {
	b := consume(t, scenario()...)
	got, err := b.FrameList()
	if err != nil {
		t.Fatalf("FrameList: %v", err)
	}
	want, err := replay.Build("s1", scenario())
	if err != nil {
		t.Fatalf("replay.Build: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("live frames differ from replay:\n%+v\n%+v", got, want)
	}
}

func TestConsumeLiveEqualsReplayOutOfOrder(t *testing.T) {
	events := scenario()
	b := consume(t, events[3], events[1], events[0], events[2])
	got, err := b.FrameList()
	if err != nil {
		t.Fatalf("FrameList: %v", err)
	}
	want, err := replay.Build("s1", events)
	if err != nil {
		t.Fatalf("replay.Build: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("live frames differ from replay:\n%+v\n%+v", got, want)
	}
	if len(b.Events()) != len(events) {
		t.Fatalf("accumulated %d events, want %d", len(b.Events()), len(events))
	}
}

func TestCursorOverLiveFrames(t *testing.T) {
	b := consume(t, scenario()...)
	c, err := b.Cursor()
	if err != nil {
		t.Fatalf("Cursor: %v", err)
	}
	f, err := c.SeekSequence(3)
	if err != nil {
		t.Fatalf("SeekSequence: %v", err)
	}
	if f.Sequence != 3 || c.Index() != 2 {
		t.Fatalf("seek = seq %d index %d", f.Sequence, c.Index())
	}
	if !c.AtEnd() {
		f, err = c.Step(1)
		if err != nil {
			t.Fatalf("Step: %v", err)
		}
		if f.Sequence != 4 {
			t.Fatalf("step = seq %d, want 4", f.Sequence)
		}
	}
}

func TestApplyValidatesEvents(t *testing.T) {
	b, err := New("s1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bad := ev("e1", 1, "WORK_PACKAGE_CREATED", "work_package", "wp1")
	bad.Type = "NOT_A_TYPE"
	if err := b.Apply(bad); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if len(b.Events()) != 0 {
		t.Fatal("invalid event was accumulated")
	}
}

func TestNewRejectsEmptySession(t *testing.T) {
	if _, err := New(""); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestConsumeEmptySubscription(t *testing.T) {
	f := source.NewFake()
	sub, err := f.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	b, err := New("s1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := b.Consume(context.Background(), sub); err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if _, err := b.FrameList(); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestConsumeRespectsContext(t *testing.T) {
	f := source.NewFake()
	sub, err := f.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()
	b, err := New("s1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.Consume(ctx, sub); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestConsumeRejectsNilSubscription(t *testing.T) {
	b, err := New("s1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := b.Consume(context.Background(), nil); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}
