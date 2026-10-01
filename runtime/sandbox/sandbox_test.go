package sandbox

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/greadee/aa/runtime/worker"
)

type fakeRunner struct {
	calls int
	cmd   Command
	out   Output
	err   error
	block bool
}

func (f *fakeRunner) Run(ctx context.Context, cmd Command) (Output, error) {
	f.calls++
	f.cmd = cmd
	if f.block {
		<-ctx.Done()
		return Output{}, ctx.Err()
	}
	return f.out, f.err
}

type allowPolicy map[string]bool

func (p allowPolicy) Allowed(caps []string) bool {
	for _, c := range caps {
		if !p[c] {
			return false
		}
	}
	return true
}

type recorder struct{ events []Event }

func (r *recorder) Sandbox(e Event) { r.events = append(r.events, e) }

func newSandbox(t *testing.T, runner Runner, policy Policy, obs Observer) *Sandbox {
	t.Helper()
	s, err := New(Config{
		Root:         t.TempDir(),
		EnvAllowlist: []string{"PATH", "HOME"},
		Limits:       Limits{Timeout: time.Second, MaxOutputBytes: 8},
		Policy:       policy,
		Observer:     obs,
		Runner:       runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRunConfinesEnvAndDir(t *testing.T) {
	runner := &fakeRunner{out: Output{ExitCode: 0, Stdout: []byte("ok")}}
	s := newSandbox(t, runner, nil, nil)
	root := s.root

	result, err := s.Run(context.Background(), Spec{
		Command: []string{"echo", "hi"},
		Env:     map[string]string{"PATH": "/bin", "HOME": "/home/u", "SECRET": "leak"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || string(result.Stdout) != "ok" {
		t.Fatalf("result = %+v", result)
	}
	if got := strings.Join(runner.cmd.Env, ","); got != "HOME=/home/u,PATH=/bin" {
		t.Fatalf("env leaked or unordered: %q", got)
	}
	if !strings.HasPrefix(runner.cmd.Dir, root+string(os.PathSeparator)) {
		t.Fatalf("dir %q not under root %q", runner.cmd.Dir, root)
	}
	if runner.cmd.Path != "echo" || len(runner.cmd.Args) != 1 || runner.cmd.Args[0] != "hi" {
		t.Fatalf("command = %+v", runner.cmd)
	}
}

func TestRunDeniedBeforeExecution(t *testing.T) {
	runner := &fakeRunner{}
	s := newSandbox(t, runner, allowPolicy{"read": true}, nil)
	_, err := s.Run(context.Background(), Spec{Command: []string{"x"}, Permissions: []string{"run_tests"}})
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("runner called despite denial")
	}
}

func TestRunRejectsPathEscape(t *testing.T) {
	runner := &fakeRunner{}
	s := newSandbox(t, runner, nil, nil)
	if _, err := s.Run(context.Background(), Spec{Command: []string{"x"}, Dir: "../outside"}); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunTimeoutFailsClosed(t *testing.T) {
	runner := &fakeRunner{block: true}
	s := newSandbox(t, runner, nil, nil)
	s.limits.Timeout = 10 * time.Millisecond
	if _, err := s.Run(context.Background(), Spec{Command: []string{"sleep"}}); !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunCancellationPropagates(t *testing.T) {
	runner := &fakeRunner{block: true}
	s := newSandbox(t, runner, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Run(ctx, Spec{Command: []string{"x"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunOutputLimitTruncatesAndErrors(t *testing.T) {
	runner := &fakeRunner{out: Output{ExitCode: 0, Stdout: []byte("0123456789abcdef")}}
	s := newSandbox(t, runner, nil, nil)
	result, err := s.Run(context.Background(), Spec{Command: []string{"x"}})
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("err = %v", err)
	}
	if !result.Truncated || len(result.Stdout) != 8 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRunInvalidSpec(t *testing.T) {
	s := newSandbox(t, &fakeRunner{}, nil, nil)
	if _, err := s.Run(context.Background(), Spec{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunCleansUpEphemeralWorkspaceAndEmitsEvents(t *testing.T) {
	obs := &recorder{}
	runner := &fakeRunner{out: Output{ExitCode: 0}}
	s := newSandbox(t, runner, nil, obs)
	if _, err := s.Run(context.Background(), Spec{Command: []string{"x"}, Labels: map[string]string{"workPackageId": "wp_1"}}); err != nil {
		t.Fatal(err)
	}
	var started, completed bool
	var dir string
	for _, e := range obs.events {
		switch e.Type {
		case "SANDBOX_STARTED":
			started = true
			dir = e.Dir
		case "SANDBOX_COMPLETED":
			completed = true
		}
	}
	if !started || !completed || dir == "" {
		t.Fatalf("events = %+v", obs.events)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("ephemeral workspace not cleaned up: %v", err)
	}
}

func TestNewRequiresRoot(t *testing.T) {
	if _, err := New(Config{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestAdapterRunsThroughSandbox(t *testing.T) {
	runner := &fakeRunner{out: Output{ExitCode: 0, Stdout: []byte("done")}}
	s := newSandbox(t, runner, allowPolicy{"read": true}, nil)
	adapter := Adapter{
		Sandbox: s,
		Planner: func(req worker.Request) (Spec, bool) {
			return Spec{Command: []string{"echo", req.WorkPackageID}, Permissions: []string{"read"}}, true
		},
	}
	result, err := adapter.Run(context.Background(), worker.Request{WorkPackageID: "wp_1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "succeeded" || result.Summary != "done" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAdapterClassifiesDeniedAsFailure(t *testing.T) {
	runner := &fakeRunner{}
	s := newSandbox(t, runner, allowPolicy{}, nil)
	adapter := Adapter{Sandbox: s, Planner: func(worker.Request) (Spec, bool) {
		return Spec{Command: []string{"x"}, Permissions: []string{"run_tests"}}, true
	}}
	result, err := adapter.Run(context.Background(), worker.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || result.Failure == nil || result.Failure.Class != "sandbox_denied" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAdapterUnconfiguredAndUnplannable(t *testing.T) {
	blocked, _ := Adapter{}.Run(context.Background(), worker.Request{})
	if blocked.Status != "failed" || blocked.Failure.Class != "sandbox_unconfigured" {
		t.Fatalf("unconfigured = %+v", blocked)
	}
	adapter := Adapter{Sandbox: newSandbox(t, &fakeRunner{}, nil, nil), Planner: func(worker.Request) (Spec, bool) { return Spec{}, false }}
	unplannable, _ := adapter.Run(context.Background(), worker.Request{})
	if unplannable.Status != "blocked" || unplannable.Failure.Class != "not_plannable" {
		t.Fatalf("unplannable = %+v", unplannable)
	}
}
