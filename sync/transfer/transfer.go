// Package transfer implements resumable, hash-verified chunked transfer.
//
// A payload is described by a content-addressed manifest. Every chunk and the
// assembled payload are verified against the manifest, and a transfer resumes
// from whatever the receiver already holds.
package transfer

import (
	"context"
	"fmt"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/transport"
)

// DefaultChunkSize is used when a caller does not choose a chunk size.
const DefaultChunkSize = 64 * 1024

// Source is a payload plus its manifest, ready to send.
type Source struct {
	Manifest aasync.Manifest
	Data     []byte
}

// Build returns a manifest for data with the given chunk size.
func Build(id, name string, data []byte, chunkSize int) (aasync.Manifest, error) {
	if id == "" {
		return aasync.Manifest{}, fmt.Errorf("%w: manifest id is required", aasync.ErrInvalid)
	}
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	m := aasync.Manifest{
		ID:        id,
		Name:      name,
		Size:      int64(len(data)),
		ChunkSize: chunkSize,
		Hash:      aasync.Hash(data),
	}
	for offset, index := 0, 0; offset < len(data); index, offset = index+1, offset+chunkSize {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		part := data[offset:end]
		m.Chunks = append(m.Chunks, aasync.Chunk{
			Index:  index,
			Offset: int64(offset),
			Length: len(part),
			Hash:   aasync.Hash(part),
		})
	}
	return m, nil
}

// NewSource builds a source payload.
func NewSource(id, name string, data []byte, chunkSize int) (Source, error) {
	m, err := Build(id, name, data, chunkSize)
	if err != nil {
		return Source{}, err
	}
	return Source{Manifest: m, Data: data}, nil
}

// ChunkData returns the bytes for a chunk index.
func (s Source) ChunkData(index int) ([]byte, error) {
	if index < 0 || index >= len(s.Manifest.Chunks) {
		return nil, fmt.Errorf("%w: chunk %d", aasync.ErrNotFound, index)
	}
	ch := s.Manifest.Chunks[index]
	start := ch.Offset
	end := start + int64(ch.Length)
	if end > int64(len(s.Data)) {
		return nil, fmt.Errorf("%w: chunk %d out of range", aasync.ErrCorrupt, index)
	}
	return s.Data[start:end], nil
}

// Receiver accumulates verified chunks for one manifest.
type Receiver struct {
	manifest aasync.Manifest
	parts    map[int][]byte
	have     map[int]bool
}

// NewReceiver returns an empty receiver for a manifest.
func NewReceiver(manifest aasync.Manifest) *Receiver {
	return &Receiver{
		manifest: manifest,
		parts:    map[int][]byte{},
		have:     map[int]bool{},
	}
}

