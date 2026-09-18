package visualizer

import (
	"errors"
	"testing"
)

func TestFixedClockAdvancesByStep(t *testing.T) {
	c := NewFixedClock()
	first := c.Now()
	second := c.Now()
	if !second.After(first) {
		t.Fatalf("clock did not advance: %v then %v", first, second)
	}
	if second.Sub(first).Seconds() != 1 {
		t.Fatalf("step = %v, want 1s", second.Sub(first))
	}
}

func TestGraphNodeByID(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "n1", Kind: NodeWorkPackage}, {ID: "n2", Kind: NodeActor}}}
	n, ok := g.NodeByID("n2")
	if !ok || n.Kind != NodeActor {
		t.Fatalf("NodeByID = %+v, %v", n, ok)
	}
	if _, ok := g.NodeByID("ghost"); ok {
		t.Fatal("NodeByID found a missing node")
	}
}

func TestSentinelErrorsAreDistinct(t *testing.T) {
	for _, pair := range [][2]error{
		{ErrInvalid, ErrNotFound},
		{ErrInvalid, ErrEmpty},
		{ErrEmpty, ErrBudget},
	} {
		if errors.Is(pair[0], pair[1]) {
			t.Fatalf("%v should not match %v", pair[0], pair[1])
		}
	}
	if !errors.Is(invalid("x"), ErrInvalid) {
		t.Fatal("invalid() does not wrap ErrInvalid")
	}
}
