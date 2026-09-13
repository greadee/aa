package registry

import "testing"

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	r := New()
	workers := []Worker{
		{ID: "w_fast", Trade: "backend", Roles: []Role{"Builder"}, Capabilities: []string{"read_project", "write_workspace", "run_tests"}, Available: true, CostWeight: 1},
		{ID: "w_slow", Trade: "backend", Roles: []Role{"Builder"}, Capabilities: []string{"read_project", "write_workspace", "run_tests"}, Available: true, CostWeight: 5},
		{ID: "w_busy", Trade: "backend", Roles: []Role{"Builder"}, Capabilities: []string{"read_project", "write_workspace"}, Available: false},
		{ID: "w_front", Trade: "frontend", Roles: []Role{"Builder"}, Capabilities: []string{"read_project", "write_workspace"}, Available: true},
		{ID: "w_qa", Trade: "backend", Roles: []Role{"Inspector"}, Capabilities: []string{"read_project", "run_tests"}, Available: true},
	}
	for _, w := range workers {
		if err := r.Add(w); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestAddRejectsDuplicate(t *testing.T) {
	r := testRegistry(t)
	if err := r.Add(Worker{ID: "w_fast"}); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestSelectPrefersLowCostAndReportsRejections(t *testing.T) {
	r := testRegistry(t)
	accepted, rejected := r.Select(Requirement{
		Trade: "backend", Roles: []Role{"Builder"}, Capabilities: []string{"write_workspace"},
	})
	if len(accepted) != 2 {
		t.Fatalf("expected 2 accepted, got %d: %+v", len(accepted), accepted)
	}
	if accepted[0].Worker.ID != "w_fast" || accepted[1].Worker.ID != "w_slow" {
		t.Fatalf("unexpected order: %+v", accepted)
	}
	reasons := map[WorkerID]string{}
	for _, rej := range rejected {
		reasons[rej.Worker.ID] = rej.Reason
	}
	if reasons["w_busy"] != "unavailable" {
		t.Errorf("w_busy reason = %q", reasons["w_busy"])
	}
	if reasons["w_front"] != "trade mismatch" {
		t.Errorf("w_front reason = %q", reasons["w_front"])
	}
	if reasons["w_qa"] != "role mismatch" {
		t.Errorf("w_qa reason = %q", reasons["w_qa"])
	}
}

func TestSelectRejectsMissingCapabilities(t *testing.T) {
	r := testRegistry(t)
	accepted, rejected := r.Select(Requirement{
		Trade: "backend", Capabilities: []string{"read_project", "run_tests", "deploy"},
	})
	if len(accepted) != 0 {
		t.Fatalf("expected none, got %+v", accepted)
	}
	found := false
	for _, rej := range rejected {
		if rej.Reason == "missing capabilities: deploy" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing deploy rejection, got %+v", rejected)
	}
}

func TestSelectOne(t *testing.T) {
	r := testRegistry(t)
	w, ok := r.SelectOne(Requirement{Trade: "backend", Capabilities: []string{"write_workspace"}})
	if !ok || w.ID != "w_fast" {
		t.Fatalf("got %+v ok=%v", w, ok)
	}
	if _, ok := r.SelectOne(Requirement{Trade: "nope"}); ok {
		t.Fatal("expected no worker")
	}
}

func TestRolesFor(t *testing.T) {
	roles := RolesFor([]string{"run_tests", "write_workspace", "deploy"})
	want := []Role{"Builder", "Commissioner", "Inspector"}
	if len(roles) != len(want) {
		t.Fatalf("got %v", roles)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("got %v, want %v", roles, want)
		}
	}
	if !IsKnownRole("Builder") || IsKnownRole("Wizard") {
		t.Fatal("IsKnownRole failed")
	}
}
