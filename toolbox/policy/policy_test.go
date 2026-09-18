package policy

import (
	"testing"

	toolbox "github.com/greadee/aa/toolbox"
)

type enforcer map[string]bool

func (e enforcer) CanEnforce(kind string) bool { return e[kind] }

func manifest(id string, caps ...string) toolbox.Manifest {
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

func TestGrantIsIntersection(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	d := e.Evaluate(manifest("t", "read_project", "run_tests"), contract("read_project", "run_tests", "merge"))
	if !d.Allowed {
		t.Fatalf("allowed = false, reason %q", d.Reason)
	}
	if len(d.Granted) != 2 {
		t.Fatalf("granted = %v, want 2", d.Granted)
	}
	// merge is in the contract but not declared by the tool: not granted.
	for _, c := range d.Granted {
		if c == "merge" {
			t.Fatalf("granted an undeclared capability: %v", d.Granted)
		}
	}
}

func TestMissingCapabilityDenies(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	d := e.Evaluate(manifest("t", "read_project", "run_tests"), contract("read_project"))
	if d.Allowed {
		t.Fatal("allowed with a missing capability")
	}
	if len(d.Missing) != 1 || d.Missing[0] != "run_tests" {
		t.Fatalf("missing = %v", d.Missing)
	}
}

func TestNoImplicitGrant(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	d := e.Evaluate(manifest("t", "read_project"), contract())
	if d.Allowed {
		t.Fatal("granted a capability with an empty contract")
	}
	if len(d.Missing) != 1 {
		t.Fatalf("missing = %v", d.Missing)
	}
}

func TestExplicitDenialWins(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	c := contract("run_tests")
	c.Denied = []toolbox.Capability{"run_tests"}
	d := e.Evaluate(manifest("t", "run_tests"), c)
	if d.Allowed {
		t.Fatal("denied capability was allowed")
	}
	if len(d.Denied) != 1 || len(d.Missing) != 0 {
		t.Fatalf("denied = %v, missing = %v", d.Denied, d.Missing)
	}
}

func TestSandboxNetworkRequiresCapability(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	m := manifest("t", "read_project")
	m.Sandbox = &toolbox.SandboxSpec{Required: false, Kind: "process", Network: true}
	d := e.Evaluate(m, contract("read_project"))
	if d.Allowed {
		t.Fatal("network sandbox allowed without network_access")
	}
	if d.Sandbox.Allowed {
		t.Fatal("sandbox decision allowed; want denied")
	}

	d2 := e.Evaluate(m, contract("read_project", "network_access"))
	if !d2.Allowed {
		t.Fatalf("network sandbox denied with network_access: %q", d2.Reason)
	}
}

func TestRequiredSandboxFailsClosedWithoutEnforcer(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	m := manifest("t", "read_project")
	m.Sandbox = &toolbox.SandboxSpec{Required: true, Kind: "container"}
	d := e.Evaluate(m, contract("read_project"))
	if d.Allowed || d.Sandbox.Allowed {
		t.Fatal("required sandbox allowed with no enforcer")
	}
}

func TestRequiredSandboxAllowedWhenEnforceable(t *testing.T) {
	e := New(enforcer{"container": true}, toolbox.NewFixedClock())
	m := manifest("t", "read_project")
	m.Sandbox = &toolbox.SandboxSpec{Required: true, Kind: "container"}
	d := e.Evaluate(m, contract("read_project"))
	if !d.Allowed {
		t.Fatalf("enforceable sandbox denied: %q", d.Reason)
	}
}

func TestRequiredSandboxKindNoneDenies(t *testing.T) {
	e := New(enforcer{"container": true}, toolbox.NewFixedClock())
	m := manifest("t", "read_project")
	m.Sandbox = &toolbox.SandboxSpec{Required: true, Kind: "none"}
	d := e.Evaluate(m, contract("read_project"))
	if d.Allowed {
		t.Fatal("required sandbox with kind none was allowed")
	}
}

func TestAuditLogIsOrderedAndCopied(t *testing.T) {
	e := New(nil, toolbox.NewFixedClock())
	e.Evaluate(manifest("t1", "read_project"), contract("read_project"))
	e.Evaluate(manifest("t2", "run_tests"), contract())
	log := e.Audit()
	if len(log) != 2 {
		t.Fatalf("log = %d, want 2", len(log))
	}
	if log[0].Seq != 1 || log[1].Seq != 2 {
		t.Fatalf("seq out of order: %d, %d", log[0].Seq, log[1].Seq)
	}
	if !log[0].Allowed || log[1].Allowed {
		t.Fatalf("decisions = %v, %v", log[0].Allowed, log[1].Allowed)
	}
	if !log[1].At.After(log[0].At) {
		t.Fatalf("timestamps not ordered: %v, %v", log[0].At, log[1].At)
	}
	log[0].Allowed = false
	if !e.Audit()[0].Allowed {
		t.Fatal("Audit returned an aliased slice")
	}
}
