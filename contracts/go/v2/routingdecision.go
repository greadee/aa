package v2

import "fmt"

// RoutingDecision is the allocator's routing/budget outcome, passed to the
// inference execution path so inference executes the decision instead of
// re-deciding locality, tier, or budget.
type RoutingDecision struct {
	Locality         string  `json:"locality"`
	Tier             string  `json:"tier"`
	Blocked          bool    `json:"blocked,omitempty"`
	Reason           string  `json:"reason,omitempty"`
	EstimatedCostUSD float64 `json:"estimatedCostUsd,omitempty"`
	DecisionLevel    string  `json:"decisionLevel,omitempty"`
	// RequiresHumanApproval carries the control plane's approval state.
	RequiresHumanApproval bool `json:"requiresHumanApproval,omitempty"`
}

// Validate checks the routing-decision contract.
func (d RoutingDecision) Validate() error {
	if err := RequireEnum("locality", d.Locality, "local", "cloud"); err != nil {
		return err
	}
	if err := RequireEnum("tier", d.Tier, "local", "expert"); err != nil {
		return err
	}
	if d.Locality == "local" && d.Tier != "local" {
		return fmt.Errorf("tier: local locality requires the local tier")
	}
	if d.Locality == "cloud" && d.Tier != "expert" {
		return fmt.Errorf("tier: cloud locality requires the expert tier")
	}
	if d.EstimatedCostUSD < 0 {
		return fmt.Errorf("estimatedCostUsd: must be >= 0")
	}
	if d.DecisionLevel != "" {
		if err := RequireEnum("decisionLevel", d.DecisionLevel, "routine", "significant", "major", "critical"); err != nil {
			return err
		}
	}
	return nil
}
