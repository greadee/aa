package control

import (
	"errors"
	"testing"
	"time"
)

func TestTransitionLifecycle(t *testing.T) {
	a := New("asg_1", "prj_1", "wp_1", "w_1")
	steps := []State{StateLeased, StatePreparing, StateRunning, StateCollecting, StateAwaitingGates, StateAccepted}
	for _, s := range steps {
		if err := a.Transition(s); err != nil {
			t.Fatalf("transition to %s: %v", s, err)
		}
	}
	if a.State != StateAccepted {
		t.Fatalf("state = %s", a.State)
	}
}

func TestTransitionFailsClosed(t *testing.T) {
	a := New("asg_1", "prj_1", "wp_1", "w_1")
	if err := a.Transition(StateRunning); err == nil {
		t.Fatal("expected invalid planned -> running")
	}
	if a.CanTransition(StateRunning) {
		t.Fatal("CanTransition should be false")
	}
	if err := a.Transition(StateLeased); err != nil {
		t.Fatal(err)
	}
	if err := a.Transition(State("bogus")); err == nil {
		t.Fatal("expected invalid state error")
	}
}

func TestLeaseExpiry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := New("asg_1", "prj_1", "wp_1", "w_1")
	if err := a.Transition(StateLeased); err != nil {
		t.Fatal(err)
	}
	a.LeaseFor("kernel", now, time.Minute)
	if a.Lease.Expired(now) {
		t.Fatal("lease should not be expired yet")
	}
	if a.Lease.Expired(now.Add(30 * time.Second)) {
		t.Fatal("lease should not be expired at 30s")
	}
	if !a.Lease.Expired(now.Add(2 * time.Minute)) {
		t.Fatal("lease should be expired")
	}
}

func TestExpireAndRetry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := New("asg_1", "prj_1", "wp_1", "w_1")
	_ = a.Transition(StateLeased)
	a.LeaseFor("kernel", now, time.Minute)
	if err := a.Expire(now); err != nil {
		t.Fatal(err)
	}
	if a.State != StateLeased {
		t.Fatalf("state = %s", a.State)
	}
	if err := a.Expire(now.Add(2 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if a.State != StateExpired {
		t.Fatalf("state = %s", a.State)
	}
	if err := a.Retry(); err != nil {
		t.Fatal(err)
	}
	if a.State != StatePlanned || a.Attempt != 1 {
		t.Fatalf("state=%s attempt=%d", a.State, a.Attempt)
	}
}

func TestRetryRejectedFromRunning(t *testing.T) {
	a := New("asg_1", "prj_1", "wp_1", "w_1")
	_ = a.Transition(StateLeased)
	_ = a.Transition(StatePreparing)
	_ = a.Transition(StateRunning)
	if err := a.Retry(); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v", err)
	}
}
