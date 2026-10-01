package kernel

import (
	"context"
	"errors"
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

func computerUseHost(t *testing.T, enforcer policy.SandboxEnforcer) (*host.Host, *computeruse.Fake) {
	t.Helper()
	driver := computeruse.NewFake()
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
