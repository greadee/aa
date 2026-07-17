package v1

import "fmt"

// RouteRequest asks aa-sifter to route a prompt.
type RouteRequest struct {
	Envelope
	AttemptID string         `json:"attemptId,omitempty"`
	Prompt    string         `json:"prompt"`
	Context   string         `json:"context,omitempty"`
	Policy    string         `json:"policy,omitempty"`
	Budget    *Budget        `json:"budget,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Validate checks the route-request contract.
func (r RouteRequest) Validate() error {
	if r.Kind != "route_request" {
		return fmt.Errorf("kind: expected %q, got %q", "route_request", r.Kind)
	}
	if err := r.Envelope.Validate(); err != nil {
		return err
	}
	if r.Prompt == "" {
		return fmt.Errorf("prompt: is required")
	}
	if r.Policy != "" {
		if err := RequireEnum("policy", r.Policy,
			"local_only", "cloud_only", "local_first", "expert_first", "adaptive", "budget_constrained"); err != nil {
			return err
		}
	}
	return OptionalIdentifier("attemptId", r.AttemptID)
}

// RouteResponse is aa-sifter's routing decision.
type RouteResponse struct {
	Envelope
	RequestID             string   `json:"requestId"`
	Route                 string   `json:"route"`
	DecisionLevel         string   `json:"decisionLevel"`
	PlannerTier           string   `json:"plannerTier,omitempty"`
	ExecutorTier          string   `json:"executorTier,omitempty"`
	ReviewTier            string   `json:"reviewTier,omitempty"`
	RequiresHumanApproval bool     `json:"requiresHumanApproval"`
	BlockedReason         string   `json:"blockedReason,omitempty"`
	EstimatedCostUSD      *float64 `json:"estimatedCostUsd,omitempty"`
	Reasons               []string `json:"reasons,omitempty"`
}

// Validate checks the route-response contract.
func (r RouteResponse) Validate() error {
	if r.Kind != "route_response" {
		return fmt.Errorf("kind: expected %q, got %q", "route_response", r.Kind)
	}
	if err := r.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireIdentifier("requestId", r.RequestID); err != nil {
		return err
	}
	if err := RequireEnum("route", r.Route, "local", "hybrid", "cloud", "blocked"); err != nil {
		return err
	}
	if err := RequireEnum("decisionLevel", r.DecisionLevel, "routine", "significant", "major", "critical"); err != nil {
		return err
	}
	for field, tier := range map[string]string{
		"plannerTier": r.PlannerTier, "executorTier": r.ExecutorTier, "reviewTier": r.ReviewTier,
	} {
		if tier == "" {
			continue
		}
		if err := RequireEnum(field, tier, "local", "expert", "none"); err != nil {
			return err
		}
	}
	return nil
}
