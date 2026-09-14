// Package parallel coordinates multi-machine work placement.
//
// A coordinator assigns work packages to nodes deterministically and evenly so
// that dispatch is reproducible. It moves work; it never starts an agent
// runtime and therefore exposes no execution surface.
package parallel

import (
	"fmt"
	"sort"

	aasync "github.com/greadee/aa/sync"
)

// Assignment is the set of work packages placed on one node.
type Assignment struct {
	Node     aasync.NodeID `json:"node"`
	Packages []string      `json:"packages"`
}

// Coordinator places work packages across a fixed set of nodes.
type Coordinator struct {
	nodes []aasync.NodeID
}

// NewCoordinator returns a coordinator over the given nodes. Nodes are
// deduplicated and sorted so placement is deterministic.
func NewCoordinator(nodes []aasync.NodeID) (*Coordinator, error) {
	seen := map[aasync.NodeID]bool{}
	clean := make([]aasync.NodeID, 0, len(nodes))
	for _, n := range nodes {
		if n == "" {
			return nil, fmt.Errorf("%w: node id is required", aasync.ErrInvalid)
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		clean = append(clean, n)
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("%w: at least one node is required", aasync.ErrInvalid)
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i] < clean[j] })
	return &Coordinator{nodes: clean}, nil
}

// Nodes returns the coordinator's nodes in assignment order.
func (c *Coordinator) Nodes() []aasync.NodeID {
	out := make([]aasync.NodeID, len(c.nodes))
	copy(out, c.nodes)
	return out
}

// Assign places package IDs across the nodes in balanced round-robin order.
// Package IDs are sorted first, so the result is independent of input order.
func (c *Coordinator) Assign(packageIDs []string) ([]Assignment, error) {
	ids := append([]string(nil), packageIDs...)
	sort.Strings(ids)
	for _, id := range ids {
		if id == "" {
			return nil, fmt.Errorf("%w: work package id is required", aasync.ErrInvalid)
		}
	}
	assignments := make([]Assignment, len(c.nodes))
	for i, n := range c.nodes {
		assignments[i] = Assignment{Node: n}
	}
	for i, id := range ids {
		idx := i % len(c.nodes)
		assignments[idx].Packages = append(assignments[idx].Packages, id)
	}
	return assignments, nil
}

// MostLoaded returns the largest number of packages placed on any node.
func MostLoaded(assignments []Assignment) int {
	most := 0
	for _, a := range assignments {
		if len(a.Packages) > most {
			most = len(a.Packages)
		}
	}
	return most
}

// LeastLoaded returns the smallest number of packages placed on any node.
func LeastLoaded(assignments []Assignment) int {
	if len(assignments) == 0 {
		return 0
	}
	least := -1
	for _, a := range assignments {
		if least < 0 || len(a.Packages) < least {
			least = len(a.Packages)
		}
	}
	return least
}
