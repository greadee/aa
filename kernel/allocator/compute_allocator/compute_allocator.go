// Package compute_allocator decides worker count, parallelism, budgets, and
// placement.
//
// It owns resource quantity and topology and must never change model identity.
// Parallelism is opt-in: it requires an explicit justification (latency,
// quality, coverage, or reliability). Idle agents are never a reason to spawn
// workers, and parallelism is bounded by the plan's concurrency ceiling.
package compute_allocator

import (
	"errors"
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

// ErrInvalid is returned for an invalid compute input.
var ErrInvalid = errors.New("compute_allocator: invalid input")

// Justification is why a work unit may run with more than one worker.
type Justification string

// Justifications.
const (
	JustifyNone        Justification = ""
	JustifyLatency     Justification = "latency"
	JustifyQuality     Justification = "quality"
	JustifyCoverage    Justification = "coverage"
	JustifyReliability Justification = "reliability"
)

var knownJustifications = map[Justification]bool{
	JustifyLatency: true, JustifyQuality: true, JustifyCoverage: true, JustifyReliability: true,
}

// Input is the compute allocation request.
type Input struct {
	Units            int
	IndependentUnits int
	Justification    Justification
	// MaxConcurrency is the ceiling the plan allows; 0 or 1 means serial.
	MaxConcurrency int
	Budget         v2.Budget
	Placement      []string
}

// Compute is the allocated resource quantity and topology.
type Compute struct {
	Workers     int
	Concurrency int
	Placement   []string
	Justified   bool
	Reason      string
}

// Plan allocates compute deterministically. Without a valid justification it
// stays serial (one worker); with one, workers are bounded by the number of
// independent units and the concurrency ceiling.
func Plan(in Input) (Compute, error) {
	if in.Units < 0 || in.IndependentUnits < 0 {
		return Compute{}, fmt.Errorf("%w: negative counts", ErrInvalid)
	}
	if in.Justification != JustifyNone && !knownJustifications[in.Justification] {
		return Compute{}, fmt.Errorf("%w: unknown justification %q", ErrInvalid, in.Justification)
	}
	compute := Compute{Workers: 1, Concurrency: 1, Placement: append([]string(nil), in.Placement...), Reason: "no parallelism justification"}

	ceiling := in.MaxConcurrency
	if ceiling < 1 {
		ceiling = 1
	}
	if in.Justification == JustifyNone {
		return compute, nil
	}
	if in.IndependentUnits <= 1 {
		compute.Reason = "single independent unit; parallelism not useful"
		return compute, nil
	}
	workers := in.IndependentUnits
	if workers > ceiling {
		workers = ceiling
	}
	if workers < 1 {
		workers = 1
	}
	compute.Workers = workers
	compute.Concurrency = workers
	compute.Justified = workers > 1
	compute.Reason = "parallelism justified by " + string(in.Justification)
	return compute, nil
}
