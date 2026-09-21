// Package backtest compares a learned policy against a deterministic baseline
// over a bounded set of labeled historical samples.
//
// A backtest is deterministic: the same dataset, policy, baseline, and config
// always produce the same report, independent of sample order. The report
// exposes the sample count, the improvement over the baseline, and a pass
// verdict, which is what an evidence gate consumes. The real baseline is the
// sifter recommender reached over RPC; until then a Predictor seam and
// deterministic fakes stand in.
package backtest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/greadee/aa/kernel/joblearn"
)

// Features is the input a predictor sees.
type Features map[string]string

// Sample is one labeled historical case. Weight defaults to 1 when not set.
type Sample struct {
	ID       string   `json:"id"`
	Features Features `json:"features,omitempty"`
	Actual   string   `json:"actual"`
	Weight   float64  `json:"weight,omitempty"`
}

// Predictor maps features to a predicted label.
type Predictor interface {
	Name() string
	Predict(Features) string
}

// PredictorFunc adapts a function to a Predictor.
type PredictorFunc struct {
	N string
	F func(Features) string
}

// Name returns the predictor name.
func (p PredictorFunc) Name() string { return p.N }

// Predict returns the predicted label.
func (p PredictorFunc) Predict(f Features) string { return p.F(f) }

// Metric is the comparison metric.
type Metric string

// MetricAccuracy is the weighted fraction of correctly predicted samples.
const MetricAccuracy Metric = "accuracy"

// Budget bounds a single backtest. A non-positive bound is unlimited.
type Budget struct {
	MaxSamples int `json:"maxSamples"`
	MaxSteps   int `json:"maxSteps"`
}

// DefaultBudget is the large-dataset budget.
func DefaultBudget() Budget {
	return Budget{MaxSamples: 100000, MaxSteps: 1000000}
}

// Validate checks that the bounds are non-negative.
func (b Budget) Validate() error {
	if b.MaxSamples < 0 || b.MaxSteps < 0 {
		return invalid("budget must be >= 0")
	}
	return nil
}

// Config configures a backtest.
type Config struct {
	Metric         Metric  `json:"metric"`
	MinImprovement float64 `json:"minImprovement"`
	Budget         Budget  `json:"budget"`
}

// DefaultConfig returns a config requiring a small positive improvement.
func DefaultConfig() Config {
	return Config{Metric: MetricAccuracy, MinImprovement: 0.05, Budget: DefaultBudget()}
}

// Validate checks the metric, threshold, and budget.
func (c Config) Validate() error {
	if c.Metric != "" && c.Metric != MetricAccuracy {
		return invalid("unknown metric %q", c.Metric)
	}
	if c.MinImprovement < 0 || c.MinImprovement > 1 {
		return invalid("minImprovement %v is outside [0,1]", c.MinImprovement)
	}
	return c.Budget.Validate()
}

// Report is the result of a backtest.
type Report struct {
	Policy         string  `json:"policy"`
	Baseline       string  `json:"baseline"`
	Metric         Metric  `json:"metric"`
	Samples        int     `json:"samples"`
	Weight         float64 `json:"weight"`
	PolicyMetric   float64 `json:"policyMetric"`
	BaselineMetric float64 `json:"baselineMetric"`
	Improvement    float64 `json:"improvement"`
	MinImprovement float64 `json:"minImprovement"`
	Pass           bool    `json:"pass"`
}

// Run compares policy against baseline over the dataset and returns a report.
// An empty dataset returns ErrNotReady; exceeding the budget returns ErrBudget.
func Run(dataset []Sample, policy, baseline Predictor, cfg Config) (Report, error) {
	if policy == nil || baseline == nil {
		return Report{}, invalid("policy and baseline are required")
	}
	if cfg.Metric == "" {
		cfg.Metric = MetricAccuracy
	}
	if err := cfg.Validate(); err != nil {
		return Report{}, err
	}
	if len(dataset) == 0 {
		return Report{}, fmt.Errorf("%w: empty dataset", joblearn.ErrNotReady)
	}
	if cfg.Budget.MaxSamples > 0 && len(dataset) > cfg.Budget.MaxSamples {
		return Report{}, fmt.Errorf("%w: %d samples exceeds %d", joblearn.ErrBudget, len(dataset), cfg.Budget.MaxSamples)
	}
	steps := 2 * len(dataset)
	if cfg.Budget.MaxSteps > 0 && steps > cfg.Budget.MaxSteps {
		return Report{}, fmt.Errorf("%w: %d steps exceeds %d", joblearn.ErrBudget, steps, cfg.Budget.MaxSteps)
	}

	ordered := canonical(dataset)
	var total, policyCorrect, baselineCorrect float64
	for _, s := range ordered {
		if s.Weight < 0 {
			return Report{}, invalid("sample %q has negative weight", s.ID)
		}
		weight := s.Weight
		if weight == 0 {
			weight = 1
		}
		total += weight
		if policy.Predict(s.Features) == s.Actual {
			policyCorrect += weight
		}
		if baseline.Predict(s.Features) == s.Actual {
			baselineCorrect += weight
		}
	}

	policyMetric, baselineMetric := 0.0, 0.0
	if total > 0 {
		policyMetric = policyCorrect / total
		baselineMetric = baselineCorrect / total
	}
	improvement := policyMetric - baselineMetric
	return Report{
		Policy:         policy.Name(),
		Baseline:       baseline.Name(),
		Metric:         cfg.Metric,
		Samples:        len(ordered),
		Weight:         total,
		PolicyMetric:   policyMetric,
		BaselineMetric: baselineMetric,
		Improvement:    improvement,
		MinImprovement: cfg.MinImprovement,
		Pass:           improvement >= cfg.MinImprovement,
	}, nil
}

// canonical returns a copy of the dataset sorted by a total, content-derived
// key so the weighted sums do not depend on input order.
func canonical(dataset []Sample) []Sample {
	ordered := append([]Sample(nil), dataset...)
	sort.Slice(ordered, func(i, j int) bool { return sampleKey(ordered[i]) < sampleKey(ordered[j]) })
	return ordered
}

func sampleKey(s Sample) string {
	return s.ID + "|" + s.Actual + "|" + featuresKey(s.Features)
}

func featuresKey(f Features) string {
	if len(f) == 0 {
		return ""
	}
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(f[k])
		b.WriteByte(';')
	}
	return b.String()
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
