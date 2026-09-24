package distill

import (
	"fmt"
	"sort"
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/features"
)

func record(i int, outcome joblearn.Outcome, phase, errClass, role string) Record {
	id := fmt.Sprintf("trc_%d", i)
	return Record{
		Result: attribution.Result{
			Attribution: joblearn.Attribution{
				WorkPackageID: "wp_1",
				AttemptID:     fmt.Sprintf("att_%d", i),
				Role:          role,
				Trade:         "backend",
				Outcome:       outcome,
				Evidence:      []joblearn.Reference{{Kind: "trace", ID: id}},
			},
			Score: joblearn.Score{Version: joblearn.MetricVersion},
		},
		Set: features.Set{
			TraceID: id,
			Steps:   []features.Step{{Phase: phase, Outcome: string(outcome), ErrorClass: errClass}},
		},
	}
}

func keys(candidates []joblearn.Candidate) []string {
	out := make([]string, len(candidates))
	for i, c := range candidates {
		out[i] = string(c.Kind) + "|" + c.Scope + "|" + c.Title
	}
	sort.Strings(out)
	return out
}

func TestDistillStrategyForSuccessfulRole(t *testing.T) {
	distiller, err := New(DefaultPolicy())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	records := []Record{
		record(1, joblearn.OutcomeSucceeded, "test", "", "engineer"),
		record(2, joblearn.OutcomeSucceeded, "test", "", "engineer"),
		record(3, joblearn.OutcomeSucceeded, "test", "", "engineer"),
	}

	candidates, err := distiller.Distill(records)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	found := false
	for _, c := range candidates {
		if c.Kind == joblearn.CandidateStrategy && c.Scope == "role:engineer" {
			found = true
			if len(c.Evidence) != 3 {
				t.Fatalf("expected three trace references, got %+v", c.Evidence)
			}
			if c.Provenance == nil || c.Provenance.Source != DefaultSource {
				t.Fatalf("missing provenance: %+v", c.Provenance)
			}
			if c.Content == "" {
				t.Fatal("expected content")
			}
		}
	}
	if !found {
		t.Fatalf("no role strategy candidate: %+v", candidates)
	}
}

func TestDistillPitfallNamesRecurringErrorClass(t *testing.T) {
	distiller, _ := New(DefaultPolicy())
	records := []Record{
		record(1, joblearn.OutcomeFailed, "test", "assertion", "engineer"),
		record(2, joblearn.OutcomeFailed, "test", "assertion", "engineer"),
		record(3, joblearn.OutcomeFailed, "test", "timeout", "engineer"),
	}

	candidates, err := distiller.Distill(records)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	found := false
	for _, c := range candidates {
		if c.Kind == joblearn.CandidatePitfall && c.Scope == "role:engineer" {
			found = true
			if c.Content == "" || !contains(c.Content, "assertion") {
				t.Fatalf("expected the recurring error class in content: %q", c.Content)
			}
		}
	}
	if !found {
		t.Fatalf("no pitfall candidate: %+v", candidates)
	}
}

func TestDistillSkipsInsufficientEvidence(t *testing.T) {
	distiller, _ := New(Policy{MinOutcomes: 3, SuccessThreshold: 0.8, FailureThreshold: 0.5})
	records := []Record{
		record(1, joblearn.OutcomeSucceeded, "test", "", "engineer"),
		record(2, joblearn.OutcomeSucceeded, "test", "", "engineer"),
	}

	candidates, err := distiller.Distill(records)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %+v", candidates)
	}
}

func TestDistillIsOrderIndependent(t *testing.T) {
	distiller, _ := New(DefaultPolicy())
	records := []Record{
		record(1, joblearn.OutcomeSucceeded, "test", "", "engineer"),
		record(2, joblearn.OutcomeSucceeded, "test", "", "engineer"),
		record(3, joblearn.OutcomeSucceeded, "test", "", "engineer"),
	}
	shuffled := []Record{records[2], records[0], records[1]}

	first, _ := distiller.Distill(records)
	second, _ := distiller.Distill(shuffled)

	if fmt.Sprint(keys(first)) != fmt.Sprint(keys(second)) {
		t.Fatalf("order dependence:\n%v\n%v", keys(first), keys(second))
	}
}

func TestDistillEmpty(t *testing.T) {
	distiller, _ := New(DefaultPolicy())

	candidates, err := distiller.Distill(nil)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %+v", candidates)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
