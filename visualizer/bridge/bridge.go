// Package bridge connects a live event subscription to the same projection and
// layout path replay uses.
//
// The bridge accumulates events from a source.Subscription and re-derives its
// frame list with replay.Build, so a live session and its replay are always the
// same frames. It does not own the observation transport: the subscription is
// supplied by the caller, and the real aa-obsv transport remains a follow-up.
package bridge

import (
	"context"
	"fmt"
	"sync"

	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/replay"
	"github.com/greadee/aa/visualizer/source"
)

// Bridge folds a live subscription through the replay projection and layout
// path. It is safe for concurrent use.
type Bridge struct {
	mu        sync.Mutex
	sessionID visualizer.SessionID
	events    []visualizer.Event
	frames    replay.FrameList
	built     bool
}

// New returns a bridge for a session.
func New(sessionID visualizer.SessionID) (*Bridge, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	return &Bridge{sessionID: sessionID}, nil
}

// SessionID returns the session the bridge is observing.
func (b *Bridge) SessionID() visualizer.SessionID { return b.sessionID }

// Apply validates and accumulates events, invalidating the derived frame list.
// Events may arrive in any order; the frame list is always ordered by Sequence.
func (b *Bridge) Apply(events ...visualizer.Event) error {
	for i, e := range events {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("%w: events[%d]: %s", visualizer.ErrInvalid, i, err)
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, events...)
	b.built = false
	return nil
}

// Events returns a copy of the accumulated events.
func (b *Bridge) Events() []visualizer.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]visualizer.Event(nil), b.events...)
}

// FrameList re-derives the frame list through the replay path if needed and
// returns it.
func (b *Bridge) FrameList() (replay.FrameList, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.built {
		list, err := replay.Build(b.sessionID, b.events)
		if err != nil {
			return replay.FrameList{}, err
		}
		b.frames = list
		b.built = true
	}
	return b.frames, nil
}

// Cursor returns a time-travel cursor over the live frame list.
func (b *Bridge) Cursor() (*replay.Cursor, error) {
	list, err := b.FrameList()
	if err != nil {
		return nil, err
	}
	return list.Cursor()
}

// Consume reads a subscription to completion, applying every event. It returns
// when the subscription closes or ctx is done. The caller owns the subscription.
func (b *Bridge) Consume(ctx context.Context, sub source.Subscription) error {
	if sub == nil {
		return fmt.Errorf("%w: subscription is required", visualizer.ErrInvalid)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e, ok := <-sub.Events():
			if !ok {
				return nil
			}
			if err := b.Apply(e); err != nil {
				return err
			}
		}
	}
}
