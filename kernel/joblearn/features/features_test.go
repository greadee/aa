package features

import (
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

func intPtr(v int) *int         { return &v }
func f64Ptr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool      { return &v }

func step(seq int, phase v1.TracePhase, outcome v1.TraceOutcome) v1.TraceStep {
	return v1.TraceStep{Sequence: intPtr(seq), Phase: phase, Outcome: outcome}
}

func baseTrace() v1.Trace {
	return v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: v1.Version, Kind: "trace", ID: "trc_1"},
		AttemptID: "att_1",
		Steps: []v1.TraceStep{
			{
				Sequence:   intPtr(0),
				Phase:      v1.TraceModel,
				Operation:  "generate",
				Outcome:    v1.TraceSucceeded,
				DurationMS: intPtr(120),
				Tokens:     &v1.TokenCounts{Input: intPtr(100), Output: intPtr(50)},
				CostUSD:    f64Ptr(0.02),
			},
			{
				Sequence:   intPtr(1),
				Phase:      v1.TraceTest,
				Outcome:    v1.TraceFailed,
				ErrorClass: "assertion",
				DurationMS: intPtr(30),
			},
		},
	}
}

func TestExtractCountsAndTotals(t *testing.T) {
	set := Extract(baseTrace())

	if set.TraceID != "trc_1" || set.AttemptID != "att_1" {
		t.Fatalf("unexpected identity: %+v", set)
	}
	if set.Succeeded != 1 || set.Failed != 1 || set.Blocked != 0 || set.Skipped != 0 {
		t.Fatalf("unexpected counts: %+v", set)
	}
	if set.DurationMS != 150 || set.TokensIn != 100 || set.TokensOut != 50 || set.CostUSD != 0.02 {
		t.Fatalf("unexpected totals: %+v", set)
	}
	if len(set.Steps) != 2 || set.Steps[1].ErrorClass != "assertion" {
		t.Fatalf("unexpected steps: %+v", set.Steps)
	}
}

func TestExtractNilPointersAreZero(t *testing.T) {
	trace := v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: v1.Version, Kind: "trace", ID: "trc_1"},
		AttemptID: "att_1",
		Steps:     []v1.TraceStep{step(0, v1.TracePlan, v1.TraceSucceeded)},
	}

	set := Extract(trace)

	if set.Steps[0].DurationMS != 0 || set.Steps[0].TokensIn != 0 || set.Steps[0].CostUSD != 0 {
		t.Fatalf("expected zeroed optional fields: %+v", set.Steps[0])
	}
}

func TestExtractSurfacesRedactionAndTruncation(t *testing.T) {
	trace := baseTrace()
	trace.Truncated = boolPtr(true)
	trace.Steps[0].Redacted = boolPtr(true)

	set := Extract(trace)

	if !set.Truncated || !set.Redacted || !set.Steps[0].Redacted {
		t.Fatalf("expected redaction and truncation surfaced: %+v", set)
	}
}

func TestExtractIgnoresContentBearingFields(t *testing.T) {
	original := baseTrace()
	withHash := baseTrace()
	withHash.Steps[0].InputHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	withHash.Steps[0].OutputHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	withHash.Steps[0].Target = &v1.Reference{Kind: "file", ID: "secret.txt"}

	if Extract(original).Key() != Extract(withHash).Key() {
		t.Fatalf("feature key changed with content-bearing fields")
	}
}

func TestKeyIsStableAcrossPointerIdentity(t *testing.T) {
	first := Extract(baseTrace())
	second := Extract(baseTrace())

	if first.Key() != second.Key() {
		t.Fatalf("identical traces produced different keys:\n%s\n%s", first.Key(), second.Key())
	}
}

func TestExtractAllIsSorted(t *testing.T) {
	makeTrace := func(id string) v1.Trace {
		trace := baseTrace()
		trace.ID = id
		return trace
	}

	sets := ExtractAll([]v1.Trace{makeTrace("trc_c"), makeTrace("trc_a"), makeTrace("trc_b")})

	if len(sets) != 3 || sets[0].TraceID != "trc_a" || sets[2].TraceID != "trc_c" {
		t.Fatalf("expected sorted sets, got %+v", sets)
	}
}
