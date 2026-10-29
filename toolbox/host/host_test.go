package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
)

func manifest(id string, caps ...string) toolbox.Manifest {
	if len(caps) == 0 {
		caps = []string{"read_project"}
	}
	return toolbox.Manifest{
		Envelope:     toolbox.Envelope{ContractVersion: "2.0", Kind: "tool_manifest", ID: id},
		Name:         id,
		ToolKind:     toolbox.KindTool,
		Version:      "1.0",
		Capabilities: caps,
	}
}

func contract(caps ...toolbox.Capability) toolbox.ExecutionContract {
	return toolbox.ExecutionContract{
		Envelope:      toolbox.Envelope{ContractVersion: "2.0", Kind: "execution_contract", ID: "ec_1"},
		WorkPackageID: "wp_1",
		AssignmentID:  "as_1",
		Capabilities:  caps,
	}
}

func newHost(t *testing.T) (*Host, *FakeProvider, *registry.Registry) {
	t.Helper()
	reg := registry.New()
	p := NewFakeProvider(toolbox.KindTool)
	if err := reg.RegisterProvider(p); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	if err := reg.Register(manifest("tool_a")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	eng := policy.New(nil, toolbox.NewFixedClock())
	return New(reg, eng), p, reg
}

func TestInvokeSuccess(t *testing.T) {
	h, p, _ := newHost(t)
	res, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}}, contract("read_project"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if res.Output["tool"] != "tool_a" || res.Output["x"] != 1 {
		t.Fatalf("output = %v", res.Output)
	}
	if len(p.Calls()) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(p.Calls()))
	}
}

func TestInvokeDeniedWithoutCapability(t *testing.T) {
	h, p, _ := newHost(t)
	_, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a"}, contract())
	if !errors.Is(err, toolbox.ErrDenied) {
		t.Fatalf("err = %v, want ErrDenied", err)
	}
	if len(p.Calls()) != 0 {
		t.Fatal("provider ran despite a denial")
	}
	log := h.Audit()
	if len(log) != 1 || log[0].Allowed {
		t.Fatalf("audit = %+v", log)
	}
}

func TestInvokeSandboxDenied(t *testing.T) {
	reg := registry.New()
	_ = reg.RegisterProvider(NewFakeProvider(toolbox.KindTool))
	m := manifest("tool_s")
	m.Sandbox = &toolbox.SandboxSpec{Required: true, Kind: "container"}
	if err := reg.Register(m); err != nil {
		t.Fatalf("Register: %v", err)
	}
	h := New(reg, policy.New(nil, toolbox.NewFixedClock()))
	_, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_s"}, contract("read_project"))
	if !errors.Is(err, toolbox.ErrSandbox) {
		t.Fatalf("err = %v, want ErrSandbox", err)
	}
}

func TestInvokeUnknownTool(t *testing.T) {
	h, _, _ := newHost(t)
	_, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "nope"}, contract("read_project"))
	if !errors.Is(err, toolbox.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestInvokeNoProvider(t *testing.T) {
	reg := registry.New()
	_ = reg.Register(manifest("tool_a"))
	h := New(reg, policy.New(nil, toolbox.NewFixedClock()))
	_, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a"}, contract("read_project"))
	if !errors.Is(err, toolbox.ErrNoProvider) {
		t.Fatalf("err = %v, want ErrNoProvider", err)
	}
}

func TestInvokeIsIdempotent(t *testing.T) {
	h, p, _ := newHost(t)
	inv := toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}, IdempotencyKey: "k1"}
	first, err := h.Invoke(context.Background(), inv, contract("read_project"))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Replayed {
		t.Fatal("first call marked replayed")
	}
	second, err := h.Invoke(context.Background(), inv, contract("read_project"))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !second.Replayed {
		t.Fatal("second call not marked replayed")
	}
	if len(p.Calls()) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(p.Calls()))
	}
}

func TestInvokeDistinctKeysExecuteTwice(t *testing.T) {
	h, p, _ := newHost(t)
	base := toolbox.Invocation{ToolID: "tool_a"}
	base.IdempotencyKey = "k1"
	if _, err := h.Invoke(context.Background(), base, contract("read_project")); err != nil {
		t.Fatalf("first: %v", err)
	}
	base.IdempotencyKey = "k2"
	if _, err := h.Invoke(context.Background(), base, contract("read_project")); err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(p.Calls()) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(p.Calls()))
	}
}

