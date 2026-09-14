package fake

import (
	"context"
	"fmt"

	"github.com/greadee/aa/forge"
)

func (f *Fake) nextPRNumber(repoID string) int {
	f.counters["prnumber:"+repoID]++
	return f.counters["prnumber:"+repoID]
}

// OpenPullRequest opens a pull request, idempotently by head, base, and title.
func (f *Fake) OpenPullRequest(ctx context.Context, repoID string, pr forge.PullRequest) (forge.PullRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.PullRequest{}, err
	}
	if _, ok := f.repos[repoID]; !ok {
		return forge.PullRequest{}, fmt.Errorf("%w: repository %s", forge.ErrNotFound, repoID)
	}
	if err := pr.Validate(); err != nil {
		return forge.PullRequest{}, err
	}
	key := forge.Key("pull.open", repoID, pr.Head, pr.Base, pr.Title)
	if existing, ok := f.idem[key]; ok {
		old := existing.(forge.PullRequest)
		f.record("pull.open", key, repoID, old.Ref, map[string]string{"idempotent": "replay"})
		return old, nil
	}
	if err := f.maybeFail("pull.open"); err != nil {
		return forge.PullRequest{}, err
	}
	if pr.ID == "" {
		pr.ID = f.nextID("pull")
	}
	if pr.Number == 0 {
		pr.Number = f.nextPRNumber(repoID)
	}
	if pr.State == "" {
		pr.State = "open"
	}
	pr.Ref = forge.Ref{Kind: "pull_request", ID: pr.ID}
	if f.pulls[repoID] == nil {
		f.pulls[repoID] = map[int]forge.PullRequest{}
	}
	f.pulls[repoID][pr.Number] = pr
	f.idem[key] = pr
	f.record("pull.open", key, repoID, pr.Ref, map[string]string{"draft": fmt.Sprint(pr.Draft)})
	return pr, nil
}

// UpdatePullRequest updates the mutable fields of an open pull request.
func (f *Fake) UpdatePullRequest(ctx context.Context, repoID string, pr forge.PullRequest) (forge.PullRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.PullRequest{}, err
	}
	current, ok := f.pulls[repoID][pr.Number]
	if !ok {
		return forge.PullRequest{}, fmt.Errorf("%w: pull request %d", forge.ErrNotFound, pr.Number)
	}
	if current.State == "merged" {
		return forge.PullRequest{}, fmt.Errorf("%w: pull request %d is already merged", forge.ErrConflict, pr.Number)
	}
	if pr.Title != "" {
		current.Title = pr.Title
	}
	if pr.Body != "" {
		current.Body = pr.Body
	}
	if pr.State != "" {
		current.State = pr.State
	}
	current.Draft = pr.Draft
	f.pulls[repoID][pr.Number] = current
	key := forge.Key("pull.update", repoID, fmt.Sprint(pr.Number))
	f.record("pull.update", key, repoID, current.Ref, map[string]string{"draft": fmt.Sprint(current.Draft)})
	return current, nil
}

// MergePullRequest merges a pull request. It fails closed without a human
// approval: merging is a distinct, gated operation (ADR-P5-003).
func (f *Fake) MergePullRequest(ctx context.Context, repoID string, number int, approval forge.MergeApproval) (forge.PullRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.PullRequest{}, err
	}
	current, ok := f.pulls[repoID][number]
	if !ok {
		return forge.PullRequest{}, fmt.Errorf("%w: pull request %d", forge.ErrNotFound, number)
	}
	if err := approval.Validate(); err != nil {
		return forge.PullRequest{}, fmt.Errorf("%w: %v", forge.ErrHumanGate, err)
	}
	if current.State == "merged" {
		key := forge.Key("pull.merge", repoID, fmt.Sprint(number))
		f.record("pull.merge", key, repoID, current.Ref, map[string]string{"idempotent": "replay"})
		return current, nil
	}
	if err := f.maybeFail("pull.merge"); err != nil {
		return forge.PullRequest{}, err
	}
	current.State = "merged"
	current.Draft = false
	f.pulls[repoID][number] = current
	key := forge.Key("pull.merge", repoID, fmt.Sprint(number))
	f.record("pull.merge", key, repoID, current.Ref, map[string]string{"actor": approval.Actor})
	return current, nil
}

// CreateCheckpoint records a phase checkpoint, idempotently by phase and commit.
func (f *Fake) CreateCheckpoint(ctx context.Context, repoID string, checkpoint forge.Checkpoint) (forge.Checkpoint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Checkpoint{}, err
	}
	if err := checkpoint.Validate(); err != nil {
		return forge.Checkpoint{}, err
	}
	key := forge.Key("checkpoint.create", repoID, checkpoint.Phase, checkpoint.Commit)
	if existing, ok := f.idem[key]; ok {
		f.record("checkpoint.create", key, repoID, forge.Ref{Kind: "checkpoint", ID: existing.(forge.Checkpoint).ID}, map[string]string{"idempotent": "replay"})
		return existing.(forge.Checkpoint), nil
	}
	if err := f.maybeFail("checkpoint.create"); err != nil {
		return forge.Checkpoint{}, err
	}
	if checkpoint.ID == "" {
		checkpoint.ID = f.nextID("checkpoint")
	}
	if checkpoint.CreatedAt.IsZero() {
		checkpoint.CreatedAt = f.clock.Now()
	}
	if f.checkpoints[repoID] == nil {
		f.checkpoints[repoID] = map[string]forge.Checkpoint{}
	}
	f.checkpoints[repoID][checkpoint.ID] = checkpoint
	f.idem[key] = checkpoint
	f.record("checkpoint.create", key, repoID, forge.Ref{Kind: "checkpoint", ID: checkpoint.ID}, map[string]string{"phase": checkpoint.Phase})
	return checkpoint, nil
}

// CreateRelease creates a release/tag, idempotently by tag.
func (f *Fake) CreateRelease(ctx context.Context, repoID string, release forge.Release) (forge.Release, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Release{}, err
	}
	if err := release.Validate(); err != nil {
		return forge.Release{}, err
	}
	key := forge.Key("release.create", repoID, release.Tag)
	if existing, ok := f.idem[key]; ok {
		f.record("release.create", key, repoID, forge.Ref{Kind: "release", ID: existing.(forge.Release).ID}, map[string]string{"idempotent": "replay"})
		return existing.(forge.Release), nil
	}
	if err := f.maybeFail("release.create"); err != nil {
		return forge.Release{}, err
	}
	if release.ID == "" {
		release.ID = f.nextID("release")
	}
	if release.CreatedAt.IsZero() {
		release.CreatedAt = f.clock.Now()
	}
	if f.releases[repoID] == nil {
		f.releases[repoID] = map[string]forge.Release{}
	}
	f.releases[repoID][release.Tag] = release
	f.idem[key] = release
	f.record("release.create", key, repoID, forge.Ref{Kind: "release", ID: release.ID}, map[string]string{"tag": release.Tag})
	return release, nil
}

// PullRequest returns a pull request by repository and number.
func (f *Fake) PullRequest(repoID string, number int) (forge.PullRequest, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	pr, ok := f.pulls[repoID][number]
	return pr, ok
}

// Checkpoint returns a checkpoint by ID.
func (f *Fake) Checkpoint(repoID, id string) (forge.Checkpoint, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp, ok := f.checkpoints[repoID][id]
	return cp, ok
}

// Release returns a release by tag.
func (f *Fake) Release(repoID, tag string) (forge.Release, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rel, ok := f.releases[repoID][tag]
	return rel, ok
}
