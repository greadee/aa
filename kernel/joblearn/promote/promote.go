// Package promote persists learning candidates as CANDIDATE memory records.
//
// The kernel never writes canonical state directly: it maps its candidates to
// memory_record contracts, derives a stable identity for each so re-proposing
// is idempotent, and hands them to a Sink. Promotion out of CANDIDATE stays
// explicit and is validated by memory's lifecycle state machine. No model is
// called and no candidate is ever auto-promoted.
package promote

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
)

// DefaultSource names the producer recorded on promoted records.
const DefaultSource = "aa-kernel.joblearn"

// Sink persists candidate records on behalf of the kernel. memory/repo
// implements it; tests use a fake so the engine stays unaware of the store.
type Sink interface {
	// Propose stores a record in the CANDIDATE lifecycle and returns the
	// stored revision.
	Propose(record v1.MemoryRecord) (v1.MemoryRecord, error)
	// Transition applies a validated lifecycle transition to a stored record.
	Transition(id string, to v1.MemoryLifecycle) (v1.MemoryRecord, error)
}

// Options configure a Promoter.
type Options struct {
	// ProjectID scopes the records.
	ProjectID string
	// Source overrides the provenance source used when a candidate carries
	// none.
	Source string
	// Now overrides the clock; deterministic tests inject a fixed one.
	Now func() time.Time
}

// Promoter maps candidates to memory records and persists them through a Sink.
type Promoter struct {
	sink Sink
	opts Options
}

// New validates the sink and returns a promoter.
func New(sink Sink, opts Options) (*Promoter, error) {
	if sink == nil {
		return nil, invalid("sink is required")
	}
	if opts.Source == "" {
		opts.Source = DefaultSource
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Promoter{sink: sink, opts: opts}, nil
}

// Propose maps candidates to CANDIDATE memory records and persists them,
// returning the stored records in first-seen order. Re-proposing a candidate is
// idempotent because identity derives from its kind, level, scope, and title.
func (p *Promoter) Propose(candidates []joblearn.Candidate) ([]v1.MemoryRecord, error) {
	seen := make(map[string]bool, len(candidates))
	out := make([]v1.MemoryRecord, 0, len(candidates))
	for _, c := range candidates {
		if err := c.Validate(); err != nil {
			return nil, err
		}
		record, err := p.record(c)
		if err != nil {
			return nil, err
		}
		if seen[record.ID] {
			continue
		}
		seen[record.ID] = true
		stored, err := p.sink.Propose(record)
		if err != nil {
			return nil, err
		}
		out = append(out, stored)
	}
	return out, nil
}

// Promote explicitly advances a stored record's lifecycle. It delegates the
// transition to the Sink, so memory's state machine remains authoritative and
// invalid or skipping transitions are rejected.
func (p *Promoter) Promote(id string, to v1.MemoryLifecycle) (v1.MemoryRecord, error) {
	if id == "" {
		return v1.MemoryRecord{}, invalid("record id is required")
	}
	if err := to.Validate(); err != nil {
		return v1.MemoryRecord{}, invalid("target lifecycle: %v", err)
	}
	return p.sink.Transition(id, to)
}

// record maps one candidate to a CANDIDATE memory record.
func (p *Promoter) record(c joblearn.Candidate) (v1.MemoryRecord, error) {
	confidence := c.Confidence
	record := v1.MemoryRecord{
		Envelope: v1.Envelope{
			ContractVersion: v1.Version,
			Kind:            "memory_record",
			ID:              identity(c),
			ProjectID:       p.opts.ProjectID,
			CreatedAt:       p.opts.Now().UTC().Format(time.RFC3339),
			Provenance:      p.provenance(c),
		},
		Level:         string(c.Level),
		Lifecycle:     v1.MemoryCandidate,
		Title:         c.Title,
		Content:       v1.MemoryContent{Summary: c.Content, Tags: tags(c)},
		Applicability: c.Applicability,
		Evidence:      append([]joblearn.Reference(nil), c.Evidence...),
		Confidence:    &confidence,
	}
	if err := record.Validate(); err != nil {
		return v1.MemoryRecord{}, fmt.Errorf("promote: %w", err)
	}
	return record, nil
}

// provenance carries the candidate's provenance forward, normalizing the source
// to the contract identifier form and defaulting missing fields.
func (p *Promoter) provenance(c joblearn.Candidate) *v1.Provenance {
	if c.Provenance == nil {
		return &v1.Provenance{Source: normalizeSource(p.opts.Source), ProducedAt: p.opts.Now().UTC().Format(time.RFC3339)}
	}
	prov := *c.Provenance
	prov.Source = normalizeSource(prov.Source)
	if prov.ProducedAt == "" {
		prov.ProducedAt = p.opts.Now().UTC().Format(time.RFC3339)
	}
	return &prov
}

// identity derives a stable record id from the candidate's identity fields.
func identity(c joblearn.Candidate) string {
	key := strings.Join([]string{string(c.Kind), string(c.Level), c.Scope, c.Title}, "\x00")
	sum := sha256.Sum256([]byte(key))
	return "mem_" + hex.EncodeToString(sum[:16])
}

// tags labels a record with its candidate kind, level, and scope.
func tags(c joblearn.Candidate) []string {
	out := []string{string(c.Kind), string(c.Level)}
	if c.Scope != "" {
		out = append(out, "scope:"+c.Scope)
	}
	return out
}

// normalizeSource returns a valid contract identifier derived from source.
func normalizeSource(source string) string {
	if source == "" {
		return DefaultSource
	}
	if v1.RequireIdentifier("source", source) == nil {
		return source
	}
	var b strings.Builder
	for _, r := range source {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '_', r == '.', r == ':', r == '@', r == '+', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	if out == "" {
		return DefaultSource
	}
	if first := out[0]; !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')) {
		out = "src" + out
	}
	if len(out) > 256 {
		out = out[:256]
	}
	return out
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
