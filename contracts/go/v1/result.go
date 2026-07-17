package v1

import "fmt"

// ResultArtifact is an artifact produced by an attempt.
type ResultArtifact struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Hash string `json:"hash,omitempty"`
	URI  string `json:"uri,omitempty"`
}

// TestResult is a single test outcome reported by an attempt.
type TestResult struct {
	Name       string `json:"name"`
	Outcome    string `json:"outcome"`
	DurationMS *int   `json:"durationMs,omitempty"`
}

// Failure describes why an attempt failed.
type Failure struct {
	Class     string `json:"class,omitempty"`
	Message   string `json:"message,omitempty"`
	Retryable *bool  `json:"retryable,omitempty"`
}

// ResultEnvelope is the untrusted structured output of an attempt.
type ResultEnvelope struct {
	Envelope
	AttemptID     string           `json:"attemptId"`
	AssignmentID  string           `json:"assignmentId"`
	WorkPackageID string           `json:"workPackageId"`
	Status        string           `json:"status"`
	Summary       string           `json:"summary,omitempty"`
	Artifacts     []ResultArtifact `json:"artifacts,omitempty"`
	Tests         []TestResult     `json:"tests,omitempty"`
	TelemetryRef  *Reference       `json:"telemetryRef,omitempty"`
	Failure       *Failure         `json:"failure,omitempty"`
	ProducedBy    *Actor           `json:"producedBy,omitempty"`
}

// Validate checks the result-envelope contract.
func (r ResultEnvelope) Validate() error {
	if r.Kind != "result_envelope" {
		return fmt.Errorf("kind: expected %q, got %q", "result_envelope", r.Kind)
	}
	if err := r.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireIdentifier("attemptId", r.AttemptID); err != nil {
		return err
	}
	if err := RequireIdentifier("assignmentId", r.AssignmentID); err != nil {
		return err
	}
	if err := RequireIdentifier("workPackageId", r.WorkPackageID); err != nil {
		return err
	}
	if err := RequireEnum("status", r.Status, "succeeded", "failed", "partial", "blocked", "cancelled"); err != nil {
		return err
	}
	for i, a := range r.Artifacts {
		if err := RequireIdentifier(fmt.Sprintf("artifacts[%d].kind", i), a.Kind); err != nil {
			return err
		}
		if err := RequireIdentifier(fmt.Sprintf("artifacts[%d].id", i), a.ID); err != nil {
			return err
		}
		if a.Hash != "" {
			if err := RequireHash(fmt.Sprintf("artifacts[%d].hash", i), a.Hash); err != nil {
				return err
			}
		}
	}
	for i, t := range r.Tests {
		if t.Name == "" {
			return fmt.Errorf("tests[%d].name: is required", i)
		}
		if err := RequireEnum(fmt.Sprintf("tests[%d].outcome", i), t.Outcome, "passed", "failed", "skipped", "error"); err != nil {
			return err
		}
	}
	if r.TelemetryRef != nil {
		if err := r.TelemetryRef.Validate(); err != nil {
			return err
		}
	}
	if r.ProducedBy != nil {
		return r.ProducedBy.Validate()
	}
	return nil
}
