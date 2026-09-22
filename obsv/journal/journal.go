// Package journal is aa-obsv's append-only observation journal.
//
// The journal assigns the sequence, so ordering never depends on a wall clock
// and replay is deterministic. It offers cursor replay, live subscription, and
// a bounded projected view. The in-memory journal is the default; the file
// journal adds a durable JSONL record that survives a reopen.
package journal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/greadee/aa/obsv/protocol"
)

// Policy bounds a projected view of a journal.
type Policy struct {
	// MaxEvents keeps at most this many most-recent events. Zero or less means
	// no limit.
	MaxEvents int
	// MinSequence drops events below this sequence floor. Zero or less means no
	// floor.
	MinSequence int
}

// Subscription delivers journaled events in sequence order. Delivery is
// failure-open: a subscriber that cannot keep up drops events rather than
// blocking the producer.
type Subscription struct {
	ch        chan protocol.Event
	closeOnce sync.Once
	journal   *Memory

	mu      sync.Mutex
	dropped int
	closed  bool
}

// Events returns the receive channel. It is closed when the subscription is
// closed or the journal closes.
func (s *Subscription) Events() <-chan protocol.Event { return s.ch }

// Dropped returns how many events were dropped because the subscriber lagged.
func (s *Subscription) Dropped() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

// Close unregisters the subscription and closes its channel.
func (s *Subscription) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		s.journal.remove(s)
		close(s.ch)
	})
}

func (s *Subscription) deliver(ev protocol.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	select {
	case s.ch <- ev:
	default:
		s.dropped++
	}
}

// Journal is an append-only, sequence-ordered observation record.
type Journal interface {
	// Append stores an event, assigning its sequence. It returns the stored
	// event.
	Append(protocol.Event) (protocol.Event, error)
	// Replay returns events with sequence >= from, in order.
	Replay(from int) ([]protocol.Event, error)
	// Subscribe returns a live subscription starting at from.
	Subscribe(from int) (*Subscription, error)
	// Len returns the number of stored events.
	Len() int
	// Trim returns a bounded projected view; the journal is not modified.
	Trim(Policy) []protocol.Event
	// Close releases resources and closes subscriptions.
	Close() error
}

// Memory is the in-memory journal.
type Memory struct {
	mu     sync.Mutex
	events []protocol.Event
	next   int
	subs   map[*Subscription]struct{}
	closed bool
}

// NewMemory returns an empty in-memory journal.
func NewMemory() *Memory {
	return &Memory{subs: make(map[*Subscription]struct{})}
}

// Append stores an event and assigns the next sequence.
func (m *Memory) Append(ev protocol.Event) (protocol.Event, error) {
	stored, err := m.appendAssigned(ev)
	return stored, err
}

func (m *Memory) appendAssigned(ev protocol.Event) (protocol.Event, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return protocol.Event{}, fmt.Errorf("obsv/journal: journal is closed")
	}
	if ev.Version == "" {
		ev.Version = protocol.Version
	}
	ev.Sequence = m.next
	if err := ev.Validate(); err != nil {
		m.mu.Unlock()
		return protocol.Event{}, err
	}
	m.events = append(m.events, ev)
	m.next++
	subs := make([]*Subscription, 0, len(m.subs))
	for s := range m.subs {
		subs = append(subs, s)
	}
	m.mu.Unlock()

	for _, s := range subs {
		s.deliver(ev)
	}
	return ev, nil
}

// Replay returns events with sequence >= from in order.
func (m *Memory) Replay(from int) ([]protocol.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]protocol.Event, 0, len(m.events))
	for _, ev := range m.events {
		if ev.Sequence >= from {
			out = append(out, ev)
		}
	}
	return out, nil
}

