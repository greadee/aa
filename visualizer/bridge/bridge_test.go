package bridge

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/obsv/transport"
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
	e.OccurredAt = "2026-01-01T00:00:00Z"
	e.Type = typ
	e.Aggregate = visualizer.EventAggregate{Kind: aggKind, ID: aggID}
	e.Actor = visualizer.Actor{Kind: "agent", ID: "worker_1"}
	return e
}

func proto(sourceType protocol.SourceType, source, action, workPackageID string) protocol.Event {
	return protocol.Event{
		Version:       protocol.Version,
		SessionID:     "s1",
		OccurredAt:    "2026-01-01T00:00:00Z",
		SourceType:    sourceType,
		Source:        source,
		Action:        action,
		Confidence:    protocol.ConfidenceObserved,
		Actor:         "worker_1",
		WorkPackageID: workPackageID,
	}
}

func scenario() []protocol.Event {
	return []protocol.Event{
		proto(protocol.SourceTool, "bash", "run", "wp1"),
		proto(protocol.SourceWorkDelta, "edit", "write", "wp1"),
		proto(protocol.SourceFile, "a.go", "read", ""),
		proto(protocol.SourceWorkDelta, "test", "complete", "wp1"),
	}
}

func newSource(t *testing.T) (*source.OBsv, transport.Server) {
	t.Helper()
	server := transport.NewLocal("test", nil)
	t.Cleanup(func() { _ = server.Close() })
	src, err := source.New(server)
	if err != nil {
		t.Fatalf("source.New: %v", err)
	}
	return src, server
}

func appendTo(t *testing.T, server transport.Server, sessionID string, events ...protocol.Event) {
	t.Helper()
	client, err := transport.Dial(server, sessionID)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	for i, e := range events {
		if _, err := client.Append(e); err != nil {
			t.Fatalf("Append[%d]: %v", i, err)
		}
	}
}

// consume follows a live obsv session through the bridge and returns the bridge
// plus the events an equivalent replay would project.
func consume(t *testing.T, events ...protocol.Event) (*Bridge, []visualizer.Event) {
	t.Helper()
	src, server := newSource(t)
	sub, err := src.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	b, err := New("s1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	appendTo(t, server, "s1", events...)
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := b.Consume(context.Background(), sub); err != nil {
		t.Fatalf("Consume: %v", err)
	}
	// The replay side reads the same session back through the same obsv
	// transport, so live and replay share one source and one mapping.
	replayed, err := src.Replay(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	return b, replayed
}

func TestConsumeLiveEqualsReplay(t *testing.T) {
	b, adapted := consume(t, scenario()...)
	got, err := b.FrameList()
	if err != nil {
		t.Fatalf("FrameList: %v", err)
	}
	want, err := replay.Build("s1", adapted)
	if err != nil {
		t.Fatalf("replay.Build: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("live frames differ from replay:\n%+v\n%+v", got, want)
	}
}

func TestConsumeAccumulatesEveryEvent(t *testing.T) {
	events := scenario()
	b, _ := consume(t, events...)
	if len(b.Events()) != len(events) {
		t.Fatalf("accumulated %d events, want %d", len(b.Events()), len(events))
	}
}

func TestCursorOverLiveFrames(t *testing.T) {
	b, _ := consume(t, scenario()...)
	c, err := b.Cursor()
	if err != nil {
		t.Fatalf("Cursor: %v", err)
	}
	f, err := c.SeekSequence(3)
	if err != nil {
		t.Fatalf("SeekSequence: %v", err)
	}
	if f.Sequence != 3 || c.Index() != 3 {
		t.Fatalf("seek = seq %d index %d", f.Sequence, c.Index())
	}
	if !c.AtEnd() {
		t.Fatal("cursor should be at end after seeking the last frame")
	}
	if f, err := c.Step(-1); err != nil || f.Sequence != 2 {
		t.Fatalf("step = %+v, %v", f, err)
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
	src, _ := newSource(t)
	sub, err := src.Subscribe(context.Background(), "s1")
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
	src, _ := newSource(t)
	sub, err := src.Subscribe(context.Background(), "s1")
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
