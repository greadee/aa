package repo

import (
	"errors"
	"testing"

	v2 "github.com/greadee/aa/contracts/go/v2"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func TestProjectRepositoryCreateGetListAndDuplicate(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	proj := projection.NewMem()
	r := NewProjectRepository(st, proj)
	record, err := r.Create("p1", "Project One")
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "p1" || record.Name != "Project One" || record.State != v2.ProjectCreated {
		t.Fatalf("record = %+v", record)
	}
	got, err := r.Get("p1")
	if err != nil || got.ID != "p1" {
		t.Fatalf("get = %+v err = %v", got, err)
	}
	list, err := r.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v err = %v", list, err)
	}
	if _, err := r.Create("p1", "Duplicate"); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate err = %v, want ErrExists", err)
	}
	// Restart: a new repository over the same store must see the record.
	reopened := NewProjectRepository(st, proj)
	if _, err := reopened.Get("p1"); err != nil {
		t.Fatalf("reopened get: %v", err)
	}
}

func TestWorkPackageRepositoryDefaultsAndRestart(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	proj := projection.NewMem()
	r := NewWorkPackageRepository(st, proj)
	record, err := r.Create(WorkPackageRecord{
		WorkPackage: v2.WorkPackage{
			Envelope:   v2.Envelope{ProjectID: "p1"},
			Title:      "Persist work",
			Acceptance: []string{"Tests pass."},
		},
		Capabilities: []string{"read_project"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID == "" || record.State != v2.WorkPackageProposed {
		t.Fatalf("record = %+v", record)
	}
	if len(record.Acceptance) != 1 || record.Acceptance[0] != "Tests pass." {
		t.Fatalf("acceptance = %v", record.Acceptance)
	}
	if len(record.Capabilities) != 1 || record.Capabilities[0] != "read_project" {
		t.Fatalf("capabilities = %v", record.Capabilities)
	}
	updated, err := r.UpdateState(record.ID, v2.WorkPackageReady)
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != v2.WorkPackageReady || updated.Revision == nil || *updated.Revision != 2 {
		t.Fatalf("updated = %+v", updated)
	}
	list, err := r.ListByProject("p1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v err = %v", list, err)
	}
	reopened := NewWorkPackageRepository(st, proj)
	got, err := reopened.Get(record.ID)
	if err != nil {
		t.Fatalf("reopened get: %v", err)
	}
	if got.State != v2.WorkPackageReady || len(got.Capabilities) != 1 {
		t.Fatalf("reopened = %+v", got)
	}
}

func TestWorkPackageRepositoryDefaultsAcceptanceWhenOmitted(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := NewWorkPackageRepository(st, projection.NewMem())
	record, err := r.Create(WorkPackageRecord{WorkPackage: v2.WorkPackage{
		Envelope: v2.Envelope{ProjectID: "p1"},
		Title:    "Default acceptance",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Acceptance) == 0 {
		t.Fatal("acceptance was not defaulted")
	}
	if _, err := r.UpdateState(record.ID, v2.WorkPackageState("BOGUS")); err == nil {
		t.Fatal("expected invalid state error")
	}
}
