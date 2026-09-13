package v1

import "fmt"

// ProjectRecord is the canonical project manifest.
type ProjectRecord struct {
	Envelope
	Name          string       `json:"name"`
	State         ProjectState `json:"state"`
	LayoutVersion string       `json:"layoutVersion"`
	AuthorityID   string       `json:"authorityId,omitempty"`
	ReplicaIDs    []string     `json:"replicaIds,omitempty"`
	Records       []Reference  `json:"records,omitempty"`
	Digest        string       `json:"digest,omitempty"`
}

// Validate checks the project-record contract.
func (p ProjectRecord) Validate() error {
	if p.Kind != "project_record" {
		return fmt.Errorf("kind: expected %q, got %q", "project_record", p.Kind)
	}
	if err := p.Envelope.Validate(); err != nil {
		return err
	}
	if p.Name == "" {
		return fmt.Errorf("name: is required")
	}
	if err := p.State.Validate(); err != nil {
		return err
	}
	if p.LayoutVersion == "" {
		return fmt.Errorf("layoutVersion: is required")
	}
	if err := OptionalIdentifier("authorityId", p.AuthorityID); err != nil {
		return err
	}
	if p.Digest != "" {
		return RequireHash("digest", p.Digest)
	}
	return nil
}

// MemoryContent is the payload of a memory record.
type MemoryContent struct {
	Summary string   `json:"summary"`
	Details string   `json:"details,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// Applicability scopes a memory record to roles, trades, or languages.
type Applicability struct {
	Roles     []string `json:"roles,omitempty"`
	Trades    []string `json:"trades,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

// MemoryRecord is a knowledge record in the memory hierarchy.
type MemoryRecord struct {
	Envelope
	Level         string          `json:"level"`
	Lifecycle     MemoryLifecycle `json:"lifecycle"`
	Title         string          `json:"title,omitempty"`
	Content       MemoryContent   `json:"content"`
	Applicability *Applicability  `json:"applicability,omitempty"`
	Evidence      []Reference     `json:"evidence,omitempty"`
	Confidence    *float64        `json:"confidence,omitempty"`
	Supersedes    string          `json:"supersedes,omitempty"`
}

// Validate checks the memory-record contract.
func (m MemoryRecord) Validate() error {
	if m.Kind != "memory_record" {
		return fmt.Errorf("kind: expected %q, got %q", "memory_record", m.Kind)
	}
	if err := m.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireEnum("level", m.Level, "session", "task", "project", "role", "workforce"); err != nil {
		return err
	}
	if err := m.Lifecycle.Validate(); err != nil {
		return err
	}
	if m.Content.Summary == "" {
		return fmt.Errorf("content.summary: is required")
	}
	if m.Confidence != nil && (*m.Confidence < 0 || *m.Confidence > 1) {
		return fmt.Errorf("confidence: must be between 0 and 1")
	}
	return OptionalIdentifier("supersedes", m.Supersedes)
}

// IssueLinks ties an issue to work, commits, pull requests, and ADRs.
type IssueLinks struct {
	WorkPackages []string    `json:"workPackages,omitempty"`
	Commits      []string    `json:"commits,omitempty"`
	PullRequests []Reference `json:"pullRequests,omitempty"`
	ADRs         []string    `json:"adrs,omitempty"`
}

// Issue is a canonical issue, deficiency, or finding.
type Issue struct {
	Envelope
	Title      string      `json:"title"`
	Type       string      `json:"type"`
	Status     IssueState  `json:"status"`
	Severity   string      `json:"severity,omitempty"`
	Body       string      `json:"body,omitempty"`
	Milestone  string      `json:"milestone,omitempty"`
	ForgeRef   *Reference  `json:"forgeRef,omitempty"`
	Links      *IssueLinks `json:"links,omitempty"`
	DeferredTo string      `json:"deferredTo,omitempty"`
}

// Validate checks the issue contract.
func (i Issue) Validate() error {
	if i.Kind != "issue" {
		return fmt.Errorf("kind: expected %q, got %q", "issue", i.Kind)
	}
	if err := i.Envelope.Validate(); err != nil {
		return err
	}
	if i.Title == "" {
		return fmt.Errorf("title: is required")
	}
	if err := RequireEnum("type", i.Type,
		"feature", "user_story", "bug", "tech_debt", "refactor",
		"test", "documentation", "audit", "investigation", "deficiency"); err != nil {
		return err
	}
	if err := i.Status.Validate(); err != nil {
		return err
	}
	if i.Severity != "" {
		if err := RequireEnum("severity", i.Severity, "P0", "P1", "P2", "P3", "none"); err != nil {
			return err
		}
	}
	return OptionalIdentifier("deferredTo", i.DeferredTo)
}

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
