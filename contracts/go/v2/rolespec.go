package v2

import "fmt"

// RoleSpec is a durable specification of an agent role's intended function. It
// is neither a running agent nor a model. See the role registry (aa-registry).
type RoleSpec struct {
	Envelope
	Title               string   `json:"title"`
	Capabilities        []string `json:"capabilities,omitempty"`
	Rules               []string `json:"rules,omitempty"`
	Tools               []string `json:"tools,omitempty"`
	Permissions         []string `json:"permissions,omitempty"`
	ContextExpectations []string `json:"contextExpectations,omitempty"`
	EvalCriteria        []string `json:"evalCriteria,omitempty"`
	Observability       []string `json:"observability,omitempty"`
	CostExpectation     string   `json:"costExpectation,omitempty"`
}

// Validate checks the role-spec contract.
func (r RoleSpec) Validate() error {
	if r.Kind != "role_spec" {
		return fmt.Errorf("kind: expected %q, got %q", "role_spec", r.Kind)
	}
	if err := r.Envelope.Validate(); err != nil {
		return err
	}
	if r.Title == "" {
		return fmt.Errorf("title: is required")
	}
	return nil
}
