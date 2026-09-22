package tracesource

import (
	"testing"
	"time"

	v1 "github.com/greadee/aa/contracts/go/v1"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/distill"
	"github.com/greadee/aa/kernel/joblearn/features"
	"github.com/greadee/aa/kernel/joblearn/promote"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/repo"
	"github.com/greadee/aa/memory/store"
)

func fixture(id, attempt string) v1.Trace {
	return v1.Trace{
		Envelope:      v1.Envelope{ContractVersion: v1.Version, Kind: "trace", ID: id},
		AttemptID:     attempt,
		WorkPackageID: "wp_1",
		Steps: []v1.TraceStep{
			{Sequence: intPtr(0), Phase: v1.TracePlan, Outcome: v1.TraceSucceeded},
			{Sequence: intPtr(1), Phase: v1.TraceTest, Outcome: v1.TraceSucceeded},
		},
	}
}

func TestTraceStoreToPromotionEndToEnd(t *testing.T) {
	memoryStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	traces := repo.NewTraceRepository(memoryStore, projection.NewMem())
	for _, trace := range []v1.Trace{fixture("trc_1", "att_1"), fixture("trc_2", "att_2"), fixture("trc_3", "att_3")} {
		if _, _, err := traces.Ingest(trace); err != nil {
			t.Fatalf("Ingest: %v", err)
		}
	}

	src, err := NewMemorySource(traces)
	if err != nil {
		t.Fatalf("NewMemorySource: %v", err)
	}
	loaded, err := Load(src, DefaultBounds())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected three traces, got %d", len(loaded))
	}

	meta := attribution.Meta{ProjectID: "prj_1", Role: "engineer", Trade: "backend"}
	results, err := attribution.DeriveTraces(loaded, func(v1.Trace) attribution.Meta { return meta }, attribution.DefaultLimits())
	if err != nil {
		t.Fatalf("DeriveTraces: %v", err)
	}
	records := make([]distill.Record, len(loaded))
	for i, trace := range loaded {
		records[i] = distill.Record{Result: results[i], Set: features.Extract(trace)}
	}

	distiller, err := distill.New(distill.DefaultPolicy())
	if err != nil {
		t.Fatalf("distill.New: %v", err)
	}
	candidates, err := distiller.Distill(records)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("expected at least one distilled candidate")
	}

	memoryRecords := repo.NewMemoryRecordRepository(memoryStore, projection.NewMem())
	sink, err := promote.NewMemorySink(memoryRecords)
	if err != nil {
		t.Fatalf("NewMemorySink: %v", err)
	}
	promoter, err := promote.New(sink, promote.Options{
		ProjectID: "prj_1",
		Now:       func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("promote.New: %v", err)
	}
	if _, err := promoter.Propose(candidates); err != nil {
		t.Fatalf("Propose: %v", err)
	}

	stored, err := memoryRecords.ListByLifecycle(v1.MemoryCandidate)
	if err != nil {
		t.Fatalf("ListByLifecycle: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("expected candidate-only records in memory")
	}
	for _, record := range stored {
		if record.Lifecycle != v1.MemoryCandidate {
			t.Fatalf("record %s is %s, want CANDIDATE", record.ID, record.Lifecycle)
		}
	}

	// Candidates with the same identity re-propose idempotently.
	if _, err := promoter.Propose(candidates); err != nil {
		t.Fatalf("re-Propose: %v", err)
	}
	again, _ := memoryRecords.ListByLifecycle(v1.MemoryCandidate)
	if len(again) != len(stored) {
		t.Fatalf("re-proposing changed the record count: %d vs %d", len(again), len(stored))
	}
}

func TestMemorySourceRequiresRepository(t *testing.T) {
	if _, err := NewMemorySource(nil); err == nil {
		t.Fatal("expected an error for a nil repository")
	}
}
