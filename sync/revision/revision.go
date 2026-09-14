// Package revision implements one-way revision sync with tombstones and
// deletion guards.
//
// A replica is a set of path -> latest record. The source is authoritative:
// absent paths are never deleted implicitly, and deletions happen only through
// an explicit tombstone whose revision is not older than the replica's copy.
package revision

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	aasync "github.com/greadee/aa/sync"
)

// Set is one replica's view of a tree.
type Set struct {
	records map[string]aasync.RevisionRecord
}

// NewSet returns an empty replica.
func NewSet() *Set {
	return &Set{records: map[string]aasync.RevisionRecord{}}
}

// Put records a revision for a path. An older revision never overwrites a
// newer one.
func (s *Set) Put(rec aasync.RevisionRecord) error {
	if rec.Path == "" {
		return fmt.Errorf("%w: path is required", aasync.ErrInvalid)
	}
	if existing, ok := s.records[rec.Path]; ok && existing.Revision > rec.Revision {
		return fmt.Errorf("%w: path %q has revision %d newer than %d", aasync.ErrConflict, rec.Path, existing.Revision, rec.Revision)
	}
	s.records[rec.Path] = rec
	return nil
}

// Get returns the record for a path.
func (s *Set) Get(path string) (aasync.RevisionRecord, bool) {
	rec, ok := s.records[path]
	return rec, ok
}

// Records returns the records sorted by path.
func (s *Set) Records() []aasync.RevisionRecord {
	out := make([]aasync.RevisionRecord, 0, len(s.records))
	for _, rec := range s.records {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Digest returns a stable digest of the replica contents.
func (s *Set) Digest() string {
	var b strings.Builder
	for _, rec := range s.Records() {
		fmt.Fprintf(&b, "%s|%d|%s|%t\n", rec.Path, rec.Revision, rec.Hash, rec.Deleted)
	}
	return aasync.Hash([]byte(b.String()))
}

// Diff returns the ordered operations that bring replica up to source.
//
// Only source records newer than the replica's are emitted, including
// tombstones. A path that is missing from source is left untouched: deletion
// is explicit and guarded.
func Diff(source, replica *Set) []Op {
	if source == nil {
		return nil
	}
	var paths []string
	for path := range source.records {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var ops []Op
	for _, path := range paths {
		src := source.records[path]
		local, ok := replica.Get(path)
		if ok && local.Revision >= src.Revision && local.Deleted == src.Deleted {
			continue
		}
		if ok && local.Revision > src.Revision {
			continue
		}
		kind := OpPut
		if src.Deleted {
			kind = OpDelete
		}
		ops = append(ops, Op{Kind: kind, Record: src})
	}
	return ops
}

// Op is one revision operation.
type Op = aasync.Op

// Revision operation kinds.
const (
	OpPut    = aasync.OpPut
	OpDelete = aasync.OpDelete
)

// Refusal records an operation the deletion guard rejected.
type Refusal struct {
	Path   string
	Reason string
}

// Result reports what an Apply did.
type Result struct {
	Applied int
	Refused []Refusal
}

// Apply applies ops to replica in order. A delete is refused when the replica
// holds a newer revision than the tombstone (the local copy would be lost).
func Apply(replica *Set, ops []Op) (Result, error) {
	var res Result
	for _, op := range ops {
		local, ok := replica.Get(op.Record.Path)
		if ok && local.Revision == op.Record.Revision && local.Hash == op.Record.Hash && local.Deleted == op.Record.Deleted {
			continue // already converged
		}
		if op.Kind == OpDelete && ok && local.Revision > op.Record.Revision {
			res.Refused = append(res.Refused, Refusal{Path: op.Record.Path, Reason: "replica revision is newer"})
			continue
		}
		if op.Kind == OpDelete && ok && local.Deleted {
			continue
		}
		if err := replica.Put(op.Record); err != nil {
			if errors.Is(err, aasync.ErrConflict) {
				res.Refused = append(res.Refused, Refusal{Path: op.Record.Path, Reason: err.Error()})
				continue
			}
			return res, err
		}
		if op.Kind == OpDelete && !ok {
			// Tombstone for an absent path: record it so the deletion converges.
			res.Applied++
			continue
		}
		res.Applied++
	}
	return res, nil
}
