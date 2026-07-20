// Package query provides deterministic read access to aa-memory's projection
// and derives work, project, and job histories from the event log.
package query

import (
	"sort"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// Query reads from a projection and the canonical store.
type Query struct {
	proj  projection.Projection
	store *store.Store
}

// New returns a Query over the projection and store.
func New(p projection.Projection, s *store.Store) *Query {
	return &Query{proj: p, store: s}
}

// ByID returns the entry for a kind and id.
func (q *Query) ByID(kind, id string) (projection.Entry, bool) {
	return q.proj.Get(kind, id)
}

// ByKind returns all entries of a kind, sorted by id.
func (q *Query) ByKind(kind string) []projection.Entry {
	return q.proj.List(kind)
}

// Filter returns entries of a kind that satisfy keep, sorted by id.
func (q *Query) Filter(kind string, keep func(projection.Entry) bool) []projection.Entry {
	var out []projection.Entry
	for _, e := range q.proj.List(kind) {
		if keep(e) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Count returns the number of entries of a kind.
func (q *Query) Count(kind string) int {
	return len(q.proj.List(kind))
}

// Events returns the canonical event log in insertion order.
func (q *Query) Events() ([]store.EventRecord, error) {
	return q.store.Events()
}
