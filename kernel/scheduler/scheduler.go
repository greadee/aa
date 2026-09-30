// Package scheduler runs aa-kernel's deterministic supervised control cycle:
// ready -> select worker -> build contract -> compile context -> lease -> run
// -> intake -> gates -> accept -> telemetry. It owns ordering, dispatch
// readiness, concurrency policy, the assignment state machine, and leases. No
// model is called in the control path; execution goes through an adapter and is
// disabled unless enabled.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	kcontext "github.com/greadee/aa/kernel/context"
	"github.com/greadee/aa/kernel/contract"
	"github.com/greadee/aa/kernel/gate"
	"github.com/greadee/aa/kernel/intake"
	"github.com/greadee/aa/kernel/telemetry"
	"github.com/greadee/aa/runtime/worker"
)

// Sink records events and records for the outside world (memory or RPC).
type Sink interface {
	Event(sequence int, eventType string, payload map[string]any) error
	Record(kind, id string, data []byte) error
}

// NopSink discards everything.
type NopSink struct{}

// Event implements Sink.
func (NopSink) Event(int, string, map[string]any) error { return nil }

// Record implements Sink.
func (NopSink) Record(string, string, []byte) error { return nil }

// Config configures the scheduler.
type Config struct {
	Registry      *role_allocator.Registry
	Runtime       worker.Adapter
	Gates         []gate.Gate
	Compiler      kcontext.Compiler
	Sink          Sink
	Permitted     []string
	Enabled       bool
	LeaseTTL      time.Duration
	ContextInputs func(workPackageID string) []kcontext.Input
	Now           func() time.Time
	// MaxConcurrency bounds how many ready work packages run at once. Zero or
	// one means serial (deterministic order).
	MaxConcurrency int
	// MaxAttempts is the total attempts per work package (including the first).
	// Zero or one means no retry.
	MaxAttempts int
}

// Scheduler runs the control cycle over one project graph.
type Scheduler struct {
	projectID string
	graph     *planner.Graph
	cfg       Config

	mu                 sync.Mutex
	intake             *intake.Service
	assignments        map[string]*Assignment
	attempts           map[string]int
	results            map[string]worker.Result
	telemetryRecords   []telemetry.Record
	learningCandidates []telemetry.Candidate
	sequence           int
}

