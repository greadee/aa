// Package model_allocator is the reserved boundary for model allocation.
//
// It will select a model from the model registry for each required role/worker
// assignment, owning model identity selection. It must not decide worker count,
// concurrency, or compute budgets. The refactor reserves the boundary; model
// selection behavior is introduced by later issues and contracts v2.
package model_allocator
