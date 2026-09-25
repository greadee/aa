package orchestrator

import (
	"context"
	"testing"

	"github.com/greadee/aa/kernel/allocator/planner"
	kcontext "github.com/greadee/aa/kernel/context"
	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/kernel/registry"
	"github.com/greadee/aa/runtime/worker"
)

type recSink struct {
	events []string
}

func (s *recSink) Event(_ int, eventType string, _ map[string]any) error {
	s.events = append(s.events, eventType)
	return nil
}

func (s *recSink) Record(string, string, []byte) error { return nil }

func newTestOrchestrator(t *testing.T, enabled bool, sink Sink) (*Orchestrator, *worker.Fake, *gate.HumanGate) {
	t.Helper()
	graph := planner.NewGraph("prj_1", "gph_1")
	if err := graph.Add(planner.WorkPackage{ID: "wp_a", Title: "a", Trade: "backend", Capabilities: []string{"write_workspace", "run_tests"}}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Add(planner.WorkPackage{ID: "wp_b", Title: "b", Trade: "backend", Capabilities: []string{"write_workspace", "run_tests"}, DependsOn: []string{"wp_a"}}); err != nil {
		t.Fatal(err)
	}

	reg := registry.New()
	if err := reg.Add(registry.Worker{ID: "w1", Trade: "backend", Capabilities: []string{"write_workspace", "run_tests"}, Available: true}); err != nil {
		t.Fatal(err)
	}

	fake := worker.NewFake()
	fake.Script("wp_a", worker.Result{Status: "succeeded", Summary: "a", Tests: []worker.TestResult{{Name: "ta", Outcome: "passed"}}})
	fake.Script("wp_b", worker.Result{Status: "succeeded", Summary: "b", Tests: []worker.TestResult{{Name: "tb", Outcome: "passed"}}})

	human := gate.NewHumanGate()
	o, err := New(graph, Config{
		Registry:  reg,
		Runtime:   fake,
		Gates:     []gate.Gate{gate.TestsGate{}, human},
		Enabled:   enabled,
		Permitted: []string{"write_workspace", "run_tests"},
		Sink:      sink,
		ContextInputs: func(workPackageID string) []kcontext.Input {
			return []kcontext.Input{{Kind: "issue", ID: "iss_1", Text: "do " + workPackageID}}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return o, fake, human
}

func TestSupervisedRun(t *testing.T) {
	sink := &recSink{}
	o, fake, _ := newTestOrchestrator(t, true, sink)

	// wp_a dispatches first and waits on the human gate.
	report, dispatched, err := o.Dispatch(context.Background())
	if err != nil || !dispatched {
		t.Fatalf("dispatch dispatched=%v err=%v", dispatched, err)
	}
	if report.WorkPackageID != "wp_a" || report.Status != "pending" {
		t.Fatalf("report = %+v", report)
	}

	o.Approve("wp_a")
	resolved, err := o.Resolve("wp_a")
	if err != nil || resolved.Status != "succeeded" {
		t.Fatalf("resolve = %+v err=%v", resolved, err)
	}

	// wp_b becomes ready only after wp_a completes.
	report, dispatched, err = o.Dispatch(context.Background())
	if err != nil || !dispatched || report.WorkPackageID != "wp_b" || report.Status != "pending" {
		t.Fatalf("second dispatch = %+v dispatched=%v err=%v", report, dispatched, err)
	}
	o.Approve("wp_b")
	if resolved, err = o.Resolve("wp_b"); err != nil || resolved.Status != "succeeded" {
		t.Fatalf("second resolve = %+v err=%v", resolved, err)
	}

	// No more ready work.
	if _, dispatched, _ = o.Dispatch(context.Background()); dispatched {
		t.Fatal("expected no more work")
	}

	status := o.Status()
	for _, wp := range status.WorkPackages {
		if wp.State != planner.StateCompleted {
			t.Fatalf("work package %s state = %s", wp.ID, wp.State)
		}
	}
	if len(status.Assignments) != 2 {
		t.Fatalf("assignments = %+v", status.Assignments)
	}
	if fake.CallCount() != 2 {
		t.Fatalf("runtime calls = %d", fake.CallCount())
	}
	if len(o.Candidates()) != 2 {
		t.Fatalf("candidates = %+v", o.Candidates())
	}
	if len(o.Telemetry()) != 2 {
		t.Fatalf("telemetry = %+v", o.Telemetry())
	}
	if len(sink.events) == 0 {
		t.Fatal("expected sink events")
	}
}

func TestExecutionDisabled(t *testing.T) {
	sink := &recSink{}
	o, fake, _ := newTestOrchestrator(t, false, sink)
	report, dispatched, err := o.Dispatch(context.Background())
	if err != nil || !dispatched {
		t.Fatalf("dispatch dispatched=%v err=%v", dispatched, err)
	}
	if report.Status != "failed" {
		t.Fatalf("report = %+v", report)
	}
	if fake.CallCount() != 0 {
		t.Fatalf("runtime should not be called when disabled, calls=%d", fake.CallCount())
	}
}

func TestDispatchHonorsPermittedCapabilities(t *testing.T) {
	graph := planner.NewGraph("prj_1", "gph_1")
	_ = graph.Add(planner.WorkPackage{ID: "wp_a", Trade: "backend", Capabilities: []string{"write_workspace", "deploy"}})
	reg := registry.New()
	_ = reg.Add(registry.Worker{ID: "w1", Trade: "backend", Capabilities: []string{"write_workspace", "deploy"}, Available: true})
	fake := worker.NewFake()
	fake.Script("wp_a", worker.Result{Status: "succeeded", Tests: []worker.TestResult{{Name: "t", Outcome: "passed"}}})
	o, err := New(graph, Config{
		Registry: reg, Runtime: fake, Gates: []gate.Gate{gate.TestsGate{}},
		Enabled: true, Permitted: []string{"write_workspace"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls := fake.Calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %+v", calls)
	}
	for _, cap := range calls[0].Capabilities {
		if cap == "deploy" {
			t.Fatal("deploy must not be granted")
		}
	}
}

func TestNoEligibleWorker(t *testing.T) {
	graph := planner.NewGraph("prj_1", "gph_1")
	_ = graph.Add(planner.WorkPackage{ID: "wp_a", Trade: "mobile", Capabilities: []string{"write_workspace"}})
	reg := registry.New()
	_ = reg.Add(registry.Worker{ID: "w1", Trade: "backend", Capabilities: []string{"write_workspace"}, Available: true})
	o, err := New(graph, Config{Registry: reg, Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.Dispatch(context.Background()); err == nil {
		t.Fatal("expected no eligible worker error")
	}
}
