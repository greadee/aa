package v2

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

// Applicability scopes a memory record to roles or languages.
type Applicability struct {
	Roles     []string `json:"roles,omitempty"`
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
