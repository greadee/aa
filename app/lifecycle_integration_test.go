package main

import (
	"context"
	"testing"

	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	"github.com/greadee/aa/kernel/scheduler"
	"github.com/greadee/aa/runtime/lifecycle"
	"github.com/greadee/aa/runtime/worker"
)

func lifecycleGraph(t *testing.T) *planner.Graph {
	t.Helper()
	graph := planner.NewGraph("prj_life", "gph_life")
	if err := graph.Add(planner.WorkPackage{ID: "wp_life", Title: "work", Capabilities: []string{"write_workspace"}}); err != nil {
		t.Fatal(err)
	}
	return graph
}

func lifecycleRegistry(t *testing.T) *role_allocator.Registry {
	t.Helper()
	reg := role_allocator.New()
	if err := reg.Add(worker.Worker{ID: "w1", Capabilities: []string{"write_workspace"}, Available: true}); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestSchedulerDispatchesThroughLifecycle(t *testing.T) {
	fake := worker.NewFake()
	fake.Script("wp_life", worker.Result{Status: "succeeded"})
	manager := lifecycle.New(lifecycle.Config{Adapter: fake})

	sched, err := scheduler.New(lifecycleGraph(t), scheduler.Config{
		Registry:  lifecycleRegistry(t),
		Runtime:   manager,
		Enabled:   true,
		Permitted: []string{"write_workspace"},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := sched.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1 || fake.CallCount() != 1 {
		t.Fatalf("report = %+v calls = %d", report, fake.CallCount())
	}
	workers := manager.Workers()
	if len(workers) != 1 || workers[0].State != lifecycle.StateStopped {
		t.Fatalf("workers = %+v", workers)
	}
	if len(manager.Events()) != 4 {
		t.Fatalf("events = %+v", manager.Events())
	}
}

func TestDisabledDispatchIsStillObserved(t *testing.T) {
	fake := worker.NewFake()
	fake.Script("wp_life", worker.Result{Status: "succeeded"})
	manager := lifecycle.New(lifecycle.Config{Adapter: fake})

	sched, err := scheduler.New(lifecycleGraph(t), scheduler.Config{
		Registry:  lifecycleRegistry(t),
		Lifecycle: manager,
		Enabled:   false,
		Permitted: []string{"write_workspace"},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, dispatched, err := sched.Dispatch(context.Background())
	if err != nil || !dispatched {
		t.Fatalf("dispatch=%v err=%v", dispatched, err)
	}
	if report.Status != "failed" {
		t.Fatalf("report = %+v", report)
	}
	if fake.CallCount() != 0 {
		t.Fatalf("adapter ran while disabled: %d", fake.CallCount())
	}
	workers := manager.Workers()
	if len(workers) != 1 || workers[0].State != lifecycle.StateStopped || workers[0].SelectedWorker != "w1" {
		t.Fatalf("workers = %+v", workers)
	}
}
