package sandbox

import "testing"

func TestEnforcerReportsProcessKind(t *testing.T) {
	var e Enforcer = NewEnforcer()
	if !e.CanEnforce(KindProcess) {
		t.Fatalf("process must be enforceable")
	}
	if e.CanEnforce("vm") || e.CanEnforce("none") || e.CanEnforce("") {
		t.Fatalf("only the process kind is enforceable")
	}
}
