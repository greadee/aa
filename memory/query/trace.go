package query

import (
	"encoding/json"
	"sort"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

// TraceSummary is a deterministic, derived view of one trace for learning and
// evaluation. It is computed from canonical records and never stored.
type TraceSummary struct {
	TraceID      string
	AttemptID    string
	ProjectID    string
	Steps        int
	FailedSteps  int
	BlockedSteps int
	DurationMS   int
	TokensIn     int
	TokensOut    int
	CostUSD      float64
	Truncated    bool
	Redacted     bool
	Outcome      string
}

// TraceSummaries returns a summary of every canonical trace, sorted by id.
func (q *Query) TraceSummaries() ([]TraceSummary, error) {
	entries := q.proj.List("trace")
	out := make([]TraceSummary, 0, len(entries))
	for _, e := range entries {
		var trace v1.Trace
		if err := json.Unmarshal(e.Data, &trace); err != nil {
			return nil, err
		}
		out = append(out, Summarize(trace))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TraceID < out[j].TraceID })
	return out, nil
}

// TraceSummariesByAttempt returns summaries for one attempt, sorted by id.
func (q *Query) TraceSummariesByAttempt(attemptID string) ([]TraceSummary, error) {
	if err := v1.RequireIdentifier("attemptId", attemptID); err != nil {
		return nil, err
	}
	all, err := q.TraceSummaries()
	if err != nil {
		return nil, err
	}
	var out []TraceSummary
	for _, s := range all {
		if s.AttemptID == attemptID {
			out = append(out, s)
		}
	}
	return out, nil
}

// Summarize derives a deterministic trace summary. The outcome is failed when
// any step failed, blocked when any step blocked, skipped when every step was
// skipped, and succeeded otherwise.
func Summarize(trace v1.Trace) TraceSummary {
	s := TraceSummary{
		TraceID:   trace.ID,
		AttemptID: trace.AttemptID,
		ProjectID: trace.ProjectID,
		Steps:     len(trace.Steps),
	}
	if trace.Truncated != nil {
		s.Truncated = *trace.Truncated
	}
	allSkipped := len(trace.Steps) > 0
	for _, step := range trace.Steps {
		switch step.Outcome {
		case v1.TraceFailed:
			s.FailedSteps++
		case v1.TraceBlocked:
			s.BlockedSteps++
		}
		if step.Outcome != v1.TraceSkipped {
			allSkipped = false
		}
		if step.DurationMS != nil {
			s.DurationMS += *step.DurationMS
		}
		if step.Tokens != nil {
			if step.Tokens.Input != nil {
				s.TokensIn += *step.Tokens.Input
			}
			if step.Tokens.Output != nil {
				s.TokensOut += *step.Tokens.Output
			}
		}
		if step.CostUSD != nil {
			s.CostUSD += *step.CostUSD
		}
		if step.Redacted != nil && *step.Redacted {
			s.Redacted = true
		}
	}
	switch {
	case s.FailedSteps > 0:
		s.Outcome = "failed"
	case s.BlockedSteps > 0:
		s.Outcome = "blocked"
	case allSkipped:
		s.Outcome = "skipped"
	default:
		s.Outcome = "succeeded"
	}
	return s
}
