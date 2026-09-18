package layout

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	visualizer "github.com/greadee/aa/visualizer"
)

func graph(n int) visualizer.Graph {
	g := visualizer.Graph{SessionID: "s1"}
	for i := 0; i < n; i++ {
		g.Nodes = append(g.Nodes, visualizer.Node{
			ID:   visualizer.NodeID(fmt.Sprintf("n%06d", i)),
			Kind: visualizer.NodeWorkPackage,
		})
	}
	return g
}

func TestLayoutIsDeterministic(t *testing.T) {
	g := sessionGraph()
	a, err := Layout(g)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	b, err := Layout(g)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("layout is not deterministic")
	}
	if len(a) != len(g.Nodes) {
		t.Fatalf("placed %d of %d nodes", len(a), len(g.Nodes))
	}
}

func TestLayoutIsOrderIndependent(t *testing.T) {
	g := sessionGraph()
	shuffled := g
	shuffled.Nodes = []visualizer.Node{g.Nodes[2], g.Nodes[0], g.Nodes[1]}
	a, _ := Layout(g)
	b, _ := Layout(shuffled)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("layout depends on input order")
	}
}

func TestLayoutSpreadAndBounds(t *testing.T) {
	p, err := Layout(graph(20))
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	seen := map[Point]bool{}
	for _, pt := range p {
		seen[pt] = true
	}
	if len(seen) < 2 {
		t.Fatal("all placements collapsed to one point")
	}
	min, max := Bounds(p)
	if min == max {
		t.Fatalf("degenerate bounds: %+v", min)
	}
}

func TestLayoutBudget(t *testing.T) {
	_, err := LayoutBudgeted(graph(11), Budget{MaxNodes: 10, MaxEdges: 10})
	if !errors.Is(err, visualizer.ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
	g := graph(1)
	g.Edges = []visualizer.Edge{{From: "a", To: "b", Kind: visualizer.EdgeContains}}
	_, err = LayoutBudgeted(g, Budget{MaxNodes: 10, MaxEdges: 0})
	if !errors.Is(err, visualizer.ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
}

func TestLayoutLargeGraphWithinBudget(t *testing.T) {
	const n = 50000
	start := time.Now()
	p, err := LayoutBudgeted(graph(n), DefaultBudget())
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	elapsed := time.Since(start)
	if len(p) != n {
		t.Fatalf("placed %d of %d", len(p), n)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("layout took %v", elapsed)
	}
}

func BenchmarkLayoutLargeGraph(b *testing.B) {
	g := graph(100000)
	for i := 0; i < b.N; i++ {
		if _, err := LayoutBudgeted(g, DefaultBudget()); err != nil {
			b.Fatal(err)
		}
	}
}

// sessionGraph builds a small multi-kind graph for layout tests.
func sessionGraph() visualizer.Graph {
	return visualizer.Graph{
		SessionID: "s1",
		Nodes: []visualizer.Node{
			{ID: "session:s1", Kind: visualizer.NodeSession},
			{ID: "work_package:wp1", Kind: visualizer.NodeWorkPackage},
			{ID: "actor:worker_1", Kind: visualizer.NodeActor},
		},
	}
}
