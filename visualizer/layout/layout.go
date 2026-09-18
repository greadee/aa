// Package layout places a projected graph deterministically in 3D.
//
// Layout is a pure function of the graph: the same nodes always receive the
// same positions, independent of the order they are supplied in. It does no
// wall-clock-dependent or random work, so replay is reproducible. A budget
// bounds the graph a single pass will accept.
package layout

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"

	visualizer "github.com/greadee/aa/visualizer"
)

const (
	layerGap     = 6.0
	baseRadius   = 4.0
	ringStep     = 0.6
	goldenAngle  = 2.399963229728653
	jitterAmount = 0.25
)

// Point is a 3D position.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Placement maps a node's identity to its position.
type Placement map[visualizer.NodeID]Point

// Budget bounds a single layout pass.
type Budget struct {
	MaxNodes int
	MaxEdges int
}

// DefaultBudget is the large-graph budget.
func DefaultBudget() Budget {
	return Budget{MaxNodes: 250000, MaxEdges: 1000000}
}

// Layout places every node using the default budget.
func Layout(g visualizer.Graph) (Placement, error) {
	return LayoutBudgeted(g, DefaultBudget())
}

// LayoutBudgeted places every node deterministically, or returns ErrBudget.
//
// Nodes are grouped by kind into layers; within a layer nodes are ordered by id
// and spread by the golden angle. Positions depend only on the sorted graph, so
// they are stable across replay.
func LayoutBudgeted(g visualizer.Graph, b Budget) (Placement, error) {
	if len(g.Nodes) > b.MaxNodes {
		return nil, fmt.Errorf("%w: %d nodes exceeds %d", visualizer.ErrBudget, len(g.Nodes), b.MaxNodes)
	}
	if len(g.Edges) > b.MaxEdges {
		return nil, fmt.Errorf("%w: %d edges exceeds %d", visualizer.ErrBudget, len(g.Edges), b.MaxEdges)
	}

	byKind := make(map[visualizer.NodeKind][]visualizer.Node, len(g.Nodes))
	for _, n := range g.Nodes {
		byKind[n.Kind] = append(byKind[n.Kind], n)
	}
	kinds := make([]visualizer.NodeKind, 0, len(byKind))
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	placement := make(Placement, len(g.Nodes))
	for layer, kind := range kinds {
		nodes := byKind[kind]
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
		count := len(nodes)
		radius := baseRadius + ringStep*math.Sqrt(float64(count))
		z := float64(layer) * layerGap
		for i, node := range nodes {
			angle := goldenAngle * float64(i)
			r := radius
			if count > 1 {
				r = radius * (0.85 + 0.3*float64(i)/float64(count-1))
			}
			jx, jy, jz := jitter(node.ID)
			placement[node.ID] = Point{
				X: r*math.Cos(angle) + jx,
				Y: r*math.Sin(angle) + jy,
				Z: z + jz,
			}
		}
	}
	return placement, nil
}

// Bounds returns the axis-aligned bounds of a placement.
func Bounds(p Placement) (min, max Point) {
	first := true
	for _, pt := range p {
		if first {
			min, max, first = pt, pt, false
			continue
		}
		min.X, max.X = math.Min(min.X, pt.X), math.Max(max.X, pt.X)
		min.Y, max.Y = math.Min(min.Y, pt.Y), math.Max(max.Y, pt.Y)
		min.Z, max.Z = math.Min(min.Z, pt.Z), math.Max(max.Z, pt.Z)
	}
	return min, max
}

// jitter derives a small deterministic offset from a node id.
func jitter(id visualizer.NodeID) (float64, float64, float64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	sum := h.Sum64()
	unit := func(shift uint) float64 {
		v := (sum >> shift) & 0xffff
		return (float64(v)/float64(0xffff))*2 - 1
	}
	return unit(0) * jitterAmount, unit(16) * jitterAmount, unit(32) * jitterAmount
}
