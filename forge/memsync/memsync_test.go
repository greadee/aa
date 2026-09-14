package memsync

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/forge"
)

type recorder struct {
	issues   []v1.Issue
	memories []v1.MemoryRecord
	failWith error
}

func (r *recorder) PutIssue(_ context.Context, issue v1.Issue) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.issues = append(r.issues, issue)
	return nil
}

func (r *recorder) PutMemoryRecord(_ context.Context, record v1.MemoryRecord) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.memories = append(r.memories, record)
	return nil
}

func testExporter() *Exporter {
	return NewExporter("proj_1").WithClock(func() time.Time { return time.Unix(0, 0).UTC() })
}

func sampleRepo() forge.Repository {
	return forge.Repository{ID: "repo_1", Owner: "greadee", Name: "aa1"}
}

func TestIssueMappingValidatesAgainstContracts(t *testing.T) {
	exporter := testExporter()
	issue := forge.Issue{
		ID: "issue_1", Number: 1, Title: "add forge", Body: "body",
		State: "open", Milestone: "milestone_1",
		Ref: forge.Ref{Kind: "issue", ID: "issue_1"},
	}
	mapped, err := exporter.Issue(sampleRepo(), issue)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := mapped.Validate(); err != nil {
		t.Fatalf("contract validation: %v", err)
	}
	if mapped.Status != v1.IssueOpen || mapped.Type != "feature" {
		t.Fatalf("unexpected mapping: %+v", mapped)
	}
	if mapped.ForgeRef == nil || mapped.ForgeRef.ID != "issue_1" {
		t.Fatalf("forge ref missing: %+v", mapped.ForgeRef)
	}
	if mapped.Provenance == nil || mapped.Provenance.Source != DefaultSource {
		t.Fatalf("provenance missing: %+v", mapped.Provenance)
	}
}

func TestIssueStateMapping(t *testing.T) {
	exporter := testExporter()
	cases := map[string]v1.IssueState{
		"":            v1.IssueOpen,
		"open":        v1.IssueOpen,
		"in_progress": v1.IssueInProgress,
		"review":      v1.IssueReview,
		"closed":      v1.IssueClosed,
		"deferred":    v1.IssueDeferred,
		"cancelled":   v1.IssueCancelled,
	}
	for state, want := range cases {
		mapped, err := exporter.Issue(sampleRepo(), forge.Issue{ID: "issue_1", Title: "t", State: state})
		if err != nil {
			t.Fatalf("state %q: %v", state, err)
		}
		if mapped.Status != want {
			t.Errorf("state %q -> %q, want %q", state, mapped.Status, want)
		}
	}
	if _, err := exporter.Issue(sampleRepo(), forge.Issue{ID: "issue_1", Title: "t", State: "weird"}); err == nil {
		t.Fatal("unknown state should fail")
	}
}

func TestCheckpointMapping(t *testing.T) {
	exporter := testExporter()
	checkpoint := forge.Checkpoint{
		ID: "checkpoint_1", Phase: "ph5-forge", Commit: "deadbeef", CreatedAt: time.Unix(5, 0).UTC(),
		Issues: []forge.Ref{{Kind: "issue", ID: "issue_1"}},
		PRs:    []forge.Ref{{Kind: "pull_request", ID: "pull_1"}},
	}
	record, err := exporter.CheckpointRecord(sampleRepo(), checkpoint)
	if err != nil {
		t.Fatalf("CheckpointRecord: %v", err)
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("contract validation: %v", err)
	}
	if record.Level != "project" || record.Lifecycle != v1.MemoryCandidate {
		t.Fatalf("unexpected record: %+v", record)
	}
	if len(record.Evidence) != 2 {
		t.Fatalf("expected 2 evidence refs, got %d", len(record.Evidence))
	}
	if record.ID != "mem_checkpoint_1" {
		t.Fatalf("id = %q", record.ID)
	}
}

func TestSyncDeliversToSink(t *testing.T) {
	exporter := testExporter()
	sink := &recorder{}
	if _, err := exporter.SyncIssue(context.Background(), sink, sampleRepo(), forge.Issue{ID: "issue_1", Title: "t"}); err != nil {
		t.Fatalf("SyncIssue: %v", err)
	}
	if _, err := exporter.SyncCheckpoint(context.Background(), sink, sampleRepo(), forge.Checkpoint{ID: "checkpoint_1", Phase: "ph5-forge", Commit: "abc"}); err != nil {
		t.Fatalf("SyncCheckpoint: %v", err)
	}
	if len(sink.issues) != 1 || len(sink.memories) != 1 {
		t.Fatalf("sink received %d issues, %d memories", len(sink.issues), len(sink.memories))
	}
}

func TestSinkErrorPropagates(t *testing.T) {
	exporter := testExporter()
	sink := &recorder{failWith: errors.New("memory down")}
	if _, err := exporter.SyncIssue(context.Background(), sink, sampleRepo(), forge.Issue{ID: "issue_1", Title: "t"}); err == nil {
		t.Fatal("expected sink error")
	}
}
