package sync

import (
	"errors"
	"testing"
	"time"
)

func TestHashIsStable(t *testing.T) {
	got := Hash([]byte("aa-sync"))
	if len(got) != 64 {
		t.Fatalf("hash length = %d, want 64", len(got))
	}
	if Hash([]byte("aa-sync")) != got {
		t.Fatal("hash is not stable")
	}
	if Hash([]byte("aa-sync!")) == got {
		t.Fatal("different input produced the same hash")
	}
}

func TestKeyTrimsAndJoins(t *testing.T) {
	if got := Key(" distribute ", "wp1", " node-a "); got != "distribute:wp1:node-a" {
		t.Fatalf("Key = %q", got)
	}
	if got := TransferKey("artifact", "a1", NodeID("n1")); got != "artifact:a1:n1" {
		t.Fatalf("TransferKey = %q", got)
	}
}

func TestFixedClockAdvances(t *testing.T) {
	c := NewFixedClock()
	first := c.Now()
	second := c.Now()
	if !second.After(first) {
		t.Fatalf("clock did not advance: %v then %v", first, second)
	}
	if first != time.Unix(0, 0).UTC() {
		t.Fatalf("first = %v, want unix epoch", first)
	}
}

func TestInvalidWrapsSentinel(t *testing.T) {
	err := invalid("missing %s", "id")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err %v is not ErrInvalid", err)
	}
}
