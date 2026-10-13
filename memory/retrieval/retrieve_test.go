package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func retrieveSetup(t *testing.T) *Query {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	records := []struct {
		kind string
		id   string
		data string
	}{
		{"strategy", "str_1", `{"kind":"strategy","id":"str_1","title":"always write tests"}`},
		{"strategy", "str_2", `{"kind":"strategy","id":"str_2","title":"write docs"}`},
		{"issue", "iss_1", `{"kind":"issue","id":"iss_1","title":"write tests for retrieval"}`},
		{"memory_record", "mem_1", `{"kind":"memory_record","id":"mem_1","summary":"deterministic ranking"}`},
	}
	for _, r := range records {
		rec, err := store.NewRecord(r.kind, r.id, 1, []byte(r.data))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.PutRecord(rec); err != nil {
			t.Fatal(err)
		}
	}
	proj := projection.NewMem()
	if err := projection.Rebuild(proj, s); err != nil {
		t.Fatal(err)
	}
	return New(proj, s)
}

func ids(results []Result) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.Candidate.Kind + "/" + r.Candidate.ID
	}
	return out
}

func TestRetrieveIsOrderIndependentAndDeterministic(t *testing.T) {
	q := retrieveSetup(t)
	ctx := context.Background()

	a, err := q.Retrieve(ctx, Request{Kinds: []string{"strategy", "issue"}, Terms: []string{"write", "tests"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := q.Retrieve(ctx, Request{Kinds: []string{"issue", "strategy"}, Terms: []string{"tests", "write"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids(a), ids(b)) {
		t.Fatalf("order-dependent results: %v vs %v", ids(a), ids(b))
	}
	// Expected: equal scores tie-break by kind then id.
	want := []string{"issue/iss_1", "strategy/str_1", "strategy/str_2"}
	if !reflect.DeepEqual(ids(a), want) {
		t.Fatalf("ranked ids = %v, want %v", ids(a), want)
	}
	if a[0].Rank != 1 || a[1].Rank != 2 {
		t.Fatalf("ranks not sequential: %+v", a)
	}
}

func TestRetrieveProvenanceAndText(t *testing.T) {
	q := retrieveSetup(t)
	results, err := q.Retrieve(context.Background(), Request{Kinds: []string{"memory_record"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results", len(results))
	}
	c := results[0].Candidate
	if c.Text != "deterministic ranking" {
		t.Fatalf("text = %q", c.Text)
	}
	if c.Provenance.Source != "projection" || c.Provenance.Kind != "memory_record" || c.Provenance.ID != "mem_1" {
		t.Fatalf("provenance = %+v", c.Provenance)
	}
	if c.Provenance.Revision != 1 || c.Provenance.Hash == "" || c.Hash == "" {
		t.Fatalf("missing revision/hash: %+v", c)
	}
}

func TestRetrieveFilterAndLimit(t *testing.T) {
	q := retrieveSetup(t)
	results, err := q.Retrieve(context.Background(), Request{
		Terms:  []string{"write", "tests"},
		Filter: func(c Candidate) bool { return c.Kind == "strategy" },
		Limit:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Candidate.ID != "str_1" || results[0].Rank != 1 {
		t.Fatalf("filtered/limited = %v", ids(results))
	}
}

func TestRetrieveSearchAllKindsByDefault(t *testing.T) {
	q := retrieveSetup(t)
	results, err := q.Retrieve(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 candidates across all kinds, got %d", len(results))
	}
}

func TestRetrieveTypedErrorsAndMalformedData(t *testing.T) {
	q := retrieveSetup(t)
	ctx := context.Background()

	if _, err := q.Retrieve(ctx, Request{Limit: -1}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("negative limit: %v", err)
	}
	if _, err := q.Retrieve(ctx, Request{Limit: MaxLimit + 1}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("over-max limit: %v", err)
	}
	if _, err := q.Retrieve(ctx, Request{Kinds: []string{""}}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("empty kind: %v", err)
	}
	// Malformed record data never panics: candidateText returns "" (see
	// TestCandidateTextIgnoresUnknownShape).
}

func TestQueryImplementsRetriever(t *testing.T) {
	var r Retriever = retrieveSetup(t)
	if _, err := r.Retrieve(context.Background(), Request{Kinds: []string{"strategy"}}); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateTextIgnoresUnknownShape(t *testing.T) {
	if got := candidateText(json.RawMessage(`{"n":1}`)); got != "" {
		t.Fatalf("text = %q", got)
	}
	if got := candidateText(nil); got != "" {
		t.Fatalf("nil text = %q", got)
	}
}
