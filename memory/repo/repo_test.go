package repo

import (
	"errors"
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func newRepos(t *testing.T) (*store.Store, *projection.MemProjection, *IssueRepository, *StrategyRepository) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := projection.NewMem()
	return s, p, NewIssueRepository(s, p), NewStrategyRepository(s, p)
}

func TestIssueRepositoryLifecycle(t *testing.T) {
	s, p, issues, _ := newRepos(t)
	issue, err := issues.Create(v1.Issue{Title: "Add contract spine", Type: "feature"})
	if err != nil {
		t.Fatal(err)
	}
	if issue.ID == "" || issue.Status != v1.IssueOpen {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	if _, ok := p.Get("issue", issue.ID); !ok {
		t.Fatal("projection not updated")
	}
	updated, err := issues.SetStatus(issue.ID, v1.IssueInProgress)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != v1.IssueInProgress {
		t.Fatalf("status = %s", updated.Status)
	}
	list, err := issues.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v err = %v", list, err)
	}
	if _, err := issues.SetStatus(issue.ID, v1.IssueState("bogus")); err == nil {
		t.Fatal("expected invalid status error")
	}
	if _, err := issues.Get("missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	_ = s
}

func TestStrategyPromotion(t *testing.T) {
	_, _, _, strategies := newRepos(t)
	strategy, err := strategies.Propose(v1.Strategy{Title: "Contracts before behavior", Guidance: "Do it."})
	if err != nil {
		t.Fatal(err)
	}
	if strategy.Lifecycle != v1.MemoryCandidate {
		t.Fatalf("lifecycle = %s", strategy.Lifecycle)
	}
	if _, err := strategies.Transition(strategy.ID, v1.MemoryActive); err == nil {
		t.Fatal("expected illegal CANDIDATE -> ACTIVE transition")
	}
	if _, err := strategies.Transition(strategy.ID, v1.MemoryValidated); err != nil {
		t.Fatal(err)
	}
	if _, err := strategies.Transition(strategy.ID, v1.MemoryActive); err != nil {
		t.Fatal(err)
	}
	active, err := strategies.ListActive()
	if err != nil || len(active) != 1 {
		t.Fatalf("active = %v err = %v", active, err)
	}
}

func TestProposeRejectsNonCandidate(t *testing.T) {
	_, _, _, strategies := newRepos(t)
	if _, err := strategies.Propose(v1.Strategy{
		Title: "x", Guidance: "y", Lifecycle: v1.MemoryActive,
	}); err == nil {
		t.Fatal("expected propose to require CANDIDATE")
	}
}

func TestSupersede(t *testing.T) {
	_, _, _, strategies := newRepos(t)
	old, err := strategies.Propose(v1.Strategy{Title: "old", Guidance: "g"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := strategies.Transition(old.ID, v1.MemoryValidated); err != nil {
		t.Fatal(err)
	}
	if _, err := strategies.Transition(old.ID, v1.MemoryActive); err != nil {
		t.Fatal(err)
	}
	next, err := strategies.Propose(v1.Strategy{Title: "new", Guidance: "g2"})
	if err != nil {
		t.Fatal(err)
	}
	if err := strategies.Supersede(old.ID, next.ID); err != nil {
		t.Fatal(err)
	}
	gotOld, err := strategies.Get(old.ID)
	if err != nil || gotOld.Lifecycle != v1.MemorySuperseded {
		t.Fatalf("old = %+v err = %v", gotOld, err)
	}
	gotNext, err := strategies.Get(next.ID)
	if err != nil || gotNext.Supersedes != old.ID {
		t.Fatalf("next = %+v err = %v", gotNext, err)
	}
}

func TestCanTransition(t *testing.T) {
	if !CanTransition(v1.MemoryEphemeral, v1.MemoryCandidate) {
		t.Error("EPHEMERAL -> CANDIDATE should be allowed")
	}
	if CanTransition(v1.MemoryArchived, v1.MemoryActive) {
		t.Error("ARCHIVED -> ACTIVE should be denied")
	}
}
