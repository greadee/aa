// Package role_allocator selects the worker instances whose expertise satisfies
// a work requirement, deterministically, with rejection reasons.
//
// It answers one allocation question: what expertise is required? It allocates
// against role, capability, and (transitionally) trade; it does not choose a
// model or decide compute quantity. Selection is deterministic: accepted
// workers are ordered by cost weight then id, and rejected workers are reported
// with a reason.
package role_allocator

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/greadee/aa/registry/roles"
	"github.com/greadee/aa/runtime/worker"
)

// Registry is a set of worker instances the allocator can select from.
type Registry struct {
	workers map[worker.WorkerID]worker.Worker
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{workers: make(map[worker.WorkerID]worker.Worker)}
}

// ErrDuplicate is returned when a worker id already exists.
var ErrDuplicate = errors.New("role_allocator: duplicate worker")

// Add registers a worker.
func (r *Registry) Add(w worker.Worker) error {
	if w.ID == "" {
		return fmt.Errorf("role_allocator: worker id is required")
	}
	if _, ok := r.workers[w.ID]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicate, w.ID)
	}
	r.workers[w.ID] = w
	return nil
}

// Get returns a worker by id.
func (r *Registry) Get(id worker.WorkerID) (worker.Worker, bool) {
	w, ok := r.workers[id]
	return w, ok
}

// List returns all workers sorted by id.
func (r *Registry) List() []worker.Worker {
	out := make([]worker.Worker, 0, len(r.workers))
	for _, w := range r.workers {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Requirement describes what a work package needs.
type Requirement struct {
	Trade        worker.Trade
	Roles        []roles.Role
	Capabilities []string
}

// Candidate is an eligible worker with selection reasons.
type Candidate struct {
	Worker  worker.Worker
	Reasons []string
}

// Rejection is a worker that did not satisfy a requirement, with a reason.
type Rejection struct {
	Worker worker.Worker
	Reason string
}

// Select returns eligible workers in deterministic preference order, plus the
// rejected workers and why. Order is by cost weight then id.
func (r *Registry) Select(req Requirement) ([]Candidate, []Rejection) {
	var accepted []Candidate
	var rejected []Rejection
	for _, w := range r.List() {
		switch {
		case !w.Available:
			rejected = append(rejected, Rejection{w, "unavailable"})
		case req.Trade != "" && w.Trade != req.Trade:
			rejected = append(rejected, Rejection{w, "trade mismatch"})
		case len(req.Roles) > 0 && !hasAnyRole(w.Roles, req.Roles):
			rejected = append(rejected, Rejection{w, "role mismatch"})
		default:
			if missing := missingCapabilities(w.Capabilities, req.Capabilities); len(missing) > 0 {
				rejected = append(rejected, Rejection{w, "missing capabilities: " + strings.Join(missing, ",")})
				continue
			}
			accepted = append(accepted, Candidate{Worker: w, Reasons: []string{"eligible"}})
		}
	}
	sort.Slice(accepted, func(i, j int) bool {
		if accepted[i].Worker.CostWeight != accepted[j].Worker.CostWeight {
			return accepted[i].Worker.CostWeight < accepted[j].Worker.CostWeight
		}
		return accepted[i].Worker.ID < accepted[j].Worker.ID
	})
	sort.Slice(rejected, func(i, j int) bool { return rejected[i].Worker.ID < rejected[j].Worker.ID })
	return accepted, rejected
}

// SelectOne returns the preferred eligible worker, if any.
func (r *Registry) SelectOne(req Requirement) (worker.Worker, bool) {
	accepted, _ := r.Select(req)
	if len(accepted) == 0 {
		return worker.Worker{}, false
	}
	return accepted[0].Worker, true
}

func hasAnyRole(have, want []roles.Role) bool {
	for _, w := range want {
		for _, h := range have {
			if h == w {
				return true
			}
		}
	}
	return false
}

func missingCapabilities(have, want []string) []string {
	set := make(map[string]bool, len(have))
	for _, c := range have {
		set[c] = true
	}
	var missing []string
	for _, c := range want {
		if !set[c] {
			missing = append(missing, c)
		}
	}
	sort.Strings(missing)
	return missing
}
