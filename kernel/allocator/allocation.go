package allocator

import (
	"fmt"
	"sort"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/kernel/allocator/compute_allocator"
	"github.com/greadee/aa/kernel/allocator/model_allocator"
	"github.com/greadee/aa/registry/models"
	"github.com/greadee/aa/registry/roles"
)

// Unit is a work unit to allocate: its capabilities (role) and, when it needs a
// model, its model requirements.
type Unit struct {
	ID            string
	Capabilities  []string
	RequiresModel bool
	Model         model_allocator.Requirement
}

// Request is the allocator input.
type Request struct {
	ProjectID string
	PlanID    string
	Units     []Unit
	Compute   compute_allocator.Input
}

// Allocate produces an ExecutionPlan for the scheduler: a deterministic
// role/model/worker assignment per unit plus the plan-wide concurrency, budget,
// and placement. Role, model, and compute are allocated independently.
func Allocate(req Request, modelReg *models.Registry) (ExecutionPlan, error) {
	if len(req.Units) == 0 {
		return ExecutionPlan{}, fmt.Errorf("allocator: at least one unit is required")
	}
	byID := make(map[string]Unit, len(req.Units))
	ids := make([]string, 0, len(req.Units))
	for _, unit := range req.Units {
		if unit.ID == "" {
			return ExecutionPlan{}, fmt.Errorf("allocator: unit id is required")
		}
		if _, dup := byID[unit.ID]; dup {
			return ExecutionPlan{}, fmt.Errorf("allocator: duplicate unit %q", unit.ID)
		}
		byID[unit.ID] = unit
		ids = append(ids, unit.ID)
	}
	sort.Strings(ids)

	compute, err := compute_allocator.Plan(req.Compute)
	if err != nil {
		return ExecutionPlan{}, err
	}

	assignments := make([]PlanAssignment, 0, len(ids))
	for _, id := range ids {
		unit := byID[id]
		assignment := PlanAssignment{Role: primaryRole(unit.Capabilities), Workers: compute.Workers}
		if unit.RequiresModel {
			if modelReg == nil {
				return ExecutionPlan{}, fmt.Errorf("allocator: model registry is required for unit %q", id)
			}
			allocation, err := model_allocator.Allocate(modelReg, unit.Model)
			if err != nil {
				return ExecutionPlan{}, fmt.Errorf("allocator: unit %q: %w", id, err)
			}
			assignment.Model = allocation.Primary.ID
		}
		assignments = append(assignments, assignment)
	}

	planID := req.PlanID
	if planID == "" {
		planID = "plan"
	}
	budget := req.Compute.Budget
	plan := ExecutionPlan{
		Envelope:    v2.Envelope{ContractVersion: "2.0", Kind: "execution_plan", ID: "epl_" + planID, ProjectID: req.ProjectID},
		Assignments: assignments,
		Budget:      &budget,
		Concurrency: compute.Concurrency,
		Placement:   compute.Placement,
	}
	if err := plan.Validate(); err != nil {
		return ExecutionPlan{}, err
	}
	return plan, nil
}

func primaryRole(capabilities []string) string {
	rl := roles.RolesFor(capabilities)
	if len(rl) == 0 {
		return ""
	}
	return string(rl[0])
}
