package projection

import (
	"encoding/json"
	"testing"

	"github.com/greadee/aa/memory/store"
)

func put(t *testing.T, s *store.Store, kind, id string, data string) {
	t.Helper()
	rec, err := store.NewRecord(kind, id, 1, []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PutRecord(rec); err != nil {
		t.Fatal(err)
	}
}

func TestPutGetList(t *testing.T) {
	p := NewMem()
	if err := p.Put(Entry{Kind: "issue", ID: "iss_2", Revision: 1, Hash: "h2", Data: json.RawMessage(`{"x":2}`)}); err != nil {
		t.Fatal(err)
	}
	if err := p.Put(Entry{Kind: "issue", ID: "iss_1", Revision: 1, Hash: "h1", Data: json.RawMessage(`{"x":1}`)}); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Get("issue", "iss_1"); !ok {
		t.Fatal("expected iss_1")
	}
	list := p.List("issue")
	if len(list) != 2 || list[0].ID != "iss_1" {
		t.Fatalf("unexpected list: %+v", list)
	}
	if p.Len() != 2 {
		t.Fatalf("Len = %d", p.Len())
	}
}

func TestRebuildIsEquivalent(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	put(t, s, "issue", "iss_1", `{"title":"one"}`)
	put(t, s, "strategy", "str_1", `{"title":"two"}`)

	incremental := NewMem()
	records, err := s.ListAllRecords()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		if err := incremental.Put(FromRecord(r)); err != nil {
			t.Fatal(err)
		}
	}

	rebuilt := NewMem()
	if err := Rebuild(rebuilt, s); err != nil {
		t.Fatal(err)
	}
	if incremental.Digest() != rebuilt.Digest() {
		t.Fatalf("digests differ: %s vs %s", incremental.Digest(), rebuilt.Digest())
	}

	again := NewMem()
	if err := Rebuild(again, s); err != nil {
		t.Fatal(err)
	}
	if again.Digest() != rebuilt.Digest() {
		t.Fatalf("rebuild not stable: %s vs %s", again.Digest(), rebuilt.Digest())
	}
}

func TestDigestChangesWithContent(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	put(t, s, "issue", "iss_1", `{"title":"one"}`)
	a := NewMem()
	if err := Rebuild(a, s); err != nil {
		t.Fatal(err)
	}
	put(t, s, "issue", "iss_2", `{"title":"two"}`)
	b := NewMem()
	if err := Rebuild(b, s); err != nil {
		t.Fatal(err)
	}
	if a.Digest() == b.Digest() {
		t.Fatal("digest should change when content changes")
	}
}

func TestPutRejectsStaleRevision(t *testing.T) {
	p := NewMem()
	if err := p.Put(Entry{Kind: "issue", ID: "iss_1", Revision: 2, Hash: "h2"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Put(Entry{Kind: "issue", ID: "iss_1", Revision: 1, Hash: "h1"}); err == nil {
		t.Fatal("expected stale revision error")
	}
}

func TestPutIdempotentForSameHash(t *testing.T) {
	p := NewMem()
	e := Entry{Kind: "issue", ID: "iss_1", Revision: 1, Hash: "h1"}
	if err := p.Put(e); err != nil {
		t.Fatal(err)
	}
	if err := p.Put(e); err != nil {
		t.Fatal(err)
	}
	if p.Len() != 1 {
		t.Fatalf("Len = %d", p.Len())
	}
}
