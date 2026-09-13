package v1

import "fmt"

// Capability is a granted authority in an execution contract.
type Capability string

// Capability vocabulary.
const (
	CapReadProject         Capability = "read_project"
	CapWriteWorkspace      Capability = "write_workspace"
	CapExecuteCommand      Capability = "execute_command"
	CapRunTests            Capability = "run_tests"
	CapNetworkAccess       Capability = "network_access"
	CapInstallDependencies Capability = "install_dependencies"
	CapCallModel           Capability = "call_model"
	CapReadSecrets         Capability = "read_secrets"
	CapCreateArtifact      Capability = "create_artifact"
	CapOpenPullRequest     Capability = "open_pull_request"
	CapMerge               Capability = "merge"
	CapDeploy              Capability = "deploy"
)

var capabilities = map[Capability]bool{
	CapReadProject: true, CapWriteWorkspace: true, CapExecuteCommand: true,
	CapRunTests: true, CapNetworkAccess: true, CapInstallDependencies: true,
	CapCallModel: true, CapReadSecrets: true, CapCreateArtifact: true,
	CapOpenPullRequest: true, CapMerge: true, CapDeploy: true,
}

// RuntimeSpec selects where an attempt runs.
type RuntimeSpec struct {
	Kind     string `json:"kind"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	NodeID   string `json:"nodeId,omitempty"`
}

// WorkspaceSpec selects the isolation for an attempt.
type WorkspaceSpec struct {
	Required bool   `json:"required,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Root     string `json:"root,omitempty"`
}

// ExecutionContract is the immutable, least-privilege authority for an attempt.
type ExecutionContract struct {
	Envelope
	WorkPackageID string         `json:"workPackageId"`
	AssignmentID  string         `json:"assignmentId"`
	Capabilities  []Capability   `json:"capabilities"`
	Denied        []Capability   `json:"denied,omitempty"`
	Runtime       *RuntimeSpec   `json:"runtime,omitempty"`
	Workspace     *WorkspaceSpec `json:"workspace,omitempty"`
	Budget        Budget         `json:"budget"`
	Gates         []string       `json:"gates,omitempty"`
	ExpiresAt     string         `json:"expiresAt,omitempty"`
	Digest        string         `json:"digest,omitempty"`
}

// Validate checks the execution-contract contract.
func (c ExecutionContract) Validate() error {
	if c.Kind != "execution_contract" {
		return fmt.Errorf("kind: expected %q, got %q", "execution_contract", c.Kind)
	}
	if err := c.Envelope.Validate(); err != nil {
		return err
	}
	if err := RequireIdentifier("workPackageId", c.WorkPackageID); err != nil {
		return err
	}
	if err := RequireIdentifier("assignmentId", c.AssignmentID); err != nil {
		return err
	}
	for i, cap := range c.Capabilities {
		if !capabilities[cap] {
			return fmt.Errorf("capabilities[%d]: unknown capability %q", i, cap)
		}
	}
	for i, cap := range c.Denied {
		if !capabilities[cap] {
			return fmt.Errorf("denied[%d]: unknown capability %q", i, cap)
		}
	}
	if err := OptionalTimestamp("expiresAt", c.ExpiresAt); err != nil {
		return err
	}
	if c.Digest != "" {
		return RequireHash("digest", c.Digest)
	}
	return nil
}
