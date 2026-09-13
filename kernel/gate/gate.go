// Package gate evaluates deterministic and human gates before work is accepted.
// A gate never mutates the subject; it only reports a decision.
package gate

import (
	"sort"
	"sync"
)

// TestSummary is a test outcome seen by a gate.
type TestSummary struct {
	Name    string
	Outcome string
}

// Subject is what gates evaluate.
type Subject struct {
	WorkPackageID string
	Status        string
	Tests         []TestSummary
	ContractID    string
}

// Decision is the outcome of one gate.
type Decision struct {
	Gate    string `json:"gate"`
	Passed  bool   `json:"passed"`
	Pending bool   `json:"pending"`
	Reason  string `json:"reason,omitempty"`
}

// Gate evaluates a subject.
type Gate interface {
	Name() string
	Evaluate(Subject) Decision
}

// TestsGate requires at least one test, all passed.
type TestsGate struct{}

// Name returns the gate name.
func (TestsGate) Name() string { return "tests" }

// Evaluate checks the reported tests.
func (TestsGate) Evaluate(s Subject) Decision {
	if len(s.Tests) == 0 {
		return Decision{Gate: "tests", Reason: "no tests reported"}
	}
	for _, t := range s.Tests {
		if t.Outcome != "passed" {
			return Decision{Gate: "tests", Reason: "test did not pass: " + t.Name}
		}
	}
	return Decision{Gate: "tests", Passed: true, Reason: "all tests passed"}
}

// HumanGate requires an explicit approval keyed by work package id.
type HumanGate struct {
	mu       sync.Mutex
	approved map[string]bool
}

// NewHumanGate returns an unapproved human gate.
func NewHumanGate() *HumanGate {
	return &HumanGate{approved: make(map[string]bool)}
}

// Name returns the gate name.
func (g *HumanGate) Name() string { return "human_review" }

// Approve records human approval for a work package.
func (g *HumanGate) Approve(workPackageID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.approved[workPackageID] = true
}

// Evaluate reports pending until the work package is approved.
func (g *HumanGate) Evaluate(s Subject) Decision {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.approved[s.WorkPackageID] {
		return Decision{Gate: "human_review", Passed: true, Reason: "approved"}
	}
	return Decision{Gate: "human_review", Pending: true, Reason: "awaiting human approval"}
}

// Report is the combined outcome of a set of gates.
type Report struct {
	Passed    bool       `json:"passed"`
	Pending   bool       `json:"pending"`
	Decisions []Decision `json:"decisions"`
}

// Evaluate runs gates by name (sorted) and reports the combined outcome.
func Evaluate(gates []Gate, s Subject) Report {
	sorted := make([]Gate, len(gates))
	copy(sorted, gates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name() < sorted[j].Name() })

	report := Report{Passed: true}
	for _, g := range sorted {
		d := g.Evaluate(s)
		report.Decisions = append(report.Decisions, d)
		if d.Pending {
			report.Pending = true
			report.Passed = false
			continue
		}
		if !d.Passed {
			report.Passed = false
		}
	}
	return report
}
