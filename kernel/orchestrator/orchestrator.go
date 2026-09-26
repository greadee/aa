// Package orchestrator runs aa-kernel's deterministic supervised control cycle:
// ready -> select worker -> build contract -> compile context -> lease -> run
// -> intake -> gates -> accept -> telemetry. No model is called in the control
// path; execution goes through an adapter and is disabled unless enabled.
package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/greadee/aa/kernel/allocator/planner"
	"github.com/greadee/aa/kernel/allocator/role_allocator"
	kcontext "github.com/greadee/aa/kernel/context"
	"github.com/greadee/aa/kernel/contract"
	"github.com/greadee/aa/kernel/control"
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

// Config configures the orchestrator.
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
}

// Orchestrator runs the control cycle over one project graph.
type Orchestrator struct {
	graph              *planner.Graph
	cfg                Config
	intake             *intake.Service
	assignments        map[string]*control.Assignment
	results            map[string]worker.Result
	telemetryRecords   []telemetry.Record
	learningCandidates []telemetry.Candidate
	sequence           int
}

// New validates the config and graph and returns an orchestrator.
func New(graph *planner.Graph, cfg Config) (*Orchestrator, error) {
	if graph == nil {
		return nil, fmt.Errorf("orchestrator: graph is required")
	}
	if cfg.Registry == nil {
		return nil, fmt.Errorf("orchestrator: registry is required")
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
		return nil, fmt.Errorf("orchestrator: runtime is required when execution is enabled")
	}
	if !cfg.Enabled {
		cfg.Runtime = worker.Disabled{}
	}
	return &Orchestrator{
		graph:       graph,
		cfg:         cfg,
		intake:      intake.New(),
		assignments: make(map[string]*control.Assignment),
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
func (o *Orchestrator) Dispatch(ctx context.Context) (DispatchReport, bool, error) {
	ready := o.graph.Ready()
	if len(ready) == 0 {
		return DispatchReport{}, false, nil
	}
	workPackageID := ready[0]
	wp, _ := o.graph.Get(workPackageID)

	selected, ok := o.cfg.Registry.SelectOne(role_allocator.Requirement{
		Trade:        worker.Trade(wp.Trade),
		Capabilities: wp.Capabilities,
	})
	if !ok {
		return DispatchReport{}, false, fmt.Errorf("orchestrator: no eligible worker for %s", workPackageID)
	}

	assignmentID := "asg_" + workPackageID
	executionContract, err := contract.Build(contract.Request{
		ID:            "ec_" + assignmentID,
		ProjectID:     o.graph.ProjectID,
		WorkPackageID: workPackageID,
		AssignmentID:  assignmentID,
		Requested:     wp.Capabilities,
		Permitted:     o.cfg.Permitted,
		Runtime:       &contract.RuntimeSpec{Kind: "runtime"},
		Gates:         gateNames(o.cfg.Gates),
	})
	if err != nil {
		return DispatchReport{}, false, err
	}

	var inputs []kcontext.Input
	if o.cfg.ContextInputs != nil {
		inputs = o.cfg.ContextInputs(workPackageID)
	}
	bundle := o.cfg.Compiler.Compile(o.graph.ProjectID, workPackageID, inputs)

	assignment := control.New(assignmentID, o.graph.ProjectID, workPackageID, string(selected.ID))
	for _, state := range []control.State{control.StateLeased, control.StatePreparing, control.StateRunning} {
		if err := assignment.Transition(state); err != nil {
			return DispatchReport{}, false, err
		}
	}
	assignment.LeaseFor(string(selected.ID), o.cfg.Now(), o.cfg.LeaseTTL)
	_ = o.graph.SetState(workPackageID, planner.StateRunning)
	o.emit("EXECUTION_STARTED", map[string]any{
		"workPackageId": workPackageID, "assignmentId": assignmentID, "workerId": string(selected.ID),
	})

	result, err := o.cfg.Runtime.Run(ctx, worker.Request{
		AssignmentID:   assignmentID,
		WorkPackageID:  workPackageID,
		ContractID:     executionContract.ID,
		ContractDigest: executionContract.Digest,
		ContextDigest:  bundle.Digest,
		Capabilities:   executionContract.Capabilities,
	})
	if err != nil {
		_ = assignment.Transition(control.StateFailed)
		_ = o.graph.SetState(workPackageID, planner.StateDeficient)
		o.assignments[workPackageID] = assignment
		return DispatchReport{WorkPackageID: workPackageID, WorkerID: string(selected.ID), State: string(assignment.State), Status: "failed"}, true, nil
	}
	if err := assignment.Transition(control.StateCollecting); err != nil {
		return DispatchReport{}, false, err
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
		return DispatchReport{}, false, err
	}

	o.assignments[workPackageID] = assignment
	o.results[workPackageID] = result
	o.recordTelemetry(assignment, result)

	report := gate.Evaluate(o.cfg.Gates, o.subject(workPackageID, result))

	if result.Status != "succeeded" && result.Status != "partial" {
		_ = assignment.Transition(control.StateFailed)
		_ = o.graph.SetState(workPackageID, planner.StateDeficient)
		return DispatchReport{WorkPackageID: workPackageID, WorkerID: string(selected.ID), State: string(assignment.State), Status: "failed", Gates: report, IntakeAccepted: accepted}, true, nil
	}

	if err := assignment.Transition(control.StateAwaitingGates); err != nil {
		return DispatchReport{}, false, err
	}
	if report.Passed {
		return o.accept(assignment, report, accepted), true, nil
	}
	status := "pending"
	if !report.Pending {
		_ = assignment.Transition(control.StateFailed)
		_ = o.graph.SetState(workPackageID, planner.StateDeficient)
		status = "failed"
	}
	return DispatchReport{WorkPackageID: workPackageID, WorkerID: string(selected.ID), State: string(assignment.State), Status: status, Gates: report, IntakeAccepted: accepted}, true, nil
}

// Resolve re-evaluates gates for a work package that is awaiting gates.
func (o *Orchestrator) Resolve(workPackageID string) (DispatchReport, error) {
	assignment, ok := o.assignments[workPackageID]
	if !ok {
		return DispatchReport{}, fmt.Errorf("orchestrator: unknown assignment for %s", workPackageID)
	}
	if assignment.State != control.StateAwaitingGates {
		return DispatchReport{}, fmt.Errorf("orchestrator: %s is not awaiting gates", workPackageID)
	}
	result := o.results[workPackageID]
	report := gate.Evaluate(o.cfg.Gates, o.subject(workPackageID, result))
	if report.Passed {
		return o.accept(assignment, report, true), nil
	}
	if report.Pending {
		return DispatchReport{WorkPackageID: workPackageID, WorkerID: assignment.WorkerID, State: string(assignment.State), Status: "pending", Gates: report}, nil
	}
	_ = assignment.Transition(control.StateFailed)
	_ = o.graph.SetState(workPackageID, planner.StateDeficient)
	return DispatchReport{WorkPackageID: workPackageID, WorkerID: assignment.WorkerID, State: string(assignment.State), Status: "failed", Gates: report}, nil
}

// Approve records human approval for a work package in every approver gate.
func (o *Orchestrator) Approve(workPackageID string) {
	for _, g := range o.cfg.Gates {
		if approver, ok := g.(interface{ Approve(string) }); ok {
			approver.Approve(workPackageID)
		}
	}
}

// Telemetry returns recorded telemetry in order.
func (o *Orchestrator) Telemetry() []telemetry.Record {
	out := make([]telemetry.Record, len(o.telemetryRecords))
	copy(out, o.telemetryRecords)
	return out
}

// Candidates returns derived learning candidates in order.
func (o *Orchestrator) Candidates() []telemetry.Candidate {
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
func (o *Orchestrator) Status() ProjectStatus {
	status := ProjectStatus{ProjectID: o.graph.ProjectID, GraphID: o.graph.ID, WorkPackages: o.graph.List()}
	for _, id := range sortedKeys(o.assignments) {
		a := o.assignments[id]
		status.Assignments = append(status.Assignments, AssignmentStatus{
			ID: a.ID, WorkPackageID: a.WorkPackageID, WorkerID: a.WorkerID, State: string(a.State), Attempt: a.Attempt,
		})
	}
	return status
}

func (o *Orchestrator) accept(assignment *control.Assignment, report gate.Report, accepted bool) DispatchReport {
	_ = assignment.Transition(control.StateAccepted)
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

func (o *Orchestrator) subject(workPackageID string, result worker.Result) gate.Subject {
	var tests []gate.TestSummary
	for _, t := range result.Tests {
		tests = append(tests, gate.TestSummary{Name: t.Name, Outcome: t.Outcome})
	}
	return gate.Subject{WorkPackageID: workPackageID, Status: result.Status, Tests: tests}
}

func (o *Orchestrator) recordTelemetry(assignment *control.Assignment, result worker.Result) {
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

func (o *Orchestrator) emit(eventType string, payload map[string]any) {
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

func sortedKeys(m map[string]*control.Assignment) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
