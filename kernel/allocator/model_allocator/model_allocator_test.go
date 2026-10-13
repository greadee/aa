package model_allocator

import (
	"errors"
	"testing"

	"github.com/greadee/aa/registry/models"
)

func defaultRegistry(t *testing.T) *models.Registry {
	t.Helper()
	reg, err := models.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestAllocateIsDeterministicAndEscalates(t *testing.T) {
	reg := defaultRegistry(t)
	allocation, err := Allocate(reg, Requirement{Capabilities: []string{"coding"}, Locality: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if allocation.Primary.ID == "" || allocation.Primary.Locality != "local" {
		t.Fatalf("primary = %+v", allocation.Primary)
	}
	if allocation.Escalation == nil || allocation.Escalation.ID != "mdl_ollama_qwen3.5:32b" {
		t.Fatalf("escalation = %+v", allocation.Escalation)
	}

	again, _ := Allocate(reg, Requirement{Capabilities: []string{"coding"}, Locality: "local"})
	if again.Primary.ID != allocation.Primary.ID {
		t.Fatalf("non-deterministic: %q vs %q", allocation.Primary.ID, again.Primary.ID)
	}
}

func TestAllocateHonorsCostCapAndLocality(t *testing.T) {
	reg := defaultRegistry(t)
	allocation, err := Allocate(reg, Requirement{Capabilities: []string{"coding"}, Locality: "cloud", MaxInputCostPer1K: 0.0002})
	if err != nil {
		t.Fatal(err)
	}
	if allocation.Primary.ID != "mdl_deepseek_deepseek-chat" {
		t.Fatalf("primary = %q", allocation.Primary.ID)
	}
	// The only reasoning cloud model is over the cap, so no escalation target.
	if allocation.Escalation != nil {
		t.Fatalf("unexpected escalation: %+v", allocation.Escalation)
	}
}

func TestAllocateNoEligibleModel(t *testing.T) {
	reg := defaultRegistry(t)
	_, err := Allocate(reg, Requirement{Capabilities: []string{"quantum"}, Locality: "local"})
	if !errors.Is(err, ErrNoModel) {
		t.Fatalf("err = %v", err)
	}
}
