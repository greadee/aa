package api

import (
	"context"
	"errors"
	"testing"

	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/kernel/orchestrator"
	"github.com/greadee/aa/kernel/plan"
	"github.com/greadee/aa/kernel/registry"
	"github.com/greadee/aa/runtime/worker"
)

func testOrchestrator(t *testing.T) *orchestrator.Orchestrator {
	t.Helper()
	graph := plan.NewGraph("prj_1", "gph_1")
	_ = graph.Add(plan.WorkPackage{ID: "wp_a", Trade: "backend", Capabilities: []string{"write_workspace", "run_tests"}})
	reg := registry.New()
	_ = reg.Add(registry.Worker{ID: "w1", Trade: "backend", Capabilities: []string{"write_workspace", "run_tests"}, Available: true})
	fake := worker.NewFake()
	fake.Script("wp_a", worker.Result{Status: "succeeded", Tests: []worker.TestResult{{Name: "t", Outcome: "passed"}}})
	human := gate.NewHumanGate()
	o, err := orchestrator.New(graph, orchestrator.Config{
		Registry: reg, Runtime: fake, Gates: []gate.Gate{gate.TestsGate{}, human},
		Enabled: true, Permitted: []string{"write_workspace", "run_tests"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestNewRequiresOrchestrator(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected orchestrator requirement")
	}
}

func TestDisabledServiceRefusesDispatch(t *testing.T) {
	svc, err := New(Config{Orchestrator: testOrchestrator(t), Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if svc.Enabled() {
		t.Fatal("should be disabled")
	}
	if _, _, err := svc.Dispatch(context.Background()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnabledServiceRuns(t *testing.T) {
	svc, err := New(Config{Orchestrator: testOrchestrator(t), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	report, dispatched, err := svc.Dispatch(context.Background())
	if err != nil || !dispatched {
		t.Fatalf("dispatched=%v err=%v", dispatched, err)
	}
	if report.Status != "pending" {
		t.Fatalf("report = %+v", report)
	}
	svc.Approve("wp_a")
	resolved, err := svc.Resolve("wp_a")
	if err != nil || resolved.Status != "succeeded" {
		t.Fatalf("resolved = %+v err=%v", resolved, err)
	}
	status := svc.Status()
	if len(status.WorkPackages) != 1 || status.WorkPackages[0].State != plan.StateCompleted {
		t.Fatalf("status = %+v", status)
	}
}
