package compute_allocator

import (
	"errors"
	"testing"
)

func TestPlanRequiresJustificationForParallelism(t *testing.T) {
	compute, err := Plan(Input{Units: 4, IndependentUnits: 4, MaxConcurrency: 4})
	if err != nil {
		t.Fatal(err)
	}
	if compute.Workers != 1 || compute.Concurrency != 1 || compute.Justified {
		t.Fatalf("compute = %+v", compute)
	}
}

func TestPlanJustifiedAndBounded(t *testing.T) {
	compute, err := Plan(Input{Units: 4, IndependentUnits: 4, Justification: JustifyLatency, MaxConcurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	if compute.Workers != 2 || compute.Concurrency != 2 || !compute.Justified {
		t.Fatalf("compute = %+v", compute)
	}
}

func TestPlanSingleIndependentUnitStaysSerial(t *testing.T) {
	compute, err := Plan(Input{Units: 5, IndependentUnits: 1, Justification: JustifyQuality, MaxConcurrency: 4})
	if err != nil {
		t.Fatal(err)
	}
	if compute.Workers != 1 || compute.Justified {
		t.Fatalf("compute = %+v", compute)
	}
}

func TestPlanRejectsInvalid(t *testing.T) {
	if _, err := Plan(Input{Units: -1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("negative units: %v", err)
	}
	if _, err := Plan(Input{Units: 1, Justification: "because"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown justification: %v", err)
	}
}
