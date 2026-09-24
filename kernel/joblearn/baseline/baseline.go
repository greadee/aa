// Package baseline adapts the governed sifter recommender to a backtest
// baseline.
//
// The sifter is never imported: the kernel reaches it over the aa inter-module
// RPC, so the engine depends only on the Recommender seam. The production
// adapter (the kernel host) implements that seam; tests provide deterministic
// fakes. A predictor always carries a deterministic fallback, so a disabled or
// unreachable sifter never changes a comparison.
package baseline

import (
	"fmt"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/backtest"
)

// Recommender is the governed sifter recommender reached over RPC. It maps
// content-free features to a predicted label.
type Recommender interface {
	// Name identifies the recommender for reporting.
	Name() string
	// Recommend returns the predicted label for features.
	Recommend(features map[string]string) (string, error)
}

// SifterPredictor adapts a recommender to a backtest Predictor. When the
// recommender is unavailable or returns no label, it falls back to the
// deterministic predictor it was built with.
type SifterPredictor struct {
	recommender Recommender
	fallback    backtest.Predictor
}

// NewSifterPredictor validates the seam and returns a predictor. Both the
// recommender and the deterministic fallback are required.
func NewSifterPredictor(recommender Recommender, fallback backtest.Predictor) (*SifterPredictor, error) {
	if recommender == nil {
		return nil, invalid("recommender is required")
	}
	if fallback == nil {
		return nil, invalid("deterministic fallback is required")
	}
	return &SifterPredictor{recommender: recommender, fallback: fallback}, nil
}

// Name returns the predictor name used in backtest reports.
func (p *SifterPredictor) Name() string {
	return "sifter/" + p.recommender.Name()
}

// Predict returns the recommender's label, or the deterministic fallback when
// the recommender is unavailable or empty. It never panics and never blocks the
// caller on a missing fallback.
func (p *SifterPredictor) Predict(features backtest.Features) string {
	label, err := p.recommender.Recommend(features)
	if err != nil || label == "" {
		return p.fallback.Predict(features)
	}
	return label
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", joblearn.ErrInvalid, fmt.Sprintf(format, args...))
}
