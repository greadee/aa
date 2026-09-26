// Package lifecycle is the reserved boundary for worker lifecycle.
//
// It will own instantiation, start/stop, termination, and lifecycle events for
// running workers. The runtime module currently moves execution mechanics here
// structurally; lifecycle behavior is introduced by later issues, not by the
// architecture refactor.
package lifecycle
