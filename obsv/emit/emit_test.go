package emit

import (
	"errors"
	"testing"

	"github.com/greadee/aa/obsv/protocol"
)

type capture struct{ events []protocol.Event }

func (c *capture) Append(ev protocol.Event) error {
	c.events = append(c.events, ev)
	return nil
}

type failer struct{ err error }

func (f failer) Append(protocol.Event) error { return f.err }

type panicker struct{}

func (panicker) Append(protocol.Event) error { panic("boom") }

func event() protocol.Event {
	return protocol.Event{
		Version:    protocol.Version,
		SessionID:  "ses_1",
		SourceType: protocol.SourceTool,
		Confidence: protocol.ConfidenceObserved,
	}
}

func TestEmitAcceptsAndSanitizes(t *testing.T) {
	c := &capture{}
	e := New(c)
	ev := event()
	ev.Metadata = map[string]string{"tool": "gopls", "prompt": "leak"}
	if !e.Emit(ev) {
		t.Fatal("Emit() = false, want true")
	}
	if len(c.events) != 1 {
		t.Fatalf("wrote %d events, want 1", len(c.events))
	}
	if _, ok := c.events[0].Metadata["prompt"]; ok {
		t.Fatal("prompt metadata survived sanitization")
	}
	if c.events[0].Metadata["tool"] != "gopls" {
		t.Fatal("allowlisted metadata was dropped")
	}
	if got := e.Stats(); got.Accepted != 1 {
		t.Fatalf("Stats().Accepted = %d, want 1", got.Accepted)
	}
}

func TestEmitDropsInvalidEvent(t *testing.T) {
	e := New(&capture{})
	ev := event()
	ev.Confidence = "certain"
	if e.Emit(ev) {
		t.Fatal("Emit() = true, want false")
	}
	if got := e.Stats().Dropped; got != 1 {
		t.Fatalf("Stats().Dropped = %d, want 1", got)
	}
}

func TestEmitIsFailureOpenOnError(t *testing.T) {
	e := New(failer{err: errors.New("disk gone")})
	if e.Emit(event()) {
		t.Fatal("Emit() = true, want false")
	}
	if got := e.Stats().Failed; got != 1 {
		t.Fatalf("Stats().Failed = %d, want 1", got)
	}
}

func TestEmitDoesNotLetPanicsEscape(t *testing.T) {
	e := New(panicker{})
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic escaped Emit: %v", r)
			}
		}()
		if e.Emit(event()) {
			t.Fatal("Emit() = true, want false")
		}
	}()
	if got := e.Stats().Failed; got != 1 {
		t.Fatalf("Stats().Failed = %d, want 1", got)
	}
}

func TestNewNilTargetDiscards(t *testing.T) {
	e := New(nil)
	if !e.Emit(event()) {
		t.Fatal("Emit() = false, want true")
	}
}

func TestBufferIsBounded(t *testing.T) {
	b := NewBuffer(2)
	if err := b.Append(event()); err != nil {
		t.Fatalf("Append() = %v", err)
	}
	if err := b.Append(event()); err != nil {
		t.Fatalf("Append() = %v", err)
	}
	if err := b.Append(event()); !errors.Is(err, ErrFull) {
		t.Fatalf("Append() = %v, want ErrFull", err)
	}
	if b.Len() != 2 || b.Dropped() != 1 {
		t.Fatalf("Len=%d Dropped=%d, want 2/1", b.Len(), b.Dropped())
	}
}

func TestEmitterDropsOnFullTarget(t *testing.T) {
	e := New(NewBuffer(1))
	if !e.Emit(event()) {
		t.Fatal("first Emit() = false, want true")
	}
	if e.Emit(event()) {
		t.Fatal("second Emit() = true, want false")
	}
	if got := e.Stats().Dropped; got != 1 {
		t.Fatalf("Stats().Dropped = %d, want 1", got)
	}
}

func TestBufferDrainPreservesOrder(t *testing.T) {
	b := NewBuffer(0)
	for i := 0; i < 3; i++ {
		ev := event()
		ev.Sequence = i
		if err := b.Append(ev); err != nil {
			t.Fatalf("Append() = %v", err)
		}
	}
	c := &capture{}
	n, err := b.Drain(c)
	if err != nil || n != 3 {
		t.Fatalf("Drain() = %d, %v, want 3, nil", n, err)
	}
	if b.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", b.Len())
	}
	for i, ev := range c.events {
		if ev.Sequence != i {
			t.Fatalf("event %d sequence = %d, want %d", i, ev.Sequence, i)
		}
	}
}

func TestBufferDrainStopsOnError(t *testing.T) {
	b := NewBuffer(0)
	for i := 0; i < 3; i++ {
		_ = b.Append(event())
	}
	n, err := b.Drain(failer{err: errors.New("no")})
	if err == nil || n != 0 {
		t.Fatalf("Drain() = %d, %v, want 0, error", n, err)
	}
	if b.Len() != 3 {
		t.Fatalf("Len() = %d, want 3 (nothing written)", b.Len())
	}
}
