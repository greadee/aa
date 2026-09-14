package forge

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by forge implementations.
var (
	// ErrNotFound means the referenced object does not exist.
	ErrNotFound = errors.New("forge: not found")
	// ErrConflict means a conflicting object exists (for example a duplicate tag).
	ErrConflict = errors.New("forge: conflict")
	// ErrRateLimited means the provider asked the caller to back off.
	ErrRateLimited = errors.New("forge: rate limited")
	// ErrHumanGate means a merge was attempted without a human approval.
	ErrHumanGate = errors.New("forge: human approval required to merge")
	// ErrInvalid means the input did not satisfy the forge rules.
	ErrInvalid = errors.New("forge: invalid input")
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}
