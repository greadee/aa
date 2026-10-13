package context

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode"

	"github.com/greadee/aa/memory/retrieval"
)

// DefaultKinds is the kind precedence used when a need does not name kinds.
var DefaultKinds = []string{"strategy", "issue", "memory_record", "trace"}

// Needs describes what context to gather. It is discovered deterministically
// from the task, never from a model.
type Needs struct {
	Kinds []string
	Terms []string
	Limit int
}

// ErrBudgetExceeded is returned (fail closed) when the selected candidates do
// not fit the compiler's token budget. The partial bundle is returned with it.
var ErrBudgetExceeded = errors.New("context: token budget exceeded")

// DeriveNeeds extracts deterministic context needs from an objective and work
// package id: kinds default to DefaultKinds; terms are the distinct lowercased
// words of length >= 3 (plus the work package id), sorted. It is pure and
// order-independent.
func DeriveNeeds(objective, workPackageID string) Needs {
	seen := map[string]struct{}{}
	var terms []string
	add := func(raw string) {
		term := strings.ToLower(strings.TrimSpace(raw))
		if len([]rune(term)) < 3 {
			return
		}
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	add(workPackageID)
	for _, word := range strings.FieldsFunc(objective, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		add(word)
	}
	sort.Strings(terms)
	return Needs{Kinds: append([]string(nil), DefaultKinds...), Terms: terms, Limit: retrieval.DefaultLimit}
}

// CompileFromRetrieval selects candidates through the retrieval interface and
// assembles a bounded bundle. Selection and assembly are deterministic. If the
// selected candidates do not fit the budget, the bundle is truncated and
// ErrBudgetExceeded is returned (fail closed) with the partial bundle.
func (c Compiler) CompileFromRetrieval(ctx context.Context, r retrieval.Retriever, projectID, workPackageID string, needs Needs) (Bundle, error) {
	if r == nil {
		return Bundle{}, errors.New("context: retriever is required")
	}
	limit := needs.Limit
	if limit == 0 {
		limit = retrieval.DefaultLimit
	}
	results, err := r.Retrieve(ctx, retrieval.Request{Kinds: needs.Kinds, Terms: needs.Terms, Limit: limit})
	if err != nil {
		return Bundle{}, err
	}
	inputs := make([]Input, 0, len(results))
	for _, result := range results {
		provenance := result.Candidate.Provenance
		inputs = append(inputs, Input{
			Kind:       result.Candidate.Kind,
			ID:         result.Candidate.ID,
			Text:       result.Candidate.Text,
			Provenance: &provenance,
		})
	}
	bundle := c.Compile(projectID, workPackageID, inputs)
	if bundle.Truncated {
		return bundle, ErrBudgetExceeded
	}
	return bundle, nil
}
