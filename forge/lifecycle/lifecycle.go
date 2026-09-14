// Package lifecycle drives a complete phase lifecycle over a Forge.
//
// It enforces the phase-process rules: the branch name equals the phase folder
// name (ph{N}-{scope}), the phase pull request exists as a Draft from phase
// start, and merging requires an explicit human approval.
package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/greadee/aa/forge"
	"github.com/greadee/aa/forge/memsync"
	"github.com/greadee/aa/forge/template"
)

// Forge is the composite capability set the lifecycle needs.
type Forge interface {
	forge.Repositories
	forge.Milestones
	forge.Issues
	forge.PullRequests
	forge.Checkpoints
	forge.Releases
	forge.Audits
}

// Phase is the state collected while driving a phase.
type Phase struct {
	Repo          forge.Repository
	Number        int
	Scope         string
	Branch        string
	Milestone     forge.Milestone
	TrackingIssue forge.Issue
	PhasePR       forge.PullRequest
}

// BranchName returns the expected phase branch for the phase.
func (p Phase) BranchName() string { return forge.PhaseBranch(p.Number, p.Scope) }

// PhaseSpec describes a phase to start.
type PhaseSpec struct {
	Repository forge.Repository
	Number     int
	Scope      string
	Objective  string
	Milestone  string
}

// TrackingIssueData is the template data for the phase tracking issue.
type TrackingIssueData struct {
	Number       int
	Objective    string
	Scope        []string
	Issues       []int
	Dependencies string
	Risks        string
}

// Manager orchestrates a phase over a Forge.
type Manager struct {
	forge     Forge
	templates *template.Engine
	exporter  *memsync.Exporter
	sink      memsync.Sink
}

// Option configures a Manager.
type Option func(*Manager)

// WithTemplates enables data-driven rendering from the process templates.
func WithTemplates(engine *template.Engine) Option {
	return func(m *Manager) { m.templates = engine }
}

// WithMemory enables forge-to-memory synchronization.
func WithMemory(exporter *memsync.Exporter, sink memsync.Sink) Option {
	return func(m *Manager) { m.exporter = exporter; m.sink = sink }
}

