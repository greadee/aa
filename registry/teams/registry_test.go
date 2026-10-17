package teams

import (
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

func spec(id, title string) TeamSpec {
	return TeamSpec{
		Envelope: v2.Envelope{ContractVersion: "2.0", Kind: "team_spec", ID: id},
		Title:    title,
		Roles:    []string{"Builder", "Inspector"},
	}
}

func TestRegistryAddGetList(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(spec("team_engineering", "Engineering")); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(spec("team_qa", "QA")); err != nil {
		t.Fatal(err)
	}
	if r.Count() != 2 {
		t.Fatalf("count = %d", r.Count())
	}
	got, ok := r.Get("team_engineering")
	if !ok || got.Title != "Engineering" {
		t.Fatalf("get = %+v, %v", got, ok)
	}
	list := r.List()
	if len(list) != 2 || list[0].ID != "team_engineering" || list[1].ID != "team_qa" {
		t.Fatalf("list = %+v", list)
	}
}

func TestRegistryRejectsDuplicateAndInvalid(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(spec("team_engineering", "Engineering")); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(spec("team_engineering", "Engineering")); err == nil {
		t.Fatal("expected a duplicate error")
	}
	if err := r.Add(spec("team_bad", "")); err == nil {
		t.Fatal("expected a validation error")
	}
	if r.Count() != 1 {
		t.Fatalf("count = %d", r.Count())
	}
}
