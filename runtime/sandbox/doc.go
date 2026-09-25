// Package sandbox is the reserved boundary for execution isolation.
//
// Sandboxing does **not** exist and is intentionally not implemented by the
// architecture refactor. This package reserves the architectural boundary for
// process/workload isolation, filesystem and environment boundaries, resource
// limits, and tool permissions. Sandbox functionality is delivered by the
// ten-issue phase (Sandbox Execution).
package sandbox
