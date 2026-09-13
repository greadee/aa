package v1

import (
	"fmt"
	"regexp"
	"time"
)

var (
	identifierRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:@+-]*$`)
	versionRE    = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	hashRE       = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// RequireIdentifier returns an error unless value is a valid identifier.
func RequireIdentifier(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s: identifier is required", field)
	}
	if len(value) > 256 || !identifierRE.MatchString(value) {
		return fmt.Errorf("%s: invalid identifier %q", field, value)
	}
	return nil
}

// OptionalIdentifier validates value only when it is set.
func OptionalIdentifier(field, value string) error {
	if value == "" {
		return nil
	}
	return RequireIdentifier(field, value)
}

// RequireVersion returns an error unless value is MAJOR.MINOR.
func RequireVersion(field, value string) error {
	if !versionRE.MatchString(value) {
		return fmt.Errorf("%s: invalid contract version %q", field, value)
	}
	return nil
}

// RequireTimestamp returns an error unless value is RFC 3339.
func RequireTimestamp(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s: timestamp is required", field)
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s: invalid timestamp %q", field, value)
	}
	return nil
}

// OptionalTimestamp validates value only when it is set.
func OptionalTimestamp(field, value string) error {
	if value == "" {
		return nil
	}
	return RequireTimestamp(field, value)
}

// RequireEnum returns an error unless value is one of allowed.
func RequireEnum(field, value string, allowed ...string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s: invalid value %q", field, value)
}

// RequireHash returns an error unless value is a lowercase SHA-256 hex digest.
func RequireHash(field, value string) error {
	if !hashRE.MatchString(value) {
		return fmt.Errorf("%s: invalid sha256 hash", field)
	}
	return nil
}

// Validate checks the provenance contract.
func (p Provenance) Validate() error {
	if err := RequireIdentifier("provenance.source", p.Source); err != nil {
		return err
	}
	if err := RequireTimestamp("provenance.producedAt", p.ProducedAt); err != nil {
		return err
	}
	if p.Confidence != "" {
		if err := RequireEnum("provenance.confidence", p.Confidence, "exact", "correlated", "observed", "inferred"); err != nil {
			return err
		}
	}
	if p.ContentHash != "" {
		if err := RequireHash("provenance.contentHash", p.ContentHash); err != nil {
			return err
		}
	}
	if p.ProducedBy != nil {
		if err := p.ProducedBy.Validate(); err != nil {
			return err
		}
	}
	for i, e := range p.Evidence {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("provenance.evidence[%d]: %w", i, err)
		}
	}
	return nil
}

// Validate checks the reference contract.
func (r Reference) Validate() error {
	if err := RequireIdentifier("reference.kind", r.Kind); err != nil {
		return err
	}
	return RequireIdentifier("reference.id", r.ID)
}

// Validate checks the actor contract.
func (a Actor) Validate() error {
	if err := RequireEnum("actor.kind", a.Kind, "human", "agent", "system", "service"); err != nil {
		return err
	}
	return RequireIdentifier("actor.id", a.ID)
}

// Validate checks the envelope contract.
func (e Envelope) Validate() error {
	if err := RequireVersion("contractVersion", e.ContractVersion); err != nil {
		return err
	}
	if err := RequireIdentifier("kind", e.Kind); err != nil {
		return err
	}
	if err := RequireIdentifier("id", e.ID); err != nil {
		return err
	}
	if err := OptionalIdentifier("projectId", e.ProjectID); err != nil {
		return err
	}
	if err := OptionalTimestamp("createdAt", e.CreatedAt); err != nil {
		return err
	}
	if e.Provenance != nil {
		return e.Provenance.Validate()
	}
	return nil
}

// Validate checks enum membership for a project state.
func (s ProjectState) Validate() error {
	return RequireEnum("state", string(s),
		"CREATED", "DISCOVERY", "PLANNING", "EXECUTING", "VERIFYING",
		"COMPLETE", "BLOCKED", "FAILED", "CANCELLED", "PAUSED")
}

// Validate checks enum membership for a work-package state.
func (s WorkPackageState) Validate() error {
	return RequireEnum("state", string(s),
		"PROPOSED", "READY", "QUEUED", "RUNNING", "COMPLETE", "VERIFYING", "VERIFIED", "DEFICIENT")
}

// Validate checks enum membership for an assignment state.
func (s AssignmentState) Validate() error {
	return RequireEnum("state", string(s),
		"planned", "leased", "preparing", "running", "paused", "collecting",
		"awaiting_gates", "accepted", "failed", "canceled", "expired")
}

// Validate checks enum membership for a readiness state.
func (s ReadinessState) Validate() error {
	return RequireEnum("readiness", string(s), "blocked", "ready", "dispatched", "satisfied")
}

// Validate checks enum membership for an issue state.
func (s IssueState) Validate() error {
	return RequireEnum("status", string(s), "open", "in_progress", "review", "closed", "deferred", "cancelled")
}

// Validate checks enum membership for a memory lifecycle.
func (s MemoryLifecycle) Validate() error {
	return RequireEnum("lifecycle", string(s),
		"EPHEMERAL", "CANDIDATE", "VALIDATED", "ACTIVE", "SUPERSEDED", "ARCHIVED")
}
