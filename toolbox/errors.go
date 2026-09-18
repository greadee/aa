package toolbox

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by toolbox components.
var (
	// ErrInvalid means the input did not satisfy the toolbox rules.
	ErrInvalid = errors.New("toolbox: invalid input")
	// ErrNotFound means the referenced tool, workflow, or object does not exist.
	ErrNotFound = errors.New("toolbox: not found")
	// ErrConflict means a conflicting registration or state exists.
	ErrConflict = errors.New("toolbox: conflict")
	// ErrDenied means the caller lacks a required capability or permission.
	ErrDenied = errors.New("toolbox: denied")
	// ErrSandbox means a required sandbox or network grant is missing.
	ErrSandbox = errors.New("toolbox: sandbox required")
	// ErrNoProvider means no provider is registered for a tool's kind.
	ErrNoProvider = errors.New("toolbox: no provider for kind")
	// ErrCycle means a workflow dependency graph contains a cycle.
	ErrCycle = errors.New("toolbox: workflow cycle")
	// ErrFailed means a workflow step failed and its policy did not recover.
	ErrFailed = errors.New("toolbox: workflow step failed")
	// ErrApproval means a workflow is waiting on a human approval gate.
	ErrApproval = errors.New("toolbox: approval required")
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}
