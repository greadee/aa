// Package compute_allocator is the reserved boundary for compute allocation.
//
// It will decide worker count, parallelism, reasoning/execution budgets,
// replication, and placement — resource quantity and topology — without ever
// changing model identity. The refactor reserves the boundary; compute
// allocation behavior is introduced by later issues.
package compute_allocator
