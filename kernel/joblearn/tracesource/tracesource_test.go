package tracesource

import (
	"errors"
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
)

func intPtr(v int) *int { return &v }

func trace(id, attempt string, steps int) v1.Trace {
	built := make([]v1.TraceStep, 0, steps)
	for i := 0; i < steps; i++ {
		built = append(built, v1.TraceStep{
			Sequence: intPtr(i),
			Phase:    v1.TracePlan,
			Outcome:  v1.TraceSucceeded,
		})
	}
	return v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: v1.Version, Kind: "trace", ID: id},
		AttemptID: attempt,
		Steps:     built,
	}
}

func TestStaticSortsDeterministically(t *testing.T) {
	src := NewStatic([]v1.Trace{
		trace("trc_c", "att_c", 1),
		trace("trc_a", "att_a", 1),
		trace("trc_b", "att_b", 1),
	})

	got, err := src.Traces()
	if err != nil {
		t.Fatalf("Traces: %v", err)
	}
	if len(got) != 3 || got[0].ID != "trc_a" || got[1].ID != "trc_b" || got[2].ID != "trc_c" {
		t.Fatalf("expected sorted ids, got %v", ids(got))
	}
}

func TestStaticDoesNotMutateOrAliasInput(t *testing.T) {
	input := []v1.Trace{trace("trc_b", "att_b", 1), trace("trc_a", "att_a", 1)}
	src := NewStatic(input)

	got, _ := src.Traces()
	got[0].ID = "mutated"

	again, _ := src.Traces()
	if again[0].ID != "trc_a" {
		t.Fatalf("source aliased its output: %v", ids(again))
	}
	if input[0].ID != "trc_b" {
		t.Fatalf("source mutated its input: %v", ids(input))
	}
}

func TestBoundLimitsTraces(t *testing.T) {
	traces := []v1.Trace{trace("trc_a", "att_a", 1), trace("trc_b", "att_b", 1)}

	got, err := Bound(traces, Bounds{MaxTraces: 1})
	if err != nil {
		t.Fatalf("Bound: %v", err)
	}
	if len(got) != 1 || got[0].ID != "trc_a" {
		t.Fatalf("expected one trace, got %v", ids(got))
	}
}

func TestBoundLimitsStepsByWholeTraces(t *testing.T) {
	traces := []v1.Trace{trace("trc_a", "att_a", 3), trace("trc_b", "att_b", 3)}

	got, err := Bound(traces, Bounds{MaxSteps: 4})
	if err != nil {
		t.Fatalf("Bound: %v", err)
	}
	if len(got) != 1 || got[0].ID != "trc_a" {
		t.Fatalf("expected the first whole trace only, got %v", ids(got))
	}
}

func TestBoundUnlimitedKeepsAll(t *testing.T) {
	traces := []v1.Trace{trace("trc_a", "att_a", 1), trace("trc_b", "att_b", 1)}

	got, err := Bound(traces, DefaultBounds())
	if err != nil {
		t.Fatalf("Bound: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected all traces, got %v", ids(got))
	}
}

func TestBoundRejectsNegative(t *testing.T) {
	if _, err := Bound(nil, Bounds{MaxTraces: -1}); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestLoadValidatesEveryTrace(t *testing.T) {
	bad := trace("trc_a", "", 1)
	if _, err := Load(NewFake(bad), DefaultBounds()); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a missing attemptId, got %v", err)
	}
}

func TestLoadRequiresSource(t *testing.T) {
	if _, err := Load(nil, DefaultBounds()); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestLoadPropagatesSourceError(t *testing.T) {
	sentinel := errors.New("store down")
	src := &Fake{Err: sentinel}

	if _, err := Load(src, DefaultBounds()); !errors.Is(err, sentinel) {
		t.Fatalf("expected source error, got %v", err)
	}
}

func TestFakeRecordsReads(t *testing.T) {
	src := NewFake(trace("trc_a", "att_a", 1))

	if _, err := Load(src, DefaultBounds()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if src.Reads != 1 {
		t.Fatalf("expected one read, got %d", src.Reads)
	}
}

func ids(traces []v1.Trace) []string {
	out := make([]string, len(traces))
	for i, tr := range traces {
		out[i] = tr.ID
	}
	return out
}
