package rpc

import (
	"context"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/host"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
	"github.com/greadee/aa/toolbox/workflow"
)

type resolver map[string]toolbox.ExecutionContract

func (r resolver) Resolve(id string) (toolbox.ExecutionContract, bool) {
	c, ok := r[id]
	return c, ok
}

func newService(t *testing.T) (*Service, *host.FakeProvider) {
	t.Helper()
	reg := registry.New()
	p := host.NewFakeProvider(toolbox.KindTool)
	if err := reg.RegisterProvider(p); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	m := toolbox.Manifest{
		Envelope:     toolbox.Envelope{ContractVersion: "1.0", Kind: "tool_manifest", ID: "tool_a"},
		Name:         "tool_a",
		ToolKind:     toolbox.KindTool,
		Version:      "1.0",
		Capabilities: []string{"read_project"},
	}
	if err := reg.Register(m); err != nil {
		t.Fatalf("Register: %v", err)
	}
	h := host.New(reg, policy.New(nil, toolbox.NewFixedClock()))
	contracts := resolver{
		"ec_ok": {Capabilities: []toolbox.Capability{"read_project"}},
		"ec_no": {Capabilities: []toolbox.Capability{"run_tests"}},
	}
	return New(h, contracts), p
}

func req(method string, params map[string]any) Request {
	return Request{JSONRPC: "2.0", ID: 1, Method: method, Params: params}
}

func TestInvokeSuccess(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok", "input": map[string]any{"x": 1},
	}))
	if resp.Error != nil {
		t.Fatalf("error = %+v", resp.Error)
	}
	result := resp.Result.(map[string]any)
	output := result["output"].(map[string]any)
	if output["tool"] != "tool_a" || output["x"] != 1 {
		t.Fatalf("output = %v", output)
	}
}

func TestInvokeDeniedIsUnauthorized(t *testing.T) {
	s, p := newService(t)
	resp := s.Handle(context.Background(), req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_no",
	}))
	if resp.Error == nil || resp.Error.Code != CodeUnauthorized {
		t.Fatalf("error = %+v, want unauthorized", resp.Error)
	}
	if len(p.Calls()) != 0 {
		t.Fatal("provider ran despite denial")
	}
}

func TestInvokeMissingContract(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ghost",
	}))
	if resp.Error == nil || resp.Error.Code != CodeNotFound {
		t.Fatalf("error = %+v, want not found", resp.Error)
	}
}

func TestInvokeIdempotentReplay(t *testing.T) {
	s, p := newService(t)
	params := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok",
		"input": map[string]any{"x": 1}, "idempotencyKey": "k1",
	}
	first := s.Handle(context.Background(), req(MethodInvoke, params))
	second := s.Handle(context.Background(), req(MethodInvoke, params))
	if first.Error != nil || second.Error != nil {
		t.Fatalf("errors: %+v %+v", first.Error, second.Error)
	}
	if len(p.Calls()) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(p.Calls()))
	}
}

func TestCompileWorkflow(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodCompileWorkflow, map[string]any{
		"workflow": map[string]any{
			"contractVersion": "1.0", "kind": "workflow", "id": "wf_1", "version": "1.0",
			"steps": []any{
				map[string]any{"id": "a", "kind": "task"},
				map[string]any{"id": "b", "kind": "task", "dependsOn": []any{"a"}},
			},
		},
	}))
	if resp.Error != nil {
		t.Fatalf("error = %+v", resp.Error)
	}
	plan, ok := resp.Result.(map[string]any)["plan"].(workflow.Plan)
	if !ok {
		t.Fatalf("plan type = %T", resp.Result.(map[string]any)["plan"])
	}
	if len(plan.Order) != 2 || plan.Order[0] != "a" || plan.Order[1] != "b" {
		t.Fatalf("order = %v", plan.Order)
	}
}

func TestCompileWorkflowCycleIsInvalidParams(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodCompileWorkflow, map[string]any{
		"workflow": map[string]any{
			"contractVersion": "1.0", "kind": "workflow", "id": "wf", "version": "1.0",
			"steps": []any{
				map[string]any{"id": "a", "kind": "task", "dependsOn": []any{"b"}},
				map[string]any{"id": "b", "kind": "task", "dependsOn": []any{"a"}},
			},
		},
	}))
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Fatalf("error = %+v, want invalid params", resp.Error)
	}
}

func TestRegistry(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodRegistry, map[string]any{}))
	if resp.Error != nil {
		t.Fatalf("error = %+v", resp.Error)
	}
	tools := resp.Result.(map[string]any)["tools"].([]toolbox.Manifest)
	if len(tools) != 1 || tools[0].ID != "tool_a" {
		t.Fatalf("tools = %+v", tools)
	}
	resp2 := s.Handle(context.Background(), req(MethodRegistry, map[string]any{"toolKind": toolbox.KindPlugin}))
	tools2 := resp2.Result.(map[string]any)["tools"].([]toolbox.Manifest)
	if len(tools2) != 0 {
		t.Fatalf("filtered tools = %+v", tools2)
	}
}

func TestUnknownMethod(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req("toolbox.nope", nil))
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("error = %+v", resp.Error)
	}
}

func TestIncompatibleRPCVersion(t *testing.T) {
	s, _ := newService(t)
	r := req(MethodRegistry, map[string]any{})
	r.AA = &AA{RPCVersion: "2.0"}
	resp := s.Handle(context.Background(), r)
	if resp.Error == nil || resp.Error.Code != CodeIncompatible {
		t.Fatalf("error = %+v", resp.Error)
	}
}

func TestInvalidJSONRPC(t *testing.T) {
	s, _ := newService(t)
	r := req(MethodRegistry, map[string]any{})
	r.JSONRPC = "1.0"
	resp := s.Handle(context.Background(), r)
	if resp.Error == nil || resp.Error.Code != CodeInvalidRequest {
		t.Fatalf("error = %+v", resp.Error)
	}
}
