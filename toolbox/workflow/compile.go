// Package workflow validates declarative workflows into deterministic plans
// and runs them resumably.
//
// A workflow is data: the compiler validates its structure and dependency graph
// and yields a deterministic topological plan; the runtime executes that plan
// over a Runner seam with bounded retries, gates, and per-step failure policy.
package workflow

import (
	"fmt"
	"sort"

	toolbox "github.com/greadee/aa/toolbox"
)

// Plan is a validated, deterministic execution plan for a workflow.
type Plan struct {
	WorkflowID string                          `json:"workflowId"`
	Version    string                          `json:"version"`
	Budget     *toolbox.Budget                 `json:"budget,omitempty"`
	Order      []string                        `json:"order"`
	Steps      map[string]toolbox.WorkflowStep `json:"steps"`
	DependsOn  map[string][]string             `json:"dependsOn"`
}

// Compile validates a workflow and produces a deterministic topological plan.
//
// It rejects a workflow that fails the contract, a tool step without a tool, a
// gate step without gates, a dangling or self dependency, and any cycle.
func Compile(w toolbox.Workflow) (Plan, error) {
	if err := w.Validate(); err != nil {
		return Plan{}, fmt.Errorf("%w: %s", toolbox.ErrInvalid, err)
	}

	steps := make(map[string]toolbox.WorkflowStep, len(w.Steps))
	for _, s := range w.Steps {
		steps[s.ID] = s
	}

	depends := make(map[string][]string, len(w.Steps))
	for _, s := range w.Steps {
		if s.Kind == "tool" && s.ToolID == "" {
			return Plan{}, fmt.Errorf("%w: step %q is a tool step without a toolId", toolbox.ErrInvalid, s.ID)
		}
		if s.Kind == "gate" && len(s.Gates) == 0 {
			return Plan{}, fmt.Errorf("%w: step %q is a gate step without gates", toolbox.ErrInvalid, s.ID)
		}
		if s.Retries != nil && *s.Retries < 0 {
			return Plan{}, fmt.Errorf("%w: step %q has negative retries", toolbox.ErrInvalid, s.ID)
		}
		deps := make([]string, 0, len(s.DependsOn))
		seen := map[string]bool{}
		for _, dep := range s.DependsOn {
			if dep == s.ID {
				return Plan{}, fmt.Errorf("%w: step %q depends on itself", toolbox.ErrInvalid, s.ID)
			}
			if _, ok := steps[dep]; !ok {
				return Plan{}, fmt.Errorf("%w: step %q depends on unknown step %q", toolbox.ErrInvalid, s.ID, dep)
			}
			if seen[dep] {
				continue
			}
			seen[dep] = true
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		depends[s.ID] = deps
	}

	order, err := topoOrder(steps, depends)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		WorkflowID: w.ID,
		Version:    w.Version,
		Budget:     w.Budget,
		Order:      order,
		Steps:      steps,
		DependsOn:  depends,
	}, nil
}

// topoOrder returns a deterministic order: among ready steps, the
// lexicographically smallest ID is emitted first.
func topoOrder(steps map[string]toolbox.WorkflowStep, depends map[string][]string) ([]string, error) {
	indegree := make(map[string]int, len(steps))
	dependents := make(map[string][]string, len(steps))
	for id := range steps {
		indegree[id] = 0
	}
	for id, deps := range depends {
		indegree[id] = len(deps)
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], id)
		}
	}
	ready := make([]string, 0, len(steps))
	for id, deg := range indegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(steps))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		next := append([]string(nil), dependents[id]...)
		sort.Strings(next)
		for _, dep := range next {
			indegree[dep]--
			if indegree[dep] == 0 {
				ready = append(ready, dep)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(steps) {
		return nil, fmt.Errorf("%w: dependency graph has a cycle", toolbox.ErrCycle)
	}
	return order, nil
}

// Step returns a step by ID.
func (p Plan) Step(id string) (toolbox.WorkflowStep, bool) {
	s, ok := p.Steps[id]
	return s, ok
}

// Dependencies returns the sorted dependencies of a step.
func (p Plan) Dependencies(id string) []string {
	deps := p.DependsOn[id]
	out := make([]string, len(deps))
	copy(out, deps)
	return out
}
