package kernel

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/greadee/aa/runtime/sandbox"
	toolbox "github.com/greadee/aa/toolbox"
	"github.com/greadee/aa/toolbox/computeruse"
	"github.com/greadee/aa/toolbox/host"
	"github.com/greadee/aa/toolbox/policy"
	"github.com/greadee/aa/toolbox/registry"
)

// The kernel composes the toolbox (tool capability) with the runtime sandbox
// (isolation) at the policy seam; roles and the scheduler never see either.

// fakeDriver is a local, deterministic computeruse.Driver for these tests. The
// production driver seam carries no test double; each consumer supplies its own.
type fakeDriver struct {
	mu    sync.Mutex
	calls []computeruse.Action
}

func (f *fakeDriver) Perform(ctx context.Context, action computeruse.Action) (computeruse.Outcome, error) {
	f.mu.Lock()
	f.calls = append(f.calls, action)
	f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return computeruse.Outcome{}, err
	}
	return computeruse.Outcome{Output: map[string]any{"action": string(action.Kind)}}, nil
}

func (f *fakeDriver) Close() error { return nil }

func (f *fakeDriver) Calls() []computeruse.Action {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]computeruse.Action(nil), f.calls...)
}

func computerUseHost(t *testing.T, enforcer policy.SandboxEnforcer) (*host.Host, *fakeDriver) {
	t.Helper()
	driver := &fakeDriver{}
	provider, err := computeruse.NewProvider(driver)
	if err != nil {
		t.Fatal(err)
	}
	reg := registry.New()
	if err := reg.RegisterProvider(provider); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(computeruse.Manifest("tool_screen")); err != nil {
		t.Fatal(err)
	}
	return host.New(reg, policy.New(enforcer, toolbox.NewFixedClock())), driver
}

func computerUseContract() toolbox.ExecutionContract {
	return toolbox.ExecutionContract{
		Envelope:     toolbox.Envelope{ID: "ec_1", Kind: "execution_contract", ContractVersion: "2.0"},
		Capabilities: []toolbox.Capability{toolbox.CapReadProject, toolbox.CapCreateArtifact},
	}
}

func screenshotInvocation() toolbox.Invocation {
	return toolbox.Invocation{
		ToolID: "tool_screen",
		Input:  map[string]any{"action": map[string]any{"kind": "screenshot"}},
	}
}

func TestComputerUseFlowsThroughRuntimeSandbox(t *testing.T) {
	h, driver := computerUseHost(t, sandbox.Enforcer{})
	result, err := h.Invoke(context.Background(), screenshotInvocation(), computerUseContract())
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["permission"] != computeruse.PermissionScreenCapture {
		t.Fatalf("output = %+v", result.Output)
	}
	if len(driver.Calls()) != 1 {
		t.Fatalf("driver calls = %d", len(driver.Calls()))
	}
}

func TestComputerUseFailsClosedWithoutSandboxEnforcer(t *testing.T) {
	h, driver := computerUseHost(t, nil)
	if _, err := h.Invoke(context.Background(), screenshotInvocation(), computerUseContract()); !errors.Is(err, toolbox.ErrSandbox) {
		t.Fatalf("err = %v", err)
	}
	if len(driver.Calls()) != 0 {
		t.Fatalf("driver ran despite missing isolation")
	}
}
