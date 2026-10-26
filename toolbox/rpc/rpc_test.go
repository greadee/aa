package rpc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func newServiceWithMetrics(t *testing.T) (*Service, *toolbox.MemMetrics) {
	t.Helper()
	metrics := toolbox.NewMemMetrics()
	// Rebuild with the observable collector so RPC-level counters are checked.
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
	h := host.New(reg, policy.New(nil, toolbox.NewFixedClock(), metrics), metrics)
	contracts := resolver{
		"ec_ok": {Capabilities: []toolbox.Capability{"read_project"}},
		"ec_no": {Capabilities: []toolbox.Capability{"run_tests"}},
	}
	return New(h, contracts, metrics), metrics
}

func req(method string, params map[string]any) Request {
	return Request{JSONRPC: "2.0", ID: 1, Method: method, Params: params}
}

type blockingProvider struct {
	kind    string
	started chan struct{}
	release chan struct{}
	calls   int32
}

func (p *blockingProvider) Kind() string { return p.kind }

func (p *blockingProvider) Invoke(ctx context.Context, _ toolbox.Manifest, _ map[string]any) (map[string]any, error) {
	atomic.AddInt32(&p.calls, 1)
	select {
	case p.started <- struct{}{}:
	default:
	}
	select {
	case <-p.release:
		return map[string]any{"ok": true}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func newBlockingService(t *testing.T) (*Service, *blockingProvider, *toolbox.MemMetrics) {
	t.Helper()
	reg := registry.New()
	p := &blockingProvider{kind: toolbox.KindTool, started: make(chan struct{}, 1), release: make(chan struct{})}
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
	metrics := toolbox.NewMemMetrics()
	h := host.New(reg, policy.New(nil, toolbox.NewFixedClock(), metrics), metrics)
	contracts := resolver{"ec_ok": {Capabilities: []toolbox.Capability{"read_project"}}}
	return New(h, contracts, metrics), p, metrics
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

func TestConcurrentIdempotencyExecutesOnce(t *testing.T) {
	s, provider, metrics := newBlockingService(t)
	params := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok",
		"input": map[string]any{"x": 1}, "idempotencyKey": "concurrent-k1",
	}
	const callers = 8
	responses := make([]Response, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			responses[i] = s.Handle(context.Background(), req(MethodInvoke, params))
		}(i)
	}
	<-provider.started
	time.Sleep(20 * time.Millisecond)
	close(provider.release)
	wg.Wait()

	if got := atomic.LoadInt32(&provider.calls); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	for i, resp := range responses {
		if resp.Error != nil {
			t.Fatalf("caller %d error = %+v", i, resp.Error)
		}
	}
	if got := metrics.Snapshot(); got.RPCRequests != callers || got.Invocations != 1 {
		t.Fatalf("metrics = %+v", got)
	}
}

func TestIdempotencyCacheServesCompletedRequest(t *testing.T) {
	s, provider, metrics := newBlockingService(t)
	params := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok", "idempotencyKey": "replay-k1",
	}
	firstDone := make(chan Response, 1)
	go func() { firstDone <- s.Handle(context.Background(), req(MethodInvoke, params)) }()
	<-provider.started
	close(provider.release)
	first := <-firstDone
	if first.Error != nil {
		t.Fatalf("first error = %+v", first.Error)
	}
	second := s.Handle(context.Background(), req(MethodInvoke, params))
	if second.Error != nil {
		t.Fatalf("second error = %+v", second.Error)
	}
	if got := atomic.LoadInt32(&provider.calls); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	if got := metrics.Snapshot(); got.CacheHits < 1 || got.CacheMisses < 1 || got.RPCRequests != 2 {
		t.Fatalf("metrics = %+v", got)
	}
}

func TestTimeoutReturnsStructuredError(t *testing.T) {
	s, provider, _ := newBlockingService(t)
	request := req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok", "idempotencyKey": "timeout-k1",
	})
	request.AA = &AA{TimeoutMS: 20}
	started := time.Now()
	resp := s.Handle(context.Background(), request)
	if resp.Error == nil || resp.Error.Code != CodeUnavailable {
		t.Fatalf("error = %+v, want unavailable", resp.Error)
	}
	if resp.Error.Message != "request timed out" {
		t.Fatalf("message = %q", resp.Error.Message)
	}
	if resp.Error.Data["retryable"] != true || resp.Error.Data["category"] != "timeout" {
		t.Fatalf("data = %+v", resp.Error.Data)
	}
	if time.Since(started) > time.Second {
		t.Fatal("timeout took too long")
	}
	if got := atomic.LoadInt32(&provider.calls); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	close(provider.release)
}

