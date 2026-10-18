package syncplane

import (
	"context"
	"testing"
	"time"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/revision"
	"github.com/greadee/aa/sync/rpc"
	"github.com/greadee/aa/sync/transport"
)

func TestLocalNodeIsRegistered(t *testing.T) {
	p := New(Config{Name: "alpha", Seed: "alpha"})
	if p.LocalID() == "" {
		t.Fatal("local node has no id")
	}
	nodes := p.Nodes()
	if len(nodes) != 1 || !nodes[0].Local || nodes[0].State != "ready" {
		t.Fatalf("nodes = %+v", nodes)
	}
	st := p.Status(context.Background())
	if st.LocalNode != p.LocalID() {
		t.Fatalf("status local = %s, want %s", st.LocalNode, p.LocalID())
	}
}

func TestDistributionRoundTrip(t *testing.T) {
	wire := transport.NewWire()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a := New(Config{Name: "a", Seed: "a", Transport: wire, Clock: aasync.NewFixedClock()})
	b := New(Config{Name: "b", Seed: "b", Transport: wire, Clock: aasync.NewFixedClock()})
	if err := a.AddPeer(b.LocalID(), "b"); err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- b.Serve(ctx, a.LocalID()) }()

	wp := aasync.WorkPackage{ID: "wp-1", ProjectID: "p1", Name: "first", Payload: []byte("hello sync")}
	receipt, err := a.Distribute(ctx, wp, b.LocalID())
	if err != nil {
		t.Fatalf("distribute: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if receipt.State != aasync.TransferComplete || receipt.Hash != aasync.Hash(wp.Payload) {
		t.Fatalf("receipt = %+v", receipt)
	}

	received := b.ReceivedPackages()
	if len(received) != 1 || string(received[0].Payload) != "hello sync" {
		t.Fatalf("received = %+v", received)
	}
	st := a.Status(ctx)
	if len(st.Transfers) != 1 || st.Transfers[0].TransferID != receipt.TransferID {
		t.Fatalf("status transfers = %+v", st.Transfers)
	}
}

func TestReconcileAppliesAuthoritativeRevision(t *testing.T) {
	p := New(Config{Name: "a", Seed: "a"})
	if err := p.Publish(aasync.RevisionRecord{ID: "r1", Path: "work/a", Revision: 1, Hash: "h1"}); err != nil {
		t.Fatal(err)
	}
	rec, err := p.Reconcile([]aasync.Change{
		{Record: aasync.RevisionRecord{ID: "r2", Path: "work/a", Revision: 2, Hash: "h2"}, Node: "peer"},
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(rec.Decision.Apply) != 1 || rec.Applied.Applied != 1 {
		t.Fatalf("reconciliation = %+v", rec)
	}

	want := revision.NewSet()
	if err := want.Put(aasync.RevisionRecord{ID: "r2", Path: "work/a", Revision: 2, Hash: "h2"}); err != nil {
		t.Fatal(err)
	}
	if p.ReplicaDigest() != want.Digest() {
		t.Fatalf("replica digest = %s, want %s", p.ReplicaDigest(), want.Digest())
	}
}

func TestHandleStatusRPC(t *testing.T) {
	p := New(Config{Name: "a", Seed: "a"})
	resp := p.Handle(context.Background(), rpc.Request{JSONRPC: "2.0", ID: 1, Method: rpc.MethodStatus})
	if resp.Error != nil {
		t.Fatalf("rpc error = %+v", resp.Error)
	}
	if _, ok := resp.Result.(map[string]any); !ok {
		t.Fatalf("result = %T", resp.Result)
	}
}

func TestAddPeerRejectsEmptyID(t *testing.T) {
	p := New(Config{Seed: "a"})
	if err := p.AddPeer("", "peer"); err == nil {
		t.Fatal("expected an error for an empty peer id")
	}
}

func TestSeedFromRootIsStable(t *testing.T) {
	dir := t.TempDir()
	first, err := SeedFromRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SeedFromRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first != second {
		t.Fatalf("seed = %q then %q", first, second)
	}
	if p := New(Config{Seed: first}); p.LocalID() != New(Config{Seed: second}).LocalID() {
		t.Fatal("same seed produced different identities")
	}
}
