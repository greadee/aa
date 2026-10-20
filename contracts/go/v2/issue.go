package v2

import "fmt"

// IssueLinks ties an issue to work, commits, pull requests, and ADRs.
type IssueLinks struct {
	WorkPackages []string    `json:"workPackages,omitempty"`
	Commits      []string    `json:"commits,omitempty"`
	PullRequests []Reference `json:"pullRequests,omitempty"`
	ADRs         []string    `json:"adrs,omitempty"`
}

// Issue is a canonical issue, deficiency, or finding.
type Issue struct {
	Envelope
	Title      string      `json:"title"`
	Type       string      `json:"type"`
	Status     IssueState  `json:"status"`
	Severity   string      `json:"severity,omitempty"`
	Body       string      `json:"body,omitempty"`
	Milestone  string      `json:"milestone,omitempty"`
	ForgeRef   *Reference  `json:"forgeRef,omitempty"`
	Links      *IssueLinks `json:"links,omitempty"`
	DeferredTo string      `json:"deferredTo,omitempty"`
}

// Validate checks the issue contract.
func (i Issue) Validate() error {
	if i.Kind != "issue" {
		return fmt.Errorf("kind: expected %q, got %q", "issue", i.Kind)
	}
	if err := i.Envelope.Validate(); err != nil {
		return err
	}
	if i.Title == "" {
		return fmt.Errorf("title: is required")
	}
	if err := RequireEnum("type", i.Type,
		"feature", "user_story", "bug", "tech_debt", "refactor",
		"test", "documentation", "audit", "investigation", "deficiency"); err != nil {
		return err
	}
	if err := i.Status.Validate(); err != nil {
		return err
	}
	if i.Severity != "" {
		if err := RequireEnum("severity", i.Severity, "P0", "P1", "P2", "P3", "none"); err != nil {
			return err
		}
	}
	return OptionalIdentifier("deferredTo", i.DeferredTo)
}
