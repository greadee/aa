// Package worker defines the provider-neutral worker-execution adapter and a
// deterministic scripted fake. It is the runtime boundary that runs an
// allocated attempt; the kernel never calls a model directly, it invokes an
// adapter supplied at composition time.
package worker

import (
	"context"
	"sync"
)

// Request is handed to an adapter for one attempt.
type Request struct {
	AssignmentID   string
	WorkPackageID  string
	ContractID     string
	ContractDigest string
	ContextDigest  string
	Instructions   string
	Capabilities   []string
}

// TestResult is one test outcome reported by an attempt.
type TestResult struct {
	Name    string
	Outcome string // passed, failed, skipped, error
}

// Artifact is an artifact produced by an attempt.
type Artifact struct {
	ID   string
	Kind string
	URI  string
}

// Failure describes why an attempt failed.
type Failure struct {
	Class     string
	Message   string
	Retryable bool
}

// Result is the untrusted output of an attempt.
type Result struct {
	Status    string // succeeded, failed, partial, blocked, cancelled
	Summary   string
	Tests     []TestResult
	Artifacts []Artifact
	Failure   *Failure
}

// Adapter runs an attempt.
type Adapter interface {
	Run(ctx context.Context, req Request) (Result, error)
}

// Fake is a deterministic, scripted adapter for tests and pilots.
type Fake struct {
	mu       sync.Mutex
	scripted map[string]Result
	calls    []Request
}

// NewFake returns an empty fake adapter.
func NewFake() *Fake {
	return &Fake{scripted: make(map[string]Result)}
}

// Script sets the result returned for a work package id.
func (f *Fake) Script(workPackageID string, result Result) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripted[workPackageID] = result
}

// Run returns the scripted result for the request's work package.
func (f *Fake) Run(ctx context.Context, req Request) (Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, req)
	result, ok := f.scripted[req.WorkPackageID]
	f.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{
			Status:  "failed",
			Failure: &Failure{Class: "no_script", Message: "no scripted result"},
		}, nil
	}
	return result, nil
}

// Calls returns the recorded requests in order.
func (f *Fake) Calls() []Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Request, len(f.calls))
	copy(out, f.calls)
	return out
}

// CallCount returns the number of recorded calls.
func (f *Fake) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// Disabled is an adapter used when execution is not enabled. It refuses to run.
type Disabled struct{}

// Run always reports a blocked result.
func (Disabled) Run(context.Context, Request) (Result, error) {
	return Result{
		Status:  "blocked",
		Failure: &Failure{Class: "execution_disabled", Message: "execution is disabled"},
	}, nil
}
