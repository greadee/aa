package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("store: record not found")

// EventRecord is one entry in the append-only event log.
type EventRecord struct {
	ID       string
	Sequence int
	Data     json.RawMessage
}

// Store is a canonical, portable record and event store.
type Store struct {
	layout Layout
	mu     sync.Mutex
	seen   map[string]bool
}

// Open prepares the store rooted at root, creating the layout directories.
func Open(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("root: is required")
	}
	layout := NewLayout(root)
	if err := os.MkdirAll(layout.RecordsRoot(), 0o700); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	return &Store{layout: layout, seen: make(map[string]bool)}, nil
}

// Layout returns the store layout.
func (s *Store) Layout() Layout { return s.layout }

// InitProject writes the project manifest once. It is idempotent.
func (s *Store) InitProject(manifest []byte) (bool, error) {
	canonical, err := Canonicalize(manifest)
	if err != nil {
		return false, err
	}
	path := s.layout.ManifestPath()
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	if err := writeFileAtomic(path, canonical, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

// Manifest returns the canonical project manifest.
func (s *Store) Manifest() (json.RawMessage, error) {
	data, err := os.ReadFile(s.layout.ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// PutRecord writes a record atomically. It is idempotent for the same content.
// The second result reports whether the store changed.
func (s *Store) PutRecord(rec Record) (bool, error) {
	if err := rec.Validate(); err != nil {
		return false, err
	}
	path, err := s.layout.RecordPath(rec.Kind, rec.ID)
	if err != nil {
		return false, err
	}
	if existing, err := readRecord(path); err == nil {
		if existing.Hash == rec.Hash {
			return false, nil
		}
		if rec.Revision <= existing.Revision {
			return false, fmt.Errorf("record %s/%s: stale revision %d (have %d)",
				rec.Kind, rec.ID, rec.Revision, existing.Revision)
		}
	} else if !errors.Is(err, ErrNotFound) {
		return false, err
	}
	blob, err := json.Marshal(rec)
	if err != nil {
		return false, err
	}
	if err := writeFileAtomic(path, blob, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

// GetRecord reads a record by kind and id.
func (s *Store) GetRecord(kind, id string) (Record, error) {
	path, err := s.layout.RecordPath(kind, id)
	if err != nil {
		return Record{}, err
	}
	return readRecord(path)
}

// ListRecords lists all records of a kind, sorted by id.
func (s *Store) ListRecords(kind string) ([]Record, error) {
	dir, err := s.layout.KindDir(kind)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	records := make([]Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		rec, err := readRecord(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	return records, nil
}

// ListAllRecords lists every record in the store, sorted by kind then id.
func (s *Store) ListAllRecords() ([]Record, error) {
	var all []Record
	err := filepath.WalkDir(s.layout.RecordsRoot(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		rec, err := readRecord(path)
		if err != nil {
			return err
		}
		all = append(all, rec)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Kind != all[j].Kind {
			return all[i].Kind < all[j].Kind
		}
		return all[i].ID < all[j].ID
	})
	return all, nil
}

// AppendEvent appends a canonical event to the log. Duplicate ids are ignored.
// The second result reports whether the log changed.
func (s *Store) AppendEvent(event []byte) (bool, error) {
	canonical, err := Canonicalize(event)
	if err != nil {
		return false, err
	}
	var probe struct {
		ID       string `json:"id"`
		Sequence int    `json:"sequence"`
	}
	if err := json.Unmarshal(canonical, &probe); err != nil {
		return false, err
	}
	if probe.ID == "" {
		return false, fmt.Errorf("event.id: is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadSeen(); err != nil {
		return false, err
	}
	if s.seen[probe.ID] {
		return false, nil
	}
	f, err := os.OpenFile(s.layout.EventsPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return false, err
	}
	defer f.Close()
	if _, err := f.Write(append(canonical, '\n')); err != nil {
		return false, err
	}
	s.seen[probe.ID] = true
	return true, nil
}

// Events returns the event log in insertion order.
func (s *Store) Events() ([]EventRecord, error) {
	f, err := os.Open(s.layout.EventsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []EventRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var probe struct {
			ID       string `json:"id"`
			Sequence int    `json:"sequence"`
		}
		if err := json.Unmarshal(line, &probe); err != nil {
			return nil, fmt.Errorf("store: corrupt event log: %w", err)
		}
		data := make([]byte, len(line))
		copy(data, line)
		events = append(events, EventRecord{ID: probe.ID, Sequence: probe.Sequence, Data: data})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) loadSeen() error {
	if s.seen == nil {
		s.seen = make(map[string]bool)
	}
	if len(s.seen) > 0 {
		return nil
	}
	events, err := s.Events()
	if err != nil {
		return err
	}
	for _, e := range events {
		s.seen[e.ID] = true
	}
	return nil
}

func readRecord(path string) (Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, ErrNotFound
		}
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, fmt.Errorf("store: corrupt record %s: %w", path, err)
	}
	return rec, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
