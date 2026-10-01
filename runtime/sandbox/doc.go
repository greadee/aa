// Package sandbox implements runtime execution isolation.
//
// It is the execution-isolation boundary behind the worker adapter: the kernel
// scheduler and agent roles never name a sandbox. A run is confined to a
// working directory under a configured root, receives only allowlisted
// environment variables, is bounded by a wall-clock timeout and an output cap,
// and must hold every capability it declares (decided by a caller-supplied
// policy, so this package does not import toolbox).
//
// Every run emits sandbox events through an Observer supplied by the control
// plane; the kernel bridges those to obsv. Failures propagate as typed errors
// (ErrDenied, ErrPathEscape, ErrTimeout, ErrLimitExceeded, ErrExecution,
// ErrInvalid) and the run's ephemeral workspace is always cleaned up.
package sandbox
