// Package transport defines aa-sync's transport seam and a deterministic
// in-memory implementation for tests.
//
// The seam is deliberately small: a node dials a peer and then sends and
// receives envelope messages. A live network transport is a follow-up; the
// fake keeps the default test suite offline and deterministic.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	aasync "github.com/greadee/aa/sync"
)

// Sentinel errors returned by transports.
var (
	// ErrClosed means the connection or wire is closed.
	ErrClosed = errors.New("transport: closed")
	// ErrUnknownPeer means the peer is not registered on the wire.
	ErrUnknownPeer = errors.New("transport: unknown peer")
	// ErrDropped means the wire injected a delivery failure.
	ErrDropped = errors.New("transport: message dropped")
)

// Kind classifies envelope messages.
type Kind string

// Envelope kinds used by the sync protocols.
const (
	KindHello    Kind = "hello"
	KindChunk    Kind = "chunk"
	KindAck      Kind = "ack"
	KindRevision Kind = "revision"
	KindReceipt  Kind = "receipt"
	KindClose    Kind = "close"
)

// Message is one envelope frame between two nodes.
type Message struct {
	Kind Kind            `json:"kind"`
	From aasync.NodeID   `json:"from"`
	To   aasync.NodeID   `json:"to"`
	Seq  uint64          `json:"seq"`
	Body json.RawMessage `json:"body,omitempty"`
}

// Conn is a one-way-addressed duplex connection to a peer.
type Conn interface {
	Send(ctx context.Context, msg Message) error
	Receive(ctx context.Context) (Message, error)
	Close() error
	Peer() aasync.NodeID
}

// Transport dials peers.
type Transport interface {
	Dial(ctx context.Context, local, remote aasync.NodeID) (Conn, error)
}

// NewMessage builds a message with a JSON-encoded body.
func NewMessage(kind Kind, from, to aasync.NodeID, seq uint64, body any) (Message, error) {
	msg := Message{Kind: kind, From: from, To: to, Seq: seq}
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return Message{}, fmt.Errorf("%w: encode body: %v", aasync.ErrInvalid, err)
		}
		msg.Body = raw
	}
	return msg, nil
}

// DecodeBody decodes a message body into target.
func DecodeBody(msg Message, target any) error {
	if len(msg.Body) == 0 {
		return fmt.Errorf("%w: empty body", aasync.ErrInvalid)
	}
	if err := json.Unmarshal(msg.Body, target); err != nil {
		return fmt.Errorf("%w: decode body: %v", aasync.ErrInvalid, err)
	}
	return nil
}

// Wire is a deterministic in-memory transport. Every registered node has a
// FIFO inbox; sends append to the peer's inbox and receives pop the local one.
type Wire struct {
	mu        sync.Mutex
	capacity  int
	inbox     map[aasync.NodeID]chan Message
	closed    map[aasync.NodeID]bool
	seq       map[string]uint64
	fail      map[aasync.NodeID]int
	delivered map[aasync.NodeID]int
}

// NewWire returns an empty wire with a default inbox capacity.
func NewWire() *Wire {
	return &Wire{
		capacity:  1024,
		inbox:     map[aasync.NodeID]chan Message{},
		closed:    map[aasync.NodeID]bool{},
		seq:       map[string]uint64{},
		fail:      map[aasync.NodeID]int{},
		delivered: map[aasync.NodeID]int{},
	}
}

// Register adds nodes to the wire.
func (w *Wire) Register(nodes ...aasync.NodeID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ensureLocked(nodes...)
}

func (w *Wire) ensureLocked(nodes ...aasync.NodeID) {
	for _, n := range nodes {
		if _, ok := w.inbox[n]; !ok {
			w.inbox[n] = make(chan Message, w.capacity)
			w.closed[n] = false
		}
	}
}

// FailNext makes the next n sends to node fail, simulating a dropped link.
func (w *Wire) FailNext(node aasync.NodeID, n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ensureLocked(node)
	w.fail[node] += n
}

// Delivered returns how many messages a node has successfully received.
func (w *Wire) Delivered(node aasync.NodeID) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.delivered[node]
}

// Dial returns a connection between two registered nodes.
func (w *Wire) Dial(_ context.Context, local, remote aasync.NodeID) (Conn, error) {
	if local == "" || remote == "" {
		return nil, aasync.ErrInvalid
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ensureLocked(local, remote)
	if w.closed[local] || w.closed[remote] {
		return nil, ErrClosed
	}
	return &memConn{wire: w, local: local, remote: remote}, nil
}

type memConn struct {
	wire   *Wire
	local  aasync.NodeID
	remote aasync.NodeID
	closed bool
}

func (c *memConn) Peer() aasync.NodeID { return c.remote }

func (c *memConn) Send(_ context.Context, msg Message) error {
	c.wire.mu.Lock()
	defer c.wire.mu.Unlock()
	if c.closed || c.wire.closed[c.local] {
		return ErrClosed
	}
	if msg.From == "" {
		msg.From = c.local
	}
	if msg.To == "" {
		msg.To = c.remote
	}
	if msg.From != c.local || msg.To != c.remote {
		return fmt.Errorf("%w: envelope does not match connection", aasync.ErrInvalid)
	}
	if c.wire.fail[c.remote] > 0 {
		c.wire.fail[c.remote]--
		return ErrDropped
	}
	ch, ok := c.wire.inbox[c.remote]
	if !ok {
		return ErrUnknownPeer
	}
	select {
	case ch <- msg:
		return nil
	default:
		return fmt.Errorf("%w: inbox full", aasync.ErrConflict)
	}
}

func (c *memConn) Receive(ctx context.Context) (Message, error) {
	c.wire.mu.Lock()
	if c.closed {
		c.wire.mu.Unlock()
		return Message{}, ErrClosed
	}
	ch := c.wire.inbox[c.local]
	c.wire.mu.Unlock()
	select {
	case msg := <-ch:
		c.wire.mu.Lock()
		c.wire.delivered[c.local]++
		c.wire.mu.Unlock()
		return msg, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}

func (c *memConn) Close() error {
	c.wire.mu.Lock()
	defer c.wire.mu.Unlock()
	c.closed = true
	return nil
}

// Seq returns the next sequence number for a connection, keyed by direction.
func (w *Wire) NextSeq(from, to aasync.NodeID) uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := string(from) + "->" + string(to)
	w.seq[key]++
	return w.seq[key]
}
