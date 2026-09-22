package attribution

import (
	"sort"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/features"
)

// TraceOutcome derives an attempt outcome from a bounded trace. A failed step
// dominates a blocked step; a truncated trace is partial because its evidence is
// incomplete; a trace with only skipped steps carries no outcome.
func TraceOutcome(trace v1.Trace) joblearn.Outcome {
	set := features.Extract(trace)
	switch {
	case set.Failed > 0:
		return joblearn.OutcomeFailed
	case set.Blocked > 0:
		return joblearn.OutcomeBlocked
	case set.Truncated:
		return joblearn.OutcomePartial
	case len(set.Steps) == 0 || set.Skipped == len(set.Steps):
		return joblearn.OutcomeUnknown
	default:
		return joblearn.OutcomeSucceeded
	}
}

// AttributeTrace links a trace to its work package, role, trade, and worker.
// The trace id is always recorded as evidence.
func AttributeTrace(trace v1.Trace, meta Meta) (joblearn.Attribution, error) {
	if trace.WorkPackageID == "" {
		return joblearn.Attribution{}, invalid("trace.workPackageId is required")
	}
	if trace.AttemptID == "" {
		return joblearn.Attribution{}, invalid("trace.attemptId is required")
	}
	return joblearn.Attribution{
		ProjectID:     meta.ProjectID,
		WorkPackageID: trace.WorkPackageID,
		AttemptID:     trace.AttemptID,
		AssignmentID:  trace.AssignmentID,
		Role:          meta.Role,
		Trade:         meta.Trade,
		Worker:        meta.Worker,
		Outcome:       TraceOutcome(trace),
		Sequence:      meta.Sequence,
		Evidence:      traceEvidence(trace, meta.References),
	}, nil
}

// ScoreTrace normalizes a trace's metadata-only features into a versioned score.
// Cost, duration, and retry components are lower-is-better; the overall score
// rewards their complement. No content is read.
func ScoreTrace(trace v1.Trace, meta Meta, limits Limits) (joblearn.Score, error) {
	if err := limits.Validate(); err != nil {
		return joblearn.Score{}, err
	}
	set := features.Extract(trace)
	s := joblearn.Score{
		Version:  joblearn.MetricVersion,
		Success:  successScore(TraceOutcome(trace)),
		Cost:     normalize(set.CostUSD, limits.MaxCostUSD),
		Duration: normalize(float64(set.DurationMS), float64(limits.MaxDurationMS)),
		Retries:  normalize(float64(meta.Retries), float64(limits.MaxRetries)),
		Quality:  qualityScore(meta.Gates),
	}
	s.Overall = clamp01(0.5*s.Success +
		0.2*s.Quality +
		0.1*(1-s.Cost) +
		0.1*(1-s.Duration) +
		0.1*(1-s.Retries))
	if err := s.Validate(); err != nil {
		return joblearn.Score{}, err
	}
	return s, nil
}

// DeriveTrace attributes and scores one trace.
func DeriveTrace(trace v1.Trace, meta Meta, limits Limits) (Result, error) {
	attr, err := AttributeTrace(trace, meta)
	if err != nil {
		return Result{}, err
	}
	score, err := ScoreTrace(trace, meta, limits)
	if err != nil {
		return Result{}, err
	}
	return Result{Attribution: attr, Score: score}, nil
}

// DeriveTraces attributes and scores every trace, preserving input order.
func DeriveTraces(traces []v1.Trace, metaFor func(v1.Trace) Meta, limits Limits) ([]Result, error) {
	out := make([]Result, 0, len(traces))
	for _, trace := range traces {
		result, err := DeriveTrace(trace, metaFor(trace), limits)
		if err != nil {
			return nil, err
		}
		out = append(out, result)
	}
	return out, nil
}

// traceEvidence returns the unique evidence references of a trace, sorted:
// the trace itself plus every step's references and the caller's references.
func traceEvidence(trace v1.Trace, extra []joblearn.Reference) []joblearn.Reference {
	refs := []joblearn.Reference{{Kind: "trace", ID: trace.ID}}
	for _, step := range trace.Steps {
		refs = append(refs, step.Evidence...)
	}
	refs = append(refs, extra...)
	seen := map[string]bool{}
	out := make([]joblearn.Reference, 0, len(refs))
	for _, ref := range refs {
		if ref.ID == "" {
			continue
		}
		key := ref.Kind + "\x00" + ref.ID + "\x00" + ref.Version
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}
