package rpc

import (
	"context"
	"testing"

	"github.com/greadee/aa/forge"
	"github.com/greadee/aa/forge/fake"
)

func testService(t *testing.T) (*Service, string) {
	t.Helper()
	f := fake.New()
	repo, err := f.CreateRepository(context.Background(), forge.Repository{Owner: "greadee", Name: "aa1"})
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	return New(f), repo.ID
}

func TestCreateIssueAndIdempotentReplay(t *testing.T) {
	service, repoID := testService(t)
	req := Request{
		JSONRPC: "2.0", ID: "rpc_1", Method: MethodCreateIssue,
		Params: map[string]any{
			"repoId":         repoID,
			"issue":          map[string]any{"title": "add forge", "body": "b"},
			"idempotencyKey": "create-issue-1",
		},
		AA: &AA{RPCVersion: "1.0"},
	}
	first := service.Handle(context.Background(), req)
	if first.Error != nil {
		t.Fatalf("error: %+v", first.Error)
	}
	ref, ok := first.Result.(map[string]any)["forgeRef"].(map[string]any)
	if !ok || ref["id"] == "" {
		t.Fatalf("missing forgeRef: %+v", first.Result)
	}
	second := service.Handle(context.Background(), req)
	if second.Error != nil {
		t.Fatalf("replay error: %+v", second.Error)
	}
	if ref["id"] != second.Result.(map[string]any)["forgeRef"].(map[string]any)["id"] {
		t.Fatal("replay returned a different reference")
	}
}

func TestAllForgeMethods(t *testing.T) {
	service, repoID := testService(t)
	ctx := context.Background()

	pr := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 1, Method: MethodOpenPR, Params: map[string]any{
		"repoId":      repoID,
		"pullRequest": map[string]any{"title": "child", "base": "main", "head": "feature/x"},
	}})
	if pr.Error != nil {
		t.Fatalf("openPullRequest: %+v", pr.Error)
	}

	checkpoint := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 2, Method: MethodCheckpoint, Params: map[string]any{
		"repoId":     repoID,
		"checkpoint": map[string]any{"phase": "ph5-forge", "commit": "deadbeef"},
	}})
	if checkpoint.Error != nil {
		t.Fatalf("checkpoint: %+v", checkpoint.Error)
	}

	release := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 3, Method: MethodRelease, Params: map[string]any{
		"repoId":  repoID,
		"release": map[string]any{"tag": "v0.5.0", "notes": "phase 5"},
	}})
	if release.Error != nil {
		t.Fatalf("release: %+v", release.Error)
	}
}

func TestErrors(t *testing.T) {
	service, repoID := testService(t)
	ctx := context.Background()

	unknown := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 1, Method: "forge.merge", Params: map[string]any{}})
	if unknown.Error == nil || unknown.Error.Code != CodeMethodNotFound {
		t.Fatalf("expected method-not-found, got %+v", unknown.Error)
	}

	incompatible := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 2, Method: MethodCreateIssue, AA: &AA{RPCVersion: "2.0"}})
	if incompatible.Error == nil || incompatible.Error.Code != CodeIncompatible {
		t.Fatalf("expected incompatible, got %+v", incompatible.Error)
	}

	invalid := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 3, Method: MethodCreateIssue, Params: map[string]any{
		"repoId": repoID,
	}})
	if invalid.Error == nil || invalid.Error.Code != CodeInvalidParams {
		t.Fatalf("expected invalid params, got %+v", invalid.Error)
	}
}
