package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/store"
)

func TestAssignmentRepositorySaveLoadAndRestart(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	proj := projection.NewMem()
	r := NewAssignmentRepository(st, proj)
	record := AssignmentRecord{
		ID:             "asg_1",
		ProjectID:      "p1",
		WorkPackageID:  "wp_1",
		WorkerID:       "w1",
		State:          "running",
		Attempt:        1,
		LeaseOwner:     "w1",
		LeaseExpiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano),
	}
	if _, err := r.Save(record); err != nil {
		t.Fatal(err)
	}
	updated := record
	updated.State = "accepted"
	if _, err := r.Save(updated); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(record.ID)
	if err != nil || got.State != "accepted" {
		t.Fatalf("get = %+v err = %v", got, err)
	}
	list, err := r.ListByProject("p1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v err = %v", list, err)
	}
	reopened := NewAssignmentRepository(st, proj)
	got, err = reopened.Get(record.ID)
	if err != nil || got.State != "accepted" {
		t.Fatalf("reopened = %+v err = %v", got, err)
	}
	if _, err := r.Save(AssignmentRecord{ID: "bad"}); err == nil {
		t.Fatal("expected missing fields error")
	}
}

func TestAssignmentRepositorySaveJSON(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := NewAssignmentRepository(st, projection.NewMem())
	if err := r.SaveJSON([]byte(`{"id":"asg_1","projectId":"p1","workPackageId":"wp_1","workerId":"w1","state":"leased","attempt":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get("asg_1")
	if err != nil || got.State != "leased" {
		t.Fatalf("get = %+v err = %v", got, err)
	}
}

func TestApprovalRepositorySaveListAndRestart(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	proj := projection.NewMem()
	r := NewApprovalRepository(st, proj)
	record := ApprovalRecord{
		WorkPackageID: "wp_1",
		ProjectID:     "p1",
		Decision:      DecisionApprove,
		Actor:         "human",
		Reason:        "looks good",
		DecidedAt:     "2026-01-01T00:00:00Z",
	}
	if _, err := r.Save(record); err != nil {
		t.Fatal(err)
	}
	got, err := r.Get("wp_1")
	if err != nil || got.Decision != DecisionApprove {
		t.Fatalf("get = %+v err = %v", got, err)
	}
	list, err := r.ListByProject("p1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v err = %v", list, err)
	}
	reopened := NewApprovalRepository(st, proj)
	got, err = reopened.Get("wp_1")
	if err != nil || got.Decision != DecisionApprove {
		t.Fatalf("reopened = %+v err = %v", got, err)
	}
	if _, err := r.Save(ApprovalRecord{WorkPackageID: "wp_2", ProjectID: "p1", Decision: "bogus", Actor: "human"}); err == nil {
		t.Fatal("expected invalid decision error")
	}
	if _, err := r.Save(ApprovalRecord{WorkPackageID: "wp_2", ProjectID: "p1", Decision: DecisionReject}); err == nil {
		t.Fatal("expected missing actor error")
	}
}

func TestApprovalRepositoryGetMissing(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := NewApprovalRepository(st, projection.NewMem())
	if _, err := r.Get("missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
