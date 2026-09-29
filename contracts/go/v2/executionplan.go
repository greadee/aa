package v2

import "fmt"

// PlanAssignment is the allocation of a role (and optionally a model) to a work
// unit, with a worker count.
type PlanAssignment struct {
	Role    string `json:"role"`
	Model   string `json:"model,omitempty"`
	Workers int    `json:"workers,omitempty"`
}

// ExecutionPlan is the allocator's output: a WorkPlan plus the allocated roles,
// models, worker counts, budgets, concurrency, and placement. It is consumed by
// the scheduler and runtime.
type ExecutionPlan struct {
	Envelope
	WorkPlan    *Reference       `json:"workPlan,omitempty"`
	Assignments []PlanAssignment `json:"assignments,omitempty"`
	Budget      *Budget          `json:"budget,omitempty"`
	Concurrency int              `json:"concurrency,omitempty"`
	Placement   []string         `json:"placement,omitempty"`
	Constraints []string         `json:"constraints,omitempty"`
}

// Validate checks the execution-plan contract.
func (p ExecutionPlan) Validate() error {
	if p.Kind != "execution_plan" {
		return fmt.Errorf("kind: expected %q, got %q", "execution_plan", p.Kind)
	}
	if err := p.Envelope.Validate(); err != nil {
		return err
	}
	for i, a := range p.Assignments {
		if err := OptionalIdentifier(fmt.Sprintf("assignments[%d].role", i), a.Role); err != nil {
			return err
		}
		if a.Workers < 0 {
			return fmt.Errorf("assignments[%d].workers: must be >= 0", i)
		}
	}
	if p.Concurrency < 0 {
		return fmt.Errorf("concurrency: must be >= 0")
	}
	return nil
}
