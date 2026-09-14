package fake

import (
	"context"
	"errors"
	"testing"

	"github.com/greadee/aa/forge"
)

var _ forge.Audits = (*Fake)(nil)

func TestRecordAuditSortsAndDerivesResult(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	audit, err := f.RecordAudit(context.Background(), repo.ID, forge.Audit{
		Phase: "ph5-forge",
		Findings: []forge.AuditFinding{
			{Title: "minor", Classification: forge.ClassRefactor, Severity: "P3"},
			{Title: "leak", Classification: forge.ClassSecurity, Severity: "P0"},
		},
	})
	if err != nil {
		t.Fatalf("record audit: %v", err)
	}
	if audit.Result != forge.AuditNotReady {
		t.Fatalf("result = %q", audit.Result)
	}
	if audit.Findings[0].Severity != "P0" {
		t.Fatalf("findings not sorted: %+v", audit.Findings)
	}
	if audit.Findings[0].ID == "" {
		t.Fatal("finding IDs were not assigned")
	}
}

func TestAuditIdempotentByPhase(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	first, _ := f.RecordAudit(context.Background(), repo.ID, forge.Audit{Phase: "ph5-forge"})
	second, _ := f.RecordAudit(context.Background(), repo.ID, forge.Audit{
		Phase:    "ph5-forge",
		Findings: []forge.AuditFinding{{Title: "late", Classification: forge.ClassTest, Severity: "P2"}},
	})
	if first.ID != second.ID || second.Result != forge.AuditReady {
		t.Fatalf("audit not idempotent: %+v vs %+v", first, second)
	}
}

func TestAuditRejectsBadFinding(t *testing.T) {
	f := New()
	repo := testRepo(t, f)
	_, err := f.RecordAudit(context.Background(), repo.ID, forge.Audit{
		Phase:    "ph5-forge",
		Findings: []forge.AuditFinding{{Title: "x", Classification: "nonsense", Severity: "P0"}},
	})
	if !errors.Is(err, forge.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestAuditRequiresRepository(t *testing.T) {
	f := New()
	if _, err := f.RecordAudit(context.Background(), "repo_x", forge.Audit{Phase: "p"}); !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
