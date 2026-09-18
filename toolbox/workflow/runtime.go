package workflow

import (
	"context"
	"errors"
	"fmt"

	toolbox "github.com/greadee/aa/toolbox"
)

// Runner executes one workflow step. Attempt is 1-based.
type Runner interface {
	Run(ctx context.Context, step toolbox.WorkflowStep, attempt int) (map[string]any, error)
}

// Compensator can undo a step that failed under the compensate policy. A Runner
// may optionally implement it.
type Compensator interface {
	Compensate(ctx context.Context, step toolbox.WorkflowStep, output map[string]any) error
}

// GateEvaluator decides whether a gate or human-approval step passes. Returning
// an error wrapping ErrApproval leaves the run awaiting approval.
type GateEvaluator interface {
	Evaluate(ctx context.Context, step toolbox.WorkflowStep) (bool, error)
}

// RunStatus is the lifecycle state of a run.
type RunStatus string

// Run statuses.
const (
	RunRunning          RunStatus = "running"
	RunComplete         RunStatus = "complete"
	RunFailed           RunStatus = "failed"
	RunAwaitingApproval RunStatus = "awaiting_approval"
)

// StepState is the lifecycle state of a step within a run.
type StepState string

// Step states.
const (
	StepPending      StepState = "pending"
	StepRunning      StepState = "running"
	StepComplete     StepState = "complete"
	StepSkipped      StepState = "skipped"
	StepCompensated  StepState = "compensated"
	StepFailed       StepState = "failed"
	StepAwaitingGate StepState = "awaiting_gate"
)

// StepResult records the outcome of one step.
type StepResult struct {
	ID       string         `json:"id"`
	State    StepState      `json:"state"`
	Attempts int            `json:"attempts"`
	Output   map[string]any `json:"output,omitempty"`
	Reason   string         `json:"reason,omitempty"`
}

// RunState is a serializable run state used to resume a workflow.
type RunState struct {
	WorkflowID string                `json:"workflowId"`
	Version    string                `json:"version"`
	Status     RunStatus             `json:"status"`
	Order      []string              `json:"order"`
	Results    map[string]StepResult `json:"results"`
}

// Result returns a step result.
func (s RunState) Result(id string) (StepResult, bool) {
	r, ok := s.Results[id]
	return r, ok
}

// Terminal reports whether a step needs no further work on resume.
func (r StepResult) Terminal() bool {
	switch r.State {
	case StepComplete, StepSkipped, StepCompensated:
		return true
	default:
		return false
	}
}

// Clone returns a deep copy of the run state.
func (s RunState) Clone() RunState {
	out := RunState{
		WorkflowID: s.WorkflowID,
		Version:    s.Version,
		Status:     s.Status,
		Order:      append([]string(nil), s.Order...),
		Results:    make(map[string]StepResult, len(s.Results)),
	}
	for id, r := range s.Results {
		cp := r
		if r.Output != nil {
			cp.Output = make(map[string]any, len(r.Output))
			for k, v := range r.Output {
				cp.Output[k] = v
			}
		}
		out.Results[id] = cp
	}
	return out
}

// Runtime executes a compiled plan over a Runner seam.
type Runtime struct {
	runner Runner
	gates  GateEvaluator
}

// New returns a runtime. A nil gate evaluator leaves gate and approval steps
// awaiting approval, which fails closed.
func New(runner Runner, gates GateEvaluator) *Runtime {
	return &Runtime{runner: runner, gates: gates}
}

