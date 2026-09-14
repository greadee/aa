package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// NodeID is a peer's stable identity: the fingerprint of its public key.
type NodeID string

// Peer is a remote node known to the local node.
type Peer struct {
	ID   NodeID `json:"id"`
	Name string `json:"name,omitempty"`
}

// TransferID identifies one transfer between two nodes.
type TransferID string

// TransferState is the lifecycle state of a transfer.
type TransferState string

// Transfer states.
const (
	TransferPending  TransferState = "pending"
	TransferActive   TransferState = "active"
	TransferComplete TransferState = "complete"
	TransferFailed   TransferState = "failed"
)

// Chunk describes one content-addressed slice of a payload.
type Chunk struct {
	Index  int    `json:"index"`
	Offset int64  `json:"offset"`
	Length int    `json:"length"`
	Hash   string `json:"hash"`
}

// Manifest describes a payload as a sequence of verified chunks.
type Manifest struct {
	ID        string  `json:"id"`
	Name      string  `json:"name,omitempty"`
	Size      int64   `json:"size"`
	ChunkSize int     `json:"chunkSize"`
	Hash      string  `json:"hash"`
	Chunks    []Chunk `json:"chunks"`
}

// WorkPackage is a unit of work moved to another node. It carries a payload
// only; it never carries execution authority.
type WorkPackage struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name,omitempty"`
	Payload   []byte `json:"payload,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// Artifact is a named, content-addressed output moved between nodes.
type Artifact struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
}

// Receipt records the outcome of a distribution or fetch.
type Receipt struct {
	TransferID    TransferID    `json:"transferId"`
	WorkPackageID string        `json:"workPackageId,omitempty"`
	ArtifactID    string        `json:"artifactId,omitempty"`
	NodeID        NodeID        `json:"nodeId"`
	State         TransferState `json:"state"`
	Hash          string        `json:"hash,omitempty"`
	Bytes         int64         `json:"bytes"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// RevisionRecord is one version of a path in a replica. Ordering is by
// Revision, never by wall clock.
type RevisionRecord struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Revision uint64 `json:"revision"`
	Hash     string `json:"hash,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Deleted  bool   `json:"deleted,omitempty"`
}

// Change is a revision observed on a node, queued for reconciliation.
type Change struct {
	Record RevisionRecord `json:"record"`
	Node   NodeID         `json:"node,omitempty"`
}

// OpKind is the kind of a revision operation.
type OpKind string

// Revision operations.
const (
	OpPut    OpKind = "put"
	OpDelete OpKind = "delete"
)

// Op is one application step produced by a revision diff.
type Op struct {
	Kind   OpKind         `json:"kind"`
	Record RevisionRecord `json:"record"`
}

// Hash returns the hex-encoded SHA-256 of data.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
