package fake

import (
	"context"
	"fmt"

	"github.com/greadee/aa/forge"
)

// RecordAudit stores a repository audit, idempotently by phase. Findings are
// ordered by severity and the result is derived when the caller leaves it empty.
func (f *Fake) RecordAudit(ctx context.Context, repoID string, audit forge.Audit) (forge.Audit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return forge.Audit{}, err
	}
	if _, ok := f.repos[repoID]; !ok {
		return forge.Audit{}, fmt.Errorf("%w: repository %s", forge.ErrNotFound, repoID)
	}
	if err := audit.Validate(); err != nil {
		return forge.Audit{}, err
	}
	key := forge.Key("audit.record", repoID, audit.Phase)
	if existing, ok := f.idem[key]; ok {
		f.record("audit.record", key, repoID, forge.Ref{Kind: "audit", ID: existing.(forge.Audit).ID}, map[string]string{"idempotent": "replay"})
		return existing.(forge.Audit), nil
	}
	if err := f.maybeFail("audit.record"); err != nil {
		return forge.Audit{}, err
	}
	if audit.ID == "" {
		audit.ID = f.nextID("audit")
	}
	for i := range audit.Findings {
		if audit.Findings[i].ID == "" {
			audit.Findings[i].ID = fmt.Sprintf("%s-f%d", audit.ID, i+1)
		}
	}
	audit.Findings = forge.SortFindings(audit.Findings)
	if audit.Result == "" {
		audit.Result = deriveResult(audit.Findings)
	}
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = f.clock.Now()
	}
	if f.audits[repoID] == nil {
		f.audits[repoID] = map[string]forge.Audit{}
	}
	f.audits[repoID][audit.ID] = audit
	f.idem[key] = audit
	f.record("audit.record", key, repoID, forge.Ref{Kind: "audit", ID: audit.ID}, map[string]string{"result": audit.Result})
	return audit, nil
}

// Audit returns an audit by repository and ID.
func (f *Fake) Audit(repoID, id string) (forge.Audit, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	audit, ok := f.audits[repoID][id]
	return audit, ok
}

// deriveResult follows the repository-audits policy: any P0/P1 blocks, any P2
// requires conditions, otherwise the repository is ready.
func deriveResult(findings []forge.AuditFinding) string {
	result := forge.AuditReady
	for _, finding := range findings {
		switch finding.Severity {
		case "P0", "P1":
			return forge.AuditNotReady
		case "P2":
			result = forge.AuditReadyWithConditions
		}
	}
	return result
}
