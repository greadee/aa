package baseline

import (
	"errors"
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/backtest"
)

type fakeRecommender struct {
	name     string
	label    string
	err      error
	calls    int
	features []map[string]string
}

func (f *fakeRecommender) Name() string { return f.name }

func (f *fakeRecommender) Recommend(features map[string]string) (string, error) {
	f.calls++
	f.features = append(f.features, features)
	if f.err != nil {
		return "", f.err
	}
	return f.label, nil
}

func TestPredictUsesRecommender(t *testing.T) {
	recommender := &fakeRecommender{name: "prod", label: "succeeded"}
	predictor, err := NewSifterPredictor(recommender, backtest.ConstantBaseline("failed"))
	if err != nil {
		t.Fatalf("NewSifterPredictor: %v", err)
	}

	if got := predictor.Predict(backtest.Features{"role": "engineer"}); got != "succeeded" {
		t.Fatalf("got %q want succeeded", got)
	}
	if recommender.calls != 1 {
		t.Fatalf("expected one recommender call, got %d", recommender.calls)
	}
}

func TestPredictFallsBackOnError(t *testing.T) {
	recommender := &fakeRecommender{name: "prod", err: errors.New("unreachable")}
	predictor, _ := NewSifterPredictor(recommender, backtest.ConstantBaseline("fallback"))

	if got := predictor.Predict(backtest.Features{}); got != "fallback" {
		t.Fatalf("got %q want fallback", got)
	}
}

func TestPredictFallsBackOnEmptyLabel(t *testing.T) {
	recommender := &fakeRecommender{name: "prod", label: ""}
	predictor, _ := NewSifterPredictor(recommender, backtest.ConstantBaseline("fallback"))

	if got := predictor.Predict(backtest.Features{}); got != "fallback" {
		t.Fatalf("got %q want fallback", got)
	}
}

func TestNameIncludesRecommender(t *testing.T) {
	predictor, _ := NewSifterPredictor(&fakeRecommender{name: "prod"}, backtest.ConstantBaseline("x"))

	if predictor.Name() != "sifter/prod" {
		t.Fatalf("got %q", predictor.Name())
	}
}

func TestNewRequiresSeamAndFallback(t *testing.T) {
	if _, err := NewSifterPredictor(nil, backtest.ConstantBaseline("x")); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a missing recommender, got %v", err)
	}
	if _, err := NewSifterPredictor(&fakeRecommender{name: "prod"}, nil); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a missing fallback, got %v", err)
	}
}
