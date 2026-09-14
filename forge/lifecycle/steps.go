package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/greadee/aa/forge"
)

// ChildIssueSpec describes an issue that belongs to a phase.
type ChildIssueSpec struct {
	Title        string
	Goal         string
	Requirements []string
	Acceptance   []string
	Notes        string
	Type         string
}

// ChildPRSpec describes a pull request that targets the phase branch.
type ChildPRSpec struct {
	Title   string
	Branch  string
	Summary string
	Changes []string
	Notes   string
	Issue   int
}

// CreateChildIssue renders and creates a phase child issue, linked to the
// phase milestone, and syncs it to memory when configured.
func (m *Manager) CreateChildIssue(ctx context.Context, phase Phase, spec ChildIssueSpec) (forge.Issue, error) {
	issueType := spec.Type
	if issueType == "" {
		issueType = "feature"
	}
	body := m.render("github/feature-issue.md", map[string]any{
		"Title":        spec.Title,
		"Goal":         spec.Goal,
		"Requirements": spec.Requirements,
		"Acceptance":   spec.Acceptance,
		"Notes":        spec.Notes,
	})
	issue, err := m.forge.CreateIssue(ctx, phase.Repo.ID, forge.Issue{
		Title:     spec.Title,
		Body:      body,
		Type:      issueType,
		Milestone: phase.Milestone.ID,
	})
	if err != nil {
		return forge.Issue{}, fmt.Errorf("child issue: %w", err)
	}
	if err := m.syncIssue(ctx, phase.Repo, issue); err != nil {
		return forge.Issue{}, err
	}
	return issue, nil
}

// OpenChildPR creates the child branch and a pull request into the phase branch.
func (m *Manager) OpenChildPR(ctx context.Context, phase Phase, spec ChildPRSpec) (forge.PullRequest, error) {
	if spec.Branch == "" {
		return forge.PullRequest{}, fmt.Errorf("%w: child PR branch is required", forge.ErrInvalid)
	}
	if _, err := m.forge.CreateBranch(ctx, phase.Repo.ID, forge.Branch{Name: spec.Branch}); err != nil {
		return forge.PullRequest{}, fmt.Errorf("child PR: branch: %w", err)
	}
	body := m.render("github/child-pr.md", map[string]any{
		"Summary":    spec.Summary,
		"Issue":      spec.Issue,
		"Changes":    spec.Changes,
		"Notes":      spec.Notes,
		"PhaseIssue": phase.TrackingIssue.Number,
		"Milestone":  phase.Milestone.Title,
	})
	pr, err := m.forge.OpenPullRequest(ctx, phase.Repo.ID, forge.PullRequest{
		Title: spec.Title,
		Body:  body,
		Base:  phase.Branch,
		Head:  spec.Branch,
		Draft: false,
	})
	if err != nil {
		return forge.PullRequest{}, fmt.Errorf("child PR: %w", err)
	}
	return pr, nil
}

// Checkpoint records a phase checkpoint and syncs it to memory.
func (m *Manager) Checkpoint(ctx context.Context, phase Phase, commit string, issues, prs []forge.Ref) (forge.Checkpoint, error) {
	checkpoint, err := m.forge.CreateCheckpoint(ctx, phase.Repo.ID, forge.Checkpoint{
		Phase:  phase.BranchName(),
		Commit: commit,
		Issues: issues,
		PRs:    prs,
	})
	if err != nil {
		return forge.Checkpoint{}, fmt.Errorf("checkpoint: %w", err)
	}
	if m.exporter != nil && m.sink != nil {
		if _, err := m.exporter.SyncCheckpoint(ctx, m.sink, phase.Repo, checkpoint); err != nil {
			return forge.Checkpoint{}, fmt.Errorf("checkpoint: sync: %w", err)
		}
	}
	return checkpoint, nil
}

// Audit records a phase-end audit.
func (m *Manager) Audit(ctx context.Context, phase Phase, findings []forge.AuditFinding) (forge.Audit, error) {
	audit, err := m.forge.RecordAudit(ctx, phase.Repo.ID, forge.Audit{
		Phase:    phase.BranchName(),
		Findings: findings,
	})
	if err != nil {
		return forge.Audit{}, fmt.Errorf("audit: %w", err)
	}
	return audit, nil
}

// Release creates a release/tag for the phase commit.
func (m *Manager) Release(ctx context.Context, phase Phase, tag, notes, commit string) (forge.Release, error) {
	release, err := m.forge.CreateRelease(ctx, phase.Repo.ID, forge.Release{Tag: tag, Notes: notes, Commit: commit})
	if err != nil {
		return forge.Release{}, fmt.Errorf("release: %w", err)
	}
	return release, nil
}

// FindingsFromAudit maps an audit's findings into issue specs, so an audit can
// file issues without auto-fixing them (ADR-P5-008).
func FindingsFromAudit(audit forge.Audit) []ChildIssueSpec {
	specs := make([]ChildIssueSpec, 0, len(audit.Findings))
	for _, finding := range audit.Findings {
		specs = append(specs, ChildIssueSpec{
			Title:        finding.Title,
			Goal:         finding.Recommendation,
			Requirements: nonEmpty(finding.Evidence, finding.Impact),
			Notes:        strings.Join([]string{finding.Scope, finding.SuggestedPhase}, " "),
			Type:         auditType(finding.Classification),
		})
	}
	return specs
}

func auditType(classification string) string {
	switch classification {
	case forge.ClassBug:
		return "bug"
	case forge.ClassTechDebt:
		return "tech_debt"
	case forge.ClassRefactor:
		return "refactor"
	case forge.ClassTest:
		return "test"
	case forge.ClassDocumentation:
		return "documentation"
	case forge.ClassSecurity:
		return "audit"
	default:
		return "investigation"
	}
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
