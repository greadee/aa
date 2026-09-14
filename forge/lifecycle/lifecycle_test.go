package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/forge"
	"github.com/greadee/aa/forge/fake"
	"github.com/greadee/aa/forge/memsync"
	"github.com/greadee/aa/forge/template"
)

var _ Forge = (*fake.Fake)(nil)

type recorder struct {
	issues   []v1.Issue
	memories []v1.MemoryRecord
}

func (r *recorder) PutIssue(_ context.Context, issue v1.Issue) error {
	r.issues = append(r.issues, issue)
	return nil
}

func (r *recorder) PutMemoryRecord(_ context.Context, record v1.MemoryRecord) error {
	r.memories = append(r.memories, record)
	return nil
}

func testManager(t *testing.T) (*Manager, *fake.Fake, *recorder) {
	t.Helper()
	root, ok := template.FindRoot(".")
	if !ok {
		t.Fatal("template root not found")
	}
	engine, err := template.NewEngine(root)
	if err != nil {
		t.Fatalf("template engine: %v", err)
	}
	f := fake.New()
	sink := &recorder{}
	exporter := memsync.NewExporter("proj_1").WithClock(func() time.Time { return time.Unix(0, 0).UTC() })
	return New(f, WithTemplates(engine), WithMemory(exporter, sink)), f, sink
}

func spec() PhaseSpec {
	return PhaseSpec{
		Repository: forge.Repository{Owner: "greadee", Name: "aa1"},
		Number:     5,
		Scope:      "forge",
		Objective:  "Git/GitHub orchestration",
	}
}

func TestFullPhaseLifecycle(t *testing.T) {
	manager, f, sink := testManager(t)
	ctx := context.Background()

	phase, err := manager.StartPhase(ctx, spec())
	if err != nil {
		t.Fatalf("StartPhase: %v", err)
	}
	if phase.Branch != "ph5-forge" {
		t.Fatalf("branch = %q", phase.Branch)
	}
	if !phase.PhasePR.Draft {
		t.Fatal("phase PR must start as a Draft")
	}
	if phase.PhasePR.Base != "main" || phase.PhasePR.Head != "ph5-forge" {
		t.Fatalf("unexpected phase PR: %+v", phase.PhasePR)
	}
	if phase.TrackingIssue.Number == 0 || phase.Milestone.ID == "" {
		t.Fatalf("missing tracking issue or milestone: %+v", phase)
	}
	if phase.TrackingIssue.Body == "" {
		t.Fatal("tracking issue body was not rendered")
	}
	if len(sink.issues) == 0 {
		t.Fatal("tracking issue was not synced to memory")
	}

	child, err := manager.CreateChildIssue(ctx, phase, ChildIssueSpec{
		Title: "add forge core", Goal: "deliver the core", Requirements: []string{"types"},
	})
	if err != nil {
		t.Fatalf("CreateChildIssue: %v", err)
	}
	if child.Milestone != phase.Milestone.ID {
		t.Fatalf("child issue not linked to milestone: %+v", child)
	}

	childPR, err := manager.OpenChildPR(ctx, phase, ChildPRSpec{
		Title: "add forge core", Branch: "feature/1-forge-core", Summary: "core", Issue: child.Number,
	})
	if err != nil {
		t.Fatalf("OpenChildPR: %v", err)
	}
	if childPR.Base != phase.Branch {
		t.Fatalf("child PR base = %q, want %q", childPR.Base, phase.Branch)
	}

	if _, err := manager.Checkpoint(ctx, phase, "deadbeef", nil, nil); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	if len(sink.memories) == 0 {
		t.Fatal("checkpoint was not synced to memory")
	}

	audit, err := manager.Audit(ctx, phase, []forge.AuditFinding{
		{Title: "stale doc", Classification: forge.ClassDocumentation, Severity: "P2"},
	})
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if audit.Result != forge.AuditReadyWithConditions {
		t.Fatalf("audit result = %q", audit.Result)
	}

	if _, err := manager.Release(ctx, phase, "v0.5.0", "phase 5", "deadbeef"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Merging fails closed without a human approval.
	if _, err := manager.CompletePhase(ctx, phase, "deadbeef", forge.MergeApproval{}); !errors.Is(err, forge.ErrHumanGate) {
		t.Fatalf("expected ErrHumanGate, got %v", err)
	}
	merged, err := manager.CompletePhase(ctx, phase, "deadbeef", forge.MergeApproval{Actor: "human:connor"})
	if err != nil {
		t.Fatalf("CompletePhase: %v", err)
	}
	if merged.State != "merged" {
		t.Fatalf("phase PR state = %q", merged.State)
	}
	if got, _ := f.PullRequest(phase.Repo.ID, phase.PhasePR.Number); got.State != "merged" {
		t.Fatalf("phase PR not merged: %+v", got)
	}
}

func TestStartPhaseRejectsBadScope(t *testing.T) {
	manager, _, _ := testManager(t)
	if _, err := manager.StartPhase(context.Background(), PhaseSpec{Scope: "Forge"}); !errors.Is(err, forge.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestChildPRRequiresBranch(t *testing.T) {
	manager, _, _ := testManager(t)
	phase, err := manager.StartPhase(context.Background(), spec())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.OpenChildPR(context.Background(), phase, ChildPRSpec{Title: "x"}); !errors.Is(err, forge.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestFindingsFromAuditMapsTypes(t *testing.T) {
	audit := forge.Audit{Findings: []forge.AuditFinding{
		{Title: "bug", Classification: forge.ClassBug, Severity: "P1"},
		{Title: "sec", Classification: forge.ClassSecurity, Severity: "P0"},
		{Title: "debt", Classification: forge.ClassTechDebt, Severity: "P3"},
	}}
	specs := FindingsFromAudit(audit)
	if len(specs) != 3 || specs[0].Type != "bug" || specs[1].Type != "audit" || specs[2].Type != "tech_debt" {
		t.Fatalf("unexpected specs: %+v", specs)
	}
}
