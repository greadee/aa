package repo

import (
	"errors"
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func newRecordRepo(t *testing.T) (*store.Store, *projection.MemProjection, *MemoryRecordRepository) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := projection.NewMem()
	return s, p, NewMemoryRecordRepository(s, p)
}

func record(id string) v1.MemoryRecord {
	return v1.MemoryRecord{
		Envelope: v1.Envelope{ID: id, ProjectID: "prj_1"},
		Level:    "project",
		Title:    "Prefer contracts first",
		Content:  v1.MemoryContent{Summary: "Define contracts before behavior."},
	}
}

func TestMemoryRecordProposeIsCandidateOnly(t *testing.T) {
	_, p, records := newRecordRepo(t)
	rec, err := records.Propose(record("mem_1"))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Kind != "memory_record" || rec.Lifecycle != v1.MemoryCandidate {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if rec.Revision == nil || *rec.Revision != 1 {
		t.Fatalf("revision = %v", rec.Revision)
	}
	if _, ok := p.Get("memory_record", rec.ID); !ok {
		t.Fatal("projection not updated")
	}
	list, err := records.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v err = %v", list, err)
	}
}

func TestMemoryRecordProposeRejectsPromoted(t *testing.T) {
	_, _, records := newRecordRepo(t)
	rec := record("mem_1")
	rec.Lifecycle = v1.MemoryActive
	if _, err := records.Propose(rec); err == nil {
		t.Fatal("expected non-CANDIDATE proposal to be rejected")
	}
}

func TestMemoryRecordTransitionStaysExplicit(t *testing.T) {
	_, _, records := newRecordRepo(t)
	rec, err := records.Propose(record("mem_1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := records.Transition(rec.ID, v1.MemoryActive); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("CANDIDATE -> ACTIVE err = %v, want ErrInvalidTransition", err)
	}
	if _, err := records.Transition(rec.ID, v1.MemoryValidated); err != nil {
		t.Fatal(err)
	}
	active, err := records.Transition(rec.ID, v1.MemoryActive)
	if err != nil {
		t.Fatal(err)
	}
	if active.Lifecycle != v1.MemoryActive {
		t.Fatalf("lifecycle = %s", active.Lifecycle)
	}
	validated, err := records.ListByLifecycle(v1.MemoryValidated)
	if err != nil || len(validated) != 0 {
		t.Fatalf("validated = %v err = %v", validated, err)
	}
	got, err := records.ListByLifecycle(v1.MemoryActive)
	if err != nil || len(got) != 1 {
		t.Fatalf("active = %v err = %v", got, err)
	}
}

func TestMemoryRecordProposeIsIdempotent(t *testing.T) {
	_, _, records := newRecordRepo(t)
	first, err := records.Propose(record("mem_1"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := records.Propose(record("mem_1"))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("ids differ: %s vs %s", first.ID, second.ID)
	}
	list, err := records.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v err = %v", list, err)
	}
}

func TestMemoryRecordSupersede(t *testing.T) {
	_, _, records := newRecordRepo(t)
	old, err := records.Propose(record("mem_old"))
	if err != nil {
		t.Fatal(err)
	}
	next, err := records.Propose(record("mem_new"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := records.Transition(old.ID, v1.MemoryValidated); err != nil {
		t.Fatal(err)
	}
	if _, err := records.Transition(old.ID, v1.MemoryActive); err != nil {
		t.Fatal(err)
	}
	if err := records.Supersede(old.ID, next.ID); err != nil {
		t.Fatal(err)
	}
	superseded, err := records.Get(old.ID)
	if err != nil || superseded.Lifecycle != v1.MemorySuperseded {
		t.Fatalf("old = %+v err = %v", superseded, err)
	}
	updated, err := records.Get(next.ID)
	if err != nil || updated.Supersedes != old.ID {
		t.Fatalf("new = %+v err = %v", updated, err)
	}
}
