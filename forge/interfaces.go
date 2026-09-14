package forge

import "context"

// Repositories manages repositories and branches.
type Repositories interface {
	CreateRepository(ctx context.Context, repo Repository) (Repository, error)
	CreateBranch(ctx context.Context, repoID string, branch Branch) (Branch, error)
}

// Milestones manages phase milestones.
type Milestones interface {
	CreateMilestone(ctx context.Context, repoID string, milestone Milestone) (Milestone, error)
}

// Issues manages issues.
type Issues interface {
	CreateIssue(ctx context.Context, repoID string, issue Issue) (Issue, error)
	UpdateIssue(ctx context.Context, repoID string, issue Issue) (Issue, error)
}

// PullRequests manages pull requests and the human merge gate.
type PullRequests interface {
	OpenPullRequest(ctx context.Context, repoID string, pr PullRequest) (PullRequest, error)
	UpdatePullRequest(ctx context.Context, repoID string, pr PullRequest) (PullRequest, error)
	MergePullRequest(ctx context.Context, repoID string, number int, approval MergeApproval) (PullRequest, error)
}

// Checkpoints records phase integration points.
type Checkpoints interface {
	CreateCheckpoint(ctx context.Context, repoID string, checkpoint Checkpoint) (Checkpoint, error)
}

// Releases manages releases and tags.
type Releases interface {
	CreateRelease(ctx context.Context, repoID string, release Release) (Release, error)
}

// Audits records repository audits.
type Audits interface {
	RecordAudit(ctx context.Context, repoID string, audit Audit) (Audit, error)
}
