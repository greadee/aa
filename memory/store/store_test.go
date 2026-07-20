package store

import (
	"errors"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestInitProjectIsIdempotent(t *testing.T) {
	s := openTemp(t)
	changed, err := s.InitProject([]byte(`{"kind":"project_record","id":"prj_1","name":"aa"}`))
	if err != nil || !changed {
		t.Fatalf("first InitProject changed=%v err=%v", changed, err)
	}
	changed, err = s.InitProject([]byte(`{"kind":"project_record","id":"prj_1","name":"aa"}`))
	if err != nil || changed {
		t.Fatalf("second InitProject changed=%v err=%v", changed, err)
	}
	manifest, err := s.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest) == 0 {
		t.Fatal("empty manifest")
	}
}

func TestPutGetRecordRoundTrip(t *testing.T) {
	s := openTemp(t)
	rec, err := NewRecord("issue", "iss_1", 1, []byte(`{"kind":"issue","id":"iss_1","title":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := s.PutRecord(rec)
	if err != nil || !changed {
		t.Fatalf("PutRecord changed=%v err=%v", changed, err)
	}
	got, err := s.GetRecord("issue", "iss_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != rec.Hash || got.Revision != 1 {
		t.Fatalf("got %+v, want %+v", got, rec)
	}
}

func TestPutRecordIdempotent(t *testing.T) {
	s := openTemp(t)
	rec, _ := NewRecord("issue", "iss_1", 1, []byte(`{"kind":"issue","id":"iss_1","title":"x"}`))
	if _, err := s.PutRecord(rec); err != nil {
		t.Fatal(err)
	}
	changed, err := s.PutRecord(rec)
	if err != nil || changed {
		t.Fatalf("expected idempotent no-op, changed=%v err=%v", changed, err)
	}
}

func TestPutRecordRejectsStaleRevision(t *testing.T) {
	s := openTemp(t)
	first, _ := NewRecord("issue", "iss_1", 2, []byte(`{"title":"first"}`))
	if _, err := s.PutRecord(first); err != nil {
		t.Fatal(err)
	}
	stale, _ := NewRecord("issue", "iss_1", 1, []byte(`{"title":"stale"}`))
	if _, err := s.PutRecord(stale); err == nil {
		t.Fatal("expected stale revision error")
	}
}

func TestGetMissingRecord(t *testing.T) {
	s := openTemp(t)
	if _, err := s.GetRecord("issue", "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListRecordsSorted(t *testing.T) {
	s := openTemp(t)
	for _, id := range []string{"iss_3", "iss_1", "iss_2"} {
		rec, _ := NewRecord("issue", id, 1, []byte(`{"title":"x"}`))
		if _, err := s.PutRecord(rec); err != nil {
			t.Fatal(err)
		}
	}
	records, err := s.ListRecords("issue")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 || records[0].ID != "iss_1" || records[2].ID != "iss_3" {
		t.Fatalf("unexpected order: %+v", records)
	}
}

func TestAppendEventDedupes(t *testing.T) {
	s := openTemp(t)
	event := []byte(`{"kind":"event","id":"evt_1","sequence":1,"type":"PROJECT_REGISTERED"}`)
	changed, err := s.AppendEvent(event)
	if err != nil || !changed {
		t.Fatalf("first append changed=%v err=%v", changed, err)
	}
	changed, err = s.AppendEvent(event)
	if err != nil || changed {
		t.Fatalf("duplicate append changed=%v err=%v", changed, err)
	}
	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "evt_1" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestListAllRecords(t *testing.T) {
	s := openTemp(t)
	a, _ := NewRecord("issue", "iss_1", 1, []byte(`{"a":1}`))
	b, _ := NewRecord("strategy", "str_1", 1, []byte(`{"b":2}`))
	if _, err := s.PutRecord(a); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PutRecord(b); err != nil {
		t.Fatal(err)
	}
	all, err := s.ListAllRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Kind != "issue" || all[1].Kind != "strategy" {
		t.Fatalf("unexpected records: %+v", all)
	}
}
