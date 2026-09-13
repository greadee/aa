package v1

import "fmt"

// WorkflowStep is one step in a workflow.
type WorkflowStep struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Role      string         `json:"role,omitempty"`
	Trade     string         `json:"trade,omitempty"`
	ToolID    string         `json:"toolId,omitempty"`
	DependsOn []string       `json:"dependsOn,omitempty"`
	Retries   *int           `json:"retries,omitempty"`
	Gates     []string       `json:"gates,omitempty"`
	OnFailure string         `json:"onFailure,omitempty"`
	Config    map[string]any `json:"config,omitempty"`
}

// Workflow is a declarative, resumable process definition.
type Workflow struct {
	Envelope
	Name        string         `json:"name,omitempty"`
	Version     string         `json:"version"`
	Description string         `json:"description,omitempty"`
	Budget      *Budget        `json:"budget,omitempty"`
	Steps       []WorkflowStep `json:"steps"`
}

// Validate checks the workflow contract.
func (w Workflow) Validate() error {
	if w.Kind != "workflow" {
		return fmt.Errorf("kind: expected %q, got %q", "workflow", w.Kind)
	}
	if err := w.Envelope.Validate(); err != nil {
		return err
	}
	if w.Version == "" {
		return fmt.Errorf("version: is required")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("steps: at least one step is required")
	}
	seen := make(map[string]bool, len(w.Steps))
	for i, s := range w.Steps {
		if err := RequireIdentifier(fmt.Sprintf("steps[%d].id", i), s.ID); err != nil {
			return err
		}
		if seen[s.ID] {
			return fmt.Errorf("steps[%d].id: duplicate id %q", i, s.ID)
		}
		seen[s.ID] = true
		if err := RequireEnum(fmt.Sprintf("steps[%d].kind", i), s.Kind,
			"task", "tool", "gate", "human_approval", "parallel", "conditional"); err != nil {
			return err
		}
		if s.OnFailure != "" {
			if err := RequireEnum(fmt.Sprintf("steps[%d].onFailure", i), s.OnFailure,
				"fail", "retry", "skip", "compensate"); err != nil {
				return err
			}
		}
	}
	return nil
}
