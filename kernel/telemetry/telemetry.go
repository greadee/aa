// Package telemetry records bounded execution evidence and derives
// deterministic learning candidates. Candidates are proposals for aa-memory;
// they are never authoritative and never bypass a gate.
package telemetry

import (
	"fmt"
	"sort"
	"strings"
)

// Record is bounded execution evidence for one attempt.
type Record struct {
	AttemptID     string
	AssignmentID  string
	WorkPackageID string
	Outcome       string
	Calls         int
	Tokens        int
	CostUSD       float64
	DurationMS    int
}

// Candidate is a proposed piece of knowledge awaiting validation.
type Candidate struct {
	Kind       string   `json:"kind"`
	Level      string   `json:"level"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Evidence   []string `json:"evidence"`
	Confidence float64  `json:"confidence"`
}

// Derive returns deterministic learning candidates. failedTests are the names
// of tests that did not pass.
func Derive(record Record, failedTests []string) []Candidate {
	var candidates []Candidate
	switch record.Outcome {
	case "succeeded":
		candidates = append(candidates, Candidate{
			Kind:       "success",
			Level:      "task",
			Title:      fmt.Sprintf("Work package %s succeeded", record.WorkPackageID),
			Content:    fmt.Sprintf("Succeeded with %d calls and %d tokens", record.Calls, record.Tokens),
			Evidence:   []string{record.AttemptID},
			Confidence: 0.6,
		})
	case "failed":
		if len(failedTests) > 0 {
			sorted := make([]string, len(failedTests))
			copy(sorted, failedTests)
			sort.Strings(sorted)
			candidates = append(candidates, Candidate{
				Kind:       "pitfall",
				Level:      "project",
				Title:      fmt.Sprintf("Failure in %s", record.WorkPackageID),
				Content:    "Tests failed: " + strings.Join(sorted, ", "),
				Evidence:   []string{record.AttemptID},
				Confidence: 0.5,
			})
		} else {
			candidates = append(candidates, Candidate{
				Kind:       "failure",
				Level:      "project",
				Title:      fmt.Sprintf("Failure in %s", record.WorkPackageID),
				Content:    "Attempt failed without test detail",
				Evidence:   []string{record.AttemptID},
				Confidence: 0.4,
			})
		}
	}
	if record.CostUSD >= 1.0 {
		candidates = append(candidates, Candidate{
			Kind:       "cost",
			Level:      "project",
			Title:      fmt.Sprintf("High cost for %s", record.WorkPackageID),
			Content:    fmt.Sprintf("Cost %.2f USD exceeded the 1.00 USD threshold", record.CostUSD),
			Evidence:   []string{record.AttemptID},
			Confidence: 0.5,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Kind != candidates[j].Kind {
			return candidates[i].Kind < candidates[j].Kind
		}
		return candidates[i].Title < candidates[j].Title
	})
	return candidates
}

// Summary aggregates a set of records.
type Summary struct {
	Attempts    int
	Successes   int
	Failures    int
	TotalCalls  int
	TotalTokens int
	TotalCost   float64
}

// Summarize aggregates records deterministically.
func Summarize(records []Record) Summary {
	var s Summary
	for _, r := range records {
		s.Attempts++
		switch r.Outcome {
		case "succeeded":
			s.Successes++
		case "failed":
			s.Failures++
		}
		s.TotalCalls += r.Calls
		s.TotalTokens += r.Tokens
		s.TotalCost += r.CostUSD
	}
	return s
}
