package lifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/greadee/aa/runtime/worker"
)

type recSink struct {
	mu     sync.Mutex
	events []Event
}

func (s *recSink) Emit(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
}

func (s *recSink) states() []State {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]State, len(s.events))
	for i, ev := range s.events {
		out[i] = ev.State
	}
	return out
}

func TestRunReachesTerminalState(t *testing.T) {
	fake := worker.NewFake()
	fake.Script("wp_1", worker.Result{Status: "succeeded", Summary: "done"})
	sink := &recSink{}
	m := New(Config{Adapter: fake, Sink: sink})

	result, err := m.Run(context.Background(), worker.Request{AssignmentID: "asg_1", WorkPackageID: "wp_1"})
	if err != nil || result.Status != "succeeded" {
		t.Fatalf("result = %+v err = %v", result, err)
	}
	want := []State{StateStarting, StateRunning, StateStopping, StateStopped}
	states := sink.states()
	if len(states) != len(want) {
		t.Fatalf("states = %v, want %v", states, want)
	}
	for i := range want {
		if states[i] != want[i] {
			t.Fatalf("states = %v, want %v", states, want)
		}
	}
	workers := m.Workers()
	if len(workers) != 1 || workers[0].ID != "wrk_asg_1" || workers[0].State != StateStopped {
		t.Fatalf("workers = %+v", workers)
	}
	if workers[0].Status != "succeeded" || workers[0].EndedAt.IsZero() {
		t.Fatalf("worker = %+v", workers[0])
	}
	if m.Events()[0].WorkerID != "wrk_asg_1" {
		t.Fatalf("events = %+v", m.Events())
	}
}

type failingAdapter struct{}

func (failingAdapter) Run(context.Context, worker.Request) (worker.Result, error) {
	return worker.Result{}, errors.New("boom")
}

func TestFailureReachesFailedState(t *testing.T) {
	m := New(Config{Adapter: failingAdapter{}})
	_, err := m.Run(context.Background(), worker.Request{AssignmentID: "asg_2", WorkPackageID: "wp_2"})
	if err == nil {
		t.Fatal("expected an error")
	}
	workers := m.Workers()
	if len(workers) != 1 || workers[0].State != StateFailed || workers[0].Error != "boom" {
		t.Fatalf("workers = %+v", workers)
	}
}

type blockingAdapter struct {
	started chan struct{}
	once    sync.Once
}

func (b *blockingAdapter) Run(ctx context.Context, _ worker.Request) (worker.Result, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	return worker.Result{}, ctx.Err()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met")
}

func TestStopAndTerminate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action func(*Manager, string) bool
		want   State
	}{
		{"stop", (*Manager).Stop, StateStopped},
		{"terminate", (*Manager).Terminate, StateTerminated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &blockingAdapter{started: make(chan struct{})}
			m := New(Config{Adapter: adapter})
			errCh := make(chan error, 1)
			go func() {
				_, err := m.Run(context.Background(), worker.Request{AssignmentID: "asg_3", WorkPackageID: "wp_3"})
				errCh <- err
			}()
			<-adapter.started
			if !m.Active("wrk_asg_3") {
				t.Fatal("worker is not active")
			}
			if !tc.action(m, "wrk_asg_3") {
				t.Fatal("stop/terminate was not accepted")
			}
			if err := <-errCh; !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v", err)
			}
			workers := m.Workers()
			if len(workers) != 1 || workers[0].State != tc.want {
				t.Fatalf("workers = %+v", workers)
			}
			if m.Active("wrk_asg_3") {
				t.Fatal("worker still active after terminal state")
			}
		})
	}
}

func TestCanceledContextDoesNotInstantiate(t *testing.T) {
	m := New(Config{Adapter: worker.NewFake()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Run(ctx, worker.Request{AssignmentID: "asg_4"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if len(m.Workers()) != 0 {
		t.Fatal("a worker was instantiated for a canceled context")
	}
}

type panicSink struct{}

func (panicSink) Emit(Event) { panic("sink failure") }

func TestPanickingSinkIsFailureOpen(t *testing.T) {
	fake := worker.NewFake()
	fake.Script("wp_5", worker.Result{Status: "succeeded"})
	m := New(Config{Adapter: fake, Sink: panicSink{}})
	if _, err := m.Run(context.Background(), worker.Request{AssignmentID: "asg_5", WorkPackageID: "wp_5"}); err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(m.Events()) != 4 {
		t.Fatalf("events = %d", len(m.Events()))
	}
}

func TestEventsAreBounded(t *testing.T) {
	fake := worker.NewFake()
	fake.Script("wp_6", worker.Result{Status: "succeeded"})
	m := New(Config{Adapter: fake, Retain: 2})
	if _, err := m.Run(context.Background(), worker.Request{AssignmentID: "asg_6", WorkPackageID: "wp_6"}); err != nil {
		t.Fatal(err)
	}
	events := m.Events()
	if len(events) != 2 || events[1].State != StateStopped {
		t.Fatalf("events = %+v", events)
	}
}

func TestStopUnknownWorker(t *testing.T) {
	m := New(Config{})
	if m.Stop("missing") || m.Terminate("missing") {
		t.Fatal("unknown worker was stopped or terminated")
	}
}

func TestBeginEndDrivesLifecycle(t *testing.T) {
	sink := &recSink{}
	m := New(Config{Sink: sink})
	id := m.Begin("asg_7", "wp_7", "w1", []string{"write_workspace"})
	if id != "wrk_asg_7" {
		t.Fatalf("id = %s", id)
	}
	if workers := m.Workers(); len(workers) != 1 || workers[0].State != StateRunning || workers[0].SelectedWorker != "w1" {
		t.Fatalf("workers = %+v", workers)
	}
	m.End(id, worker.Result{Status: "succeeded"}, nil)
	workers := m.Workers()
	if workers[0].State != StateStopped || workers[0].Status != "succeeded" {
		t.Fatalf("workers = %+v", workers)
	}
	want := []State{StateStarting, StateRunning, StateStopping, StateStopped}
	states := sink.states()
	if len(states) != len(want) {
		t.Fatalf("states = %v", states)
	}
	for i := range want {
		if states[i] != want[i] {
			t.Fatalf("states = %v", states)
		}
	}
}

func TestEndFailureAndCancellation(t *testing.T) {
	m := New(Config{})
	id := m.Begin("asg_8", "wp_8", "w1", nil)
	m.End(id, worker.Result{}, errors.New("boom"))
	if workers := m.Workers(); workers[0].State != StateFailed || workers[0].Error != "boom" {
		t.Fatalf("workers = %+v", workers)
	}

	id = m.Begin("asg_9", "wp_9", "w1", nil)
	m.End(id, worker.Result{}, context.Canceled)
	if workers := m.Workers(); len(workers) != 2 || workers[1].State != StateStopped {
		t.Fatalf("workers = %+v", workers)
	}
}
