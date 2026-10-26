package v2

import "fmt"

// Strategy is a validated, reusable approach.
type Strategy struct {
	Envelope
	Title         string          `json:"title"`
	Lifecycle     MemoryLifecycle `json:"lifecycle"`
	Guidance      string          `json:"guidance"`
	WhenToUse     string          `json:"whenToUse,omitempty"`
	WhenNotToUse  string          `json:"whenNotToUse,omitempty"`
	Applicability *Applicability  `json:"applicability,omitempty"`
	Evidence      []Reference     `json:"evidence,omitempty"`
	Supersedes    string          `json:"supersedes,omitempty"`
}

// Validate checks the strategy contract.
func (s Strategy) Validate() error {
	if s.Kind != "strategy" {
		return fmt.Errorf("kind: expected %q, got %q", "strategy", s.Kind)
	}
	if err := s.Envelope.Validate(); err != nil {
		return err
	}
	if s.Title == "" {
		return fmt.Errorf("title: is required")
	}
	if err := s.Lifecycle.Validate(); err != nil {
		return err
	}
	if s.Guidance == "" {
		return fmt.Errorf("guidance: is required")
	}
	return OptionalIdentifier("supersedes", s.Supersedes)
}
