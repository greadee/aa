// Package browse lists and loads past sessions from aa-memory through a query
// seam.
//
// The visualizer never owns canonical history: it reads the canonical event log
// through the EventSource seam (satisfied by aa-memory's query API) and derives
// session summaries with the same session package the live path uses.
package browse

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/greadee/aa/memory/store"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/replay"
	"github.com/greadee/aa/visualizer/session"
)

// DefaultSessionID is the session used for events that carry neither an
// explicit session id nor a project id.
const DefaultSessionID visualizer.SessionID = "default"

// EventSource is the aa-memory query seam. It is satisfied by *query.Query; the
// visualizer reads canonical history but never writes it.
type EventSource interface {
	// Events returns the canonical event log in insertion order.
	Events() ([]store.EventRecord, error)
}

// Browser lists and loads sessions from the canonical event log.
type Browser struct {
	source EventSource
}

// New returns a browser over an event source.
func New(source EventSource) (*Browser, error) {
	if source == nil {
		return nil, fmt.Errorf("%w: event source is required", visualizer.ErrInvalid)
	}
	return &Browser{source: source}, nil
}

// SessionKey returns the deterministic session an event belongs to.
//
// The canonical event taxonomy carries no explicit session id, so the
// visualizer derives one: an observation session id in the payload wins, then
// the durable ProjectID, then DefaultSessionID. Every event belongs to exactly
// one session, so no history is dropped.
func SessionKey(e visualizer.Event) visualizer.SessionID {
	if e.Payload != nil {
		if v, ok := e.Payload["sessionId"].(string); ok && v != "" {
			return visualizer.SessionID(v)
		}
	}
	if e.ProjectID != "" {
		return visualizer.SessionID(e.ProjectID)
	}
	return DefaultSessionID
}

// List returns session summaries sorted by session id.
func (b *Browser) List() ([]visualizer.Session, error) {
	events, err := b.events()
	if err != nil {
		return nil, err
	}
	groups := make(map[visualizer.SessionID][]visualizer.Event)
	for _, e := range events {
		id := SessionKey(e)
		groups[id] = append(groups[id], e)
	}
	sessions := make([]visualizer.Session, 0, len(groups))
	for id, group := range groups {
		summary, err := session.Summarize(id, group)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, summary)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].ID < sessions[j].ID })
	return sessions, nil
}

// Load returns one session's events in Sequence order. An unknown session
// returns ErrNotFound.
func (b *Browser) Load(sessionID visualizer.SessionID) ([]visualizer.Event, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	events, err := b.events()
	if err != nil {
		return nil, err
	}
	var out []visualizer.Event
	for _, e := range events {
		if SessionKey(e) == sessionID {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: session %q", visualizer.ErrNotFound, sessionID)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out, nil
}

// Timeline loads a session and projects it into a replay frame list through the
// same path live rendering uses.
func (b *Browser) Timeline(sessionID visualizer.SessionID) (replay.FrameList, error) {
	events, err := b.Load(sessionID)
	if err != nil {
		return replay.FrameList{}, err
	}
	return replay.Build(sessionID, events)
}

func (b *Browser) events() ([]visualizer.Event, error) {
	records, err := b.source.Events()
	if err != nil {
		return nil, err
	}
	events := make([]visualizer.Event, 0, len(records))
	for _, rec := range records {
		var e visualizer.Event
		if err := json.Unmarshal(rec.Data, &e); err != nil {
			return nil, fmt.Errorf("%w: event %q: %s", visualizer.ErrInvalid, rec.ID, err)
		}
		events = append(events, e)
	}
	return events, nil
}