// New validates the config and graph and returns a scheduler.
func New(graph *planner.Graph, cfg Config) (*Scheduler, error) {
	if graph == nil {
		return nil, fmt.Errorf("scheduler: graph is required")
	}
	if cfg.Registry == nil {
		return nil, fmt.Errorf("scheduler: registry is required")
	}
	if err := graph.Validate(); err != nil {
		return nil, err
	}
	if cfg.Sink == nil {
		cfg.Sink = NopSink{}
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.LeaseTTL <= 0 {
		cfg.LeaseTTL = time.Minute
	}
	if cfg.Enabled && cfg.Runtime == nil {
		return nil, fmt.Errorf("scheduler: runtime is required when execution is enabled")
	}
	if !cfg.Enabled {
		cfg.Runtime = worker.Disabled{}
	}
	return &Scheduler{
		projectID:   graph.ProjectID,
		graph:       graph,
		cfg:         cfg,
		intake:      intake.New(),
		assignments: make(map[string]*Assignment),
		attempts:    make(map[string]int),
		results:     make(map[string]worker.Result),
	}, nil
}

// DispatchReport describes the outcome of dispatching or resolving one unit.
type DispatchReport struct {
	WorkPackageID  string      `json:"workPackageId"`
	WorkerID       string      `json:"workerId,omitempty"`
	State          string      `json:"state"`
	Status         string      `json:"status"` // succeeded, pending, failed
	Gates          gate.Report `json:"gates"`
	IntakeAccepted bool        `json:"intakeAccepted"`
}

// Dispatch runs the next ready work package if one exists. The boolean reports
// whether work was dispatched.
func (o *Scheduler) Dispatch(ctx context.Context) (DispatchReport, bool, error) {
	o.mu.Lock()
	ready := o.graph.Ready()
	if len(ready) == 0 {
		o.mu.Unlock()
		return DispatchReport{}, false, nil
	}
	workPackageID := ready[0]
	o.mu.Unlock()

	report, err := o.dispatchOne(ctx, workPackageID)
	return report, true, err
}

// dispatchOne runs the full control cycle for one work package. Shared state is
// mutated only while holding o.mu; the runtime call runs outside the lock so a
// caller (Run) can dispatch several work packages concurrently.
func (o *Scheduler) dispatchOne(ctx context.Context, workPackageID string) (DispatchReport, error) {
	o.mu.Lock()
	wp, ok := o.graph.Get(workPackageID)
	if !ok {
		o.mu.Unlock()
		return DispatchReport{}, fmt.Errorf("scheduler: unknown work package %s", workPackageID)
	}
	selected, ok := o.cfg.Registry.SelectOne(role_allocator.Requirement{Capabilities: wp.Capabilities})
	if !ok {
		o.mu.Unlock()
		return DispatchReport{}, fmt.Errorf("scheduler: no eligible worker for %s", workPackageID)
	}

	o.attempts[workPackageID]++
	attempt := o.attempts[workPackageID]
	assignmentID := fmt.Sprintf("asg_%s_%d", workPackageID, attempt)
	executionContract, err := contract.Build(contract.Request{
		ID:            "ec_" + assignmentID,
		ProjectID:     o.projectID,
		WorkPackageID: workPackageID,
		AssignmentID:  assignmentID,
		Requested:     wp.Capabilities,
		Permitted:     o.cfg.Permitted,
		Runtime:       &contract.RuntimeSpec{Kind: "runtime"},
		Gates:         gateNames(o.cfg.Gates),
	})
	if err != nil {
		o.mu.Unlock()
		return DispatchReport{}, err
	}

	var inputs []kcontext.Input
	if o.cfg.ContextInputs != nil {
		inputs = o.cfg.ContextInputs(workPackageID)
	}
	bundle := o.cfg.Compiler.Compile(o.projectID, workPackageID, inputs)

	assignment := NewAssignment(assignmentID, o.projectID, workPackageID, string(selected.ID))
	assignment.Attempt = attempt
	for _, state := range []State{StateLeased, StatePreparing, StateRunning} {
		if err := assignment.Transition(state); err != nil {
			o.mu.Unlock()
			return DispatchReport{}, err
		}
	}
	assignment.LeaseFor(string(selected.ID), o.cfg.Now(), o.cfg.LeaseTTL)
	_ = o.graph.SetState(workPackageID, planner.StateRunning)
	o.emit("EXECUTION_STARTED", map[string]any{
		"workPackageId": workPackageID, "assignmentId": assignmentID, "workerId": string(selected.ID),
	})
	o.mu.Unlock()

	result, err := o.cfg.Runtime.Run(ctx, worker.Request{
		AssignmentID:   assignmentID,
		WorkPackageID:  workPackageID,
		ContractID:     executionContract.ID,
		ContractDigest: executionContract.Digest,
		ContextDigest:  bundle.Digest,
		Capabilities:   executionContract.Capabilities,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			o.mu.Lock()
			_ = assignment.Transition(StateCanceled)
			o.assignments[workPackageID] = assignment
			o.mu.Unlock()
			return DispatchReport{}, err
		}
		o.mu.Lock()
		_ = assignment.Transition(StateFailed)
		_ = o.graph.SetState(workPackageID, planner.StateDeficient)
		o.assignments[workPackageID] = assignment
		o.mu.Unlock()
		return DispatchReport{WorkPackageID: workPackageID, WorkerID: string(selected.ID), State: string(assignment.State), Status: "failed"}, nil
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if err := assignment.Transition(StateCollecting); err != nil {
		return DispatchReport{}, err
	}
	accepted, err := o.intake.Submit(intake.Result{
		ID:            "res_" + assignmentID,
		AttemptID:     "att_" + assignmentID,
		AssignmentID:  assignmentID,
		WorkPackageID: workPackageID,
		Status:        result.Status,
		Summary:       result.Summary,
		Tests:         result.Tests,
		Artifacts:     result.Artifacts,
		Failure:       result.Failure,
	})
	if err != nil {
		return DispatchReport{}, err
	}

	o.assignments[workPackageID] = assignment
	o.results[workPackageID] = result
	o.recordTelemetry(assignment, result)

	report := gate.Evaluate(o.cfg.Gates, o.subject(workPackageID, result))

	if result.Status != "succeeded" && result.Status != "partial" {
		_ = assignment.Transition(StateFailed)
		_ = o.graph.SetState(workPackageID, planner.StateDeficient)
		return DispatchReport{WorkPackageID: workPackageID, WorkerID: string(selected.ID), State: string(assignment.State), Status: "failed", Gates: report, IntakeAccepted: accepted}, nil
	}

	if err := assignment.Transition(StateAwaitingGates); err != nil {
		return DispatchReport{}, err
	}
	if report.Passed {
		return o.accept(assignment, report, accepted), nil
	}
	status := "pending"
	if !report.Pending {
		_ = assignment.Transition(StateFailed)
		_ = o.graph.SetState(workPackageID, planner.StateDeficient)
		status = "failed"
	}
	return DispatchReport{WorkPackageID: workPackageID, WorkerID: string(selected.ID), State: string(assignment.State), Status: status, Gates: report, IntakeAccepted: accepted}, nil
}

// Resolve re-evaluates gates for a work package that is awaiting gates.
func (o *Scheduler) Resolve(workPackageID string) (DispatchReport, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	assignment, ok := o.assignments[workPackageID]
	if !ok {
		return DispatchReport{}, fmt.Errorf("scheduler: unknown assignment for %s", workPackageID)
	}
	if assignment.State != StateAwaitingGates {
		return DispatchReport{}, fmt.Errorf("scheduler: %s is not awaiting gates", workPackageID)
	}
	result := o.results[workPackageID]
	report := gate.Evaluate(o.cfg.Gates, o.subject(workPackageID, result))
	if report.Passed {
		return o.accept(assignment, report, true), nil
	}
	if report.Pending {
		return DispatchReport{WorkPackageID: workPackageID, WorkerID: assignment.WorkerID, State: string(assignment.State), Status: "pending", Gates: report}, nil
	}
	_ = assignment.Transition(StateFailed)
	_ = o.graph.SetState(workPackageID, planner.StateDeficient)
	return DispatchReport{WorkPackageID: workPackageID, WorkerID: assignment.WorkerID, State: string(assignment.State), Status: "failed", Gates: report}, nil
}

// Approve records human approval for a work package in every approver gate.
func (o *Scheduler) Approve(workPackageID string) {
	for _, g := range o.cfg.Gates {
		if approver, ok := g.(interface{ Approve(string) }); ok {
			approver.Approve(workPackageID)
		}
	}
}

// Telemetry returns recorded telemetry in order.
func (o *Scheduler) Telemetry() []telemetry.Record {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]telemetry.Record, len(o.telemetryRecords))
	copy(out, o.telemetryRecords)
	return out
}

