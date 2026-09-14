// Package queue implements the offline change queue and reconciliation.
//
// Changes are ordered by their record revision, never by wall clock. When two
// replicas reconnect, Reconcile decides deterministically which changes to
// send and which to apply.
package queue

import (
	"fmt"
	"sort"

	aasync "github.com/greadee/aa/sync"
)

// Queue is an offline set of local changes, newest per path.
type Queue struct {
	changes map[string]aasync.Change
}

// New returns an empty queue.
func New() *Queue {
	return &Queue{changes: map[string]aasync.Change{}}
}

// Len returns the number of queued paths.
func (q *Queue) Len() int { return len(q.changes) }

// Enqueue adds a change. For a path, the higher revision wins; a tie is broken
// by node ID so the result is deterministic.
func (q *Queue) Enqueue(c aasync.Change) error {
	if c.Record.Path == "" {
		return fmt.Errorf("%w: path is required", aasync.ErrInvalid)
	}
	if existing, ok := q.changes[c.Record.Path]; ok {
		if !newer(c, existing) {
			return nil
		}
	}
	q.changes[c.Record.Path] = c
	return nil
}

func newer(a, b aasync.Change) bool {
	if a.Record.Revision != b.Record.Revision {
		return a.Record.Revision > b.Record.Revision
	}
	return a.Node > b.Node
}

// Pending returns the queued changes sorted by path.
func (q *Queue) Pending() []aasync.Change {
	out := make([]aasync.Change, 0, len(q.changes))
	for _, c := range q.changes {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Record.Path < out[j].Record.Path })
	return out
}

// Drain returns the pending changes and empties the queue.
func (q *Queue) Drain() []aasync.Change {
	out := q.Pending()
	q.changes = map[string]aasync.Change{}
	return out
}

// Conflict is a same-revision divergence between two replicas.
type Conflict struct {
	Path   string
	Local  aasync.Change
	Remote aasync.Change
}

// Decision is the outcome of reconciling two change sets.
type Decision struct {
	Send      []aasync.Change
	Apply     []aasync.Change
	Converged []string
	Conflicts []Conflict
}

// Reconcile compares local and remote changes and decides what each side
// should do. The higher revision wins; equal revisions with equal content
// converge; equal revisions with different content conflict.
func Reconcile(local, remote []aasync.Change) (Decision, error) {
	localBest, err := bestByPath(local)
	if err != nil {
		return Decision{}, err
	}
	remoteBest, err := bestByPath(remote)
	if err != nil {
		return Decision{}, err
	}

	paths := map[string]bool{}
	for p := range localBest {
		paths[p] = true
	}
	for p := range remoteBest {
		paths[p] = true
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	var d Decision
	for _, path := range sorted {
		lc, hasL := localBest[path]
		rc, hasR := remoteBest[path]
		switch {
		case hasL && !hasR:
			d.Send = append(d.Send, lc)
		case hasR && !hasL:
			d.Apply = append(d.Apply, rc)
		case lc.Record.Revision > rc.Record.Revision:
			d.Send = append(d.Send, lc)
		case rc.Record.Revision > lc.Record.Revision:
			d.Apply = append(d.Apply, rc)
		case same(lc, rc):
			d.Converged = append(d.Converged, path)
		default:
			d.Conflicts = append(d.Conflicts, Conflict{Path: path, Local: lc, Remote: rc})
		}
	}
	return d, nil
}

func bestByPath(changes []aasync.Change) (map[string]aasync.Change, error) {
	best := map[string]aasync.Change{}
	for _, c := range changes {
		if c.Record.Path == "" {
			return nil, fmt.Errorf("%w: path is required", aasync.ErrInvalid)
		}
		if existing, ok := best[c.Record.Path]; !ok || newer(c, existing) {
			best[c.Record.Path] = c
		}
	}
	return best, nil
}

func same(a, b aasync.Change) bool {
	return a.Record.Deleted == b.Record.Deleted && a.Record.Hash == b.Record.Hash
}
