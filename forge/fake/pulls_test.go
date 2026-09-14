package fake

import (
	"context"
	"errors"
	"testing"

	"github.com/greadee/aa/forge"
)

var (
	_ forge.PullRequests = (*Fake)(nil)
	_ forge.Checkpoints  = (*Fake)(nil)
	_ forge.Releases     = (*Fake)(nil)
)

func openPR(t *testing.T, f *Fake, repo forge.Repository, title string, draft bool) forge.PullRequest {
	t.Helper()
	pr, err := f.OpenPullRequest(context.Background(), repo.ID, forge.PullRequest{
		Title: title, Base: "main", Head: "ph5-forge", Draft: draft,
	})
	if err != nil {
		t.Fatalf("open pull request: %v", err)
	}
	return pr
}

func TestOpenPullRequestIsIdempotent(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	first := openPR(t, f, repo, "Phase 5: forge", true)
	if !first.Draft || first.Number != 1 {
		t.Fatalf("unexpected PR: %+v", first)
	}
	second := openPR(t, f, repo, "Phase 5: forge", true)
	if first.ID != second.ID || first.Number != second.Number {
		t.Fatalf("not idempotent: %+v vs %+v", first, second)
	}
}

func TestMergeRequiresHumanApproval(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	pr := openPR(t, f, repo, "child PR", false)

	_, err := f.MergePullRequest(context.Background(), repo.ID, pr.Number, forge.MergeApproval{})
	if !errors.Is(err, forge.ErrHumanGate) {
		t.Fatalf("expected ErrHumanGate, got %v", err)
	}
	if got, _ := f.PullRequest(repo.ID, pr.Number); got.State == "merged" {
		t.Fatal("PR must not merge without approval")
	}

	merged, err := f.MergePullRequest(context.Background(), repo.ID, pr.Number, forge.MergeApproval{Actor: "human:connor"})
	if err != nil {
		t.Fatalf("merge with approval: %v", err)
	}
	if merged.State != "merged" || merged.Draft {
		t.Fatalf("unexpected merged PR: %+v", merged)
	}
	again, err := f.MergePullRequest(context.Background(), repo.ID, pr.Number, forge.MergeApproval{Actor: "human:connor"})
	if err != nil || again.State != "merged" {
		t.Fatalf("merge replay: %v %+v", err, again)
	}
}

func TestUpdateMergedPullRequestConflicts(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	pr := openPR(t, f, repo, "child PR", false)
	if _, err := f.MergePullRequest(context.Background(), repo.ID, pr.Number, forge.MergeApproval{Actor: "human:x"}); err != nil {
		t.Fatal(err)
	}
	_, err := f.UpdatePullRequest(context.Background(), repo.ID, forge.PullRequest{Number: pr.Number, Title: "edit"})
	if !errors.Is(err, forge.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCheckpointIsIdempotent(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	cp := forge.Checkpoint{Phase: "ph5-forge", Commit: "deadbeef", Issues: []forge.Ref{{Kind: "issue", ID: "issue_1"}}}
	first, err := f.CreateCheckpoint(context.Background(), repo.ID, cp)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if first.CreatedAt.IsZero() {
		t.Fatal("checkpoint timestamp not set")
	}
	second, _ := f.CreateCheckpoint(context.Background(), repo.ID, cp)
	if first.ID != second.ID {
		t.Fatalf("checkpoint not idempotent: %s vs %s", first.ID, second.ID)
	}
}

func TestReleaseIsIdempotentAndValidated(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	if _, err := f.CreateRelease(context.Background(), repo.ID, forge.Release{Tag: "1.0"}); !errors.Is(err, forge.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	first, err := f.CreateRelease(context.Background(), repo.ID, forge.Release{Tag: "v0.1.0", Notes: "first"})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	second, _ := f.CreateRelease(context.Background(), repo.ID, forge.Release{Tag: "v0.1.0", Notes: "changed"})
	if first != second || second.Notes != "first" {
		t.Fatalf("release not idempotent: %+v vs %+v", first, second)
	}
}
