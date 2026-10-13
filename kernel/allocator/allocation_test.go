package allocator

import (
	"testing"

	"github.com/greadee/aa/kernel/allocator/compute_allocator"
	"github.com/greadee/aa/kernel/allocator/model_allocator"
	"github.com/greadee/aa/registry/models"
)

func TestAllocateProducesExecutionPlan(t *testing.T) {
	reg, err := models.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Allocate(Request{
		ProjectID: "prj_1",
		PlanID:    "wp_1",
		Units: []Unit{
			{ID: "u2", Capabilities: []string{"run_tests"}},
			{ID: "u1", Capabilities: []string{"write_workspace"}, RequiresModel: true,
				Model: model_allocator.Requirement{Capabilities: []string{"coding"}, Locality: "local"}},
		},
		Compute: compute_allocator.Input{Units: 2, IndependentUnits: 2, Justification: compute_allocator.JustifyCoverage, MaxConcurrency: 2},
	}, reg)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != "execution_plan" || plan.Concurrency != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Assignments) != 2 {
		t.Fatalf("assignments = %+v", plan.Assignments)
	}
	// Deterministic: sorted by unit id (u1 before u2).
	if plan.Assignments[0].Role != "Builder" || plan.Assignments[0].Model == "" || plan.Assignments[0].Workers != 2 {
		t.Fatalf("u1 assignment = %+v", plan.Assignments[0])
	}
	if plan.Assignments[1].Role != "Inspector" {
		t.Fatalf("u2 assignment = %+v", plan.Assignments[1])
	}
}

func TestAllocateRejectsDuplicateUnits(t *testing.T) {
	_, err := Allocate(Request{
		Units: []Unit{{ID: "u1"}, {ID: "u1"}},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
