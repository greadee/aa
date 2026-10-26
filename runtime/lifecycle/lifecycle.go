// Package lifecycle owns worker instantiation, start/stop/termination, and
// lifecycle events for running workers.
//
// A Manager is a worker.Adapter: the scheduler's dispatch path calls Run, which
// instantiates a worker, starts it, runs the provider-neutral adapter, and
// always reaches a terminal state (stopped, terminated, or failed). Cancellation
// and explicit stop/terminate requests clean up the worker and are recorded.
// Lifecycle events are emitted failure-open: an unavailable sink never changes
// the outcome of the run.
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/greadee/aa/runtime/worker"
)

// State is the lifecycle state of a worker instance.
type State string

// Worker lifecycle states.
const (
	StateCreated    State = "created"
	StateStarting   State = "starting"
	StateRunning    State = "running"
	StateStopping   State = "stopping"
	StateStopped    State = "stopped"
	StateTerminated State = "terminated"
	StateFailed     State = "failed"
)

// Terminal reports whether the state ends the worker lifecycle.
func (s State) Terminal() bool {
	return s == StateStopped || s == StateTerminated || s == StateFailed
}

// Event is one lifecycle transition.
type Event struct {
	WorkerID      string    `json:"workerId"`
	AssignmentID  string    `json:"assignmentId,omitempty"`
	WorkPackageID string    `json:"workPackageId,omitempty"`
	State         State     `json:"state"`
	At            time.Time `json:"at"`
	Reason        string    `json:"reason,omitempty"`
}

// Snapshot is a read view of one worker instance.
type Snapshot struct {
	ID             string    `json:"id"`
	SelectedWorker string    `json:"selectedWorker,omitempty"`
	AssignmentID   string    `json:"assignmentId,omitempty"`
	WorkPackageID  string    `json:"workPackageId,omitempty"`
	Capabilities   []string  `json:"capabilities,omitempty"`
	State          State     `json:"state"`
	CreatedAt      time.Time `json:"createdAt"`
	EndedAt        time.Time `json:"endedAt,omitempty"`
	Status         string    `json:"status,omitempty"`
	Summary        string    `json:"summary,omitempty"`
	Error          string    `json:"error,omitempty"`
}

// Sink receives lifecycle events. Emitting is failure-open: a nil sink discards
// events, and a sink that errors or panics never changes a run.
type Sink interface {
	Emit(Event)
}

// Config configures a lifecycle manager.
type Config struct {
	// Adapter runs the attempt. Nil behaves like worker.Disabled.
	Adapter worker.Adapter
	// Sink receives lifecycle transitions. Optional.
	Sink Sink
	// Now stamps transitions. Defaults to UTC wall clock.
	Now func() time.Time
	// Retain bounds the in-memory event log. Zero or less uses 1024.
	Retain int
}

// record is the mutable state of one worker instance.
type record struct {
	id             string
	selectedWorker string
	assignmentID   string
	workPackageID  string
	capabilities   []string
	state          State
	createdAt      time.Time
	endedAt        time.Time
	status         string
	summary        string
	err            string
}

// control signals a running worker to stop or terminate.
type control struct {
	cancel     context.CancelFunc
	stopped    bool
	terminated bool
}

// Manager instantiates and runs workers.
type Manager struct {
	cfg Config

	mu      sync.Mutex
	seq     int
	workers map[string]*record
	active  map[string]*control
	events  []Event
}

// New returns a lifecycle manager.
func New(cfg Config) *Manager {
	if cfg.Adapter == nil {
		cfg.Adapter = worker.Disabled{}
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Retain <= 0 {
		cfg.Retain = 1024
	}
	return &Manager{
		cfg:     cfg,
		workers: map[string]*record{},
		active:  map[string]*control{},
	}
}

// Run instantiates a worker, starts it, runs the adapter, and reaches a
// terminal state. It implements worker.Adapter so the scheduler dispatches
// through the lifecycle manager.
func (m *Manager) Run(ctx context.Context, req worker.Request) (worker.Result, error) {
	if err := ctx.Err(); err != nil {
		return worker.Result{}, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	id, rec := m.instantiate(req)
	ctrl := &control{cancel: cancel}
	m.mu.Lock()
	m.workers[id] = rec
	m.active[id] = ctrl
	m.mu.Unlock()

	m.transition(id, StateStarting, "")
	m.transition(id, StateRunning, "")

	result, err := m.cfg.Adapter.Run(runCtx, req)

	m.mu.Lock()
	delete(m.active, id)
	terminated := ctrl.terminated
	stopped := ctrl.stopped
	m.mu.Unlock()

	state := StateStopped
	reason := "completed"
	switch {
	case terminated:
		state = StateTerminated
		reason = "terminated"
	case stopped:
		state = StateStopped
		reason = "stopped"
	case err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)):
		state = StateStopped
		reason = "canceled"
	case err != nil:
		state = StateFailed
		reason = err.Error()
	}

	m.finish(id, result, err)
	m.transition(id, StateStopping, reason)
	m.transition(id, state, reason)
	return result, err
}

