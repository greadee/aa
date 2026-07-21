package store

import "testing"

func TestRetentionExcludesBelowFloor(t *testing.T) {
	s := openTemp(t)
	for i := 1; i <= 3; i++ {
		event := []byte(`{"id":"evt_` + string(rune('0'+i)) + `","sequence":` + string(rune('0'+i)) + `}`)
		if _, err := s.AppendEvent(event); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RetainAfter(2); err != nil {
		t.Fatal(err)
	}
	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sequence != 3 {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestRetentionIsMonotonic(t *testing.T) {
	s := openTemp(t)
	if err := s.RetainAfter(5); err != nil {
		t.Fatal(err)
	}
	if err := s.RetainAfter(3); err == nil {
		t.Fatal("expected monotonic floor error")
	}
	if err := s.RetainAfter(5); err != nil {
		t.Fatalf("same floor should be a no-op: %v", err)
	}
}

func TestIngestRecordsIdempotent(t *testing.T) {
	s := openTemp(t)
	rec, _ := NewRecord("issue", "iss_1", 1, []byte(`{"title":"x"}`))
	first, err := s.IngestRecords([]Record{rec}, IngestOptions{})
	if err != nil || first.New != 1 || first.Unchanged != 0 {
		t.Fatalf("first ingest = %+v err=%v", first, err)
	}
	second, err := s.IngestRecords([]Record{rec}, IngestOptions{})
	if err != nil || second.New != 0 || second.Unchanged != 1 {
		t.Fatalf("second ingest = %+v err=%v", second, err)
	}
}

func TestIngestEventsIdempotent(t *testing.T) {
	s := openTemp(t)
	event := []byte(`{"id":"evt_1","sequence":1}`)
	first, err := s.IngestEvent(event)
	if err != nil || first.New != 1 {
		t.Fatalf("first = %+v err=%v", first, err)
	}
	second, err := s.IngestEvent(event)
	if err != nil || second.Unchanged != 1 {
		t.Fatalf("second = %+v err=%v", second, err)
	}
}

func TestRequireProvenance(t *testing.T) {
	without := []byte(`{"title":"x"}`)
	if err := ValidateProvenance(without); err != nil {
		t.Fatalf("absent provenance should validate: %v", err)
	}
	if err := RequireProvenance(without); err == nil {
		t.Fatal("expected required provenance error")
	}
	good := []byte(`{"title":"x","provenance":{"source":"kernel","producedAt":"2026-01-01T00:00:00Z"}}`)
	if err := RequireProvenance(good); err != nil {
		t.Fatalf("good provenance: %v", err)
	}
	bad := []byte(`{"title":"x","provenance":{"source":"kernel","producedAt":"nope"}}`)
	if err := RequireProvenance(bad); err == nil {
		t.Fatal("expected bad timestamp error")
	}
}

func TestIngestRequiresProvenance(t *testing.T) {
	s := openTemp(t)
	rec, _ := NewRecord("memory_record", "mem_1", 1, []byte(`{"title":"x"}`))
	if _, err := s.IngestRecords([]Record{rec}, IngestOptions{RequireProvenance: true}); err == nil {
		t.Fatal("expected provenance requirement to reject")
	}
}
