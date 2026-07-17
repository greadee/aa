package v1

import "fmt"

// Scope bounds what a work package may touch.
type Scope struct {
	Allowed   []string `json:"allowed,omitempty"`
	Inspect   []string `json:"inspect,omitempty"`
	Forbidden []string `json:"forbidden,omitempty"`
}

// WorkPackage is a bounded, verifiable unit of engineering work.
type WorkPackage struct {
	Envelope
	Title        string           `json:"title"`
	Description  string           `json:"description,omitempty"`
	State        WorkPackageState `json:"state"`
	Trade        string           `json:"trade,omitempty"`
	Role         string           `json:"role,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"`
	Inputs       []Reference      `json:"inputs,omitempty"`
	Deliverables []string         `json:"deliverables,omitempty"`
	Acceptance   []string         `json:"acceptance"`
	Scope        *Scope           `json:"scope,omitempty"`
	Budget       *Budget          `json:"budget,omitempty"`
	Risk         string           `json:"risk,omitempty"`
}

// Validate checks the work-package contract.
func (w WorkPackage) Validate() error {
	if w.Kind != "work_package" {
		return fmt.Errorf("kind: expected %q, got %q", "work_package", w.Kind)
	}
	if err := w.Envelope.Validate(); err != nil {
		return err
	}
	if w.Title == "" {
		return fmt.Errorf("title: is required")
	}
	if err := w.State.Validate(); err != nil {
		return err
	}
	if len(w.Acceptance) == 0 {
		return fmt.Errorf("acceptance: at least one criterion is required")
	}
	for i, d := range w.Dependencies {
		if err := RequireIdentifier(fmt.Sprintf("dependencies[%d]", i), d); err != nil {
			return err
		}
	}
	for i, in := range w.Inputs {
		if err := in.Validate(); err != nil {
			return fmt.Errorf("inputs[%d]: %w", i, err)
		}
	}
	if w.Risk != "" {
		if err := RequireEnum("risk", w.Risk, "low", "moderate", "high", "critical"); err != nil {
			return err
		}
	}
	return nil
}
