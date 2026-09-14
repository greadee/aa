package sync

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by sync components.
var (
	// ErrInvalid means the input did not satisfy the sync rules.
	ErrInvalid = errors.New("sync: invalid input")
	// ErrNotFound means the referenced object does not exist.
	ErrNotFound = errors.New("sync: not found")
	// ErrConflict means a conflicting object or revision exists.
	ErrConflict = errors.New("sync: conflict")
	// ErrUnauthorized means a peer failed authentication or pairing.
	ErrUnauthorized = errors.New("sync: unauthorized")
	// ErrStale means a pairing token or revision is no longer current.
	ErrStale = errors.New("sync: stale")
	// ErrCorrupt means a chunk or payload failed hash verification.
	ErrCorrupt = errors.New("sync: corrupt payload")
	// ErrDeletionGuard means a tombstone was refused by the deletion guard.
	ErrDeletionGuard = errors.New("sync: deletion refused")
	// ErrAuthority means an operation tried to gain execution authority.
	ErrAuthority = errors.New("sync: execution authority is not granted")
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}
