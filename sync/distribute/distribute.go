// Package distribute moves work packages and artifacts between nodes.
//
// Distribution is idempotent: a repeated call for the same object and node
// returns the original receipt and moves nothing. Transfers ride the
// transport seam, so the whole path is testable offline.
package distribute

import (
	"context"
	"fmt"
	"sort"
	"sync"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/transfer"
	"github.com/greadee/aa/sync/transport"
)

const (
	objectWorkPackage = "workpackage"
	objectArtifact    = "artifact"
)

type helloBody struct {
	Object   string          `json:"object"`
	Manifest aasync.Manifest `json:"manifest"`
	ID       string          `json:"id"`
	Name     string          `json:"name,omitempty"`
}

type requestBody struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

// Distributor sends work packages and pulls artifacts on behalf of one node.
type Distributor struct {
	local     aasync.NodeID
	transport transport.Transport
	clock     aasync.Clock

	mu        sync.Mutex
	receipts  map[string]aasync.Receipt
	packages  map[string]aasync.WorkPackage
	artifacts map[string][]byte
}

// New returns a distributor for the local node.
func New(local aasync.NodeID, tr transport.Transport, clock aasync.Clock) *Distributor {
	if clock == nil {
		clock = aasync.NewFixedClock()
	}
	return &Distributor{
		local:     local,
		transport: tr,
		clock:     clock,
		receipts:  map[string]aasync.Receipt{},
		packages:  map[string]aasync.WorkPackage{},
		artifacts: map[string][]byte{},
	}
}

// Distribute sends a work package to a remote node. It is idempotent per
// (work package, node).
func (d *Distributor) Distribute(ctx context.Context, wp aasync.WorkPackage, remote aasync.NodeID) (aasync.Receipt, error) {
	if wp.ID == "" || remote == "" {
		return aasync.Receipt{}, fmt.Errorf("%w: work package id and node are required", aasync.ErrInvalid)
	}
	key := aasync.TransferKey("distribute", wp.ID, remote)
	if r, ok := d.cached(key); ok {
		return r, nil
	}
	wp.Hash = aasync.Hash(wp.Payload)
	src, err := transfer.NewSource(wp.ID, wp.Name, wp.Payload, 0)
	if err != nil {
		return aasync.Receipt{}, err
	}
	conn, err := d.transport.Dial(ctx, d.local, remote)
	if err != nil {
		return aasync.Receipt{}, err
	}
	hello, err := transport.NewMessage(transport.KindHello, "", "", 0, helloBody{
		Object: objectWorkPackage, Manifest: src.Manifest, ID: wp.ID, Name: wp.Name,
	})
	if err != nil {
		return aasync.Receipt{}, err
	}
	if err := conn.Send(ctx, hello); err != nil {
		return aasync.Receipt{}, err
	}
	if err := transfer.Serve(ctx, conn, src); err != nil {
		return aasync.Receipt{}, err
	}
	d.mu.Lock()
	d.packages[wp.ID] = wp
	d.mu.Unlock()
	r := aasync.Receipt{
		TransferID:    aasync.TransferID(key),
		WorkPackageID: wp.ID,
		NodeID:        remote,
		State:         aasync.TransferComplete,
		Hash:          src.Manifest.Hash,
		Bytes:         src.Manifest.Size,
		CreatedAt:     d.clock.Now(),
	}
	d.store(key, r)
	return r, nil
}

// FetchArtifact pulls an artifact held by a remote node. It is idempotent per
// (artifact, node).
func (d *Distributor) FetchArtifact(ctx context.Context, artifactID string, remote aasync.NodeID) (aasync.Receipt, error) {
	if artifactID == "" || remote == "" {
		return aasync.Receipt{}, fmt.Errorf("%w: artifact id and node are required", aasync.ErrInvalid)
	}
	key := aasync.TransferKey("artifact", artifactID, remote)
	if r, ok := d.cached(key); ok {
		return r, nil
	}
	conn, err := d.transport.Dial(ctx, d.local, remote)
	if err != nil {
		return aasync.Receipt{}, err
	}
	req, err := transport.NewMessage(transport.KindRevision, "", "", 0, requestBody{Object: objectArtifact, ID: artifactID})
	if err != nil {
		return aasync.Receipt{}, err
	}
	if err := conn.Send(ctx, req); err != nil {
		return aasync.Receipt{}, err
	}
	helloMsg, err := conn.Receive(ctx)
	if err != nil {
		return aasync.Receipt{}, err
	}
	if helloMsg.Kind == transport.KindClose {
		return aasync.Receipt{}, fmt.Errorf("%w: artifact %s", aasync.ErrNotFound, artifactID)
	}
	if helloMsg.Kind != transport.KindHello {
		return aasync.Receipt{}, fmt.Errorf("%w: expected hello, got %s", aasync.ErrInvalid, helloMsg.Kind)
	}
	var hb helloBody
	if err := transport.DecodeBody(helloMsg, &hb); err != nil {
		return aasync.Receipt{}, err
	}
	recv := transfer.NewReceiver(hb.Manifest)
	data, err := recv.Run(ctx, conn, conn)
	if err != nil {
		return aasync.Receipt{}, err
	}
	d.mu.Lock()
	d.artifacts[artifactID] = data
	d.mu.Unlock()
	r := aasync.Receipt{
		TransferID: aasync.TransferID(key),
		ArtifactID: artifactID,
		NodeID:     remote,
		State:      aasync.TransferComplete,
		Hash:       hb.Manifest.Hash,
		Bytes:      hb.Manifest.Size,
		CreatedAt:  d.clock.Now(),
	}
	d.store(key, r)
	return r, nil
}

