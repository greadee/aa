package v2

import "fmt"

// RoutineStage is one stage of a Routine.
type RoutineStage struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Teams []string `json:"teams,omitempty"`
	Roles []string `json:"roles,omitempty"`
	Gates []string `json:"gates,omitempty"`
}

// Routine is an organization-owned, reusable specification for recurring work.
// A project deploys a routine as a workflow (see Workflow).
type Routine struct {
	Envelope
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Description   string         `json:"description,omitempty"`
	Teams         []string       `json:"teams,omitempty"`
	Roles         []string       `json:"roles,omitempty"`
	Stages        []RoutineStage `json:"stages"`
	Inputs        []string       `json:"inputs,omitempty"`
	Outputs       []string       `json:"outputs,omitempty"`
	Observability []string       `json:"observability,omitempty"`
}

// Validate checks the routine contract.
func (r Routine) Validate() error {
	if r.Kind != "routine" {
		return fmt.Errorf("kind: expected %q, got %q", "routine", r.Kind)
	}
	if err := r.Envelope.Validate(); err != nil {
		return err
	}
	if r.Name == "" {
		return fmt.Errorf("name: is required")
	}
	if r.Version == "" {
		return fmt.Errorf("version: is required")
	}
	if len(r.Stages) == 0 {
		return fmt.Errorf("stages: at least one is required")
	}
	seen := make(map[string]bool, len(r.Stages))
	for i, s := range r.Stages {
		if err := RequireIdentifier(fmt.Sprintf("stages[%d].id", i), s.ID); err != nil {
			return err
		}
		if seen[s.ID] {
			return fmt.Errorf("stages[%d].id: duplicate id %q", i, s.ID)
		}
		seen[s.ID] = true
	}
	return nil
}
