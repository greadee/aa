package routing

import (
	"errors"
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

func ptr[T any](v T) *T { return &v }

func TestPolicySelectsTier(t *testing.T) {
	cases := []struct {
		policy   Policy
		sig      Significance
		wantTier string
	}{
		{LocalOnly, Critical, "local"},
		{CloudOnly, Routine, "expert"},
		{LocalFirst, Routine, "local"},
		{LocalFirst, Major, "expert"},
		{ExpertFirst, Routine, "local"},
		{ExpertFirst, Significant, "expert"},
		{Adaptive, Critical, "expert"},
		{Adaptive, Significant, "local"},
	}
	for _, c := range cases {
		decision, err := Decide(Request{Policy: c.policy, Significance: c.sig, CloudAllowed: true})
		if err != nil {
			t.Fatal(err)
		}
		if decision.Tier != c.wantTier {
			t.Fatalf("%s/%s tier = %s, want %s", c.policy, c.sig, decision.Tier, c.wantTier)
		}
	}
}

func TestCloudDisabledBlocksExpertTier(t *testing.T) {
	decision, err := Decide(Request{Policy: CloudOnly, CloudAllowed: false})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Blocked || decision.Reason == "" {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestBudgetGateBlocksOverBudgetExpertTier(t *testing.T) {
	decision, err := Decide(Request{
		Policy:                CloudOnly,
		CloudAllowed:          true,
		EstimatedInputTokens:  1000,
		EstimatedOutputTokens: 1000,
		ExpertInputCostPer1K:  0.00014,
		ExpertOutputCostPer1K: 0.00028,
		Budget:                v2.Budget{MaxCostUSD: ptr(0.0001)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Blocked || decision.Reason != "estimated cost exceeds the budget" {
		t.Fatalf("decision = %+v", decision)
	}
	if decision.EstimatedCostUSD == 0 {
		t.Fatal("expected a cost estimate")
	}
}

func TestUnknownPolicyIsInvalid(t *testing.T) {
	if _, err := Decide(Request{Policy: "whatever"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if _, err := Decide(Request{Policy: LocalOnly, EstimatedInputTokens: -1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("negative tokens: %v", err)
	}
}
