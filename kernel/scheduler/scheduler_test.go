package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	kcontext "github.com/greadee/aa/kernel/context"
	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/runtime/worker"
)

type sinkRecord struct {
	kind string
	id   string
	data []byte
}

type recSink struct {
	events  []string
	records []sinkRecord
}

func (s *recSink) Event(_ int, eventType string, _ map[string]any) error {
	s.events = append(s.events, eventType)
	return nil
}

func (s *recSink) Record(kind, id string, data []byte) error {
	s.records = append(s.records, sinkRecord{kind: kind, id: id, data: append([]byte(nil), data...)})
	return nil
}

func newTestScheduler(t *testing.T, enabled bool, sink Sink) (*Scheduler, *worker.Fake, *gate.HumanGate) {
	t.Helper()
	graph := planner.NewGraph("prj_1", "gph_1")
	if err := graph.Add(planner.WorkPackage{ID: "wp_a", Title: "a", Capabilities: []string{"write_workspace", "run_tests"}}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Add(planner.WorkPackage{ID: "wp_b", Title: "b", Capabilities: []string{"write_workspace", "run_tests"}, DependsOn: []string{"wp_a"}}); err != nil {
		t.Fatal(err)
	}

	reg := role_allocator.New()
	if err := reg.Add(worker.Worker{ID: "w1", Capabilities: []string{"write_workspace", "run_tests"}, Available: true}); err != nil {
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
	o, fake, _ := newTestScheduler(t, true, sink)

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
	if len(sink.records) == 0 {
		t.Fatal("expected sink assignment records")
	}
	last := sink.records[len(sink.records)-1]
	if last.kind != "assignment" {
		t.Fatalf("last record kind = %q", last.kind)
	}
	var record AssignmentRecord
	if err := json.Unmarshal(last.data, &record); err != nil {
		t.Fatal(err)
	}
	if record.State != string(StateAccepted) || record.ProjectID != "prj_1" {
		t.Fatalf("last record = %+v", record)
	}
}

func TestExecutionDisabled(t *testing.T) {
	sink := &recSink{}
	o, fake, _ := newTestScheduler(t, false, sink)
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
	_ = graph.Add(planner.WorkPackage{ID: "wp_a", Capabilities: []string{"write_workspace", "deploy"}})
	reg := role_allocator.New()
	_ = reg.Add(worker.Worker{ID: "w1", Capabilities: []string{"write_workspace", "deploy"}, Available: true})
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
	_ = graph.Add(planner.WorkPackage{ID: "wp_a", Capabilities: []string{"deploy"}})
	reg := role_allocator.New()
	_ = reg.Add(worker.Worker{ID: "w1", Capabilities: []string{"write_workspace"}, Available: true})
	o, err := New(graph, Config{Registry: reg, Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.Dispatch(context.Background()); err == nil {
		t.Fatal("expected no eligible worker error")
	}
}

func TestRejectMarksAssignmentFailed(t *testing.T) {
	sink := &recSink{}
	o, _, _ := newTestScheduler(t, true, sink)
	report, dispatched, err := o.Dispatch(context.Background())
	if err != nil || !dispatched || report.Status != "pending" {
		t.Fatalf("dispatch = %+v dispatched=%v err=%v", report, dispatched, err)
	}
	o.Reject("wp_a", "not acceptable")
	resolved, err := o.Resolve("wp_a")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Status != "failed" {
		t.Fatalf("resolved = %+v", resolved)
	}
	if report := o.Status(); len(report.Assignments) != 1 || report.Assignments[0].State != string(StateFailed) {
		t.Fatalf("status = %+v", report)
	}
}

type failingRecordSink struct {
	recSink
	fail bool
}

func (s *failingRecordSink) Record(kind, id string, data []byte) error {
	if s.fail {
		return errors.New("disk full")
	}
	return s.recSink.Record(kind, id, data)
}

func TestDispatchPropagatesAssignmentPersistenceFailure(t *testing.T) {
	sink := &failingRecordSink{fail: true}
	o, fake, _ := newTestScheduler(t, true, sink)

	_, dispatched, err := o.Dispatch(context.Background())
	if !dispatched || err == nil {
		t.Fatalf("dispatch dispatched=%v err=%v, want an observable persistence failure", dispatched, err)
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err = %v", err)
	}
	if fake.CallCount() != 0 {
		t.Fatalf("runtime called %d times after a persistence failure", fake.CallCount())
	}
	found := false
	for _, event := range sink.events {
		if event == "ASSIGNMENT_PERSIST_FAILED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("events = %v, want ASSIGNMENT_PERSIST_FAILED", sink.events)
	}
}

func TestResolvePropagatesAssignmentPersistenceFailure(t *testing.T) {
	sink := &failingRecordSink{}
	o, _, _ := newTestScheduler(t, true, sink)

	if _, _, err := o.Dispatch(context.Background()); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	o.Approve("wp_a")
	sink.fail = true
	if _, err := o.Resolve("wp_a"); err == nil {
		t.Fatal("expected resolve to report the persistence failure")
	}
}

func TestRestoreContinuesAttemptNumbering(t *testing.T) {
	sink := &recSink{}
	o, _, _ := newTestScheduler(t, true, sink)
	updated, err := o.Restore([]RestoreAssignment{{
		ID: "asg_wp_a_2", ProjectID: "prj_1", WorkPackageID: "wp_a",
		WorkerID: "w1", State: "failed", Attempt: 2,
	}}, time.Now().UTC())
	if err != nil || len(updated) != 0 {
		t.Fatalf("restore updated=%+v err=%v", updated, err)
	}
	if _, _, err := o.Dispatch(context.Background()); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	status := o.Status()
	if len(status.Assignments) != 1 {
		t.Fatalf("assignments = %+v", status.Assignments)
	}
	if status.Assignments[0].ID != "asg_wp_a_3" || status.Assignments[0].Attempt != 3 {
		t.Fatalf("assignment = %+v, want attempt 3", status.Assignments[0])
	}
}

func TestRestoreExpiresLeases(t *testing.T) {
	sink := &recSink{}
	o, _, _ := newTestScheduler(t, false, sink)
	now := time.Now().UTC()
	updated, err := o.Restore([]RestoreAssignment{{
		ID: "asg_wp_a_1", ProjectID: "prj_1", WorkPackageID: "wp_a",
		WorkerID: "w1", State: "leased", Attempt: 1,
		LeaseOwner: "w1", LeaseExpiresAt: now.Add(-time.Hour).Format(time.RFC3339Nano),
	}}, now)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if len(updated) != 1 || updated[0].State != "expired" {
		t.Fatalf("updated = %+v, want expired", updated)
	}
	status := o.Status()
	if len(status.Assignments) != 1 || status.Assignments[0].State != "expired" {
		t.Fatalf("assignments = %+v", status.Assignments)
	}
}

type stateSink struct {
	recSink
	states []string
}

func (s *stateSink) RecordWorkPackageState(_, _, state string) error {
	s.states = append(s.states, state)
	return nil
}

func TestWorkPackageStateTransitionsArePersistedThroughSink(t *testing.T) {
	sink := &stateSink{}
	o, _, _ := newTestScheduler(t, true, sink)
	if _, _, err := o.Dispatch(context.Background()); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	o.Approve("wp_a")
	if _, err := o.Resolve("wp_a"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	foundRunning, foundCompleted := false, false
	for _, state := range sink.states {
		if state == string(planner.StateRunning) {
			foundRunning = true
		}
		if state == string(planner.StateCompleted) {
			foundCompleted = true
		}
	}
	if !foundRunning || !foundCompleted {
		t.Fatalf("states = %v, want RUNNING and COMPLETED", sink.states)
	}
}
