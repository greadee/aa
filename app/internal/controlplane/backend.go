package controlplane

import "encoding/json"

// Project is a project summary.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// WorkPackageView is a work-package summary.
type WorkPackageView struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	State     string   `json:"state,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

// AssignmentView is an assignment summary.
type AssignmentView struct {
	ID            string `json:"id"`
	WorkPackageID string `json:"workPackageId"`
	WorkerID      string `json:"workerId"`
	State         string `json:"state"`
	Attempt       int    `json:"attempt"`
}

// Node is a compute/sync node summary.
type Node struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Authority bool   `json:"authority"`
	State     string `json:"state"`
}

// SyncNode is one node in the sync read model.
type SyncNode struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Local bool   `json:"local"`
	State string `json:"state"`
}

// SyncTransfer is a completed sync transfer.
type SyncTransfer struct {
	TransferID    string `json:"transferId"`
	NodeID        string `json:"nodeId,omitempty"`
	WorkPackageID string `json:"workPackageId,omitempty"`
	ArtifactID    string `json:"artifactId,omitempty"`
	State         string `json:"state"`
	Hash          string `json:"hash,omitempty"`
	Bytes         int64  `json:"bytes"`
	CreatedAt     string `json:"createdAt,omitempty"`
}

// SyncStatus is the sync read model served on /sync/status.
type SyncStatus struct {
	LocalNode string         `json:"localNode"`
	Nodes     []SyncNode     `json:"nodes"`
	Transfers []SyncTransfer `json:"transfers"`
}

// Tool is a registered tool/plugin manifest summary.
type Tool struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Sandbox bool   `json:"sandbox,omitempty"`
}

// Backend supplies the control plane's resources. It is implemented by the
// composition root (app/), which composes memory and the kernel scheduler; the
// transport never reaches into modules itself.
type Backend interface {
	Projects() []Project
	CreateProject(id, name string) error
	WorkPackages(projectID string) []WorkPackageView
	SubmitWork(projectID, title string, dependsOn []string) (string, error)
	Assignments() []AssignmentView
	Approve(gateID string) error
	Issues(projectID string) []json.RawMessage
	Strategies(projectID string) []json.RawMessage
	Events(projectID string) []json.RawMessage
	Nodes() []Node
	SyncStatus() SyncStatus
	Tools() []Tool
}
