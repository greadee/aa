package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Provenance identifies where a candidate came from so consumers can attribute
// and re-derive it.
type Provenance struct {
	Source   string `json:"source"`
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Revision int    `json:"revision"`
	Hash     string `json:"hash"`
}

// Candidate is a retrieved item's information, before any context assembly.
type Candidate struct {
	Kind       string     `json:"kind"`
	ID         string     `json:"id"`
	Revision   int        `json:"revision"`
	Hash       string     `json:"hash"`
	Text       string     `json:"text"`
	Provenance Provenance `json:"provenance"`
}

// Result is a ranked candidate. Rank is 1-based over the returned set.
type Result struct {
	Candidate Candidate `json:"candidate"`
	Score     int       `json:"score"`
	Rank      int       `json:"rank"`
}

// Request is a retrieval request. It is order-independent: Results do not depend
// on the order of Kinds or Terms, or on the wall clock.
type Request struct {
	// Kinds restricts the search. An empty list searches every projected kind.
	Kinds []string
	// Terms are case-insensitive substrings scored against a candidate's text.
	// An empty list scores every candidate 0.
	Terms []string
	// Filter, when set, drops candidates before ranking.
	Filter func(Candidate) bool
	// Limit caps the number of results. Zero means DefaultLimit; negative is invalid.
	Limit int
}

// DefaultLimit and MaxLimit bound a retrieval request.
const (
	DefaultLimit = 100
	MaxLimit     = 10000
)

// Typed errors. Callers branch with errors.Is; malformed records never panic.
var ErrInvalidRequest = errors.New("retrieval: invalid request")

// Retriever is the stable retrieval interface. It returns candidate information
// only; assembling the final context window is the caller's job.
type Retriever interface {
	Retrieve(ctx context.Context, req Request) ([]Result, error)
}

// Retrieve returns deterministically ranked candidates.
//
// Ranking is by descending match score (the number of distinct Terms found in a
// candidate's text), then by ascending (Kind, ID). The result is independent of
// the order of Kinds and Terms and of wall-clock time.
func (q *Query) Retrieve(ctx context.Context, req Request) ([]Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit, err := resolveLimit(req.Limit)
	if err != nil {
		return nil, err
	}
	kinds, err := q.resolveKinds(req.Kinds)
	if err != nil {
		return nil, err
	}
	terms := normalizeTerms(req.Terms)

	var results []Result
	for _, kind := range kinds {
		for _, entry := range q.proj.List(kind) {
			candidate := Candidate{
				Kind:     entry.Kind,
				ID:       entry.ID,
				Revision: entry.Revision,
				Hash:     entry.Hash,
				Text:     candidateText(entry.Data),
				Provenance: Provenance{
					Source:   "projection",
					Kind:     entry.Kind,
					ID:       entry.ID,
					Revision: entry.Revision,
					Hash:     entry.Hash,
				},
			}
			if req.Filter != nil && !req.Filter(candidate) {
				continue
			}
			results = append(results, Result{Candidate: candidate, Score: score(candidate.Text, terms)})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Candidate.Kind != b.Candidate.Kind {
			return a.Candidate.Kind < b.Candidate.Kind
		}
		return a.Candidate.ID < b.Candidate.ID
	})
	if len(results) > limit {
		results = results[:limit]
	}
	for i := range results {
		results[i].Rank = i + 1
	}
	return results, nil
}

func resolveLimit(limit int) (int, error) {
	if limit < 0 {
		return 0, fmt.Errorf("%w: limit must not be negative", ErrInvalidRequest)
	}
	if limit == 0 {
		return DefaultLimit, nil
	}
	if limit > MaxLimit {
		return 0, fmt.Errorf("%w: limit %d exceeds max %d", ErrInvalidRequest, limit, MaxLimit)
	}
	return limit, nil
}

// resolveKinds deduplicates and sorts the requested kinds; empty means every
// projected kind.
func (q *Query) resolveKinds(kinds []string) ([]string, error) {
	if len(kinds) == 0 {
		return q.proj.Kinds(), nil
	}
	seen := make(map[string]struct{}, len(kinds))
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if k == "" {
			return nil, fmt.Errorf("%w: empty kind", ErrInvalidRequest)
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func normalizeTerms(terms []string) []string {
	seen := make(map[string]struct{}, len(terms))
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}
		if _, dup := seen[term]; dup {
			continue
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	sort.Strings(out)
	return out
}

func score(text string, terms []string) int {
	if len(terms) == 0 || text == "" {
		return 0
	}
	lower := strings.ToLower(text)
	total := 0
	for _, term := range terms {
		if strings.Contains(lower, term) {
			total++
		}
	}
	return total
}

// candidateText derives a deterministic snippet from a record. It reads a small
// allowlist of human-readable fields; malformed or absent data yields "".
func candidateText(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		return ""
	}
	for _, key := range []string{"text", "summary", "title", "name", "description", "body", "prompt"} {
		if value, ok := fields[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

var _ Retriever = (*Query)(nil)
