package v2

import "fmt"

// WorkUnit is one unit of work in a WorkPlan.
type WorkUnit struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Outputs    []string `json:"outputs,omitempty"`
	Completion []string `json:"completion,omitempty"`
	DependsOn  []string `json:"dependsOn,omitempty"`
}

// WorkPlan is the Planner's output: what work needs to happen and how it
// depends, independent of role, model, or compute selection.
type WorkPlan struct {
	Envelope
	SourceWorkflow *Reference `json:"sourceWorkflow,omitempty"`
	Units          []WorkUnit `json:"units"`
	Constraints    []string   `json:"constraints,omitempty"`
}

// Validate checks the work-plan contract.
func (w WorkPlan) Validate() error {
	if w.Kind != "work_plan" {
		return fmt.Errorf("kind: expected %q, got %q", "work_plan", w.Kind)
	}
	if err := w.Envelope.Validate(); err != nil {
		return err
	}
	if len(w.Units) == 0 {
		return fmt.Errorf("units: at least one is required")
	}
	seen := make(map[string]bool, len(w.Units))
	for i, u := range w.Units {
		if err := RequireIdentifier(fmt.Sprintf("units[%d].id", i), u.ID); err != nil {
			return err
		}
		if seen[u.ID] {
			return fmt.Errorf("units[%d].id: duplicate id %q", i, u.ID)
		}
		seen[u.ID] = true
	}
	return nil
}
