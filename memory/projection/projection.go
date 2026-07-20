// Package projection provides aa-memory's derived, disposable views over
// canonical records. A projection can always be rebuilt from the store.
package projection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/greadee/aa/memory/store"
)

// Entry is one projected record.
type Entry struct {
	Kind     string
	ID       string
	Revision int
	Hash     string
	Data     json.RawMessage
}

// Projection is a derived view over canonical records.
type Projection interface {
	// Put inserts or replaces an entry.
	Put(Entry) error
	// Get returns an entry by kind and id.
	Get(kind, id string) (Entry, bool)
	// List returns all entries of a kind, sorted by id.
	List(kind string) []Entry
	// Len returns the number of entries.
	Len() int
	// Reset discards all entries.
	Reset()
	// Digest returns a deterministic digest of the projection contents.
	Digest() string
}

// FromRecord converts a canonical record into a projection entry.
func FromRecord(r store.Record) Entry {
	return Entry{Kind: r.Kind, ID: r.ID, Revision: r.Revision, Hash: r.Hash, Data: r.Data}
}

func key(kind, id string) string { return kind + "\x00" + id }

// MemProjection is the reference in-memory projection.
type MemProjection struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

// NewMem returns an empty in-memory projection.
func NewMem() *MemProjection {
	return &MemProjection{entries: make(map[string]Entry)}
}

// Put inserts or replaces an entry.
func (p *MemProjection) Put(e Entry) error {
	if e.Kind == "" || e.ID == "" {
		return fmt.Errorf("projection: kind and id are required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	k := key(e.Kind, e.ID)
	if existing, ok := p.entries[k]; ok {
		if existing.Hash == e.Hash {
			return nil
		}
		if e.Revision < existing.Revision {
			return fmt.Errorf("projection: stale revision %d for %s/%s", e.Revision, e.Kind, e.ID)
		}
	}
	p.entries[k] = e
	return nil
}

// Get returns an entry by kind and id.
func (p *MemProjection) Get(kind, id string) (Entry, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.entries[key(kind, id)]
	return e, ok
}

// List returns all entries of a kind, sorted by id.
func (p *MemProjection) List(kind string) []Entry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []Entry
	for _, e := range p.entries {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Len returns the number of entries.
func (p *MemProjection) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.entries)
}

// Reset discards all entries.
func (p *MemProjection) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = make(map[string]Entry)
}

// Digest returns a deterministic digest over all entries.
func (p *MemProjection) Digest() string {
	p.mu.RLock()
	keys := make([]string, 0, len(p.entries))
	for k := range p.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		e := p.entries[k]
		fmt.Fprintf(h, "%s\x00%s\x00%d\x00%s\n", e.Kind, e.ID, e.Revision, e.Hash)
	}
	p.mu.RUnlock()
	return hex.EncodeToString(h.Sum(nil))
}

// DigestOf returns the digest for an arbitrary set of entries.
func DigestOf(entries []Entry) string {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		return sorted[i].ID < sorted[j].ID
	})
	var b strings.Builder
	for _, e := range sorted {
		fmt.Fprintf(&b, "%s\x00%s\x00%d\x00%s\n", e.Kind, e.ID, e.Revision, e.Hash)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
