package workflow

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
)

type fakeRunner struct {
	mu       sync.Mutex
	calls    []string
	outcomes map[string][]error
	comp     []string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outcomes: map[string][]error{}}
}

func (r *fakeRunner) fail(step string, errs ...error) {
	r.outcomes[step] = errs
}

func (r *fakeRunner) Run(_ context.Context, step toolbox.WorkflowStep, attempt int) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, fmt.Sprintf("%s:%d", step.ID, attempt))
	if seq := r.outcomes[step.ID]; attempt-1 < len(seq) && seq[attempt-1] != nil {
		return nil, seq[attempt-1]
	}
	return map[string]any{"step": step.ID, "attempt": attempt}, nil
}

func (r *fakeRunner) Compensate(_ context.Context, step toolbox.WorkflowStep, _ map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comp = append(r.comp, step.ID)
	return nil
}

func (r *fakeRunner) callsList() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

type mapGates map[string]bool

func (g mapGates) Evaluate(_ context.Context, step toolbox.WorkflowStep) (bool, error) {
	if pass, ok := g[step.ID]; ok {
		if pass {
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("%w: %s", toolbox.ErrApproval, step.ID)
}

func withRetries(step toolbox.WorkflowStep, n int) toolbox.WorkflowStep {
	step.Retries = &n
	return step
}

func TestRunCompletesInOrder(t *testing.T) {
	runner := newFakeRunner()
	plan, err := Compile(wf("wf", step("a", "task"), step("b", "task", "a")))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if state.Status != RunComplete {
		t.Fatalf("status = %q", state.Status)
	}
	if !reflect.DeepEqual(runner.callsList(), []string{"a:1", "b:1"}) {
		t.Fatalf("calls = %v", runner.callsList())
	}
	if r, _ := state.Result("b"); r.Output["attempt"] != 1 {
		t.Fatalf("output = %v", r.Output)
	}
}

func TestRetryThenSuccess(t *testing.T) {
	runner := newFakeRunner()
	boom := errors.New("boom")
	runner.fail("a", boom, nil)
	m := withRetries(step("a", "task"), 2)
	m.OnFailure = "retry"
	plan, _ := Compile(wf("wf", m))
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, _ := state.Result("a")
	if !r.Terminal() || r.Attempts != 2 {
		t.Fatalf("result = %+v", r)
	}
	if !reflect.DeepEqual(runner.callsList(), []string{"a:1", "a:2"}) {
		t.Fatalf("calls = %v", runner.callsList())
	}
}

func TestRetriesExhaustedFails(t *testing.T) {
	runner := newFakeRunner()
	boom := errors.New("boom")
	runner.fail("a", boom, boom, boom)
	m := withRetries(step("a", "task"), 2)
	m.OnFailure = "fail"
	plan, _ := Compile(wf("wf", m))
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if !errors.Is(err, toolbox.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if state.Status != RunFailed {
		t.Fatalf("status = %q", state.Status)
	}
	r, _ := state.Result("a")
	if r.Attempts != 3 || r.State != StepFailed {
		t.Fatalf("result = %+v", r)
	}
}

func TestOnFailureSkip(t *testing.T) {
	runner := newFakeRunner()
	boom := errors.New("boom")
	runner.fail("a", boom)
	m := step("a", "task")
	m.OnFailure = "skip"
	plan, _ := Compile(wf("wf", m, step("b", "task", "a")))
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r, _ := state.Result("a"); r.State != StepSkipped {
		t.Fatalf("a = %+v", r)
	}
	if r, _ := state.Result("b"); r.State != StepComplete {
		t.Fatalf("b = %+v", r)
	}
}

func TestOnFailureCompensate(t *testing.T) {
	runner := newFakeRunner()
	boom := errors.New("boom")
	runner.fail("a", boom)
	m := step("a", "task")
	m.OnFailure = "compensate"
	plan, _ := Compile(wf("wf", m))
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r, _ := state.Result("a"); r.State != StepCompensated {
		t.Fatalf("a = %+v", r)
	}
	if !reflect.DeepEqual(runner.comp, []string{"a"}) {
		t.Fatalf("comp calls = %v", runner.comp)
	}
}

func TestOnFailureCompensateWithoutCompensatorFailsClosed(t *testing.T) {
	runner := &plainRunner{}
	m := step("a", "task")
	m.OnFailure = "compensate"
	plan, _ := Compile(wf("wf", m))
	_, err := New(runner, nil).Run(context.Background(), plan, nil)
	if !errors.Is(err, toolbox.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
}

type plainRunner struct{}

func (plainRunner) Run(context.Context, toolbox.WorkflowStep, int) (map[string]any, error) {
	return nil, errors.New("boom")
}

func TestGatePasses(t *testing.T) {
	runner := newFakeRunner()
	plan, _ := Compile(wf("wf",
		step("a", "task"),
		toolbox.WorkflowStep{ID: "g", Kind: "gate", DependsOn: []string{"a"}, Gates: []string{"tests"}},
	))
	state, err := New(runner, mapGates{"g": true}).Run(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r, _ := state.Result("g"); r.State != StepComplete {
		t.Fatalf("g = %+v", r)
	}
}

func TestGateNotPassedAwaitsApproval(t *testing.T) {
	runner := newFakeRunner()
	plan, _ := Compile(wf("wf", toolbox.WorkflowStep{ID: "g", Kind: "human_approval"}))
	state, err := New(runner, mapGates{"g": false}).Run(context.Background(), plan, nil)
	if !IsApproval(err) {
		t.Fatalf("err = %v, want approval", err)
	}
	if state.Status != RunAwaitingApproval {
		t.Fatalf("status = %q", state.Status)
	}
}

func TestApprovalWithoutEvaluatorAwaits(t *testing.T) {
	runner := newFakeRunner()
	plan, _ := Compile(wf("wf", toolbox.WorkflowStep{ID: "g", Kind: "human_approval"}))
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if !IsApproval(err) {
		t.Fatalf("err = %v, want approval", err)
	}
	if state.Status != RunAwaitingApproval {
		t.Fatalf("status = %q", state.Status)
	}
}

func TestResumeAfterApproval(t *testing.T) {
	runner := newFakeRunner()
	plan, _ := Compile(wf("wf",
		step("a", "task"),
		toolbox.WorkflowStep{ID: "g", Kind: "human_approval", DependsOn: []string{"a"}},
		step("c", "task", "g"),
	))
	// First run: g awaits approval.
	state, err := New(runner, mapGates{"g": false}).Run(context.Background(), plan, nil)
	if !IsApproval(err) {
		t.Fatalf("first err = %v", err)
	}
	if !reflect.DeepEqual(runner.callsList(), []string{"a:1"}) {
		t.Fatalf("first calls = %v", runner.callsList())
	}
	// Resume: g passes; a must not re-run.
	state2, err := New(runner, mapGates{"g": true}).Run(context.Background(), plan, &state)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if state2.Status != RunComplete {
		t.Fatalf("status = %q", state2.Status)
	}
	if !reflect.DeepEqual(runner.callsList(), []string{"a:1", "c:1"}) {
		t.Fatalf("resume calls = %v", runner.callsList())
	}
}

func TestResumePreservesAttempts(t *testing.T) {
	runner := newFakeRunner()
	boom := errors.New("boom")
	runner.fail("a", boom)
	m := step("a", "task")
	m.OnFailure = "fail"
	plan, _ := Compile(wf("wf", m))
	state, err := New(runner, nil).Run(context.Background(), plan, nil)
	if !errors.Is(err, toolbox.ErrFailed) {
		t.Fatalf("first err = %v", err)
	}
	if r, _ := state.Result("a"); r.Attempts != 1 {
		t.Fatalf("attempts = %d", r.Attempts)
	}
	// Make the step retryable and resume; attempts accumulate.
	m2 := withRetries(m, 3)
	plan2, _ := Compile(wf("wf", m2))
	runner.outcomes["a"] = []error{boom, nil}
	state2, err := New(runner, nil).Run(context.Background(), plan2, &state)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if r, _ := state2.Result("a"); r.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", r.Attempts)
	}
}

func TestCloneIsDeep(t *testing.T) {
	s := RunState{Results: map[string]StepResult{"a": {ID: "a", State: StepComplete, Output: map[string]any{"x": 1}}}}
	c := s.Clone()
	c.Results["a"].Output["x"] = 2
	if s.Results["a"].Output["x"] != 1 {
		t.Fatal("Clone aliased output")
	}
}
