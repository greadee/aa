package backtest

import (
	"sort"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
)

// TraceDataset builds one labeled sample per attributed trace result. The
// features are content-free scope metadata and the label is the observed
// outcome, so a policy that predicts outcomes from scope can be compared
// against a deterministic baseline over real trace evidence.
func TraceDataset(results []attribution.Result) []Sample {
	out := make([]Sample, 0, len(results))
	for _, result := range results {
		attr := result.Attribution
		features := Features{}
		if attr.Role != "" {
			features["role"] = attr.Role
		}
		if attr.Trade != "" {
			features["trade"] = attr.Trade
		}
		if attr.WorkPackageID != "" {
			features["workPackageId"] = attr.WorkPackageID
		}
		out = append(out, Sample{ID: attr.AttemptID, Features: features, Actual: string(attr.Outcome)})
	}
	return out
}

// ConstantBaseline returns a predictor that always predicts label.
func ConstantBaseline(label string) Predictor {
	return PredictorFunc{N: "constant", F: func(Features) string { return label }}
}

// MajorityBaseline returns a deterministic predictor that always predicts the
// most common label in the dataset; ties are broken lexically. It is the
// baseline a learned policy must beat.
func MajorityBaseline(dataset []Sample) Predictor {
	counts := map[string]int{}
	for _, sample := range dataset {
		counts[sample.Actual]++
	}
	best := ""
	bestCount := 0
	for _, label := range sortedLabels(counts) {
		if counts[label] > bestCount {
			best, bestCount = label, counts[label]
		}
	}
	return PredictorFunc{N: "majority", F: func(Features) string { return best }}
}

// ScopeBaseline returns a deterministic predictor that chooses the majority
// label within a sample's work package, falling back to the global majority.
// It is a stronger baseline than the global majority.
func ScopeBaseline(dataset []Sample) Predictor {
	global := MajorityBaseline(dataset)
	byPackage := map[string]map[string]int{}
	for _, sample := range dataset {
		pkg := sample.Features["workPackageId"]
		if pkg == "" {
			continue
		}
		if byPackage[pkg] == nil {
			byPackage[pkg] = map[string]int{}
		}
		byPackage[pkg][sample.Actual]++
	}
	majority := func(counts map[string]int) string {
		best := ""
		bestCount := 0
		for _, label := range sortedLabels(counts) {
			if counts[label] > bestCount {
				best, bestCount = label, counts[label]
			}
		}
		return best
	}
	lookup := make(map[string]string, len(byPackage))
	for pkg, counts := range byPackage {
		lookup[pkg] = majority(counts)
	}
	return PredictorFunc{N: "scope-majority", F: func(f Features) string {
		if label := lookup[f["workPackageId"]]; label != "" {
			return label
		}
		return global.Predict(f)
	}}
}

func sortedLabels(counts map[string]int) []string {
	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}

// OutcomeLabeler maps an attributed result to the label a backtest predicts.
// It exists so a caller can narrow the dataset to a task, for example dropping
// indeterminate outcomes.
type OutcomeLabeler func(joblearn.Outcome) (string, bool)
