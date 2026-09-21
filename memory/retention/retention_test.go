package retention

import (
	"reflect"
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

func step(seq int) v1.TraceStep {
	return v1.TraceStep{Sequence: &seq, Phase: v1.TraceTool, Outcome: v1.TraceSucceeded}
}

func trace(id, attempt string, revision int, seqs ...int) v1.Trace {
	steps := make([]v1.TraceStep, len(seqs))
	for i, s := range seqs {
		steps[i] = step(s)
	}
	rev := revision
	return v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: id, Revision: &rev},
		AttemptID: attempt,
		Steps:     steps,
	}
}

func sequences(steps []v1.TraceStep) []int {
	out := make([]int, len(steps))
	for i, s := range steps {
		out[i] = *s.Sequence
	}
	return out
}

func ids(traces []v1.Trace) []string {
	out := make([]string, len(traces))
	for i, t := range traces {
		out[i] = t.ID
	}
	return out
}

func TestTrimMaxStepsKeepsNewest(t *testing.T) {
	got := Trim([]v1.Trace{trace("trc_1", "att_1", 1, 1, 2, 3, 4)}, Policy{MaxStepsPerTrace: 2})
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if !reflect.DeepEqual(sequences(got[0].Steps), []int{3, 4}) {
		t.Fatalf("sequences = %v", sequences(got[0].Steps))
	}
}

func TestTrimMaxTracesPerAttemptKeepsHighestRevision(t *testing.T) {
	in := []v1.Trace{
		trace("trc_1", "att_1", 1, 0),
		trace("trc_2", "att_1", 3, 0),
		trace("trc_3", "att_1", 2, 0),
		trace("trc_4", "att_2", 1, 0),
	}
	got := Trim(in, Policy{MaxTracesPerAttempt: 2})
	if !reflect.DeepEqual(ids(got), []string{"trc_2", "trc_3", "trc_4"}) {
		t.Fatalf("ids = %v", ids(got))
	}
}

func TestTrimDoesNotMutateInput(t *testing.T) {
	original := []v1.Trace{
		trace("trc_1", "att_1", 1, 1, 2, 3),
		trace("trc_2", "att_1", 2, 1, 2, 3),
	}
	beforeIDs := ids(original)
	beforeSteps := sequences(original[0].Steps)
	_ = Trim(original, Policy{MaxStepsPerTrace: 1, MaxTracesPerAttempt: 1})
	if !reflect.DeepEqual(ids(original), beforeIDs) {
		t.Fatalf("input order mutated: %v", ids(original))
	}
	if !reflect.DeepEqual(sequences(original[0].Steps), beforeSteps) {
		t.Fatalf("input steps mutated: %v", sequences(original[0].Steps))
	}
}

func TestTrimNoLimitReturnsAllSortedByID(t *testing.T) {
	got := Trim([]v1.Trace{
		trace("trc_2", "att_1", 1, 0),
		trace("trc_1", "att_1", 1, 0),
	}, DefaultPolicy())
	if !reflect.DeepEqual(ids(got), []string{"trc_1", "trc_2"}) {
		t.Fatalf("ids = %v", ids(got))
	}
}

func TestTrimEmpty(t *testing.T) {
	if got := Trim(nil, Policy{MaxStepsPerTrace: 1, MaxTracesPerAttempt: 1}); len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}
