package registry

import (
	"context"
	"errors"
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
)

func manifest(id, kind string, caps ...string) toolbox.Manifest {
	if len(caps) == 0 {
		caps = []string{"read_project"}
	}
	return toolbox.Manifest{
		Envelope: toolbox.Envelope{
			ContractVersion: "1.0",
			Kind:            "tool_manifest",
			ID:              id,
		},
		Name:         id,
		ToolKind:     kind,
		Version:      "1.0",
		Capabilities: caps,
	}
}

type fakeProvider struct {
	kind string
}

func (f fakeProvider) Kind() string { return f.kind }

func (f fakeProvider) Invoke(context.Context, toolbox.Manifest, map[string]any) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

func TestRegisterValidManifest(t *testing.T) {
	r := New()
	if err := r.Register(manifest("tool_a", toolbox.KindTool)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := r.Get("tool_a")
	if !ok {
		t.Fatal("Get: not found")
	}
	if got.Name != "tool_a" {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestRegisterRejectsInvalidManifest(t *testing.T) {
	r := New()
	bad := manifest("tool_b", toolbox.KindTool)
	bad.Capabilities = nil
	if err := r.Register(bad); !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("Register invalid = %v, want ErrInvalid", err)
	}
	bad2 := manifest("tool_c", "not_a_kind")
	if err := r.Register(bad2); !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("Register bad kind = %v, want ErrInvalid", err)
	}
}

func TestRegisterDuplicateIsConflict(t *testing.T) {
	r := New()
	if err := r.Register(manifest("tool_a", toolbox.KindTool)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Register(manifest("tool_a", toolbox.KindTool)); !errors.Is(err, toolbox.ErrConflict) {
		t.Fatalf("duplicate = %v, want ErrConflict", err)
	}
}

func TestReplaceOverwrites(t *testing.T) {
	r := New()
	if err := r.Register(manifest("tool_a", toolbox.KindTool)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	updated := manifest("tool_a", toolbox.KindTool, "read_project", "run_tests")
	if err := r.Replace(updated); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	got, _ := r.Get("tool_a")
	if len(got.Capabilities) != 2 {
		t.Fatalf("capabilities = %v, want 2", got.Capabilities)
	}
}

func TestListFiltersAndSorts(t *testing.T) {
	r := New()
	_ = r.Register(manifest("tool_b", toolbox.KindTool))
	_ = r.Register(manifest("tool_a", toolbox.KindTool))
	_ = r.Register(manifest("plugin_a", toolbox.KindPlugin))
	all := r.List("")
	if len(all) != 3 {
		t.Fatalf("all = %d, want 3", len(all))
	}
	if all[0].ID != "plugin_a" || all[2].ID != "tool_b" {
		t.Fatalf("not sorted: %v", ids(all))
	}
	plugins := r.List(toolbox.KindPlugin)
	if len(plugins) != 1 || plugins[0].ID != "plugin_a" {
		t.Fatalf("filtered = %v", ids(plugins))
	}
}

func TestRemove(t *testing.T) {
	r := New()
	_ = r.Register(manifest("tool_a", toolbox.KindTool))
	if !r.Remove("tool_a") {
		t.Fatal("Remove existing = false")
	}
	if r.Remove("tool_a") {
		t.Fatal("Remove missing = true")
	}
	if r.Count() != 0 {
		t.Fatalf("count = %d, want 0", r.Count())
	}
}

func TestRegisterProviderAndNewKindWithoutCoreChange(t *testing.T) {
	r := New()
	if err := r.RegisterProvider(fakeProvider{kind: toolbox.KindTool}); err != nil {
		t.Fatalf("RegisterProvider: %v", err)
	}
	if err := r.RegisterProvider(fakeProvider{kind: toolbox.KindTool}); !errors.Is(err, toolbox.ErrConflict) {
		t.Fatalf("duplicate provider = %v, want ErrConflict", err)
	}
	// A new MCP kind is added by registration alone.
	if err := r.RegisterProvider(fakeProvider{kind: toolbox.KindMCP}); err != nil {
		t.Fatalf("RegisterProvider mcp: %v", err)
	}
	if err := r.Register(manifest("mcp_a", toolbox.KindMCP)); err != nil {
		t.Fatalf("Register mcp: %v", err)
	}
	p, ok := r.ProviderFor(toolbox.KindMCP)
	if !ok || p.Kind() != toolbox.KindMCP {
		t.Fatalf("ProviderFor mcp = %v, %v", p, ok)
	}
	out, err := p.Invoke(context.Background(), toolbox.Manifest{}, nil)
	if err != nil || out["ok"] != true {
		t.Fatalf("Invoke = %v, %v", out, err)
	}
}

func TestRegisterProviderRejectsInvalid(t *testing.T) {
	r := New()
	if err := r.RegisterProvider(nil); !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("nil provider = %v, want ErrInvalid", err)
	}
	if err := r.RegisterProvider(fakeProvider{}); !errors.Is(err, toolbox.ErrInvalid) {
		t.Fatalf("empty kind = %v, want ErrInvalid", err)
	}
}

func ids(ms []toolbox.Manifest) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.ID
	}
	return out
}
