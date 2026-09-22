// Package emit provides aa-obsv's bounded, failure-open observation emitter.
//
// Emitting observation must never change agent behavior. The emitter therefore
// never blocks, never propagates an error, and never lets a panic escape: a
// failing target or a full buffer is counted and the event is dropped.
package emit

import (
	"errors"
	"sync"

	"github.com/greadee/aa/obsv/protocol"
)

// Target receives observation events. A Target may report ErrFull when it is
// bounded and saturated.
type Target interface {
	Append(protocol.Event) error
}

// ErrFull is returned by a bounded Target that cannot accept another event.
var ErrFull = errors.New("obsv/emit: target is full")

// Stats accounts for how an Emitter handled events.
type Stats struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"` // invalid event or a full target
	Failed   int `json:"failed"`  // the target errored or panicked
}

// Emitter is a failure-open, sanitizing writer to a Target.
type Emitter struct {
	mu     sync.Mutex
	target Target
	stats  Stats
}

// New returns an emitter writing to target. A nil target discards events.
func New(target Target) *Emitter {
	if target == nil {
		target = discard{}
	}
	return &Emitter{target: target}
}

// Emit normalizes and sanitizes the event, validates it, and writes it to the
// target. It returns whether the event was accepted. It never panics and never
// returns an error.
func (e *Emitter) Emit(ev protocol.Event) bool {
	if ev.Version == "" {
		ev.Version = protocol.Version
	}
	ev.Metadata = protocol.SanitizeMetadata(ev.Metadata)
	if err := ev.Validate(); err != nil {
		e.mu.Lock()
		e.stats.Dropped++
		e.mu.Unlock()
		return false
	}

	outcome := write(e.target, ev)

	e.mu.Lock()
	switch outcome {
	case outcomeAccepted:
		e.stats.Accepted++
	case outcomeDropped:
		e.stats.Dropped++
	default:
		e.stats.Failed++
	}
	e.mu.Unlock()
	return outcome == outcomeAccepted
}

type result int

const (
	outcomeAccepted result = iota
	outcomeDropped
	outcomeFailed
)

// write appends to the target, recovering a panic, and classifies the outcome.
func write(target Target, ev protocol.Event) (outcome result) {
	defer func() {
		if r := recover(); r != nil {
			outcome = outcomeFailed
		}
	}()
	if err := target.Append(ev); err != nil {
		if errors.Is(err, ErrFull) {
			return outcomeDropped
		}
		return outcomeFailed
	}
	return outcomeAccepted
}

// Stats returns a snapshot of the emitter counters.
func (e *Emitter) Stats() Stats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}

type discard struct{}

func (discard) Append(protocol.Event) error { return nil }

// Buffer is a bounded Target. It keeps the most recent events and drops the
// newest once capacity is reached, returning ErrFull so the drop is visible and
// accounted. A capacity of zero or less is unbounded.
type Buffer struct {
	mu      sync.Mutex
	cap     int
	events  []protocol.Event
	dropped int
}

// NewBuffer returns a bounded buffer with the given capacity.
func NewBuffer(capacity int) *Buffer {
	return &Buffer{cap: capacity}
}

// Append implements Target.
func (b *Buffer) Append(ev protocol.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cap > 0 && len(b.events) >= b.cap {
		b.dropped++
		return ErrFull
	}
	b.events = append(b.events, ev)
	return nil
}

// Len returns the number of buffered events.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}

// Dropped returns the number of events the buffer refused.
func (b *Buffer) Dropped() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dropped
}

// Drain writes buffered events to dst in order and removes the events it wrote.
// On the first write error it returns the number written and the error, leaving
// the unwritten events buffered.
func (b *Buffer) Drain(dst Target) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	written := 0
	for i, ev := range b.events {
		if err := dst.Append(ev); err != nil {
			b.events = b.events[i:]
			return written, err
		}
		written++
	}
	b.events = b.events[:0]
	return written, nil
}
