package v1

import "fmt"

// TracePhase labels the kind of work a trace step represents.
type TracePhase string

// Trace phases.
const (
	TraceObserve   TracePhase = "observe"
	TracePlan      TracePhase = "plan"
	TraceModel     TracePhase = "model"
	TraceTool      TracePhase = "tool"
	TraceEdit      TracePhase = "edit"
	TraceTest      TracePhase = "test"
	TraceGate      TracePhase = "gate"
	TraceVerify    TracePhase = "verify"
	TraceIntegrate TracePhase = "integrate"
)

// TraceOutcome is the result of a trace step.
type TraceOutcome string

// Trace step outcomes.
const (
	TraceSucceeded TraceOutcome = "succeeded"
	TraceFailed    TraceOutcome = "failed"
	TraceSkipped   TraceOutcome = "skipped"
	TraceBlocked   TraceOutcome = "blocked"
)

// TraceStep is one bounded, redacted unit of work evidence. Content-bearing
// fields are withheld or hashed; hashes exist for deduplication and correlation
// only and never carry recoverable input.
type TraceStep struct {
	Sequence   *int         `json:"sequence"`
	At         string       `json:"at,omitempty"`
	Phase      TracePhase   `json:"phase"`
	Actor      *Actor       `json:"actor,omitempty"`
	Operation  string       `json:"operation,omitempty"`
	Target     *Reference   `json:"target,omitempty"`
	InputHash  string       `json:"inputHash,omitempty"`
	OutputHash string       `json:"outputHash,omitempty"`
	Outcome    TraceOutcome `json:"outcome"`
	ErrorClass string       `json:"errorClass,omitempty"`
	DurationMS *int         `json:"durationMs,omitempty"`
	Tokens     *TokenCounts `json:"tokens,omitempty"`
	CostUSD    *float64     `json:"costUsd,omitempty"`
	Redacted   *bool        `json:"redacted,omitempty"`
	Evidence   []Reference  `json:"evidence,omitempty"`
}

// Trace is bounded, redacted per-step evidence for one attempt. It is derived
// operational evidence, not canonical history and not the observation protocol.
type Trace struct {
	Envelope
	AttemptID        string      `json:"attemptId"`
	AssignmentID     string      `json:"assignmentId,omitempty"`
	WorkPackageID    string      `json:"workPackageId,omitempty"`
	RedactionVersion string      `json:"redactionVersion,omitempty"`
	Truncated        *bool       `json:"truncated,omitempty"`
	Steps            []TraceStep `json:"steps"`
}

// Validate checks a trace step.
func (s TraceStep) Validate() error {
	if s.Sequence == nil || *s.Sequence < 0 {
		return fmt.Errorf("sequence: is required and must be >= 0")
	}
	if err := RequireEnum("phase", string(s.Phase),
		"observe", "plan", "model", "tool", "edit", "test", "gate", "verify", "integrate"); err != nil {
		return err
	}
	if err := RequireEnum("outcome", string(s.Outcome),
		"succeeded", "failed", "skipped", "blocked"); err != nil {
		return err
	}
	if err := OptionalTimestamp("at", s.At); err != nil {
		return err
	}
	if s.InputHash != "" {
		if err := RequireHash("inputHash", s.InputHash); err != nil {
			return err
		}
	}
	if s.OutputHash != "" {
		if err := RequireHash("outputHash", s.OutputHash); err != nil {
			return err
		}
	}
	if s.Actor != nil {
		if err := s.Actor.Validate(); err != nil {
			return fmt.Errorf("actor: %w", err)
		}
	}
	if s.Target != nil {
		if err := s.Target.Validate(); err != nil {
			return fmt.Errorf("target: %w", err)
		}
	}
	for i, e := range s.Evidence {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("evidence[%d]: %w", i, err)
		}
	}
	return nil
}

// Validate checks the trace contract.
func (t Trace) Validate() error {
	if t.Kind != "trace" {
		return fmt.Errorf("kind: expected %q, got %q", "trace", t.Kind)
	}
	if err := t.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireIdentifier("attemptId", t.AttemptID); err != nil {
		return err
	}
	if err := OptionalIdentifier("assignmentId", t.AssignmentID); err != nil {
		return err
	}
	if err := OptionalIdentifier("workPackageId", t.WorkPackageID); err != nil {
		return err
	}
	if len(t.Steps) == 0 {
		return fmt.Errorf("steps: at least one step is required")
	}
	for i, s := range t.Steps {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("steps[%d]: %w", i, err)
		}
	}
	return nil
}
