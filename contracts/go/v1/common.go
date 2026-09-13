// Package v1 contains the Go bindings for the aa contract schemas v1.
//
// The JSON Schema files under contracts/schemas/v1 are the source of truth;
// these types mirror them. The conformance suite guards against drift.
package v1

// Version is the contract MAJOR.MINOR version for this package.
const Version = "1.0"

// Reference points at another contract object.
type Reference struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

// Actor identifies who produced or performed something.
type Actor struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Role     string `json:"role,omitempty"`
	Trade    string `json:"trade,omitempty"`
	WorkerID string `json:"workerId,omitempty"`
	Model    string `json:"model,omitempty"`
}

// Provenance records where a record came from.
type Provenance struct {
	Source      string      `json:"source"`
	ProducedAt  string      `json:"producedAt"`
	ProducedBy  *Actor      `json:"producedBy,omitempty"`
	Confidence  string      `json:"confidence,omitempty"`
	Evidence    []Reference `json:"evidence,omitempty"`
	ContentHash string      `json:"contentHash,omitempty"`
}

// Envelope is the shared header for contract objects.
type Envelope struct {
	ContractVersion string      `json:"contractVersion"`
	Kind            string      `json:"kind"`
	ID              string      `json:"id"`
	ProjectID       string      `json:"projectId,omitempty"`
	CreatedAt       string      `json:"createdAt,omitempty"`
	Revision        *int        `json:"revision,omitempty"`
	Provenance      *Provenance `json:"provenance,omitempty"`
}

// Budget bounds an attempt or workflow.
type Budget struct {
	MaxCalls           *int     `json:"maxCalls,omitempty"`
	MaxTokens          *int     `json:"maxTokens,omitempty"`
	MaxCostUSD         *float64 `json:"maxCostUsd,omitempty"`
	MaxDurationSeconds *int     `json:"maxDurationSeconds,omitempty"`
}

// ProjectState is the lifecycle state of a project.
type ProjectState string

// Project states.
const (
	ProjectCreated   ProjectState = "CREATED"
	ProjectDiscovery ProjectState = "DISCOVERY"
	ProjectPlanning  ProjectState = "PLANNING"
	ProjectExecuting ProjectState = "EXECUTING"
	ProjectVerifying ProjectState = "VERIFYING"
	ProjectComplete  ProjectState = "COMPLETE"
	ProjectBlocked   ProjectState = "BLOCKED"
	ProjectFailed    ProjectState = "FAILED"
	ProjectCancelled ProjectState = "CANCELLED"
	ProjectPaused    ProjectState = "PAUSED"
)

// WorkPackageState is the lifecycle state of a work package.
type WorkPackageState string

// Work package states.
const (
	WorkPackageProposed  WorkPackageState = "PROPOSED"
	WorkPackageReady     WorkPackageState = "READY"
	WorkPackageQueued    WorkPackageState = "QUEUED"
	WorkPackageRunning   WorkPackageState = "RUNNING"
	WorkPackageComplete  WorkPackageState = "COMPLETE"
	WorkPackageVerifying WorkPackageState = "VERIFYING"
	WorkPackageVerified  WorkPackageState = "VERIFIED"
	WorkPackageDeficient WorkPackageState = "DEFICIENT"
)

// AssignmentState is the lifecycle state of an assignment.
type AssignmentState string

// Assignment states.
const (
	AssignmentPlanned      AssignmentState = "planned"
	AssignmentLeased       AssignmentState = "leased"
	AssignmentPreparing    AssignmentState = "preparing"
	AssignmentRunning      AssignmentState = "running"
	AssignmentPaused       AssignmentState = "paused"
	AssignmentCollecting   AssignmentState = "collecting"
	AssignmentAwaitingGate AssignmentState = "awaiting_gates"
	AssignmentAccepted     AssignmentState = "accepted"
	AssignmentFailed       AssignmentState = "failed"
	AssignmentCanceled     AssignmentState = "canceled"
	AssignmentExpired      AssignmentState = "expired"
)

// ReadinessState is the dispatch readiness of a work package.
type ReadinessState string

// Readiness states.
const (
	ReadinessBlocked    ReadinessState = "blocked"
	ReadinessReady      ReadinessState = "ready"
	ReadinessDispatched ReadinessState = "dispatched"
	ReadinessSatisfied  ReadinessState = "satisfied"
)

// IssueState is the lifecycle state of an issue.
type IssueState string

// Issue states.
const (
	IssueOpen       IssueState = "open"
	IssueInProgress IssueState = "in_progress"
	IssueReview     IssueState = "review"
	IssueClosed     IssueState = "closed"
	IssueDeferred   IssueState = "deferred"
	IssueCancelled  IssueState = "cancelled"
)

// MemoryLifecycle is the promotion lifecycle of a memory record.
type MemoryLifecycle string

// Memory lifecycles.
const (
	MemoryEphemeral  MemoryLifecycle = "EPHEMERAL"
	MemoryCandidate  MemoryLifecycle = "CANDIDATE"
	MemoryValidated  MemoryLifecycle = "VALIDATED"
	MemoryActive     MemoryLifecycle = "ACTIVE"
	MemorySuperseded MemoryLifecycle = "SUPERSEDED"
	MemoryArchived   MemoryLifecycle = "ARCHIVED"
)
