package contract

import "testing"

func intPtr(v int) *int { return &v }

func TestBuildGrantsIntersection(t *testing.T) {
	c, err := Build(Request{
		ID:            "ec_1",
		ProjectID:     "prj_1",
		WorkPackageID: "wp_1",
		AssignmentID:  "asg_1",
		Requested:     []string{"write_workspace", "run_tests", "deploy"},
		Permitted:     []string{"read_project", "write_workspace", "run_tests"},
		Budget:        Budget{MaxCalls: intPtr(5)},
		Gates:         []string{"human_review", "tests"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Capabilities) != 2 || !c.Has("write_workspace") || !c.Has("run_tests") {
		t.Fatalf("capabilities = %v", c.Capabilities)
	}
	if len(c.Denied) != 1 || c.Denied[0] != "deploy" {
		t.Fatalf("denied = %v", c.Denied)
	}
	if c.Digest == "" {
		t.Fatal("missing digest")
	}
	if len(c.Gates) != 2 || c.Gates[0] != "human_review" || c.Gates[1] != "tests" {
		t.Fatalf("gates = %v", c.Gates)
	}
}

func TestBuildRejectsUnknownCapability(t *testing.T) {
	if _, err := Build(Request{WorkPackageID: "wp", AssignmentID: "a", Requested: []string{"sudo"}}); err == nil {
		t.Fatal("expected unknown capability error")
	}
	if _, err := Build(Request{AssignmentID: "a"}); err == nil {
		t.Fatal("expected required id error")
	}
}

func TestDigestIsOrderIndependent(t *testing.T) {
	base := Request{
		WorkPackageID: "wp", AssignmentID: "a",
		Requested: []string{"write_workspace", "run_tests"},
		Permitted: []string{"write_workspace", "run_tests"},
		Gates:     []string{"b", "a"},
	}
	c1, err := Build(base)
	if err != nil {
		t.Fatal(err)
	}
	base.Requested = []string{"run_tests", "write_workspace"}
	base.Gates = []string{"a", "b"}
	c2, err := Build(base)
	if err != nil {
		t.Fatal(err)
	}
	if c1.Digest != c2.Digest {
		t.Fatalf("digests differ: %s vs %s", c1.Digest, c2.Digest)
	}
}

func TestDeniedNeverGranted(t *testing.T) {
	c, err := Build(Request{WorkPackageID: "wp", AssignmentID: "a", Requested: []string{"merge"}})
	if err != nil {
		t.Fatal(err)
	}
	if c.Has("merge") {
		t.Fatal("merge must not be granted when not permitted")
	}
	if len(c.Denied) != 1 {
		t.Fatalf("denied = %v", c.Denied)
	}
}
