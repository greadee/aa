package models

import "testing"

func TestDefaultCatalogIsValidAndDeterministic(t *testing.T) {
	reg, err := NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Count() != 10 {
		t.Fatalf("expected 10 catalog models, got %d", reg.Count())
	}
	for _, spec := range reg.List() {
		if err := spec.Validate(); err != nil {
			t.Fatalf("catalog model %q invalid: %v", spec.ID, err)
		}
	}
	// List is sorted by id.
	listed := reg.List()
	for i := 1; i < len(listed); i++ {
		if listed[i-1].ID >= listed[i].ID {
			t.Fatalf("not sorted: %q >= %q", listed[i-1].ID, listed[i].ID)
		}
	}
}

func TestAddRejectsInvalidAndDuplicate(t *testing.T) {
	reg := NewRegistry()
	valid := DefaultCatalog()[0]
	if err := reg.Add(valid); err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(valid); err == nil {
		t.Fatal("expected duplicate error")
	}
	bad := valid
	bad.Envelope.ID = "mdl_bad"
	bad.Provider = ""
	if err := reg.Add(bad); err == nil {
		t.Fatal("expected validation error")
	}
}
