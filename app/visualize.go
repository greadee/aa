package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/greadee/aa/app/internal/controlplane"
	"github.com/greadee/aa/app/internal/visplane"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/replay"
)

// visualizerBackend adapts the composed visualizer plane to the control-plane
// transport types.
type visualizerBackend struct {
	plane *visplane.Plane
}

func (v visualizerBackend) Sessions(ctx context.Context) ([]controlplane.VisualizerSession, error) {
	if v.plane == nil {
		return nil, nil
	}
	sessions, err := v.plane.Sessions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]controlplane.VisualizerSession, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, visualizerSession(s))
	}
	return out, nil
}

func (v visualizerBackend) Timeline(ctx context.Context, sessionID string) (controlplane.VisualizerTimeline, error) {
	if v.plane == nil {
		return controlplane.VisualizerTimeline{}, nil
	}
	list, err := v.plane.Timeline(ctx, visualizer.SessionID(sessionID))
	if err != nil {
		return controlplane.VisualizerTimeline{}, err
	}
	return visualizerTimeline(list), nil
}

func (v visualizerBackend) Graph(ctx context.Context, sessionID string) (controlplane.VisualizerGraph, error) {
	if v.plane == nil {
		return controlplane.VisualizerGraph{}, nil
	}
	graph, err := v.plane.Graph(ctx, visualizer.SessionID(sessionID))
	if err != nil {
		return controlplane.VisualizerGraph{}, err
	}
	return visualizerGraph(graph), nil
}

func (v visualizerBackend) Subscribe(ctx context.Context, sessionID string) (<-chan json.RawMessage, error) {
	if v.plane == nil {
		out := make(chan json.RawMessage)
		close(out)
		return out, nil
	}
	sub, err := v.plane.Subscribe(ctx, visualizer.SessionID(sessionID))
	if err != nil {
		return nil, err
	}
	out := make(chan json.RawMessage, 256)
	go func() {
		defer close(out)
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Events():
				if !ok {
					return
				}
				data, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				select {
				case out <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func visualizerSession(s visualizer.Session) controlplane.VisualizerSession {
	out := controlplane.VisualizerSession{
		ID:            string(s.ID),
		ProjectID:     s.ProjectID,
		Actors:        s.Actors,
		StartSequence: s.StartSequence,
		EndSequence:   s.EndSequence,
		EventCount:    s.EventCount,
	}
	if !s.StartedAt.IsZero() {
		out.StartedAt = s.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if !s.EndedAt.IsZero() {
		out.EndedAt = s.EndedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func visualizerTimeline(list replay.FrameList) controlplane.VisualizerTimeline {
	out := controlplane.VisualizerTimeline{
		SessionID: string(list.SessionID),
		Frames:    make([]controlplane.VisualizerFrame, 0, len(list.Frames)),
	}
	for _, f := range list.Frames {
		out.Frames = append(out.Frames, controlplane.VisualizerFrame{
			Index:    f.Index,
			Sequence: f.Sequence,
			EventID:  f.EventID,
			Type:     string(f.Type),
			Nodes:    len(f.Graph.Nodes),
			Edges:    len(f.Graph.Edges),
		})
	}
	return out
}

func visualizerGraph(graph visualizer.Graph) controlplane.VisualizerGraph {
	out := controlplane.VisualizerGraph{
		SessionID: string(graph.SessionID),
		Nodes:     make([]controlplane.VisualizerNode, 0, len(graph.Nodes)),
		Edges:     make([]controlplane.VisualizerEdge, 0, len(graph.Edges)),
	}
	for _, n := range graph.Nodes {
		out.Nodes = append(out.Nodes, controlplane.VisualizerNode{
			ID: string(n.ID), Kind: string(n.Kind), Label: n.Label, Ref: n.Ref,
			Sequence: n.Sequence, Metadata: n.Metadata,
		})
	}
	for _, e := range graph.Edges {
		out.Edges = append(out.Edges, controlplane.VisualizerEdge{
			From: string(e.From), To: string(e.To), Kind: string(e.Kind), Sequence: e.Sequence,
		})
	}
	return out
}