// Accept verifies and stores one chunk. A chunk that is out of range, the
// wrong length, or the wrong hash is rejected.
func (r *Receiver) Accept(chunk aasync.Chunk, data []byte) error {
	if chunk.Index < 0 || chunk.Index >= len(r.manifest.Chunks) {
		return fmt.Errorf("%w: chunk index %d", aasync.ErrInvalid, chunk.Index)
	}
	want := r.manifest.Chunks[chunk.Index]
	if len(data) != want.Length {
		return fmt.Errorf("%w: chunk %d length %d, want %d", aasync.ErrCorrupt, chunk.Index, len(data), want.Length)
	}
	if aasync.Hash(data) != want.Hash {
		return fmt.Errorf("%w: chunk %d hash mismatch", aasync.ErrCorrupt, chunk.Index)
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	r.parts[chunk.Index] = cp
	r.have[chunk.Index] = true
	return nil
}

// Have returns the received chunk indexes in order.
func (r *Receiver) Have() []int {
	out := make([]int, 0, len(r.have))
	for i := 0; i < len(r.manifest.Chunks); i++ {
		if r.have[i] {
			out = append(out, i)
		}
	}
	return out
}

// Missing returns the chunk indexes not yet received.
func (r *Receiver) Missing() []int {
	var out []int
	for i := range r.manifest.Chunks {
		if !r.have[i] {
			out = append(out, i)
		}
	}
	return out
}

// Progress returns the fraction of chunks received.
func (r *Receiver) Progress() float64 {
	if len(r.manifest.Chunks) == 0 {
		return 1
	}
	return float64(len(r.have)) / float64(len(r.manifest.Chunks))
}

// Complete assembles the payload and verifies the whole-content hash.
func (r *Receiver) Complete() ([]byte, error) {
	if len(r.Missing()) != 0 {
		return nil, fmt.Errorf("%w: %d chunks missing", aasync.ErrNotFound, len(r.Missing()))
	}
	buf := make([]byte, 0, r.manifest.Size)
	for i := range r.manifest.Chunks {
		buf = append(buf, r.parts[i]...)
	}
	if aasync.Hash(buf) != r.manifest.Hash {
		return nil, fmt.Errorf("%w: payload hash mismatch", aasync.ErrCorrupt)
	}
	return buf, nil
}

type chunkBody struct {
	Chunk aasync.Chunk `json:"chunk"`
	Data  []byte       `json:"data"`
}

type offerBody struct {
	Have []int `json:"have"`
}

// push sends every chunk not in have, then a close frame.
func (s Source) push(ctx context.Context, conn transport.Conn, have []int) error {
	skip := map[int]bool{}
	for _, i := range have {
		skip[i] = true
	}
	for _, chunk := range s.Manifest.Chunks {
		if skip[chunk.Index] {
			continue
		}
		data, err := s.ChunkData(chunk.Index)
		if err != nil {
			return err
		}
		msg, err := transport.NewMessage(transport.KindChunk, "", "", uint64(chunk.Index+1), chunkBody{Chunk: chunk, Data: data})
		if err != nil {
			return err
		}
		if err := conn.Send(ctx, msg); err != nil {
			return err
		}
	}
	closeMsg, err := transport.NewMessage(transport.KindClose, "", "", 0, nil)
	if err != nil {
		return err
	}
	return conn.Send(ctx, closeMsg)
}

// WaitOffer reads until the receiver offers its received set.
func WaitOffer(ctx context.Context, conn transport.Conn) ([]int, error) {
	for {
		msg, err := conn.Receive(ctx)
		if err != nil {
			return nil, err
		}
		switch msg.Kind {
		case transport.KindAck:
			var offer offerBody
			if err := transport.DecodeBody(msg, &offer); err != nil {
				return nil, err
			}
			return offer.Have, nil
		default:
			return nil, fmt.Errorf("%w: expected offer, got %s", aasync.ErrInvalid, msg.Kind)
		}
	}
}

// Serve waits for an offer and sends the missing chunks over conn.
func Serve(ctx context.Context, conn transport.Conn, src Source) error {
	have, err := WaitOffer(ctx, conn)
	if err != nil {
		return err
	}
	return src.push(ctx, conn, have)
}

// Run receives an offer request and collects chunks from readConn.
func (r *Receiver) Run(ctx context.Context, readConn, writeConn transport.Conn) ([]byte, error) {
	offer, err := transport.NewMessage(transport.KindAck, "", "", 0, offerBody{Have: r.Have()})
	if err != nil {
		return nil, err
	}
	if err := writeConn.Send(ctx, offer); err != nil {
		return nil, err
	}
	for {
		msg, err := readConn.Receive(ctx)
		if err != nil {
			return nil, err
		}
		switch msg.Kind {
		case transport.KindChunk:
			var body chunkBody
			if err := transport.DecodeBody(msg, &body); err != nil {
				return nil, err
			}
			if err := r.Accept(body.Chunk, body.Data); err != nil {
				return nil, err
			}
		case transport.KindClose:
			return r.Complete()
		default:
			return nil, fmt.Errorf("%w: unexpected frame %s", aasync.ErrInvalid, msg.Kind)
		}
	}
}

// Transfer runs a full resumable transfer between two directional connections.
//
//	senderConn   — the sender's connection (send chunks, receive the offer)
//	receiverConn — the receiver's connection (send the offer, receive chunks)
func Transfer(ctx context.Context, senderConn, receiverConn transport.Conn, src Source, recv *Receiver) error {
	errCh := make(chan error, 1)
	go func() {
		_, err := recv.Run(ctx, receiverConn, receiverConn)
		errCh <- err
	}()
	if err := Serve(ctx, senderConn, src); err != nil {
		return err
	}
	return <-errCh
}