func TestWaiterHonorsOwnTimeout(t *testing.T) {
	s, provider, _ := newBlockingService(t)
	params := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok", "idempotencyKey": "waiter-k1",
	}
	firstDone := make(chan Response, 1)
	go func() {
		r := req(MethodInvoke, params)
		r.ID = "first"
		firstDone <- s.Handle(context.Background(), r)
	}()
	<-provider.started
	second := req(MethodInvoke, params)
	second.ID = "second"
	second.AA = &AA{TimeoutMS: 20}
	resp := s.Handle(context.Background(), second)
	if resp.Error == nil || resp.Error.Code != CodeUnavailable || resp.Error.Message != "request timed out" {
		t.Fatalf("waiter error = %+v, want timeout", resp.Error)
	}
	close(provider.release)
	first := <-firstDone
	if first.Error != nil {
		t.Fatalf("first error = %+v", first.Error)
	}
}

func TestTimeoutDoesNotCacheFailure(t *testing.T) {
	s, provider, _ := newBlockingService(t)
	params := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok", "idempotencyKey": "timeout-retry-k1",
	}
	first := req(MethodInvoke, params)
	first.AA = &AA{TimeoutMS: 20}
	if resp := s.Handle(context.Background(), first); resp.Error == nil {
		t.Fatal("expected first request to time out")
	}
	close(provider.release)
	time.Sleep(50 * time.Millisecond)
	second := s.Handle(context.Background(), req(MethodInvoke, params))
	if second.Error != nil {
		t.Fatalf("retry after timeout failed: %+v", second.Error)
	}
	if got := atomic.LoadInt32(&provider.calls); got != 2 {
		t.Fatalf("provider calls = %d, want 2 (first timed out, retry executed)", got)
	}
}

func TestConcurrentKeyReuseWithDifferentRequestConflicts(t *testing.T) {
	s, provider, _ := newBlockingService(t)
	firstParams := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok",
		"input": map[string]any{"x": 1}, "idempotencyKey": "shared-k1",
	}
	firstDone := make(chan Response, 1)
	go func() { firstDone <- s.Handle(context.Background(), req(MethodInvoke, firstParams)) }()
	<-provider.started

	secondParams := map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok",
		"input": map[string]any{"x": 2}, "idempotencyKey": "shared-k1",
	}
	resp := s.Handle(context.Background(), req(MethodInvoke, secondParams))
	if resp.Error == nil || resp.Error.Code != CodeConflict {
		t.Fatalf("second error = %+v, want conflict", resp.Error)
	}
	close(provider.release)
	if first := <-firstDone; first.Error != nil {
		t.Fatalf("first error = %+v", first.Error)
	}
}

func TestRPCLevelErrorMetrics(t *testing.T) {
	s, _ := newServiceWithMetrics(t)
	bad := req(MethodRegistry, nil)
	bad.JSONRPC = "1.0"
	if resp := s.Handle(context.Background(), bad); resp.Error == nil {
		t.Fatal("expected invalid jsonrpc error")
	}
	snap := s.Metrics()
	if snap.RPCRequests != 1 || snap.RPCFailures != 1 || snap.RPCErrors != 1 || snap.RPCByMethod[MethodRegistry] != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestStructuredInvokeErrorData(t *testing.T) {
	s, _ := newService(t)
	resp := s.Handle(context.Background(), req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_no",
	}))
	if resp.Error == nil {
		t.Fatal("expected an error")
	}
	data := resp.Error.Data
	if data["category"] != "unauthorized" || data["toolboxError"] != "ErrDenied" ||
		data["toolId"] != "tool_a" || data["executionContractId"] != "ec_no" {
		t.Fatalf("data = %+v", data)
	}
}

func TestNilAAEnvelopeIsValid(t *testing.T) {
	s, _ := newService(t)
	r := req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok",
	})
	r.AA = nil
	resp := s.Handle(context.Background(), r)
	if resp.Error != nil {
		t.Fatalf("error = %+v", resp.Error)
	}
}

func TestTimeoutAboveMaximumIsRejected(t *testing.T) {
	s, _ := newService(t)
	request := req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok", "idempotencyKey": "too-long-k1",
	})
	request.AA = &AA{TimeoutMS: int(maxRequestTimeout/time.Millisecond) + 1}
	resp := s.Handle(context.Background(), request)
	if resp.Error == nil || resp.Error.Code != CodeInvalidParams || resp.Error.Data["category"] != "invalid" {
		t.Fatalf("error = %+v, want invalid params", resp.Error)
	}
}

func TestNonIdempotentCallHonorsCallerDeadline(t *testing.T) {
	s, provider, _ := newBlockingService(t)
	request := req(MethodInvoke, map[string]any{
		"toolId": "tool_a", "executionContractId": "ec_ok",
	})
	request.AA = &AA{TimeoutMS: 20}
	started := time.Now()
	resp := s.Handle(context.Background(), request)
	if resp.Error == nil || resp.Error.Code != CodeUnavailable || resp.Error.Message != "request timed out" {
		t.Fatalf("error = %+v, want timeout", resp.Error)
	}
	if time.Since(started) > time.Second {
		t.Fatal("non-idempotent timeout took too long")
	}
	<-provider.started
	close(provider.release)
}
