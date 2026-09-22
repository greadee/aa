// Package retention trims a projected trace view deterministically.
//
// Retention is applied to derived trace views, never to aa-memory's canonical
// records: the store remains authoritative and rebuild-equivalent. Trimming
// keeps the highest-revision traces per attempt and the most recent steps per
// trace, and never mutates its input.
package retention

import (
	"sort"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

// Policy bounds a projected trace view.
type Policy struct {
	// MaxStepsPerTrace keeps at most this many most-recent steps (highest
	// sequence). Zero or less means no step limit.
	MaxStepsPerTrace int
	// MaxTracesPerAttempt keeps at most this many traces per attempt, by
	// highest revision (ties broken by id). Zero or less means no trace limit.
	MaxTracesPerAttempt int
}

// DefaultPolicy retains every trace and step.
func DefaultPolicy() Policy { return Policy{} }

// Trim returns a new, id-sorted trace slice with the policy applied. The input
// and the traces it points at are never modified.
func Trim(traces []v1.Trace, p Policy) []v1.Trace {
	out := make([]v1.Trace, 0, len(traces))
	for _, t := range traces {
		t.Steps = trimSteps(t.Steps, p.MaxStepsPerTrace)
		out = append(out, t)
	}
	if p.MaxTracesPerAttempt > 0 {
		out = limitPerAttempt(out, p.MaxTracesPerAttempt)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func trimSteps(steps []v1.TraceStep, limit int) []v1.TraceStep {
	if len(steps) == 0 {
		return steps
	}
	kept := make([]v1.TraceStep, len(steps))
	copy(kept, steps)
	sort.SliceStable(kept, func(i, j int) bool { return sequenceOf(kept[i]) < sequenceOf(kept[j]) })
	if limit > 0 && len(kept) > limit {
		kept = kept[len(kept)-limit:]
	}
	return kept
}

func limitPerAttempt(traces []v1.Trace, limit int) []v1.Trace {
	byAttempt := make(map[string][]v1.Trace)
	order := make([]string, 0)
	for _, t := range traces {
		if _, ok := byAttempt[t.AttemptID]; !ok {
			order = append(order, t.AttemptID)
		}
		byAttempt[t.AttemptID] = append(byAttempt[t.AttemptID], t)
	}
	sort.Strings(order)
	var out []v1.Trace
	for _, attempt := range order {
		group := byAttempt[attempt]
		sort.Slice(group, func(i, j int) bool {
			ri, rj := revisionOf(group[i]), revisionOf(group[j])
			if ri != rj {
				return ri > rj
			}
			return group[i].ID < group[j].ID
		})
		if len(group) > limit {
			group = group[:limit]
		}
		out = append(out, group...)
	}
	return out
}

func sequenceOf(step v1.TraceStep) int {
	if step.Sequence == nil {
		return 0
	}
	return *step.Sequence
}

func revisionOf(trace v1.Trace) int {
	if trace.Revision == nil {
		return 0
	}
	return *trace.Revision
}