// New returns a Manager over f.
func New(f Forge, opts ...Option) *Manager {
	manager := &Manager{forge: f}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

// StartPhase creates the phase branch, milestone, tracking issue, and Draft
// phase pull request. Every step is idempotent.
func (m *Manager) StartPhase(ctx context.Context, spec PhaseSpec) (Phase, error) {
	if err := forge.ValidateScope(spec.Scope); err != nil {
		return Phase{}, err
	}
	branch := forge.PhaseBranch(spec.Number, spec.Scope)
	if err := forge.ValidatePhaseBranch(branch, spec.Number, spec.Scope); err != nil {
		return Phase{}, err
	}
	repo, err := m.forge.CreateRepository(ctx, spec.Repository)
	if err != nil {
		return Phase{}, fmt.Errorf("start phase: repository: %w", err)
	}
	if _, err := m.forge.CreateBranch(ctx, repo.ID, forge.Branch{Name: branch}); err != nil {
		return Phase{}, fmt.Errorf("start phase: branch: %w", err)
	}
	milestoneTitle := spec.Milestone
	if milestoneTitle == "" {
		milestoneTitle = fmt.Sprintf("Phase %d — %s", spec.Number, titleCase(spec.Scope))
	}
	milestone, err := m.forge.CreateMilestone(ctx, repo.ID, forge.Milestone{Title: milestoneTitle})
	if err != nil {
		return Phase{}, fmt.Errorf("start phase: milestone: %w", err)
	}
	issue, err := m.CreateTrackingIssue(ctx, repo, spec, milestone.ID)
	if err != nil {
		return Phase{}, err
	}
	pr, err := m.forge.OpenPullRequest(ctx, repo.ID, forge.PullRequest{
		Title: fmt.Sprintf("Phase %d: %s", spec.Number, spec.Objective),
		Body:  m.phasePRBody(spec, issue),
		Base:  repo.DefaultBranch,
		Head:  branch,
		Draft: true,
	})
	if err != nil {
		return Phase{}, fmt.Errorf("start phase: phase PR: %w", err)
	}
	return Phase{
		Repo: repo, Number: spec.Number, Scope: spec.Scope, Branch: branch,
		Milestone: milestone, TrackingIssue: issue, PhasePR: pr,
	}, nil
}

// CreateTrackingIssue creates the phase tracking issue linked to milestoneID.
func (m *Manager) CreateTrackingIssue(ctx context.Context, repo forge.Repository, spec PhaseSpec, milestoneID string) (forge.Issue, error) {
	body := m.render("github/phase-tracking-issue.md", TrackingIssueData{
		Number:    spec.Number,
		Objective: spec.Objective,
		Scope:     []string{},
		Issues:    []int{},
	})
	issue, err := m.forge.CreateIssue(ctx, repo.ID, forge.Issue{
		Title:     fmt.Sprintf("Phase %d: %s", spec.Number, spec.Objective),
		Body:      body,
		Type:      "feature",
		Milestone: milestoneID,
	})
	if err != nil {
		return forge.Issue{}, fmt.Errorf("start phase: tracking issue: %w", err)
	}
	if err := m.syncIssue(ctx, repo, issue); err != nil {
		return forge.Issue{}, err
	}
	return issue, nil
}

// CompletePhase marks the phase pull request ready and merges it through the
// human gate. It fails closed without an approving actor.
func (m *Manager) CompletePhase(ctx context.Context, phase Phase, commit string, approval forge.MergeApproval) (forge.PullRequest, error) {
	if approval.Actor == "" {
		return forge.PullRequest{}, fmt.Errorf("%w: merge actor is required", forge.ErrHumanGate)
	}
	if _, err := m.Checkpoint(ctx, phase, commit, []forge.Ref{refIssue(phase.TrackingIssue)}, []forge.Ref{refPR(phase.PhasePR)}); err != nil {
		return forge.PullRequest{}, err
	}
	ready, err := m.forge.UpdatePullRequest(ctx, phase.Repo.ID, forge.PullRequest{Number: phase.PhasePR.Number, Draft: false})
	if err != nil {
		return forge.PullRequest{}, err
	}
	merged, err := m.forge.MergePullRequest(ctx, phase.Repo.ID, ready.Number, approval)
	if err != nil {
		return forge.PullRequest{}, err
	}
	return merged, nil
}

func (m *Manager) render(name string, data any) string {
	if m.templates == nil {
		return ""
	}
	out, err := m.templates.Render(name, data)
	if err != nil {
		return ""
	}
	return out
}

func (m *Manager) phasePRBody(spec PhaseSpec, issue forge.Issue) string {
	body := m.render("github/phase-pr.md", map[string]any{
		"Objective":      spec.Objective,
		"TrackingIssue":  issue.Number,
		"Milestone":      spec.Milestone,
		"Features":       []string{},
		"Infrastructure": []string{},
		"Testing":        []string{},
		"Documentation":  []string{},
		"ChildPRs":       []int{},
		"Status":         "Phase started.",
		"Risks":          "",
	})
	if body != "" {
		return body
	}
	return issue.Body
}

func (m *Manager) syncIssue(ctx context.Context, repo forge.Repository, issue forge.Issue) error {
	if m.exporter == nil || m.sink == nil {
		return nil
	}
	if _, err := m.exporter.SyncIssue(ctx, m.sink, repo, issue); err != nil {
		return fmt.Errorf("lifecycle: sync issue: %w", err)
	}
	return nil
}

func refIssue(issue forge.Issue) forge.Ref {
	kind := issue.Ref.Kind
	if kind == "" {
		kind = "issue"
	}
	return forge.Ref{Kind: kind, ID: issue.Ref.ID}
}

func refPR(pr forge.PullRequest) forge.Ref {
	kind := pr.Ref.Kind
	if kind == "" {
		kind = "pull_request"
	}
	return forge.Ref{Kind: kind, ID: pr.Ref.ID}
}

func titleCase(scope string) string {
	parts := strings.Split(scope, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
