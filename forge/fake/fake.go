// Package fake is a deterministic, in-memory forge implementation.
//
// It is the reference implementation for tests and the offline substrate for
// the phase lifecycle: no network, no credentials, stable IDs and ordering.
package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/greadee/aa/forge"
)

// Fake is a deterministic in-memory forge.
type Fake struct {
	mu    sync.Mutex
	clock forge.Clock
	log   *forge.AuditLog

	repos       map[string]forge.Repository
	branches    map[string]map[string]forge.Branch
	milestones  map[string]map[string]forge.Milestone
	issues      map[string]map[int]forge.Issue
	pulls       map[string]map[int]forge.PullRequest
	checkpoints map[string]map[string]forge.Checkpoint
	releases    map[string]map[string]forge.Release
	audits      map[string]map[string]forge.Audit

	counters map[string]int
	idem     map[string]any
	failures map[string]failure
}

type failure struct {
	remaining int
	err       error
}

// New returns an empty fake with a real clock.
func New() *Fake {
	return NewWithClock(forge.NewFixedClock())
}

// NewWithClock returns an empty fake using the given clock.
func NewWithClock(clock forge.Clock) *Fake {
	if clock == nil {
		clock = forge.NewFixedClock()
	}
	return &Fake{
		clock:       clock,
		log:         forge.NewAuditLog(),
		repos:       map[string]forge.Repository{},
		branches:    map[string]map[string]forge.Branch{},
		milestones:  map[string]map[string]forge.Milestone{},
		issues:      map[string]map[int]forge.Issue{},
		pulls:       map[string]map[int]forge.PullRequest{},
		checkpoints: map[string]map[string]forge.Checkpoint{},
		releases:    map[string]map[string]forge.Release{},
		audits:      map[string]map[string]forge.Audit{},
		counters:    map[string]int{},
		idem:        map[string]any{},
		failures:    map[string]failure{},
	}
}

// AuditLog returns the forge audit log.
func (f *Fake) AuditLog() *forge.AuditLog { return f.log }

// Inject makes the next calls to an operation fail with err, enabling
// retry/idempotency tests. Use operation names such as "repository.create".
func (f *Fake) Inject(operation string, times int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures[operation] = failure{remaining: times, err: err}
}

func (f *Fake) maybeFail(operation string) error {
	if fail, ok := f.failures[operation]; ok && fail.remaining > 0 {
		f.failures[operation] = failure{remaining: fail.remaining - 1, err: fail.err}
		return fail.err
	}
	return nil
}

func (f *Fake) nextID(prefix string) string {
	f.counters[prefix]++
	return fmt.Sprintf("%s_%d", prefix, f.counters[prefix])
}

func (f *Fake) nextNumber(repoID string) int {
	f.counters["number:"+repoID]++
	return f.counters["number:"+repoID]
}

func (f *Fake) record(operation, key, repoID string, target forge.Ref, fields map[string]string) {
	f.log.Record(forge.Operation{
		Operation: operation,
		Key:       key,
		RepoID:    repoID,
		Target:    target,
		Fields:    fields,
		At:        f.clock.Now(),
	})
}

// Repository returns a repository by ID.
func (f *Fake) Repository(id string) (forge.Repository, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	repo, ok := f.repos[id]
	return repo, ok
}

// Issue returns an issue by repository and number.
func (f *Fake) Issue(repoID string, number int) (forge.Issue, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	issue, ok := f.issues[repoID][number]
	return issue, ok
}

// Milestone returns a milestone by repository and ID.
func (f *Fake) Milestone(repoID, milestoneID string) (forge.Milestone, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.milestones[repoID][milestoneID]
	return m, ok
}

// Branch returns a branch by repository and name.
func (f *Fake) Branch(repoID, name string) (forge.Branch, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.branches[repoID][name]
	return b, ok
}

// CreateRepository creates a repository, idempotently by owner and name.
func (f *Fake) CreateRepository(ctx context.Context, repo forge.Repository) (forge.Repository, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Repository{}, err
	}
	if err := repo.Validate(); err != nil {
		return forge.Repository{}, err
	}
	key := forge.Key("repository.create", repo.Owner, repo.Name)
	if existing, ok := f.idem[key]; ok {
		f.record("repository.create", key, existing.(forge.Repository).ID, forge.Ref{}, map[string]string{"idempotent": "replay"})
		return existing.(forge.Repository), nil
	}
	if err := f.maybeFail("repository.create"); err != nil {
		return forge.Repository{}, err
	}
	if repo.ID == "" {
		repo.ID = f.nextID("repo")
	}
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}
	repo.Ref = forge.Ref{Kind: "repository", ID: repo.ID, URL: fmt.Sprintf("forge://%s/%s", repo.Owner, repo.Name)}
	f.repos[repo.ID] = repo
	f.idem[key] = repo
	f.record("repository.create", key, repo.ID, repo.Ref, nil)
	return repo, nil
}

