// Package source is the visualizer's event seam.
//
// Live and replay both consume events through EventSource. The real aa-obsv
// Replay/Subscribe transport is a follow-up; until then a deterministic
// in-memory Fake stands in. The seam reuses the shared contracts.Event
// taxonomy and does not define a new observation protocol.
package source

import (
	"context"
	"fmt"
	"sort"
	"sync"

	visualizer "github.com/greadee/aa/visualizer"
)

// EventSource yields a session's events for replay and for a live subscription.
type EventSource interface {
	// Replay returns every event of a session in sequence order.
	Replay(ctx context.Context, sessionID visualizer.SessionID) ([]visualizer.Event, error)
	// Subscribe returns a live subscription to a session's future events.
	Subscribe(ctx context.Context, sessionID visualizer.SessionID) (Subscription, error)
}

// Subscription is a live stream of events.
type Subscription interface {
	Events() <-chan visualizer.Event
	Close() error
}

// Fake is a deterministic in-memory EventSource for tests.
type Fake struct {
	mu       sync.Mutex
	sessions map[visualizer.SessionID][]visualizer.Event
	subs     map[visualizer.SessionID][]*subscriber
	nextID   int
}

// NewFake returns an empty fake source.
func NewFake() *Fake {
	return &Fake{
		sessions: map[visualizer.SessionID][]visualizer.Event{},
		subs:     map[visualizer.SessionID][]*subscriber{},
	}
}

// Append validates and stores events for a session. Events may be appended out
// of sequence order; Replay and the projection order them by Sequence.
func (f *Fake) Append(sessionID visualizer.SessionID, events ...visualizer.Event) error {
	if sessionID == "" {
		return fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	for i, e := range events {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%w: events[%d]: %s", visualizer.ErrInvalid, i, err)
		}
	}
	f.mu.Lock()
	f.sessions[sessionID] = append(f.sessions[sessionID], events...)
	subs := append([]*subscriber(nil), f.subs[sessionID]...)
	f.mu.Unlock()

	// Deliver after releasing the lock so a consumer can call back safely.
	for _, s := range subs {
		for _, e := range events {
			s.deliver(e)
		}
	}
	return nil
}

// Replay implements EventSource, returning a sequence-ordered copy.
func (f *Fake) Replay(_ context.Context, sessionID visualizer.SessionID) ([]visualizer.Event, error) {
	f.mu.Lock()
	stored, ok := f.sessions[sessionID]
	f.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("%w: session %q", visualizer.ErrNotFound, sessionID)
	}
	if len(stored) == 0 {
		return nil, fmt.Errorf("%w: session %q", visualizer.ErrEmpty, sessionID)
	}
	out := append([]visualizer.Event(nil), stored...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out, nil
}

// Subscribe implements EventSource, delivering events appended after the call.
func (f *Fake) Subscribe(_ context.Context, sessionID visualizer.SessionID) (Subscription, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s := &subscriber{ch: make(chan visualizer.Event, 256), parent: f, session: sessionID, id: f.nextID}
	f.nextID++
	f.subs[sessionID] = append(f.subs[sessionID], s)
	return s, nil
}

func (f *Fake) remove(s *subscriber) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.subs[s.session]
	for i, candidate := range list {
		if candidate == s {
			f.subs[s.session] = append(list[:i], list[i+1:]...)
			break
		}
	}
}

type subscriber struct {
	ch      chan visualizer.Event
	parent  *Fake
	session visualizer.SessionID
	id      int
	once    sync.Once
}

func (s *subscriber) Events() <-chan visualizer.Event { return s.ch }

func (s *subscriber) Close() error {
	s.once.Do(func() {
		s.parent.remove(s)
		close(s.ch)
	})
	return nil
}

func (s *subscriber) deliver(e visualizer.Event) {
	defer func() { _ = recover() }() // a closed subscriber must not panic the source
	s.ch <- e
}
