package archtest

import (
	"path/filepath"
	"testing"
)

func TestModuleFromImport(t *testing.T) {
	cases := []struct {
		imp    string
		name   string
		wantOK bool
	}{
		{"github.com/greadee/aa/contracts/go/v1", "contracts", true},
		{"github.com/greadee/aa/kernel", "kernel", true},
		{"github.com/greadee/aa/", "", false},
		{"fmt", "", false},
		{"github.com/other/mod", "", false},
	}
	for _, c := range cases {
		name, ok := ModuleFromImport(c.imp)
		if ok != c.wantOK || name != c.name {
			t.Errorf("ModuleFromImport(%q) = (%q, %v), want (%q, %v)", c.imp, name, ok, c.name, c.wantOK)
		}
	}
}

func TestCheckDetectsViolation(t *testing.T) {
	allowed := map[string]bool{"contracts": true}
	deps := []string{
		"github.com/greadee/aa/contracts/go/v1",
		"github.com/greadee/aa/sync",
		"fmt",
		"github.com/greadee/aa/obsv", // self
	}
	got := Check("obsv", allowed, deps)
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(got), got)
	}
	if got[0].Dependency != "sync" {
		t.Errorf("expected sync violation, got %v", got[0])
	}
}

func TestCheckAllowsPermitted(t *testing.T) {
	allowed := map[string]bool{"contracts": true, "obsv": true}
	deps := []string{
		"github.com/greadee/aa/contracts/go/v1",
		"github.com/greadee/aa/obsv/journal",
		"strings",
	}
	if got := Check("memory", allowed, deps); len(got) != 0 {
		t.Fatalf("expected no violations, got %v", got)
	}
}

func TestLayeringMatchesArchitecture(t *testing.T) {
	allowed := Allowed()
	if len(allowed["contracts"]) != 0 {
		t.Error("contracts must have no aa dependencies")
	}
	if allowed["kernel"]["sync"] || allowed["kernel"]["forge"] {
		t.Error("kernel must reach sync/forge over RPC, not by import")
	}
	if allowed["visualizer"]["kernel"] {
		t.Error("visualizer must not import kernel")
	}
	if !allowed["kernel"]["memory"] || !allowed["kernel"]["obsv"] {
		t.Error("kernel must be allowed to import memory and obsv")
	}
}

func TestWorkspaceBoundaries(t *testing.T) {
	root := filepath.Join("..", "..")
	violations, err := CheckWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		t.Error(v.String())
	}
}
