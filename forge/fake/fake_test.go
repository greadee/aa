package fake

import (
	"context"
	"errors"
	"testing"

	"github.com/greadee/aa/forge"
)

var (
	_ forge.Repositories = (*Fake)(nil)
	_ forge.Milestones   = (*Fake)(nil)
	_ forge.Issues       = (*Fake)(nil)
)

func testRepo(t *testing.T, f *Fake) forge.Repository {
	t.Helper()
	repo, err := f.CreateRepository(context.Background(), forge.Repository{Owner: "greadee", Name: "aa1"})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	return repo
}

func TestCreateRepositoryIsIdempotent(t *testing.T) {
	f := New()
	first := testRepo(t, f)
	second := testRepo(t, f)
	if first.ID != second.ID {
		t.Fatalf("ids differ: %s vs %s", first.ID, second.ID)
	}
	if first.DefaultBranch != "main" {
		t.Fatalf("default branch = %q", first.DefaultBranch)
	}
	if len(f.repos) != 1 {
		t.Fatalf("expected one repository, got %d", len(f.repos))
	}
	replays := 0
	for _, op := range f.AuditLog().Entries() {
		if op.Fields["idempotent"] == "replay" {
			replays++
		}
	}
	if replays != 1 {
		t.Fatalf("expected one replay, got %d", replays)
	}
}

func TestCreateBranchRequiresRepository(t *testing.T) {
	f := New()
	_, err := f.CreateBranch(context.Background(), "repo_missing", forge.Branch{Name: "feature/x"})
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	repo := testRepo(t, f)
	if _, err := f.CreateBranch(context.Background(), repo.ID, forge.Branch{Name: "bad name"}); !errors.Is(err, forge.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	first, err := f.CreateBranch(context.Background(), repo.ID, forge.Branch{Name: "feature/12-forge", CommitSHA: "abc123"})
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}
	second, _ := f.CreateBranch(context.Background(), repo.ID, forge.Branch{Name: "feature/12-forge", CommitSHA: "abc123"})
	if first != second {
		t.Fatalf("branch not idempotent: %+v vs %+v", first, second)
	}
}

func TestCreateMilestoneAndIssue(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	milestone, err := f.CreateMilestone(context.Background(), repo.ID, forge.Milestone{Title: "Phase 5 — Forge"})
	if err != nil {
		t.Fatalf("create milestone: %v", err)
	}
	issue, err := f.CreateIssue(context.Background(), repo.ID, forge.Issue{
		Title:     "add forge core",
		Body:      "body",
		Milestone: milestone.ID,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if issue.Number != 1 || issue.State != "open" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	second, _ := f.CreateIssue(context.Background(), repo.ID, forge.Issue{Title: "second"})
	if second.Number != 2 {
		t.Fatalf("expected issue number 2, got %d", second.Number)
	}
	// Same title is idempotent.
	dup, _ := f.CreateIssue(context.Background(), repo.ID, forge.Issue{Title: "add forge core"})
	if dup.Number != issue.Number {
		t.Fatalf("duplicate title created a new issue: %+v", dup)
	}
	updated, err := f.UpdateIssue(context.Background(), repo.ID, forge.Issue{Number: issue.Number, State: "closed"})
	if err != nil {
		t.Fatalf("update issue: %v", err)
	}
	if updated.State != "closed" {
		t.Fatalf("state = %q", updated.State)
	}
	if _, ok := f.Issue(repo.ID, issue.Number); !ok {
		t.Fatal("issue not found")
	}
}

func TestInjectedFailureAndRetry(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	f.Inject("issue.create", 1, forge.ErrRateLimited)
	_, err := f.CreateIssue(context.Background(), repo.ID, forge.Issue{Title: "flaky"})
	if !errors.Is(err, forge.ErrRateLimited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
	created, err := f.CreateIssue(context.Background(), repo.ID, forge.Issue{Title: "flaky"})
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	again, _ := f.CreateIssue(context.Background(), repo.ID, forge.Issue{Title: "flaky"})
	if created.Number != again.Number {
		t.Fatalf("retry was not idempotent: %d vs %d", created.Number, again.Number)
	}
}

func TestRunsAreDeterministic(t *testing.T) {
	run := func() []forge.Operation {
		f := New()
		repo := testRepo(t, f)
		_, _ = f.CreateMilestone(context.Background(), repo.ID, forge.Milestone{Title: "M"})
		_, _ = f.CreateIssue(context.Background(), repo.ID, forge.Issue{Title: "I"})
		_, _ = f.CreateBranch(context.Background(), repo.ID, forge.Branch{Name: "ph5-forge"})
		return f.AuditLog().Entries()
	}
	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("different entry counts: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Operation != b[i].Operation || a[i].Key != b[i].Key || !a[i].At.Equal(b[i].At) {
			t.Fatalf("entry %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}
