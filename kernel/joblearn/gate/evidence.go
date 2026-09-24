package gate

import (
	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
)

// OutcomeCounts aggregates trace-derived attributed outcomes.
type OutcomeCounts struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// CountOutcomes aggregates attributed, trace-derived results by outcome. The
// count is order-independent because it is a sum.
func CountOutcomes(results []attribution.Result) OutcomeCounts {
	var counts OutcomeCounts
	for _, result := range results {
		counts.Total++
		switch result.Attribution.Outcome {
		case joblearn.OutcomeSucceeded:
			counts.Succeeded++
		case joblearn.OutcomeFailed:
			counts.Failed++
		}
	}
	return counts
}

// EvidenceFromTraces builds gate evidence from trace-derived outcomes and a
// measured backtest improvement. Gates stay disabled unless every requirement
// is met, so passing fallback or noGateBypass as false disables the capability
// even when the evidence is otherwise sufficient.
func EvidenceFromTraces(results []attribution.Result, improvement float64, baseline, fallback, noGateBypass bool) Evidence {
	return Evidence{
		Outcomes:     len(results),
		Baseline:     baseline,
		Improvement:  improvement,
		Fallback:     fallback,
		NoGateBypass: noGateBypass,
	}
}
