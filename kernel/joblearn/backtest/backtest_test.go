package backtest

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
)

func good() Predictor {
	return PredictorFunc{N: "policy", F: func(Features) string { return "good" }}
}

func bad() Predictor {
	return PredictorFunc{N: "baseline", F: func(Features) string { return "bad" }}
}

func dataset() []Sample {
	return []Sample{
		{ID: "s1", Features: Features{"role": "Builder"}, Actual: "good", Weight: 3},
		{ID: "s2", Features: Features{"role": "Architect"}, Actual: "bad", Weight: 1},
	}
}

func TestRunPolicyBeatsBaseline(t *testing.T) {
	report, err := Run(dataset(), good(), bad(), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.PolicyMetric != 0.75 || report.BaselineMetric != 0.25 {
		t.Fatalf("metrics = %v / %v", report.PolicyMetric, report.BaselineMetric)
	}
	if report.Improvement != 0.5 || !report.Pass {
		t.Fatalf("improvement = %v, pass = %v", report.Improvement, report.Pass)
	}
	if report.Samples != 2 || report.Weight != 4 {
		t.Fatalf("samples/weight = %d/%v", report.Samples, report.Weight)
	}
	if report.Metric != MetricAccuracy {
		t.Fatalf("metric = %q", report.Metric)
	}
}

func TestRunNoImprovement(t *testing.T) {
	report, err := Run(dataset(), good(), good(), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Improvement != 0 || report.Pass {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunDefaultsMetric(t *testing.T) {
	cfg := Config{Budget: DefaultBudget()}
	report, err := Run(dataset(), good(), bad(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Metric != MetricAccuracy {
		t.Fatalf("metric = %q", report.Metric)
	}
}

func TestRunOrderIndependent(t *testing.T) {
	first, err := Run(dataset(), good(), bad(), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	shuffled := dataset()
	shuffled[0], shuffled[1] = shuffled[1], shuffled[0]
	second, err := Run(shuffled, good(), bad(), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("report depends on input order:\n%+v\n%+v", first, second)
	}
}

func TestRunDeterministic(t *testing.T) {
	a, err := Run(dataset(), good(), bad(), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	b, err := Run(dataset(), good(), bad(), DefaultConfig())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("backtest is not deterministic")
	}
}

func TestRunEmptyDataset(t *testing.T) {
	if _, err := Run(nil, good(), bad(), DefaultConfig()); !errors.Is(err, joblearn.ErrNotReady) {
		t.Fatalf("err = %v, want ErrNotReady", err)
	}
}

func TestRunBudget(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Budget = Budget{MaxSamples: 1}
	if _, err := Run(dataset(), good(), bad(), cfg); !errors.Is(err, joblearn.ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
	cfg = DefaultConfig()
	cfg.Budget = Budget{MaxSteps: 3}
	if _, err := Run(dataset(), good(), bad(), cfg); !errors.Is(err, joblearn.ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
}

func TestRunRejectsInvalidInput(t *testing.T) {
	if _, err := Run(dataset(), nil, bad(), DefaultConfig()); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	negative := []Sample{{ID: "s1", Actual: "good", Weight: -1}}
	if _, err := Run(negative, good(), bad(), DefaultConfig()); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestConfigValidate(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	if err := (Config{Metric: "f1", Budget: DefaultBudget()}).Validate(); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if err := (Config{MinImprovement: 2, Budget: DefaultBudget()}).Validate(); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if err := (Config{Budget: Budget{MaxSamples: -1}}).Validate(); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestPredictorFunc(t *testing.T) {
	p := PredictorFunc{N: "const", F: func(Features) string { return "x" }}
	if p.Name() != "const" || p.Predict(Features{}) != "x" {
		t.Fatalf("predictor = %q / %q", p.Name(), p.Predict(nil))
	}
}

func synthetic(n int) []Sample {
	out := make([]Sample, n)
	for i := 0; i < n; i++ {
		out[i] = Sample{ID: fmt.Sprintf("s%06d", i), Features: Features{"role": "Builder"}, Actual: "good"}
	}
	return out
}

func BenchmarkRun(b *testing.B) {
	data := synthetic(50000)
	cfg := DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Run(data, good(), bad(), cfg); err != nil {
			b.Fatal(err)
		}
	}
}
