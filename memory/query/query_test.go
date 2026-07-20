package query

import (
	"testing"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func setup(t *testing.T) *Query {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	events := []string{
		`{"id":"evt_1","sequence":1,"type":"WORK_PACKAGE_CREATED","occurredAt":"2026-01-01T00:00:00Z","aggregate":{"kind":"work_package","id":"wp_1"}}`,
		`{"id":"evt_2","sequence":2,"type":"WORK_PACKAGE_READY","occurredAt":"2026-01-01T00:01:00Z","aggregate":{"kind":"work_package","id":"wp_1"}}`,
		`{"id":"evt_3","sequence":3,"type":"EXECUTION_STARTED","occurredAt":"2026-01-01T00:02:00Z","aggregate":{"kind":"assignment","id":"asg_1"},"payload":{"workPackageId":"wp_1"}}`,
		`{"id":"evt_4","sequence":4,"type":"EXECUTION_COMPLETED","occurredAt":"2026-01-01T00:03:00Z","aggregate":{"kind":"assignment","id":"asg_1"},"payload":{"workPackageId":"wp_1"}}`,
		`{"id":"evt_5","sequence":5,"type":"WORK_PACKAGE_COMPLETED","occurredAt":"2026-01-01T00:04:00Z","aggregate":{"kind":"work_package","id":"wp_1"}}`,
	}
	for _, e := range events {
		if _, err := s.AppendEvent([]byte(e)); err != nil {
			t.Fatal(err)
		}
	}
	proj := projection.NewMem()
	if err := projection.Rebuild(proj, s); err != nil {
		t.Fatal(err)
	}
	return New(proj, s)
}

func TestWorkHistory(t *testing.T) {
	q := setup(t)
	history, err := q.WorkHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(history))
	}
	e := history[0]
	if e.WorkPackageID != "wp_1" || e.State != "completed" {
		t.Fatalf("unexpected entry: %+v", e)
	}
	if e.CreatedAt == "" || e.ReadyAt == "" || e.CompletedAt == "" {
		t.Fatalf("missing timestamps: %+v", e)
	}
}

func TestJobHistory(t *testing.T) {
	q := setup(t)
	jobs, err := q.JobHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	j := jobs[0]
	if j.AssignmentID != "asg_1" || j.WorkPackageID != "wp_1" || j.Outcome != "succeeded" {
		t.Fatalf("unexpected job: %+v", j)
	}
	if j.StartedAt == "" || j.FinishedAt == "" {
		t.Fatalf("missing span: %+v", j)
	}
}

func TestFilterAndCount(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := store.NewRecord("issue", "iss_1", 1, []byte(`{"kind":"issue","id":"iss_1","status":"open"}`))
	if _, err := s.PutRecord(rec); err != nil {
		t.Fatal(err)
	}
	proj := projection.NewMem()
	if err := projection.Rebuild(proj, s); err != nil {
		t.Fatal(err)
	}
	q := New(proj, s)
	if q.Count("issue") != 1 {
		t.Fatalf("Count = %d", q.Count("issue"))
	}
	got := q.Filter("issue", func(e projection.Entry) bool { return e.ID == "iss_1" })
	if len(got) != 1 {
		t.Fatalf("Filter returned %d", len(got))
	}
}
