package experience

import (
	"errors"
	"testing"

	"github.com/greadee/aa/kernel/joblearn"
)

func verifiedObservation() Observation {
	return Observation{
		ProjectID:     "prj_1",
		WorkPackageID: "wp_1",
		AttemptID:     "att_1",
		Role:          "Builder",
		Model:         "deepseek-reasoner",
		Outcome:       joblearn.OutcomeSucceeded,
		Verified:      true,
		Confidence:    0.9,
		Evidence:      []joblearn.Reference{{Kind: "trace", ID: "trc_1"}},
		Lessons:       []Lesson{{Title: "Batch the writes", Content: "flush once per attempt"}},
	}
}

func TestDeriveProducesProjectLessonsWithProvenance(t *testing.T) {
	feedback, err := Derive(DefaultPolicy(), verifiedObservation())
	if err != nil {
		t.Fatal(err)
	}
	if feedback.State != Apprenticing || len(feedback.Candidates) != 1 {
		t.Fatalf("feedback = %+v", feedback)
	}
	candidate := feedback.Candidates[0]
	if candidate.Level != joblearn.LevelProject || candidate.Scope != "project:prj_1" {
		t.Fatalf("candidate = %+v", candidate)
	}
	if candidate.Provenance == nil || candidate.Provenance.Source == "" {
		t.Fatalf("missing provenance: %+v", candidate.Provenance)
	}
	// The work package and attempt are always in evidence.
	ids := map[string]bool{}
	for _, ref := range candidate.Provenance.Evidence {
		ids[ref.Kind+":"+ref.ID] = true
	}
	if !ids["work_package:wp_1"] || !ids["attempt:att_1"] {
		t.Fatalf("provenance evidence = %+v", candidate.Provenance.Evidence)
	}
	if feedback.Mentorship == nil || feedback.Mentorship.Teacher != "deepseek-reasoner" {
		t.Fatalf("mentorship = %+v", feedback.Mentorship)
	}
}

func TestDeriveRefusesContaminatedLearning(t *testing.T) {
	cases := map[string]func(*Observation){
		"unverified":     func(o *Observation) { o.Verified = false },
		"failed":         func(o *Observation) { o.Outcome = joblearn.OutcomeFailed },
		"low confidence": func(o *Observation) { o.Confidence = 0.1 },
		"no evidence":    func(o *Observation) { o.Evidence = nil },
	}
	for name, mutate := range cases {
		obs := verifiedObservation()
		mutate(&obs)
		feedback, err := Derive(DefaultPolicy(), obs)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(feedback.Candidates) != 0 || feedback.Reason == "" {
			t.Fatalf("%s: feedback = %+v", name, feedback)
		}
	}
}

func TestDeriveRejectsInvalidObservation(t *testing.T) {
	obs := verifiedObservation()
	obs.AttemptID = ""
	if _, err := Derive(DefaultPolicy(), obs); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestGraduateToOrganizationScope(t *testing.T) {
	feedback, _ := Derive(DefaultPolicy(), verifiedObservation())
	graduated, err := Graduate(feedback.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	if graduated.Level != joblearn.LevelWorkforce || graduated.Scope != "organization" {
		t.Fatalf("graduated = %+v", graduated)
	}
	if State(graduated.Level) != Studying || State(joblearn.LevelProject) != Apprenticing {
		t.Fatalf("state mapping wrong")
	}
	if _, err := Graduate(graduated); !errors.Is(err, joblearn.ErrInvalid) {
		t.Fatalf("expected error graduating a non-project lesson, got %v", err)
	}
}

func TestRecordIntoMemStore(t *testing.T) {
	feedback, _ := Derive(DefaultPolicy(), verifiedObservation())
	store := NewMemStore()
	written, err := Record(store, feedback)
	if err != nil || written != 1 {
		t.Fatalf("written=%d err=%v", written, err)
	}
	// Re-recording is idempotent.
	if _, err := Record(store, feedback); err != nil {
		t.Fatal(err)
	}
	if got := store.List("project:prj_1"); len(got) != 1 {
		t.Fatalf("store = %+v", got)
	}
	if got := store.List("organization"); len(got) != 0 {
		t.Fatalf("scope filter failed: %+v", got)
	}
}
