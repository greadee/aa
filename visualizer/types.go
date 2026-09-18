package visualizer

import (
	"time"

	"github.com/greadee/aa/contracts/go/v1"
)

// Event is the shared canonical event consumed by the visualizer. It is not a
// new observation protocol; it is the contracts taxonomy behind a seam.
type Event = v1.Event

type (
	// EventType enumerates the canonical event types.
	EventType = v1.EventType
	// EventAggregate identifies the entity an event applies to.
	EventAggregate = v1.EventAggregate
	// Actor identifies who produced or performed something.
	Actor = v1.Actor
)

// SessionID identifies an observed session: work correlated under one id.
type SessionID string

// NodeID is a re-derived identity for a graph node. It is stable for a given
// aggregate but is never persisted as truth.
type NodeID string

// NodeKind is the kind of a graph node.
type NodeKind string

// Node kinds.
const (
	NodeSession     NodeKind = "session"
	NodeProject     NodeKind = "project"
	NodeWorkPackage NodeKind = "work_package"
	NodeAssignment  NodeKind = "assignment"
	NodeAttempt     NodeKind = "attempt"
	NodeArtifact    NodeKind = "artifact"
	NodeIssue       NodeKind = "issue"
	NodeStrategy    NodeKind = "strategy"
	NodeMemory      NodeKind = "memory"
	NodeTool        NodeKind = "tool"
	NodeWorkflow    NodeKind = "workflow"
	NodeActor       NodeKind = "actor"
)

// Node is a graph vertex derived from the event stream.
type Node struct {
	ID       NodeID            `json:"id"`
	Kind     NodeKind          `json:"kind"`
	Label    string            `json:"label,omitempty"`
	Ref      string            `json:"ref,omitempty"`
	Sequence int               `json:"sequence"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EdgeKind is the relationship an edge expresses.
type EdgeKind string

// Edge kinds.
const (
	EdgeCorrelation EdgeKind = "correlation"
	EdgeCausation   EdgeKind = "causation"
	EdgeContains    EdgeKind = "contains"
	EdgeProduces    EdgeKind = "produces"
	EdgeAssigns     EdgeKind = "assigns"
	EdgeAccess      EdgeKind = "access"
)

// Edge is a graph relation derived from the event stream.
type Edge struct {
	From     NodeID   `json:"from"`
	To       NodeID   `json:"to"`
	Kind     EdgeKind `json:"kind"`
	Sequence int      `json:"sequence"`
}

// Graph is a deterministic projection of one session.
type Graph struct {
	SessionID SessionID `json:"sessionId"`
	Nodes     []Node    `json:"nodes"`
	Edges     []Edge    `json:"edges"`
}

// NodeByID returns a node by id.
func (g Graph) NodeByID(id NodeID) (Node, bool) {
	for _, n := range g.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// Session summarizes an observed session.
type Session struct {
	ID            SessionID `json:"id"`
	ProjectID     string    `json:"projectId,omitempty"`
	Actors        []string  `json:"actors,omitempty"`
	StartSequence int       `json:"startSequence"`
	EndSequence   int       `json:"endSequence"`
	EventCount    int       `json:"eventCount"`
	StartedAt     time.Time `json:"startedAt,omitempty"`
	EndedAt       time.Time `json:"endedAt,omitempty"`
}
