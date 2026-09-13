# aa-issue.md

Aspect directive for the **issue repository** inside the `memory` module.

## Responsibility

Own the canonical, durable record of issues: bugs, features, investigations, tech debt, audits, and deficiencies. GitHub issues are an execution surface; this repository is the durable knowledge and cross-reference layer.

## Owns

- Canonical issue records (identity, type, status, severity, provenance).
- Issue-to-work-package, issue-to-artifact, and issue-to-commit links.
- Deficiency records raised by gates and inspections.
- Audit findings promoted to issues.
- The crosswalk between issue states and forge milestones/checkpoints.

## Must Not

- Perform forge operations (that is `forge`).
- Lose an issue or finding because a session ended.

## Rules

1. Every issue has a stable identity and a durable record.
2. Findings that must survive a session become issues.
3. Issues are superseded or deferred with a destination, never silently deleted.
4. Forge is the source of live status; memory is the source of durable truth.

## Canonical references

- [GitHub issues and stories](../docs/github-os/issues-and-stories.md)
- [Terminology — issue states](../docs/reference/terminology.md)
