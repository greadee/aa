// Package syncplane composes aa-sync for the control-plane daemon.
//
// v1 is single-node (D6): the daemon owns one local identity, serves the
// local node in the /nodes read model, and reports sync status. Distribution
// and revision reconciliation are composed over the transport seam so a later
// multi-node phase can enable them without re-plumbing the daemon. The plane
// moves work only; it never grants execution authority and never exposes a
// remote shell.
package syncplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/distribute"
	"github.com/greadee/aa/sync/identity"
	"github.com/greadee/aa/sync/queue"
	"github.com/greadee/aa/sync/revision"
	"github.com/greadee/aa/sync/rpc"
	"github.com/greadee/aa/sync/transport"
)

// Config configures the composed sync plane.
type Config struct {
	// Name is the local node's display name.
	Name string
	// Seed derives the local Ed25519 identity. Production persists a random
	// seed; tests pass a fixed seed for determinism.
	Seed string
	// Clock stamps receipts. Defaults to the deterministic sync clock.
	Clock aasync.Clock
	// Transport carries peer sessions. Defaults to the in-memory wire used for
	// the single-node control plane.
	Transport transport.Transport
}

// Node is a node known to the local plane.
type Node struct {
	ID    aasync.NodeID `json:"id"`
	Name  string        `json:"name,omitempty"`
	Local bool          `json:"local"`
	State string        `json:"state"`
}

// Status is the sync read model served to the console.
type Status struct {
	LocalNode aasync.NodeID    `json:"localNode"`
	Nodes     []Node           `json:"nodes"`
	Transfers []aasync.Receipt `json:"transfers"`
}

// Reconciliation is the composed offline-reconcile outcome.
type Reconciliation struct {
	Decision queue.Decision
	Applied  revision.Result
}

// Plane is the composed sync service.
type Plane struct {
	local       identity.Node
	distributor *distribute.Distributor
	peer        *distribute.Peer
	rpc         *rpc.Service

	mu    sync.Mutex
	peers map[aasync.NodeID]Node

	revMu   sync.Mutex
	source  *revision.Set
	replica *revision.Set
	pending *queue.Queue
}

// New composes a sync plane. It registers the local identity on the transport
// and creates the distributor, peer endpoint, and RPC adapter over it.
func New(cfg Config) *Plane {
	if cfg.Name == "" {
		cfg.Name = "local"
	}
	if cfg.Clock == nil {
		cfg.Clock = aasync.NewFixedClock()
	}
	if cfg.Transport == nil {
		cfg.Transport = transport.NewWire()
	}
	local := identity.NewNode(identity.Generate(cfg.Seed), cfg.Name)
	if wire, ok := cfg.Transport.(*transport.Wire); ok {
		wire.Register(local.ID)
	}
	distributor := distribute.New(local.ID, cfg.Transport, cfg.Clock)
	return &Plane{
		local:       local,
		distributor: distributor,
		peer:        distribute.NewPeer(local.ID, cfg.Transport),
		rpc:         rpc.New(distributor),
		peers:       map[aasync.NodeID]Node{},
		source:      revision.NewSet(),
		replica:     revision.NewSet(),
		pending:     queue.New(),
	}
}

// LocalID returns the local node's stable identity.
func (p *Plane) LocalID() aasync.NodeID { return p.local.ID }

// LocalName returns the local node's display name.
func (p *Plane) LocalName() string { return p.local.Name }

