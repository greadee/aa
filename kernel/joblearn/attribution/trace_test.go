package attribution

import (
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
)

func intPtr(v int) *int { return &v }

func traceWith(steps ...v1.TraceStep) v1.Trace {
	return v1.Trace{
		Envelope:      v1.Envelope{ContractVersion: v1.Version, Kind: "trace", ID: "trc_1"},
		AttemptID:     "att_1",
		WorkPackageID: "wp_1",
		Steps:         steps,
	}
}

func step(seq int, outcome v1.TraceOutcome) v1.TraceStep {
	return v1.TraceStep{Sequence: intPtr(seq), Phase: v1.TraceTest, Outcome: outcome}
}

func TestTraceOutcomePrecedence(t *testing.T) {
	cases := []struct {
		name  string
		trace v1.Trace
		want  joblearn.Outcome
	}{
		{"succeeded", traceWith(step(0, v1.TraceSucceeded)), joblearn.OutcomeSucceeded},
		{"failed dominates blocked", traceWith(step(0, v1.TraceBlocked), step(1, v1.TraceFailed)), joblearn.OutcomeFailed},
		{"blocked", traceWith(step(0, v1.TraceSucceeded), step(1, v1.TraceBlocked)), joblearn.OutcomeBlocked},
		{"all skipped is unknown", traceWith(step(0, v1.TraceSkipped)), joblearn.OutcomeUnknown},
	}
	for _, tc := range cases {
		if got := TraceOutcome(tc.trace); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestTraceOutcomeTruncatedIsPartial(t *testing.T) {
	trace := traceWith(step(0, v1.TraceSucceeded))
	truncated := true
	trace.Truncated = &truncated

	if got := TraceOutcome(trace); got != joblearn.OutcomePartial {
		t.Fatalf("got %q want partial", got)
	}
}

func TestAttributeTraceRecordsIdentityAndEvidence(t *testing.T) {
	trace := traceWith(step(0, v1.TraceSucceeded))
	meta := Meta{ProjectID: "proj_1", Role: "engineer", Trade: "backend", Worker: "w_1", Sequence: 7}

	attr, err := AttributeTrace(trace, meta)
	if err != nil {
		t.Fatalf("AttributeTrace: %v", err)
	}
	if attr.WorkPackageID != "wp_1" || attr.AttemptID != "att_1" || attr.Sequence != 7 {
		t.Fatalf("unexpected attribution: %+v", attr)
	}
	if attr.Outcome != joblearn.OutcomeSucceeded || attr.Role != "engineer" {
		t.Fatalf("unexpected outcome or role: %+v", attr)
	}
	found := false
	for _, ref := range attr.Evidence {
		if ref.Kind == "trace" && ref.ID == "trc_1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("trace evidence missing: %+v", attr.Evidence)
	}
}

func TestAttributeTraceRequiresScope(t *testing.T) {
	trace := traceWith(step(0, v1.TraceSucceeded))
	trace.WorkPackageID = ""

	if _, err := AttributeTrace(trace, Meta{}); err == nil {
		t.Fatal("expected an error for a missing work package id")
	}
}

func TestScoreTraceIsBoundedAndDeterministic(t *testing.T) {
	trace := traceWith(step(0, v1.TraceFailed))
	trace.Steps[0].DurationMS = intPtr(500)
	cost := 0.5
	trace.Steps[0].CostUSD = &cost

	first, err := ScoreTrace(trace, Meta{Retries: 1}, DefaultLimits())
	if err != nil {
		t.Fatalf("ScoreTrace: %v", err)
	}
	second, _ := ScoreTrace(trace, Meta{Retries: 1}, DefaultLimits())

	if first != second {
		t.Fatalf("score is not deterministic: %+v vs %+v", first, second)
	}
	if first.Success != 0 || first.Version != joblearn.MetricVersion {
		t.Fatalf("unexpected score: %+v", first)
	}
}

func TestDeriveTracesPreservesOrder(t *testing.T) {
	first := traceWith(step(0, v1.TraceSucceeded))
	second := traceWith(step(0, v1.TraceFailed))
	second.ID = "trc_2"
	second.AttemptID = "att_2"

	results, err := DeriveTraces([]v1.Trace{first, second}, func(v1.Trace) Meta { return Meta{} }, DefaultLimits())
	if err != nil {
		t.Fatalf("DeriveTraces: %v", err)
	}
	if len(results) != 2 || results[0].Attribution.AttemptID != "att_1" || results[1].Attribution.AttemptID != "att_2" {
		t.Fatalf("order not preserved: %+v", results)
	}
}
