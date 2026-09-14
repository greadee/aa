package forge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	scopeRE  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	branchRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
	tagRE    = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.]+)*$`)
)

// Merge classifications (aligned with the audit template and contracts issue types).
const (
	ClassBug           = "bug"
	ClassTechDebt      = "tech_debt"
	ClassRefactor      = "refactor"
	ClassSecurity      = "security"
	ClassTest          = "test"
	ClassDocumentation = "documentation"
)

var classifications = map[string]bool{
	ClassBug: true, ClassTechDebt: true, ClassRefactor: true,
	ClassSecurity: true, ClassTest: true, ClassDocumentation: true,
}

var severities = map[string]bool{"P0": true, "P1": true, "P2": true, "P3": true, "none": true}

// ValidateScope checks a phase scope (lowercase kebab-case).
func ValidateScope(scope string) error {
	if scope == "" {
		return invalid("scope is required")
	}
	if !scopeRE.MatchString(scope) {
		return invalid("scope %q must be lowercase kebab-case", scope)
	}
	return nil
}

// PhaseBranch returns the branch (and folder) name for a phase.
func PhaseBranch(phase int, scope string) string {
	return fmt.Sprintf("ph%d-%s", phase, scope)
}

// ValidatePhaseBranch enforces the rule that the branch name equals the phase
// folder name, ph{N}-{scope}.
func ValidatePhaseBranch(name string, phase int, scope string) error {
	if err := ValidateScope(scope); err != nil {
		return err
	}
	want := PhaseBranch(phase, scope)
	if name != want {
		return invalid("phase branch %q must equal %q", name, want)
	}
	return nil
}

// ValidateBranchName checks a git branch name.
func ValidateBranchName(name string) error {
	if name == "" {
		return invalid("branch name is required")
	}
	if strings.ContainsAny(name, " \t\n") || !branchRE.MatchString(name) || strings.Contains(name, "..") {
		return invalid("branch name %q is not valid", name)
	}
	return nil
}

// ValidateTag checks a release tag (vMAJOR.MINOR.PATCH).
func ValidateTag(tag string) error {
	if !tagRE.MatchString(tag) {
		return invalid("tag %q must look like v1.2.3", tag)
	}
	return nil
}

// ValidateSeverity checks an audit severity.
func ValidateSeverity(severity string) error {
	if !severities[severity] {
		return invalid("severity %q must be one of P0, P1, P2, P3, none", severity)
	}
	return nil
}

// ValidateClassification checks an audit classification.
func ValidateClassification(class string) error {
	if !classifications[class] {
		return invalid("classification %q is not recognized", class)
	}
	return nil
}

// SeverityRank orders severities; lower is more severe (P0 = 0).
func SeverityRank(severity string) int {
	switch severity {
	case "P0":
		return 0
	case "P1":
		return 1
	case "P2":
		return 2
	case "P3":
		return 3
	default:
		return 4
	}
}

// SortFindings sorts findings by severity (most severe first), then by ID.
func SortFindings(findings []AuditFinding) []AuditFinding {
	out := append([]AuditFinding(nil), findings...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := SeverityRank(out[i].Severity), SeverityRank(out[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Validate checks a repository.
func (r Repository) Validate() error {
	if r.Owner == "" || r.Name == "" {
		return invalid("repository owner and name are required")
	}
	return nil
}

// Validate checks an issue.
func (i Issue) Validate() error {
	if i.Title == "" {
		return invalid("issue title is required")
	}
	return nil
}

// Validate checks a pull request.
func (p PullRequest) Validate() error {
	if p.Title == "" {
		return invalid("pull request title is required")
	}
	if p.Base == "" || p.Head == "" {
		return invalid("pull request base and head are required")
	}
	if p.Base == p.Head {
		return invalid("pull request head must differ from base")
	}
	return nil
}

// Validate checks a checkpoint.
func (c Checkpoint) Validate() error {
	if c.Phase == "" {
		return invalid("checkpoint phase is required")
	}
	if c.Commit == "" {
		return invalid("checkpoint commit is required")
	}
	return nil
}

// Validate checks a release.
func (r Release) Validate() error {
	return ValidateTag(r.Tag)
}

// Validate checks an audit and its findings.
func (a Audit) Validate() error {
	if a.Phase == "" {
		return invalid("audit phase is required")
	}
	for i, f := range a.Findings {
		if f.Title == "" {
			return invalid("findings[%d].title is required", i)
		}
		if err := ValidateClassification(f.Classification); err != nil {
			return err
		}
		if err := ValidateSeverity(f.Severity); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks a merge approval.
func (m MergeApproval) Validate() error {
	if m.Actor == "" {
		return invalid("merge approval actor is required")
	}
	return nil
}
