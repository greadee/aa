package query

import (
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

func newTraceQuery(t *testing.T) (*Query, *repo.TraceRepository) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := projection.NewMem()
	return New(p, s), repo.NewTraceRepository(s, p)
}

func step(seq int, phase v1.TracePhase, outcome v1.TraceOutcome) v1.TraceStep {
	s := seq
	return v1.TraceStep{Sequence: &s, Phase: phase, Outcome: outcome}
}

func TestTraceSummariesDeriveDeterministically(t *testing.T) {
	q, traces := newTraceQuery(t)

	dur := 120
	tin, tout := 900, 300
	cost := 0.02
	redacted := true
	failed := v1.TraceStep{
		Sequence:   intp(1),
		Phase:      v1.TraceTest,
		Outcome:    v1.TraceFailed,
		DurationMS: &dur,
		Tokens:     &v1.TokenCounts{Input: &tin, Output: &tout},
		CostUSD:    &cost,
		Redacted:   &redacted,
	}
	trace := v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: "trc_1", ProjectID: "prj_1"},
		AttemptID: "att_1",
		Steps:     []v1.TraceStep{step(0, v1.TraceObserve, v1.TraceSucceeded), failed},
	}
	if _, _, err := traces.Ingest(trace); err != nil {
		t.Fatal(err)
	}
	if _, _, err := traces.Ingest(v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: "trc_0", ProjectID: "prj_1"},
		AttemptID: "att_1",
		Steps:     []v1.TraceStep{step(0, v1.TracePlan, v1.TraceSkipped)},
	}); err != nil {
		t.Fatal(err)
	}

	summaries, err := q.TraceSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 || summaries[0].TraceID != "trc_0" || summaries[1].TraceID != "trc_1" {
		t.Fatalf("summaries order = %+v", summaries)
	}
	got := summaries[1]
	if got.Steps != 2 || got.FailedSteps != 1 || got.Outcome != "failed" {
		t.Fatalf("summary = %+v", got)
	}
	if got.DurationMS != 120 || got.TokensIn != 900 || got.TokensOut != 300 || got.CostUSD != 0.02 {
		t.Fatalf("aggregates = %+v", got)
	}
	if !got.Redacted {
		t.Fatal("expected redacted summary")
	}
	if summaries[0].Outcome != "skipped" {
		t.Fatalf("skipped outcome = %q", summaries[0].Outcome)
	}

	// Recomputing yields the same result regardless of projection order.
	again, err := q.TraceSummaries()
	if err != nil {
		t.Fatal(err)
	}
	for i := range summaries {
		if summaries[i] != again[i] {
			t.Fatalf("non-deterministic summary at %d: %+v vs %+v", i, summaries[i], again[i])
		}
	}
}

func TestTraceSummariesByAttempt(t *testing.T) {
	q, traces := newTraceQuery(t)
	if _, _, err := traces.Ingest(v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: "trc_1"},
		AttemptID: "att_1",
		Steps:     []v1.TraceStep{step(0, v1.TraceObserve, v1.TraceSucceeded)},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := traces.Ingest(v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: "trc_2"},
		AttemptID: "att_2",
		Steps:     []v1.TraceStep{step(0, v1.TraceObserve, v1.TraceBlocked)},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := q.TraceSummariesByAttempt("att_1")
	if err != nil || len(got) != 1 || got[0].TraceID != "trc_1" {
		t.Fatalf("byAttempt = %+v err=%v", got, err)
	}
	if _, err := q.TraceSummariesByAttempt(""); err == nil {
		t.Fatal("expected invalid attemptId to be rejected")
	}
}

func intp(v int) *int { return &v }