// Subscribe returns a live subscription. Events already journaled at or after
// from are delivered before new ones.
func (m *Memory) Subscribe(from int) (*Subscription, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, fmt.Errorf("obsv/journal: journal is closed")
	}
	backlog := make([]protocol.Event, 0, len(m.events))
	for _, ev := range m.events {
		if ev.Sequence >= from {
			backlog = append(backlog, ev)
		}
	}
	s := &Subscription{
		ch:      make(chan protocol.Event, len(backlog)+64),
		journal: m,
	}
	m.subs[s] = struct{}{}
	m.mu.Unlock()

	for _, ev := range backlog {
		s.ch <- ev
	}
	return s, nil
}

func (m *Memory) remove(s *Subscription) {
	m.mu.Lock()
	delete(m.subs, s)
	m.mu.Unlock()
}

// Len returns the number of stored events.
func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

// Trim returns a bounded projected view. The journal is not modified.
func (m *Memory) Trim(p Policy) []protocol.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return trim(m.events, p)
}

// Close closes the journal and every subscription.
func (m *Memory) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	subs := make([]*Subscription, 0, len(m.subs))
	for s := range m.subs {
		subs = append(subs, s)
	}
	m.subs = make(map[*Subscription]struct{})
	m.mu.Unlock()

	for _, s := range subs {
		s.Close()
	}
	return nil
}

func trim(events []protocol.Event, p Policy) []protocol.Event {
	kept := make([]protocol.Event, 0, len(events))
	for _, ev := range events {
		if p.MinSequence > 0 && ev.Sequence < p.MinSequence {
			continue
		}
		kept = append(kept, ev)
	}
	if p.MaxEvents > 0 && len(kept) > p.MaxEvents {
		kept = kept[len(kept)-p.MaxEvents:]
	}
	out := make([]protocol.Event, len(kept))
	copy(out, kept)
	return out
}

// File is a durable JSONL journal. Events are appended to the file and indexed
// in memory for replay and subscription.
type File struct {
	mu   sync.Mutex
	path string
	f    *os.File
	mem  *Memory
}

// OpenFile opens or creates a durable journal at path, loading existing events.
func OpenFile(path string) (*File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("obsv/journal: open %s: %w", path, err)
	}
	mem := NewMemory()
	loaded, err := load(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	for _, ev := range loaded {
		// Preserve the durable sequence: appendAssigned assigns m.next, so set
		// it to the stored value first.
		if ev.Sequence > mem.next {
			mem.next = ev.Sequence
		}
		if _, err := mem.appendAssigned(ev); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	return &File{path: path, f: f, mem: mem}, nil
}

// Append writes an event to the file and indexes it.
func (f *File) Append(ev protocol.Event) (protocol.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, err := f.mem.appendAssigned(ev)
	if err != nil {
		return protocol.Event{}, err
	}
	line, err := json.Marshal(stored)
	if err != nil {
		return protocol.Event{}, fmt.Errorf("obsv/journal: encode: %w", err)
	}
	if _, err := f.f.Write(append(line, '\n')); err != nil {
		return protocol.Event{}, fmt.Errorf("obsv/journal: write: %w", err)
	}
	return stored, nil
}

// Replay delegates to the in-memory index.
func (f *File) Replay(from int) ([]protocol.Event, error) { return f.mem.Replay(from) }

// Subscribe delegates to the in-memory index.
func (f *File) Subscribe(from int) (*Subscription, error) { return f.mem.Subscribe(from) }

// Len delegates to the in-memory index.
func (f *File) Len() int { return f.mem.Len() }

// Trim returns a bounded projected view.
func (f *File) Trim(p Policy) []protocol.Event { return f.mem.Trim(p) }

// Path returns the journal path.
func (f *File) Path() string { return f.path }

// Close closes the journal and the file.
func (f *File) Close() error {
	if err := f.mem.Close(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.f == nil {
		return nil
	}
	err := f.f.Close()
	f.f = nil
	return err
}

func load(f *os.File) ([]protocol.Event, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("obsv/journal: seek: %w", err)
	}
	var events []protocol.Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev protocol.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("obsv/journal: decode: %w", err)
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("obsv/journal: read: %w", err)
	}
	return events, nil
}
