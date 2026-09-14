package forge

import (
	"errors"
	"testing"
	"time"
)

func TestPhaseBranchEqualsFolderName(t *testing.T) {
	if got := PhaseBranch(5, "forge"); got != "ph5-forge" {
		t.Fatalf("PhaseBranch = %q", got)
	}
	if err := ValidatePhaseBranch("ph5-forge", 5, "forge"); err != nil {
		t.Fatalf("valid branch rejected: %v", err)
	}
	err := ValidatePhaseBranch("ph5-forge-impl", 5, "forge")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestValidateScopeRejectsBadScopes(t *testing.T) {
	for _, bad := range []string{"", "Forge", "forge_x", "-forge", "forge-", "forge--x"} {
		if err := ValidateScope(bad); err == nil {
			t.Errorf("scope %q should be rejected", bad)
		}
	}
	for _, good := range []string{"forge", "joblearn", "toolbox", "contracts", "console"} {
		if err := ValidateScope(good); err != nil {
			t.Errorf("scope %q should be accepted: %v", good, err)
		}
	}
}

func TestValidateBranchAndTag(t *testing.T) {
	if err := ValidateBranchName("feature/12-forge"); err != nil {
		t.Fatalf("valid branch rejected: %v", err)
	}
	for _, bad := range []string{"", "bad branch", "-leading", "a..b"} {
		if err := ValidateBranchName(bad); err == nil {
			t.Errorf("branch %q should be rejected", bad)
		}
	}
	if err := ValidateTag("v0.5.0"); err != nil {
		t.Fatalf("valid tag rejected: %v", err)
	}
	for _, bad := range []string{"0.5.0", "v1", "v1.2", "release-1"} {
		if err := ValidateTag(bad); err == nil {
			t.Errorf("tag %q should be rejected", bad)
		}
	}
}

func TestSeverityOrderingAndSort(t *testing.T) {
	if SeverityRank("P0") >= SeverityRank("P3") {
		t.Fatal("P0 must rank more severe than P3")
	}
	findings := []AuditFinding{
		{ID: "f3", Title: "c", Classification: ClassBug, Severity: "P3"},
		{ID: "f1", Title: "a", Classification: ClassBug, Severity: "P1"},
		{ID: "f2", Title: "b", Classification: ClassTest, Severity: "P0"},
	}
	sorted := SortFindings(findings)
	if sorted[0].ID != "f2" || sorted[1].ID != "f1" || sorted[2].ID != "f3" {
		t.Fatalf("unexpected order: %+v", sorted)
	}
	if findings[0].ID != "f3" {
		t.Fatal("SortFindings must not mutate its input")
	}
}

func TestAuditValidation(t *testing.T) {
	audit := Audit{Phase: "ph5-forge", Findings: []AuditFinding{
		{ID: "f1", Title: "leak", Classification: ClassSecurity, Severity: "P0"},
	}}
	if err := audit.Validate(); err != nil {
		t.Fatalf("valid audit rejected: %v", err)
	}
	bad := Audit{Phase: "ph5-forge", Findings: []AuditFinding{
		{ID: "f1", Title: "x", Classification: "nonsense", Severity: "P0"},
	}}
	if err := bad.Validate(); err == nil {
		t.Fatal("bad classification should be rejected")
	}
}

func TestDomainValidation(t *testing.T) {
	if err := (Repository{Owner: "greadee", Name: "aa1"}).Validate(); err != nil {
		t.Fatalf("repository: %v", err)
	}
	if err := (Repository{Owner: "greadee"}).Validate(); err == nil {
		t.Fatal("repository without name should be rejected")
	}
	if err := (PullRequest{Title: "x", Base: "main", Head: "main"}).Validate(); err == nil {
		t.Fatal("PR head == base should be rejected")
	}
	if err := (Release{Tag: "v1.0"}).Validate(); err == nil {
		t.Fatal("bad release tag should be rejected")
	}
	if err := (MergeApproval{}).Validate(); err == nil {
		t.Fatal("merge approval without actor should be rejected")
	}
}

func TestKeyIsStable(t *testing.T) {
	a := Key("issue.create", "repo_1", "Add feature")
	b := Key("issue.create", "repo_1", "  Add feature ")
	if a != b {
		t.Fatalf("key not normalized: %q != %q", a, b)
	}
	if a == Key("issue.create", "repo_1", "Add feature 2") {
		t.Fatal("different parts must yield different keys")
	}
}

func TestAuditLogOrdersOperations(t *testing.T) {
	log := NewAuditLog()
	log.Record(Operation{Operation: "repo.create", Key: "k1"})
	log.Record(Operation{Operation: "issue.create", Key: "k2"})
	entries := log.Entries()
	if len(entries) != 2 || entries[0].Seq != 1 || entries[1].Seq != 2 {
		t.Fatalf("unexpected entries: %+v", entries)
	}
	// Entries is a copy.
	entries[0].Operation = "mutated"
	if log.Entries()[0].Operation != "repo.create" {
		t.Fatal("Entries must return a copy")
	}
}

func TestFixedClockIsDeterministic(t *testing.T) {
	clock := NewFixedClock()
	first := clock.Now()
	second := clock.Now()
	if !first.Equal(time.Unix(0, 0).UTC()) {
		t.Fatalf("first = %v", first)
	}
	if second.Sub(first) != time.Second {
		t.Fatalf("step = %v", second.Sub(first))
	}
}