// AddPeer records a known remote node. Registration is explicit and local; it
// is not authentication. Pairing and mutual authentication stay out of the
// single-node v1 scope.
func (p *Plane) AddPeer(id aasync.NodeID, name string) error {
	if id == "" {
		return fmt.Errorf("%w: peer id is required", aasync.ErrInvalid)
	}
	if id == p.local.ID {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers[id] = Node{ID: id, Name: name, State: "declared"}
	return nil
}

// Nodes returns the local node followed by known peers, ordered by ID. The
// local node is the only authority node.
func (p *Plane) Nodes() []Node {
	p.mu.Lock()
	peers := make([]Node, 0, len(p.peers))
	for _, n := range p.peers {
		peers = append(peers, n)
	}
	p.mu.Unlock()
	sort.Slice(peers, func(i, j int) bool { return peers[i].ID < peers[j].ID })

	out := make([]Node, 0, len(peers)+1)
	out = append(out, Node{ID: p.local.ID, Name: p.local.Name, Local: true, State: "ready"})
	return append(out, peers...)
}

// Status reports the local identity, the node registry, and completed
// transfers. It dispatches through the sync RPC adapter so the daemon exercises
// the same path a remote caller would.
func (p *Plane) Status(ctx context.Context) Status {
	st := Status{LocalNode: p.local.ID, Nodes: p.Nodes()}
	resp := p.rpc.Handle(ctx, rpc.Request{JSONRPC: "2.0", ID: "status", Method: rpc.MethodStatus})
	if resp.Error != nil {
		return st
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		return st
	}
	if receipts, ok := result["transfers"].([]aasync.Receipt); ok {
		st.Transfers = receipts
	}
	return st
}

// Handle dispatches one sync RPC request through the composed adapter.
func (p *Plane) Handle(ctx context.Context, req rpc.Request) rpc.Response {
	return p.rpc.Handle(ctx, req)
}

// Distribute sends a work package to a known peer. The package carries a
// payload only; it grants no execution authority.
func (p *Plane) Distribute(ctx context.Context, wp aasync.WorkPackage, remote aasync.NodeID) (aasync.Receipt, error) {
	p.mu.Lock()
	_, known := p.peers[remote]
	p.mu.Unlock()
	if !known {
		return aasync.Receipt{}, fmt.Errorf("%w: peer %s", aasync.ErrNotFound, remote)
	}
	return p.distributor.Distribute(ctx, wp, remote)
}

// Serve accepts one incoming distribution session from a peer.
func (p *Plane) Serve(ctx context.Context, from aasync.NodeID) error {
	return p.peer.Serve(ctx, from)
}

// ReceivedPackages returns the work packages this node has received.
func (p *Plane) ReceivedPackages() []aasync.WorkPackage {
	return p.peer.Packages()
}

// Publish records a local revision and queues it for offline reconciliation.
func (p *Plane) Publish(rec aasync.RevisionRecord) error {
	p.revMu.Lock()
	defer p.revMu.Unlock()
	if err := p.source.Put(rec); err != nil {
		return err
	}
	return p.pending.Enqueue(aasync.Change{Record: rec, Node: p.local.ID})
}

// Pending returns the queued local changes ordered by path.
func (p *Plane) Pending() []aasync.Change {
	p.revMu.Lock()
	defer p.revMu.Unlock()
	return p.pending.Pending()
}

// Reconcile merges remote offline changes with the local queue and applies the
// authoritative side to the local replica. Ordering derives from record
// revisions, never the wall clock; deletions stay guard-protected.
func (p *Plane) Reconcile(remote []aasync.Change) (Reconciliation, error) {
	p.revMu.Lock()
	defer p.revMu.Unlock()
	decision, err := queue.Reconcile(p.pending.Pending(), remote)
	if err != nil {
		return Reconciliation{}, err
	}
	source := revision.NewSet()
	for _, change := range decision.Apply {
		if err := source.Put(change.Record); err != nil {
			return Reconciliation{}, err
		}
	}
	applied, err := revision.Apply(p.replica, revision.Diff(source, p.replica))
	if err != nil {
		return Reconciliation{}, err
	}
	return Reconciliation{Decision: decision, Applied: applied}, nil
}

// ReplicaDigest returns a stable digest of the local replica.
func (p *Plane) ReplicaDigest() string {
	p.revMu.Lock()
	defer p.revMu.Unlock()
	return p.replica.Digest()
}

// SeedFromRoot loads the local identity seed from dir, creating a random
// 0600 seed file on first use so a node keeps one stable identity.
func SeedFromRoot(dir string) (string, error) {
	path := filepath.Join(dir, "node.seed")
	data, err := os.ReadFile(path)
	if err == nil {
		seed := strings.TrimSpace(string(data))
		if seed == "" {
			return "", errors.New("syncplane: empty node seed")
		}
		return seed, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("syncplane: read node seed: %w", err)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("syncplane: generate node seed: %w", err)
	}
	seed := hex.EncodeToString(buf)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("syncplane: create state root: %w", err)
	}
	if err := os.WriteFile(path, []byte(seed+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("syncplane: write node seed: %w", err)
	}
	return seed, nil
}
