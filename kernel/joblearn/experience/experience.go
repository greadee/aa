// Package experience turns verified execution results into reusable lessons.
//
// It closes the learning loop: execution → result/evaluation → reusable lesson
// → experience store → future retrieval/context. Derivation is deterministic
// and model-free; lessons are proposals (candidates) that carry provenance and
// remain at project scope until deliberately graduated to organization scope.
// Contamination is controlled by requiring a verified success, sufficient
// confidence, and evidence before any lesson is produced.
package experience

import (
	"fmt"
	"sort"
	"sync"

	"github.com/greadee/aa/kernel/joblearn"
)

// LearningState is a role's learning posture for a scope.
type LearningState string

// Learning states.
const (
	// Apprenticing is project-scope learning: a role learns within one project.
	Apprenticing LearningState = "apprenticing"
	// Studying is organization-scope learning: a lesson is reusable organization-wide.
	Studying LearningState = "studying"
	// Practicing is a graduated role with no active learning posture.
	Practicing LearningState = "practicing"
)

// Lesson is a reusable piece of experience offered by an execution result.
type Lesson struct {
	Kind    joblearn.CandidateKind
	Title   string
	Content string
}

// Observation is one execution result / evaluation feeding learning.
type Observation struct {
	ProjectID     string
	WorkPackageID string
	AttemptID     string
	Role          string
	Model         string
	Outcome       joblearn.Outcome
	Verified      bool
	Confidence    float64
	Evidence      []joblearn.Reference
	Lessons       []Lesson
}

// Mentorship is a selective teaching hint: a stronger model shares one verified
// lesson with cheaper workers. It is per-lesson, never a permanent senior/junior
// role pairing.
type Mentorship struct {
	Teacher string
	Reason  string
	Scope   string
}

// Feedback is the learning outcome of one observation.
type Feedback struct {
	State      LearningState
	Candidates []joblearn.Candidate
	Mentorship *Mentorship
	Reason     string
}

// Policy bounds what may become a lesson (contamination control).
type Policy struct {
	MinConfidence   float64
	RequireVerified bool
	RequireEvidence bool
}

// DefaultPolicy is conservative: a verified success, at least 0.5 confidence,
// and evidence.
func DefaultPolicy() Policy {
	return Policy{MinConfidence: 0.5, RequireVerified: true, RequireEvidence: true}
}

// Derive produces project-scope lessons from a verified execution result. It
// never calls a model and makes no candidate authoritative.
func Derive(policy Policy, obs Observation) (Feedback, error) {
	if obs.ProjectID == "" || obs.WorkPackageID == "" || obs.AttemptID == "" {
		return Feedback{}, fmt.Errorf("%w: projectId, workPackageId, and attemptId are required", joblearn.ErrInvalid)
	}
	if !obs.Outcome.Valid() {
		return Feedback{}, fmt.Errorf("%w: outcome %q is unknown", joblearn.ErrInvalid, obs.Outcome)
	}
	if obs.Confidence < 0 || obs.Confidence > 1 {
		return Feedback{}, fmt.Errorf("%w: confidence %v is outside [0,1]", joblearn.ErrInvalid, obs.Confidence)
	}

	apprenticing := Feedback{State: Apprenticing}
	switch {
	case policy.RequireVerified && !obs.Verified:
		apprenticing.Reason = "outcome not verified; no learning"
		return apprenticing, nil
	case obs.Outcome != joblearn.OutcomeSucceeded:
		apprenticing.Reason = "non-successful outcome; no lesson"
		return apprenticing, nil
	case obs.Confidence < policy.MinConfidence:
		apprenticing.Reason = "confidence below threshold"
		return apprenticing, nil
	case policy.RequireEvidence && len(obs.Evidence) == 0:
		apprenticing.Reason = "no evidence; refusing to learn"
		return apprenticing, nil
	}

	provenance := &joblearn.Provenance{
		Source:   "aa-kernel/joblearn experience",
		Evidence: append(evidence(obs), obs.Evidence...),
	}
	var applicability *joblearn.Applicability
	if obs.Role != "" {
		applicability = &joblearn.Applicability{Roles: []string{obs.Role}}
	}

	candidates := make([]joblearn.Candidate, 0, len(obs.Lessons))
	for _, lesson := range obs.Lessons {
		kind := lesson.Kind
		if kind == "" {
			kind = joblearn.CandidateStrategy
		}
		candidate := joblearn.Candidate{
			Kind:          kind,
			Level:         joblearn.LevelProject,
			Scope:         "project:" + obs.ProjectID,
			Title:         lesson.Title,
			Content:       lesson.Content,
			Applicability: applicability,
			Evidence:      obs.Evidence,
			Provenance:    provenance,
			Confidence:    obs.Confidence,
		}
		if err := candidate.Validate(); err != nil {
			return Feedback{}, err
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Title != candidates[j].Title {
			return candidates[i].Title < candidates[j].Title
		}
		return candidates[i].Content < candidates[j].Content
	})

	feedback := Feedback{State: Apprenticing, Candidates: candidates}
	if len(candidates) > 0 && obs.Model != "" {
		feedback.Mentorship = &Mentorship{
			Teacher: obs.Model,
			Reason:  "share a verified lesson with cheaper workers",
			Scope:   "project:" + obs.ProjectID,
		}
	}
	return feedback, nil
}

