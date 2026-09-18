package host

import (
	"context"
	"errors"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
)

func manifest(id string, caps ...string) toolbox.Manifest {
	if len(caps) == 0 {
		caps = []string{"read_project"}
	}
	return toolbox.Manifest{
		Envelope:     toolbox.Envelope{ContractVersion: "1.0", Kind: "tool_manifest", ID: id},
		Name:         id,
		ToolKind:     toolbox.KindTool,
		Version:      "1.0",
		Capabilities: caps,
	}
}

func contract(caps ...toolbox.Capability) toolbox.ExecutionContract {
	return toolbox.ExecutionContract{
		Envelope:      toolbox.Envelope{ContractVersion: "1.0", Kind: "execution_contract", ID: "ec_1"},
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
