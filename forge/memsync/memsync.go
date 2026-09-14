// Package memsync maps forge objects to aa-contracts records and syncs them
// through a Sink.
//
// The forge never writes canonical memory directly: it validates against the
// contract bindings and hands records to a Sink, which the host wires to
// aa-memory (over RPC in a later phase).
package memsync

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/forge"
)

// DefaultSource names the producer recorded in provenance.
const DefaultSource = "aa-forge"

// Sink receives validated contract records.
type Sink interface {
	PutIssue(ctx context.Context, issue v1.Issue) error
	PutMemoryRecord(ctx context.Context, record v1.MemoryRecord) error
}

// Exporter deterministically maps forge objects to contracts.
type Exporter struct {
	projectID string
	source    string
	now       func() time.Time
}

// NewExporter returns an exporter scoped to a project.
func NewExporter(projectID string) *Exporter {
	return &Exporter{projectID: projectID, source: DefaultSource, now: time.Now}
}

// WithClock overrides the time source (useful for deterministic tests).
func (e *Exporter) WithClock(now func() time.Time) *Exporter {
	e.now = now
	return e
}

// Issue maps a forge issue to a v1.Issue.
func (e *Exporter) Issue(repo forge.Repository, issue forge.Issue) (v1.Issue, error) {
	_ = repo // reserved for repository-scoped provenance
	if err := issue.Validate(); err != nil {
		return v1.Issue{}, err
	}
	status, err := mapIssueState(issue.State)
	if err != nil {
		return v1.Issue{}, err
	}
	issueType := issue.Type
	if issueType == "" {
		issueType = "feature"
	}
	if err := v1.RequireEnum("type", issueType,
		"feature", "user_story", "bug", "tech_debt", "refactor",
		"test", "documentation", "audit", "investigation", "deficiency"); err != nil {
		return v1.Issue{}, err
	}
	created := e.now().UTC().Format(time.RFC3339)
	out := v1.Issue{
		Envelope: v1.Envelope{
			ContractVersion: v1.Version,
			Kind:            "issue",
			ID:              issue.ID,
			ProjectID:       e.projectID,
			CreatedAt:       created,
			Provenance: &v1.Provenance{
				Source:     e.source,
				ProducedAt: created,
				Confidence: "exact",
			},
		},
		Title:     issue.Title,
		Type:      issueType,
		Status:    status,
		Body:      issue.Body,
		Milestone: issue.Milestone,
	}
	if issue.Ref.ID != "" {
		out.ForgeRef = &v1.Reference{Kind: defaultKind(issue.Ref.Kind, "issue"), ID: issue.Ref.ID}
	}
	if err := out.Validate(); err != nil {
		return v1.Issue{}, fmt.Errorf("memsync: issue: %w", err)
	}
	return out, nil
}

// CheckpointRecord maps a forge checkpoint to a candidate v1.MemoryRecord.
func (e *Exporter) CheckpointRecord(repo forge.Repository, checkpoint forge.Checkpoint) (v1.MemoryRecord, error) {
	_ = repo
	if err := checkpoint.Validate(); err != nil {
		return v1.MemoryRecord{}, err
	}
	at := checkpoint.CreatedAt
	if at.IsZero() {
		at = e.now()
	}
	created := at.UTC().Format(time.RFC3339)
	record := v1.MemoryRecord{
		Envelope: v1.Envelope{
			ContractVersion: v1.Version,
			Kind:            "memory_record",
			ID:              "mem_" + checkpoint.ID,
			ProjectID:       e.projectID,
			CreatedAt:       created,
			Provenance: &v1.Provenance{
				Source:     e.source,
				ProducedAt: created,
				Confidence: "exact",
			},
		},
		Level:      "project",
		Lifecycle:  v1.MemoryCandidate,
		Title:      "Checkpoint " + checkpoint.Phase,
		Content:    v1.MemoryContent{Summary: fmt.Sprintf("Phase %s checkpoint at %s", checkpoint.Phase, checkpoint.Commit)},
		Confidence: float64Ptr(1.0),
	}
	for _, ref := range checkpoint.Issues {
		record.Evidence = append(record.Evidence, v1.Reference{Kind: defaultKind(ref.Kind, "issue"), ID: ref.ID})
	}
	for _, ref := range checkpoint.PRs {
		record.Evidence = append(record.Evidence, v1.Reference{Kind: defaultKind(ref.Kind, "pull_request"), ID: ref.ID})
	}
	if err := record.Validate(); err != nil {
		return v1.MemoryRecord{}, fmt.Errorf("memsync: checkpoint: %w", err)
	}
	return record, nil
}

// SyncIssue maps and delivers an issue to the sink.
func (e *Exporter) SyncIssue(ctx context.Context, sink Sink, repo forge.Repository, issue forge.Issue) (v1.Issue, error) {
	record, err := e.Issue(repo, issue)
	if err != nil {
		return v1.Issue{}, err
	}
	if err := sink.PutIssue(ctx, record); err != nil {
		return v1.Issue{}, err
	}
	return record, nil
}

// SyncCheckpoint maps and delivers a checkpoint to the sink.
func (e *Exporter) SyncCheckpoint(ctx context.Context, sink Sink, repo forge.Repository, checkpoint forge.Checkpoint) (v1.MemoryRecord, error) {
	record, err := e.CheckpointRecord(repo, checkpoint)
	if err != nil {
		return v1.MemoryRecord{}, err
	}
	if err := sink.PutMemoryRecord(ctx, record); err != nil {
		return v1.MemoryRecord{}, err
	}
	return record, nil
}

func mapIssueState(state string) (v1.IssueState, error) {
	switch state {
	case "", "open":
		return v1.IssueOpen, nil
	case "in_progress":
		return v1.IssueInProgress, nil
	case "review":
		return v1.IssueReview, nil
	case "closed":
		return v1.IssueClosed, nil
	case "deferred":
		return v1.IssueDeferred, nil
	case "cancelled":
		return v1.IssueCancelled, nil
	default:
		return "", fmt.Errorf("memsync: unknown issue state %q", state)
	}
}

func defaultKind(kind, fallback string) string {
	if kind == "" {
		return fallback
	}
	return kind
}

func float64Ptr(v float64) *float64 { return &v }
