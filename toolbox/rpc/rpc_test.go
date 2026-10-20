package rpc

import (
	"context"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/host"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
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
		Envelope:     toolbox.Envelope{ContractVersion: "2.0", Kind: "tool_manifest", ID: "tool_a"},
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
	return New(h, contracts, nil), p
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
	// Result is now a typed InvokeOutput.
	out, ok := resp.Result.(InvokeOutput)
	if !ok {
		t.Fatalf("result type = %T, want InvokeOutput", resp.Result)
	}
	output := out.Output
	if output["tool"] != "tool_a" {
		t.Fatalf("output[tool] = %v, want tool_a", output["tool"])
	}
	x := output["x"]
	if f, ok := x.(float64); !ok || f != 1 {
		t.Fatalf("output[x] = %v (%T), want 1", x, x)
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
			"contractVersion": "2.0", "kind": "workflow", "id": "wf_1", "version": "1.0",
			"steps": []any{
				map[string]any{"id": "a", "kind": "task"},
				map[string]any{"id": "b", "kind": "task", "dependsOn": []any{"a"}},
			},
		},
	}))
	if resp.Error != nil {
		t.Fatalf("error = %+v", resp.Error)
	}
	// Result is now a typed CompileWorkflowResponse.
	wr, ok := resp.Result.(CompileWorkflowResponse)
	if !ok {
		t.Fatalf("result type = %T, want CompileWorkflowResponse", resp.Result)
	}
	plan := wr.Plan
	if len(plan.Order) != 2 || plan.Order[0] != "a" || plan.Order[1] != "b" {
		t.Fatalf("order = %v", plan.Order)
	}
}

func TestCompileWorkflowCycleIsInvalidParams(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodCompileWorkflow, map[string]any{
		"workflow": map[string]any{
			"contractVersion": "2.0", "kind": "workflow", "id": "wf", "version": "1.0",
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
	// Result is now a typed ListRegistryResponse.
	lr, ok := resp.Result.(ListRegistryResponse)
	if !ok {
		t.Fatalf("result type = %T, want ListRegistryResponse", resp.Result)
	}
	tools := lr.Tools
	if len(tools) != 1 || tools[0].ID != "tool_a" {
		t.Fatalf("tools = %+v", tools)
	}
	resp2 := s.Handle(context.Background(), req(MethodRegistry, map[string]any{"toolKind": toolbox.KindPlugin}))
	lr2, ok := resp2.Result.(ListRegistryResponse)
	if !ok {
		t.Fatalf("result2 type = %T, want ListRegistryResponse", resp2.Result)
	}
	tools2 := lr2.Tools
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

func TestVersionCompatibilityAndCapabilities(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	minor := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 1, Method: MethodRegistry, AA: &AA{RPCVersion: "1.7", ContractVersion: "2.3"}})
	if minor.Error != nil {
		t.Fatalf("minor versions must be compatible: %+v", minor.Error)
	}

	contract := service.Handle(ctx, Request{JSONRPC: "2.0", ID: 2, Method: MethodRegistry, AA: &AA{ContractVersion: "1.0"}})
	if contract.Error == nil || contract.Error.Code != CodeIncompatible {
		t.Fatalf("expected aa.incompatible, got %+v", contract.Error)
	}

	caps := service.Capabilities()
	if caps.ContractVersion != "2.0" || len(caps.Methods) != 3 {
		t.Fatalf("capabilities = %+v", caps)
	}
}
