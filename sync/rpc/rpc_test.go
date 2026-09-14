package rpc

import (
	"context"
	"testing"

	aasync "github.com/greadee/aa/sync"
	"github.com/greadee/aa/sync/distribute"
	"github.com/greadee/aa/sync/transport"
)

func newService() (*Service, *distribute.Distributor, *transport.Wire) {
	w := transport.NewWire()
	d := distribute.New(aasync.NodeID("sender"), w, aasync.NewFixedClock())
	return New(d), d, w
}

func TestDistributeMethodAndReplay(t *testing.T) {
	ctx := context.Background()
	svc, d, w := newService()
	peer := distribute.NewPeer(aasync.NodeID("receiver"), w)
	if err := d.Register(aasync.WorkPackage{ID: "wp1", Payload: []byte("payload")}); err != nil {
		t.Fatalf("register: %v", err)
	}
	go func() { _ = peer.Serve(ctx, aasync.NodeID("sender")) }()

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  MethodDistribute,
		Params:  map[string]any{"workPackageId": "wp1", "nodeId": "receiver", "idempotencyKey": "k1"},
	}
	resp := svc.Handle(ctx, req)
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok || result["accepted"] != true {
		t.Fatalf("result = %#v", resp.Result)
	}
	if result["transferId"] == "" {
		t.Fatal("missing transferId")
	}

	// A replayed call returns the cached result without redistributing.
	replay := svc.Handle(ctx, req)
	if replay.Error != nil {
		t.Fatalf("replay error: %+v", replay.Error)
	}
	if replay.Result.(map[string]any)["transferId"] != result["transferId"] {
		t.Fatalf("replay result = %#v, want cached", replay.Result)
	}
}

func TestFetchArtifactMethod(t *testing.T) {
	ctx := context.Background()
	svc, _, w := newService()
	peer := distribute.NewPeer(aasync.NodeID("owner"), w)
	if err := peer.Publish(aasync.Artifact{ID: "a1", Name: "x"}, []byte("artifact")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	go func() { _ = peer.Serve(ctx, aasync.NodeID("sender")) }()

	resp := svc.Handle(ctx, Request{
		JSONRPC: "2.0", ID: 2, Method: MethodFetchArtifact,
		Params: map[string]any{"artifactId": "a1", "nodeId": "owner"},
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	result := resp.Result.(map[string]any)
	if result["hash"] == "" || result["transferId"] == "" {
		t.Fatalf("result = %#v", result)
	}
}

func TestStatusMethod(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newService()
	resp := svc.Handle(ctx, Request{JSONRPC: "2.0", ID: 3, Method: MethodStatus, Params: map[string]any{}})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	if _, ok := resp.Result.(map[string]any)["transfers"]; !ok {
		t.Fatalf("result = %#v", resp.Result)
	}
}

func TestUnknownMethod(t *testing.T) {
	svc, _, _ := newService()
	resp := svc.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 4, Method: "sync.nope"})
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("error = %+v", resp.Error)
	}
}

func TestIncompatibleMajor(t *testing.T) {
	svc, _, _ := newService()
	resp := svc.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 5, Method: MethodStatus, AA: &AA{RPCVersion: "2.1"}})
	if resp.Error == nil || resp.Error.Code != CodeIncompatible {
		t.Fatalf("error = %+v", resp.Error)
	}
}

func TestInvalidParams(t *testing.T) {
	svc, _, _ := newService()
	resp := svc.Handle(context.Background(), Request{
		JSONRPC: "2.0", ID: 6, Method: MethodDistribute, Params: map[string]any{"nodeId": "n"},
	})
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Fatalf("error = %+v", resp.Error)
	}
}
