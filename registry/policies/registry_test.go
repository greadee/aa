package policies

import (
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

func spec(id, title string, kind Kind) PolicySpec {
	return PolicySpec{
		Envelope:   v2.Envelope{ContractVersion: "2.0", Kind: "policy_definition", ID: id},
		Title:      title,
		PolicyKind: kind,
		Version:    "1.0",
	}
}

func TestRegistryAddGetList(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(spec("policy_budget", "Budget", KindBudget)); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(spec("policy_gate", "Gate", KindGate)); err != nil {
		t.Fatal(err)
	}
	if r.Count() != 2 {
		t.Fatalf("count = %d", r.Count())
	}
	got, ok := r.Get("policy_budget")
	if !ok || got.PolicyKind != KindBudget {
		t.Fatalf("get = %+v, %v", got, ok)
	}
	list := r.List()
	if len(list) != 2 || list[0].ID != "policy_budget" || list[1].ID != "policy_gate" {
		t.Fatalf("list = %+v", list)
	}
}

func TestRegistryRejectsDuplicateAndInvalid(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(spec("policy_budget", "Budget", KindBudget)); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(spec("policy_budget", "Budget", KindBudget)); err == nil {
		t.Fatal("expected a duplicate error")
	}
	if err := r.Add(spec("policy_bad", "Bad", Kind("unknown"))); err == nil {
		t.Fatal("expected an invalid kind error")
	}
	if err := r.Add(spec("policy_bad", "", KindBudget)); err == nil {
		t.Fatal("expected a missing title error")
	}
	if r.Count() != 1 {
		t.Fatalf("count = %d", r.Count())
	}
}

func TestKinds(t *testing.T) {
	for _, kind := range Kinds() {
		if !ValidKind(kind) {
			t.Fatalf("kind %q is not valid", kind)
		}
	}
	if ValidKind("other") {
		t.Fatal("unknown kind reported valid")
	}
}
