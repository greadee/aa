// Package routing owns the deterministic model-locality/tier policy and the
// budget gate.
//
// It is control-plane policy: the allocator decides where a unit runs and
// within what budget; the inference service only provides and executes models.
// Routing is deterministic — no model is called to make a routing decision.
package routing

import (
	"errors"
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

// ErrInvalid is returned for an invalid routing request.
var ErrInvalid = errors.New("routing: invalid request")

// Policy selects the locality preference.
type Policy string

// Policies.
const (
	LocalOnly         Policy = "local_only"
	CloudOnly         Policy = "cloud_only"
	LocalFirst        Policy = "local_first"
	ExpertFirst       Policy = "expert_first"
	Adaptive          Policy = "adaptive"
	BudgetConstrained Policy = "budget_constrained"
)

// Significance classifies how consequential a unit is.
type Significance string

// Significance levels.
const (
	Routine     Significance = "routine"
	Significant Significance = "significant"
	Major       Significance = "major"
	Critical    Significance = "critical"
)

// Request is a routing/budget request.
type Request struct {
	Policy       Policy
	Significance Significance
	// CloudAllowed reflects the active profile; cloud is blocked when false.
	CloudAllowed bool
	// Estimated tokens and the candidate expert model's per-1K costs, used for
	// the budget gate.
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	ExpertInputCostPer1K  float64
	ExpertOutputCostPer1K float64
	Budget                v2.Budget
}

// Decision is the routing/budget outcome.
type Decision struct {
	Locality         string
	Tier             string
	Blocked          bool
	Reason           string
	EstimatedCostUSD float64
	// Escalate marks a decision that hands work to the expert/cloud tier.
	Escalate bool
}

// Decide applies the policy and budget gate deterministically.
func Decide(req Request) (Decision, error) {
	if !knownPolicy(req.Policy) {
		return Decision{}, fmt.Errorf("%w: unknown policy %q", ErrInvalid, req.Policy)
	}
	if req.EstimatedInputTokens < 0 || req.EstimatedOutputTokens < 0 {
		return Decision{}, fmt.Errorf("%w: negative token estimate", ErrInvalid)
	}

	wantsExpert := preferExpert(req.Policy, req.Significance)
	cost := estimateCost(req)

	decision := Decision{EstimatedCostUSD: cost}
	if wantsExpert {
		decision.Locality, decision.Tier, decision.Escalate = "cloud", "expert", true
		decision.Reason = "policy selects the expert tier"
	} else {
		decision.Locality, decision.Tier = "local", "local"
		decision.Reason = "policy selects the local tier"
	}

	if decision.Escalate && !req.CloudAllowed {
		decision.Blocked = true
		decision.Reason = "cloud is disabled by the active profile"
		return decision, nil
	}
	if decision.Escalate && req.Budget.MaxCostUSD != nil && cost > *req.Budget.MaxCostUSD {
		decision.Blocked = true
		decision.Reason = "estimated cost exceeds the budget"
		return decision, nil
	}
	return decision, nil
}

// preferExpert resolves the policy and significance into a tier preference,
// honoring the budget-constrained policy's fallback to local.
func preferExpert(policy Policy, significance Significance) bool {
	switch policy {
	case LocalOnly:
		return false
	case CloudOnly:
		return true
	case ExpertFirst:
		return significance != Routine
	case LocalFirst:
		return significance == Major || significance == Critical
	case Adaptive, BudgetConstrained:
		return significance == Major || significance == Critical
	default:
		return false
	}
}

func estimateCost(req Request) float64 {
	in := float64(req.EstimatedInputTokens) / 1000 * req.ExpertInputCostPer1K
	out := float64(req.EstimatedOutputTokens) / 1000 * req.ExpertOutputCostPer1K
	return in + out
}

func knownPolicy(policy Policy) bool {
	switch policy {
	case LocalOnly, CloudOnly, LocalFirst, ExpertFirst, Adaptive, BudgetConstrained:
		return true
	default:
		return false
	}
}