// Stop requests a graceful stop of a running worker.
func (m *Manager) Stop(workerID string) bool { return m.signal(workerID, false) }

// Terminate requests an immediate termination of a running worker.
func (m *Manager) Terminate(workerID string) bool { return m.signal(workerID, true) }

func (m *Manager) signal(workerID string, terminate bool) bool {
	m.mu.Lock()
	ctrl, ok := m.active[workerID]
	if ok {
		if terminate {
			ctrl.terminated = true
		} else {
			ctrl.stopped = true
		}
	}
	m.mu.Unlock()
	if !ok {
		return false
	}
	ctrl.cancel()
	return true
}

// Workers returns worker snapshots sorted by id.
func (m *Manager) Workers() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Snapshot, 0, len(m.workers))
	for _, rec := range m.workers {
		out = append(out, Snapshot{
			ID:             rec.id,
			SelectedWorker: rec.selectedWorker,
			AssignmentID:   rec.assignmentID,
			WorkPackageID:  rec.workPackageID,
			Capabilities:   append([]string(nil), rec.capabilities...),
			State:          rec.state,
			CreatedAt:      rec.createdAt,
			EndedAt:        rec.endedAt,
			Status:         rec.status,
			Summary:        rec.summary,
			Error:          rec.err,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Events returns the retained lifecycle event log in order.
func (m *Manager) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.events...)
}

// Active returns whether a worker is currently running.
func (m *Manager) Active(workerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.active[workerID]
	return ok
}

// Begin records a worker instantiated by the scheduler's dispatch path, moves
// it through starting and running, and returns the worker instance id. It is
// the lifecycle hook for a scheduler that runs its own adapter.
func (m *Manager) Begin(assignmentID, workPackageID, workerID string, capabilities []string) string {
	m.mu.Lock()
	id := m.nextIDLocked(assignmentID)
	rec := &record{
		id:             id,
		selectedWorker: workerID,
		assignmentID:   assignmentID,
		workPackageID:  workPackageID,
		capabilities:   append([]string(nil), capabilities...),
		state:          StateCreated,
		createdAt:      m.cfg.Now(),
	}
	m.workers[id] = rec
	m.mu.Unlock()

	m.transition(id, StateStarting, "")
	m.transition(id, StateRunning, "")
	return id
}

// End completes a worker started with Begin. Cancellation yields stopped,
// runtime errors yield failed, and a returned result yields stopped.
func (m *Manager) End(workerID string, result worker.Result, err error) {
	state := StateStopped
	reason := "completed"
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		state = StateFailed
		reason = err.Error()
	} else if err != nil {
		reason = "canceled"
	}
	m.finish(workerID, result, err)
	m.transition(workerID, StateStopping, reason)
	m.transition(workerID, state, reason)
}

func (m *Manager) nextIDLocked(assignmentID string) string {
	if assignmentID != "" {
		return "wrk_" + assignmentID
	}
	m.seq++
	return fmt.Sprintf("wrk_%d", m.seq)
}

func (m *Manager) instantiate(req worker.Request) (string, *record) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextIDLocked(req.AssignmentID)
	return id, &record{
		id:            id,
		assignmentID:  req.AssignmentID,
		workPackageID: req.WorkPackageID,
		capabilities:  append([]string(nil), req.Capabilities...),
		state:         StateCreated,
		createdAt:     m.cfg.Now(),
	}
}

func (m *Manager) finish(workerID string, result worker.Result, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.workers[workerID]
	if !ok {
		return
	}
	rec.status = result.Status
	rec.summary = result.Summary
	if err != nil {
		rec.err = err.Error()
	}
	rec.endedAt = m.cfg.Now()
}

func (m *Manager) transition(workerID string, state State, reason string) {
	m.mu.Lock()
	rec, ok := m.workers[workerID]
	if !ok {
		m.mu.Unlock()
		return
	}
	rec.state = state
	event := Event{
		WorkerID:      rec.id,
		AssignmentID:  rec.assignmentID,
		WorkPackageID: rec.workPackageID,
		State:         state,
		At:            m.cfg.Now(),
		Reason:        reason,
	}
	m.events = append(m.events, event)
	if len(m.events) > m.cfg.Retain {
		m.events = append([]Event(nil), m.events[len(m.events)-m.cfg.Retain:]...)
	}
	m.mu.Unlock()
	m.emit(event)
}

// emit is failure-open: a panicking or misbehaving sink cannot alter the run.
func (m *Manager) emit(event Event) {
	if m.cfg.Sink == nil {
		return
	}
	defer func() { _ = recover() }()
	m.cfg.Sink.Emit(event)
}
