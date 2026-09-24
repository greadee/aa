// Package features extracts step-level, content-free features from a trace.
//
// The feature vector is a deterministic function of metadata only: phase,
// operation, outcome, error class, duration, tokens, and cost. It never
// includes prompts, payloads, hashes, or any recoverable input. Redaction and
// truncation are surfaced rather than hidden, so a bounded trace cannot subtly
// distort a comparison.
package features

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

// Step is the content-free feature vector of one trace step.
type Step struct {
	Sequence   int     `json:"sequence"`
	Phase      string  `json:"phase"`
	Operation  string  `json:"operation,omitempty"`
	Outcome    string  `json:"outcome"`
	ErrorClass string  `json:"errorClass,omitempty"`
	DurationMS int     `json:"durationMs"`
	TokensIn   int     `json:"tokensIn"`
	TokensOut  int     `json:"tokensOut"`
	CostUSD    float64 `json:"costUsd"`
	Redacted   bool    `json:"redacted,omitempty"`
}

// Set is the feature vector of one whole trace plus its deterministic totals.
type Set struct {
	TraceID    string  `json:"traceId"`
	AttemptID  string  `json:"attemptId"`
	Steps      []Step  `json:"steps"`
	Truncated  bool    `json:"truncated,omitempty"`
	Redacted   bool    `json:"redacted,omitempty"`
	Succeeded  int     `json:"succeeded"`
	Failed     int     `json:"failed"`
	Blocked    int     `json:"blocked"`
	Skipped    int     `json:"skipped"`
	DurationMS int     `json:"durationMs"`
	TokensIn   int     `json:"tokensIn"`
	TokensOut  int     `json:"tokensOut"`
	CostUSD    float64 `json:"costUsd"`
}

// Extract builds the content-free feature set of one trace. Steps keep their
// sequence order; a nil duration, token count, or cost contributes zero.
func Extract(trace v1.Trace) Set {
	set := Set{
		TraceID:   trace.ID,
		AttemptID: trace.AttemptID,
		Steps:     make([]Step, 0, len(trace.Steps)),
		Truncated: trace.Truncated != nil && *trace.Truncated,
	}
	for _, step := range trace.Steps {
		feature := Step{
			Sequence:   sequence(step),
			Phase:      string(step.Phase),
			Operation:  step.Operation,
			Outcome:    string(step.Outcome),
			ErrorClass: step.ErrorClass,
		}
		if step.DurationMS != nil {
			feature.DurationMS = *step.DurationMS
		}
		if step.Tokens != nil {
			if step.Tokens.Input != nil {
				feature.TokensIn = *step.Tokens.Input
			}
			if step.Tokens.Output != nil {
				feature.TokensOut = *step.Tokens.Output
			}
		}
		if step.CostUSD != nil {
			feature.CostUSD = *step.CostUSD
		}
		if step.Redacted != nil {
			feature.Redacted = *step.Redacted
		}
		switch step.Outcome {
		case v1.TraceSucceeded:
			set.Succeeded++
		case v1.TraceFailed:
			set.Failed++
		case v1.TraceBlocked:
			set.Blocked++
		case v1.TraceSkipped:
			set.Skipped++
		}
		set.DurationMS += feature.DurationMS
		set.TokensIn += feature.TokensIn
		set.TokensOut += feature.TokensOut
		set.CostUSD += feature.CostUSD
		set.Redacted = set.Redacted || feature.Redacted
		set.Steps = append(set.Steps, feature)
	}
	return set
}

// ExtractAll extracts every trace's features, sorted by trace id.
func ExtractAll(traces []v1.Trace) []Set {
	out := make([]Set, 0, len(traces))
	for _, trace := range traces {
		out = append(out, Extract(trace))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TraceID < out[j].TraceID })
	return out
}

// Key returns a stable, content-free key for the set, so two identical traces
// always hash to the same key regardless of pointer identity.
func (s Set) Key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s|%s|%t|%t|%d|%d|%d|%d|%d|%d|%d|%g",
		s.TraceID, s.AttemptID, s.Truncated, s.Redacted,
		s.Succeeded, s.Failed, s.Blocked, s.Skipped,
		s.DurationMS, s.TokensIn, s.TokensOut, s.CostUSD)
	for _, step := range s.Steps {
		fmt.Fprintf(&b, "|%d,%s,%s,%s,%s,%d,%d,%d,%g,%t",
			step.Sequence, step.Phase, step.Operation, step.Outcome, step.ErrorClass,
			step.DurationMS, step.TokensIn, step.TokensOut, step.CostUSD, step.Redacted)
	}
	return b.String()
}

func sequence(step v1.TraceStep) int {
	if step.Sequence == nil {
		return 0
	}
	return *step.Sequence
}
