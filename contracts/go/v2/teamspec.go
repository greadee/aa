package v2

import "fmt"

// TeamSpec is a durable grouping of related roles for organization, policy, and
// performance analysis. Teams group roles; crews group workers at runtime.
type TeamSpec struct {
	Envelope
	Title string   `json:"title"`
	Roles []string `json:"roles,omitempty"`
}

// Validate checks the team-spec contract.
func (t TeamSpec) Validate() error {
	if t.Kind != "team_spec" {
		return fmt.Errorf("kind: expected %q, got %q", "team_spec", t.Kind)
	}
	if err := t.Envelope.Validate(); err != nil {
		return err
	}
	if t.Title == "" {
		return fmt.Errorf("title: is required")
	}
	return nil
}
