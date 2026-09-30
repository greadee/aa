package context

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/greadee/aa/memory/retrieval"
)

type fakeRetriever struct {
	results []retrieval.Result
	err     error
	got     retrieval.Request
}

func (f *fakeRetriever) Retrieve(_ context.Context, req retrieval.Request) ([]retrieval.Result, error) {
	f.got = req
	return f.results, f.err
}

func candidate(kind, id, text string) retrieval.Result {
	return retrieval.Result{Candidate: retrieval.Candidate{
		Kind: kind, ID: id, Text: text, Revision: 1, Hash: "h_" + id,
		Provenance: retrieval.Provenance{Source: "projection", Kind: kind, ID: id, Revision: 1, Hash: "h_" + id},
	}}
}

func TestDeriveNeedsIsDeterministic(t *testing.T) {
	a := DeriveNeeds("Fix the flaky retrieval test!", "wp_9")
	b := DeriveNeeds("Fix the flaky retrieval test!", "wp_9")
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("nondeterministic needs: %+v vs %+v", a, b)
	}
	if !reflect.DeepEqual(a.Kinds, DefaultKinds) {
		t.Fatalf("kinds = %v", a.Kinds)
	}
	// "fp"/"ok" style short words and duplicates are dropped; terms are sorted.
	want := []string{"fix", "flaky", "retrieval", "test", "the", "wp_9"}
	if !reflect.DeepEqual(a.Terms, want) {
		t.Fatalf("terms = %v, want %v", a.Terms, want)
	}
}

func TestCompileFromRetrievalPreservesProvenanceAndDigest(t *testing.T) {
	r := &fakeRetriever{results: []retrieval.Result{
		candidate("strategy", "str_1", "always write tests"),
		candidate("issue", "iss_1", "fix retrieval"),
	}}
	bundle, err := Compiler{MaxTokens: 1000}.CompileFromRetrieval(context.Background(), r, "prj", "wp_1", DeriveNeeds("fix retrieval", "wp_1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Sections) != 2 {
		t.Fatalf("sections = %d", len(bundle.Sections))
	}
	if bundle.Sections[0].Kind != "issue" {
		t.Fatalf("not sorted by kind: %+v", bundle.Sections)
	}
	for _, section := range bundle.Sections {
		if section.Provenance == nil || section.Provenance.Source != "projection" {
			t.Fatalf("missing provenance: %+v", section)
		}
	}
	if r.got.Limit != retrieval.DefaultLimit {
		t.Fatalf("limit not defaulted: %+v", r.got)
	}
}

func TestCompileFromRetrievalOverflowFailsClosed(t *testing.T) {
	long := ""
	for i := 0; i < 400; i++ {
		long += "x"
	}
	r := &fakeRetriever{results: []retrieval.Result{
		candidate("strategy", "str_1", long),
		candidate("issue", "iss_1", long),
	}}
	bundle, err := Compiler{MaxTokens: 50}.CompileFromRetrieval(context.Background(), r, "prj", "wp_1", DeriveNeeds("x", "wp_1"))
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}
	if !bundle.Truncated {
		t.Fatalf("bundle should report truncation: %+v", bundle)
	}
}

func TestCompileFromRetrievalPropagatesRetrieverError(t *testing.T) {
	r := &fakeRetriever{err: retrieval.ErrInvalidRequest}
	if _, err := (Compiler{}).CompileFromRetrieval(context.Background(), r, "prj", "wp", DeriveNeeds("x", "wp")); !errors.Is(err, retrieval.ErrInvalidRequest) {
		t.Fatalf("error not propagated: %v", err)
	}
}
