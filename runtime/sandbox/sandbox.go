// Package sandbox implements runtime execution isolation.
//
// The sandbox is the execution-isolation boundary behind the worker adapter:
// the kernel scheduler and agent roles never name a sandbox, so isolation is a
// runtime policy. A run is confined to a working directory under Config.Root,
// receives only allowlisted environment variables, is bounded by a wall-clock
// timeout and an output cap, and must hold every capability it declares.
//
// The package does not import toolbox: capability decisions are delegated to a
// caller-supplied Policy (the toolbox policy engine at composition time).
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Typed errors. Errors propagate fail-closed and are matched with errors.Is.
var (
	ErrInvalid       = errors.New("sandbox: invalid spec")
	ErrDenied        = errors.New("sandbox: permission denied")
	ErrPathEscape    = errors.New("sandbox: path escapes root")
	ErrTimeout       = errors.New("sandbox: timed out")
	ErrLimitExceeded = errors.New("sandbox: resource limit exceeded")
	ErrExecution     = errors.New("sandbox: execution failed")
)

// Limits bound a run.
type Limits struct {
	// Timeout is the wall-clock limit; a zero value disables it.
	Timeout time.Duration
	// MaxOutputBytes caps each of stdout and stderr; zero disables the cap.
	MaxOutputBytes int
}

// Spec is one isolated run.
type Spec struct {
	Command     []string
	Dir         string            // optional, relative to Config.Root
	Env         map[string]string // only Config.EnvAllowlist keys survive
	Stdin       []byte
	Permissions []string // capabilities the run must hold
	Labels      map[string]string
}

// Result is the bounded outcome of a run.
type Result struct {
	ExitCode  int
	Stdout    []byte
	Stderr    []byte
	Truncated bool
	Duration  time.Duration
}

// Event is a sandbox observability record.
type Event struct {
	Type       string
	Command    []string
	Dir        string
	ExitCode   int
	DurationMS int64
	Reason     string
	Labels     map[string]string
}

// Policy decides whether a run may hold the requested capabilities. It is
// implemented by the toolbox policy engine and supplied at composition time.
type Policy interface {
	Allowed(capabilities []string) bool
}

// Observer receives sandbox events. It is optional and must not block a run.
type Observer interface {
	Sandbox(event Event)
}

// Config configures a Sandbox. Root is required.
type Config struct {
	Root         string
	EnvAllowlist []string
	Limits       Limits
	Policy       Policy
	Observer     Observer
	Runner       Runner
}

// Sandbox runs bounded, isolated commands.
type Sandbox struct {
	root         string
	envAllowlist map[string]struct{}
	limits       Limits
	policy       Policy
	observer     Observer
	runner       Runner
}

// New validates cfg and returns a Sandbox. Runner defaults to the OS process
// runner.
func New(cfg Config) (*Sandbox, error) {
	if strings.TrimSpace(cfg.Root) == "" {
		return nil, fmt.Errorf("%w: root is required", ErrInvalid)
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("%w: root: %v", ErrInvalid, err)
	}
	allow := make(map[string]struct{}, len(cfg.EnvAllowlist))
	for _, key := range cfg.EnvAllowlist {
		if key = strings.TrimSpace(key); key != "" {
			allow[key] = struct{}{}
		}
	}
	runner := cfg.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	return &Sandbox{
		root:         root,
		envAllowlist: allow,
		limits:       cfg.Limits,
		policy:       cfg.Policy,
		observer:     cfg.Observer,
		runner:       runner,
	}, nil
}

