package policies

import (
	"fmt"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

// Kind classifies a policy definition. Applicability to a workflow, project,
// or execution is decided elsewhere; this is only the definition catalogue.
type Kind string

// Policy-definition kinds.
const (
	KindApproval Kind = "approval"
	KindBudget   Kind = "budget"
	KindRouting  Kind = "routing"
	KindGate     Kind = "gate"
)

// Kinds returns the supported policy kinds in stable order.
func Kinds() []Kind {
	return []Kind{KindApproval, KindBudget, KindRouting, KindGate}
}

// ValidKind reports whether kind is a supported policy definition kind.
func ValidKind(kind Kind) bool {
	switch kind {
	case KindApproval, KindBudget, KindRouting, KindGate:
		return true
	default:
		return false
	}
}

// PolicySpec is a durable policy definition. The envelope kind is
// "policy_definition"; PolicyKind classifies what kind of policy it defines.
type PolicySpec struct {
	v2.Envelope
	Title       string `json:"title"`
	PolicyKind  Kind   `json:"policyKind"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Validate checks the policy-definition contract.
func (p PolicySpec) Validate() error {
	if p.Kind != "policy_definition" {
		return fmt.Errorf("kind: expected %q, got %q", "policy_definition", p.Kind)
	}
	if err := p.Envelope.Validate(); err != nil {
		return err
	}
	if p.Title == "" {
		return fmt.Errorf("title: is required")
	}
	if !ValidKind(p.PolicyKind) {
		return fmt.Errorf("policyKind: unknown policy kind %q", p.PolicyKind)
	}
	if p.Version == "" {
		return fmt.Errorf("version: is required")
	}
	return nil
}
