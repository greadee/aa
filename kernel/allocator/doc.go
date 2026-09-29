// Package allocator owns pre-execution planning and resource allocation.
//
// It answers three orthogonal questions for an enriched task:
//
//   - what work needs to happen and how it depends — `planner`;
//   - what expertise and model are required — `role_allocator` and the reserved
//     `model_allocator`;
//   - how much compute, where, and how parallel — the reserved `compute_allocator`.
//
// The allocator produces plans; it does not execute them. The `scheduler`
// consumes the resulting execution plan and the `runtime` module runs it.
//
// Only planning and role allocation carry current behavior. Model allocation
// and compute allocation are reserved architectural boundaries: the refactor
// does not add new allocation algorithms.
package allocator