// Run executes spec under isolation and returns the bounded result. On any
// failure the returned error is typed and the run's workspace is cleaned up.
func (s *Sandbox) Run(ctx context.Context, spec Spec) (Result, error) {
	if len(spec.Command) == 0 {
		return Result{}, fmt.Errorf("%w: command is required", ErrInvalid)
	}
	if s.policy != nil && len(spec.Permissions) > 0 && !s.policy.Allowed(spec.Permissions) {
		s.emit(Event{Type: "SANDBOX_DENIED", Command: spec.Command, Reason: "permission denied", Labels: spec.Labels})
		return Result{}, fmt.Errorf("%w: %s", ErrDenied, strings.Join(spec.Permissions, ","))
	}

	dir, cleanup, err := s.prepareDir(spec.Dir)
	if err != nil {
		return Result{}, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	runCtx := ctx
	if s.limits.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, s.limits.Timeout)
		defer cancel()
	}

	s.emit(Event{Type: "SANDBOX_STARTED", Command: spec.Command, Dir: dir, Labels: spec.Labels})
	started := time.Now()
	out, err := s.runner.Run(runCtx, Command{
		Path:  spec.Command[0],
		Args:  spec.Command[1:],
		Dir:   dir,
		Env:   s.buildEnv(spec.Env),
		Stdin: spec.Stdin,
	})
	duration := time.Since(started)

	if err != nil {
		switch {
		case errors.Is(runCtx.Err(), context.DeadlineExceeded):
			s.emit(Event{Type: "SANDBOX_TIMEOUT", Command: spec.Command, Dir: dir, DurationMS: duration.Milliseconds(), Labels: spec.Labels})
			return Result{Duration: duration}, fmt.Errorf("%w after %s", ErrTimeout, duration)
		case errors.Is(runCtx.Err(), context.Canceled):
			s.emit(Event{Type: "SANDBOX_CANCELLED", Command: spec.Command, Dir: dir, DurationMS: duration.Milliseconds(), Labels: spec.Labels})
			return Result{Duration: duration}, ctx.Err()
		default:
			s.emit(Event{Type: "SANDBOX_FAILED", Command: spec.Command, Dir: dir, DurationMS: duration.Milliseconds(), Reason: err.Error(), Labels: spec.Labels})
			return Result{Duration: duration}, fmt.Errorf("%w: %v", ErrExecution, err)
		}
	}

	result := Result{ExitCode: out.ExitCode, Stdout: out.Stdout, Stderr: out.Stderr, Duration: duration}
	if max := s.limits.MaxOutputBytes; max > 0 && (len(out.Stdout) > max || len(out.Stderr) > max) {
		result.Stdout = truncate(out.Stdout, max)
		result.Stderr = truncate(out.Stderr, max)
		result.Truncated = true
		s.emit(Event{Type: "SANDBOX_LIMIT_EXCEEDED", Command: spec.Command, Dir: dir, ExitCode: out.ExitCode, DurationMS: duration.Milliseconds(), Reason: "output limit", Labels: spec.Labels})
		return result, fmt.Errorf("%w: output exceeded %d bytes", ErrLimitExceeded, max)
	}

	s.emit(Event{Type: "SANDBOX_COMPLETED", Command: spec.Command, Dir: dir, ExitCode: out.ExitCode, DurationMS: duration.Milliseconds(), Labels: spec.Labels})
	return result, nil
}

// prepareDir confines the run to Root. An empty Dir creates a fresh ephemeral
// directory that is removed by the returned cleanup; an explicit Dir must
// resolve inside Root and is caller-owned.
func (s *Sandbox) prepareDir(dir string) (string, func(), error) {
	if strings.TrimSpace(dir) == "" {
		if err := os.MkdirAll(s.root, 0o700); err != nil {
			return "", nil, fmt.Errorf("%w: root: %v", ErrInvalid, err)
		}
		ephemeral, err := os.MkdirTemp(s.root, "run-")
		if err != nil {
			return "", nil, fmt.Errorf("%w: workspace: %v", ErrInvalid, err)
		}
		return ephemeral, func() { _ = os.RemoveAll(ephemeral) }, nil
	}
	resolved := filepath.Join(s.root, dir)
	clean := filepath.Clean(resolved)
	if clean != s.root && !strings.HasPrefix(clean, s.root+string(os.PathSeparator)) {
		return "", nil, fmt.Errorf("%w: %q", ErrPathEscape, dir)
	}
	return clean, nil, nil
}

// buildEnv returns the allowlisted environment, sorted for determinism.
func (s *Sandbox) buildEnv(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		if _, ok := s.envAllowlist[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+env[key])
	}
	return out
}

func (s *Sandbox) emit(event Event) {
	if s.observer != nil {
		s.observer.Sandbox(event)
	}
}

func truncate(data []byte, max int) []byte {
	if len(data) <= max {
		return data
	}
	return data[:max]
}
