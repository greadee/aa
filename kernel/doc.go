// Package kernel is aa's deterministic control plane.
//
// It plans work, selects roles and workers, builds least-privilege execution
// contracts, compiles bounded context, leases assignments, runs work through a
// runtime adapter, validates results, evaluates gates, records telemetry, and
// derives learning candidates. It never calls a model directly and never opens
// a remote shell; execution is disabled unless explicitly enabled.
package kernel
