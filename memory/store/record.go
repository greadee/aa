// Package store implements aa-memory's canonical, portable record store.
//
// Canonical records live on disk as JSON; events are an append-only log. The
// store is the source of truth. Projections are derived and disposable.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	v1 "github.com/greadee/aa/contracts/go/v1"
)

// Revision is the initial revision of a record.
const Revision = 1

// Record is a canonical, materialized contract object.
type Record struct {
	Kind     string          `json:"kind"`
	ID       string          `json:"id"`
	Revision int             `json:"revision"`
	Data     json.RawMessage `json:"data"`
	Hash     string          `json:"hash"`
}

// Canonicalize returns deterministic JSON bytes: compact and key-sorted.
func Canonicalize(data []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("canonicalize: %w", err)
	}
	out, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("canonicalize: %w", err)
	}
	return out, nil
}

// HashBytes returns the lowercase SHA-256 hex of canonicalized bytes.
func HashBytes(data []byte) (string, error) {
	canonical, err := Canonicalize(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// NewRecord builds a validated record, canonicalizing data and computing its hash.
func NewRecord(kind, id string, revision int, data []byte) (Record, error) {
	if err := v1.RequireIdentifier("kind", kind); err != nil {
		return Record{}, err
	}
	if err := v1.RequireIdentifier("id", id); err != nil {
		return Record{}, err
	}
	if revision < 0 {
		return Record{}, fmt.Errorf("revision: must be >= 0")
	}
	canonical, err := Canonicalize(data)
	if err != nil {
		return Record{}, err
	}
	hash, err := HashBytes(canonical)
	if err != nil {
		return Record{}, err
	}
	return Record{Kind: kind, ID: id, Revision: revision, Data: canonical, Hash: hash}, nil
}

// Validate checks the record's envelope and recomputes its hash.
func (r Record) Validate() error {
	if err := v1.RequireIdentifier("kind", r.Kind); err != nil {
		return err
	}
	if err := v1.RequireIdentifier("id", r.ID); err != nil {
		return err
	}
	if r.Revision < 0 {
		return fmt.Errorf("revision: must be >= 0")
	}
	hash, err := HashBytes(r.Data)
	if err != nil {
		return err
	}
	if hash != r.Hash {
		return fmt.Errorf("hash: does not match data")
	}
	return nil
}

// Value decodes the record data into a map.
func (r Record) Value() (map[string]any, error) {
	var value map[string]any
	if err := json.Unmarshal(r.Data, &value); err != nil {
		return nil, fmt.Errorf("record %s/%s: %w", r.Kind, r.ID, err)
	}
	return value, nil
}
