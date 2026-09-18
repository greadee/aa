package source

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	visualizer "github.com/greadee/aa/visualizer"
)

func ev(id string, seq int, aggKind, aggID string) visualizer.Event {
	e := visualizer.Event{}
	e.ContractVersion = "1.0"
	e.Kind = "event"
	e.ID = id
	e.Sequence = seq
	e.OccurredAt = "2026-01-01T00:00:00Z"
	e.Type = "WORK_PACKAGE_CREATED"
	e.Aggregate = visualizer.EventAggregate{Kind: aggKind, ID: aggID}
	e.Actor = visualizer.Actor{Kind: "agent", ID: "worker_1"}
	return e
}

func TestReplaySortsBySequence(t *testing.T) {
	f := NewFake()
	if err := f.Append("s1", ev("e2", 2, "work_package", "wp1"), ev("e1", 1, "work_package", "wp1"), ev("e3", 3, "work_package", "wp1")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := f.Replay(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	var seqs []int
	for _, e := range got {
		seqs = append(seqs, e.Sequence)
	}
	if !reflect.DeepEqual(seqs, []int{1, 2, 3}) {
		t.Fatalf("sequences = %v", seqs)
	}
}

func TestReplayUnknownSession(t *testing.T) {
	f := NewFake()
	_, err := f.Replay(context.Background(), "ghost")
	if !errors.Is(err, visualizer.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestReplayEmptySession(t *testing.T) {
	f := NewFake()
	if err := f.Append("s1"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	_, err := f.Replay(context.Background(), "s1")
	if !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestAppendValidates(t *testing.T) {
	f := NewFake()
	bad := ev("e1", 1, "work_package", "wp1")
	bad.Type = "NOT_A_TYPE"
	if err := f.Append("s1", bad); !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if _, err := f.Replay(context.Background(), "s1"); !errors.Is(err, visualizer.ErrNotFound) {
		t.Fatalf("invalid event was stored: %v", err)
	}
}

func TestReplayReturnsCopy(t *testing.T) {
	f := NewFake()
	_ = f.Append("s1", ev("e1", 1, "work_package", "wp1"))
	first, _ := f.Replay(context.Background(), "s1")
	first[0].ID = "mutated"
	second, _ := f.Replay(context.Background(), "s1")
	if second[0].ID != "e1" {
		t.Fatalf("Replay aliased stored events: %q", second[0].ID)
	}
}

func TestSubscribeReceivesAppended(t *testing.T) {
	f := NewFake()
	sub, err := f.Subscribe(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()
	if err := f.Append("s1", ev("e1", 1, "work_package", "wp1"), ev("e2", 2, "work_package", "wp1")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	for _, want := range []string{"e1", "e2"} {
		select {
		case got := <-sub.Events():
			if got.ID != want {
				t.Fatalf("event = %q, want %q", got.ID, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
}

func TestSubscribeCloseStopsDelivery(t *testing.T) {
	f := NewFake()
	sub, _ := f.Subscribe(context.Background(), "s1")
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, open := <-sub.Events(); open {
		t.Fatal("channel still open after Close")
	}
	// Appending after close must not panic or block.
	if err := f.Append("s1", ev("e1", 1, "work_package", "wp1")); err != nil {
		t.Fatalf("Append after close: %v", err)
	}
}
