package visualizer

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by visualizer components.
var (
	// ErrInvalid means the input did not satisfy the visualizer rules.
	ErrInvalid = errors.New("visualizer: invalid input")
	// ErrNotFound means the referenced session, node, or object does not exist.
	ErrNotFound = errors.New("visualizer: not found")
	// ErrEmpty means an event stream contained no events.
	ErrEmpty = errors.New("visualizer: empty event stream")
	// ErrBudget means a layout or render exceeded its performance budget.
	ErrBudget = errors.New("visualizer: performance budget exceeded")
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}
