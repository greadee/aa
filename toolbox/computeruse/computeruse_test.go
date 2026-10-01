package computeruse

import (
	"context"
	"errors"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
)

func mustProvider(t *testing.T, driver Driver) *Provider {
	t.Helper()
	p, err := NewProvider(driver)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestManifestDeclaresCapabilityAndPermissions(t *testing.T) {
	m := Manifest("tool_screen")
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest invalid: %v", err)
	}
	if m.ToolKind != toolbox.KindBuiltin {
		t.Fatalf("toolKind = %q", m.ToolKind)
	}
	if !declaresPermission(m, PermissionScreenCapture) || !declaresPermission(m, PermissionInputInjection) {
		t.Fatalf("permissions = %v", m.Permissions)
	}
	if m.Sandbox == nil || !m.Sandbox.Required || m.Sandbox.Kind != "process" {
		t.Fatalf("sandbox = %+v", m.Sandbox)
	}
}

func TestProviderRunsActionAndReportsArtifacts(t *testing.T) {
	driver := NewFake()
	driver.Script(ActionScreenshot, Outcome{
		Output:    map[string]any{"width": 1280},
		Artifacts: []Artifact{{ID: "shot_1", Kind: "screenshot", Hash: "abc"}},
	})
	provider := mustProvider(t, driver)

	out, err := provider.Invoke(context.Background(), Manifest("tool_screen"), map[string]any{
		"action": map[string]any{"kind": "screenshot"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["permission"] != PermissionScreenCapture || out["width"] != 1280 {
		t.Fatalf("output = %+v", out)
	}
	artifacts, ok := out["artifacts"].([]map[string]any)
	if !ok || len(artifacts) != 1 || artifacts[0]["id"] != "shot_1" {
		t.Fatalf("artifacts = %+v", out["artifacts"])
	}
	if len(driver.Calls()) != 1 || driver.Calls()[0].Kind != ActionScreenshot {
		t.Fatalf("calls = %+v", driver.Calls())
	}
}

func TestProviderDeniesUndeclaredPermission(t *testing.T) {
	m := Manifest("tool_screen")
	m.Permissions = []string{PermissionInputInjection} // screen_capture not declared
	_, err := mustProvider(t, NewFake()).Invoke(context.Background(), m, map[string]any{
		"action": map[string]any{"kind": "screenshot"},
	})
	if !errors.Is(err, toolbox.ErrDenied) {
		t.Fatalf("err = %v", err)
	}
}

func TestProviderRejectsInvalidAction(t *testing.T) {
	provider := mustProvider(t, NewFake())
	manifest := Manifest("tool_screen")
	cases := []map[string]any{
		{}, // missing action
		{"action": map[string]any{"kind": "hover"}},          // unknown kind
		{"action": map[string]any{"kind": "type"}},           // missing text
		{"action": map[string]any{"kind": "click", "x": -1}}, // negative coordinate
	}
	for _, input := range cases {
		if _, err := provider.Invoke(context.Background(), manifest, input); !errors.Is(err, toolbox.ErrInvalid) {
			t.Fatalf("input %v: err = %v", input, err)
		}
	}
}

func TestProviderPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mustProvider(t, NewFake()).Invoke(ctx, Manifest("tool_screen"), map[string]any{
		"action": map[string]any{"kind": "screenshot"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestProviderWrapsDriverFailure(t *testing.T) {
	driver := NewFake()
	driver.ScriptError(ActionKey, errors.New("input device unavailable"))
	_, err := mustProvider(t, driver).Invoke(context.Background(), Manifest("tool_screen"), map[string]any{
		"action": map[string]any{"kind": "key", "key": "Enter"},
	})
	if !errors.Is(err, toolbox.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestProviderKind(t *testing.T) {
	if got := mustProvider(t, NewFake()).Kind(); got != toolbox.KindBuiltin {
		t.Fatalf("kind = %q", got)
	}
}
