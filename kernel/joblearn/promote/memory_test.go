package promote

import (
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

func newMemoryPromoter(t *testing.T) (*Promoter, *repo.MemoryRecordRepository) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	records := repo.NewMemoryRecordRepository(s, projection.NewMem())
	sink, err := NewMemorySink(records)
	if err != nil {
		t.Fatal(err)
	}
	p, err := New(sink, Options{ProjectID: "prj_1", Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	return p, records
}

func TestMemoryPromotionEndToEnd(t *testing.T) {
	p, records := newMemoryPromoter(t)
	stored, err := p.Propose([]joblearn.Candidate{candidate()})
	if err != nil {
		t.Fatal(err)
	}
	id := stored[0].ID

	got, err := records.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Lifecycle != v1.MemoryCandidate {
		t.Fatalf("lifecycle = %s, want CANDIDATE", got.Lifecycle)
	}

	if _, err := p.Propose([]joblearn.Candidate{candidate()}); err != nil {
		t.Fatal(err)
	}
	candidates, err := records.ListByLifecycle(v1.MemoryCandidate)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("candidates = %d err = %v, want 1", len(candidates), err)
	}

	if _, err := p.Promote(id, v1.MemoryActive); err == nil {
		t.Fatal("expected explicit CANDIDATE -> ACTIVE to be rejected")
	}
	if _, err := p.Promote(id, v1.MemoryValidated); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Promote(id, v1.MemoryActive); err != nil {
		t.Fatal(err)
	}
	active, err := records.ListByLifecycle(v1.MemoryActive)
	if err != nil || len(active) != 1 {
		t.Fatalf("active = %d err = %v, want 1", len(active), err)
	}
}
