package routines

import (
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

func spec(id, name string) RoutineSpec {
	return RoutineSpec{
		Envelope: v2.Envelope{ContractVersion: "2.0", Kind: "routine", ID: id},
		Name:     name,
		Version:  "1.0",
		Stages:   []v2.RoutineStage{{ID: "s1", Title: "Implement"}},
	}
}

func TestRegistryAddGetList(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(spec("routine_feature", "Feature Delivery")); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(spec("routine_release", "Release")); err != nil {
		t.Fatal(err)
	}
	if r.Count() != 2 {
		t.Fatalf("count = %d", r.Count())
	}
	got, ok := r.Get("routine_feature")
	if !ok || got.Name != "Feature Delivery" {
		t.Fatalf("get = %+v, %v", got, ok)
	}
	list := r.List()
	if len(list) != 2 || list[0].ID != "routine_feature" || list[1].ID != "routine_release" {
		t.Fatalf("list = %+v", list)
	}
}

func TestRegistryRejectsDuplicateAndInvalid(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(spec("routine_feature", "Feature Delivery")); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(spec("routine_feature", "Feature Delivery")); err == nil {
		t.Fatal("expected a duplicate error")
	}
	bad := spec("routine_bad", "Bad")
	bad.Stages = nil
	if err := r.Add(bad); err == nil {
		t.Fatal("expected a validation error")
	}
	if r.Count() != 1 {
		t.Fatalf("count = %d", r.Count())
	}
}
