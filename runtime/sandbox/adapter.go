package sandbox

import (
	"context"
	"errors"

	"github.com/greadee/aa/runtime/worker"
)

// Planner derives a sandbox Spec from a worker request. It is supplied by the
// control plane at composition time, so agent roles never carry sandbox
// knowledge.
type Planner func(worker.Request) (Spec, bool)

// Adapter runs worker requests through a Sandbox, so execution isolation is a
// runtime detail the kernel scheduler and roles never see.
type Adapter struct {
	Sandbox *Sandbox
	Planner Planner
}

// Run implements worker.Adapter.
func (a Adapter) Run(ctx context.Context, req worker.Request) (worker.Result, error) {
	if a.Sandbox == nil || a.Planner == nil {
		return worker.Result{
			Status:  "failed",
			Failure: &worker.Failure{Class: "sandbox_unconfigured", Message: "sandbox adapter is not configured"},
		}, nil
	}
	spec, ok := a.Planner(req)
	if !ok {
		return worker.Result{
			Status:  "blocked",
			Failure: &worker.Failure{Class: "not_plannable", Message: "request cannot be planned into a sandbox spec"},
		}, nil
	}
	result, err := a.Sandbox.Run(ctx, spec)
	if err != nil {
		return worker.Result{
			Status:  "failed",
			Failure: &worker.Failure{Class: sandboxClass(err), Message: err.Error(), Retryable: errors.Is(err, ErrTimeout)},
		}, nil
	}
	status := "succeeded"
	if result.ExitCode != 0 {
		status = "failed"
	}
	return worker.Result{Status: status, Summary: string(result.Stdout)}, nil
}

func sandboxClass(err error) string {
	switch {
	case errors.Is(err, ErrDenied):
		return "sandbox_denied"
	case errors.Is(err, ErrPathEscape):
		return "sandbox_path_escape"
	case errors.Is(err, ErrTimeout):
		return "sandbox_timeout"
	case errors.Is(err, ErrLimitExceeded):
		return "sandbox_limit"
	case errors.Is(err, ErrInvalid):
		return "sandbox_invalid"
	default:
		return "sandbox_error"
	}
}
