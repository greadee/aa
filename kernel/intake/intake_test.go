package intake

import (
	"errors"
	"testing"

	"github.com/greadee/aa/kernel/runtime"
)

func validResult() Result {
	return Result{
		ID:            "res_1",
		AttemptID:     "att_1",
		AssignmentID:  "asg_1",
		WorkPackageID: "wp_1",
		Status:        "succeeded",
		Summary:       "done",
		Tests:         []runtime.TestResult{{Name: "t", Outcome: "passed"}},
	}
}

func TestSubmitValidAndIdempotent(t *testing.T) {
	s := New()
	accepted, err := s.Submit(validResult())
	if err != nil || !accepted {
		t.Fatalf("accepted=%v err=%v", accepted, err)
	}
	accepted, err = s.Submit(validResult())
	if err != nil || accepted {
		t.Fatalf("second accepted=%v err=%v", accepted, err)
	}
	if s.Len() != 1 {
		t.Fatalf("len = %d", s.Len())
	}
}

func TestSubmitConflict(t *testing.T) {
	s := New()
	if _, err := s.Submit(validResult()); err != nil {
		t.Fatal(err)
	}
	changed := validResult()
	changed.Summary = "different"
	if _, err := s.Submit(changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v", err)
	}
}

func TestSubmitValidation(t *testing.T) {
	s := New()
	missing := validResult()
	missing.ID = ""
	if _, err := s.Submit(missing); err == nil {
		t.Fatal("expected missing id error")
	}
	badStatus := validResult()
	badStatus.ID = "res_2"
	badStatus.Status = "weird"
	if _, err := s.Submit(badStatus); err == nil {
		t.Fatal("expected invalid status error")
	}
	failedNoFailure := validResult()
	failedNoFailure.ID = "res_3"
	failedNoFailure.Status = "failed"
	if _, err := s.Submit(failedNoFailure); err == nil {
		t.Fatal("expected failure requirement")
	}
}
