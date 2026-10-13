// Package model_allocator selects a model from the model registry.
//
// It answers one allocation question: which model identity / inference
// configuration runs the work? It owns model identity selection, including an
// escalation target, and must not decide worker count, concurrency, or compute
// budgets — that is compute_allocator. Selection is deterministic.
package model_allocator

import (
	"errors"
	"fmt"
	"sort"

	"github.com/greadee/aa/registry/models"
)

// ErrNoModel is returned when no registered model satisfies a requirement.
var ErrNoModel = errors.New("model_allocator: no eligible model")

// Requirement describes a work unit's model needs.
type Requirement struct {
	Capabilities []string
	// Locality is "", "either", "local", or "cloud".
	Locality string
	// MaxInputCostPer1K caps cloud cost in USD per thousand input tokens; 0 = no cap.
	MaxInputCostPer1K float64
}

// Allocation is the selected primary model and an optional escalation target.
type Allocation struct {
	Primary    models.ModelSpec
	Escalation *models.ModelSpec
	Reasons    []string
}

// Allocate selects a primary model deterministically (lowest cost, then lowest
// latency, then id) and, unless the primary already reasons, a stronger
// reasoning target for escalation. It is pure over the registry.
func Allocate(reg *models.Registry, req Requirement) (Allocation, error) {
	eligible := make([]models.ModelSpec, 0, reg.Count())
	for _, spec := range reg.List() {
		if satisfies(spec, req) {
			eligible = append(eligible, spec)
		}
	}
	if len(eligible) == 0 {
		return Allocation{}, fmt.Errorf("%w: capabilities=%v locality=%q", ErrNoModel, req.Capabilities, req.Locality)
	}
	sort.Slice(eligible, func(i, j int) bool {
		ci, cj := inputCost(eligible[i]), inputCost(eligible[j])
		if ci != cj {
			return ci < cj
		}
		if eligible[i].LatencyMS != eligible[j].LatencyMS {
			return eligible[i].LatencyMS < eligible[j].LatencyMS
		}
		return eligible[i].ID < eligible[j].ID
	})

	allocation := Allocation{Primary: eligible[0], Reasons: []string{"lowest cost eligible model"}}
	if !hasCapability(allocation.Primary, "reasoning") {
		if strongest, ok := strongestReasoning(eligible); ok {
			allocation.Escalation = &strongest
			allocation.Reasons = append(allocation.Reasons, "escalation target: reasoning-capable model")
		}
	}
	return allocation, nil
}

func satisfies(spec models.ModelSpec, req Requirement) bool {
	if !spec.Available {
		return false
	}
	if req.Locality != "" && req.Locality != "either" && spec.Locality != req.Locality {
		return false
	}
	if req.MaxInputCostPer1K > 0 && inputCost(spec) > req.MaxInputCostPer1K {
		return false
	}
	for _, required := range req.Capabilities {
		if !hasCapability(spec, required) {
			return false
		}
	}
	return true
}

func strongestReasoning(eligible []models.ModelSpec) (models.ModelSpec, bool) {
	var best models.ModelSpec
	found := false
	for _, spec := range eligible {
		if !hasCapability(spec, "reasoning") {
			continue
		}
		if !found || len(spec.Capabilities) > len(best.Capabilities) ||
			(len(spec.Capabilities) == len(best.Capabilities) && spec.ID < best.ID) {
			best = spec
			found = true
		}
	}
	return best, found
}

func hasCapability(spec models.ModelSpec, capability string) bool {
	for _, c := range spec.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

func inputCost(spec models.ModelSpec) float64 {
	if spec.CostPer1KInput == nil {
		return 0
	}
	return *spec.CostPer1KInput
}
