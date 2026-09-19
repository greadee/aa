package route

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
	"github.com/greadee/aa/kernel/joblearn/attribution"
	"github.com/greadee/aa/kernel/joblearn/gate"
)

func enabledGates(t *testing.T) *gate.Registry {
	t.Helper()
	r, err := gate.NewRegistry(joblearn.GatePolicy{})
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if _, err := r.Evaluate(joblearn.CapabilityLearnedRouting, gate.Evidence{}); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !r.Enabled(joblearn.CapabilityLearnedRouting) {
		t.Fatal("learned_routing not enabled")
	}
	return r
}

func disabledGates(t *testing.T) *gate.Registry {
	t.Helper()
	r, err := gate.NewRegistry(joblearn.DefaultGatePolicy())
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return r
}

func result(role, trade, worker string, overall float64) attribution.Result {
	return attribution.Result{
		Attribution: joblearn.Attribution{Role: role, Trade: trade, Worker: worker, Outcome: joblearn.OutcomeSucceeded},
		Score:       joblearn.Score{Version: joblearn.MetricVersion, Overall: overall},
	}
}

func backendHistory() []attribution.Result {
	var out []attribution.Result
	for i := 0; i < 5; i++ {
		out = append(out, result("Builder", "backend", "w1", 0.9))
	}
	for i := 0; i < 5; i++ {
		out = append(out, result("Builder", "backend", "w2", 0.3))
	}
	return out
}

func request() Request {
	return Request{WorkPackageID: "wp1", Role: "Builder", Trade: "backend", Fallback: "fb"}
}

func mustRouter(t *testing.T, gates *gate.Registry, policy Policy) *Router {
	t.Helper()
	r, err := New(gates, policy)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func TestDisabledWithholdsButFallsBack(t *testing.T) {
	router := mustRouter(t, disabledGates(t), DefaultPolicy())
	hint, err := router.Route(request(), backendHistory())
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if hint.Learned || hint.Option != "" {
		t.Fatalf("disabled capability offered a hint: %+v", hint)
	}
	if hint.Fallback != "fb" || hint.Choice() != "fb" {
		t.Fatalf("fallback = %q choice = %q", hint.Fallback, hint.Choice())
	}
	if hint.Authoritative {
		t.Fatal("hint must never be authoritative")
	}
	if len(hint.Reasons) == 0 || !strings.Contains(hint.Reasons[0], "disabled") {
		t.Fatalf("reasons = %v", hint.Reasons)
	}
}

func TestEnabledOffersLearnedHint(t *testing.T) {
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	hint, err := router.Route(request(), backendHistory())
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if !hint.Learned || hint.Option != "w1" {
		t.Fatalf("hint = %+v", hint)
	}
	if hint.Choice() != "w1" || hint.Fallback != "fb" {
		t.Fatalf("choice = %q fallback = %q", hint.Choice(), hint.Fallback)
	}
	if hint.Authoritative {
		t.Fatal("hint must never be authoritative")
	}
}

func TestFallbackWhenNoMargin(t *testing.T) {
	var history []attribution.Result
	for i := 0; i < 5; i++ {
		history = append(history, result("Builder", "backend", "w1", 0.5), result("Builder", "backend", "w2", 0.5))
	}
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	hint, err := router.Route(request(), history)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if hint.Learned || hint.Choice() != "fb" {
		t.Fatalf("hint = %+v", hint)
	}
	if !hasReason(hint.Reasons, "baseline") {
		t.Fatalf("reasons = %v", hint.Reasons)
	}
}

func TestFallbackWhenInsufficientOutcomes(t *testing.T) {
	router := mustRouter(t, enabledGates(t), Policy{MinWorkerOutcomes: 10, MinImprovement: 0.1})
	hint, err := router.Route(request(), backendHistory())
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if hint.Learned {
		t.Fatalf("hint = %+v", hint)
	}
	if !hasReason(hint.Reasons, "outcomes") {
		t.Fatalf("reasons = %v", hint.Reasons)
	}
}

func TestApprovalWithheld(t *testing.T) {
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	req := request()
	req.RequiresApproval = true
	hint, err := router.Route(req, backendHistory())
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if hint.Learned || hint.Choice() != "fb" {
		t.Fatalf("hint = %+v", hint)
	}
	if !hasReason(hint.Reasons, "approval") {
		t.Fatalf("reasons = %v", hint.Reasons)
	}
}

func TestContractIsNotBypassed(t *testing.T) {
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	req := request()
	req.ContractID = "c1"
	hint, err := router.Route(req, backendHistory())
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if !hint.Learned || hint.Authoritative {
		t.Fatalf("hint = %+v", hint)
	}
	if !hasReason(hint.Reasons, "contract") {
		t.Fatalf("reasons = %v", hint.Reasons)
	}
}

func TestScopeRequired(t *testing.T) {
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	hint, err := router.Route(Request{Fallback: "fb"}, backendHistory())
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if hint.Learned || !hasReason(hint.Reasons, "scope") {
		t.Fatalf("hint = %+v", hint)
	}
}

func TestRouteOrderIndependent(t *testing.T) {
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	history := backendHistory()
	first, err := router.Route(request(), history)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	shuffled := append([]attribution.Result(nil), history...)
	for i, j := 0, len(shuffled)-1; i < j; i, j = i+1, j-1 {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	second, err := router.Route(request(), shuffled)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("hint depends on history order:\n%+v\n%+v", first, second)
	}
}

func TestFallbackRequired(t *testing.T) {
	router := mustRouter(t, enabledGates(t), DefaultPolicy())
	if _, err := router.Route(Request{Trade: "backend"}, backendHistory()); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(nil, DefaultPolicy()); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if _, err := New(disabledGates(t), Policy{MinImprovement: 2}); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if err := DefaultPolicy().Validate(); err != nil {
		t.Fatalf("default policy invalid: %v", err)
	}
}

func hasReason(reasons []string, substr string) bool {
	for _, r := range reasons {
		if strings.Contains(r, substr) {
			return true
		}
	}
	return false
}
