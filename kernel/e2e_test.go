package kernel

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greadee/aa/kernel/allocator"
	"github.com/greadee/aa/kernel/allocator/compute_allocator"
	"github.com/greadee/aa/kernel/allocator/model_allocator"
	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/kernel/observability"
	"github.com/greadee/aa/kernel/scheduler"
	"github.com/greadee/aa/registry/models"
	"github.com/greadee/aa/runtime/sandbox"
	"github.com/greadee/aa/runtime/worker"
	"github.com/greadee/aa/toolbox/computeruse"
)

func e2eModels(t *testing.T) *models.Registry {
	t.Helper()
	reg, err := models.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func e2eGraph(units []planner.WorkPackage) *planner.Graph {
	g := planner.NewGraph("prj_1", "gph_1")
	for _, unit := range units {
		_ = g.Add(unit)
	}
	return g
}

func e2eScheduler(t *testing.T, g *planner.Graph, adapter worker.Adapter, concurrency int) *scheduler.Scheduler {
	t.Helper()
	reg := role_allocator.New()
	if err := reg.Add(worker.Worker{ID: "w1", Capabilities: []string{"write_workspace", "run_tests", "read_project"}, Available: true}); err != nil {
		t.Fatal(err)
	}
	s, err := scheduler.New(g, scheduler.Config{
		Registry: reg, Runtime: adapter, Gates: []gate.Gate{gate.TestsGate{}}, Enabled: true,
		Permitted: []string{"write_workspace", "run_tests", "read_project"}, MaxConcurrency: concurrency,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func e2eIdentity() observability.Identity {
	return observability.Identity{
		Organization: "acme", Project: "aa", Subtask: "wp_1", Crew: "crew_a",
		Worker: "w1", Role: "Builder", Team: "site", Model: "ollama/qwen3.5:9b",
	}
}

func TestE2ESimpleTaskSingleWorker(t *testing.T) {
	reg := e2eModels(t)
	plan, err := allocator.Allocate(allocator.Request{
		ProjectID: "prj_1", PlanID: "wp_1",
		Units: []allocator.Unit{{ID: "wp_1", Capabilities: []string{"write_workspace"}, RequiresModel: true,
			Model: model_allocator.Requirement{Capabilities: []string{"coding"}, Locality: "local"}}},
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Concurrency != 1 || len(plan.Assignments) != 1 || plan.Assignments[0].Model == "" {
		t.Fatalf("plan = %+v", plan)
	}

	fake := worker.NewFake()
	fake.Script("wp_1", worker.Result{Status: "succeeded", Tests: []worker.TestResult{{Name: "t", Outcome: "passed"}}})
	g := e2eGraph([]planner.WorkPackage{{ID: "wp_1", Capabilities: []string{"write_workspace"}}})
	report, err := e2eScheduler(t, g, fake, plan.Concurrency).Run(context.Background())
	if err != nil || report.Accepted != 1 {
		t.Fatalf("run = %+v err=%v", report, err)
	}

	rec := observability.NewMemRecorder()
	_ = rec.Record(observability.Event{Name: observability.TaskReceived, CorrelationID: "wp_1", Identity: e2eIdentity()})
	_ = rec.Record(observability.Event{Name: observability.SchedulerDispatch, CorrelationID: "wp_1", Identity: e2eIdentity()})
	_ = rec.Record(observability.Event{Name: observability.AgentResult, CorrelationID: "wp_1", Identity: e2eIdentity(), Tokens: 120, CostUSD: 0.001, ComputeMS: 40})
	m := observability.Summarize(rec.Events())
	if m.Events != 3 || m.Tokens != 120 || m.ComputeMS != 40 {
		t.Fatalf("metrics = %+v", m)
	}
	accounts := observability.Accounting(rec.Events())
	if len(accounts) != 1 || !strings.HasPrefix(accounts[0].Chain, "acme/aa/wp_1/") {
		t.Fatalf("accounts = %+v", accounts)
	}
}

func TestE2EDifficultTaskEscalatesCapability(t *testing.T) {
	reg := e2eModels(t)
	plan, err := allocator.Allocate(allocator.Request{
		ProjectID: "prj_1", PlanID: "wp_1",
		Units: []allocator.Unit{{ID: "wp_1", Capabilities: []string{"write_workspace"}, RequiresModel: true,
			Model: model_allocator.Requirement{Capabilities: []string{"reasoning"}, Locality: "local"}}},
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	// A reasoning-capable local model is chosen (cheapest reasoning class).
	if plan.Assignments[0].Model != "mdl_ollama_mistral-nemo:12b" {
		t.Fatalf("expected a reasoning model, got %q", plan.Assignments[0].Model)
	}
}

type e2eProbe struct {
	current int32
	peak    int32
	delay   time.Duration
}

func (p *e2eProbe) Run(ctx context.Context, _ worker.Request) (worker.Result, error) {
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
	return worker.Result{Status: "succeeded", Tests: []worker.TestResult{{Name: "t", Outcome: "passed"}}}, nil
}

func TestE2EDecomposableTaskParallelAggregation(t *testing.T) {
	reg := e2eModels(t)
	plan, err := allocator.Allocate(allocator.Request{
		ProjectID: "prj_1", PlanID: "wp_1",
		Units: []allocator.Unit{
			{ID: "u1", Capabilities: []string{"write_workspace"}},
			{ID: "u2", Capabilities: []string{"write_workspace"}},
			{ID: "u3", Capabilities: []string{"run_tests"}},
		},
		Compute: compute_allocator.Input{Units: 3, IndependentUnits: 3, Justification: compute_allocator.JustifyCoverage, MaxConcurrency: 2},
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Concurrency != 2 {
		t.Fatalf("plan concurrency = %d", plan.Concurrency)
	}

	probe := &e2eProbe{delay: 20 * time.Millisecond}
	g := e2eGraph([]planner.WorkPackage{
		{ID: "u1", Capabilities: []string{"write_workspace"}},
		{ID: "u2", Capabilities: []string{"write_workspace"}},
		{ID: "u3", Capabilities: []string{"run_tests"}},
	})
	report, err := e2eScheduler(t, g, probe, plan.Concurrency).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 3 || report.Dispatched != 3 {
		t.Fatalf("report = %+v", report)
	}
	if peak := atomic.LoadInt32(&probe.peak); peak != 2 {
		t.Fatalf("peak concurrency = %d, want 2", peak)
	}
}

func TestE2EToolUsingTaskThroughSandbox(t *testing.T) {
	// The computer-use host is built in toolchain_test.go; drive it and record a
	// structured tool event attributed to the identity chain.
	h, driver := computerUseHost(t, sandbox.Enforcer{})
	result, err := h.Invoke(context.Background(), screenshotInvocation(), computerUseContract())
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["permission"] != computeruse.PermissionScreenCapture {
		t.Fatalf("output = %+v", result.Output)
	}
	if len(driver.Calls()) != 1 {
		t.Fatalf("driver calls = %d", len(driver.Calls()))
	}

	rec := observability.NewMemRecorder()
	_ = rec.Record(observability.Event{
		Name: observability.ToolCall, CorrelationID: "wp_1", Identity: e2eIdentity(),
		Attributes: map[string]any{"tool": "tool_screen", "action": "screenshot"},
	})
	if got := rec.Events(); len(got) != 1 || got[0].Name != observability.ToolCall {
		t.Fatalf("events = %+v", got)
	}
}
