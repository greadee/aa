// Package control owns the assignment state machine and leases. Transitions
// fail closed and leases expire deterministically.
package control

import (
	"errors"
	"fmt"
	"time"
)

// State is the lifecycle state of an assignment.
type State string

// Assignment states, aligned with the canonical state registry.
const (
	StatePlanned       State = "planned"
	StateLeased        State = "leased"
	StatePreparing     State = "preparing"
	StateRunning       State = "running"
	StatePaused        State = "paused"
	StateCollecting    State = "collecting"
	StateAwaitingGates State = "awaiting_gates"
	StateAccepted      State = "accepted"
	StateFailed        State = "failed"
	StateCanceled      State = "canceled"
	StateExpired       State = "expired"
)

// ErrInvalidTransition is returned for an illegal assignment transition.
var ErrInvalidTransition = errors.New("control: invalid assignment transition")

var transitions = map[State]map[State]bool{
	StatePlanned:       {StateLeased: true, StateCanceled: true},
	StateLeased:        {StatePreparing: true, StateExpired: true, StateCanceled: true},
	StatePreparing:     {StateRunning: true, StateFailed: true, StateExpired: true, StateCanceled: true},
	StateRunning:       {StatePaused: true, StateCollecting: true, StateFailed: true, StateCanceled: true},
	StatePaused:        {StateRunning: true, StateFailed: true, StateCanceled: true},
	StateCollecting:    {StateAwaitingGates: true, StateFailed: true},
	StateAwaitingGates: {StateAccepted: true, StateFailed: true},
	StateAccepted:      {},
	StateFailed:        {StatePlanned: true},
	StateCanceled:      {},
	StateExpired:       {StatePlanned: true},
}

// Lease grants an owner temporary authority over an assignment.
type Lease struct {
	Owner     string
	ExpiresAt time.Time
}

// Expired reports whether the lease has expired at now.
func (l Lease) Expired(now time.Time) bool {
	return !now.Before(l.ExpiresAt)
}

// Assignment binds a work package to a worker under a lease.
type Assignment struct {
	ID            string
	ProjectID     string
	WorkPackageID string
	WorkerID      string
	State         State
	Lease         Lease
	Attempt       int
}

// New returns a planned assignment.
func New(id, projectID, workPackageID, workerID string) *Assignment {
	return &Assignment{
		ID:            id,
		ProjectID:     projectID,
		WorkPackageID: workPackageID,
		WorkerID:      workerID,
		State:         StatePlanned,
	}
}

// Transition applies a validated state transition.
func (a *Assignment) Transition(to State) error {
	if a.State == to {
		return nil
	}
	if !transitions[a.State][to] {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, a.State, to)
	}
	a.State = to
	return nil
}

// CanTransition reports whether a transition is permitted.
func (a *Assignment) CanTransition(to State) bool {
	return a.State == to || transitions[a.State][to]
}

// LeaseFor sets a lease held by owner until now+ttl.
func (a *Assignment) LeaseFor(owner string, now time.Time, ttl time.Duration) {
	a.Lease = Lease{Owner: owner, ExpiresAt: now.Add(ttl)}
}

// Expire transitions a leased or preparing assignment whose lease has expired.
// It is a no-op otherwise.
func (a *Assignment) Expire(now time.Time) error {
	if a.State != StateLeased && a.State != StatePreparing {
		return nil
	}
	if !a.Lease.Expired(now) {
		return nil
	}
	return a.Transition(StateExpired)
}

// Retry returns an expired or failed assignment to planned and bumps the attempt.
func (a *Assignment) Retry() error {
	if a.State != StateExpired && a.State != StateFailed {
		return fmt.Errorf("%w: cannot retry from %s", ErrInvalidTransition, a.State)
	}
	if err := a.Transition(StatePlanned); err != nil {
		return err
	}
	a.Attempt++
	a.Lease = Lease{}
	return nil
}
