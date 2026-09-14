package transfer

import (
	"context"
	"errors"
	"testing"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/transport"
)

func TestBuildChunkMath(t *testing.T) {
	data := []byte("0123456789")
	m, err := Build("m1", "payload", data, 4)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if m.Size != 10 || m.ChunkSize != 4 || len(m.Chunks) != 3 {
		t.Fatalf("manifest = %+v", m)
	}
	if m.Hash != aasync.Hash(data) {
		t.Fatalf("manifest hash = %q", m.Hash)
	}
	wantOffsets := []int64{0, 4, 8}
	wantLengths := []int{4, 4, 2}
	for i, ch := range m.Chunks {
		if ch.Offset != wantOffsets[i] || ch.Length != wantLengths[i] {
			t.Fatalf("chunk %d = %+v", i, ch)
		}
	}
}

func TestBuildDefaultsAndRejects(t *testing.T) {
	m, err := Build("m2", "", []byte("x"), 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if m.ChunkSize != DefaultChunkSize {
		t.Fatalf("chunk size = %d, want default", m.ChunkSize)
	}
	if _, err := Build("", "", nil, 4); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("empty id err = %v, want ErrInvalid", err)
	}
}

func TestReceiverAcceptsAndCompletes(t *testing.T) {
	data := []byte("hello aa-sync payload")
	src, err := NewSource("m3", "p", data, 5)
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	recv := NewReceiver(src.Manifest)
	for i := range src.Manifest.Chunks {
		part, _ := src.ChunkData(i)
		if err := recv.Accept(src.Manifest.Chunks[i], part); err != nil {
			t.Fatalf("accept %d: %v", i, err)
		}
	}
	if recv.Progress() != 1 {
		t.Fatalf("progress = %v, want 1", recv.Progress())
	}
	got, err := recv.Complete()
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("payload = %q", got)
	}
}

func TestReceiverRejectsCorruptChunk(t *testing.T) {
	src, _ := NewSource("m4", "p", []byte("abcdefgh"), 4)
	recv := NewReceiver(src.Manifest)
	if err := recv.Accept(src.Manifest.Chunks[0], []byte("XXXX")); !errors.Is(err, aasync.ErrCorrupt) {
		t.Fatalf("bad hash err = %v, want ErrCorrupt", err)
	}
	if err := recv.Accept(src.Manifest.Chunks[0], []byte("abc")); !errors.Is(err, aasync.ErrCorrupt) {
		t.Fatalf("bad length err = %v, want ErrCorrupt", err)
	}
	if err := recv.Accept(aasync.Chunk{Index: 9}, []byte("x")); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("bad index err = %v, want ErrInvalid", err)
	}
	if _, err := recv.Complete(); !errors.Is(err, aasync.ErrNotFound) {
		t.Fatalf("incomplete err = %v, want ErrNotFound", err)
	}
}

func TestTransferOverWire(t *testing.T) {
	ctx := context.Background()
	data := []byte("the quick brown fox jumps over the lazy dog")
	src, _ := NewSource("m5", "p", data, 7)
	w := transport.NewWire()
	sender := aasync.NodeID("sender")
	receiver := aasync.NodeID("receiver")
	sConn, _ := w.Dial(ctx, sender, receiver)
	rConn, _ := w.Dial(ctx, receiver, sender)

	recv := NewReceiver(src.Manifest)
	if err := Transfer(ctx, sConn, rConn, src, recv); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	got, err := recv.Complete()
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("payload = %q", got)
	}
}

func TestTransferResumesFromPartial(t *testing.T) {
	ctx := context.Background()
	data := []byte("0123456789ABCDEF")
	src, _ := NewSource("m6", "p", data, 4)
	w := transport.NewWire()
	sender := aasync.NodeID("s6")
	receiver := aasync.NodeID("r6")
	sConn, _ := w.Dial(ctx, sender, receiver)
	rConn, _ := w.Dial(ctx, receiver, sender)

	recv := NewReceiver(src.Manifest)
	part0, _ := src.ChunkData(0)
	if err := recv.Accept(src.Manifest.Chunks[0], part0); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := Transfer(ctx, sConn, rConn, src, recv); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	got, err := recv.Complete()
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("payload = %q", got)
	}
	// 3 missing chunks + close reach the receiver; 1 offer reaches the sender.
	if got := w.Delivered(receiver); got != 4 {
		t.Fatalf("delivered to receiver = %d, want 4", got)
	}
	if got := w.Delivered(sender); got != 1 {
		t.Fatalf("delivered to sender = %d, want 1", got)
	}
}