// State maps a candidate level to its learning posture.
func State(level joblearn.CandidateLevel) LearningState {
	switch level {
	case joblearn.LevelSession, joblearn.LevelTask, joblearn.LevelProject:
		return Apprenticing
	case joblearn.LevelRole, joblearn.LevelWorkforce:
		return Studying
	default:
		return Practicing
	}
}

// Graduate returns the organization-scope (studying) form of a project lesson,
// retaining its provenance. Promotion stays explicit.
func Graduate(candidate joblearn.Candidate) (joblearn.Candidate, error) {
	if candidate.Level != joblearn.LevelProject {
		return joblearn.Candidate{}, fmt.Errorf("%w: only project lessons can graduate", joblearn.ErrInvalid)
	}
	graduated := candidate
	graduated.Level = joblearn.LevelWorkforce
	graduated.Scope = "organization"
	if err := graduated.Validate(); err != nil {
		return joblearn.Candidate{}, err
	}
	return graduated, nil
}

// Store is the experience-store seam: lessons flow here for future
// retrieval/context. memory implements it in production.
type Store interface {
	Put(candidate joblearn.Candidate) error
	List(scope string) []joblearn.Candidate
}

// Record persists feedback candidates into a store.
func Record(store Store, feedback Feedback) (int, error) {
	if store == nil {
		return 0, fmt.Errorf("%w: store is required", joblearn.ErrInvalid)
	}
	written := 0
	for _, candidate := range feedback.Candidates {
		if err := store.Put(candidate); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// MemStore is an in-memory experience store for tests and pilots.
type MemStore struct {
	mu    sync.Mutex
	items map[string]joblearn.Candidate
}

// NewMemStore returns an empty store.
func NewMemStore() *MemStore {
	return &MemStore{items: map[string]joblearn.Candidate{}}
}

// Put stores a candidate, deduplicating by identity.
func (s *MemStore) Put(candidate joblearn.Candidate) error {
	if err := candidate.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[identity(candidate)] = candidate
	return nil
}

// List returns stored candidates for a scope ("" means all), sorted by title.
func (s *MemStore) List(scope string) []joblearn.Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]joblearn.Candidate, 0, len(s.items))
	for _, candidate := range s.items {
		if scope != "" && candidate.Scope != scope {
			continue
		}
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].Content < out[j].Content
	})
	return out
}

func identity(candidate joblearn.Candidate) string {
	return string(candidate.Kind) + "|" + string(candidate.Level) + "|" + candidate.Scope + "|" + candidate.Title
}

func evidence(obs Observation) []joblearn.Reference {
	return []joblearn.Reference{
		{Kind: "work_package", ID: obs.WorkPackageID},
		{Kind: "attempt", ID: obs.AttemptID},
	}
}
