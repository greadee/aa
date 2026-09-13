package telemetry

import "testing"

func TestDeriveSuccess(t *testing.T) {
	got := Derive(Record{AttemptID: "att_1", WorkPackageID: "wp_1", Outcome: "succeeded", Calls: 2, Tokens: 100}, nil)
	if len(got) != 1 || got[0].Kind != "success" || got[0].Level != "task" {
		t.Fatalf("candidates = %+v", got)
	}
}

func TestDerivePitfallSorted(t *testing.T) {
	got := Derive(Record{AttemptID: "att_1", WorkPackageID: "wp_1", Outcome: "failed"}, []string{"zeta", "alpha"})
	if len(got) != 1 || got[0].Kind != "pitfall" {
		t.Fatalf("candidates = %+v", got)
	}
	if got[0].Content != "Tests failed: alpha, zeta" {
		t.Fatalf("content = %q", got[0].Content)
	}
}

func TestDeriveCost(t *testing.T) {
	got := Derive(Record{AttemptID: "att_1", WorkPackageID: "wp_1", Outcome: "succeeded", CostUSD: 2.5}, nil)
	if len(got) != 2 || got[0].Kind != "cost" || got[1].Kind != "success" {
		t.Fatalf("candidates = %+v", got)
	}
}

func TestSummarize(t *testing.T) {
	s := Summarize([]Record{
		{Outcome: "succeeded", Calls: 1, Tokens: 10, CostUSD: 0.1},
		{Outcome: "failed", Calls: 2, Tokens: 20, CostUSD: 0.2},
	})
	if s.Attempts != 2 || s.Successes != 1 || s.Failures != 1 || s.TotalCalls != 3 || s.TotalTokens != 30 {
		t.Fatalf("summary = %+v", s)
	}
}
