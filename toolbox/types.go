package toolbox

import (
	"crypto/sha256"
	"encoding/hex"

	v2 "github.com/greadee/aa/contracts/go/v2"
)

// Cross-module data structures live only in contracts; toolbox re-exports the
// ones it uses so callers can stay within one package boundary.
type (
	// Envelope is the shared header for contract objects.
	Envelope = v2.Envelope
	// Manifest declares a tool, plugin, or MCP server.
	Manifest = v2.ToolManifest
	// SandboxSpec declares a tool's isolation requirements.
	SandboxSpec = v2.SandboxSpec
	// Workflow is a declarative, resumable process definition.
	Workflow = v2.Workflow
	// WorkflowStep is one step in a workflow.
	WorkflowStep = v2.WorkflowStep
	// Capability is a granted authority in an execution contract.
	Capability = v2.Capability
	// ExecutionContract is the immutable, least-privilege authority for an attempt.
	ExecutionContract = v2.ExecutionContract
	// Budget bounds an attempt or workflow.
	Budget = v2.Budget
)

// ToolID identifies a registered tool, plugin, or MCP server.
type ToolID string

// Capability vocabulary, mirrored from the execution-contract contract.
const (
	CapReadProject         = v2.CapReadProject
	CapWriteWorkspace      = v2.CapWriteWorkspace
	CapExecuteCommand      = v2.CapExecuteCommand
	CapRunTests            = v2.CapRunTests
	CapNetworkAccess       = v2.CapNetworkAccess
	CapInstallDependencies = v2.CapInstallDependencies
	CapCallModel           = v2.CapCallModel
	CapReadSecrets         = v2.CapReadSecrets
	CapCreateArtifact      = v2.CapCreateArtifact
	CapOpenPullRequest     = v2.CapOpenPullRequest
	CapMerge               = v2.CapMerge
	CapDeploy              = v2.CapDeploy
)

// Tool kinds mirror the tool_manifest contract.
const (
	KindTool    = "tool"
	KindPlugin  = "plugin"
	KindMCP     = "mcp"
	KindBuiltin = "builtin"
)

// Invocation is one request to run a registered tool.
type Invocation struct {
	ToolID              ToolID         `json:"toolId"`
	Input               map[string]any `json:"input,omitempty"`
	ExecutionContractID string         `json:"executionContractId,omitempty"`
	IdempotencyKey      string         `json:"idempotencyKey,omitempty"`
}

// Result is one invocation outcome. Replayed is true when the result was
// served from the idempotency cache rather than re-executed.
type Result struct {
	ToolID   ToolID         `json:"toolId"`
	Output   map[string]any `json:"output,omitempty"`
	Replayed bool           `json:"replayed,omitempty"`
}

// Hash returns the hex-encoded SHA-256 of data.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HashString returns the hex-encoded SHA-256 of s.
func HashString(s string) string {
	return Hash([]byte(s))
}
