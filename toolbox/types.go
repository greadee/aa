package toolbox

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/greadee/aa/contracts/go/v1"
)

// Cross-module data structures live only in contracts; toolbox re-exports the
// ones it uses so callers can stay within one package boundary.
type (
	// Envelope is the shared header for contract objects.
	Envelope = v1.Envelope
	// Manifest declares a tool, plugin, or MCP server.
	Manifest = v1.ToolManifest
	// SandboxSpec declares a tool's isolation requirements.
	SandboxSpec = v1.SandboxSpec
	// Workflow is a declarative, resumable process definition.
	Workflow = v1.Workflow
	// WorkflowStep is one step in a workflow.
	WorkflowStep = v1.WorkflowStep
	// Capability is a granted authority in an execution contract.
	Capability = v1.Capability
	// ExecutionContract is the immutable, least-privilege authority for an attempt.
	ExecutionContract = v1.ExecutionContract
	// Budget bounds an attempt or workflow.
	Budget = v1.Budget
)

// ToolID identifies a registered tool, plugin, or MCP server.
type ToolID string

// Capability vocabulary, mirrored from the execution-contract contract.
const (
	CapReadProject         = v1.CapReadProject
	CapWriteWorkspace      = v1.CapWriteWorkspace
	CapExecuteCommand      = v1.CapExecuteCommand
	CapRunTests            = v1.CapRunTests
	CapNetworkAccess       = v1.CapNetworkAccess
	CapInstallDependencies = v1.CapInstallDependencies
	CapCallModel           = v1.CapCallModel
	CapReadSecrets         = v1.CapReadSecrets
	CapCreateArtifact      = v1.CapCreateArtifact
	CapOpenPullRequest     = v1.CapOpenPullRequest
	CapMerge               = v1.CapMerge
	CapDeploy              = v1.CapDeploy
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
