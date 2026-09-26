package roles

import "testing"

func TestRolesFor(t *testing.T) {
	got := RolesFor([]string{"run_tests", "write_workspace", "deploy"})
	want := []Role{"Builder", "Commissioner", "Inspector"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if !IsKnownRole("Builder") || IsKnownRole("Wizard") {
		t.Fatal("IsKnownRole failed")
	}
}
