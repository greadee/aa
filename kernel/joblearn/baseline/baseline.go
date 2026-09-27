// Package baseline adapts the governed inference recommender to a backtest
// baseline.
//
// The inference is never imported: the kernel reaches it over the aa inter-module
// RPC, so the engine depends only on the Recommender seam. The production
// adapter (the kernel host) implements that seam; tests provide deterministic
// fakes. A predictor always carries a deterministic fallback, so a disabled or
// unreachable inference never changes a comparison.
package baseline

import (
	"fmt"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/backtest"
)

// Recommender is the governed inference recommender reached over RPC. It maps
// content-free features to a predicted label.
type Recommender interface {
	// Name identifies the recommender for reporting.
	Name() string
	// Recommend returns the predicted label for features.
	Recommend(features map[string]string) (string, error)
}

// InferencePredictor adapts a recommender to a backtest Predictor. When the
// recommender is unavailable or returns no label, it falls back to the
// deterministic predictor it was built with.
type InferencePredictor struct {
	recommender Recommender
	fallback    backtest.Predictor
}

// NewInferencePredictor validates the seam and returns a predictor. Both the
// recommender and the deterministic fallback are required.
func NewInferencePredictor(recommender Recommender, fallback backtest.Predictor) (*InferencePredictor, error) {
	if recommender == nil {
		return nil, invalid("recommender is required")
	}
	if fallback == nil {
		return nil, invalid("deterministic fallback is required")
	}
	return &InferencePredictor{recommender: recommender, fallback: fallback}, nil
}

// Name returns the predictor name used in backtest reports.
func (p *InferencePredictor) Name() string {
	return "inference/" + p.recommender.Name()
}

// Predict returns the recommender's label, or the deterministic fallback when
// the recommender is unavailable or empty. It never panics and never blocks the
// caller on a missing fallback.
func (p *InferencePredictor) Predict(features backtest.Features) string {
	label, err := p.recommender.Recommend(features)
	if err != nil || label == "" {
		return p.fallback.Predict(features)
	}
	return label
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
