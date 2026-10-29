package planner

import (
	"fmt"
	"sync"
	"testing"
)

func testGraph(t *testing.T) *Graph {
	t.Helper()
	g := NewGraph("prj_1", "gph_1")
	if err := g.Add(WorkPackage{ID: "wp_a", Title: "a", DependsOn: nil}); err != nil {
		t.Fatal(err)
	}
	if err := g.Add(WorkPackage{ID: "wp_b", Title: "b", DependsOn: []string{"wp_a"}}); err != nil {
		t.Fatal(err)
	}
	if err := g.Add(WorkPackage{ID: "wp_c", Title: "c", DependsOn: []string{"wp_a"}}); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestAddAndGet(t *testing.T) {
	g := testGraph(t)
	wp, ok := g.Get("wp_a")
	if !ok || wp.State != StateProposed {
		t.Fatalf("got %+v ok=%v", wp, ok)
	}
	if err := g.Add(WorkPackage{ID: "wp_a"}); err == nil {
		t.Fatal("expected duplicate error")
	}
	if len(g.List()) != 3 {
		t.Fatalf("list = %v", g.List())
	}
}

func TestValidate(t *testing.T) {
	g := NewGraph("p", "g")
	_ = g.Add(WorkPackage{ID: "a", DependsOn: []string{"missing"}})
	if err := g.Validate(); err == nil {
		t.Fatal("expected missing dependency error")
	}

	g2 := NewGraph("p", "g")
	_ = g2.Add(WorkPackage{ID: "a", DependsOn: []string{"b"}})
	_ = g2.Add(WorkPackage{ID: "b", DependsOn: []string{"a"}})
	if err := g2.Validate(); err == nil {
		t.Fatal("expected cycle error")
	}

	g3 := NewGraph("p", "g")
	_ = g3.Add(WorkPackage{ID: "a", DependsOn: []string{"a"}})
	if err := g3.Validate(); err == nil {
		t.Fatal("expected self-dependency error")
	}
}

func TestTopologicalOrder(t *testing.T) {
	g := testGraph(t)
	order, err := g.Topological()
	if err != nil {
		t.Fatal(err)
	}
	if order[0] != "wp_a" {
		t.Fatalf("order = %v", order)
	}
	if order[1] != "wp_b" || order[2] != "wp_c" {
		t.Fatalf("order = %v", order)
	}
}

func TestReadiness(t *testing.T) {
	g := testGraph(t)
	ready := g.Ready()
	if len(ready) != 1 || ready[0] != "wp_a" {
		t.Fatalf("initial ready = %v", ready)
	}
	if err := g.SetState("wp_a", StateCompleted); err != nil {
		t.Fatal(err)
	}
	ready = g.Ready()
	if len(ready) != 2 || ready[0] != "wp_b" || ready[1] != "wp_c" {
		t.Fatalf("ready after a = %v", ready)
	}
	if err := g.SetState("wp_b", StateRunning); err != nil {
		t.Fatal(err)
	}
	ready = g.Ready()
	if len(ready) != 1 || ready[0] != "wp_c" {
		t.Fatalf("ready after b running = %v", ready)
	}
	if err := g.SetState("nope", StateReady); err == nil {
		t.Fatal("expected unknown node error")
	}
}

func TestGraphConcurrentReadsAndWrites(t *testing.T) {
	g := NewGraph("p1", "g1")
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("wp_%d", i)
		if err := g.Add(WorkPackage{ID: id, Title: id}); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("wp_%d", i)
			for j := 0; j < 100; j++ {
				_, _ = g.Get(id)
				_ = g.List()
				_ = g.Ready()
				_ = g.SetState(id, StateRunning)
				_ = g.SetState(id, StateProposed)
			}
		}(i)
	}
	wg.Wait()
}
