// Package visplane composes aa-visualizer over aa-obsv and aa-memory.
//
// Observation sessions replay and subscribe through the single obsv transport
// that the daemon already owns. Canonical session browsing goes through the
// memory query seam. The visualizer owns no second protocol, transport, or
// event store, and it never imports the kernel.
package visplane

import (
	"context"
	"errors"
	"sort"

	"github.com/greadee/aa/obsv/transport"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/browse"
	"github.com/greadee/aa/visualizer/replay"
	"github.com/greadee/aa/visualizer/retention"
	"github.com/greadee/aa/visualizer/session"
	"github.com/greadee/aa/visualizer/source"
)

// Config configures the visualizer plane.
type Config struct {
	// Observations is the shared obsv transport owned by the daemon.
	Observations transport.Server
	// ObservedSessions lists the observation session ids the daemon has opened.
	ObservedSessions func() []string
	// Memory is the canonical memory query seam used for session browsing.
	Memory browse.EventSource
	// Retention bounds the projected replay trail.
	Retention retention.Policy
}

// Plane is the composed visualizer service.
type Plane struct {
	source    *source.OBsv
	browser   *browse.Browser
	observed  func() []string
	retention retention.Policy
}

// New composes the visualizer over the shared observation transport and the
// memory query seam.
func New(cfg Config) (*Plane, error) {
	src, err := source.New(cfg.Observations)
	if err != nil {
		return nil, err
	}
	p := &Plane{source: src, observed: cfg.ObservedSessions, retention: cfg.Retention}
	if cfg.Memory != nil {
		browser, err := browse.New(cfg.Memory)
		if err != nil {
			return nil, err
		}
		p.browser = browser
	}
	return p, nil
}

// Sessions lists canonical memory sessions plus observation sessions that have
// not appeared in canonical history, ordered by session id.
func (p *Plane) Sessions(ctx context.Context) ([]visualizer.Session, error) {
	var out []visualizer.Session
	seen := map[visualizer.SessionID]bool{}
	if p.browser != nil {
		canonical, err := p.browser.List()
		if err != nil {
			return nil, err
		}
		for _, s := range canonical {
			seen[s.ID] = true
			out = append(out, s)
		}
	}
	if p.observed != nil {
		for _, id := range p.observed() {
			sessionID := visualizer.SessionID(id)
			if seen[sessionID] {
				continue
			}
			events, err := p.source.Replay(ctx, sessionID)
			if err != nil {
				if errors.Is(err, visualizer.ErrEmpty) {
					continue
				}
				return nil, err
			}
			summary, err := session.Summarize(sessionID, events)
			if err != nil {
				return nil, err
			}
			seen[sessionID] = true
			out = append(out, summary)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Timeline replays an observation session through the obsv source, projects it
// with the shared replay path, and applies the configured retention policy.
func (p *Plane) Timeline(ctx context.Context, sessionID visualizer.SessionID) (replay.FrameList, error) {
	events, err := p.source.Replay(ctx, sessionID)
	if err != nil {
		return replay.FrameList{}, err
	}
	list, err := replay.Build(sessionID, events)
	if err != nil {
		return replay.FrameList{}, err
	}
	return retention.TrimList(list, p.retention), nil
}

// Graph projects the final graph for an observation session.
func (p *Plane) Graph(ctx context.Context, sessionID visualizer.SessionID) (visualizer.Graph, error) {
	events, err := p.source.Replay(ctx, sessionID)
	if err != nil {
		return visualizer.Graph{}, err
	}
	return session.Project(sessionID, events)
}

// Subscribe follows a live observation session over the same obsv transport.
func (p *Plane) Subscribe(ctx context.Context, sessionID visualizer.SessionID) (source.Subscription, error) {
	return p.source.Subscribe(ctx, sessionID)
}
