// Package route exposes learned routing as a non-authoritative hint.
//
// A hint always carries a deterministic fallback and is never authoritative: it
// is withheld when the learned_routing capability is disabled by its evidence
// gate, when a human approval is pending, or when no earlier worker beats the
// scope baseline by enough. The caller still evaluates every gate and execution
// contract; the hint only informs them.
package route

import (
	"fmt"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/gate"
)

// Request is a routing request. Fallback is the deterministic choice the caller
// would make without learning; it is required and always returned.
type Request struct {
	WorkPackageID string `json:"workPackageId,omitempty"`
	Role          string `json:"role,omitempty"`
	Trade         string `json:"trade,omitempty"`
	// Fallback is the deterministic option that remains available when the
	// learned hint is withheld.
	Fallback string `json:"fallback"`
	// RequiresApproval marks a request with a pending human gate. The learned
	// hint is withheld so it cannot preempt the approval.
	RequiresApproval bool `json:"requiresApproval,omitempty"`
	// ContractID is the execution contract that still applies. It is never
	// bypassed by a hint.
	ContractID string `json:"contractId,omitempty"`
}

// Hint is a routing recommendation. It is a suggestion, never a decision.
type Hint struct {
	// Option is the learned choice; it is empty unless Learned is true.
	Option string `json:"option,omitempty"`
	// Fallback is the deterministic choice, always set.
	Fallback string `json:"fallback"`
	// Learned reports whether Option came from attributed history.
	Learned bool `json:"learned"`
	// Authoritative is always false: a hint never bypasses a gate or contract.
	Authoritative bool     `json:"authoritative"`
	Reasons       []string `json:"reasons,omitempty"`
}

// Choice returns the learned option when present, otherwise the fallback.
func (h Hint) Choice() string {
	if h.Learned && h.Option != "" {
		return h.Option
	}
	return h.Fallback
}

// Policy bounds when a learned routing hint is offered.
type Policy struct {
	// MinWorkerOutcomes is the minimum outcomes a worker needs to be suggested.
	MinWorkerOutcomes int `json:"minWorkerOutcomes"`
	// MinImprovement is the margin over the scope baseline a worker must beat.
	MinImprovement float64 `json:"minImprovement"`
}

// DefaultPolicy returns conservative routing thresholds.
func DefaultPolicy() Policy {
	return Policy{MinWorkerOutcomes: 3, MinImprovement: 0.1}
}

// Validate checks the policy's bounds.
func (p Policy) Validate() error {
	if p.MinWorkerOutcomes < 0 {
		return invalid("minWorkerOutcomes must be >= 0")
	}
	if p.MinImprovement < 0 || p.MinImprovement > 1 {
		return invalid("minImprovement %v is outside [0,1]", p.MinImprovement)
	}
	return nil
}

// Router produces routing hints under an evidence gate.
type Router struct {
	gates  *gate.Registry
	policy Policy
}

// New returns a router over a gate registry.
func New(gates *gate.Registry, policy Policy) (*Router, error) {
	if gates == nil {
		return nil, invalid("gate registry is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Router{gates: gates, policy: policy}, nil
}

// Route returns a hint for a request from attributed history. It always fills
// Fallback; Option is set only when the learned_routing capability is enabled
// and a worker beats the scope baseline by the policy margin.
func (r *Router) Route(req Request, history []attribution.Result) (Hint, error) {
	if req.Fallback == "" {
		return Hint{}, invalid("fallback is required")
	}
	hint := Hint{Fallback: req.Fallback, Authoritative: false}

	if req.RequiresApproval {
		hint.Reasons = append(hint.Reasons, "human approval pending; learned hint withheld")
		return hint, nil
	}
	if !r.gates.Enabled(joblearn.CapabilityLearnedRouting) {
		hint.Reasons = append(hint.Reasons, "learned_routing capability is disabled")
		return hint, nil
	}
	if req.ContractID != "" {
		hint.Reasons = append(hint.Reasons, fmt.Sprintf("execution contract %s still applies; hint is advisory", req.ContractID))
	}

	option, reason, ok := r.learnedOption(req, history)
	hint.Reasons = append(hint.Reasons, reason)
	if !ok {
		return hint, nil
	}
	hint.Option = option
	hint.Learned = true
	return hint, nil
}

// learnedOption returns the worker that beats the scope baseline, if any.
func (r *Router) learnedOption(req Request, history []attribution.Result) (string, string, bool) {
	if req.Trade == "" && req.Role == "" {
		return "", "no routing scope", false
	}
	scoped := filter(history, req)
	if len(scoped) == 0 {
		return "", "no attributed history for scope", false
	}
	baseline := attribution.Summarize(scoped).MeanOverall

	byWorker := map[string][]attribution.Result{}
	for _, res := range scoped {
		if res.Attribution.Worker == "" {
			continue
		}
		byWorker[res.Attribution.Worker] = append(byWorker[res.Attribution.Worker], res)
	}
	bestID := ""
	bestMean := 0.0
	bestOutcomes := 0
	for id, group := range byWorker {
		s := attribution.Summarize(group)
		if bestID == "" || s.MeanOverall > bestMean || (s.MeanOverall == bestMean && id < bestID) {
			bestID, bestMean, bestOutcomes = id, s.MeanOverall, s.Outcomes
		}
	}
	if bestID == "" {
		return "", "no worker history for scope", false
	}
	if bestOutcomes < r.policy.MinWorkerOutcomes {
		return "", fmt.Sprintf("worker %s has %d outcomes < %d", bestID, bestOutcomes, r.policy.MinWorkerOutcomes), false
	}
	margin := bestMean - baseline
	if margin < r.policy.MinImprovement {
		return "", fmt.Sprintf("best worker beats the baseline by %.3f < %.3f", margin, r.policy.MinImprovement), false
	}
	return bestID, fmt.Sprintf("worker %s beat the baseline by %.3f", bestID, margin), true
}

func filter(history []attribution.Result, req Request) []attribution.Result {
	var out []attribution.Result
	for _, res := range history {
		a := res.Attribution
		if req.Trade != "" && a.Trade != req.Trade {
			continue
		}
		if req.Role != "" && a.Role != req.Role {
			continue
		}
		out = append(out, res)
	}
	return out
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