// CreateBranch records a branch on a repository, idempotently by name.
func (f *Fake) CreateBranch(ctx context.Context, repoID string, branch forge.Branch) (forge.Branch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Branch{}, err
	}
	if _, ok := f.repos[repoID]; !ok {
		return forge.Branch{}, fmt.Errorf("%w: repository %s", forge.ErrNotFound, repoID)
	}
	if err := forge.ValidateBranchName(branch.Name); err != nil {
		return forge.Branch{}, err
	}
	key := forge.Key("branch.create", repoID, branch.Name)
	if existing, ok := f.idem[key]; ok {
		f.record("branch.create", key, repoID, forge.Ref{Kind: "branch", ID: branch.Name}, map[string]string{"idempotent": "replay"})
		return existing.(forge.Branch), nil
	}
	if err := f.maybeFail("branch.create"); err != nil {
		return forge.Branch{}, err
	}
	if f.branches[repoID] == nil {
		f.branches[repoID] = map[string]forge.Branch{}
	}
	f.branches[repoID][branch.Name] = branch
	f.idem[key] = branch
	f.record("branch.create", key, repoID, forge.Ref{Kind: "branch", ID: branch.Name}, map[string]string{"commit": branch.CommitSHA})
	return branch, nil
}

// CreateMilestone creates a milestone, idempotently by title.
func (f *Fake) CreateMilestone(ctx context.Context, repoID string, milestone forge.Milestone) (forge.Milestone, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Milestone{}, err
	}
	if _, ok := f.repos[repoID]; !ok {
		return forge.Milestone{}, fmt.Errorf("%w: repository %s", forge.ErrNotFound, repoID)
	}
	if milestone.Title == "" {
		return forge.Milestone{}, fmt.Errorf("%w: milestone title is required", forge.ErrInvalid)
	}
	key := forge.Key("milestone.create", repoID, milestone.Title)
	if existing, ok := f.idem[key]; ok {
		f.record("milestone.create", key, repoID, existing.(forge.Milestone).Ref, map[string]string{"idempotent": "replay"})
		return existing.(forge.Milestone), nil
	}
	if err := f.maybeFail("milestone.create"); err != nil {
		return forge.Milestone{}, err
	}
	if milestone.ID == "" {
		milestone.ID = f.nextID("milestone")
	}
	if milestone.State == "" {
		milestone.State = "open"
	}
	milestone.Ref = forge.Ref{Kind: "milestone", ID: milestone.ID}
	if f.milestones[repoID] == nil {
		f.milestones[repoID] = map[string]forge.Milestone{}
	}
	f.milestones[repoID][milestone.ID] = milestone
	f.idem[key] = milestone
	f.record("milestone.create", key, repoID, milestone.Ref, map[string]string{"title": milestone.Title})
	return milestone, nil
}

// CreateIssue creates an issue, idempotently by title.
func (f *Fake) CreateIssue(ctx context.Context, repoID string, issue forge.Issue) (forge.Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Issue{}, err
	}
	if _, ok := f.repos[repoID]; !ok {
		return forge.Issue{}, fmt.Errorf("%w: repository %s", forge.ErrNotFound, repoID)
	}
	if err := issue.Validate(); err != nil {
		return forge.Issue{}, err
	}
	key := forge.Key("issue.create", repoID, issue.Title)
	if existing, ok := f.idem[key]; ok {
		old := existing.(forge.Issue)
		f.record("issue.create", key, repoID, old.Ref, map[string]string{"idempotent": "replay"})
		return old, nil
	}
	if err := f.maybeFail("issue.create"); err != nil {
		return forge.Issue{}, err
	}
	if issue.ID == "" {
		issue.ID = f.nextID("issue")
	}
	if issue.State == "" {
		issue.State = "open"
	}
	if issue.Number == 0 {
		issue.Number = f.nextNumber(repoID)
	}
	issue.Ref = forge.Ref{Kind: "issue", ID: issue.ID}
	if f.issues[repoID] == nil {
		f.issues[repoID] = map[int]forge.Issue{}
	}
	f.issues[repoID][issue.Number] = issue
	f.idem[key] = issue
	f.record("issue.create", key, repoID, issue.Ref, map[string]string{"number": fmt.Sprint(issue.Number)})
	return issue, nil
}

// UpdateIssue updates the mutable fields of an existing issue.
func (f *Fake) UpdateIssue(ctx context.Context, repoID string, issue forge.Issue) (forge.Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Issue{}, err
	}
	current, ok := f.issues[repoID][issue.Number]
	if !ok {
		return forge.Issue{}, fmt.Errorf("%w: issue %d", forge.ErrNotFound, issue.Number)
	}
	if issue.Title != "" {
		current.Title = issue.Title
	}
	if issue.Body != "" {
		current.Body = issue.Body
	}
	if issue.State != "" {
		current.State = issue.State
	}
	if issue.Labels != nil {
		current.Labels = issue.Labels
	}
	if issue.Milestone != "" {
		current.Milestone = issue.Milestone
	}
	f.issues[repoID][issue.Number] = current
	key := forge.Key("issue.update", repoID, fmt.Sprint(issue.Number))
	f.record("issue.update", key, repoID, current.Ref, map[string]string{"state": current.State})
	return current, nil
}
