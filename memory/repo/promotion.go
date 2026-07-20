// Package repo provides typed repositories over the canonical store and the
// memory lifecycle promotion state machine.
package repo

import (
	"errors"
	"fmt"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

// ErrInvalidTransition is returned for an illegal lifecycle transition.
var ErrInvalidTransition = errors.New("memory: invalid lifecycle transition")

// transitions is the memory lifecycle state machine.
var transitions = map[v1.MemoryLifecycle]map[v1.MemoryLifecycle]bool{
	v1.MemoryEphemeral:  {v1.MemoryCandidate: true, v1.MemoryArchived: true},
	v1.MemoryCandidate:  {v1.MemoryValidated: true, v1.MemoryArchived: true},
	v1.MemoryValidated:  {v1.MemoryActive: true, v1.MemoryArchived: true},
	v1.MemoryActive:     {v1.MemorySuperseded: true, v1.MemoryArchived: true},
	v1.MemorySuperseded: {v1.MemoryArchived: true},
	v1.MemoryArchived:   {},
}

// CanTransition reports whether a lifecycle transition is permitted.
func CanTransition(from, to v1.MemoryLifecycle) bool {
	if from == to {
		return true
	}
	return transitions[from][to]
}

// Transition validates and returns the next lifecycle state.
func Transition(from, to v1.MemoryLifecycle) (v1.MemoryLifecycle, error) {
	if from == to {
		return to, nil
	}
	if !transitions[from][to] {
		return from, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	return to, nil
}
