package transport

import (
	"testing"
	"time"

	"github.com/greadee/aa/obsv/protocol"
)

func event() protocol.Event {
	return protocol.Event{
		Version:    protocol.Version,
		SessionID:  "ses_1",
		SourceType: protocol.SourceTool,
		Confidence: protocol.ConfidenceObserved,
	}
}

func TestHelloIsAttachOrOwn(t *testing.T) {
	l := NewLocal("host", nil)
	defer l.Close()

	first, err := l.Hello("ses_1")
	if err != nil {
		t.Fatalf("Hello() = %v", err)
	}
	if first.Attached || first.Owner != "host" {
		t.Fatalf("first hello = %+v, want owner host and not attached", first)
	}

	second, err := l.Hello("ses_1")
	if err != nil {
		t.Fatalf("Hello() = %v", err)
	}
	if !second.Attached || second.Owner != "host" {
		t.Fatalf("second hello = %+v, want attached to host", second)
	}
}

func TestAppendReplaySubscribe(t *testing.T) {
	l := NewLocal("host", nil)
	defer l.Close()
	c, err := Dial(l, "ses_1")
	if err != nil {
		t.Fatalf("Dial() = %v", err)
	}
	if c.Attached() {
		t.Fatal("first dial should own the session")
	}

	sub, err := c.Subscribe(0)
	if err != nil {
		t.Fatalf("Subscribe() = %v", err)
	}
	defer sub.Close()

	for i := 0; i < 3; i++ {
		ack, err := c.Append(event())
		if err != nil {
			t.Fatalf("Append() = %v", err)
		}
		if ack.Sequence != i {
			t.Fatalf("ack.Sequence = %d, want %d", ack.Sequence, i)
		}
	}

	got, err := c.Replay(1)
	if err != nil {
		t.Fatalf("Replay() = %v", err)
	}
	if len(got) != 2 || got[0].Sequence != 1 {
		t.Fatalf("Replay(1) = %v, want sequences 1,2", got)
	}

	for i := 0; i < 3; i++ {
		select {
		case ev := <-sub.Events():
			if ev.Sequence != i {
				t.Fatalf("subscribed sequence = %d, want %d", ev.Sequence, i)
			}
		case <-time.After(time.Second):
			t.Fatalf("missing subscribed event %d", i)
		}
	}
}

func TestSecondDialAttaches(t *testing.T) {
	l := NewLocal("host", nil)
	defer l.Close()
	if _, err := Dial(l, "ses_1"); err != nil {
		t.Fatalf("Dial() = %v", err)
	}
	second, err := Dial(l, "ses_1")
	if err != nil {
		t.Fatalf("Dial() = %v", err)
	}
	if !second.Attached() {
		t.Fatal("second dial should attach to the existing owner")
	}
}

func TestAppendUnknownSessionFails(t *testing.T) {
	l := NewLocal("host", nil)
	defer l.Close()
	if _, err := l.Append("missing", event()); err == nil {
		t.Fatal("Append(unknown) = nil, want error")
	}
}

func TestHelloRequiresSession(t *testing.T) {
	l := NewLocal("host", nil)
	defer l.Close()
	if _, err := l.Hello(""); err == nil {
		t.Fatal("Hello(\"\") = nil, want error")
	}
}
