package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/runtime/worker"
)

func runScheduler(t *testing.T, graph *planner.Graph, adapter worker.Adapter, concurrency, attempts int) *Scheduler {
	t.Helper()
	reg := role_allocator.New()
	if err := reg.Add(worker.Worker{ID: "w1", Capabilities: []string{"write_workspace"}, Available: true}); err != nil {
		t.Fatal(err)
	}
	o, err := New(graph, Config{
		Registry: reg, Runtime: adapter, Gates: []gate.Gate{gate.TestsGate{}},
		Enabled: true, Permitted: []string{"write_workspace"},
		MaxConcurrency: concurrency, MaxAttempts: attempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func independentGraph(ids ...string) *planner.Graph {
	g := planner.NewGraph("prj_1", "gph_1")
	for _, id := range ids {
		_ = g.Add(planner.WorkPackage{ID: id, Capabilities: []string{"write_workspace"}})
	}
	return g
}

func passedResult() worker.Result {
	return worker.Result{Status: "succeeded", Tests: []worker.TestResult{{Name: "t", Outcome: "passed"}}}
}

func TestRunCompletesDependencyChainInOrder(t *testing.T) {
	g := planner.NewGraph("prj_1", "gph_1")
	_ = g.Add(planner.WorkPackage{ID: "wp_a", Capabilities: []string{"write_workspace"}})
	_ = g.Add(planner.WorkPackage{ID: "wp_b", Capabilities: []string{"write_workspace"}, DependsOn: []string{"wp_a"}})
	fake := worker.NewFake()
	fake.Script("wp_a", passedResult())
	fake.Script("wp_b", passedResult())

	report, err := runScheduler(t, g, fake, 1, 1).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 2 || report.Failed != 0 || report.Dispatched != 2 {
		t.Fatalf("report = %+v", report)
	}
	calls := fake.Calls()
	if len(calls) != 2 || calls[0].WorkPackageID != "wp_a" || calls[1].WorkPackageID != "wp_b" {
		t.Fatalf("order = %+v", calls)
	}
}

type concurrencyProbe struct {
	current int32
	peak    int32
	delay   time.Duration
}

func (p *concurrencyProbe) Run(ctx context.Context, _ worker.Request) (worker.Result, error) {
	n := atomic.AddInt32(&p.current, 1)
	for {
		peak := atomic.LoadInt32(&p.peak)
		if n <= peak || atomic.CompareAndSwapInt32(&p.peak, peak, n) {
			break
		}
	}
	select {
	case <-time.After(p.delay):
	case <-ctx.Done():
		atomic.AddInt32(&p.current, -1)
		return worker.Result{}, ctx.Err()
	}
	atomic.AddInt32(&p.current, -1)
	return passedResult(), nil
}

func TestRunParallelizesUpToConcurrency(t *testing.T) {
	g := independentGraph("wp_a", "wp_b", "wp_c")
	probe := &concurrencyProbe{delay: 20 * time.Millisecond}

	if _, err := runScheduler(t, g, probe, 2, 1).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if peak := atomic.LoadInt32(&probe.peak); peak != 2 {
		t.Fatalf("peak concurrency = %d, want 2", peak)
	}
}

func TestRunSerialConcurrencyIsOne(t *testing.T) {
	g := independentGraph("wp_a", "wp_b")
	probe := &concurrencyProbe{delay: 5 * time.Millisecond}
	if _, err := runScheduler(t, g, probe, 1, 1).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if peak := atomic.LoadInt32(&probe.peak); peak != 1 {
		t.Fatalf("peak concurrency = %d, want 1", peak)
	}
}

type flakyAdapter struct {
	mu    sync.Mutex
	calls map[string]int
}

func (f *flakyAdapter) Run(_ context.Context, req worker.Request) (worker.Result, error) {
	f.mu.Lock()
	f.calls[req.WorkPackageID]++
	attempt := f.calls[req.WorkPackageID]
	f.mu.Unlock()
	if attempt == 1 {
		return worker.Result{Status: "failed", Failure: &worker.Failure{Class: "transient", Message: "boom"}}, nil
	}
	return passedResult(), nil
}

func TestRunRetriesTransientFailure(t *testing.T) {
	g := independentGraph("wp_a")
	mapCalls := &flakyAdapter{calls: map[string]int{}}
	report, err := runScheduler(t, g, mapCalls, 1, 2).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1 || report.Failed != 0 || report.Attempts["wp_a"] != 2 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunPartialFailureDoesNotAbort(t *testing.T) {
	g := independentGraph("wp_a", "wp_b")
	fake := worker.NewFake()
	fake.Script("wp_a", worker.Result{Status: "failed", Failure: &worker.Failure{Class: "x", Message: "boom"}})
	fake.Script("wp_b", passedResult())
	report, err := runScheduler(t, g, fake, 2, 1).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 1 || report.Accepted != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunHonorsCancellation(t *testing.T) {
	g := independentGraph("wp_a")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runScheduler(t, g, worker.NewFake(), 1, 1).Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