func TestInvokeProviderErrorPropagates(t *testing.T) {
	h, p, _ := newHost(t)
	sentinel := errors.New("boom")
	p.SetError("tool_a", sentinel)
	_, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a"}, contract("read_project"))
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestHostRecordsInvocationAndCacheMetrics(t *testing.T) {
	reg := registry.New()
	p := NewFakeProvider(toolbox.KindTool)
	if err := reg.RegisterProvider(p); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	if err := reg.Register(manifest("tool_a")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	metrics := toolbox.NewMemMetrics()
	h := New(reg, policy.New(nil, toolbox.NewFixedClock(), metrics), metrics)
	inv := toolbox.Invocation{ToolID: "tool_a", IdempotencyKey: "metrics-k1"}
	if _, err := h.Invoke(context.Background(), inv, contract("read_project")); err != nil {
		t.Fatalf("first invoke: %v", err)
	}
	if _, err := h.Invoke(context.Background(), inv, contract("read_project")); err != nil {
		t.Fatalf("replay invoke: %v", err)
	}
	if _, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a"}, contract()); err == nil {
		t.Fatal("expected denial")
	}
	snap := metrics.Snapshot()
	if snap.Invocations != 3 || snap.Denials != 1 || snap.CacheMisses < 1 || snap.CacheHits < 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
	if snap.InvocationsByTool["tool_a"] < 2 || snap.InvocationsByKind["tool"] < 2 || snap.DenialsByTool["tool_a"] != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestInvokeIdempotencyConflictOnDifferentInput(t *testing.T) {
	h, p, _ := newHost(t)
	first := toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}, IdempotencyKey: "k-conflict"}
	if _, err := h.Invoke(context.Background(), first, contract("read_project")); err != nil {
		t.Fatalf("first: %v", err)
	}
	second := first
	second.Input = map[string]any{"x": 2}
	if _, err := h.Invoke(context.Background(), second, contract("read_project")); !errors.Is(err, toolbox.ErrConflict) {
		t.Fatalf("second err = %v, want ErrConflict", err)
	}
	if len(p.Calls()) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(p.Calls()))
	}
}

func TestInvokeExplicitKeyIsContractScoped(t *testing.T) {
	h, p, _ := newHost(t)
	firstContract := contract("read_project")
	firstContract.ID = "ec_1"
	secondContract := contract("read_project")
	secondContract.ID = "ec_2"
	inv := toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}, IdempotencyKey: "contract-k1"}
	if _, err := h.Invoke(context.Background(), inv, firstContract); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := h.Invoke(context.Background(), inv, secondContract); err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(p.Calls()) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(p.Calls()))
	}
}

func TestInvokeWithoutKeyExecutesEveryTime(t *testing.T) {
	h, p, _ := newHost(t)
	if _, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}}, contract("read_project")); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := h.Invoke(context.Background(), toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 2}}, contract("read_project")); err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(p.Calls()) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(p.Calls()))
	}
}

func TestInvokeNilAndEmptyInputAreEquivalentForIdempotency(t *testing.T) {
	h, p, _ := newHost(t)
	inv := toolbox.Invocation{ToolID: "tool_a", IdempotencyKey: "nil-empty-k1"}
	if _, err := h.Invoke(context.Background(), inv, contract("read_project")); err != nil {
		t.Fatalf("first: %v", err)
	}
	inv.Input = map[string]any{}
	if _, err := h.Invoke(context.Background(), inv, contract("read_project")); err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(p.Calls()) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(p.Calls()))
	}
}

func TestHostCacheIsBounded(t *testing.T) {
	h, _, _ := newHost(t)
	h.mu.Lock()
	for i := 0; i < defaultCacheLimit+10; i++ {
		h.storeLocked(fmt.Sprintf("k-%d", i), hostCacheEntry{inputHash: "x"})
	}
	size := len(h.cache)
	h.mu.Unlock()
	if size != defaultCacheLimit {
		t.Fatalf("cache size = %d, want %d", size, defaultCacheLimit)
	}
}

type blockingProvider struct {
	started chan struct{}
	release chan struct{}
	calls   int32
}

func (p *blockingProvider) Kind() string { return toolbox.KindTool }

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

func TestHostConcurrentSameKeyExecutesOnce(t *testing.T) {
	reg := registry.New()
	p := &blockingProvider{started: make(chan struct{}, 1), release: make(chan struct{})}
	if err := reg.RegisterProvider(p); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	if err := reg.Register(manifest("tool_a")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	h := New(reg, policy.New(nil, toolbox.NewFixedClock()))
	inv := toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}, IdempotencyKey: "concurrent-host-k1"}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = h.Invoke(context.Background(), inv, contract("read_project"))
		}(i)
	}
	<-p.started
	time.Sleep(20 * time.Millisecond)
	close(p.release)
	wg.Wait()
	if got := atomic.LoadInt32(&p.calls); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d error = %v", i, err)
		}
	}
}

func TestCachedResultIsDeepCopied(t *testing.T) {
	h, p, _ := newHost(t)
	p.SetResult("tool_a", map[string]any{
		"nested": map[string]any{"x": 1},
		"list":   []any{map[string]any{"y": 2}},
	})
	inv := toolbox.Invocation{ToolID: "tool_a", Input: map[string]any{"x": 1}, IdempotencyKey: "deep-copy-k1"}

	first, err := h.Invoke(context.Background(), inv, contract("read_project"))
	if err != nil {
		t.Fatalf("first Invoke: %v", err)
	}
	first.Output["nested"].(map[string]any)["x"] = 999
	first.Output["list"].([]any)[0].(map[string]any)["y"] = 999

	second, err := h.Invoke(context.Background(), inv, contract("read_project"))
	if err != nil {
		t.Fatalf("second Invoke: %v", err)
	}
	if got := second.Output["nested"].(map[string]any)["x"]; got != 1 {
		t.Fatalf("cached nested output mutated: %v", got)
	}
	if got := second.Output["list"].([]any)[0].(map[string]any)["y"]; got != 2 {
		t.Fatalf("cached list output mutated: %v", got)
	}
	if len(p.Calls()) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(p.Calls()))
	}
}
