package repo

import (
	"encoding/json"
	"time"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

// clock and id generation are injectable for deterministic tests.
type clock func() time.Time

type idgen func() string

func defaultClock() time.Time { return time.Now().UTC() }

// putTyped stores a contract object and refreshes the projection.
func putTyped(s *store.Store, p projection.Projection, kind, id string, revision int, value any) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	rec, err := store.NewRecord(kind, id, revision, data)
	if err != nil {
		return false, err
	}
	changed, err := s.PutRecord(rec)
	if err != nil {
		return false, err
	}
	if changed && p != nil {
		_ = p.Put(projection.FromRecord(rec))
	}
	return changed, nil
}

// getTyped reads and decodes a contract object.
func getTyped[T any](s *store.Store, kind, id string) (T, error) {
	var zero T
	rec, err := s.GetRecord(kind, id)
	if err != nil {
		return zero, err
	}
	var out T
	if err := json.Unmarshal(rec.Data, &out); err != nil {
		return zero, err
	}
	return out, nil
}

// decode unmarshals a record's data into a contract object.
func decode(rec store.Record, into any) error {
	return json.Unmarshal(rec.Data, into)
}

// revisionOf returns the current record revision, or 0 if absent.
func revisionOf(s *store.Store, kind, id string) int {
	rec, err := s.GetRecord(kind, id)
	if err != nil {
		return 0
	}
	return rec.Revision
}
