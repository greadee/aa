package source

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/obsv/transport"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/compat"
)

func obsEvent(sourceType protocol.SourceType, workPackageID string) protocol.Event {
	return protocol.Event{
		Version:       protocol.Version,
		SessionID:     "s1",
		OccurredAt:    "2026-01-01T00:00:00Z",
		SourceType:    sourceType,
		Source:        "bash",
		Action:        "run",
		Confidence:    protocol.ConfidenceObserved,
		Actor:         "worker_1",
		WorkPackageID: workPackageID,
		Metadata:      map[string]string{"command": "go test"},
	}
}

func newLocal(t *testing.T) (*OBsv, transport.Server) {
	t.Helper()
	server := transport.NewLocal("test", nil)
	t.Cleanup(func() { _ = server.Close() })
	src, err := New(server)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return src, server
}

// appendTo seeds the real in-process transport through its session client.
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

func TestReplayReadsOBsvAndMaps(t *testing.T) {
	src, server := newLocal(t)
	appendTo(t, server, "s1", obsEvent(protocol.SourceTool, "wp1"), obsEvent(protocol.SourceWorkDelta, ""))

	got, err := src.Replay(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("events = %d, want 2", len(got))
	}
	for i, e := range got {
		if e.Sequence != i {
			t.Fatalf("events[%d].Sequence = %d, want %d", i, e.Sequence, i)
		}
		if e.Type != v1.EventTelemetryRecorded {
			t.Fatalf("events[%d].Type = %q", i, e.Type)
		}
		if err := e.Validate(); err != nil {
			t.Fatalf("events[%d] does not validate: %v", i, err)
		}
	}
	if got[0].Aggregate.Kind != "work_package" || got[0].Aggregate.ID != "wp1" {
		t.Fatalf("aggregate = %+v", got[0].Aggregate)
	}
	if got[0].Payload["sessionId"] != "s1" || got[0].Payload["workPackageId"] != "wp1" {
		t.Fatalf("payload = %#v", got[0].Payload)
	}
}

func TestReplayMetadataFeedsCompatibilityProfile(t *testing.T) {
	src, server := newLocal(t)
	ev := obsEvent(protocol.SourceFile, "wp1")
	// Sanitize is the obsv durable allowlist: the profile keys survive, the
	// content-bearing key is dropped, and the allowlisted-but-unknown key stays
	// for compat to ignore.
	ev.Metadata = protocol.Sanitize(map[string]any{
		"secondary_paths": "pkg/a.go, pkg/b.go",
		"access_sequence": "read, edit, test",
		"tool":            "gopls",
		"prompt":          "secret",
	})
	appendTo(t, server, "s1", ev)

	events, err := src.Replay(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	m, err := compat.Normalize(events[0])
	if err != nil {
		t.Fatalf("compat.Normalize: %v", err)
	}
	if !reflect.DeepEqual(m.SecondaryPaths, []string{"pkg/a.go", "pkg/b.go"}) {
		t.Fatalf("secondary paths = %v", m.SecondaryPaths)
	}
	if !reflect.DeepEqual(m.AccessSequence, []string{"read", "edit", "test"}) {
		t.Fatalf("access sequence = %v", m.AccessSequence)
	}
}

func TestReplayEmptySession(t *testing.T) {
	src, server := newLocal(t)
	if _, err := transport.Dial(server, "s1"); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if _, err := src.Replay(context.Background(), "s1"); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestReplayRequiresSessionID(t *testing.T) {
	src, _ := newLocal(t)
	if _, err := src.Replay(context.Background(), ""); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestReplayUnavailableTransport(t *testing.T) {
	server := transport.NewLocal("test", nil)
	if err := server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	src, err := New(server)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := src.Replay(context.Background(), "s1"); !errors.Is(err, visualizer.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestNewRejectsNilTransport(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestSubscribeReceivesAppended(t *testing.T) {
	src, server := newLocal(t)
	sub, err := src.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()

	appendTo(t, server, "s1", obsEvent(protocol.SourceTool, "wp1"), obsEvent(protocol.SourceWorkDelta, "wp1"))
	for i := 0; i < 2; i++ {
		select {
		case got := <-sub.Events():
			if got.Sequence != i || got.Type != v1.EventTelemetryRecorded {
				t.Fatalf("event[%d] = %+v", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}
}

func TestSubscribeCloseStopsDelivery(t *testing.T) {
	src, server := newLocal(t)
	sub, err := src.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, open := <-sub.Events(); open {
		t.Fatal("channel still open after Close")
	}
	// Appending after close must not panic or block.
	appendTo(t, server, "s1", obsEvent(protocol.SourceTool, "wp1"))
}

func TestSubscribeUnavailableTransport(t *testing.T) {
	server := transport.NewLocal("test", nil)
	if err := server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	src, err := New(server)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := src.Subscribe(context.Background(), "s1"); !errors.Is(err, visualizer.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}