// Run executes a plan, resuming from prior state when supplied. It returns the
// resulting state even on error so the caller can persist and resume it.
func (rt *Runtime) Run(ctx context.Context, plan Plan, prior *RunState) (RunState, error) {
	state := newRunState(plan)
	if prior != nil {
		state = prior.Clone()
		if state.Results == nil {
			state.Results = map[string]StepResult{}
		}
		if state.WorkflowID == "" {
			state.WorkflowID = plan.WorkflowID
		}
		if state.Version == "" {
			state.Version = plan.Version
		}
	}
	state.Status = RunRunning

	for _, id := range plan.Order {
		step, ok := plan.Steps[id]
		if !ok {
			continue
		}
		if res, ok := state.Results[id]; ok && res.Terminal() {
			continue
		}
		if !depsSatisfied(plan, state, id) {
			state.Results[id] = StepResult{ID: id, State: StepAwaitingGate, Reason: "dependencies not satisfied"}
			state.Status = RunFailed
			return state, fmt.Errorf("%w: step %q dependencies not satisfied", toolbox.ErrFailed, id)
		}
		if step.Kind == "gate" || step.Kind == "human_approval" {
			passed, err := rt.evaluateGate(ctx, step)
			if err != nil {
				state.Results[id] = StepResult{ID: id, State: StepAwaitingGate, Reason: err.Error()}
				state.Status = RunAwaitingApproval
				return state, err
			}
			if !passed {
				state.Results[id] = StepResult{ID: id, State: StepAwaitingGate, Reason: "gate not passed"}
				state.Status = RunAwaitingApproval
				return state, fmt.Errorf("%w: step %q", toolbox.ErrApproval, id)
			}
			state.Results[id] = StepResult{ID: id, State: StepComplete}
			continue
		}
		if err := rt.runWithPolicy(ctx, &state, step); err != nil {
			return state, err
		}
	}
	state.Status = RunComplete
	return state, nil
}

func (rt *Runtime) evaluateGate(ctx context.Context, step toolbox.WorkflowStep) (bool, error) {
	if rt.gates == nil {
		return false, fmt.Errorf("%w: step %q has no gate evaluator", toolbox.ErrApproval, step.ID)
	}
	return rt.gates.Evaluate(ctx, step)
}

func (rt *Runtime) runWithPolicy(ctx context.Context, state *RunState, step toolbox.WorkflowStep) error {
	if rt.runner == nil {
		state.Results[step.ID] = StepResult{ID: step.ID, State: StepFailed, Reason: "no runner configured"}
		state.Status = RunFailed
		return fmt.Errorf("%w: step %q has no runner", toolbox.ErrFailed, step.ID)
	}
	prior := state.Results[step.ID]
	max := maxAttempts(step)
	attempt := prior.Attempts
	var lastErr error
	for attempt < max {
		attempt++
		output, err := rt.runner.Run(ctx, step, attempt)
		if err == nil {
			state.Results[step.ID] = StepResult{ID: step.ID, State: StepComplete, Attempts: attempt, Output: output}
			return nil
		}
		lastErr = err
		state.Results[step.ID] = StepResult{ID: step.ID, State: StepRunning, Attempts: attempt, Reason: err.Error()}
	}

	switch onFailure(step) {
	case "skip":
		state.Results[step.ID] = StepResult{ID: step.ID, State: StepSkipped, Attempts: attempt, Reason: lastErr.Error()}
		return nil
	case "compensate":
		if comp, ok := rt.runner.(Compensator); ok {
			if cerr := comp.Compensate(ctx, step, prior.Output); cerr != nil {
				state.Results[step.ID] = StepResult{ID: step.ID, State: StepFailed, Attempts: attempt, Reason: cerr.Error()}
				state.Status = RunFailed
				return fmt.Errorf("%w: compensate step %q: %v", toolbox.ErrFailed, step.ID, cerr)
			}
			state.Results[step.ID] = StepResult{ID: step.ID, State: StepCompensated, Attempts: attempt, Reason: lastErr.Error()}
			return nil
		}
	}
	state.Results[step.ID] = StepResult{ID: step.ID, State: StepFailed, Attempts: attempt, Reason: lastErr.Error()}
	state.Status = RunFailed
	return fmt.Errorf("%w: step %q: %v", toolbox.ErrFailed, step.ID, lastErr)
}

func newRunState(plan Plan) RunState {
	return RunState{
		WorkflowID: plan.WorkflowID,
		Version:    plan.Version,
		Status:     RunRunning,
		Order:      append([]string(nil), plan.Order...),
		Results:    map[string]StepResult{},
	}
}

func depsSatisfied(plan Plan, state RunState, id string) bool {
	for _, dep := range plan.DependsOn[id] {
		res, ok := state.Results[dep]
		if !ok || !res.Terminal() {
			return false
		}
	}
	return true
}

func maxAttempts(step toolbox.WorkflowStep) int {
	retries := 0
	if step.Retries != nil {
		retries = *step.Retries
	}
	if retries < 0 {
		retries = 0
	}
	return retries + 1
}

func onFailure(step toolbox.WorkflowStep) string {
	if step.OnFailure == "" {
		return "fail"
	}
	return step.OnFailure
}

// IsApproval reports whether an error means a run is awaiting approval.
func IsApproval(err error) bool {
	return errors.Is(err, toolbox.ErrApproval)
}
