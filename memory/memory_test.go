package memory

import (
	"testing"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

func TestOpenAndRebuildEquivalence(t *testing.T) {
	m, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := m.InitProject([]byte(`{"kind":"project_record","id":"prj_1","name":"aa"}`)); err != nil || !changed {
		t.Fatalf("InitProject changed=%v err=%v", changed, err)
	}
	issue, err := m.Issues().Create(v1.Issue{
		Envelope: v1.Envelope{ID: "iss_1"},
		Title:    "Contract spine",
		Type:     "feature",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.AppendEvent([]byte(`{"id":"evt_1","sequence":1,"type":"WORK_PACKAGE_CREATED","occurredAt":"2026-01-01T00:00:00Z","aggregate":{"kind":"work_package","id":"wp_1"}}`)); err != nil {
		t.Fatal(err)
	}

	before := m.Projection().Digest()
	if err := m.Rebuild(); err != nil {
		t.Fatal(err)
	}
	after := m.Projection().Digest()
	if before != after {
		t.Fatalf("digest changed across rebuild: %s vs %s", before, after)
	}

	got, ok := m.Query().ByID("issue", issue.ID)
	if !ok {
		t.Fatal("issue not found in projection")
	}
	if got.ID != issue.ID {
		t.Fatalf("got %s, want %s", got.ID, issue.ID)
	}
	history, err := m.Query().WorkHistory()
	if err != nil || len(history) != 1 {
		t.Fatalf("history = %v err=%v", history, err)
	}
}

func TestReopenPersistsRecords(t *testing.T) {
	dir := t.TempDir()
	m, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Issues().Create(v1.Issue{
		Envelope: v1.Envelope{ID: "iss_1"},
		Title:    "x",
		Type:     "bug",
	}); err != nil {
		t.Fatal(err)
	}
	m2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m2.Issues().Get("iss_1"); err != nil {
		t.Fatalf("expected persisted issue: %v", err)
	}
}

func TestTraceStoreRebuildEquivalence(t *testing.T) {
	dir := t.TempDir()
	m, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	seq0, seq1 := 0, 1
	_, changed, err := m.Traces().Ingest(v1.Trace{
		Envelope:  v1.Envelope{ContractVersion: "1.1", ID: "trc_1", ProjectID: "prj_1"},
		AttemptID: "att_1",
		Steps: []v1.TraceStep{
			{Sequence: &seq0, Phase: v1.TraceObserve, Outcome: v1.TraceSucceeded},
			{Sequence: &seq1, Phase: v1.TraceTest, Outcome: v1.TraceFailed},
		},
	})
	if err != nil || !changed {
		t.Fatalf("ingest changed=%v err=%v", changed, err)
	}
	summaries, err := m.Query().TraceSummaries()
	if err != nil || len(summaries) != 1 || summaries[0].Outcome != "failed" {
		t.Fatalf("summaries=%+v err=%v", summaries, err)
	}
	before := m.Projection().Digest()
	if err := m.Rebuild(); err != nil {
		t.Fatal(err)
	}
	if after := m.Projection().Digest(); after != before {
		t.Fatalf("digest changed across rebuild: %s vs %s", before, after)
	}
	m2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m2.Traces().Get("trc_1"); err != nil {
		t.Fatalf("expected persisted trace: %v", err)
	}
}
