package allocator

import v2 "github.com/greadee/aa/contracts/go/v2"

// The allocator's planning contracts, owned by contracts v2: the Planner
// produces a WorkPlan; the allocators produce an ExecutionPlan consumed by the
// scheduler and runtime.
type (
	WorkPlan      = v2.WorkPlan
	ExecutionPlan = v2.ExecutionPlan
)
