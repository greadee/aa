package distribute

import (
	"context"
	"errors"
	"testing"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/transport"
)

func TestDistributeIsIdempotent(t *testing.T) {
	ctx := context.Background()
	w := transport.NewWire()
	sender := aasync.NodeID("sender")
	receiver := aasync.NodeID("receiver")
	d := New(sender, w, aasync.NewFixedClock())
	p := NewPeer(receiver, w)

	errCh := make(chan error, 1)
	go func() { errCh <- p.Serve(ctx, sender) }()

	wp := aasync.WorkPackage{ID: "wp1", ProjectID: "proj", Name: "task", Payload: []byte("do the thing")}
	first, err := d.Distribute(ctx, wp, receiver)
	if err != nil {
		t.Fatalf("distribute: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if first.State != aasync.TransferComplete || first.WorkPackageID != "wp1" {
		t.Fatalf("receipt = %+v", first)
	}
	pkgs := p.Packages()
	if len(pkgs) != 1 || string(pkgs[0].Payload) != "do the thing" {
		t.Fatalf("peer packages = %+v", pkgs)
	}
	if pkgs[0].Hash != first.Hash {
		t.Fatalf("hash mismatch: %q vs %q", pkgs[0].Hash, first.Hash)
	}

	second, err := d.Distribute(ctx, wp, receiver)
	if err != nil {
		t.Fatalf("second distribute: %v", err)
	}
	if second != first {
		t.Fatalf("second receipt = %+v, want original", second)
	}
	if got := len(d.Status(receiver)); got != 1 {
		t.Fatalf("status receipts = %d, want 1", got)
	}
}

func TestFetchArtifact(t *testing.T) {
	ctx := context.Background()
	w := transport.NewWire()
	fetcher := aasync.NodeID("fetcher")
	owner := aasync.NodeID("owner")
	d := New(fetcher, w, aasync.NewFixedClock())
	p := NewPeer(owner, w)

	data := []byte("artifact bytes")
	if err := p.Publish(aasync.Artifact{ID: "a1", Name: "out.bin"}, data); err != nil {
		t.Fatalf("publish: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- p.Serve(ctx, fetcher) }()

	r, err := d.FetchArtifact(ctx, "a1", owner)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if r.ArtifactID != "a1" || r.Bytes != int64(len(data)) {
		t.Fatalf("receipt = %+v", r)
	}
	got, ok := d.ArtifactData("a1")
	if !ok || string(got) != string(data) {
		t.Fatalf("artifact data = %q, ok = %t", got, ok)
	}
	if _, err := d.FetchArtifact(ctx, "a1", owner); err != nil {
		t.Fatalf("idempotent fetch: %v", err)
	}
}

func TestFetchUnknownArtifact(t *testing.T) {
	ctx := context.Background()
	w := transport.NewWire()
	d := New(aasync.NodeID("fetcher2"), w, aasync.NewFixedClock())
	p := NewPeer(aasync.NodeID("owner2"), w)
	go func() { _ = p.Serve(ctx, aasync.NodeID("fetcher2")) }()
	_, err := d.FetchArtifact(ctx, "missing", aasync.NodeID("owner2"))
	if !errors.Is(err, aasync.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDistributeRejectsMissingArgs(t *testing.T) {
	d := New("n", transport.NewWire(), aasync.NewFixedClock())
	if _, err := d.Distribute(context.Background(), aasync.WorkPackage{}, "peer"); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}
