package backtest

import (
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
)

func result(attempt, pkg, role string, outcome joblearn.Outcome) attribution.Result {
	return attribution.Result{
		Attribution: joblearn.Attribution{
			AttemptID:     attempt,
			WorkPackageID: pkg,
			Role:          role,
			Outcome:       outcome,
		},
		Score: joblearn.Score{Version: joblearn.MetricVersion},
	}
}

func TestTraceDatasetBuildsContentFreeSamples(t *testing.T) {
	dataset := TraceDataset([]attribution.Result{
		result("att_1", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_2", "wp_1", "engineer", joblearn.OutcomeFailed),
	})

	if len(dataset) != 2 {
		t.Fatalf("expected two samples, got %d", len(dataset))
	}
	if dataset[0].ID != "att_1" || dataset[0].Actual != "succeeded" {
		t.Fatalf("unexpected sample: %+v", dataset[0])
	}
	if dataset[0].Features["workPackageId"] != "wp_1" || dataset[0].Features["role"] != "engineer" {
		t.Fatalf("unexpected features: %+v", dataset[0].Features)
	}
}

func TestMajorityBaselinePrefersCommonLabel(t *testing.T) {
	dataset := TraceDataset([]attribution.Result{
		result("att_1", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_2", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_3", "wp_1", "engineer", joblearn.OutcomeFailed),
	})

	baseline := MajorityBaseline(dataset)

	if got := baseline.Predict(Features{}); got != "succeeded" {
		t.Fatalf("got %q want succeeded", got)
	}
}

func TestMajorityBaselineBreaksTiesLexically(t *testing.T) {
	dataset := TraceDataset([]attribution.Result{
		result("att_1", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_2", "wp_1", "engineer", joblearn.OutcomeFailed),
	})

	if got := MajorityBaseline(dataset).Predict(Features{}); got != "failed" {
		t.Fatalf("got %q want failed (lexically first)", got)
	}
}

func TestScopeBaselineUsesWorkPackageMajority(t *testing.T) {
	dataset := TraceDataset([]attribution.Result{
		result("att_1", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_2", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_3", "wp_2", "engineer", joblearn.OutcomeFailed),
		result("att_4", "wp_2", "engineer", joblearn.OutcomeFailed),
	})

	baseline := ScopeBaseline(dataset)

	if got := baseline.Predict(Features{"workPackageId": "wp_1"}); got != "succeeded" {
		t.Fatalf("wp_1: got %q want succeeded", got)
	}
	if got := baseline.Predict(Features{"workPackageId": "wp_2"}); got != "failed" {
		t.Fatalf("wp_2: got %q want failed", got)
	}
}

func TestRunOverTraceDatasetIsOrderIndependent(t *testing.T) {
	results := []attribution.Result{
		result("att_1", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_2", "wp_1", "engineer", joblearn.OutcomeSucceeded),
		result("att_3", "wp_2", "engineer", joblearn.OutcomeFailed),
		result("att_4", "wp_2", "engineer", joblearn.OutcomeFailed),
	}
	policy := PredictorFunc{N: "scope-rule", F: func(f Features) string {
		if f["workPackageId"] == "wp_1" {
			return "succeeded"
		}
		return "failed"
	}}
	shuffled := []attribution.Result{results[3], results[0], results[2], results[1]}

	firstDataset := TraceDataset(results)
	firstBaseline := MajorityBaseline(firstDataset)
	first, err := Run(firstDataset, policy, firstBaseline, DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	secondDataset := TraceDataset(shuffled)
	second, err := Run(secondDataset, policy, MajorityBaseline(secondDataset), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if first != second {
		t.Fatalf("order dependence:\n%+v\n%+v", first, second)
	}
	if !first.Pass {
		t.Fatalf("expected the policy to beat the majority baseline: %+v", first)
	}
}
