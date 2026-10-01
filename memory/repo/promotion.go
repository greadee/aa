// Package repo provides typed repositories over the canonical store and the
// memory lifecycle promotion state machine.
package repo

import (
	"errors"
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

// ErrInvalidTransition is returned for an illegal lifecycle transition.
var ErrInvalidTransition = errors.New("memory: invalid lifecycle transition")

// transitions is the memory lifecycle state machine.
var transitions = map[v2.MemoryLifecycle]map[v2.MemoryLifecycle]bool{
	v2.MemoryEphemeral:  {v2.MemoryCandidate: true, v2.MemoryArchived: true},
	v2.MemoryCandidate:  {v2.MemoryValidated: true, v2.MemoryArchived: true},
	v2.MemoryValidated:  {v2.MemoryActive: true, v2.MemoryArchived: true},
	v2.MemoryActive:     {v2.MemorySuperseded: true, v2.MemoryArchived: true},
	v2.MemorySuperseded: {v2.MemoryArchived: true},
	v2.MemoryArchived:   {},
}

// CanTransition reports whether a lifecycle transition is permitted.
func CanTransition(from, to v2.MemoryLifecycle) bool {
	if from == to {
		return true
	}
	return transitions[from][to]
}

// Transition validates and returns the next lifecycle state.
func Transition(from, to v2.MemoryLifecycle) (v2.MemoryLifecycle, error) {
	if from == to {
		return to, nil
	}
	if !transitions[from][to] {
		return from, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	return to, nil
}
