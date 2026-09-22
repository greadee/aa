package gate

import (
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
)

func attrResult(outcome joblearn.Outcome) attribution.Result {
	return attribution.Result{
		Attribution: joblearn.Attribution{
			WorkPackageID: "wp_1",
			AttemptID:     "att_1",
			Outcome:       outcome,
		},
	}
}

func TestCountOutcomes(t *testing.T) {
	results := []attribution.Result{
		attrResult(joblearn.OutcomeSucceeded),
		attrResult(joblearn.OutcomeSucceeded),
		attrResult(joblearn.OutcomeFailed),
		attrResult(joblearn.OutcomeBlocked),
	}

	counts := CountOutcomes(results)

	if counts.Total != 4 || counts.Succeeded != 2 || counts.Failed != 1 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
}

func TestEvidenceFromTraces(t *testing.T) {
	results := []attribution.Result{attrResult(joblearn.OutcomeSucceeded)}

	evidence := EvidenceFromTraces(results, 0.2, true, true, true)

	if evidence.Outcomes != 1 || evidence.Improvement != 0.2 || !evidence.Baseline || !evidence.Fallback || !evidence.NoGateBypass {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
}

func TestEvaluateTraceEvidenceEnablesWhenSatisfied(t *testing.T) {
	registry, err := NewRegistry(joblearn.DefaultGatePolicy())
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	results := make([]attribution.Result, 20)
	for i := range results {
		results[i] = attrResult(joblearn.OutcomeSucceeded)
	}

	decision, err := registry.Evaluate(joblearn.CapabilityRecommender, EvidenceFromTraces(results, 0.2, true, true, true))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.State != joblearn.CapabilityEnabled {
		t.Fatalf("expected enabled, got %+v", decision)
	}
	if !registry.Enabled(joblearn.CapabilityRecommender) {
		t.Fatal("expected the capability to be enabled")
	}
}

func TestEvaluateTraceEvidenceStaysDisabledWithoutFallback(t *testing.T) {
	registry, _ := NewRegistry(joblearn.DefaultGatePolicy())
	results := make([]attribution.Result, 20)
	for i := range results {
		results[i] = attrResult(joblearn.OutcomeSucceeded)
	}

	decision, err := registry.Evaluate(joblearn.CapabilityRecommender, EvidenceFromTraces(results, 0.2, true, false, true))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if decision.State != joblearn.CapabilityDisabled {
		t.Fatalf("expected disabled without a fallback, got %+v", decision)
	}
	if !hasReason(decision.Reasons, "fallback") {
		t.Fatalf("expected a fallback reason: %+v", decision.Reasons)
	}
}
