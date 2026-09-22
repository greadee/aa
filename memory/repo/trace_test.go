package repo

import (
	"errors"
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func newTraceRepo(t *testing.T) (*store.Store, *projection.MemProjection, *TraceRepository) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := projection.NewMem()
	return s, p, NewTraceRepository(s, p)
}

func step(seq int, phase v1.TracePhase, outcome v1.TraceOutcome) v1.TraceStep {
	return v1.TraceStep{Sequence: &seq, Phase: phase, Outcome: outcome}
}

func trace(id, attempt string) v1.Trace {
	return v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: id, ProjectID: "prj_1"},
		AttemptID: attempt,
		Steps:     []v1.TraceStep{step(0, v1.TraceObserve, v1.TraceSucceeded)},
	}
}

func TestTraceIngestAssignsDefaultsAndProjects(t *testing.T) {
	_, p, traces := newTraceRepo(t)
	in := trace("", "att_1")
	in.ID = ""
	got, changed, err := traces.Ingest(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected first ingest to change the store")
	}
	if got.ID == "" || got.Kind != "trace" {
		t.Fatalf("unexpected trace: %+v", got)
	}
	if got.Revision == nil || *got.Revision != store.Revision {
		t.Fatalf("revision = %v", got.Revision)
	}
	if _, ok := p.Get("trace", got.ID); !ok {
		t.Fatal("projection not updated")
	}
}

func TestTraceIngestIsIdempotent(t *testing.T) {
	_, _, traces := newTraceRepo(t)
	in := trace("trc_1", "att_1")
	if _, changed, err := traces.Ingest(in); err != nil || !changed {
		t.Fatalf("first ingest changed=%v err=%v", changed, err)
	}
	if _, changed, err := traces.Ingest(in); err != nil || changed {
		t.Fatalf("second ingest changed=%v err=%v, want no-op", changed, err)
	}
}

func TestTraceIngestSupersedesAndRejectsStale(t *testing.T) {
	_, _, traces := newTraceRepo(t)
	if _, _, err := traces.Ingest(trace("trc_1", "att_1")); err != nil {
		t.Fatal(err)
	}
	next := trace("trc_1", "att_1")
	rev := 2
	next.Revision = &rev
	next.Steps = append(next.Steps, step(1, v1.TraceTest, v1.TraceFailed))
	if _, changed, err := traces.Ingest(next); err != nil || !changed {
		t.Fatalf("supersede changed=%v err=%v", changed, err)
	}
	stale := trace("trc_1", "att_1")
	staleRev := 1
	stale.Revision = &staleRev
	stale.Steps = append(stale.Steps, step(2, v1.TraceVerify, v1.TraceSucceeded))
	if _, _, err := traces.Ingest(stale); err == nil {
		t.Fatal("expected stale revision to be rejected")
	}
}

func TestTraceIngestRejectsInvalid(t *testing.T) {
	_, _, traces := newTraceRepo(t)
	missingAttempt := trace("trc_1", "")
	if _, _, err := traces.Ingest(missingAttempt); err == nil {
		t.Fatal("expected missing attemptId to be rejected")
	}
	noSteps := trace("trc_2", "att_1")
	noSteps.Steps = nil
	if _, _, err := traces.Ingest(noSteps); err == nil {
		t.Fatal("expected empty steps to be rejected")
	}
}

func TestTraceListByAttemptAndProject(t *testing.T) {
	_, _, traces := newTraceRepo(t)
	for _, id := range []string{"trc_1", "trc_2"} {
		if _, _, err := traces.Ingest(trace(id, "att_1")); err != nil {
			t.Fatal(err)
		}
	}
	other := trace("trc_3", "att_2")
	other.ProjectID = "prj_2"
	if _, _, err := traces.Ingest(other); err != nil {
		t.Fatal(err)
	}

	byAttempt, err := traces.ListByAttempt("att_1")
	if err != nil || len(byAttempt) != 2 {
		t.Fatalf("byAttempt = %d err=%v, want 2", len(byAttempt), err)
	}
	byProject, err := traces.ListByProject("prj_1")
	if err != nil || len(byProject) != 2 {
		t.Fatalf("byProject = %d err=%v, want 2", len(byProject), err)
	}
	if _, err := traces.ListByAttempt(""); err == nil {
		t.Fatal("expected invalid attemptId to be rejected")
	}
}

func TestTraceGetMissing(t *testing.T) {
	_, _, traces := newTraceRepo(t)
	if _, err := traces.Get("missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
