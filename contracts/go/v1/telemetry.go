package v1

import "fmt"

// CallCounts summarizes model calls made by an attempt.
type CallCounts struct {
	Total   *int `json:"total,omitempty"`
	Local   *int `json:"local,omitempty"`
	Cloud   *int `json:"cloud,omitempty"`
	Retries *int `json:"retries,omitempty"`
}

// TokenCounts summarizes token usage.
type TokenCounts struct {
	Input  *int `json:"input,omitempty"`
	Output *int `json:"output,omitempty"`
}

// Telemetry is bounded execution evidence. It is operational, not canonical.
type Telemetry struct {
	Envelope
	AttemptID     string            `json:"attemptId"`
	AssignmentID  string            `json:"assignmentId,omitempty"`
	MetricVersion string            `json:"metricVersion"`
	Outcome       string            `json:"outcome"`
	Calls         *CallCounts       `json:"calls,omitempty"`
	Tokens        *TokenCounts      `json:"tokens,omitempty"`
	CostUSD       *float64          `json:"costUsd,omitempty"`
	DurationMS    *int              `json:"durationMs,omitempty"`
	Resources     map[string]any    `json:"resources,omitempty"`
	Versions      map[string]string `json:"versions,omitempty"`
}

// Validate checks the telemetry contract.
func (t Telemetry) Validate() error {
	if t.Kind != "telemetry" {
		return fmt.Errorf("kind: expected %q, got %q", "telemetry", t.Kind)
	}
	if err := t.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireIdentifier("attemptId", t.AttemptID); err != nil {
		return err
	}
	if t.MetricVersion == "" {
		return fmt.Errorf("metricVersion: is required")
	}
	return RequireEnum("outcome", t.Outcome,
		"succeeded", "failed", "partial", "blocked", "cancelled", "unknown")
}
