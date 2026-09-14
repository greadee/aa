package forge

import "time"

// Ref identifies a forge object across implementations.
type Ref struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	URL  string `json:"url,omitempty"`
}

// Repository describes a version-controlled repository.
type Repository struct {
	ID            string `json:"id"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	Private       bool   `json:"private,omitempty"`
	Ref           Ref    `json:"ref,omitempty"`
}

// Branch records a branch and its tip commit.
type Branch struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commitSha,omitempty"`
}

// Milestone groups issues, typically one per phase.
type Milestone struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State string `json:"state,omitempty"`
	Ref   Ref    `json:"ref,omitempty"`
}

// Issue is a forge issue, deficiency, or finding.
type Issue struct {
	ID        string   `json:"id"`
	Number    int      `json:"number,omitempty"`
	Title     string   `json:"title"`
	Body      string   `json:"body,omitempty"`
	State     string   `json:"state,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone string   `json:"milestone,omitempty"`
	Ref       Ref      `json:"ref,omitempty"`
}

// PullRequest is a forge pull request.
type PullRequest struct {
	ID     string `json:"id"`
	Number int    `json:"number,omitempty"`
	Title  string `json:"title"`
	Body   string `json:"body,omitempty"`
	Base   string `json:"base"`
	Head   string `json:"head"`
	State  string `json:"state,omitempty"`
	Draft  bool   `json:"draft,omitempty"`
	Ref    Ref    `json:"ref,omitempty"`
}

// Checkpoint records a phase integration point: a commit tied to its issues
// and pull requests.
type Checkpoint struct {
	ID        string    `json:"id"`
	Phase     string    `json:"phase"`
	Commit    string    `json:"commit"`
	Issues    []Ref     `json:"issues,omitempty"`
	PRs       []Ref     `json:"pullRequests,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Release records a tag and its notes.
type Release struct {
	ID        string    `json:"id"`
	Tag       string    `json:"tag"`
	Notes     string    `json:"notes,omitempty"`
	Commit    string    `json:"commit,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// AuditFinding is one classified audit result.
type AuditFinding struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Classification string `json:"classification"`
	Severity       string `json:"severity"`
	Evidence       string `json:"evidence,omitempty"`
	Impact         string `json:"impact,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
	Scope          string `json:"scope,omitempty"`
	SuggestedPhase string `json:"suggestedPhase,omitempty"`
}

// Audit is a repository audit and its findings.
type Audit struct {
	ID        string         `json:"id"`
	Phase     string         `json:"phase"`
	Findings  []AuditFinding `json:"findings,omitempty"`
	Result    string         `json:"result,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// MergeApproval is the explicit human gate required to merge a pull request.
type MergeApproval struct {
	Actor     string    `json:"actor"`
	Reason    string    `json:"reason,omitempty"`
	GrantedAt time.Time `json:"grantedAt"`
}

// Audit result values.
const (
	AuditReady               = "ready"
	AuditReadyWithConditions = "ready_with_conditions"
	AuditNotReady            = "not_ready"
)

// MergeActorHuman is the only actor kind permitted to authorize a merge.
const MergeActorHuman = "human"