// Candidates returns derived learning candidates in order.
func (o *Scheduler) Candidates() []telemetry.Candidate {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]telemetry.Candidate, len(o.learningCandidates))
	copy(out, o.learningCandidates)
	return out
}

// AssignmentStatus is a read view of an assignment.
type AssignmentStatus struct {
	ID            string `json:"id"`
	WorkPackageID string `json:"workPackageId"`
	WorkerID      string `json:"workerId"`
	State         string `json:"state"`
	Attempt       int    `json:"attempt"`
}

// ProjectStatus is a read view of the project.
type ProjectStatus struct {
	ProjectID    string                `json:"projectId"`
	GraphID      string                `json:"graphId"`
	WorkPackages []planner.WorkPackage `json:"workPackages"`
	Assignments  []AssignmentStatus    `json:"assignments"`
}

// Status returns the current project status.
func (o *Scheduler) Status() ProjectStatus {
	o.mu.Lock()
	defer o.mu.Unlock()
	status := ProjectStatus{ProjectID: o.graph.ProjectID, GraphID: o.graph.ID, WorkPackages: o.graph.List()}
	for _, id := range sortedKeys(o.assignments) {
		a := o.assignments[id]
		status.Assignments = append(status.Assignments, AssignmentStatus{
			ID: a.ID, WorkPackageID: a.WorkPackageID, WorkerID: a.WorkerID, State: string(a.State), Attempt: a.Attempt,
		})
	}
	return status
}

func (o *Scheduler) accept(assignment *Assignment, report gate.Report, accepted bool) DispatchReport {
	_ = assignment.Transition(StateAccepted)
	_ = o.graph.SetState(assignment.WorkPackageID, planner.StateCompleted)
	o.emit("WORK_ACCEPTED", map[string]any{
		"workPackageId": assignment.WorkPackageID, "assignmentId": assignment.ID,
	})
	return DispatchReport{
		WorkPackageID:  assignment.WorkPackageID,
		WorkerID:       assignment.WorkerID,
		State:          string(assignment.State),
		Status:         "succeeded",
		Gates:          report,
		IntakeAccepted: accepted,
	}
}

func (o *Scheduler) subject(workPackageID string, result worker.Result) gate.Subject {
	var tests []gate.TestSummary
	for _, t := range result.Tests {
		tests = append(tests, gate.TestSummary{Name: t.Name, Outcome: t.Outcome})
	}
	return gate.Subject{WorkPackageID: workPackageID, Status: result.Status, Tests: tests}
}

// recordTelemetry appends execution evidence. Callers must hold o.mu.
func (o *Scheduler) recordTelemetry(assignment *Assignment, result worker.Result) {
	record := telemetry.Record{
		AttemptID:     "att_" + assignment.ID,
		AssignmentID:  assignment.ID,
		WorkPackageID: assignment.WorkPackageID,
		Outcome:       result.Status,
	}
	o.telemetryRecords = append(o.telemetryRecords, record)
	var failed []string
	for _, t := range result.Tests {
		if t.Outcome != "passed" {
			failed = append(failed, t.Name)
		}
	}
	o.learningCandidates = append(o.learningCandidates, telemetry.Derive(record, failed)...)
}

// emit records one ordered event. Callers must hold o.mu.
func (o *Scheduler) emit(eventType string, payload map[string]any) {
	o.sequence++
	_ = o.cfg.Sink.Event(o.sequence, eventType, payload)
}

func gateNames(gates []gate.Gate) []string {
	names := make([]string, 0, len(gates))
	for _, g := range gates {
		names = append(names, g.Name())
	}
	sort.Strings(names)
	return names
}

func sortedKeys(m map[string]*Assignment) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
