// Package source is the visualizer's event seam.
//
// Live and replay both consume events through EventSource. The seam is backed
// by the shared aa-obsv transport: Replay reads a session's journal and
// Subscribe follows it, and one adapter maps obsv observations into the shared
// contracts event taxonomy. The visualizer defines no observation protocol or
// transport of its own; the transport may be the in-process one or a kernel
// host socket that implements the same surface.
package source

import (
	"context"
	"fmt"

	"github.com/greadee/aa/obsv/journal"
	"github.com/greadee/aa/obsv/transport"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/adapter"
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

// OBsv is the EventSource backed by the shared aa-obsv transport.
type OBsv struct {
	server transport.Server
}

// New returns an OBsv source over an obsv transport. The transport is the
// shared aa-obsv surface, so the in-process transport works now and a
// kernel-host socket client can take its place without changing callers.
func New(server transport.Server) (*OBsv, error) {
	if server == nil {
		return nil, fmt.Errorf("%w: obsv transport is required", visualizer.ErrInvalid)
	}
	return &OBsv{server: server}, nil
}

// NewLocal returns an OBsv source over an in-process obsv transport owned by
// owner. A nil newJournal uses an in-memory journal per session.
func NewLocal(owner string, newJournal transport.NewJournal) *OBsv {
	return &OBsv{server: transport.NewLocal(owner, newJournal)}
}

// Replay reads a session through obsv Replay and maps the observations to the
// shared taxonomy in Sequence order.
func (s *OBsv) Replay(ctx context.Context, sessionID visualizer.SessionID) ([]visualizer.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	client, err := transport.Dial(s.server, string(sessionID))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", visualizer.ErrUnavailable, err)
	}
	raw, err := client.Replay(0)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", visualizer.ErrUnavailable, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: session %q", visualizer.ErrEmpty, sessionID)
	}
	out := make([]visualizer.Event, 0, len(raw))
	for i, ev := range raw {
		mapped, err := adapter.ToContract(ev)
		if err != nil {
			return nil, fmt.Errorf("%w: event %d: %s", visualizer.ErrInvalid, i, err)
		}
		out = append(out, mapped)
	}
	return out, nil
}

// Subscribe follows a session through obsv Subscribe, mapping each observation
// to the shared taxonomy as it arrives.
func (s *OBsv) Subscribe(ctx context.Context, sessionID visualizer.SessionID) (Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	client, err := transport.Dial(s.server, string(sessionID))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", visualizer.ErrUnavailable, err)
	}
	sub, err := client.Subscribe(0)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", visualizer.ErrUnavailable, err)
	}
	return newSubscription(ctx, sub), nil
}

// subscription adapts an obsv journal subscription to the visualizer seam.
type subscription struct {
	sub *journal.Subscription
	ch  chan visualizer.Event
}

func newSubscription(ctx context.Context, sub *journal.Subscription) *subscription {
	s := &subscription{sub: sub, ch: make(chan visualizer.Event, 256)}
	go s.pump(ctx)
	return s
}

func (s *subscription) Events() <-chan visualizer.Event { return s.ch }

func (s *subscription) Close() error {
	s.sub.Close()
	return nil
}

// pump maps observations until the obsv subscription or the context closes.
// Delivery stays failure-open like obsv: an observation that cannot be mapped
// is dropped rather than stalling the stream.
func (s *subscription) pump(ctx context.Context) {
	defer close(s.ch)
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-s.sub.Events():
			if !ok {
				return
			}
			mapped, err := adapter.ToContract(ev)
			if err != nil {
				continue
			}
			select {
			case s.ch <- mapped:
			case <-ctx.Done():
				return
			}
		}
	}
}
