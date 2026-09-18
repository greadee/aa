// Package session folds an event stream into a deterministic graph and
// re-derives stable node identity.
//
// Projection is a pure function of the events: the same events always yield the
// same graph, independent of delivery order, and node identity is derived from
// the aggregate kind + id rather than persisted.
package session

import (
	"fmt"
	"sort"
	"strings"
	"time"

	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/compat"
)

// NodeIDFor derives the stable identity for an aggregate kind and reference.
func NodeIDFor(kind visualizer.NodeKind, ref string) visualizer.NodeID {
	return visualizer.NodeID(string(kind) + ":" + ref)
}

// SessionNodeID derives the identity of a session node.
func SessionNodeID(id visualizer.SessionID) visualizer.NodeID {
	return NodeIDFor(visualizer.NodeSession, string(id))
}

// ActorNodeID derives the identity of an actor node.
func ActorNodeID(id string) visualizer.NodeID {
	return NodeIDFor(visualizer.NodeActor, id)
}

// Project folds events into a deterministic graph for a session.
func Project(sessionID visualizer.SessionID, events []visualizer.Event) (visualizer.Graph, error) {
	if sessionID == "" {
		return visualizer.Graph{}, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	if len(events) == 0 {
		return visualizer.Graph{}, fmt.Errorf("%w: session %q", visualizer.ErrEmpty, sessionID)
	}
	ordered := orderedCopy(events)

	nodes := newNodes()
	edges := newEdges()
	byEventID := map[string]visualizer.NodeID{}
	firstByCorrelation := map[string]visualizer.NodeID{}

	sessionNode := nodes.ensure(SessionNodeID(sessionID), visualizer.NodeSession, string(sessionID), string(sessionID), ordered[0].Sequence)

	for _, e := range ordered {
		agg := nodes.ensure(NodeIDFor(visualizer.NodeKind(e.Aggregate.Kind), e.Aggregate.ID),
			visualizer.NodeKind(e.Aggregate.Kind), e.Aggregate.ID, e.Aggregate.ID, e.Sequence)
		if agg.ID != sessionNode.ID {
			edges.add(sessionNode.ID, agg.ID, visualizer.EdgeContains, e.Sequence)
		}
		byEventID[e.ID] = agg.ID

		if e.Actor.ID != "" {
			actor := nodes.ensure(ActorNodeID(e.Actor.ID), visualizer.NodeActor, e.Actor.ID, e.Actor.ID, e.Sequence)
			edges.add(actor.ID, agg.ID, visualizer.EdgeAccess, e.Sequence)
		}

		if e.CausationID != "" {
			if cause, ok := byEventID[e.CausationID]; ok && cause != agg.ID {
				edges.add(cause, agg.ID, visualizer.EdgeCausation, e.Sequence)
			}
		}
		if e.CorrelationID != "" {
			if first, ok := firstByCorrelation[e.CorrelationID]; ok {
				if first != agg.ID {
					edges.add(first, agg.ID, visualizer.EdgeCorrelation, e.Sequence)
				}
			} else {
				firstByCorrelation[e.CorrelationID] = agg.ID
			}
		}

		metadata, err := compat.Normalize(e)
		if err != nil {
			return visualizer.Graph{}, err
		}
		if !metadata.Empty() {
			nodes.annotate(agg.ID, metadata)
		}

		if wp, ok := payloadString(e.Payload, "workPackageId"); ok {
			wpNode := nodes.ensure(NodeIDFor(visualizer.NodeWorkPackage, wp), visualizer.NodeWorkPackage, wp, wp, e.Sequence)
			if wpNode.ID != agg.ID {
				edges.add(agg.ID, wpNode.ID, visualizer.EdgeAssigns, e.Sequence)
			}
		}
		if artifact, ok := payloadString(e.Payload, "artifactId"); ok {
			artifactNode := nodes.ensure(NodeIDFor(visualizer.NodeArtifact, artifact), visualizer.NodeArtifact, artifact, artifact, e.Sequence)
			if artifactNode.ID != agg.ID {
				edges.add(agg.ID, artifactNode.ID, visualizer.EdgeProduces, e.Sequence)
			}
		}
	}

	return visualizer.Graph{
		SessionID: sessionID,
		Nodes:     nodes.sorted(),
		Edges:     edges.sorted(),
	}, nil
}

// Summarize derives a session summary from the events.
func Summarize(sessionID visualizer.SessionID, events []visualizer.Event) (visualizer.Session, error) {
	if len(events) == 0 {
		return visualizer.Session{}, fmt.Errorf("%w: session %q", visualizer.ErrEmpty, sessionID)
	}
	ordered := orderedCopy(events)
	summary := visualizer.Session{
		ID:            sessionID,
		StartSequence: ordered[0].Sequence,
		EndSequence:   ordered[len(ordered)-1].Sequence,
		EventCount:    len(ordered),
	}
	actors := map[string]bool{}
	for _, e := range ordered {
		if e.Actor.ID != "" {
			actors[e.Actor.ID] = true
		}
		if summary.ProjectID == "" && e.ProjectID != "" {
			summary.ProjectID = e.ProjectID
		}
	}
	for id := range actors {
		summary.Actors = append(summary.Actors, id)
	}
	sort.Strings(summary.Actors)
	if t, err := time.Parse(time.RFC3339, ordered[0].OccurredAt); err == nil {
		summary.StartedAt = t
	}
	if t, err := time.Parse(time.RFC3339, ordered[len(ordered)-1].OccurredAt); err == nil {
		summary.EndedAt = t
	}
	return summary, nil
}

func orderedCopy(events []visualizer.Event) []visualizer.Event {
	out := append([]visualizer.Event(nil), events...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}

func payloadString(payload map[string]any, key string) (string, bool) {
	if payload == nil {
		return "", false
	}
	v, ok := payload[key].(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

type nodeSet struct {
	nodes map[visualizer.NodeID]visualizer.Node
}

func newNodes() *nodeSet {
	return &nodeSet{nodes: map[visualizer.NodeID]visualizer.Node{}}
}

func (s *nodeSet) ensure(id visualizer.NodeID, kind visualizer.NodeKind, label, ref string, seq int) visualizer.Node {
	if n, ok := s.nodes[id]; ok {
		return n
	}
	n := visualizer.Node{ID: id, Kind: kind, Label: label, Ref: ref, Sequence: seq}
	s.nodes[id] = n
	return n
}

func (s *nodeSet) annotate(id visualizer.NodeID, m compat.Metadata) {
	n, ok := s.nodes[id]
	if !ok {
		return
	}
	if n.Metadata == nil {
		n.Metadata = map[string]string{}
	}
	if len(m.SecondaryPaths) > 0 {
		n.Metadata[compat.KeySecondaryPaths] = strings.Join(m.SecondaryPaths, ",")
	}
	if len(m.AccessSequence) > 0 {
		n.Metadata[compat.KeyAccessSequence] = strings.Join(m.AccessSequence, ",")
	}
	s.nodes[id] = n
}

func (s *nodeSet) sorted() []visualizer.Node {
	out := make([]visualizer.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type edgeSet struct {
	edges map[string]visualizer.Edge
}

func newEdges() *edgeSet {
	return &edgeSet{edges: map[string]visualizer.Edge{}}
}

func (s *edgeSet) add(from, to visualizer.NodeID, kind visualizer.EdgeKind, seq int) {
	key := string(from) + "\x00" + string(to) + "\x00" + string(kind)
	if existing, ok := s.edges[key]; ok {
		if seq < existing.Sequence {
			existing.Sequence = seq
			s.edges[key] = existing
		}
		return
	}
	s.edges[key] = visualizer.Edge{From: from, To: to, Kind: kind, Sequence: seq}
}

func (s *edgeSet) sorted() []visualizer.Edge {
	out := make([]visualizer.Edge, 0, len(s.edges))
	for _, e := range s.edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}
