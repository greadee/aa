package v1

import "fmt"

// SandboxSpec declares a tool's isolation requirements.
type SandboxSpec struct {
	Required bool   `json:"required,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Network  bool   `json:"network,omitempty"`
}

// ToolManifest declares a tool, plugin, or MCP server.
type ToolManifest struct {
	Envelope
	Name         string         `json:"name"`
	ToolKind     string         `json:"toolKind"`
	Version      string         `json:"version"`
	Description  string         `json:"description,omitempty"`
	Capabilities []string       `json:"capabilities"`
	Permissions  []string       `json:"permissions,omitempty"`
	InputSchema  map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	Sandbox      *SandboxSpec   `json:"sandbox,omitempty"`
	Endpoint     string         `json:"endpoint,omitempty"`
}

// Validate checks the tool-manifest contract.
func (t ToolManifest) Validate() error {
	if t.Kind != "tool_manifest" {
		return fmt.Errorf("kind: expected %q, got %q", "tool_manifest", t.Kind)
	}
	if err := t.Envelope.Validate(); err != nil {
		return err
	}
	if t.Name == "" {
		return fmt.Errorf("name: is required")
	}
	if err := RequireEnum("toolKind", t.ToolKind, "tool", "plugin", "mcp", "builtin"); err != nil {
		return err
	}
	if t.Version == "" {
		return fmt.Errorf("version: is required")
	}
	if len(t.Capabilities) == 0 {
		return fmt.Errorf("capabilities: at least one is required")
	}
	return nil
}