// Status returns the receipts for a node (all nodes when node is empty),
// ordered by transfer ID.
func (d *Distributor) Status(node aasync.NodeID) []aasync.Receipt {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []aasync.Receipt
	for _, r := range d.receipts {
		if node == "" || r.NodeID == node {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransferID < out[j].TransferID })
	return out
}

// Packages returns the work packages this node has sent or received, by ID.
func (d *Distributor) Packages() []aasync.WorkPackage {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]aasync.WorkPackage, 0, len(d.packages))
	for _, wp := range d.packages {
		out = append(out, wp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ArtifactData returns the bytes of a fetched artifact.
func (d *Distributor) ArtifactData(id string) ([]byte, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	data, ok := d.artifacts[id]
	return data, ok
}

func (d *Distributor) cached(key string) (aasync.Receipt, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r, ok := d.receipts[key]
	return r, ok
}

func (d *Distributor) store(key string, r aasync.Receipt) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.receipts[key] = r
}

// Peer serves incoming distribution sessions for one node.
type Peer struct {
	local     aasync.NodeID
	transport transport.Transport

	mu           sync.Mutex
	packages     map[string]aasync.WorkPackage
	artifacts    map[string]aasync.Artifact
	artifactData map[string][]byte
}

// NewPeer returns a peer endpoint for a node.
func NewPeer(local aasync.NodeID, tr transport.Transport) *Peer {
	return &Peer{
		local:        local,
		transport:    tr,
		packages:     map[string]aasync.WorkPackage{},
		artifacts:    map[string]aasync.Artifact{},
		artifactData: map[string][]byte{},
	}
}

// Publish makes an artifact available to fetch.
func (p *Peer) Publish(a aasync.Artifact, data []byte) error {
	if a.ID == "" {
		return fmt.Errorf("%w: artifact id is required", aasync.ErrInvalid)
	}
	a.Hash = aasync.Hash(data)
	a.Size = int64(len(data))
	p.mu.Lock()
	defer p.mu.Unlock()
	p.artifacts[a.ID] = a
	p.artifactData[a.ID] = data
	return nil
}

// Serve handles one incoming session from the given node.
func (p *Peer) Serve(ctx context.Context, from aasync.NodeID) error {
	conn, err := p.transport.Dial(ctx, p.local, from)
	if err != nil {
		return err
	}
	msg, err := conn.Receive(ctx)
	if err != nil {
		return err
	}
	switch msg.Kind {
	case transport.KindHello:
		var hb helloBody
		if err := transport.DecodeBody(msg, &hb); err != nil {
			return err
		}
		if hb.Object != objectWorkPackage {
			return fmt.Errorf("%w: unexpected object %q", aasync.ErrInvalid, hb.Object)
		}
		recv := transfer.NewReceiver(hb.Manifest)
		data, err := recv.Run(ctx, conn, conn)
		if err != nil {
			return err
		}
		wp := aasync.WorkPackage{ID: hb.ID, Name: hb.Name, Payload: data, Hash: hb.Manifest.Hash}
		p.mu.Lock()
		p.packages[hb.ID] = wp
		p.mu.Unlock()
		return nil
	case transport.KindRevision:
		var rb requestBody
		if err := transport.DecodeBody(msg, &rb); err != nil {
			return err
		}
		if rb.Object != objectArtifact {
			return fmt.Errorf("%w: unexpected object %q", aasync.ErrInvalid, rb.Object)
		}
		p.mu.Lock()
		a, ok := p.artifacts[rb.ID]
		data := p.artifactData[rb.ID]
		p.mu.Unlock()
		if !ok {
			closeMsg, _ := transport.NewMessage(transport.KindClose, "", "", 0, nil)
			_ = conn.Send(ctx, closeMsg)
			return fmt.Errorf("%w: artifact %s", aasync.ErrNotFound, rb.ID)
		}
		src, err := transfer.NewSource(a.ID, a.Name, data, 0)
		if err != nil {
			return err
		}
		hello, err := transport.NewMessage(transport.KindHello, "", "", 0, helloBody{
			Object: objectArtifact, Manifest: src.Manifest, ID: a.ID, Name: a.Name,
		})
		if err != nil {
			return err
		}
		if err := conn.Send(ctx, hello); err != nil {
			return err
		}
		return transfer.Serve(ctx, conn, src)
	default:
		return fmt.Errorf("%w: unexpected frame %s", aasync.ErrInvalid, msg.Kind)
	}
}

// Packages returns the work packages this peer has received.
func (p *Peer) Packages() []aasync.WorkPackage {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]aasync.WorkPackage, 0, len(p.packages))
	for _, wp := range p.packages {
		out = append(out, wp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ArtifactData returns the bytes of a published artifact.
func (p *Peer) ArtifactData(id string) ([]byte, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	data, ok := p.artifactData[id]
	return data, ok
}
