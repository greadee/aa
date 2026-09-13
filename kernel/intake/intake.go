// Package intake validates untrusted result envelopes and deduplicates them.
// A result is data until the authority accepts it; intake never grants
// authority, it only normalizes, validates, and detects conflicts.
package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/greadee/aa/kernel/runtime"
)

// ErrConflict is returned when a result id is reused with different content.
var ErrConflict = errors.New("intake: result id reused with different content")

// Result is a normalized intake record.
type Result struct {
	ID            string
	AttemptID     string
	AssignmentID  string
	WorkPackageID string
	Status        string
	Summary       string
	Tests         []runtime.TestResult
	Artifacts     []runtime.Artifact
	Failure       *runtime.Failure
	Hash          string
}

// Service validates and deduplicates results.
type Service struct {
	mu   sync.Mutex
	seen map[string]string
}

// New returns an empty intake service.
func New() *Service {
	return &Service{seen: make(map[string]string)}
}

// Submit validates a result and records it. The boolean reports whether the
// result is new. Re-submitting identical content is a no-op.
func (s *Service) Submit(r Result) (bool, error) {
	if r.ID == "" || r.AttemptID == "" || r.AssignmentID == "" || r.WorkPackageID == "" {
		return false, fmt.Errorf("intake: id, attemptId, assignmentId, and workPackageId are required")
	}
	switch r.Status {
	case "succeeded", "failed", "partial", "blocked", "cancelled":
	default:
		return false, fmt.Errorf("intake: invalid status %q", r.Status)
	}
	if (r.Status == "failed" || r.Status == "blocked") && r.Failure == nil {
		return false, fmt.Errorf("intake: status %q requires a failure", r.Status)
	}
	hash, err := hashResult(r)
	if err != nil {
		return false, err
	}
	r.Hash = hash

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.seen[r.ID]; ok {
		if existing != hash {
			return false, fmt.Errorf("%w: %s", ErrConflict, r.ID)
		}
		return false, nil
	}
	s.seen[r.ID] = hash
	return true, nil
}

// Len returns the number of unique results recorded.
func (s *Service) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seen)
}

func hashResult(r Result) (string, error) {
	data, err := json.Marshal(struct {
		Status    string
		Summary   string
		Tests     []runtime.TestResult
		Artifacts []runtime.Artifact
	}{r.Status, r.Summary, r.Tests, r.Artifacts})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
