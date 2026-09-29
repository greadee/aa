package v2

import "fmt"

// ModelSpec is a durable specification of an available inference configuration,
// independent of roles. It is a named registry asset, not an execution backend
// identity. See the model registry (aa-registry).
type ModelSpec struct {
	Envelope
	Provider       string   `json:"provider"`
	Identifier     string   `json:"identifier"`
	ModelVersion   string   `json:"modelVersion,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	ContextWindow  int      `json:"contextWindow,omitempty"`
	ToolUse        bool     `json:"toolUse,omitempty"`
	Locality       string   `json:"locality,omitempty"`
	Available      bool     `json:"available,omitempty"`
	CostPer1KInput *float64 `json:"costPer1kInput,omitempty"`
	CostPer1KOut   *float64 `json:"costPer1kOutput,omitempty"`
	LatencyMS      int      `json:"latencyMs,omitempty"`
}

// Validate checks the model-spec contract.
func (m ModelSpec) Validate() error {
	if m.Kind != "model_spec" {
		return fmt.Errorf("kind: expected %q, got %q", "model_spec", m.Kind)
	}
	if err := m.Envelope.Validate(); err != nil {
		return err
	}
	if m.Provider == "" {
		return fmt.Errorf("provider: is required")
	}
	if m.Identifier == "" {
		return fmt.Errorf("identifier: is required")
	}
	if m.Locality != "" {
		if err := RequireEnum("locality", m.Locality, "local", "cloud", "either"); err != nil {
			return err
		}
	}
	return nil
}
